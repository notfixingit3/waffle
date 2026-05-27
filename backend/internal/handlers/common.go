package handlers

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// AppVersion is set from main.go via ldflags injection.
var AppVersion string

// pageData returns common template data including version info.
func pageData() gin.H {
	return gin.H{
		"Version": AppVersion,
		"DevMode": strings.Contains(strings.ToLower(AppVersion), "dev"),
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
