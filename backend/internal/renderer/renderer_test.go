package renderer

import (
	"html/template"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestNew(t *testing.T) {
	r := New(nil)
	if r == nil {
		t.Fatal("expected non-nil renderer")
	}
}

func TestAddFromFiles(t *testing.T) {
	r := New(nil)
	err := r.AddFromFiles(
		"../../templates/layouts/base.html",
		"../../templates/partials/header.html",
		"../../templates/partials/footer.html",
	)
	if err != nil {
		t.Fatalf("AddFromFiles failed: %v", err)
	}

	var buf strings.Builder
	err = r.Write(&buf, "base.html", gin.H{
		"Title":   "Test Page",
		"Version": "test",
		"DevMode": false,
	})
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Project Syrup") {
		t.Errorf("expected 'Project Syrup' in output")
	}
	if !strings.Contains(output, "The Waffle Maker") {
		t.Errorf("expected 'The Waffle Maker' in output")
	}
	if !strings.Contains(output, "</html>") {
		t.Errorf("expected closing html tag")
	}
}

func TestDictFunction(t *testing.T) {
	m, err := dict("key1", "val1", "key2", 42)
	if err != nil {
		t.Fatalf("dict failed: %v", err)
	}
	if m["key1"] != "val1" {
		t.Errorf("expected key1=val1, got %v", m["key1"])
	}
	if m["key2"] != 42 {
		t.Errorf("expected key2=42, got %v", m["key2"])
	}
}

func TestDictOddArgs(t *testing.T) {
	_, err := dict("a", "b", "c")
	if err == nil {
		t.Error("expected error for odd number of arguments")
	}
}

func TestDictNonStringKey(t *testing.T) {
	_, err := dict(1, "val")
	if err == nil {
		t.Error("expected error for non-string key")
	}
}

func TestSeqFunction(t *testing.T) {
	s := seq(1, 5)
	if len(s) != 5 {
		t.Fatalf("expected 5 elements, got %d", len(s))
	}
	expected := []int{1, 2, 3, 4, 5}
	for i, v := range s {
		if v != expected[i] {
			t.Fatalf("at index %d: expected %d, got %d", i, expected[i], v)
		}
	}

	empty := seq(5, 3)
	if empty != nil {
		t.Errorf("expected nil for start > end")
	}
}

func TestFormatDateFunction(t *testing.T) {
	now := time.Date(2025, 6, 15, 14, 30, 0, 0, time.UTC)
	result := formatDate("Jan 2, 2006", now)
	if result != "Jun 15, 2025" {
		t.Errorf("expected 'Jun 15, 2025', got %q", result)
	}

	var nilTime *time.Time
	result = formatDate("Jan 2, 2006", nilTime)
	if result != "" {
		t.Errorf("expected empty string for nil time, got %q", result)
	}

	result = formatDate("3:04 PM", &now)
	if result != "2:30 PM" {
		t.Errorf("expected '2:30 PM', got %q", result)
	}
}

func TestRenderString(t *testing.T) {
	r := New(nil)
	err := r.AddFromFiles(
		"../../templates/layouts/base.html",
		"../../templates/partials/header.html",
		"../../templates/partials/footer.html",
	)
	if err != nil {
		t.Fatalf("AddFromFiles failed: %v", err)
	}

	output, err := r.RenderString("base.html", gin.H{
		"Version": "test",
		"DevMode": false,
	})
	if err != nil {
		t.Fatalf("RenderString failed: %v", err)
	}
	if !strings.Contains(output, "<!DOCTYPE html>") {
		t.Errorf("expected DOCTYPE in output")
	}
}

func TestCustomFuncMap(t *testing.T) {
	fm := template.FuncMap{
		"customUpper": func(s string) string { return strings.ToUpper(s) },
	}
	r := New(fm)
	if r.funcMap["customUpper"] == nil {
		t.Error("expected customUpper function to be registered")
	}
}

func TestAddFromFilesErrors(t *testing.T) {
	r := New(nil)
	err := r.AddFromFiles("nonexistent/*.html")
	if err == nil {
		t.Error("expected error for nonexistent pattern")
	}
}

func TestWrite(t *testing.T) {
	r := New(nil)
	err := r.AddFromFiles(
		"../../templates/layouts/base.html",
		"../../templates/partials/header.html",
		"../../templates/partials/footer.html",
	)
	if err != nil {
		t.Fatalf("AddFromFiles failed: %v", err)
	}

	var buf strings.Builder
	err = r.Write(&buf, "base.html", gin.H{
		"Version": "test",
		"DevMode": false,
	})
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if !strings.Contains(buf.String(), "<!DOCTYPE html>") {
		t.Errorf("expected DOCTYPE in output")
	}
}


