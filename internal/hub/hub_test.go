package hub

import (
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/openclaw/openclaw-relay/internal/protocol"
	"github.com/openclaw/openclaw-relay/internal/store"
)

// mockStore implements store.Store for testing.
type mockStore struct {
	mu        sync.Mutex
	tokens    map[string]bool
	bandwidth map[string]int64
}

func newMockStore() *mockStore {
	return &mockStore{
		tokens:    make(map[string]bool),
		bandwidth: make(map[string]int64),
	}
}

func (s *mockStore) CreateToken(token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tokens[token] = true
	return nil
}

func (s *mockStore) DeleteToken(token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tokens, token)
	return nil
}

func (s *mockStore) TokenExists(token string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.tokens[token], nil
}

func (s *mockStore) ListTokens() ([]store.TokenInfo, error) {
	return nil, nil
}

func (s *mockStore) RecordBytes(token string, bytes int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.bandwidth[token] += bytes
	return nil
}

func (s *mockStore) GetDailyUsage(token string) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.bandwidth[token], nil
}

func (s *mockStore) GetMonthlyUsage(token string) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.bandwidth[token], nil
}

func (s *mockStore) ResetDailyUsage() error { return nil }
func (s *mockStore) Close() error           { return nil }

func TestNewHub(t *testing.T) {
	store := newMockStore()
	h := NewHub(store)
	if h == nil {
		t.Fatal("NewHub returned nil")
	}
	if h.ConnectionCount() != 0 {
		t.Fatalf("expected 0 connections, got %d", h.ConnectionCount())
	}
	if h.SessionCount() != 0 {
		t.Fatalf("expected 0 sessions, got %d", h.SessionCount())
	}
}

func TestCreateAndDeleteToken(t *testing.T) {
	store := newMockStore()
	h := NewHub(store)

	token, err := h.CreateToken()
	if err != nil {
		t.Fatalf("CreateToken failed: %v", err)
	}
	if token == "" {
		t.Fatal("CreateToken returned empty token")
	}

	// Verify token exists in store
	exists, _ := store.TokenExists(token)
	if !exists {
		t.Fatal("token not found in store after creation")
	}

	// Delete token
	if err := h.DeleteToken(token); err != nil {
		t.Fatalf("DeleteToken failed: %v", err)
	}

	// Verify token no longer exists
	exists, _ = store.TokenExists(token)
	if exists {
		t.Fatal("token still exists after deletion")
	}
}

func TestAuthenticateInvalidToken(t *testing.T) {
	store := newMockStore()
	h := NewHub(store)

	conn := NewConnection(nil, "", nil)
	authPayload, _ := json.Marshal(protocol.AuthPayload{Token: "invalid", Role: "phone"})
	env := &protocol.Envelope{
		Type:    protocol.TypeAuth,
		Payload: authPayload,
	}

	_, err := h.Authenticate(conn, env)
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
}

func TestAuthenticateInvalidRole(t *testing.T) {
	store := newMockStore()
	h := NewHub(store)

	token, _ := h.CreateToken()
	conn := NewConnection(nil, "", nil)
	authPayload, _ := json.Marshal(protocol.AuthPayload{Token: token, Role: "hacker"})
	env := &protocol.Envelope{
		Type:    protocol.TypeAuth,
		Payload: authPayload,
	}

	_, err := h.Authenticate(conn, env)
	if err == nil {
		t.Fatal("expected error for invalid role")
	}
}

func TestAuthenticateSuccess(t *testing.T) {
	store := newMockStore()
	h := NewHub(store)

	token, _ := h.CreateToken()
	conn := NewConnection(nil, "", nil)
	authPayload, _ := json.Marshal(protocol.AuthPayload{Token: token, Role: "phone"})
	env := &protocol.Envelope{
		Type:    protocol.TypeAuth,
		Payload: authPayload,
	}

	session, err := h.Authenticate(conn, env)
	if err != nil {
		t.Fatalf("Authenticate failed: %v", err)
	}
	if session == nil {
		t.Fatal("Authenticate returned nil session")
	}
	if session.Token != token {
		t.Fatalf("session token mismatch: got %s, want %s", session.Token, token)
	}
	if h.ConnectionCount() != 1 {
		t.Fatalf("expected 1 connection, got %d", h.ConnectionCount())
	}
}

func TestDisconnect(t *testing.T) {
	store := newMockStore()
	h := NewHub(store)

	token, _ := h.CreateToken()
	conn := NewConnection(nil, "", nil)
	authPayload, _ := json.Marshal(protocol.AuthPayload{Token: token, Role: "phone"})
	env := &protocol.Envelope{
		Type:    protocol.TypeAuth,
		Payload: authPayload,
	}

	session, _ := h.Authenticate(conn, env)
	h.Disconnect(session, conn)

	if h.ConnectionCount() != 0 {
		t.Fatalf("expected 0 connections after disconnect, got %d", h.ConnectionCount())
	}
}

func TestGetTokenStatus(t *testing.T) {
	store := newMockStore()
	h := NewHub(store)

	token, _ := h.CreateToken()

	// No connections yet
	phone, agent := h.GetTokenStatus(token)
	if phone || agent {
		t.Fatal("expected both offline before auth")
	}

	// Connect phone
	conn := NewConnection(nil, "", nil)
	authPayload, _ := json.Marshal(protocol.AuthPayload{Token: token, Role: "phone"})
	env := &protocol.Envelope{Type: protocol.TypeAuth, Payload: authPayload}
	h.Authenticate(conn, env)

	phone, agent = h.GetTokenStatus(token)
	if !phone {
		t.Fatal("expected phone online after auth")
	}
	if agent {
		t.Fatal("expected agent offline")
	}
}

func TestIdleSessionCleanup(t *testing.T) {
	store := newMockStore()
	h := NewHub(store)

	token, _ := h.CreateToken()

	// Manually create a session
	h.mu.Lock()
	session := &Session{Token: token}
	session.LastActive.Store(time.Now().Add(-31 * time.Minute)) // idle for 31 minutes
	h.sessions[token] = session
	h.mu.Unlock()

	if h.SessionCount() != 1 {
		t.Fatalf("expected 1 session, got %d", h.SessionCount())
	}

	h.cleanIdleSessions()

	if h.SessionCount() != 0 {
		t.Fatalf("expected 0 sessions after cleanup, got %d", h.SessionCount())
	}
}

func TestConcurrentAuth(t *testing.T) {
	store := newMockStore()
	h := NewHub(store)

	token, _ := h.CreateToken()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			role := "phone"
			if i%2 == 0 {
				role = "agent"
			}
			conn := NewConnection(nil, "", nil)
			authPayload, _ := json.Marshal(protocol.AuthPayload{Token: token, Role: role})
			env := &protocol.Envelope{Type: protocol.TypeAuth, Payload: authPayload}
			h.Authenticate(conn, env)
		}(i)
	}
	wg.Wait()
}
