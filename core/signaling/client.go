package signaling

import (
	"fmt"
	"log"
	"net/url"
	"strings"
	"sync"

	"ycair.online/core/crypto"

	"github.com/gorilla/websocket"
)

type Message struct {
	Type       string   `json:"type"`
	Room       string   `json:"room,omitempty"`
	PassHash   string   `json:"pass_hash,omitempty"`
	PubKey     string   `json:"pub_key,omitempty"`
	Endpoints  []string `json:"endpoints,omitempty"`
	PeerID     string   `json:"peer_id,omitempty"`
	Peer       *Peer    `json:"peer,omitempty"`
	ID         string   `json:"id,omitempty"`
	Error      string   `json:"error,omitempty"`
	AssignedIP string   `json:"assigned_ip,omitempty"`
	Salt       string   `json:"salt,omitempty"`
}

type Peer struct {
	ID        string   `json:"id"`
	Endpoints []string `json:"endpoints"`
	IP        string   `json:"ip"`
}

type EventType int

const (
	EventPeerJoined EventType = iota
	EventPeerLeft
	EventError
)

type Event struct {
	Type  EventType
	Peer  *Peer
	Error string
}

type Client struct {
	conn       *websocket.Conn
	peerID     string
	assignedIP string
	salt       string
	events     chan Event
	done       chan struct{}
	welcomeCh  chan struct{}
	mu         sync.Mutex
}

func Connect(serverAddr, room, password string, localEndpoints []string) (*Client, error) {
	scheme := "wss"
	host := serverAddr

	if strings.Contains(host, "localhost") || strings.Contains(host, "127.0.0.1") || strings.HasPrefix(host, "192.168.") || strings.HasPrefix(host, "10.") {
		scheme = "ws"
	}

	if u, err := url.Parse(serverAddr); err == nil && u.Scheme != "" && u.Host != "" {
		scheme = u.Scheme
		host = u.Host
		if u.Path != "" {
			host += u.Path
		}
		if scheme == "https" {
			scheme = "wss"
		}
	}

	u := url.URL{Scheme: scheme, Host: host, Path: "/ws"}

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("dial signaling server: %w", err)
	}

	c := &Client{
		conn:      conn,
		events:    make(chan Event, 64),
		done:      make(chan struct{}),
		welcomeCh: make(chan struct{}),
	}

	if err := conn.WriteJSON(Message{Type: "salt_request", Room: room}); err != nil {
		conn.Close()
		return nil, fmt.Errorf("send salt request: %w", err)
	}

	var saltMsg Message
	if err := conn.ReadJSON(&saltMsg); err != nil {
		conn.Close()
		return nil, fmt.Errorf("read salt response: %w", err)
	}
	if saltMsg.Type == "error" {
		conn.Close()
		return nil, fmt.Errorf("salt error: %s", saltMsg.Error)
	}

	salt := saltMsg.Salt
	pubKey, _, err := crypto.DeriveKeyPair(password, salt)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("derive key: %w", err)
	}

	registerMsg := Message{
		Type:      "register",
		Room:      room,
		PubKey:    pubKey,
		Endpoints: localEndpoints,
	}

	if err := conn.WriteJSON(registerMsg); err != nil {
		conn.Close()
		return nil, fmt.Errorf("send register: %w", err)
	}

	go c.readLoop()

	return c, nil
}

func (c *Client) readLoop() {
	defer close(c.done)

	for {
		var msg Message
		if err := c.conn.ReadJSON(&msg); err != nil {
			c.events <- Event{Type: EventError, Error: fmt.Sprintf("read error: %v", err)}
			return
		}

		switch msg.Type {
		case "welcome":
			c.mu.Lock()
			c.peerID = msg.PeerID
			c.assignedIP = msg.AssignedIP
			c.salt = msg.Salt
			c.mu.Unlock()
			close(c.welcomeCh)
			log.Printf("Signaling: registered as %s, assigned IP %s", msg.PeerID, msg.AssignedIP)

		case "peer_joined":
			if msg.Peer != nil {
				c.events <- Event{Type: EventPeerJoined, Peer: msg.Peer}
			}

		case "peer_left":
			c.events <- Event{Type: EventPeerLeft, Peer: &Peer{ID: msg.ID}}

		case "error":
			c.events <- Event{Type: EventError, Error: msg.Error}
		}
	}
}

func (c *Client) PeerID() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.peerID
}

func (c *Client) AssignedIP() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.assignedIP
}

func (c *Client) Salt() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.salt
}

func (c *Client) WaitForWelcome() {
	<-c.welcomeCh
}

func (c *Client) Events() <-chan Event {
	return c.events
}

func (c *Client) Close() {
	c.conn.Close()
	<-c.done
}
