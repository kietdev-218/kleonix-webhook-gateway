package middleware

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// SizeLimit restricts the maximum size of the incoming request body.
func SizeLimit(maxSize int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.ContentLength > maxSize {
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{
				"error": fmt.Sprintf("request body exceeds maximum allowed size of %d bytes", maxSize),
			})
			return
		}

		// Also enforce using MaxBytesReader in case Content-Length is missing or spoofed
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxSize)

		c.Next()
	}
}
