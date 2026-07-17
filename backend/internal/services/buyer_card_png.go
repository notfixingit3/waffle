package services

import (
	"bytes"
	"fmt"
	"image/png"
	"strings"

	"github.com/fogleman/gg"
)

// buyerCardMaxTrophies is the maximum number of trophy names rendered on the
// buyer card PNG. Additional trophies collapse into a "+N more" line.
const buyerCardMaxTrophies = 5

// GenerateBuyerCardPNG renders a PNG buyer card mirroring buyer_card.html.
// It is a pure function of the precomputed card data: it never touches the
// database. Supported formats are "story" (1080x1920) and "square"
// (1080x1080); any other value defaults to "story".
func GenerateBuyerCardPNG(card *BuyerCardData, format string) ([]byte, error) {
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

	trophyImg, err := ShareCardEmojiTrophyPNG()
	if err != nil {
		return nil, fmt.Errorf("load trophy emoji: %w", err)
	}

	// Zero-state (unknown buyer): nil Stats renders zero values. Win rate
	// truncation and luck rating text mirror BuyerCardPage exactly.
	var wins, losses, spotsClaimed, winRate int
	if card.Stats != nil {
		wins = card.Stats.TotalWins
		losses = card.Stats.TotalLosses
		spotsClaimed = card.Stats.TotalSpotsClaimed
		if spotsClaimed > 0 {
			winRate = int(card.WinRate * 100)
		}
	}

	luckRatingDisplay := "Even"
	luckColor := "#22c55e"
	if card.LuckRating > 0 {
		luckRatingDisplay = fmt.Sprintf("+%.1f%%", card.LuckRating*100)
	} else if card.LuckRating < 0 {
		luckRatingDisplay = fmt.Sprintf("%.1f%%", card.LuckRating*100)
		luckColor = "#ef4444"
	}

	// Choose vertical layout based on format.
	var (
		eyebrowY, handleY                              float64
		statNumY1, statLabelY1, statNumY2, statLabelY2 float64
		luckY, luckCaptionY                            float64
		trophyHeaderY, trophyEmojiY, trophyListY       float64
		brandY, tagY                                   float64
		eyebrowSize, handleSize                        float64
		statNumSize, statLabelSize                     float64
		luckSize, luckCaptionSize                      float64
		trophyHeaderSize, trophySize, trophyLine       float64
		brandSize, tagSize                             float64
		trophyEmojiSize                                int
	)
	if format == ShareCardFormatSquare {
		eyebrowY = 64
		handleY = 140
		statNumY1 = 270
		statLabelY1 = 318
		statNumY2 = 415
		statLabelY2 = 463
		luckY = 550
		luckCaptionY = 592
		trophyHeaderY = 662
		trophyEmojiY = 725
		trophyListY = 785
		brandY = 1015
		tagY = 1045
		eyebrowSize = 20
		handleSize = 58
		statNumSize = 50
		statLabelSize = 20
		luckSize = 42
		luckCaptionSize = 18
		trophyHeaderSize = 20
		trophySize = 26
		trophyLine = 32
		brandSize = 24
		tagSize = 18
		trophyEmojiSize = 90
	} else {
		eyebrowY = 170
		handleY = 290
		statNumY1 = 520
		statLabelY1 = 580
		statNumY2 = 740
		statLabelY2 = 800
		luckY = 940
		luckCaptionY = 1000
		trophyHeaderY = 1120
		trophyEmojiY = 1210
		trophyListY = 1330
		brandY = 1820
		tagY = 1875
		eyebrowSize = 26
		handleSize = 84
		statNumSize = 68
		statLabelSize = 26
		luckSize = 56
		luckCaptionSize = 24
		trophyHeaderSize = 26
		trophySize = 38
		trophyLine = 48
		brandSize = 42
		tagSize = 28
		trophyEmojiSize = 130
	}

	centerX := float64(width) / 2
	leftX := float64(width) * 0.27
	rightX := float64(width) * 0.73

	trophyImg = scaleImage(trophyImg, trophyEmojiSize, trophyEmojiSize)

	// Eyebrow + @handle header.
	if err := dc.LoadFontFace(regularPath, eyebrowSize); err != nil {
		return nil, fmt.Errorf("load eyebrow font face: %w", err)
	}
	dc.SetHexColor("#f59e0b")
	dc.DrawStringAnchored("WAFFLE CARD", centerX, eyebrowY, 0.5, 0.5)

	if err := dc.LoadFontFace(boldPath, handleSize); err != nil {
		return nil, fmt.Errorf("load handle font face: %w", err)
	}
	dc.SetHexColor("#f5f5f4")
	dc.DrawStringWrapped("@"+card.InstagramHandle, centerX, handleY, 0.5, 0.5, float64(width)*0.9, 1.2, gg.AlignCenter)

	// Stats block: wins / losses / spots claimed / win rate.
	if err := dc.LoadFontFace(boldPath, statNumSize); err != nil {
		return nil, fmt.Errorf("load stat font face: %w", err)
	}
	dc.SetHexColor("#22c55e")
	dc.DrawStringAnchored(fmt.Sprintf("%d", wins), leftX, statNumY1, 0.5, 0.5)
	dc.SetHexColor("#d6d3d1")
	dc.DrawStringAnchored(fmt.Sprintf("%d", losses), rightX, statNumY1, 0.5, 0.5)
	dc.SetHexColor("#f5f5f4")
	dc.DrawStringAnchored(fmt.Sprintf("%d", spotsClaimed), leftX, statNumY2, 0.5, 0.5)
	dc.SetHexColor("#f59e0b")
	dc.DrawStringAnchored(fmt.Sprintf("%d%%", winRate), rightX, statNumY2, 0.5, 0.5)

	if err := dc.LoadFontFace(regularPath, statLabelSize); err != nil {
		return nil, fmt.Errorf("load stat label font face: %w", err)
	}
	dc.SetHexColor("#a8a29e")
	dc.DrawStringAnchored("WINS", leftX, statLabelY1, 0.5, 0.5)
	dc.DrawStringAnchored("LOSSES", rightX, statLabelY1, 0.5, 0.5)
	dc.DrawStringAnchored("SPOTS CLAIMED", leftX, statLabelY2, 0.5, 0.5)
	dc.DrawStringAnchored("WIN RATE", rightX, statLabelY2, 0.5, 0.5)

	// Luck rating badge text + caption.
	if err := dc.LoadFontFace(boldPath, luckSize); err != nil {
		return nil, fmt.Errorf("load luck font face: %w", err)
	}
	dc.SetHexColor(luckColor)
	dc.DrawStringAnchored(luckRatingDisplay, centerX, luckY, 0.5, 0.5)

	if err := dc.LoadFontFace(regularPath, luckCaptionSize); err != nil {
		return nil, fmt.Errorf("load luck caption font face: %w", err)
	}
	dc.SetHexColor("#a8a29e")
	dc.DrawStringAnchored("Actual vs. expected win rate", centerX, luckCaptionY, 0.5, 0.5)

	// Trophy case.
	if err := dc.LoadFontFace(regularPath, trophyHeaderSize); err != nil {
		return nil, fmt.Errorf("load trophy header font face: %w", err)
	}
	dc.SetHexColor("#a8a29e")
	dc.DrawStringAnchored("TROPHY CASE", centerX, trophyHeaderY, 0.5, 0.5)

	dc.DrawImageAnchored(trophyImg, width/2, int(trophyEmojiY), 0.5, 0.5)

	if err := dc.LoadFontFace(boldPath, trophySize); err != nil {
		return nil, fmt.Errorf("load trophy font face: %w", err)
	}
	trophyY := trophyListY
	if len(card.Trophies) == 0 {
		dc.SetHexColor("#a8a29e")
		dc.DrawStringAnchored("No trophies yet", centerX, trophyY, 0.5, 0.5)
	} else {
		shown := card.Trophies
		if len(shown) > buyerCardMaxTrophies {
			shown = shown[:buyerCardMaxTrophies]
		}
		wrapWidth := float64(width) * 0.8
		dc.SetHexColor("#f5f5f4")
		for _, name := range shown {
			for _, line := range dc.WordWrap(name, wrapWidth) {
				dc.DrawStringAnchored(line, centerX, trophyY, 0.5, 0.5)
				trophyY += trophyLine
			}
			trophyY += trophyLine * 0.25
		}
		if extra := len(card.Trophies) - buyerCardMaxTrophies; extra > 0 {
			dc.SetHexColor("#a8a29e")
			dc.DrawStringAnchored(fmt.Sprintf("+%d more", extra), centerX, trophyY, 0.5, 0.5)
		}
	}

	// Project Syrup branding at the bottom.
	if err := dc.LoadFontFace(boldPath, brandSize); err != nil {
		return nil, fmt.Errorf("load brand font face: %w", err)
	}
	dc.SetHexColor("#a8a29e")
	dc.DrawStringAnchored("Project Syrup", centerX, brandY, 0.5, 0.5)

	if err := dc.LoadFontFace(regularPath, tagSize); err != nil {
		return nil, fmt.Errorf("load tagline font face: %w", err)
	}
	dc.DrawStringAnchored("The Waffle Maker", centerX, tagY, 0.5, 0.5)

	var buf bytes.Buffer
	if err := png.Encode(&buf, dc.Image()); err != nil {
		return nil, fmt.Errorf("encode png: %w", err)
	}

	return buf.Bytes(), nil
}
