package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Message struct {
	Type      string   `json:"type"`
	Room      string   `json:"room,omitempty"`
	PassHash  string   `json:"pass_hash,omitempty"`
	Endpoints []string `json:"endpoints,omitempty"`
	PeerID    string   `json:"peer_id,omitempty"`
	Peer      *Peer    `json:"peer,omitempty"`
	ID        string   `json:"id,omitempty"`
	Error     string   `json:"error,omitempty"`
	AssignedIP string  `json:"assigned_ip,omitempty"`
}

type Peer struct {
	ID        string   `json:"id"`
	Endpoints []string `json:"endpoints"`
	IP        string   `json:"ip"`
}

type Client struct {
	conn     *websocket.Conn
	peerID   string
	room     string
	endpoints []string
	ip       string
	send     chan []byte
}

type Room struct {
	passHash string
	clients  map[string]*Client
	nextIP   int
}

type Server struct {
	rooms map[string]*Room
	mu    sync.RWMutex
}

func (s *Server) getOrCreateRoom(name, passHash string) *Room {
	s.mu.Lock()
	defer s.mu.Unlock()

	if room, exists := s.rooms[name]; exists {
		if room.passHash != passHash {
			return nil
		}
		return room
	}

	s.rooms[name] = &Room{
		passHash: passHash,
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
	if !exists {
		return
	}

	delete(room.clients, client.peerID)

	if len(room.clients) == 0 {
		delete(s.rooms, client.room)
		return
	}

	leaveMsg := Message{
		Type: "peer_left",
		ID:   client.peerID,
	}
	s.broadcast(room, leaveMsg, client.peerID)
}

func (s *Server) broadcast(room *Room, msg Message, excludeID string) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	for id, client := range room.clients {
		if id == excludeID {
			continue
		}
		select {
		case client.send <- data:
		default:
		}
	}
}

func (s *Server) handleRegister(client *Client, msg Message) {
	if msg.Room == "" || msg.PassHash == "" {
		client.send <- mustMarshal(Message{Type: "error", Error: "room and pass_hash required"})
		return
	}

	room := s.getOrCreateRoom(msg.Room, msg.PassHash)
	if room == nil {
		client.send <- mustMarshal(Message{Type: "error", Error: "wrong password"})
		return
	}

	client.room = msg.Room
	client.endpoints = msg.Endpoints
	client.ip = s.assignIP(room)
	client.peerID = generatePeerID()

	newPeer := &Peer{
		ID:        client.peerID,
		Endpoints: client.endpoints,
		IP:        client.ip,
	}
	joinMsg := Message{
		Type: "peer_joined",
		Peer: newPeer,
	}
	s.broadcast(room, joinMsg, client.peerID)

	s.mu.Lock()
	room.clients[client.peerID] = client
	s.mu.Unlock()

	var peers []Peer
	for _, c := range room.clients {
		if c.peerID != client.peerID {
			peers = append(peers, Peer{
				ID:        c.peerID,
				Endpoints: c.endpoints,
				IP:        c.ip,
			})
		}
	}

	client.send <- mustMarshal(Message{
		Type:       "welcome",
		PeerID:     client.peerID,
		AssignedIP: client.ip,
	})

		if len(peers) > 0 {
		client.send <- mustMarshal(Message{
			Type: "peers",
			Peer: &Peer{ID: client.peerID},
		})
		for _, p := range peers {
			client.send <- mustMarshal(Message{
				Type: "peer_joined",
				Peer: &Peer{ID: p.ID, Endpoints: p.Endpoints, IP: p.IP},
			})
		}
	}
}

func handleWebSocket(server *Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("WebSocket upgrade failed: %v", err)
			return
		}

		client := &Client{
			conn: conn,
			send: make(chan []byte, 64),
		}

		go client.writePump()
		client.readPump(server)
	}
}

func (c *Client) readPump(server *Server) {
	defer func() {
		server.removeClient(c)
		c.conn.Close()
	}()

	c.conn.SetReadDeadline(time.Now().Add(90 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(90 * time.Second))
		return nil
	})

	for {
		_, raw, err := c.conn.ReadMessage()
		if err != nil {
			break
		}

		var msg Message
		if err := json.Unmarshal(raw, &msg); err != nil {
			continue
		}

		switch msg.Type {
		case "register":
			server.handleRegister(c, msg)
		case "heartbeat":
			c.send <- mustMarshal(Message{Type: "heartbeat_ack"})
		case "leave":
			return
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func generatePeerID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)[:12]
}

func mustMarshal(msg Message) []byte {
	data, _ := json.Marshal(msg)
	return data
}

func hashPassword(password string) string {
	h := sha256.Sum256([]byte(password))
	return hex.EncodeToString(h[:])
}

func main() {
	port := flag.Int("port", 9090, "signaling server port")
	flag.Parse()

	server := &Server{
		rooms: make(map[string]*Room),
	}

	http.HandleFunc("/ws", handleWebSocket(server))
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	addr := fmt.Sprintf(":%d", *port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", addr, err)
	}

	log.Printf("Signaling server running on ws://localhost%s/ws", addr)
	log.Fatal(http.Serve(listener, nil))
}
