package services

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInvalidateShareCardCache_RemovesBothFormats(t *testing.T) {
	dir := t.TempDir()
	originalDir := ShareCardCacheDir
	ShareCardCacheDir = dir
	defer func() { ShareCardCacheDir = originalDir }()

	slug := "test-waffle-abc123"

	// Create both cache files
	storyPath := filepath.Join(dir, slug+"-story.png")
	squarePath := filepath.Join(dir, slug+"-square.png")
	if err := os.WriteFile(storyPath, []byte("fake-png"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(squarePath, []byte("fake-png"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Verify they exist
	if _, err := os.Stat(storyPath); os.IsNotExist(err) {
		t.Fatal("expected story file to exist before invalidation")
	}
	if _, err := os.Stat(squarePath); os.IsNotExist(err) {
		t.Fatal("expected square file to exist before invalidation")
	}

	// Invalidate
	if err := InvalidateShareCardCache(slug); err != nil {
		t.Fatalf("InvalidateShareCardCache returned error: %v", err)
	}

	// Verify both are gone
	if _, err := os.Stat(storyPath); !os.IsNotExist(err) {
		t.Fatal("expected story file to be removed after invalidation")
	}
	if _, err := os.Stat(squarePath); !os.IsNotExist(err) {
		t.Fatal("expected square file to be removed after invalidation")
	}
}

func TestInvalidateShareCardCache_MissingFilesAreNoop(t *testing.T) {
	dir := t.TempDir()
	originalDir := ShareCardCacheDir
	ShareCardCacheDir = dir
	defer func() { ShareCardCacheDir = originalDir }()

	// No files exist — should not error
	if err := InvalidateShareCardCache("nonexistent-slug"); err != nil {
		t.Fatalf("expected no error for missing files, got: %v", err)
	}
}

func TestInvalidateShareCardCache_OnlyRemovesTargetSlug(t *testing.T) {
	dir := t.TempDir()
	originalDir := ShareCardCacheDir
	ShareCardCacheDir = dir
	defer func() { ShareCardCacheDir = originalDir }()

	// Create files for two different slugs
	slugA := "waffle-a"
	slugB := "waffle-b"

	for _, slug := range []string{slugA, slugB} {
		for _, fmt := range []string{"story", "square"} {
			path := filepath.Join(dir, slug+"-"+fmt+".png")
			if err := os.WriteFile(path, []byte("fake-png"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}

	// Invalidate only slugA
	if err := InvalidateShareCardCache(slugA); err != nil {
		t.Fatalf("InvalidateShareCardCache returned error: %v", err)
	}

	// slugA files should be gone
	for _, fmt := range []string{"story", "square"} {
		path := filepath.Join(dir, slugA+"-"+fmt+".png")
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("expected %s file for slugA to be removed", fmt)
		}
	}

	// slugB files should still exist
	for _, fmt := range []string{"story", "square"} {
		path := filepath.Join(dir, slugB+"-"+fmt+".png")
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Fatalf("expected %s file for slugB to remain", fmt)
		}
	}
}
