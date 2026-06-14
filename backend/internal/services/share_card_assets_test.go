package services

import (
	"image"
	"testing"
)

func TestShareCardAssets_EmojiWafflePNG(t *testing.T) {
	img, err := ShareCardEmojiWafflePNG()
	if err != nil {
		t.Fatalf("ShareCardEmojiWafflePNG() returned error: %v", err)
	}
	if img == nil {
		t.Fatal("ShareCardEmojiWafflePNG() returned nil image")
	}
	bounds := img.Bounds()
	if bounds.Dx() == 0 || bounds.Dy() == 0 {
		t.Fatal("ShareCardEmojiWafflePNG() returned zero-dimension image")
	}
	// Verify it's a valid RGBA/PNG image
	if _, ok := img.(*image.RGBA); !ok {
		// Accept NRGBA or other common PNG decode types too
		t.Logf("ShareCardEmojiWafflePNG() decoded as %T (acceptable)", img)
	}
}

func TestShareCardAssets_EmojiPointDownPNG(t *testing.T) {
	img, err := ShareCardEmojiPointDownPNG()
	if err != nil {
		t.Fatalf("ShareCardEmojiPointDownPNG() returned error: %v", err)
	}
	if img == nil {
		t.Fatal("ShareCardEmojiPointDownPNG() returned nil image")
	}
	bounds := img.Bounds()
	if bounds.Dx() == 0 || bounds.Dy() == 0 {
		t.Fatal("ShareCardEmojiPointDownPNG() returned zero-dimension image")
	}
}

func TestShareCardAssets_InterRegularTTF(t *testing.T) {
	data, err := ShareCardInterRegularTTF()
	if err != nil {
		t.Fatalf("ShareCardInterRegularTTF() returned error: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("ShareCardInterRegularTTF() returned empty byte slice")
	}
	// TTF files start with \x00\x01\x00\x00\x00
	if len(data) < 4 || data[0] != 0x00 || data[1] != 0x01 || data[2] != 0x00 || data[3] != 0x00 {
		t.Logf("ShareCardInterRegularTTF() first bytes: %x (expected 00010000)", data[:4])
	}
}

func TestShareCardAssets_InterBoldTTF(t *testing.T) {
	data, err := ShareCardInterBoldTTF()
	if err != nil {
		t.Fatalf("ShareCardInterBoldTTF() returned error: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("ShareCardInterBoldTTF() returned empty byte slice")
	}
	if len(data) < 4 || data[0] != 0x00 || data[1] != 0x01 || data[2] != 0x00 || data[3] != 0x00 {
		t.Logf("ShareCardInterBoldTTF() first bytes: %x (expected 00010000)", data[:4])
	}
}

func TestShareCardAssets_EmojiWafflePNGReader(t *testing.T) {
	r, err := ShareCardEmojiWafflePNGReader()
	if err != nil {
		t.Fatalf("ShareCardEmojiWafflePNGReader() returned error: %v", err)
	}
	defer r.Close()
	img, _, err := image.Decode(r)
	if err != nil {
		t.Fatalf("image.Decode from reader returned error: %v", err)
	}
	if img == nil {
		t.Fatal("image.Decode from reader returned nil")
	}
}

func TestShareCardAssets_EmojiPointDownPNGReader(t *testing.T) {
	r, err := ShareCardEmojiPointDownPNGReader()
	if err != nil {
		t.Fatalf("ShareCardEmojiPointDownPNGReader() returned error: %v", err)
	}
	defer r.Close()
	img, _, err := image.Decode(r)
	if err != nil {
		t.Fatalf("image.Decode from reader returned error: %v", err)
	}
	if img == nil {
		t.Fatal("image.Decode from reader returned nil")
	}
}
