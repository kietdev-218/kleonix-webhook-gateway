package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kleonix/webhook-gateway/internal/metrics"
)

// Metrics records HTTP request duration and count
func Metrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		duration := time.Since(start).Seconds()
		status := strconv.Itoa(c.Writer.Status())
		source := c.Param("source")
		if source == "" {
			source = "unknown"
		}

		metrics.WebhookRequestDuration.WithLabelValues(source, status).Observe(duration)
	}
}
