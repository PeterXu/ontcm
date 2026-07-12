// Package ui embeds the browser single-page app (HTML/CSS/JS) and exposes it
// as HTTP handlers. Embedding keeps the server self-contained — the binary
// carries its own UI with no filesystem path dependency, which matters because
// ONTCM is a local/offline diagnostic tool.
package ui

import (
	"embed"
	"io/fs"
	"net/http"
)

// staticFS holds the embedded static/ tree. The directive runs at compile time;
// `all:` also picks up files whose names begin with "_" or ".".
//
//go:embed all:static
var staticFS embed.FS

// rootFS is static/ re-rooted at its own contents, so a request for "/js/app.js"
// resolves to the file "js/app.js" rather than "static/js/app.js".
var rootFS fs.FS

func init() {
	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		// fs.Sub only fails if "static" is missing from the embed — a compile-
		// time invariant the embed directive guarantees, so this is unreachable.
		panic("ui: embedded static/ missing: " + err.Error())
	}
	rootFS = sub
}

// FS returns the embedded static files as a root-level fs.FS. Callers that want
// gin-style serving can wrap it with http.FS and pass it to router.StaticFS.
func FS() fs.FS {
	return rootFS
}

// StaticHandler serves the embedded files at the "/static/" prefix. It strips
// that prefix before delegating to http.FileServer, so "/static/js/app.js"
// resolves to the embedded "js/app.js". MIME types are set by the stdlib
// (e.g. "text/javascript; charset=utf-8" for .js, which is valid for ES modules).
func StaticHandler() http.Handler {
	return http.StripPrefix("/static/", http.FileServer(http.FS(rootFS)))
}

// IndexHandler serves the SPA entry point (index.html) for the root path.
func IndexHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, err := fs.ReadFile(rootFS, "index.html")
		if err != nil {
			http.Error(w, "index.html not found in embedded UI", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(data)
	})
}
