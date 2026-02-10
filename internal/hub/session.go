package hub

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/time/rate"
)

// Session represents a paired relay session identified by a pairing token.
type Session struct {
	Token     string
	mu        sync.RWMutex
	PhoneConn *Connection
	AgentConn *Connection
	LastActive atomic.Value // stores time.Time
}

// Connection represents a single WebSocket connection (phone or agent).
type Connection struct {
	WS        *websocket.Conn
	Role      string
	Limiter   *rate.Limiter
	BytesSent atomic.Int64
	BytesRecv atomic.Int64
	LastPing  time.Time
	Send      chan []byte
	Done      chan struct{}
	closeOnce sync.Once
}

// NewConnection creates a new Connection with a send channel.
func NewConnection(ws *websocket.Conn, role string, limiter *rate.Limiter) *Connection {
	return &Connection{
		WS:       ws,
		Role:     role,
		Limiter:  limiter,
		LastPing: time.Now(),
		Send:     make(chan []byte, 256),
		Done:     make(chan struct{}),
	}
}

// CloseDone safely closes the Done channel exactly once.
func (c *Connection) CloseDone() {
	c.closeOnce.Do(func() {
		close(c.Done)
	})
}

// PeerConn returns the peer's connection (phone<->agent).
func (s *Session) PeerConn(role string) *Connection {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if role == "phone" {
		return s.AgentConn
	}
	return s.PhoneConn
}

// SetConn sets the connection for a role.
func (s *Session) SetConn(role string, conn *Connection) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if role == "phone" {
		s.PhoneConn = conn
	} else {
		s.AgentConn = conn
	}
	s.LastActive.Store(time.Now())
}

// ClearConn clears the connection for a role.
func (s *Session) ClearConn(role string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if role == "phone" {
		s.PhoneConn = nil
	} else {
		s.AgentConn = nil
	}
}

// IsPaired returns true if both phone and agent are connected.
func (s *Session) IsPaired() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.PhoneConn != nil && s.AgentConn != nil
}

// SameRoleConn returns the existing connection of the same role (for replacement).
func (s *Session) SameRoleConn(role string) *Connection {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if role == "phone" {
		return s.PhoneConn
	}
	return s.AgentConn
}

// IsIdle returns true if the session has no connections and has been idle for the given duration.
func (s *Session) IsIdle(timeout time.Duration) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.PhoneConn != nil || s.AgentConn != nil {
		return false
	}
	lastActive, ok := s.LastActive.Load().(time.Time)
	if !ok {
		return true // never been active
	}
	return time.Since(lastActive) > timeout
}
