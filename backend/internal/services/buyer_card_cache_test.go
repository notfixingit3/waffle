package services

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuyerCardFileName_ValidHandles(t *testing.T) {
	cases := []struct {
		handle string
		format string
		want   string
	}{
		{"a", "story", "buyer-a-story.png"},
		{"a", "square", "buyer-a-square.png"},
		{"dani_boo_glass", "story", "buyer-dani_boo_glass-story.png"},
		{"crysis.designs", "square", "buyer-crysis.designs-square.png"},
		{"collector99", "story", "buyer-collector99-story.png"},
		{"..", "story", "buyer-..-story.png"}, // dots alone stay an ordinary prefixed file name
		{strings.Repeat("x", 30), "story", "buyer-" + strings.Repeat("x", 30) + "-story.png"},
	}
	for _, tc := range cases {
		got, ok := buyerCardCacheFileName(tc.handle, tc.format)
		if !ok {
			t.Errorf("buyerCardCacheFileName(%q, %q) ok = false, want true", tc.handle, tc.format)
			continue
		}
		if got != tc.want {
			t.Errorf("buyerCardCacheFileName(%q, %q) = %q, want %q", tc.handle, tc.format, got, tc.want)
		}
	}
}

func TestBuyerCardFileName_InvalidHandles(t *testing.T) {
	invalid := []string{
		"",                             // empty: regex requires at least 1 char
		"../../etc/x",                  // path traversal
		"../x",                         // path traversal, short form
		"/abs/x",                       // absolute path
		"a/b",                          // embedded separator
		"evil-handle",                  // dash is not in the allowed charset
		"--version",                    // flag-shaped handle must be inert
		"UPPERCASE-ok-after-normalize", // handles arrive lowercased; uppercase must fail
		"Uppercase",                    // uppercase alone fails too
		"with space",                   // whitespace is not allowed
		strings.Repeat("x", 31),        // one over the 30-char limit
	}
	for _, handle := range invalid {
		for _, format := range []string{"story", "square"} {
			if got, ok := buyerCardCacheFileName(handle, format); ok {
				t.Errorf("buyerCardCacheFileName(%q, %q) = %q, true; want ok = false", handle, format, got)
			}
		}
	}
}

func TestInvalidateBuyerCardCache_RemovesOnlyTargetBuyerFiles(t *testing.T) {
	dir := t.TempDir()
	originalDir := ShareCardCacheDir
	ShareCardCacheDir = dir
	defer func() { ShareCardCacheDir = originalDir }()

	targets := []string{
		filepath.Join(dir, "buyer-a-story.png"),
		filepath.Join(dir, "buyer-a-square.png"),
	}
	// Another buyer's card and waffle-style files share the cache dir; the
	// buyer- prefix namespace must keep them untouched.
	survivors := []string{
		filepath.Join(dir, "buyer-b-story.png"),
		filepath.Join(dir, "buyer-b-square.png"),
		filepath.Join(dir, "cool-waffle-abc123-story.png"),
		filepath.Join(dir, "cool-waffle-abc123-square.png"),
	}
	for _, path := range append(targets, survivors...) {
		if err := os.WriteFile(path, []byte("fake-png"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	if err := InvalidateBuyerCardCache("a"); err != nil {
		t.Fatalf("InvalidateBuyerCardCache returned error: %v", err)
	}

	for _, path := range targets {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("expected %s to be removed after invalidation", filepath.Base(path))
		}
	}
	for _, path := range survivors {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected %s to survive invalidation of buyer a", filepath.Base(path))
		}
	}
}

func TestInvalidateBuyerCardCache_MissingFilesAndDirAreNoop(t *testing.T) {
	dir := t.TempDir()
	originalDir := ShareCardCacheDir
	ShareCardCacheDir = dir
	defer func() { ShareCardCacheDir = originalDir }()

	// Cache dir exists but holds no buyer files.
	if err := InvalidateBuyerCardCache("ghost"); err != nil {
		t.Fatalf("expected no error for missing files, got: %v", err)
	}

	// Cache dir does not exist at all.
	ShareCardCacheDir = filepath.Join(dir, "never-created")
	if err := InvalidateBuyerCardCache("ghost"); err != nil {
		t.Fatalf("expected no error for missing cache dir, got: %v", err)
	}
}

func TestInvalidateBuyerCardCache_InvalidHandlesDeleteNothing(t *testing.T) {
	dir := t.TempDir()
	originalDir := ShareCardCacheDir
	ShareCardCacheDir = dir
	defer func() { ShareCardCacheDir = originalDir }()

	decoys := []string{
		"buyer-a-story.png",
		"buyer-a-square.png",
		"cool-waffle-abc123-story.png",
	}
	for _, name := range decoys {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("fake-png"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	invalid := []string{
		"",
		"../../etc/x",
		"../x",
		"/abs/x",
		"a/b",
		"evil-handle",
		"--version",
		"UPPERCASE-ok-after-normalize",
		"with space",
		strings.Repeat("x", 31),
	}
	for _, handle := range invalid {
		if err := InvalidateBuyerCardCache(handle); err != nil {
			t.Errorf("InvalidateBuyerCardCache(%q) returned error: %v; want nil", handle, err)
		}
	}

	for _, name := range decoys {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Errorf("invalid handles deleted decoy %s: %v", name, err)
		}
	}
}

func TestInvalidateBuyerCardCache_TraversalHandleDeletesNothingOutsideCacheDir(t *testing.T) {
	parent := t.TempDir()
	// Cache dir nested two levels down so that a naive
	// filepath.Join(ShareCardCacheDir, handle+"-story.png") resolution of the
	// probe handles lands inside this controlled parent tree.
	cacheDir := filepath.Join(parent, "inner", "cache")
	if err := os.MkdirAll(cacheDir, 0o750); err != nil {
		t.Fatal(err)
	}
	originalDir := ShareCardCacheDir
	ShareCardCacheDir = cacheDir
	defer func() { ShareCardCacheDir = originalDir }()

	// Victims sit exactly where the naive resolution of each probe would land.
	victimOutside := filepath.Join(parent, "tmp", "x-story.png")   // ../../tmp/x
	victimInner := filepath.Join(parent, "inner", "x-story.png")   // ../x
	for _, victim := range []string{victimOutside, victimInner} {
		if err := os.MkdirAll(filepath.Dir(victim), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(victim, []byte("must survive"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	decoy := filepath.Join(cacheDir, "buyer-b-story.png")
	if err := os.WriteFile(decoy, []byte("decoy"), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, handle := range []string{"../../tmp/x", "../x"} {
		if err := InvalidateBuyerCardCache(handle); err != nil {
			t.Fatalf("InvalidateBuyerCardCache(%q) returned error: %v; want nil", handle, err)
		}
	}

	for _, victim := range []string{victimOutside, victimInner} {
		if _, err := os.Stat(victim); err != nil {
			t.Errorf("traversal handle deleted file outside cache dir %s: %v", victim, err)
		}
	}
	if _, err := os.Stat(decoy); err != nil {
		t.Errorf("traversal handle deleted decoy inside cache dir: %v", err)
	}
}
