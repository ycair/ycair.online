package p2p

import (
	"fmt"
	"log"
	"net"
	"sync"

	"ycair.online/core/crypto"
	"ycair.online/core/signaling"
)

type PeerConnection struct {
	PeerID     string
	AssignedIP string
	PublicAddr *net.UDPAddr
	Channel    *crypto.SecureChannel
}

type ConnectionManager struct {
	mu          sync.Mutex
	readMu      sync.Mutex
	peers       map[string]*PeerConnection
	natInfo     *NATInfo
	udpConn     *net.UDPConn
	localPort   int
	peerID      string
}

func NewConnectionManager(localPort int) *ConnectionManager {
	return &ConnectionManager{
		peers:     make(map[string]*PeerConnection),
		localPort: localPort,
	}
}

func (cm *ConnectionManager) Start() error {
	natInfo, conn, err := DiscoverNAT(cm.localPort)
	if err != nil {
		return fmt.Errorf("nat discovery: %w", err)
	}

	cm.natInfo = natInfo
	cm.udpConn = conn

	log.Printf("NAT: local=%s public=%s",
		natInfo.LocalAddr.String(),
		natInfo.PublicAddr.String())

	return nil
}

func (cm *ConnectionManager) PublicAddr() string {
	if cm.natInfo == nil {
		return ""
	}
	return cm.natInfo.PublicAddr.String()
}

func (cm *ConnectionManager) LocalPort() int {
	if cm.natInfo == nil || cm.natInfo.LocalAddr == nil {
		return 0
	}
	return cm.natInfo.LocalAddr.Port
}

func (cm *ConnectionManager) SetPeerID(id string) {
	cm.mu.Lock()
	cm.peerID = id
	cm.mu.Unlock()
}

func (cm *ConnectionManager) UDPConn() *net.UDPConn {
	return cm.udpConn
}

func (cm *ConnectionManager) SafeReadFrom(b []byte) (int, *net.UDPAddr, error) {
	cm.readMu.Lock()
	defer cm.readMu.Unlock()
	return cm.udpConn.ReadFromUDP(b)
}

func (cm *ConnectionManager) LockRead() {
	cm.readMu.Lock()
}

func (cm *ConnectionManager) UnlockRead() {
	cm.readMu.Unlock()
}

func (cm *ConnectionManager) HandleSignalingEvent(event signaling.Event) {
	switch event.Type {
	case signaling.EventPeerJoined:
		go cm.addPeer(event.Peer)
	case signaling.EventPeerLeft:
		cm.removePeer(event.Peer.ID)
	}
}

func (cm *ConnectionManager) addPeer(peer *signaling.Peer) {
	cm.mu.Lock()
	if _, exists := cm.peers[peer.ID]; exists {
		cm.mu.Unlock()
		return
	}
	cm.mu.Unlock()

	pubAddr := cm.resolvePeerAddr(peer.Endpoints)
	if pubAddr == nil {
		log.Printf("P2P: no addr for peer %s", peer.ID)
		return
	}

	log.Printf("P2P: connecting to peer %s at %s", peer.ID, pubAddr)

	cm.LockRead()
	defer cm.UnlockRead()

	err := HolePunch(cm.udpConn, peer.Endpoints)
	if err != nil {
		log.Printf("P2P: hole punch to %s failed: %v", peer.ID, err)
		return
	}

	myID := cm.peerID
	if myID == "" && cm.natInfo != nil {
		myID = cm.natInfo.PublicAddr.String()
	}

	isInitiator := myID < peer.ID

	channel, err := crypto.PerformHandshake(cm.udpConn, pubAddr, isInitiator)
	if err != nil {
		log.Printf("P2P: handshake with %s failed: %v", peer.ID, err)
		return
	}

	pc := &PeerConnection{
		PeerID:     peer.ID,
		AssignedIP: peer.IP,
		PublicAddr: pubAddr,
		Channel:    channel,
	}

	cm.mu.Lock()
	cm.peers[peer.ID] = pc
	cm.mu.Unlock()

	log.Printf("P2P: secure channel established with peer %s (%s)", peer.ID, peer.IP)
}

func (cm *ConnectionManager) removePeer(peerID string) {
	cm.mu.Lock()
	delete(cm.peers, peerID)
	cm.mu.Unlock()
	log.Printf("P2P: removed peer %s", peerID)
}

func (cm *ConnectionManager) GetPeers() []*PeerConnection {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	peers := make([]*PeerConnection, 0, len(cm.peers))
	for _, p := range cm.peers {
		peers = append(peers, p)
	}
	return peers
}

func (cm *ConnectionManager) Close() {
	if cm.udpConn != nil {
		cm.udpConn.Close()
	}
}

func (cm *ConnectionManager) resolvePeerAddr(endpoints []string) *net.UDPAddr {
	isPrivate := func(ip net.IP) bool {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() {
			return true
		}
		if ip4 := ip.To4(); ip4 != nil {
			if ip4[0] == 100 && ip4[1] >= 64 && ip4[1] <= 127 {
				return true
			}
		}
		return false
	}

	for _, ep := range endpoints {
		addr, err := net.ResolveUDPAddr("udp", ep)
		if err == nil && addr.Port != 0 && isPrivate(addr.IP) && addr.IP.Equal(cm.natInfo.LocalAddr.IP) == false {
			return addr
		}
	}

	for _, ep := range endpoints {
		addr, err := net.ResolveUDPAddr("udp", ep)
		if err == nil && addr.Port != 0 && !isPrivate(addr.IP) {
			return addr
		}
	}

	for _, ep := range endpoints {
		addr, err := net.ResolveUDPAddr("udp", ep)
		if err == nil && addr.Port != 0 {
			return addr
		}
	}

	return nil
}
