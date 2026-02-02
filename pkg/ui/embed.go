package ui

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed dist
var dist embed.FS

// GetFileSystem returns the embedded filesystem, rooted at "dist"
func GetFileSystem() (http.FileSystem, error) {
	fsys, err := fs.Sub(dist, "dist")
	if err != nil {
		return nil, err
	}
	return http.FS(fsys), nil
}

// SPAHandler returns a file server that falls back to index.html for unknown paths
// ensuring that React Router works correctly on refresh.
func SPAHandler() http.Handler {
	fsys, _ := GetFileSystem()
	fileServer := http.FileServer(fsys)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// If the path starts with /api, let it pass (though Gin should catch this before)
		if strings.HasPrefix(r.URL.Path, "/api") {
			http.NotFound(w, r)
			return
		}

		// Check if file exists in the embedded FS
		// We need to trim the leading slash for fs.ValidPath/Open logic usually,
		// but http.FS handles some of this. Let's try to open it.
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}

		// Open the file from the embedded FS (wrapped in http.FS logic)
		f, err := fsys.Open(path)
		if err != nil {
			// If file not found, serve index.html (SPA Fallback)
			r.URL.Path = "/"
		} else {
			f.Close()
		}
		
		fileServer.ServeHTTP(w, r)
	})
}
