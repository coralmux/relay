package hub

import (
	"sync"
	"testing"
	"time"
)

func TestNewConnection(t *testing.T) {
	conn := NewConnection(nil, "phone", nil)
	if conn == nil {
		t.Fatal("NewConnection returned nil")
	}
	if conn.Role != "phone" {
		t.Fatalf("expected role 'phone', got '%s'", conn.Role)
	}
	if conn.Send == nil {
		t.Fatal("Send channel is nil")
	}
	if conn.Done == nil {
		t.Fatal("Done channel is nil")
	}
}

func TestCloseDoneOnce(t *testing.T) {
	conn := NewConnection(nil, "phone", nil)

	// Should not panic even if called multiple times
	conn.CloseDone()
	conn.CloseDone()
	conn.CloseDone()

	// Verify the channel is closed
	select {
	case <-conn.Done:
		// OK - channel is closed
	default:
		t.Fatal("Done channel should be closed")
	}
}

func TestCloseDoneConcurrent(t *testing.T) {
	conn := NewConnection(nil, "phone", nil)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			conn.CloseDone()
		}()
	}
	wg.Wait()

	// Verify the channel is closed
	select {
	case <-conn.Done:
		// OK
	default:
		t.Fatal("Done channel should be closed")
	}
}

func TestBytesSentAtomic(t *testing.T) {
	conn := NewConnection(nil, "phone", nil)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			conn.BytesSent.Add(10)
		}()
	}
	wg.Wait()

	if conn.BytesSent.Load() != 1000 {
		t.Fatalf("expected BytesSent=1000, got %d", conn.BytesSent.Load())
	}
}

func TestBytesRecvAtomic(t *testing.T) {
	conn := NewConnection(nil, "agent", nil)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			conn.BytesRecv.Add(5)
		}()
	}
	wg.Wait()

	if conn.BytesRecv.Load() != 500 {
		t.Fatalf("expected BytesRecv=500, got %d", conn.BytesRecv.Load())
	}
}

func TestSessionSetAndGetConn(t *testing.T) {
	session := &Session{Token: "test"}
	conn := NewConnection(nil, "phone", nil)

	session.SetConn("phone", conn)
	got := session.PeerConn("agent") // agent's peer is phone
	if got != conn {
		t.Fatal("PeerConn(agent) should return the phone connection")
	}

	got = session.SameRoleConn("phone")
	if got != conn {
		t.Fatal("SameRoleConn(phone) should return the phone connection")
	}

	got = session.SameRoleConn("agent")
	if got != nil {
		t.Fatal("SameRoleConn(agent) should return nil")
	}
}

func TestSessionIsPaired(t *testing.T) {
	session := &Session{Token: "test"}

	if session.IsPaired() {
		t.Fatal("should not be paired with no connections")
	}

	phone := NewConnection(nil, "phone", nil)
	session.SetConn("phone", phone)
	if session.IsPaired() {
		t.Fatal("should not be paired with only phone")
	}

	agent := NewConnection(nil, "agent", nil)
	session.SetConn("agent", agent)
	if !session.IsPaired() {
		t.Fatal("should be paired with both connections")
	}
}

func TestSessionClearConn(t *testing.T) {
	session := &Session{Token: "test"}
	phone := NewConnection(nil, "phone", nil)
	agent := NewConnection(nil, "agent", nil)

	session.SetConn("phone", phone)
	session.SetConn("agent", agent)

	session.ClearConn("phone")
	if session.SameRoleConn("phone") != nil {
		t.Fatal("phone should be nil after ClearConn")
	}
	if session.SameRoleConn("agent") != agent {
		t.Fatal("agent should still be set")
	}
}

func TestSessionIsIdle(t *testing.T) {
	session := &Session{Token: "test"}

	// Never active - should be idle
	if !session.IsIdle(time.Minute) {
		t.Fatal("session with no LastActive should be idle")
	}

	// Set active now
	session.LastActive.Store(time.Now())
	if session.IsIdle(time.Minute) {
		t.Fatal("session just made active should not be idle")
	}

	// Set active 2 hours ago
	session.LastActive.Store(time.Now().Add(-2 * time.Hour))
	if !session.IsIdle(time.Hour) {
		t.Fatal("session idle for 2 hours should be idle with 1 hour timeout")
	}

	// With a connection, should never be idle
	conn := NewConnection(nil, "phone", nil)
	session.SetConn("phone", conn)
	session.LastActive.Store(time.Now().Add(-2 * time.Hour))
	if session.IsIdle(time.Hour) {
		t.Fatal("session with active connection should not be idle")
	}
}

func TestSessionConcurrentAccess(t *testing.T) {
	session := &Session{Token: "test"}

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			conn := NewConnection(nil, "phone", nil)
			session.SetConn("phone", conn)
		}()
		go func() {
			defer wg.Done()
			session.PeerConn("agent")
			session.IsPaired()
			session.SameRoleConn("phone")
		}()
	}
	wg.Wait()
}
