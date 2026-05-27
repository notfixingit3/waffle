package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RequestID returns a middleware that ensures every request has a request ID.
// If the client sends an X-Request-ID header, it is preserved.
// Otherwise, a UUID v4 is generated.
// The request ID is stored in the Gin context (key "request_id") and set on the response header.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Request-ID")
		if id == "" {
			id = uuid.New().String()
		}
		c.Set("request_id", id)
		c.Header("X-Request-ID", id)
		c.Next()
	}
}
