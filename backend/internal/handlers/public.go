package handlers

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/syrup/backend/internal/db"
	"github.com/syrup/backend/internal/models"
	"github.com/syrup/backend/internal/renderer"
	"github.com/syrup/backend/internal/services"
	ws "github.com/syrup/backend/internal/websocket"
)

// ShareCardCacheDir is the directory where generated share card PNGs are cached.
// It is relative to the process working directory and may be overridden at startup
// (for example, when running inside the Docker image at /app).
var ShareCardCacheDir = "cmd/api/static/cache/share-cards"

// PaymentMethodDisplay is a template-friendly wrapper with a pre-computed payment URL.
type PaymentMethodDisplay struct {
	ID          string
	Type        string
	DisplayName string
	HandleOrURL string
	URL         string
}

var renderers map[string]*renderer.Renderer

// InitRenderers sets the per-page template renderers used by all handlers.
func InitRenderers(r map[string]*renderer.Renderer) {
	renderers = r
}

func AboutPage(c *gin.Context) {
	data := mergeMaps(pageData(), gin.H{
		"title": "About - Project Syrup",
	})

	if _, exists := c.Get("admin_id"); exists {
		var dbStatus string
		if err := db.Pool.Ping(c.Request.Context()); err != nil {
			dbStatus = "disconnected"
		} else {
			dbStatus = "connected"
		}

		totalWaffles, _, _ := services.CountWaffles()

		var totalAdmins int
		_ = db.Pool.QueryRow(c.Request.Context(), `SELECT COUNT(*) FROM admins WHERE active = true`).Scan(&totalAdmins)

		var wsClients int
		if hub := ws.GetHub(); hub != nil {
			wsClients = hub.TotalClients()
		}

		var startTime time.Time
		if err := db.Pool.QueryRow(c.Request.Context(), `SELECT pg_postmaster_start_time()`).Scan(&startTime); err != nil {
			startTime = time.Now()
		}
		uptime := time.Since(startTime)

		data = mergeMaps(data, gin.H{
			"IsAdmin":      true,
			"DBStatus":     dbStatus,
			"TotalWaffles": totalWaffles,
			"TotalAdmins":  totalAdmins,
			"WSClients":    wsClients,
			"Uptime":       uptime.Round(time.Second).String(),
		})
	}

	renderers["about.html"].Render(c, "about.html", data)
}

