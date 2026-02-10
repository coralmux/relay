package server

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
	"github.com/openclaw/openclaw-relay/internal/hub"
	"github.com/openclaw/openclaw-relay/internal/protocol"
	"github.com/openclaw/openclaw-relay/internal/ratelimit"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingInterval   = 30 * time.Second
	maxMessageSize = ratelimit.MaxMessageSize
)

// HandleConnection manages the lifecycle of a single WebSocket connection.
func HandleConnection(h *hub.Hub, ws *websocket.Conn) {
	conn := hub.NewConnection(ws, "", nil)
	defer ws.Close()

	ws.SetReadLimit(int64(maxMessageSize))
	ws.SetReadDeadline(time.Now().Add(pongWait))
	ws.SetPongHandler(func(string) error {
		ws.SetReadDeadline(time.Now().Add(pongWait))
		conn.LastPing = time.Now()
		return nil
	})

	// First message must be auth
	_, raw, err := ws.ReadMessage()
	if err != nil {
		log.Printf("read auth error: %v", err)
		return
	}

	var env protocol.Envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		sendError(ws, protocol.ErrInvalidMessage, "Invalid JSON")
		return
	}

	if env.Type != protocol.TypeAuth {
		sendError(ws, protocol.ErrUnauthorized, "First message must be auth")
		return
	}

	session, err := h.Authenticate(conn, &env)
	if err != nil {
		authFail, envErr := protocol.NewEnvelope(protocol.TypeAuthFail, protocol.ErrorPayload{
			Code:    protocol.ErrUnauthorized,
			Message: err.Error(),
		})
		if envErr != nil {
			log.Printf("error creating auth fail envelope: %v", envErr)
			return
		}
		data, marshalErr := authFail.Marshal()
		if marshalErr != nil {
			log.Printf("error marshaling auth fail: %v", marshalErr)
			return
		}
		ws.WriteMessage(websocket.TextMessage, data)
		return
	}

	// Ensure disconnect is always called when readPump exits
	defer h.Disconnect(session, conn)

	// Send auth.ok
	paired := session.IsPaired()
	authOk, envErr := protocol.NewEnvelope(protocol.TypeAuthOk, protocol.AuthOkPayload{
		Paired: paired,
	})
	if envErr != nil {
		log.Printf("error creating auth ok envelope: %v", envErr)
		return
	}
	data, marshalErr := authOk.Marshal()
	if marshalErr != nil {
		log.Printf("error marshaling auth ok: %v", marshalErr)
		return
	}
	ws.WriteMessage(websocket.TextMessage, data)

	// Notify peer if now paired
	if paired {
		peer := session.PeerConn(conn.Role)
		if peer != nil {
			statusEnv, err := protocol.NewEnvelope(protocol.TypeStatus, protocol.StatusPayload{Peer: "online"})
			if err != nil {
				log.Printf("error creating online status: %v", err)
			} else {
				statusData, err := statusEnv.Marshal()
				if err != nil {
					log.Printf("error marshaling online status: %v", err)
				} else {
					select {
					case peer.Send <- statusData:
					default:
					}
				}
			}
		}
	}

	// Start write pump
	go writePump(conn)

	// Read pump (blocking)
	readPump(h, session, conn)
}

func readPump(h *hub.Hub, session *hub.Session, conn *hub.Connection) {
	for {
		conn.WS.SetReadDeadline(time.Now().Add(pongWait))
		_, raw, err := conn.WS.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("ws read error: %v", err)
			}
			return
		}

		var env protocol.Envelope
		if err := json.Unmarshal(raw, &env); err != nil {
			// Send error back for malformed JSON
			sendError(conn.WS, protocol.ErrInvalidMessage, "Invalid JSON message")
			continue
		}

		// Role-based message validation
		if conn.Role == protocol.RolePhone {
			switch env.Type {
			case protocol.TypeChatStream, protocol.TypeChatDone:
				// Phone cannot send streaming messages
				sendError(conn.WS, protocol.ErrInvalidMessage, "Phone cannot send stream messages")
				continue
			}
		}

		switch env.Type {
		case protocol.TypePing:
			pong, err := protocol.NewEnvelope(protocol.TypePong, nil)
			if err != nil {
				log.Printf("error creating pong: %v", err)
				continue
			}
			data, err := pong.Marshal()
			if err != nil {
				log.Printf("error marshaling pong: %v", err)
				continue
			}
			select {
			case conn.Send <- data:
			default:
			}

		case protocol.TypeChatSend, protocol.TypeChatStream, protocol.TypeChatDone, protocol.TypeChatError, protocol.TypeKeyExchange,
			protocol.TypeAgentList, protocol.TypeAgentListResult, protocol.TypeAgentCreate, protocol.TypeAgentUpdate, protocol.TypeAgentDelete, protocol.TypeAgentResult,
			protocol.TypeChatHistory, protocol.TypeChatHistoryResult,
			protocol.TypeChatToolStatus,
			protocol.TypeMemorySearch, protocol.TypeMemorySearchResult:
			h.ForwardMessage(session, conn, raw)
		}
	}
}

func writePump(conn *hub.Connection) {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()

	for {
		select {
		case msg, ok := <-conn.Send:
			conn.WS.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				conn.WS.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := conn.WS.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}

		case <-ticker.C:
			conn.WS.SetWriteDeadline(time.Now().Add(writeWait))
			if err := conn.WS.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}

		case <-conn.Done:
			return
		}
	}
}

func sendError(ws *websocket.Conn, code, message string) {
	env, err := protocol.NewEnvelope(protocol.TypeChatError, protocol.ErrorPayload{
		Code:    code,
		Message: message,
	})
	if err != nil {
		log.Printf("error creating sendError envelope: %v", err)
		return
	}
	data, err := env.Marshal()
	if err != nil {
		log.Printf("error marshaling sendError: %v", err)
		return
	}
	ws.WriteMessage(websocket.TextMessage, data)
}
