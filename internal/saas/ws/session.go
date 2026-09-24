package ws

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/agent/protocol"
)

type Session struct {
	AgentID     string
	InstanceIDs []uint
	conn        *websocket.Conn
	sendCh      chan []byte
	hub         *Hub
	mu          sync.Mutex
	lastSeen    time.Time
	closed      bool
}

func newSession(conn *websocket.Conn, hub *Hub, agentID string) *Session {
	return &Session{
		AgentID:  agentID,
		conn:     conn,
		sendCh:   make(chan []byte, 64),
		hub:      hub,
		lastSeen: time.Now(),
	}
}

func (s *Session) Touch() {
	s.mu.Lock()
	s.lastSeen = time.Now()
	s.mu.Unlock()
}

func (s *Session) LastSeen() time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastSeen
}

func (s *Session) Send(msgType protocol.Type, requestID string, payload any) error {
	raw, err := protocol.Encode(msgType, requestID, payload)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ErrNoAgent
	}
	select {
	case s.sendCh <- raw:
		return nil
	default:
		return ErrNoAgent
	}
}

func (s *Session) close() {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	s.mu.Unlock()
	_ = s.conn.Close()
}

func (s *Session) writePump() {
	defer s.close()
	for raw := range s.sendCh {
		_ = s.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		if err := s.conn.WriteMessage(websocket.TextMessage, raw); err != nil {
			return
		}
	}
}

func (s *Session) readPump() {
	defer func() {
		s.hub.Unregister(s.AgentID)
		s.close()
	}()
	_ = s.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	s.conn.SetPongHandler(func(string) error {
		s.Touch()
		_ = s.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})
	for {
		_, data, err := s.conn.ReadMessage()
		if err != nil {
			return
		}
		_ = s.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		s.hub.handleInbound(s, data)
	}
}
