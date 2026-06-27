package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Message struct {
	Type       string   `json:"type"`
	Room       string   `json:"room,omitempty"`
	CredHash   string   `json:"cred_hash,omitempty"`
	Endpoints  []string `json:"endpoints,omitempty"`
	PeerID     string   `json:"peer_id,omitempty"`
	Peer       *Peer    `json:"peer,omitempty"`
	ID         string   `json:"id,omitempty"`
	Error      string   `json:"error,omitempty"`
	AssignedIP string   `json:"assigned_ip,omitempty"`
	Salt       string   `json:"salt,omitempty"`
	Token      string   `json:"token,omitempty"`
}

type Peer struct {
	ID        string   `json:"id"`
	Endpoints []string `json:"endpoints"`
	IP        string   `json:"ip"`
}

type Client struct {
	conn      *websocket.Conn
	peerID    string
	room      string
	endpoints []string
	ip        string
	send      chan []byte
}

type Room struct {
	credHash string
	salt     string
	clients  map[string]*Client
	nextIP   int
}

type Server struct {
	rooms     map[string]*Room
	rateLimit map[string][]time.Time
	serverPri ed25519.PrivateKey
	mu        sync.RWMutex
}

const (
	maxRoomNameLen  = 64
	maxCredHashLen  = 128
	maxRooms        = 1000
	maxPeersPerRoom = 253
	rateLimitWindow = 10 * time.Second
	rateLimitMax    = 5
)

func loadServerKey() (ed25519.PrivateKey, error) {
	seed, err := os.ReadFile(os.ExpandEnv("$HOME/.ycair-server-key"))
	if err != nil {
		return nil, fmt.Errorf("read server key: %w", err)
	}
	return ed25519.NewKeyFromSeed(seed), nil
}

func (s *Server) checkRateLimit(ip string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	cutoff := now.Add(-rateLimitWindow)
	recent := make([]time.Time, 0)
	for _, t := range s.rateLimit[ip] {
		if t.After(cutoff) {
			recent = append(recent, t)
		}
	}
	recent = append(recent, now)
	s.rateLimit[ip] = recent
	return len(recent) <= rateLimitMax
}

func (s *Server) getOrCreateRoom(name, credHash string) *Room {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.rooms) >= maxRooms {
		return nil
	}
	if room, exists := s.rooms[name]; exists {
		if room.credHash != credHash {
			return nil
		}
		return room
	}
	s.rooms[name] = &Room{
		credHash: credHash,
		salt:     generateSalt(),
		clients:  make(map[string]*Client),
		nextIP:   2,
	}
	return s.rooms[name]
}

func (s *Server) assignIP(room *Room) string {
	ip := fmt.Sprintf("10.99.0.%d", room.nextIP)
	room.nextIP++
	return ip
}

func (s *Server) removeClient(client *Client) {
	s.mu.Lock()
	defer s.mu.Unlock()
	room, exists := s.rooms[client.room]
	if !exists { return }
	delete(room.clients, client.peerID)
	if len(room.clients) == 0 {
		delete(s.rooms, client.room)
		return
	}
	s.broadcast(room, Message{Type: "peer_left", ID: client.peerID}, client.peerID)
}

func (s *Server) broadcast(room *Room, msg Message, excludeID string) {
	data, _ := json.Marshal(msg)
	for id, client := range room.clients {
		if id == excludeID { continue }
		select { case client.send <- data: default: }
	}
}

func (s *Server) signToken(room, credHash string) string {
	expiry := time.Now().Add(24 * time.Hour).Unix()
	payload := fmt.Sprintf("%s:%s:%d", room, credHash, expiry)
	sig := ed25519.Sign(s.serverPri, []byte(payload))
	return payload + ":" + base64.StdEncoding.EncodeToString(sig)
}

func (s *Server) handleSaltRequest(client *Client, msg Message) {
	if len(msg.Room) == 0 || len(msg.Room) > maxRoomNameLen {
		client.send <- mustMarshal(Message{Type: "error", Error: "invalid room code"})
		return
	}
	s.mu.RLock()
	room, exists := s.rooms[msg.Room]
	s.mu.RUnlock()
	if !exists {
		client.send <- mustMarshal(Message{Type: "salt", Salt: generateSalt()})
		return
	}
	client.send <- mustMarshal(Message{Type: "salt", Salt: room.salt})
}

