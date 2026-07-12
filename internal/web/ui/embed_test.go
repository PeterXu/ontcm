package ui

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestEmbeddedFilesPresent guards against a broken embed directive or a
// renamed asset: every file the SPA needs at runtime must exist in the FS.
func TestEmbeddedFilesPresent(t *testing.T) {
	for _, name := range []string{
		"index.html",
		"css/style.css",
		"js/app.js",
		"js/api.js",
		"js/diagnostic.js",
		"js/lookup.js",
	} {
		if _, err := fs.Stat(rootFS, name); err != nil {
			t.Errorf("embedded static/%s missing: %v", name, err)
		}
	}
}

// TestIndexHandlerServesHTML verifies the SPA entry point is served as HTML
// at the root with the expected title.
func TestIndexHandlerServesHTML(t *testing.T) {
	rec := httptest.NewRecorder()
	IndexHandler().ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))

	if rec.Code != 200 {
		t.Fatalf("want status 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("want text/html content-type, got %q", ct)
	}
	if !strings.Contains(rec.Body.String(), "ONTCM") {
		t.Errorf("index body missing ONTCM title marker: %q", rec.Body.String())
	}
}

// TestStaticServing checks assets resolve with a correct MIME type (browsers
// reject ES modules served as text/plain) and that missing files 404.
func TestStaticServing(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle("/static/", StaticHandler())

	cases := []struct {
		name     string
		path     string
		wantCode int
		wantCT   string // substring the Content-Type must contain; "" = don't check
	}{
		{"js module", "/static/js/app.js", 200, "javascript"},
		{"stylesheet", "/static/css/style.css", 200, "text/css"},
		{"missing file", "/static/does-not-exist.js", 404, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, httptest.NewRequest("GET", tc.path, nil))
			if rec.Code != tc.wantCode {
				t.Fatalf("want code %d, got %d (body: %q)", tc.wantCode, rec.Code, rec.Body.String())
			}
			if tc.wantCT != "" {
				if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, tc.wantCT) {
					t.Errorf("want content-type containing %q, got %q", tc.wantCT, ct)
				}
			}
		})
	}
}
