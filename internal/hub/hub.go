package hub

import (
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/openclaw/openclaw-relay/internal/protocol"
	"github.com/openclaw/openclaw-relay/internal/ratelimit"
	"github.com/openclaw/openclaw-relay/internal/store"
)

const (
	// IdleSessionTimeout is how long a session with no connections stays in memory.
	IdleSessionTimeout = 30 * time.Minute
	// idleCleanupInterval is how often we scan for idle sessions.
	idleCleanupInterval = 5 * time.Minute
)

// Hub manages all sessions and routes messages between paired connections.
type Hub struct {
	mu           sync.RWMutex
	sessions     map[string]*Session
	store        store.Store
	quotaChecker *ratelimit.QuotaChecker
	connCount    atomic.Int64
	startTime    time.Time
}

func NewHub(s store.Store) *Hub {
	h := &Hub{
		sessions:     make(map[string]*Session),
		store:        s,
		quotaChecker: ratelimit.NewQuotaChecker(s),
		startTime:    time.Now(),
	}
	go h.idleCleanupLoop()
	return h
}

func (h *Hub) ConnectionCount() int64 {
	return h.connCount.Load()
}

// StartTime returns when the hub was created.
func (h *Hub) StartTime() time.Time {
	return h.startTime
}

// SessionCount returns the number of active sessions.
func (h *Hub) SessionCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.sessions)
}

func (h *Hub) Authenticate(conn *Connection, env *protocol.Envelope) (*Session, error) {
	var auth protocol.AuthPayload
	if err := env.ParsePayload(&auth); err != nil {
		return nil, err
	}

	exists, err := h.store.TokenExists(auth.Token)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("invalid token")
	}

	if auth.Role != protocol.RolePhone && auth.Role != protocol.RoleAgent {
		return nil, fmt.Errorf("invalid role: %s", auth.Role)
	}

	conn.Role = auth.Role

	if auth.Role == protocol.RolePhone {
		conn.Limiter = ratelimit.NewPhoneLimiter()
	} else {
		conn.Limiter = ratelimit.NewAgentLimiter()
	}

	h.mu.Lock()
	session, ok := h.sessions[auth.Token]
	if !ok {
		session = &Session{Token: auth.Token}
		h.sessions[auth.Token] = session
	}
	h.mu.Unlock()

	// Close existing connection of same role (use sync.Once safe close)
	existing := session.SameRoleConn(auth.Role)
	if existing != nil {
		existing.CloseDone()
	}

	session.SetConn(auth.Role, conn)
	h.connCount.Add(1)

	log.Printf("auth: token=%s role=%s paired=%v", auth.Token[:16]+"...", auth.Role, session.IsPaired())

	return session, nil
}

func (h *Hub) Disconnect(session *Session, conn *Connection) {
	if session == nil || conn == nil {
		return
	}
	session.ClearConn(conn.Role)
	h.connCount.Add(-1)

	peer := session.PeerConn(conn.Role)
	if peer != nil {
		env, err := protocol.NewEnvelope(protocol.TypeStatus, protocol.StatusPayload{Peer: "offline"})
		if err != nil {
			log.Printf("error creating offline status: %v", err)
			return
		}
		data, err := env.Marshal()
		if err != nil {
			log.Printf("error marshaling offline status: %v", err)
			return
		}
		select {
		case peer.Send <- data:
		default:
		}
	}

	log.Printf("disconnect: token=%s role=%s", session.Token[:min(16, len(session.Token))]+"...", conn.Role)
}

func (h *Hub) ForwardMessage(session *Session, sender *Connection, raw []byte) error {
	if len(raw) > ratelimit.MaxMessageSize {
		return h.sendError(sender, protocol.ErrMessageTooLarge, "Message exceeds 5MB limit", 0)
	}

	if sender.Limiter != nil && !sender.Limiter.Allow() {
		return h.sendError(sender, protocol.ErrRateLimited, "Rate limit exceeded", 1000)
	}

	code, err := h.quotaChecker.Check(session.Token)
	if err != nil {
		log.Printf("quota check error for %s: %v", session.Token, err)
	}
	if code != "" {
		return h.sendError(sender, code, "Bandwidth quota exceeded", 60000)
	}

	peer := session.PeerConn(sender.Role)
	if peer == nil {
		return h.sendError(sender, protocol.ErrPeerOffline, "Peer is not connected", 0)
	}

	msgSize := int64(len(raw))

	select {
	case peer.Send <- raw:
		// Record quota and stats only after successful send
		if err := h.quotaChecker.Record(session.Token, msgSize); err != nil {
			log.Printf("quota record error: %v", err)
		}
		sender.BytesSent.Add(msgSize)
		peer.BytesRecv.Add(msgSize)
		return nil
	default:
		return h.sendError(sender, protocol.ErrPeerOffline, "Peer send buffer full", 1000)
	}
}

func (h *Hub) sendError(conn *Connection, code, message string, retryMs int64) error {
	env, err := protocol.NewEnvelope(protocol.TypeChatError, protocol.ErrorPayload{
		Code:         code,
		Message:      message,
		RetryAfterMs: retryMs,
	})
	if err != nil {
		log.Printf("error creating error envelope: %v", err)
		return err
	}
	data, err := env.Marshal()
	if err != nil {
		log.Printf("error marshaling error envelope: %v", err)
		return err
	}
	select {
	case conn.Send <- data:
	default:
	}
	return nil
}

func (h *Hub) CreateToken() (string, error) {
	token := GenerateToken()
	if err := h.store.CreateToken(token); err != nil {
		return "", err
	}
	log.Printf("token created: %s...", token[:16])
	return token, nil
}

func (h *Hub) DeleteToken(token string) error {
	h.mu.Lock()
	if session, ok := h.sessions[token]; ok {
		session.mu.RLock()
		phone := session.PhoneConn
		agent := session.AgentConn
		session.mu.RUnlock()
		if phone != nil {
			phone.CloseDone()
		}
		if agent != nil {
			agent.CloseDone()
		}
		delete(h.sessions, token)
	}
	h.mu.Unlock()
	log.Printf("token deleted: %s...", token[:min(16, len(token))])
	return h.store.DeleteToken(token)
}

func (h *Hub) GetTokenStatus(token string) (phoneOnline, agentOnline bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if session, ok := h.sessions[token]; ok {
		session.mu.RLock()
		defer session.mu.RUnlock()
		return session.PhoneConn != nil, session.AgentConn != nil
	}
	return false, false
}

// idleCleanupLoop periodically removes sessions with no connections that have been idle.
func (h *Hub) idleCleanupLoop() {
	ticker := time.NewTicker(idleCleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		h.cleanIdleSessions()
	}
}

func (h *Hub) cleanIdleSessions() {
	h.mu.Lock()
	defer h.mu.Unlock()

	var removed int
	for token, session := range h.sessions {
		if session.IsIdle(IdleSessionTimeout) {
			delete(h.sessions, token)
			removed++
		}
	}
	if removed > 0 {
		log.Printf("cleaned %d idle sessions, %d remaining", removed, len(h.sessions))
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
