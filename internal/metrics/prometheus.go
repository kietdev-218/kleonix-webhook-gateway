package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	WebhookRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "webhook_gateway_requests_total",
			Help: "Total number of received webhooks",
		},
		[]string{"source", "event_type", "status"},
	)

	WebhookRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "webhook_gateway_request_duration_seconds",
			Help:    "Duration of webhook processing in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"source", "status"},
	)

	RabbitMQPublishSuccess = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "webhook_gateway_rabbitmq_publish_success_total",
			Help: "Total number of successful RabbitMQ publishes",
		},
		[]string{"exchange", "routing_key"},
	)

	RabbitMQPublishFailure = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "webhook_gateway_rabbitmq_publish_failure_total",
			Help: "Total number of failed RabbitMQ publishes",
		},
		[]string{"exchange", "routing_key"},
	)
)
