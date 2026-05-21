package handlers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/syrup/backend/internal/models"
	"github.com/syrup/backend/internal/renderer"
	"github.com/syrup/backend/internal/services"
	ws "github.com/syrup/backend/internal/websocket"
)

var renderers map[string]*renderer.Renderer

// InitRenderers sets the per-page template renderers used by all handlers.
func InitRenderers(r map[string]*renderer.Renderer) {
	renderers = r
}

// HomePage renders the public home page.
func HomePage(c *gin.Context) {
	renderers["home.html"].Render(c, "home.html", gin.H{
		"title": "Project Syrup - The Waffle Maker",
	})
}

// WaffleListPage renders the public waffle list page.
func WaffleListPage(c *gin.Context) {
	waffles, err := services.ListWaffles(false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	renderers["waffles.html"].Render(c, "waffles.html", gin.H{
		"title":   "Active Waffles - Project Syrup",
		"waffles": waffles,
	})
}

// WaffleDetailPage renders the public waffle detail page with an interactive spot grid.
func WaffleDetailPage(c *gin.Context) {
	slug := c.Param("slug")

	waffle, err := services.GetWaffleBySlug(slug)
	if err != nil {
		c.String(http.StatusNotFound, "Waffle not found")
		return
	}

	spots, err := services.GetSpotsByWaffleID(waffle.ID)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load spots")
		return
	}

	stats, err := services.GetWaffleStats(waffle.ID)
	if err != nil {
		stats = map[string]interface{}{}
	}

	renderers["waffle_detail.html"].Render(c, "waffle_detail.html", gin.H{
		"title":     waffle.Title + " - Project Syrup",
		"waffle":    waffle,
		"spots":     spots,
		"stats":     stats,
	})
}

// ClaimSpotsAPI handles JSON spot claim requests (kept as async fetch API).
func ClaimSpotsAPI(c *gin.Context) {
	var req models.CreateClaimRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	waffleID, err := uuid.Parse(req.WaffleID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid waffle id"})
		return
	}

	if err := services.ClaimSpots(waffleID, req.Spots, req.InstagramHandle); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	waffle, err := services.GetWaffleByID(waffleID)
	if err != nil {
		log.Printf("Failed to get waffle for broadcast: %v", err)
	} else {
		handle := services.NormalizeInstagramHandle(req.InstagramHandle)
		for _, spotNum := range req.Spots {
			ws.BroadcastSpotUpdate(waffle.Slug, spotNum, string(models.SpotStatusPending), handle)
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "spots claimed successfully"})
}

// BuyerStatsPage renders the public buyer stats page.
func BuyerStatsPage(c *gin.Context) {
	handle := services.NormalizeInstagramHandle(c.Param("handle"))

	stats, err := services.GetBuyerStats(handle)
	if err != nil {
		c.String(http.StatusNotFound, "Buyer not found")
		return
	}

	history, err := services.GetBuyerWaffleHistory(handle)
	if err != nil {
		history = []models.BuyerWaffleHistory{}
	}

	winRate := 0
	if stats.TotalWins+stats.TotalLosses > 0 {
		winRate = (stats.TotalWins * 100) / (stats.TotalWins + stats.TotalLosses)
	}

	renderers["buyer_stats.html"].Render(c, "buyer_stats.html", gin.H{
		"title":       "@" + handle + " - Project Syrup",
		"handle":      handle,
		"stats":       stats,
		"history":     history,
		"winRate":     winRate,
	})
}

// GetBuyerStats returns buyer stats as JSON.
func GetBuyerStats(c *gin.Context) {
	handle := services.NormalizeInstagramHandle(c.Param("handle"))

	stats, err := services.GetBuyerStats(handle)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "buyer not found"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetBuyerHistory returns buyer waffle history as JSON.
func GetBuyerHistory(c *gin.Context) {
	handle := services.NormalizeInstagramHandle(c.Param("handle"))

	history, err := services.GetBuyerWaffleHistory(handle)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"history": history})
}
