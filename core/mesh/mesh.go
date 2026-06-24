package mesh

import (
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"ycair.online/core/p2p"
	"ycair.online/core/tun"
)

const MTU = 1500

type Mesh struct {
	tun     *tun.Interface
	connMgr *p2p.ConnectionManager
	peers   map[string]*p2p.PeerConnection
	routes  map[string]string
	mu      sync.RWMutex
}

func New(t *tun.Interface, cm *p2p.ConnectionManager) *Mesh {
	return &Mesh{
		tun:     t,
		connMgr: cm,
		peers:   make(map[string]*p2p.PeerConnection),
		routes:  make(map[string]string),
	}
}

func (m *Mesh) Start() {
	go m.forwardLoop()
	go m.receiveLoop()
	go m.syncLoop()

	log.Printf("Mesh: started, local IP %s on %s", m.tun.IP(), m.tun.Name())
}

func (m *Mesh) forwardLoop() {
	buf := make([]byte, MTU)
	udpConn := m.connMgr.UDPConn()

	for {
		n, err := m.tun.Read(buf)
		if err != nil {
			log.Printf("Mesh: tun read error: %v", err)
			return
		}

		packet := buf[:n]
		dstIP := extractDstIP(packet)
		if dstIP == "" {
			continue
		}

		peer := m.lookupPeer(dstIP)
		if peer == nil || peer.Channel == nil {
			continue
		}

		encrypted, err := peer.Channel.Encrypt(packet)
		if err != nil {
			log.Printf("Mesh: encrypt error: %v", err)
			continue
		}

		if _, err := udpConn.WriteTo(encrypted, peer.PublicAddr); err != nil {
			log.Printf("Mesh: send to %s error: %v", peer.PeerID, err)
		}
	}
}

func (m *Mesh) receiveLoop() {
	buf := make([]byte, MTU+256)
	udpConn := m.connMgr.UDPConn()

	for {
		m.connMgr.LockRead()
		udpConn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
		n, srcAddr, err := udpConn.ReadFromUDP(buf)
		m.connMgr.UnlockRead()

		if err != nil || n == 0 {
			continue
		}

		encrypted := buf[:n]
		peers := m.connMgr.GetPeers()

		for _, peer := range peers {
			if peer.Channel == nil || peer.PublicAddr == nil {
				continue
			}
			if !addrMatch(srcAddr, peer.PublicAddr) {
				continue
			}

			plaintext, err := peer.Channel.Decrypt(encrypted)
			if err != nil {
				continue
			}

			if _, err := m.tun.Write(plaintext); err != nil {
				log.Printf("Mesh: tun write error: %v", err)
			}
			break
		}
	}
}

func (m *Mesh) syncLoop() {
	for {
		peers := m.connMgr.GetPeers()

		m.mu.Lock()
		for _, p := range peers {
			m.peers[p.PeerID] = p
			if p.AssignedIP != "" {
				m.routes[p.AssignedIP] = p.PeerID
			}
		}
		m.mu.Unlock()

		time.Sleep(1 * time.Second)
	}
}

func (m *Mesh) lookupPeer(dstIP string) *p2p.PeerConnection {
	m.mu.RLock()
	peerID, ok := m.routes[dstIP]
	m.mu.RUnlock()
	if !ok {
		return nil
	}

	m.mu.RLock()
	peer := m.peers[peerID]
	m.mu.RUnlock()
	return peer
}

func extractDstIP(packet []byte) string {
	if len(packet) < 20 {
		return ""
	}

	version := packet[0] >> 4
	if version != 4 {
		return ""
	}

	return fmt.Sprintf("%d.%d.%d.%d",
		packet[16], packet[17], packet[18], packet[19])
}

func addrMatch(a, b *net.UDPAddr) bool {
	if a == nil || b == nil {
		return false
	}
	return a.IP.Equal(b.IP) && a.Port == b.Port
}