// HomePage renders the public home page.
func HomePage(c *gin.Context) {
	scheme := "https"
	if proto := c.GetHeader("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}
	host := c.Request.Host

	renderers["home.html"].Render(c, "home.html", mergeMaps(pageData(), gin.H{
		"title":       "Project Syrup - The Waffle Maker",
		"Host":        host,
		"Description": "Browse and claim spots on Instagram waffle drops",
		"OGImage":     scheme + "://" + host + "/static/img/logo.png",
	}))
}

// WaffleListPage renders the public waffle list page.
func WaffleListPage(c *gin.Context) {
	waffles, err := services.ListWaffles(false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	waffleStats := make(map[string]map[string]interface{})
	for _, w := range waffles {
		stats, err := services.GetWaffleStats(w.ID)
		if err != nil {
			stats = map[string]interface{}{}
		}
		waffleStats[w.ID.String()] = stats
	}

	renderers["waffles.html"].Render(c, "waffles.html", mergeMaps(pageData(), gin.H{
		"title":       "Active Waffles - Project Syrup",
		"waffles":     waffles,
		"waffleStats": waffleStats,
	}))
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

	rawMethods, _ := services.GetPaymentMethodsForWaffle(waffle.ID)
	var paymentMethods []PaymentMethodDisplay
	paymentMethodsByType := make(map[string][]PaymentMethodDisplay)
	for _, pm := range rawMethods {
		display := PaymentMethodDisplay{
			ID:          pm.ID.String(),
			Type:        pm.Type,
			DisplayName: pm.DisplayName,
			HandleOrURL: pm.HandleOrURL,
			URL:         services.GeneratePaymentURL(pm),
		}
		paymentMethods = append(paymentMethods, display)
		paymentMethodsByType[pm.Type] = append(paymentMethodsByType[pm.Type], display)
	}

	scheme := "https"
	if proto := c.GetHeader("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}
	host := c.Request.Host

	description := waffle.Title + " spot board"
	if waffle.Description != nil && *waffle.Description != "" {
		description = *waffle.Description
	}

	ogImage := ""
	if waffle.ImageURL != nil && *waffle.ImageURL != "" {
		img := *waffle.ImageURL
		if strings.HasPrefix(img, "http") {
			ogImage = img
		} else {
			ogImage = scheme + "://" + host + img
		}
	}

	renderers["waffle_detail.html"].Render(c, "waffle_detail.html", mergeMaps(pageData(), gin.H{
		"title":                waffle.Title + " - Project Syrup",
		"waffle":               waffle,
		"spots":                spots,
		"stats":                stats,
		"Host":                 host,
		"Description":          description,
		"OGImage":              ogImage,
		"PaymentMethods":       paymentMethods,
		"PaymentMethodsByType": paymentMethodsByType,
	}))
}

// BuyerStatsPage renders the public buyer stats page.
func BuyerStatsPage(c *gin.Context) {
	handle := services.NormalizeInstagramHandle(c.Param("handle"))

	stats, err := services.GetBuyerStats(handle)
	if err != nil {
		c.String(http.StatusNotFound, "Buyer not found")
		return
	}

	card, cardErr := services.ComputeBuyerCardData(handle)
	if cardErr != nil {
		slog.Error("failed to compute buyer card data", "handle", handle, "error", cardErr)
		card = nil
	}

	history := []models.BuyerWaffleHistory{}
	winRate := 0
	if card != nil {
		history = card.WaffleHistory
		if card.Stats.TotalWins+card.Stats.TotalLosses > 0 {
			winRate = (card.Stats.TotalWins * 100) / (card.Stats.TotalWins + card.Stats.TotalLosses)
		}
	} else {
		if stats.TotalWins+stats.TotalLosses > 0 {
			winRate = (stats.TotalWins * 100) / (stats.TotalWins + stats.TotalLosses)
		}
	}

	renderers["buyer_stats.html"].Render(c, "buyer_stats.html", mergeMaps(pageData(), gin.H{
		"title":   "@" + handle + " - Project Syrup",
		"handle":  handle,
		"stats":   stats,
		"history": history,
		"winRate": winRate,
		"card":    card,
	}))
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

// BuyerCardPage renders the buyer card / shareable stats page.
func BuyerCardPage(c *gin.Context) {
	handle := services.NormalizeInstagramHandle(c.Param("handle"))

	card, err := services.ComputeBuyerCardData(handle)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to compute buyer card")
		return
	}

	winRate := 0
	if card.Stats != nil && card.Stats.TotalSpotsClaimed > 0 {
		winRate = int(card.WinRate * 100)
	}

	luckRatingDisplay := "Even"
	if card.LuckRating > 0 {
		luckRatingDisplay = fmt.Sprintf("+%.1f%%", card.LuckRating*100)
	} else if card.LuckRating < 0 {
		luckRatingDisplay = fmt.Sprintf("%.1f%%", card.LuckRating*100)
	}

	scheme := "https"
	if proto := c.GetHeader("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}
	host := c.Request.Host

	description := fmt.Sprintf("@%s's Waffle Stats — ", handle)
	if card.Stats != nil {
		description += fmt.Sprintf("%d wins, %d spots", card.Stats.TotalWins, card.Stats.TotalSpotsClaimed)
	} else {
		description += "no activity yet"
	}

	renderers["buyer_card.html"].Render(c, "buyer_card.html", mergeMaps(pageData(), gin.H{
		"title":             "@" + handle + "'s Waffle Card - Project Syrup",
		"handle":            handle,
		"card":              card,
		"winRate":           winRate,
		"luckRatingDisplay": luckRatingDisplay,
		"Description":       description,
		"Host":              host,
		"Scheme":            scheme,
	}))
}

// GetBuyerCard returns buyer card data as JSON. Returns data even for unknown buyers.
func GetBuyerCard(c *gin.Context) {
	handle := services.NormalizeInstagramHandle(c.Param("handle"))

	card, err := services.ComputeBuyerCardData(handle)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, card)
}

// WaffleShareCardPNG serves a downloadable PNG share card for a waffle.
// The card is generated on first request and cached to disk; subsequent
// requests serve the cached file. The format query parameter accepts
// "story" (default, 1080x1920) or "square" (1080x1080).
func WaffleShareCardPNG(c *gin.Context) {
	slug := c.Param("slug")
	format := strings.ToLower(strings.TrimSpace(c.DefaultQuery("format", services.ShareCardFormatStory)))
	if format != services.ShareCardFormatSquare {
		format = services.ShareCardFormatStory
	}

	waffle, err := services.GetWaffleBySlug(slug)
	if err != nil || waffle.Archived {
		c.String(http.StatusNotFound, "Waffle not found")
		return
	}

	cacheFileName := fmt.Sprintf("%s-%s.png", slug, format)
	cachePath := filepath.Join(ShareCardCacheDir, cacheFileName)

	if cached, err := os.ReadFile(cachePath); err == nil {
		c.Header("Content-Type", "image/png")
		c.Header("Cache-Control", "public, max-age=3600")
		c.Data(http.StatusOK, "image/png", cached)
		return
	}

	pngBytes, err := services.GenerateShareCard(waffle, format)
	if err != nil {
		slog.Error("failed to generate share card", "slug", slug, "format", format, "error", err)
		c.String(http.StatusInternalServerError, "Failed to generate share card")
		return
	}

	if err := os.MkdirAll(ShareCardCacheDir, 0o755); err != nil {
		slog.Error("failed to create share card cache directory", "path", ShareCardCacheDir, "error", err)
	} else if err := os.WriteFile(cachePath, pngBytes, 0o644); err != nil {
		slog.Error("failed to write share card cache", "path", cachePath, "error", err)
	}

	c.Header("Content-Type", "image/png")
	c.Header("Cache-Control", "public, max-age=3600")
	c.Data(http.StatusOK, "image/png", pngBytes)
}
