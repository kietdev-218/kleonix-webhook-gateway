package event

import (
	"encoding/json"
	"time"
)

// Envelope is the standard format for all webhooks
type Envelope struct {
	EventID       string                 `json:"event_id"`
	EventType     string                 `json:"event_type"`
	Source        string                 `json:"source"`
	Timestamp     time.Time              `json:"timestamp"`
	CorrelationID string                 `json:"correlation_id"`
	TraceID       string                 `json:"trace_id,omitempty"`
	IdentityID    string                 `json:"identity_id,omitempty"`
	Payload       json.RawMessage        `json:"payload"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	Version       string                 `json:"version"`
}
