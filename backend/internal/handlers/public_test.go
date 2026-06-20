package handlers

import (
	"bytes"
	"fmt"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/syrup/backend/internal/db"
	"github.com/syrup/backend/internal/models"
	"github.com/syrup/backend/internal/renderer"
	"github.com/syrup/backend/internal/services"
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
		"title":  "@testuser - Project Syrup",
		"handle": "testuser",
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
		"title":  "@activeuser - Project Syrup",
		"handle": "activeuser",
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

func TestFooterTemplateVersion(t *testing.T) {
	tmpl := setupRendererFooter(t)

	var buf bytes.Buffer
	err := tmpl.Write(&buf, "footer.html", gin.H{
		"ServerTime":    "2:32 PM",
		"TotalWaffles":  5,
		"ActiveWaffles": 3,
		"Version":       "v1.0.0-dev",
	})
	if err != nil {
		t.Fatalf("failed to render footer.html: %v", err)
	}

	if !bytes.Contains(buf.Bytes(), []byte("v1.0.0-dev")) {
		t.Fatal("rendered output missing version string")
	}
}

func setupShareCardDB(t *testing.T) {
	t.Helper()
	if db.Pool == nil {
		if _, err := db.Connect(); err != nil {
			t.Skipf("Postgres not available: %v", err)
		}
	}
}

func createTestWaffleForShareCard(t *testing.T, archived bool) *models.Waffle {
	t.Helper()
	req := models.CreateWaffleRequest{
		Title:      "Share Card Test Waffle " + uuid.New().String()[:8],
		TotalSpots: 10,
		SpotPrice:  5,
	}
	waffle, err := services.CreateWaffle(req)
	if err != nil {
		t.Fatalf("create waffle: %v", err)
	}
	if archived {
		if err := services.ArchiveWaffle(waffle.ID, true); err != nil {
			t.Fatalf("archive waffle: %v", err)
		}
	}
	return waffle
}

func deleteTestWaffleForShareCard(t *testing.T, waffle *models.Waffle) {
	t.Helper()
	if waffle == nil {
		return
	}
	if err := services.DeleteWaffle(waffle.ID); err != nil {
		t.Logf("cleanup waffle failed: %v", err)
	}
}

func TestWaffleDetailPage_ArchivedReturns404(t *testing.T) {
	setupShareCardDB(t)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/waffle/:slug", WaffleDetailPage)

	waffle := createTestWaffleForShareCard(t, true) // archived
	defer deleteTestWaffleForShareCard(t, waffle)

	req := httptest.NewRequest(http.MethodGet, "/waffle/"+waffle.Slug, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for archived waffle, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Waffle not found") {
		t.Fatalf("expected 'Waffle not found' in body, got %s", w.Body.String())
	}
}

func TestShareCard(t *testing.T) {
	setupShareCardDB(t)

	cacheDir := t.TempDir()
	originalCacheDir := services.ShareCardCacheDir
	services.ShareCardCacheDir = cacheDir
	defer func() { services.ShareCardCacheDir = originalCacheDir }()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/waffle/:slug/card.png", WaffleShareCardPNG)

	waffle := createTestWaffleForShareCard(t, false)
	defer deleteTestWaffleForShareCard(t, waffle)

	t.Run("active waffle returns story PNG", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/waffle/"+waffle.Slug+"/card.png", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		if ct := w.Header().Get("Content-Type"); ct != "image/png" {
			t.Fatalf("expected Content-Type image/png, got %s", ct)
		}
		if cc := w.Header().Get("Cache-Control"); !strings.Contains(cc, "max-age=3600") {
			t.Fatalf("expected Cache-Control to contain max-age=3600, got %s", cc)
		}

		img, err := png.Decode(bytes.NewReader(w.Body.Bytes()))
		if err != nil {
			t.Fatalf("failed to decode png: %v", err)
		}
		bounds := img.Bounds()
		if bounds.Dx() != 1080 || bounds.Dy() != 1920 {
			t.Fatalf("expected 1080x1920, got %dx%d", bounds.Dx(), bounds.Dy())
		}
	})

	t.Run("square format returns 1080x1080 PNG", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/waffle/"+waffle.Slug+"/card.png?format=square", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		img, err := png.Decode(bytes.NewReader(w.Body.Bytes()))
		if err != nil {
			t.Fatalf("failed to decode png: %v", err)
		}
		bounds := img.Bounds()
		if bounds.Dx() != 1080 || bounds.Dy() != 1080 {
			t.Fatalf("expected 1080x1080, got %dx%d", bounds.Dx(), bounds.Dy())
		}
	})

	t.Run("caches generated PNG to disk", func(t *testing.T) {
		cachePath := filepath.Join(cacheDir, fmt.Sprintf("%s-story.png", waffle.Slug))
		if _, err := os.Stat(cachePath); os.IsNotExist(err) {
			t.Fatalf("expected cached file at %s", cachePath)
		}
	})

	t.Run("serves cached PNG on subsequent request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/waffle/"+waffle.Slug+"/card.png", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		if ct := w.Header().Get("Content-Type"); ct != "image/png" {
			t.Fatalf("expected image/png, got %s", ct)
		}
	})

	archivedWaffle := createTestWaffleForShareCard(t, true)
	defer deleteTestWaffleForShareCard(t, archivedWaffle)

	t.Run("archived waffle returns 404", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/waffle/"+archivedWaffle.Slug+"/card.png", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("missing waffle returns 404", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/waffle/no-such-waffle/card.png", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})
}
