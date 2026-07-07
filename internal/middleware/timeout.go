package middleware

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
)

// Timeout adds a timeout to the request context.
// It relies on downstream handlers to respect the context cancellation.
func Timeout(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
