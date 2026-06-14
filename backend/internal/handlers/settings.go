package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/syrup/backend/internal/models"
	"github.com/syrup/backend/internal/services"
)

func SettingsPage(c *gin.Context) {
	data := adminNavData(c)
	data["title"] = "Settings - Project Syrup"

	adminIDStr, exists := c.Get("admin_id")
	if exists {
		if idStr, ok := adminIDStr.(string); ok {
			if id, err := uuid.Parse(idStr); err == nil {
				if admin, err := services.GetAdminByID(id); err == nil {
					data["Timezone"] = admin.Timezone
					data["Role"] = admin.Role
					data["FirstName"] = admin.FirstName
					data["LastName"] = admin.LastName
					data["Email"] = admin.Email
					data["SocialLinks"] = admin.SocialLinks
				}
			}
		}
	}

	if data["Role"] == "super_admin" {
		settingsKeys := []string{
			"whois_server",
			"jwt_expiration_hours",
			"password_min_length",
			"audit_retention_days",
			"login_history_retention_days",
		}
		for _, key := range settingsKeys {
			value, err := services.GetSetting(key)
			if err == nil {
				data[key] = value
			}
		}
	}

	if templates, err := services.ListMessageTemplates(); err == nil {
		data["Templates"] = templates
	}

	if role, ok := data["Role"].(string); ok {
		data["CanManageShareTemplates"] = canManageShareTemplates(role)
	}

	renderers["settings.html"].Render(c, "settings.html", data)
}

func canManageShareTemplates(role string) bool {
	return role == "admin" || role == "super_admin" || role == "waffle_manager"
}

func UpdateTimezoneAPI(c *gin.Context) {
	adminIDStr, exists := c.Get("admin_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	adminID, err := uuid.Parse(adminIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid admin"})
		return
	}

	var req struct {
		Timezone string `json:"timezone"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if req.Timezone == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "timezone is required"})
		return
	}

	if _, err := time.LoadLocation(req.Timezone); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid timezone"})
		return
	}

	admin, err := services.UpdateAdminTimezone(adminID, req.Timezone)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, admin)
}

func UpdateProfileAPI(c *gin.Context) {
	adminIDStr, exists := c.Get("admin_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	adminID, err := uuid.Parse(adminIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid admin"})
		return
	}

	var req models.UpdateAdminProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	admin, err := services.UpdateAdminProfile(adminID, req)
	if err != nil {
		if err.Error() == "email already in use" {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		if len(err.Error()) >= 17 && err.Error()[:17] == "validation failed" {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, admin)
}

func GetWhoisSettingsAPI(c *gin.Context) {
	adminIDStr, exists := c.Get("admin_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	_, err := uuid.Parse(adminIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid admin"})
		return
	}

	value, err := services.GetSetting("whois_server")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"whois_server": value})
}

func UpdateWhoisSettingsAPI(c *gin.Context) {
	adminIDStr, exists := c.Get("admin_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	adminID, err := uuid.Parse(adminIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid admin"})
		return
	}

	var req struct {
		WhoisServer string `json:"whois_server"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if req.WhoisServer == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "whois_server is required"})
		return
	}

	if err := services.SetSetting("whois_server", req.WhoisServer, adminID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	RecordAudit(c, "update_whois_settings", "settings", "", "WHOIS server updated to "+req.WhoisServer)

	c.JSON(http.StatusOK, gin.H{"whois_server": req.WhoisServer})
}

var allSettingsKeys = []string{
	"whois_server",
	"jwt_expiration_hours",
	"password_min_length",
	"audit_retention_days",
	"login_history_retention_days",
}

func GetAllSettingsAPI(c *gin.Context) {
	adminIDStr, exists := c.Get("admin_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}
	if _, err := uuid.Parse(adminIDStr.(string)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid admin"})
		return
	}

	settings := make([]gin.H, 0, len(allSettingsKeys))
	for _, key := range allSettingsKeys {
		value, err := services.GetSetting(key)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		settings = append(settings, gin.H{"key": key, "value": value})
	}

	c.JSON(http.StatusOK, gin.H{"settings": settings})
}

func UpdateSettingAPI(c *gin.Context) {
	adminIDStr, exists := c.Get("admin_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	adminID, err := uuid.Parse(adminIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid admin"})
		return
	}

	key := c.Param("key")

	var req struct {
		Value string `json:"value"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if err := services.SetSetting(key, req.Value, adminID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	RecordAudit(c, "update_setting", "settings", "", "Updated "+key+" to "+req.Value)

	c.JSON(http.StatusOK, gin.H{"key": key, "value": req.Value})
}
