package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/kleonix/webhook-gateway/internal/event"
	"github.com/kleonix/webhook-gateway/internal/metrics"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

type RabbitMQPublisher struct {
	conn           *Connection
	exchange       string
	routingKey     string
	logger         *zap.Logger
	maxRetries     int
	baseRetryDelay time.Duration
}

func NewRabbitMQPublisher(conn *Connection, exchange, routingKey string, maxRetries int, baseRetryDelay time.Duration, logger *zap.Logger) *RabbitMQPublisher {
	return &RabbitMQPublisher{
		conn:           conn,
		exchange:       exchange,
		routingKey:     routingKey,
		maxRetries:     maxRetries,
		baseRetryDelay: baseRetryDelay,
		logger:         logger,
	}
}

func (p *RabbitMQPublisher) Publish(ctx context.Context, envelope *event.Envelope) error {
	body, err := json.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("failed to marshal event envelope: %w", err)
	}

	delay := p.baseRetryDelay

	for attempt := 0; attempt <= p.maxRetries; attempt++ {
		ch, err := p.conn.AcquireChannel()
		if err != nil {
			if attempt == p.maxRetries {
				metrics.RabbitMQPublishFailure.WithLabelValues(p.exchange, p.routingKey).Inc()
				return fmt.Errorf("failed to get channel after %d attempts: %w", p.maxRetries, err)
			}
			p.logger.Warn("Failed to get channel, retrying...", zap.Error(err), zap.Int("attempt", attempt+1))

			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return ctx.Err()
			}
			delay *= 2
			continue
		}

		deferredConfirm, err := ch.PublishWithDeferredConfirmWithContext(ctx,
			p.exchange,
			p.routingKey,
			true,  // mandatory
			false, // immediate
			amqp.Publishing{
				ContentType:  "application/json",
				DeliveryMode: amqp.Persistent,
				MessageId:    envelope.EventID,
				Timestamp:    envelope.Timestamp,
				AppId:        "webhook-gateway",
				Type:         envelope.EventType,
				Body:         body,
				Headers: amqp.Table{
					"correlation_id": envelope.CorrelationID,
				},
			},
		)

		if err != nil {
			ch.Close() // Discard channel on error
			if attempt == p.maxRetries {
				metrics.RabbitMQPublishFailure.WithLabelValues(p.exchange, p.routingKey).Inc()
				return fmt.Errorf("failed to publish message: %w", err)
			}
			p.logger.Warn("Failed to publish message, retrying...", zap.Error(err), zap.Int("attempt", attempt+1))
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return ctx.Err()
			}
			delay *= 2
			continue
		}

		if deferredConfirm != nil {
			acked, err := deferredConfirm.WaitContext(ctx)
			if err != nil {
				ch.Close() // Discard channel on error
				metrics.RabbitMQPublishFailure.WithLabelValues(p.exchange, p.routingKey).Inc()
				if attempt == p.maxRetries {
					return fmt.Errorf("failed waiting for publish confirmation: %w", err)
				}
				p.logger.Warn("Failed waiting for confirm, retrying...", zap.Error(err), zap.Int("attempt", attempt+1))
				select {
				case <-time.After(delay):
				case <-ctx.Done():
					return ctx.Err()
				}
				delay *= 2
				continue
			}

			if acked {
				p.conn.ReleaseChannel(ch)
				metrics.RabbitMQPublishSuccess.WithLabelValues(p.exchange, p.routingKey).Inc()
				return nil
			}

			ch.Close() // Discard channel if nack received
			if attempt == p.maxRetries {
				metrics.RabbitMQPublishFailure.WithLabelValues(p.exchange, p.routingKey).Inc()
				return fmt.Errorf("failed to deliver message: nack received")
			}
			p.logger.Warn("Nack received, retrying...", zap.Int("attempt", attempt+1))
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return ctx.Err()
			}
			delay *= 2
			continue
		}
	}

	return fmt.Errorf("exhausted all retries to publish message")
}

func (p *RabbitMQPublisher) Close() error {
	return nil
}
