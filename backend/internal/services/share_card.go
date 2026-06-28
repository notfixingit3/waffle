package services

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fogleman/gg"
	"github.com/syrup/backend/internal/models"
)

const (
	// ShareCardFormatStory renders a 1080x1920 vertical share card.
	ShareCardFormatStory = "story"
	// ShareCardFormatSquare renders a 1080x1080 square share card.
	ShareCardFormatSquare = "square"
)

// ShareCardCacheDir is the directory where generated share card PNGs are cached.
// This mirrors the value in handlers.ShareCardCacheDir; the two should be kept
// in sync until handlers can reference this package-level variable.
var ShareCardCacheDir = "cmd/api/static/cache/share-cards"

var (
	shareCardBoldFontPath    string
	shareCardRegularFontPath string
	shareCardFontInitErr     error
	shareCardFontOnce        sync.Once
)

// initShareCardFonts writes the embedded fonts to temporary files once and
// caches their paths for reuse across card generations.
func initShareCardFonts() {
	shareCardBoldFontPath, shareCardFontInitErr = writeTempFont(ShareCardInterBoldTTF)
	if shareCardFontInitErr != nil {
		return
	}
	shareCardRegularFontPath, shareCardFontInitErr = writeTempFont(ShareCardInterRegularTTF)
}

// CleanupShareCardFonts removes the temporary font files written at startup.
// Call this once during graceful shutdown.
func CleanupShareCardFonts() {
	if shareCardBoldFontPath != "" {
		if err := os.Remove(shareCardBoldFontPath); err != nil {
			log.Printf("share_card: failed to remove bold font temp file: %v", err)
		}
	}
	if shareCardRegularFontPath != "" {
		if err := os.Remove(shareCardRegularFontPath); err != nil {
			log.Printf("share_card: failed to remove regular font temp file: %v", err)
		}
	}
}

// GenerateShareCard renders a downloadable PNG share card for a waffle.
// Supported formats are "story" (1080x1920) and "square" (1080x1080);
// any other value defaults to "story".
func GenerateShareCard(waffle *models.Waffle, format string) ([]byte, error) {
	shareCardFontOnce.Do(initShareCardFonts)
	if shareCardFontInitErr != nil {
		return nil, fmt.Errorf("prepare fonts: %w", shareCardFontInitErr)
	}
	boldPath := shareCardBoldFontPath
	regularPath := shareCardRegularFontPath

	format = strings.ToLower(strings.TrimSpace(format))
	width, height := shareCardDimensions(format)

	dc := gg.NewContext(width, height)
	dc.SetHexColor("#1a1512")
	dc.Clear()

	// Fetch spot stats for spots remaining text.
	stats, err := GetWaffleStats(waffle.ID)
	if err != nil {
		return nil, fmt.Errorf("get waffle stats: %w", err)
	}
	spotsRemaining, _ := toInt(stats["spots_remaining"])

	// Load and scale emoji assets.
	waffleImg, err := ShareCardEmojiWafflePNG()
	if err != nil {
		return nil, fmt.Errorf("load waffle emoji: %w", err)
	}
	waffleImg = scaleImage(waffleImg, 180, 180)

	pointImg, err := ShareCardEmojiPointDownPNG()
	if err != nil {
		return nil, fmt.Errorf("load point down emoji: %w", err)
	}
	pointImg = scaleImage(pointImg, 140, 140)

	// Choose vertical layout based on format.
	var (
		emojiY, titleY, priceY, spotsY, pointY, urlY, brandY, tagY float64
		titleSize, bodySize                                        float64
	)
	if format == ShareCardFormatSquare {
		emojiY = 220
		titleY = 480
		priceY = 640
		spotsY = 690
		pointY = 820
		urlY = 960
		brandY = 1060
		tagY = 1100
		titleSize = 64
		bodySize = 40
	} else {
		emojiY = 320
		titleY = 720
		priceY = 980
		spotsY = 1040
		pointY = 1240
		urlY = 1460
		brandY = 1680
		tagY = 1730
		titleSize = 72
		bodySize = 48
	}

	centerX := float64(width) / 2

	// Waffle emoji at the top.
	dc.DrawImageAnchored(waffleImg, width/2, int(emojiY), 0.5, 0.5)

	// Waffle title in bold Inter, wrapped to fit.
	if err := dc.LoadFontFace(boldPath, titleSize); err != nil {
		return nil, fmt.Errorf("load bold font face: %w", err)
	}
	dc.SetHexColor("#f5f5f4")
	dc.DrawStringWrapped(waffle.Title, centerX, titleY, 0.5, 0.5, float64(width)*0.85, 1.2, gg.AlignCenter)

	// Price and spots remaining.
	if err := dc.LoadFontFace(regularPath, bodySize); err != nil {
		return nil, fmt.Errorf("load regular font face: %w", err)
	}
	dc.SetHexColor("#d6d3d1")
	dc.DrawStringAnchored(fmt.Sprintf("$%d per spot", waffle.SpotPrice), centerX, priceY, 0.5, 0.5)
	dc.DrawStringAnchored(fmt.Sprintf("%d spots left", spotsRemaining), centerX, spotsY, 0.5, 0.5)

	// Point-down emoji drawing attention to the claim URL.
	dc.DrawImageAnchored(pointImg, width/2, int(pointY), 0.5, 0.5)

	// Public claim URL.
	if err := dc.LoadFontFace(regularPath, 40); err != nil {
		return nil, fmt.Errorf("load url font face: %w", err)
	}
	dc.SetHexColor("#f59e0b")
	dc.DrawStringAnchored(fmt.Sprintf("/waffle/%s", waffle.Slug), centerX, urlY, 0.5, 0.5)

	// Project Syrup branding at the bottom.
	if err := dc.LoadFontFace(boldPath, 42); err != nil {
		return nil, fmt.Errorf("load brand font face: %w", err)
	}
	dc.SetHexColor("#a8a29e")
	dc.DrawStringAnchored("Project Syrup", centerX, brandY, 0.5, 0.5)

	if err := dc.LoadFontFace(regularPath, 28); err != nil {
		return nil, fmt.Errorf("load tagline font face: %w", err)
	}
	dc.DrawStringAnchored("The Waffle Maker", centerX, tagY, 0.5, 0.5)

	var buf bytes.Buffer
	if err := png.Encode(&buf, dc.Image()); err != nil {
		return nil, fmt.Errorf("encode png: %w", err)
	}

	return buf.Bytes(), nil
}

