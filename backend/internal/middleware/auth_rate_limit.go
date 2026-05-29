package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/syrup/backend/internal/services"
)

// AuthRateLimit checks whether the IP+username combination is locked out
// before allowing the login handler to proceed. It extracts the username
// from either a form POST or JSON body and stores it in the context so
// downstream handlers can call RecordFailedLoginAttempt / ResetLoginAttempts.
func AuthRateLimit(c *gin.Context) {
	// Extract username from form body (for POST /admin/login)
	username := c.PostForm("username")

	// Fall back to JSON body (for POST /api/admin/login)
	if username == "" {
		body, err := c.GetRawData()
		if err == nil {
			var data struct {
				Username string `json:"username"`
			}
			if json.Unmarshal(body, &data) == nil {
				username = data.Username
			}
			// Restore body so the handler can read it again
			c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
		}
	}

	if username == "" {
		c.Next()
		return
	}

	ip := c.ClientIP()
	if services.IsLoginLockedOut(ip, username) {
		c.JSON(http.StatusTooManyRequests, gin.H{
			"error": "too many failed login attempts, try again in 15 minutes",
		})
		c.Abort()
		return
	}

	c.Next()
}
