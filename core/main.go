package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ycair.online/core/mesh"
	"ycair.online/core/p2p"
	"ycair.online/core/signaling"
	"ycair.online/core/tun"
)

type StatusPeer struct {
	ID string `json:"id"`
	IP string `json:"ip"`
}

type StatusMessage struct {
	Type       string       `json:"type"`
	AssignedIP string       `json:"assigned_ip"`
	PeerID     string       `json:"peer_id"`
	Peers      []StatusPeer `json:"peers"`
	TUN        string       `json:"tun"`
	Connected  bool         `json:"connected"`
}

func printStatus(connMgr *p2p.ConnectionManager, client *signaling.Client, tunIfce *tun.Interface) {
	peers := connMgr.GetPeers()
	peerList := make([]StatusPeer, 0, len(peers))
	for _, p := range peers {
		peerList = append(peerList, StatusPeer{ID: p.PeerID, IP: p.AssignedIP})
	}

	tunName := ""
	if tunIfce != nil {
		tunName = tunIfce.Name()
	}

	msg := StatusMessage{
		Type:       "status",
		AssignedIP: client.AssignedIP(),
		PeerID:     client.PeerID(),
		Peers:      peerList,
		TUN:        tunName,
		Connected:  true,
	}

	data, _ := json.Marshal(msg)
	fmt.Fprintf(os.Stdout, "YCAR_STATUS:%s\n", string(data))
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("usage: ycair-core <room_code> <password> [signaling_addr]")
		os.Exit(1)
	}

	room := os.Args[1]
	password := os.Args[2]

	signalingAddr := "localhost:9090"
	if len(os.Args) >= 4 {
		signalingAddr = os.Args[3]
	}

	connMgr := p2p.NewConnectionManager(0)
	if err := connMgr.Start(); err != nil {
		log.Fatalf("NAT discovery failed: %v", err)
	}
	defer connMgr.Close()

	localEndpoints := append(
		discoverLocalEndpoints(connMgr.LocalPort()),
		connMgr.PublicAddr(),
	)

	log.Printf("ycair-core: starting for room %q", room)
	log.Printf("ycair-core: public endpoint %s", connMgr.PublicAddr())

	client, err := signaling.Connect(signalingAddr, room, password, localEndpoints)
	if err != nil {
		log.Fatalf("Failed to connect to signaling server: %v", err)
	}
	defer client.Close()

	client.WaitForWelcome()

	log.Printf("ycair-core: registered as %s, assigned IP %s",
		client.PeerID(), client.AssignedIP())

	printStatus(connMgr, client, nil)

	connMgr.SetPeerID(client.PeerID())

	tunIfce, err := tun.Create(client.AssignedIP())
	if err != nil {
		log.Printf("TUN: failed to create interface: %v", err)
		log.Println("TUN: running in signaling-only mode (no virtual network)")
	} else {
		defer tunIfce.Close()
		log.Printf("TUN: interface %s created, IP %s", tunIfce.Name(), tunIfce.IP())

		meshNet := mesh.New(tunIfce, connMgr)
		meshNet.Start()
	}

	printStatus(connMgr, client, tunIfce)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	statusTicker := time.NewTicker(10 * time.Second)
	defer statusTicker.Stop()

	for {
		select {
		case event := <-client.Events():
			switch event.Type {
			case signaling.EventPeerJoined:
				log.Printf("Peer joined: id=%s ip=%s",
					event.Peer.ID, event.Peer.IP)
				connMgr.HandleSignalingEvent(event)
				printStatus(connMgr, client, tunIfce)

			case signaling.EventPeerLeft:
				log.Printf("Peer left: id=%s", event.Peer.ID)
				connMgr.HandleSignalingEvent(event)
				printStatus(connMgr, client, tunIfce)

			case signaling.EventError:
				log.Printf("Signaling error: %s", event.Error)
			}

		case <-statusTicker.C:
			peers := connMgr.GetPeers()
			tunStatus := "no"
			if tunIfce != nil {
				tunStatus = tunIfce.Name()
			}
			log.Printf("Status: %d peers, tun=%s, ip=%s",
				len(peers), tunStatus, client.AssignedIP())
			printStatus(connMgr, client, tunIfce)

		case <-sig:
			log.Println("ycair-core: shutting down...")
			return
		}
	}
}

func discoverLocalEndpoints(port int) []string {
	var endpoints []string

	interfaces, err := net.Interfaces()
	if err != nil {
		return endpoints
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			if ipNet, ok := addr.(*net.IPNet); ok && ipNet.IP.To4() != nil {
				endpoints = append(endpoints, fmt.Sprintf("%s:%d", ipNet.IP, port))
			}
		}
	}

	return endpoints
}
