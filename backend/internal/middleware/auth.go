package middleware

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func isAPIRequest(c *gin.Context) bool {
	accept := c.GetHeader("Accept")
	contentType := c.GetHeader("Content-Type")
	if strings.Contains(accept, "application/json") {
		return true
	}
	if strings.Contains(contentType, "application/json") {
		return true
	}
	if strings.HasPrefix(c.Request.URL.Path, "/api/") {
		return true
	}
	return false
}

func tokenString(c *gin.Context) string {
	cookie, err := c.Cookie("admin_token")
	if err == nil && cookie != "" {
		return cookie
	}

	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token != authHeader {
			return token
		}
	}
	return ""
}

func RequireAuth(c *gin.Context) {
	tokenStr := tokenString(c)
	if tokenStr == "" {
		if isAPIRequest(c) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "missing token",
			})
		} else {
			c.Redirect(http.StatusFound, "/admin/login")
		}
		c.Abort()
		return
	}

	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("JWT_SECRET")), nil
	})
	if err != nil || !token.Valid {
		if isAPIRequest(c) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid token",
			})
		} else {
			c.Redirect(http.StatusFound, "/admin/login")
		}
		c.Abort()
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if ok {
		if role, exists := claims["role"]; exists {
			c.Set("admin_role", role)
		}
		if adminID, exists := claims["admin_id"]; exists {
			c.Set("admin_id", adminID)
		}
	}

	c.Next()
}

func RequireSuperAdmin(c *gin.Context) {
	role := c.GetString("admin_role")
	if role != "super_admin" {
		if isAPIRequest(c) {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "super admin access required",
			})
		} else {
			c.String(http.StatusForbidden, "super admin access required")
		}
		c.Abort()
		return
	}
}
