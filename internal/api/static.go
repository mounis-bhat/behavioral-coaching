package api

import (
	"io/fs"
	"net/http"

	"github.com/mounis-bhat/starter/assets"
	"github.com/mounis-bhat/starter/internal/config"
)

func staticHandler(cfg *config.Config) http.Handler {
	if cfg.Env == "development" {
		// In development, proxy to SvelteKit dev server or serve nothing
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "In development mode, use SvelteKit dev server", http.StatusNotFound)
		})
	}

	// In production, serve embedded static files
	staticFS, err := fs.Sub(assets.StaticFiles, "static")
	if err != nil {
		panic(err)
	}

	// Read index.html for SPA fallback
	indexHTML, err := fs.ReadFile(staticFS, "index.html")
	if err != nil {
		panic(err)
	}

	fileServer := http.FileServer(http.FS(staticFS))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Serve static files if they exist
		if path != "/" {
			// Check if file exists
			if f, err := staticFS.Open(path[1:]); err == nil {
				f.Close()
				fileServer.ServeHTTP(w, r)
				return
			}
		}

		// SPA fallback: serve index.html for all other routes
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(indexHTML)
	})
}