func (s *Server) handleRegister(client *Client, msg Message) {
	if len(msg.Room) == 0 || len(msg.Room) > maxRoomNameLen {
		client.send <- mustMarshal(Message{Type: "error", Error: "invalid room code"})
		return
	}
	if len(msg.CredHash) == 0 || len(msg.CredHash) > maxCredHashLen {
		client.send <- mustMarshal(Message{Type: "error", Error: "invalid credentials"})
		return
	}
	ip, _, _ := net.SplitHostPort(client.conn.RemoteAddr().String())
	if !s.checkRateLimit(ip) {
		client.send <- mustMarshal(Message{Type: "error", Error: "too many attempts"})
		return
	}
	room := s.getOrCreateRoom(msg.Room, msg.CredHash)
	if room == nil {
		client.send <- mustMarshal(Message{Type: "error", Error: "invalid room or password"})
		return
	}
	if len(room.clients) >= maxPeersPerRoom {
		client.send <- mustMarshal(Message{Type: "error", Error: "room is full"})
		return
	}
	client.room = msg.Room
	client.endpoints = msg.Endpoints
	client.ip = s.assignIP(room)
	client.peerID = generatePeerID()

	s.broadcast(room, Message{Type: "peer_joined", Peer: &Peer{
		ID: client.peerID, Endpoints: client.endpoints, IP: client.ip,
	}}, client.peerID)

	s.mu.Lock()
	room.clients[client.peerID] = client
	s.mu.Unlock()

	var peers []Peer
	for _, c := range room.clients {
		if c.peerID != client.peerID {
			peers = append(peers, Peer{ID: c.peerID, Endpoints: c.endpoints, IP: c.ip})
		}
	}

	client.send <- mustMarshal(Message{
		Type: "welcome", PeerID: client.peerID, AssignedIP: client.ip,
		Salt: room.salt, Token: s.signToken(msg.Room, msg.CredHash),
	})

	if len(peers) > 0 {
		client.send <- mustMarshal(Message{Type: "peers", Peer: &Peer{ID: client.peerID}})
		for _, p := range peers {
			client.send <- mustMarshal(Message{Type: "peer_joined", Peer: &Peer{ID: p.ID, Endpoints: p.Endpoints, IP: p.IP}})
		}
	}
}

func handleWebSocket(server *Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil { log.Printf("WS upgrade failed: %v", err); return }
		client := &Client{conn: conn, send: make(chan []byte, 64)}
		go client.writePump()
		client.readPump(server)
	}
}

func (c *Client) readPump(server *Server) {
	defer func() { server.removeClient(c); c.conn.Close() }()
	c.conn.SetReadDeadline(time.Now().Add(90 * time.Second))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(90 * time.Second)); return nil })
	for {
		_, raw, err := c.conn.ReadMessage()
		if err != nil { break }
		var msg Message
		if json.Unmarshal(raw, &msg) != nil { continue }
		switch msg.Type {
		case "salt_request": server.handleSaltRequest(c, msg)
		case "register": server.handleRegister(c, msg)
		case "heartbeat": c.send <- mustMarshal(Message{Type: "heartbeat_ack"})
		case "leave": return
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() { ticker.Stop(); c.conn.Close() }()
	for {
		select {
		case msg, ok := <-c.send:
			if !ok { c.conn.WriteMessage(websocket.CloseMessage, []byte{}); return }
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if c.conn.WriteMessage(websocket.TextMessage, msg) != nil { return }
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if c.conn.WriteMessage(websocket.PingMessage, nil) != nil { return }
		}
	}
}

func generatePeerID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)[:12]
}

func mustMarshal(msg Message) []byte {
	data, _ := json.Marshal(msg)
	return data
}

func generateSalt() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func main() {
	port := flag.Int("port", 9090, "signaling server port")
	flag.Parse()

	pri, err := loadServerKey()
	if err != nil {
		log.Fatalf("Server key not found at ~/.ycair-server-key: %v", err)
	}
	log.Printf("Signaling server pubKey: %s", base64.StdEncoding.EncodeToString(pri.Public().(ed25519.PublicKey)))

	server := &Server{
		rooms:     make(map[string]*Room),
		rateLimit: make(map[string][]time.Time),
		serverPri: pri,
	}

	http.HandleFunc("/ws", handleWebSocket(server))
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) })

	addr := fmt.Sprintf(":%d", *port)
	listener, err := net.Listen("tcp", addr)
	if err != nil { log.Fatalf("listen: %v", err) }
	log.Printf("Signaling server running on ws://localhost%s/ws", addr)
	log.Fatal(http.Serve(listener, nil))
}
