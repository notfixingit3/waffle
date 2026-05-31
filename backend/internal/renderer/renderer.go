// Package renderer provides a multitemplate HTML renderer using Gin's
// html/template with layout composition via block/define directives.
//
// Usage:
//
//	import "github.com/syrup/backend/internal/renderer"
//
//	r := renderer.New(nil)
//	r.AddFromFiles("templates/layouts/base.html", "templates/partials/*.html")
//
//	// In a Gin handler:
//	r.Render(c, "page.html", gin.H{"title": "My Page", "data": myData})
//
// Template functions available:
//   - dict: create a map from key-value pairs
//   - seq: generate a slice of integers (inclusive)
//   - formatDate: format a time.Time or *time.Time
package renderer

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// Renderer wraps a Go html/template with layout and partial support.
type Renderer struct {
	templates *template.Template
	funcMap   template.FuncMap
}

// New creates a new Renderer with the given additional FuncMap.
// Built-in functions: dict, seq, formatDate
func New(funcs template.FuncMap) *Renderer {
	fm := template.FuncMap{
		"dict":       dict,
		"seq":        seq,
		"iterate":    seq,
		"formatDate": formatDate,
		"deref":      deref,
		"add":        add,
		"sub":        sub,
		"mul":        mul,
		"div":        div,
		"json":       toJSON,
	}

	for k, v := range funcs {
		fm[k] = v
	}

	return &Renderer{
		templates: template.New("").Funcs(fm),
		funcMap:   fm,
	}
}

// AddFromFiles parses template files using glob patterns.
func (r *Renderer) AddFromFiles(files ...string) error {
	for _, pattern := range files {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return fmt.Errorf("renderer: invalid glob pattern %q: %w", pattern, err)
		}
		if len(matches) == 0 {
			return fmt.Errorf("renderer: no files match pattern %q", pattern)
		}
		for _, match := range matches {
			if _, err := r.templates.ParseFiles(match); err != nil {
				return fmt.Errorf("renderer: failed to parse %q: %w", match, err)
			}
		}
	}
	return nil
}

// AddFromFS parses templates from an fs.FS using glob patterns.
func (r *Renderer) AddFromFS(fsys fs.FS, patterns ...string) error {
	for _, pattern := range patterns {
		matches, err := fs.Glob(fsys, pattern)
		if err != nil {
			return fmt.Errorf("renderer: invalid glob pattern %q: %w", pattern, err)
		}
		if len(matches) == 0 {
			return fmt.Errorf("renderer: no files match pattern %q in embedded FS", pattern)
		}
		t := r.templates
		for _, match := range matches {
			data, err := fs.ReadFile(fsys, match)
			if err != nil {
				return fmt.Errorf("renderer: failed to read %q: %w", match, err)
			}
			_, err = t.New(match).Parse(string(data))
			if err != nil {
				return fmt.Errorf("renderer: failed to parse %q: %w", match, err)
			}
		}
	}
	return nil
}

// Render executes the named template with data and writes HTML to the Gin context.
func (r *Renderer) Render(c *gin.Context, name string, data gin.H) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := r.templates.ExecuteTemplate(c.Writer, name, data); err != nil {
		_ = c.AbortWithError(http.StatusInternalServerError, err)
	}
}

// RenderToWriter executes the named template and writes to the Gin response writer.
func (r *Renderer) RenderToWriter(c *gin.Context, name string, data gin.H) error {
	return r.templates.ExecuteTemplate(c.Writer, name, data)
}

// RenderString executes the named template and returns the result as a string.
func (r *Renderer) RenderString(name string, data gin.H) (string, error) {
	var buf strings.Builder
	if err := r.templates.ExecuteTemplate(&buf, name, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// Write renders the named template to an io.Writer.
func (r *Renderer) Write(w io.Writer, name string, data gin.H) error {
	return r.templates.ExecuteTemplate(w, name, data)
}

// Clone returns a deep copy of the Renderer that can be independently modified.
func (r *Renderer) Clone() (*Renderer, error) {
	clone, err := r.templates.Clone()
	if err != nil {
		return nil, fmt.Errorf("renderer: clone: %w", err)
	}
	return &Renderer{
		templates: clone,
		funcMap:   r.funcMap,
	}, nil
}

// HTTPHandler returns an http.HandlerFunc that renders the named template.
func (r *Renderer) HTTPHandler(name string, data gin.H) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := r.templates.ExecuteTemplate(w, name, data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

// ---------------------------------------------------------------------------
// Built-in template functions
// ---------------------------------------------------------------------------

func dict(values ...interface{}) (map[string]interface{}, error) {
	if len(values)%2 != 0 {
		return nil, fmt.Errorf("dict: odd number of arguments (got %d)", len(values))
	}
	m := make(map[string]interface{}, len(values)/2)
	for i := 0; i < len(values); i += 2 {
		key, ok := values[i].(string)
		if !ok {
			return nil, fmt.Errorf("dict: key at index %d is not a string (got %T)", i, values[i])
		}
		m[key] = values[i+1]
	}
	return m, nil
}

func seq(start, end int) []int {
	if start > end {
		return nil
	}
	n := end - start + 1
	s := make([]int, n)
	for i := 0; i < n; i++ {
		s[i] = start + i
	}
	return s
}

func formatDate(layout string, v interface{}) string {
	switch t := v.(type) {
	case time.Time:
		return t.Format(layout)
	case *time.Time:
		if t == nil {
			return ""
		}
		return t.Format(layout)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func deref(v interface{}) interface{} {
	switch t := v.(type) {
	case *string:
		if t == nil {
			return ""
		}
		return *t
	case *int:
		if t == nil {
			return 0
		}
		return *t
	case *time.Time:
		if t == nil {
			return ""
		}
		return *t
	default:
		return v
	}
}

func add(a, b int) int {
	return a + b
}

func sub(a, b int) int {
	return a - b
}

func mul(a, b int) int {
	return a * b
}

func div(a, b int) int {
	if b == 0 {
		return 0
	}
	return a / b
}

func toJSON(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}

// NewFromFiles creates a Renderer, parses the given template files, and returns it.
func NewFromFiles(funcs template.FuncMap, files ...string) (*Renderer, error) {
	r := New(funcs)
	if err := r.AddFromFiles(files...); err != nil {
		return nil, err
	}
	return r, nil
}
