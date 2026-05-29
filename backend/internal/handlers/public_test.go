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

func setupRendererFooter(t *testing.T) *renderer.Renderer {
	t.Helper()
	tmpl := renderer.New(nil)
	err := tmpl.AddFromFiles(
		filepath.Join(baseDir(), "templates", "layouts", "base.html"),
		filepath.Join(baseDir(), "templates", "partials", "footer.html"),
	)
	if err != nil {
		t.Fatalf("failed to parse templates: %v", err)
	}
	return tmpl
}

func TestFooterTemplate(t *testing.T) {
	tmpl := setupRendererFooter(t)

	var buf bytes.Buffer
	err := tmpl.Write(&buf, "footer.html", gin.H{
		"ServerTime":    "2:32 PM",
		"TotalWaffles":  5,
		"ActiveWaffles": 3,
		"Version":       "v1.0.0",
		"DevMode":       false,
	})
	if err != nil {
		t.Fatalf("failed to render footer.html: %v", err)
	}

	out := buf.String()
	if len(out) == 0 {
		t.Fatal("rendered output is empty")
	}
	if !bytes.Contains(buf.Bytes(), []byte("UTC 2:32 PM")) {
		t.Fatal("rendered output missing server UTC time")
	}
	if !bytes.Contains(buf.Bytes(), []byte("3 active / 5 total")) {
		t.Fatal("rendered output missing waffle counts")
	}
	if !bytes.Contains(buf.Bytes(), []byte("Project Syrup")) {
		t.Fatal("rendered output missing Project Syrup link text")
	}
	if !bytes.Contains(buf.Bytes(), []byte("github.com/notfixingit3/waffle")) {
		t.Fatal("rendered output missing GitHub link")
	}
	if !bytes.Contains(buf.Bytes(), []byte("v1.0.0")) {
		t.Fatal("rendered output missing version")
	}
}

func TestFooterTemplateDevMode(t *testing.T) {
	tmpl := setupRendererFooter(t)

	var buf bytes.Buffer
	err := tmpl.Write(&buf, "footer.html", gin.H{
		"ServerTime":    "2:32 PM",
		"TotalWaffles":  5,
		"ActiveWaffles": 3,
		"Version":       "v1.0.0-dev",
		"DevMode":       true,
	})
	if err != nil {
		t.Fatalf("failed to render footer.html: %v", err)
	}

	if !bytes.Contains(buf.Bytes(), []byte("DEV")) {
		t.Fatal("rendered output missing DEV badge in dev mode")
	}
	if bytes.Contains(buf.Bytes(), []byte("v1.0.0-dev")) {
		t.Fatal("rendered output should not show version in dev mode")
	}
}
