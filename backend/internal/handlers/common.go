package handlers

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/syrup/backend/internal/services"
)

// AppVersion is set from main.go via ldflags injection.
var AppVersion string

func pageData() gin.H {
	total, active, _ := services.CountWaffles()
	return gin.H{
		"Version":       AppVersion,
		"DevMode":       strings.Contains(strings.ToLower(AppVersion), "dev"),
		"TotalWaffles":  total,
		"ActiveWaffles": active,
		"ServerTime":    time.Now().UTC().Format("3:04 PM"),
	}
}

// mergeMaps combines multiple gin.H maps into one. Later maps override earlier ones.
func mergeMaps(maps ...gin.H) gin.H {
	result := gin.H{}
	for _, m := range maps {
		for k, v := range m {
			result[k] = v
		}
	}
	return result
}
