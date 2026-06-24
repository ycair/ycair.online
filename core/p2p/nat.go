package p2p

import (
	"fmt"
	"log"
	"net"
	"time"

	"github.com/pion/stun/v3"
)

const (
	stunServer     = "stun.l.google.com:19302"
	holePunchRetry = 50
	holePunchDelay = 50 * time.Millisecond
)

type NATInfo struct {
	PublicAddr *net.UDPAddr
	LocalAddr  *net.UDPAddr
}

func DiscoverNAT(localPort int) (*NATInfo, *net.UDPConn, error) {
	addr := &net.UDPAddr{IP: net.IPv4zero, Port: localPort}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, nil, fmt.Errorf("listen udp: %w", err)
	}

	stunAddr, err := net.ResolveUDPAddr("udp", stunServer)
	if err != nil {
		conn.Close()
		return nil, nil, err
	}

	natInfo, err := stunBindingRequest(conn, stunAddr)
	if err != nil {
		conn.Close()
		return nil, nil, fmt.Errorf("stun query: %w", err)
	}

	natInfo.LocalAddr = conn.LocalAddr().(*net.UDPAddr)
	return natInfo, conn, nil
}

func stunBindingRequest(conn *net.UDPConn, serverAddr *net.UDPAddr) (*NATInfo, error) {
	msg := stun.MustBuild(stun.TransactionID, stun.BindingRequest)
	msg.Encode()

	if _, err := conn.WriteTo(msg.Raw, serverAddr); err != nil {
		return nil, fmt.Errorf("send stun request: %w", err)
	}

	buf := make([]byte, 1024)
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	n, _, err := conn.ReadFromUDP(buf)
	if err != nil {
		return nil, fmt.Errorf("read stun response: %w", err)
	}

	res := &stun.Message{Raw: buf[:n]}
	if err := res.Decode(); err != nil {
		return nil, fmt.Errorf("decode stun response: %w", err)
	}

	var xorAddr stun.XORMappedAddress
	if err := xorAddr.GetFrom(res); err != nil {
		return nil, fmt.Errorf("get xor mapped address: %w", err)
	}

	return &NATInfo{
		PublicAddr: &net.UDPAddr{IP: xorAddr.IP, Port: xorAddr.Port},
	}, nil
}

func HolePunch(conn *net.UDPConn, peerAddrs []string) error {
	var resolvedAddrs []*net.UDPAddr

	for _, addr := range peerAddrs {
		udpAddr, err := net.ResolveUDPAddr("udp", addr)
		if err != nil {
			log.Printf("HolePunch: skip unresolvable addr %s: %v", addr, err)
			continue
		}
		resolvedAddrs = append(resolvedAddrs, udpAddr)
	}

	if len(resolvedAddrs) == 0 {
		return fmt.Errorf("no resolvable peer addresses")
	}

	for i := 0; i < holePunchRetry; i++ {
		for _, addr := range resolvedAddrs {
			conn.WriteTo([]byte("ycair-punch"), addr)
		}
		time.Sleep(holePunchDelay)
	}

	return nil
}
