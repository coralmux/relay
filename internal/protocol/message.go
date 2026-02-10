package protocol

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"time"
)

// Envelope is the top-level message wrapper for all relay protocol messages.
type Envelope struct {
	V       int             `json:"v"`
	Type    string          `json:"type"`
	ID      string          `json:"id"`
	TS      int64           `json:"ts"`
	Payload json.RawMessage `json:"payload"`
}

// NewEnvelope creates a new Envelope with generated ID and current timestamp.
func NewEnvelope(msgType string, payload interface{}) (*Envelope, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return &Envelope{
		V:       ProtocolVersion,
		Type:    msgType,
		ID:      generateID(),
		TS:      time.Now().UnixMilli(),
		Payload: data,
	}, nil
}

// ParsePayload unmarshals the payload into the given target.
func (e *Envelope) ParsePayload(target interface{}) error {
	return json.Unmarshal(e.Payload, target)
}

// Marshal serializes the envelope to JSON bytes.
func (e *Envelope) Marshal() ([]byte, error) {
	return json.Marshal(e)
}

func generateID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp-based ID if crypto/rand fails
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
