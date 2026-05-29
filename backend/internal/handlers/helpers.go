package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ErrorResponse sends a JSON error response with the given status code and message.
// Produces the same JSON shape as direct c.JSON(gin.H{"error": ...}) calls.
func ErrorResponse(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"error": message})
}

// ServerErrorResponse logs the full error and sends a generic 500 response.
// The internal error details are never exposed in the response body.
func ServerErrorResponse(c *gin.Context, err error) {
	fmt.Printf("[ERROR] %v\n", err)
	c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
}
