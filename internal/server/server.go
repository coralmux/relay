package server

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"log"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/openclaw/openclaw-relay/internal/hub"
	"golang.org/x/crypto/acme/autocert"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// Config holds server configuration.
type Config struct {
	Addr      string
	TLSDomain string
	AdminKey  string
	DBPath    string
}

// Server is the relay HTTP/WebSocket server.
type Server struct {
	hub       *hub.Hub
	config    Config
	http      *http.Server
	regLimiter *ipRateLimiter
}

// ipRateLimiter tracks registration attempts per IP.
type ipRateLimiter struct {
	mu      sync.Mutex
	entries map[string][]time.Time
	limit   int
	window  time.Duration
}

func newIPRateLimiter(limit int, window time.Duration) *ipRateLimiter {
	return &ipRateLimiter{entries: make(map[string][]time.Time), limit: limit, window: window}
}

func (l *ipRateLimiter) Allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	cutoff := now.Add(-l.window)
	// Remove expired entries
	var valid []time.Time
	for _, t := range l.entries[ip] {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}
	if len(valid) >= l.limit {
		l.entries[ip] = valid
		return false
	}
	l.entries[ip] = append(valid, now)
	return true
}

func New(h *hub.Hub, cfg Config) *Server {
	s := &Server{
		hub:        h,
		config:     cfg,
		regLimiter: newIPRateLimiter(5, time.Hour), // 5 registrations per hour per IP
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", s.handleWS)
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/api/v1/pair", s.handlePair)
	mux.HandleFunc("/api/v1/pair/", s.handlePairToken)
	mux.HandleFunc("/api/v1/register", s.handleRegister)

	s.http = &http.Server{
		Addr:    cfg.Addr,
		Handler: mux,
	}

	return s
}

func (s *Server) Start() error {
	log.Printf("Relay server starting on %s", s.config.Addr)

	if s.config.TLSDomain != "" {
		// Let's Encrypt auto TLS
		m := &autocert.Manager{
			Cache:      autocert.DirCache(".autocert-cache"),
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist(s.config.TLSDomain),
		}

		// HTTP challenge server on :80
		go func() {
			h := m.HTTPHandler(nil)
			log.Printf("ACME HTTP challenge server on :80")
			if err := http.ListenAndServe(":80", h); err != nil {
				log.Printf("HTTP server error: %v", err)
			}
		}()

		s.http.TLSConfig = &tls.Config{GetCertificate: m.GetCertificate}
		return s.http.ListenAndServeTLS("", "")
	}

	return s.http.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	log.Printf("Graceful shutdown initiated")
	return s.http.Shutdown(ctx)
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws upgrade error: %v", err)
		return
	}
	go HandleConnection(s.hub, ws)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":       "ok",
		"connections":  s.hub.ConnectionCount(),
		"sessions":     s.hub.SessionCount(),
		"uptime_sec":   int64(time.Since(s.hub.StartTime()).Seconds()),
		"alloc_mb":     float64(memStats.Alloc) / 1024 / 1024,
		"sys_mb":       float64(memStats.Sys) / 1024 / 1024,
		"goroutines":   runtime.NumGoroutine(),
	})
}

func (s *Server) handlePair(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !s.checkAdmin(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	token, err := s.hub.CreateToken()
	if err != nil {
		http.Error(w, "Failed to create token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"token": token})
}

func (s *Server) handlePairToken(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimPrefix(r.URL.Path, "/api/v1/pair/")
	if token == "" {
		http.Error(w, "Token required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		phone, agent := s.hub.GetTokenStatus(token)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{
			"phone": phone,
			"agent": agent,
		})

	case http.MethodDelete:
		if !s.checkAdmin(r) {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		if err := s.hub.DeleteToken(token); err != nil {
			http.Error(w, "Failed to delete token", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Rate limit by IP
	ip := r.RemoteAddr
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		ip = strings.Split(fwd, ",")[0]
	}
	if !s.regLimiter.Allow(strings.TrimSpace(ip)) {
		http.Error(w, "Rate limit exceeded (5/hour)", http.StatusTooManyRequests)
		return
	}

	token, err := s.hub.CreateToken()
	if err != nil {
		http.Error(w, "Failed to create token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"token": token})
}

func (s *Server) checkAdmin(r *http.Request) bool {
	if s.config.AdminKey == "" {
		return true
	}
	return r.Header.Get("X-Admin-Key") == s.config.AdminKey
}
