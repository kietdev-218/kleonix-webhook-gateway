package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/kleonix/webhook-gateway/internal/event"
	"go.uber.org/zap"
)

// MockPublisher is a simple mock for rabbitmq.Publisher
type MockPublisher struct {
	PublishedEnvelopes []*event.Envelope
	ShouldFail         bool
}

func (m *MockPublisher) Publish(ctx context.Context, env *event.Envelope) error {
	if m.ShouldFail {
		return context.DeadlineExceeded // Simulate an error
	}
	m.PublishedEnvelopes = append(m.PublishedEnvelopes, env)
	return nil
}

func (m *MockPublisher) Close() error {
	return nil
}

func TestKratosHandler_HandleWebhook(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()

	tests := []struct {
		name           string
		payload        map[string]interface{}
		publisherFails bool
		expectedStatus int
		expectedType   string
	}{
		{
			name: "Valid registration event",
			payload: map[string]interface{}{
				"type": "registration",
				"identity": map[string]interface{}{
					"id": "12345-67890",
					"traits": map[string]interface{}{
						"email": "test@example.com",
					},
				},
			},
			publisherFails: false,
			expectedStatus: http.StatusOK,
			expectedType:   "kratos.registration",
		},
		{
			name: "Valid settings event using event_type",
			payload: map[string]interface{}{
				"event_type":  "settings",
				"identity_id": "user-abc",
			},
			publisherFails: false,
			expectedStatus: http.StatusOK,
			expectedType:   "kratos.settings",
		},
		{
			name:           "Invalid JSON payload",
			payload:        nil,
			publisherFails: false,
			expectedStatus: http.StatusBadRequest,
			expectedType:   "",
		},
		{
			name: "Publisher failure",
			payload: map[string]interface{}{
				"type": "registration",
			},
			publisherFails: true,
			expectedStatus: http.StatusInternalServerError,
			expectedType:   "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockPub := &MockPublisher{ShouldFail: tc.publisherFails}
			h := NewKratosHandler(mockPub, logger)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			if tc.payload == nil {
				c.Request, _ = http.NewRequest(http.MethodPost, "/webhooks/kratos", bytes.NewBuffer([]byte("invalid json")))
			} else {
				bodyBytes, _ := json.Marshal(tc.payload)
				c.Request, _ = http.NewRequest(http.MethodPost, "/webhooks/kratos", bytes.NewBuffer(bodyBytes))
			}
			c.Request.Header.Set("Content-Type", "application/json")

			h.HandleWebhook(c)

			if w.Code != tc.expectedStatus {
				t.Errorf("expected status %d, got %d", tc.expectedStatus, w.Code)
			}

			if !tc.publisherFails && tc.expectedStatus == http.StatusOK {
				if len(mockPub.PublishedEnvelopes) != 1 {
					t.Fatalf("expected 1 published envelope, got %d", len(mockPub.PublishedEnvelopes))
				}
				env := mockPub.PublishedEnvelopes[0]
				if env.EventType != tc.expectedType {
					t.Errorf("expected event type %q, got %q", tc.expectedType, env.EventType)
				}
				if env.EventID == "" {
					t.Error("expected non-empty event ID")
				}
			}
		})
	}
}
