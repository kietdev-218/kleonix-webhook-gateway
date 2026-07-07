package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kleonix/webhook-gateway/internal/event"
	"github.com/kleonix/webhook-gateway/internal/metrics"
	"github.com/kleonix/webhook-gateway/internal/middleware"
	"go.uber.org/zap"
)

// Publisher defines the contract for publishing events.
// This interface lives in the handler package (Consumer of the interface)
// to follow the Dependency Inversion Principle.
type Publisher interface {
	Publish(ctx context.Context, envelope *event.Envelope) error
	Close() error
}

type KratosHandler struct {
	publisher Publisher
	logger    *zap.Logger
}

func NewKratosHandler(pub Publisher, logger *zap.Logger) *KratosHandler {
	return &KratosHandler{
		publisher: pub,
		logger:    logger,
	}
}

func (h *KratosHandler) HandleWebhook(c *gin.Context) {
	c.Set("source", "ory.kratos")

	body, err := c.GetRawData()
	if err != nil {
		h.logger.Warn("Failed to read body", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
		return
	}

	h.logger.Debug("Received webhook payload from Kratos", zap.ByteString("raw_payload", body))

	var metadata struct {
		Type      string `json:"type"`
		EventType string `json:"event_type"`
		State     string `json:"state"`
		Identity  struct {
			ID string `json:"id"`
		} `json:"identity"`
		IdentityID string `json:"identity_id"`
	}

	if err := json.Unmarshal(body, &metadata); err != nil {
		h.logger.Warn("Invalid JSON payload", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json payload"})
		return
	}

	eventID := uuid.New().String()
	eventType := "kratos.unknown"

	if metadata.Type != "" {
		eventType = "kratos." + metadata.Type
	} else if metadata.EventType != "" {
		eventType = "kratos." + metadata.EventType
	} else if metadata.State != "" {
		eventType = "kratos.state." + metadata.State
	}

	identityID := metadata.IdentityID
	if identityID == "" {
		identityID = metadata.Identity.ID
	}

	metrics.WebhookRequestsTotal.WithLabelValues("ory.kratos", eventType, "received").Inc()

	corID, _ := c.Get(middleware.CorrelationIDKey)
	correlationID, _ := corID.(string)
	traceID := c.GetHeader("X-Trace-ID")

	env := &event.Envelope{
		EventID:       eventID,
		EventType:     eventType,
		Source:        "ory.kratos",
		Timestamp:     time.Now().UTC(),
		CorrelationID: correlationID,
		TraceID:       traceID,
		IdentityID:    identityID,
		Payload:       json.RawMessage(body),
		Metadata: map[string]interface{}{
			"client_ip":  c.ClientIP(),
			"user_agent": c.Request.UserAgent(),
		},
		Version: "1.0",
	}

	if err := h.publisher.Publish(c.Request.Context(), env); err != nil {
		h.logger.Error("Failed to publish event", zap.Error(err), zap.String("event_id", eventID))
		metrics.WebhookRequestsTotal.WithLabelValues("ory.kratos", eventType, "failed").Inc()

		if c.Request.Context().Err() != nil {
			c.JSON(http.StatusGatewayTimeout, gin.H{"error": "timeout publishing event"})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	metrics.WebhookRequestsTotal.WithLabelValues("ory.kratos", eventType, "success").Inc()
	c.JSON(http.StatusOK, gin.H{"status": "ok", "event_id": eventID})
}
