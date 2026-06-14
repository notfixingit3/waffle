package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/syrup/backend/internal/services"
)

// ListShareTemplatesAPI returns all message templates as JSON.
func ListShareTemplatesAPI(c *gin.Context) {
	adminIDStr, exists := c.Get("admin_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}
	if _, err := uuid.Parse(adminIDStr.(string)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid admin"})
		return
	}

	templates, err := services.ListMessageTemplates()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"templates": templates})
}

// CreateShareTemplateAPI creates a new message template from JSON body {name, body}.
func CreateShareTemplateAPI(c *gin.Context) {
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
		Name string `json:"name"`
		Body string `json:"body"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	template, err := services.CreateMessageTemplate(req.Name, req.Body, adminID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	RecordAudit(c, "create_share_template", "message_template", template.ID.String(), "created template '"+template.Name+"'")

	c.JSON(http.StatusCreated, template)
}

// UpdateShareTemplateAPI updates an existing message template from JSON body {name, body}.
func UpdateShareTemplateAPI(c *gin.Context) {
	adminIDStr, exists := c.Get("admin_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}
	if _, err := uuid.Parse(adminIDStr.(string)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid admin"})
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req struct {
		Name string `json:"name"`
		Body string `json:"body"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if err := services.UpdateMessageTemplate(id, req.Name, req.Body); err != nil {
		status := http.StatusBadRequest
		if err.Error() == "message template not found: "+id.String() {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	RecordAudit(c, "update_share_template", "message_template", id.String(), "updated template '"+req.Name+"'")

	c.JSON(http.StatusOK, gin.H{"message": "template updated"})
}

// DeleteShareTemplateAPI deletes a message template. Returns 400 if it's the last template.
func DeleteShareTemplateAPI(c *gin.Context) {
	adminIDStr, exists := c.Get("admin_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}
	if _, err := uuid.Parse(adminIDStr.(string)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid admin"})
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := services.DeleteMessageTemplate(id); err != nil {
		status := http.StatusBadRequest
		if err.Error() == "message template not found: "+id.String() {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	RecordAudit(c, "delete_share_template", "message_template", id.String(), "deleted template")

	c.JSON(http.StatusOK, gin.H{"message": "template deleted"})
}

// SetDefaultShareTemplateAPI sets a message template as the default.
func SetDefaultShareTemplateAPI(c *gin.Context) {
	adminIDStr, exists := c.Get("admin_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}
	if _, err := uuid.Parse(adminIDStr.(string)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid admin"})
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := services.SetDefaultMessageTemplate(id); err != nil {
		status := http.StatusBadRequest
		if err.Error() == "message template not found: "+id.String() {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	RecordAudit(c, "set_default_share_template", "message_template", id.String(), "set template as default")

	c.JSON(http.StatusOK, gin.H{"message": "default template updated"})
}
