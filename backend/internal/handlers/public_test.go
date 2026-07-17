package handlers

import (
	"bytes"
	"context"
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

// buyerCardPNGHandlePrefix identifies buyer-card-PNG test handles for cleanup.
const buyerCardPNGHandlePrefix = "buyerpng"

// createBuyerWithStats creates a waffle, claims two spots for buyerHandle,
// fills the rest with other test handles, marks every spot paid, and completes
// the waffle with the buyer winning spot 1 — leaving the buyer with a real row
// in buyer_stats (mirrors the services package's createCompletedWaffleForBuyer,
// which is not importable across packages).
func createBuyerWithStats(t *testing.T, buyerHandle string) *models.Waffle {
	t.Helper()

	waffle := createTestWaffleForShareCard(t, false)

	if err := services.ClaimSpots(waffle.ID, []int{1, 2}, buyerHandle); err != nil {
		t.Fatalf("claim buyer spots: %v", err)
	}
	for i := 3; i <= 10; i++ {
		filler := fmt.Sprintf("%sother%d", buyerCardPNGHandlePrefix, i)
		if err := services.ClaimSpots(waffle.ID, []int{i}, filler); err != nil {
			t.Fatalf("claim filler spot %d: %v", i, err)
		}
	}

	spots, err := services.GetSpotsByWaffleID(waffle.ID)
	if err != nil {
		t.Fatalf("get spots: %v", err)
	}
	for _, spot := range spots {
		if spot.Status == models.SpotStatusPending {
			if err := services.MarkSpotPaid(spot.ID); err != nil {
				t.Fatalf("mark spot %d paid: %v", spot.Number, err)
			}
		}
	}

	if err := services.SetWinner(waffle.ID, []int{1}); err != nil {
		t.Fatalf("set winner: %v", err)
	}

	return waffle
}

// cleanupBuyerCardPNGTestData removes buyer_stats and users rows created by
// buyer-card-PNG tests. Waffle cleanup goes through deleteTestWaffleForShareCard.
func cleanupBuyerCardPNGTestData(t *testing.T) {
	t.Helper()
	if _, err := db.Pool.Exec(context.Background(),
		`DELETE FROM buyer_stats WHERE instagram_handle LIKE $1`, buyerCardPNGHandlePrefix+"%"); err != nil {
		t.Logf("cleanup buyer_stats failed: %v", err)
	}
	if _, err := db.Pool.Exec(context.Background(),
		`DELETE FROM users WHERE instagram_handle LIKE $1`, buyerCardPNGHandlePrefix+"%"); err != nil {
		t.Logf("cleanup users failed: %v", err)
	}
}

func TestBuyerCardPNG(t *testing.T) {
	setupShareCardDB(t)

	cacheDir := t.TempDir()
	originalCacheDir := services.ShareCardCacheDir
	services.ShareCardCacheDir = cacheDir
	defer func() { services.ShareCardCacheDir = originalCacheDir }()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/buyer/:handle/card.png", BuyerCardPNG)

	// Cleanup order mirrors the services package tests: waffles first, then
	// buyer_stats/users rows (defers run LIFO).
	defer cleanupBuyerCardPNGTestData(t)

	buyerHandle := buyerCardPNGHandlePrefix + uuid.New().String()[:8]
	waffle := createBuyerWithStats(t, buyerHandle)
	defer deleteTestWaffleForShareCard(t, waffle)

	storyCacheName := fmt.Sprintf("buyer-%s-story.png", buyerHandle)

	t.Run("known buyer returns story PNG", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/buyer/"+buyerHandle+"/card.png", nil)
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
		req := httptest.NewRequest(http.MethodGet, "/buyer/"+buyerHandle+"/card.png?format=square", nil)
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
		cachePath := filepath.Join(cacheDir, storyCacheName)
		if _, err := os.Stat(cachePath); os.IsNotExist(err) {
			t.Fatalf("expected cached file at %s", cachePath)
		}
	})

	t.Run("serves cached PNG on subsequent request", func(t *testing.T) {
		cachePath := filepath.Join(cacheDir, storyCacheName)
		diskBytes, err := os.ReadFile(cachePath)
		if err != nil {
			t.Fatalf("read cache file: %v", err)
		}
		infoBefore, err := os.Stat(cachePath)
		if err != nil {
			t.Fatalf("stat cache file: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/buyer/"+buyerHandle+"/card.png", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		if ct := w.Header().Get("Content-Type"); ct != "image/png" {
			t.Fatalf("expected image/png, got %s", ct)
		}
		if !bytes.Equal(w.Body.Bytes(), diskBytes) {
			t.Fatal("response body does not match the cached file on disk")
		}

		infoAfter, err := os.Stat(cachePath)
		if err != nil {
			t.Fatalf("stat cache file after request: %v", err)
		}
		if !infoAfter.ModTime().Equal(infoBefore.ModTime()) {
			t.Fatal("cache file was rewritten on a cache-hit request")
		}
	})

	t.Run("unknown buyer returns zero-state PNG without caching", func(t *testing.T) {
		unknownHandle := buyerCardPNGHandlePrefix + "unknown" + uuid.New().String()[:8]
		req := httptest.NewRequest(http.MethodGet, "/buyer/"+unknownHandle+"/card.png", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		if ct := w.Header().Get("Content-Type"); ct != "image/png" {
			t.Fatalf("expected image/png, got %s", ct)
		}
		if _, err := png.Decode(bytes.NewReader(w.Body.Bytes())); err != nil {
			t.Fatalf("failed to decode zero-state png: %v", err)
		}

		entries, err := os.ReadDir(cacheDir)
		if err != nil {
			t.Fatalf("read cache dir: %v", err)
		}
		for _, entry := range entries {
			if strings.Contains(entry.Name(), unknownHandle) {
				t.Fatalf("zero-state card must not be cached, found %s", entry.Name())
			}
		}
	})

	t.Run("invalid handles return 404", func(t *testing.T) {
		targets := []string{
			"/buyer/evil-handle/card.png",                     // dash not in charset
			"/buyer/" + strings.Repeat("a", 31) + "/card.png", // over 30 chars
			"/buyer/a%20b/card.png",                           // space after decode
			"/buyer/../../x/card.png",                         // traversal — no route match
		}
		for _, target := range targets {
			req := httptest.NewRequest(http.MethodGet, target, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != http.StatusNotFound {
				t.Fatalf("expected 404 for %q, got %d", target, w.Code)
			}
		}

		entries, err := os.ReadDir(cacheDir)
		if err != nil {
			t.Fatalf("read cache dir: %v", err)
		}
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), "buyer-") && !strings.Contains(entry.Name(), buyerHandle) {
				t.Fatalf("invalid handle produced cache file %s", entry.Name())
			}
		}
	})
}