// shareCardDimensions returns the pixel dimensions for the requested format.
func shareCardDimensions(format string) (width, height int) {
	switch format {
	case ShareCardFormatSquare:
		return 1080, 1080
	case ShareCardFormatStory, "":
		return 1080, 1920
	default:
		return 1080, 1920
	}
}

func writeTempFont(getFont func() ([]byte, error)) (string, error) {
	data, err := getFont()
	if err != nil {
		return "", fmt.Errorf("read font bytes: %w", err)
	}

	f, err := os.CreateTemp("", "share-card-*.ttf")
	if err != nil {
		return "", fmt.Errorf("create temp font file: %w", err)
	}
	path := f.Name()

	if _, err := f.Write(data); err != nil {
		cleanupErr := errors.Join(f.Close(), os.Remove(path))
		return "", fmt.Errorf("write temp font file: %w", errors.Join(err, cleanupErr))
	}
	if err := f.Close(); err != nil {
		return "", fmt.Errorf("close temp font file: %w", errors.Join(err, os.Remove(path)))
	}

	return path, nil
}

// InvalidateShareCardCache removes the cached story and square PNG files for
// the given waffle slug. Missing files are treated as a no-op.
func InvalidateShareCardCache(slug string) error {
	var lastErr error
	for _, format := range []string{ShareCardFormatStory, ShareCardFormatSquare} {
		path := filepath.Join(ShareCardCacheDir, fmt.Sprintf("%s-%s.png", slug, format))
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			lastErr = err
		}
	}
	if lastErr != nil {
		return fmt.Errorf("invalidate share card cache: %w", lastErr)
	}
	return nil
}

// scaleImage returns a scaled copy of src using nearest-neighbor sampling.
func scaleImage(src image.Image, width, height int) image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	sb := src.Bounds()
	sw := sb.Dx()
	sh := sb.Dy()

	for y := range height {
		for x := range width {
			sx := sb.Min.X + (x*sw)/width
			sy := sb.Min.Y + (y*sh)/height
			dst.Set(x, y, src.At(sx, sy))
		}
	}
	return dst
}
