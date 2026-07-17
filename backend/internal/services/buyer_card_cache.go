package services

import (
	"fmt"
	"os"
	"regexp"
)

// buyerCardHandlePattern is the only charset gate between a user-controlled
// Instagram handle and a cache filename. NormalizeInstagramHandle strips "@"
// and lowercases but never validates the charset, so handles arrive here
// already lowercased — uppercase correctly fails this pattern.
var buyerCardHandlePattern = regexp.MustCompile(`^[a-z0-9_.]{1,30}$`)

// buyerCardCacheFileName returns the cache file name for a buyer handle and
// format. ok is false when the handle cannot form a safe file name; callers
// must treat that as "no such file" and never fall back to joining the raw
// handle into a path themselves.
func buyerCardCacheFileName(handle, format string) (string, bool) {
	if !buyerCardHandlePattern.MatchString(handle) {
		return "", false
	}
	return fmt.Sprintf("buyer-%s-%s.png", handle, format), true
}

// InvalidateBuyerCardCache removes the cached story and square buyer card
// PNGs for handle. Invalid handles and missing files are no-ops returning nil.
// Handles are user-controlled and persisted without charset validation, so
// deletion goes through an os.Root: even a handle that somehow produced a
// hostile file name could not resolve outside ShareCardCacheDir.
func InvalidateBuyerCardCache(handle string) error {
	// Resolve every target name before touching the filesystem so an invalid
	// handle guarantees zero deletions.
	names := make([]string, 0, 2)
	for _, format := range []string{ShareCardFormatStory, ShareCardFormatSquare} {
		name, ok := buyerCardCacheFileName(handle, format)
		if !ok {
			return nil
		}
		names = append(names, name)
	}

	root, err := os.OpenRoot(ShareCardCacheDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // no cache dir yet — nothing to invalidate
		}
		return fmt.Errorf("open share card cache root: %w", err)
	}
	defer root.Close()

	var lastErr error
	for _, name := range names {
		if err := root.Remove(name); err != nil && !os.IsNotExist(err) {
			lastErr = err
		}
	}
	if lastErr != nil {
		return fmt.Errorf("invalidate buyer card cache: %w", lastErr)
	}
	return nil
}
