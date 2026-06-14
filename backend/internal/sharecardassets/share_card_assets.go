package sharecardassets

import (
	"embed"
	"image"
	"io"

	_ "image/png"
)

//go:embed emoji/waffle.png emoji/point_down.png fonts/Inter-Regular.ttf fonts/Inter-Bold.ttf
var assets embed.FS

// EmojiWafflePNG returns the waffle emoji PNG as a decoded image.
func EmojiWafflePNG() (image.Image, error) {
	f, err := assets.Open("emoji/waffle.png")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	return img, err
}

// EmojiPointDownPNG returns the point-down emoji PNG as a decoded image.
func EmojiPointDownPNG() (image.Image, error) {
	f, err := assets.Open("emoji/point_down.png")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	return img, err
}

// InterRegularTTF returns the Inter Regular TTF font as a byte slice.
func InterRegularTTF() ([]byte, error) {
	return assets.ReadFile("fonts/Inter-Regular.ttf")
}

// InterBoldTTF returns the Inter Bold TTF font as a byte slice.
func InterBoldTTF() ([]byte, error) {
	return assets.ReadFile("fonts/Inter-Bold.ttf")
}

// EmojiWafflePNGReader returns a reader for the waffle emoji PNG.
func EmojiWafflePNGReader() (io.ReadCloser, error) {
	return assets.Open("emoji/waffle.png")
}

// EmojiPointDownPNGReader returns a reader for the point-down emoji PNG.
func EmojiPointDownPNGReader() (io.ReadCloser, error) {
	return assets.Open("emoji/point_down.png")
}
