package services

import (
	"bytes"
	"image"
	"image/png"
	"strings"
	"testing"

	"github.com/syrup/backend/internal/models"
)

func fullBuyerCardFixture() *BuyerCardData {
	return &BuyerCardData{
		InstagramHandle: "coolcollector",
		Stats: &models.BuyerStats{
			InstagramHandle:     "coolcollector",
			TotalWafflesEntered: 10,
			TotalWins:           3,
			TotalLosses:         7,
			TotalSpotsClaimed:   20,
		},
		WinRate:         0.15,
		ExpectedWinRate: 0.10,
		LuckRating:      0.05,
		Trophies:        []string{"Pocket Monstor", "Wubble", "Jelli"},
	}
}

func decodeBuyerCardPNG(t *testing.T, data []byte) image.Image {
	t.Helper()
	if len(data) == 0 {
		t.Fatal("GenerateBuyerCardPNG returned empty PNG bytes")
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("decode PNG: %v", err)
	}
	return img
}

func assertBuyerCardDimensions(t *testing.T, img image.Image, format string) {
	t.Helper()
	wantW, wantH := shareCardDimensions(format)
	b := img.Bounds()
	if b.Dx() != wantW || b.Dy() != wantH {
		t.Errorf("format %q: dimensions = %dx%d, want %dx%d", format, b.Dx(), b.Dy(), wantW, wantH)
	}
}

func TestGenerateBuyerCardPNG_FullBuyer(t *testing.T) {
	// Given a buyer with stats and three trophies
	card := fullBuyerCardFixture()

	// When the card is rendered in both formats
	for _, format := range []string{ShareCardFormatStory, ShareCardFormatSquare} {
		pngBytes, err := GenerateBuyerCardPNG(card, format)
		if err != nil {
			t.Fatalf("format %q: GenerateBuyerCardPNG returned error: %v", format, err)
		}

		// Then the PNG decodes with the format's dimensions
		assertBuyerCardDimensions(t, decodeBuyerCardPNG(t, pngBytes), format)
	}
}

func TestGenerateBuyerCardPNG_NilStatsZeroState(t *testing.T) {
	// Given an unknown buyer (nil Stats, nil Trophies)
	card := &BuyerCardData{InstagramHandle: "ghostbuyer"}

	// When the card is rendered in both formats
	for _, format := range []string{ShareCardFormatStory, ShareCardFormatSquare} {
		pngBytes, err := GenerateBuyerCardPNG(card, format)
		if err != nil {
			t.Fatalf("format %q: zero-state render returned error: %v", format, err)
		}

		// Then it still renders a decodable PNG without error
		assertBuyerCardDimensions(t, decodeBuyerCardPNG(t, pngBytes), format)
	}
}

func TestGenerateBuyerCardPNG_TrophyOverflow(t *testing.T) {
	// Given a buyer with 8 trophies (over the 5-trophy render cap)
	card := fullBuyerCardFixture()
	card.Trophies = []string{"Alpha", "Beta", "Gamma", "Delta", "Epsilon", "Zeta", "Eta", "Theta"}

	// When the card is rendered in both formats
	for _, format := range []string{ShareCardFormatStory, ShareCardFormatSquare} {
		pngBytes, err := GenerateBuyerCardPNG(card, format)
		if err != nil {
			t.Fatalf("format %q: overflow render returned error: %v", format, err)
		}

		// Then the overflow renders without error at the right dimensions
		assertBuyerCardDimensions(t, decodeBuyerCardPNG(t, pngBytes), format)
	}
}

func TestGenerateBuyerCardPNG_InvalidFormatDefaultsToStory(t *testing.T) {
	// Given a format string that matches no known format
	card := fullBuyerCardFixture()

	// When the card is rendered
	pngBytes, err := GenerateBuyerCardPNG(card, "portrait")
	if err != nil {
		t.Fatalf("invalid format render returned error: %v", err)
	}

	// Then it falls back to story dimensions (shareCardDimensions default)
	img := decodeBuyerCardPNG(t, pngBytes)
	b := img.Bounds()
	if b.Dx() != 1080 || b.Dy() != 1920 {
		t.Errorf("invalid format: dimensions = %dx%d, want story 1080x1920", b.Dx(), b.Dy())
	}
}

func TestGenerateBuyerCardPNG_HandleEdgeCases(t *testing.T) {
	// Given an empty handle and a 30-character handle (HTML maxlength boundary)
	for _, handle := range []string{"", strings.Repeat("a", 30)} {
		card := fullBuyerCardFixture()
		card.InstagramHandle = handle

		// When the card is rendered
		pngBytes, err := GenerateBuyerCardPNG(card, ShareCardFormatStory)
		if err != nil {
			t.Fatalf("handle %q: render returned error: %v", handle, err)
		}

		// Then it renders without error
		assertBuyerCardDimensions(t, decodeBuyerCardPNG(t, pngBytes), ShareCardFormatStory)
	}
}

func TestGenerateBuyerCardPNG_NoSharedStateBetweenRenders(t *testing.T) {
	// Given two different cards rendered back-to-back
	first, err := GenerateBuyerCardPNG(fullBuyerCardFixture(), ShareCardFormatStory)
	if err != nil {
		t.Fatalf("first render: %v", err)
	}
	second, err := GenerateBuyerCardPNG(&BuyerCardData{InstagramHandle: "ghostbuyer"}, ShareCardFormatStory)
	if err != nil {
		t.Fatalf("second render: %v", err)
	}

	// Then both decode independently and produce different bytes
	assertBuyerCardDimensions(t, decodeBuyerCardPNG(t, first), ShareCardFormatStory)
	assertBuyerCardDimensions(t, decodeBuyerCardPNG(t, second), ShareCardFormatStory)
	if bytes.Equal(first, second) {
		t.Error("two different cards rendered identical bytes; image state may leak between renders")
	}
}
