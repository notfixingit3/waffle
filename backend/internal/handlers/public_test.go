package handlers

import (
	"bytes"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/syrup/backend/internal/models"
	"github.com/syrup/backend/internal/renderer"
)

func baseDir() string {
	_, b, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(b), "..", "..")
}

func setupRendererAll(t *testing.T) *renderer.Renderer {
	t.Helper()
	tmpl := renderer.New(nil)
	err := tmpl.AddFromFiles(
		filepath.Join(baseDir(), "templates", "layouts", "base.html"),
		filepath.Join(baseDir(), "templates", "partials", "*.html"),
		filepath.Join(baseDir(), "templates", "pages", "public", "*.html"),
	)
	if err != nil {
		t.Fatalf("failed to parse templates: %v", err)
	}
	return tmpl
}

func setupRendererBuyerStats(t *testing.T) *renderer.Renderer {
	t.Helper()
	tmpl := renderer.New(nil)
	err := tmpl.AddFromFiles(
		filepath.Join(baseDir(), "templates", "layouts", "base.html"),
		filepath.Join(baseDir(), "templates", "partials", "*.html"),
		filepath.Join(baseDir(), "templates", "pages", "public", "buyer_stats.html"),
	)
	if err != nil {
		t.Fatalf("failed to parse templates: %v", err)
	}
	return tmpl
}

func TestTemplatesParse(t *testing.T) {
	setupRendererAll(t)
}

func TestBuyerStatsTemplate(t *testing.T) {
	tmpl := setupRendererBuyerStats(t)

	var buf bytes.Buffer
	err := tmpl.Write(&buf, "buyer_stats.html", gin.H{
		"title":   "@testuser - Project Syrup",
		"handle":  "testuser",
		"stats": models.BuyerStats{
			InstagramHandle:     "testuser",
			TotalWafflesEntered: 5,
			TotalWins:           1,
			TotalLosses:         4,
			TotalSpotsClaimed:   10,
		},
		"history": []models.BuyerWaffleHistory{
			{
				Slug:        "test-waffle",
				Title:       "Test Waffle",
				SpotPrice:   10,
				Status:      "completed",
				SpotNumbers: []int{1, 2},
				IsWinner:    true,
			},
		},
		"winRate": 20,
	})
	if err != nil {
		t.Fatalf("failed to render buyer_stats.html: %v", err)
	}

	out := buf.String()
	if len(out) == 0 {
		t.Fatal("rendered output is empty")
	}
	if !bytes.Contains(buf.Bytes(), []byte("testuser")) {
		if !bytes.Contains(buf.Bytes(), []byte("@testuser")) {
			t.Fatalf("rendered output missing handle: %s", out)
		}
	}
	if !bytes.Contains(buf.Bytes(), []byte("Winner")) {
		t.Fatal("rendered output missing Winner badge")
	}
	if !bytes.Contains(buf.Bytes(), []byte("Waffles Entered")) {
		t.Fatal("rendered output missing stats cards")
	}
}

func TestBuyerStatsTemplateActive(t *testing.T) {
	tmpl := setupRendererBuyerStats(t)

	var buf bytes.Buffer
	err := tmpl.Write(&buf, "buyer_stats.html", gin.H{
		"title":   "@activeuser - Project Syrup",
		"handle":  "activeuser",
		"stats": models.BuyerStats{
			InstagramHandle:     "activeuser",
			TotalWafflesEntered: 3,
			TotalWins:           0,
			TotalLosses:         0,
			TotalSpotsClaimed:   6,
		},
		"history": []models.BuyerWaffleHistory{
			{
				Slug:        "active-waffle",
				Title:       "Active Waffle",
				SpotPrice:   5,
				Status:      "active",
				SpotNumbers: []int{3, 4},
				IsWinner:    false,
			},
		},
		"winRate": 0,
	})
	if err != nil {
		t.Fatalf("failed to render buyer_stats.html: %v", err)
	}

	if !bytes.Contains(buf.Bytes(), []byte("Active")) {
		t.Fatal("rendered output missing Active badge")
	}
}
