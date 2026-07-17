package services

import (
	"image"
	"io"

	"github.com/syrup/backend/internal/sharecardassets"
)

// ShareCardEmojiWafflePNG returns the waffle emoji PNG as a decoded image.
func ShareCardEmojiWafflePNG() (image.Image, error) {
	return sharecardassets.EmojiWafflePNG()
}

// ShareCardEmojiPointDownPNG returns the point-down emoji PNG as a decoded image.
func ShareCardEmojiPointDownPNG() (image.Image, error) {
	return sharecardassets.EmojiPointDownPNG()
}

// ShareCardInterRegularTTF returns the Inter Regular TTF font as a byte slice.
func ShareCardInterRegularTTF() ([]byte, error) {
	return sharecardassets.InterRegularTTF()
}

// ShareCardInterBoldTTF returns the Inter Bold TTF font as a byte slice.
func ShareCardInterBoldTTF() ([]byte, error) {
	return sharecardassets.InterBoldTTF()
}

// ShareCardEmojiWafflePNGReader returns a reader for the waffle emoji PNG.
func ShareCardEmojiWafflePNGReader() (io.ReadCloser, error) {
	return sharecardassets.EmojiWafflePNGReader()
}

// ShareCardEmojiPointDownPNGReader returns a reader for the point-down emoji PNG.
func ShareCardEmojiPointDownPNGReader() (io.ReadCloser, error) {
	return sharecardassets.EmojiPointDownPNGReader()
}

// ShareCardEmojiTrophyPNG returns the trophy emoji PNG as a decoded image.
func ShareCardEmojiTrophyPNG() (image.Image, error) {
	return sharecardassets.EmojiTrophyPNG()
}

// ShareCardEmojiTrophyPNGReader returns a reader for the trophy emoji PNG.
func ShareCardEmojiTrophyPNGReader() (io.ReadCloser, error) {
	return sharecardassets.EmojiTrophyPNGReader()
}
