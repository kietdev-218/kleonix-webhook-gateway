package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const CorrelationIDKey = "correlation_id"

// CorrelationID injects a correlation ID into the request context.
// It checks the incoming headers first, and generates a new one if not present.
func CorrelationID() gin.HandlerFunc {
	return func(c *gin.Context) {
		corID := c.GetHeader("X-Correlation-ID")
		if corID == "" {
			corID = uuid.New().String()
		}
		c.Set(CorrelationIDKey, corID)
		c.Header("X-Correlation-ID", corID)
		c.Next()
	}
}
