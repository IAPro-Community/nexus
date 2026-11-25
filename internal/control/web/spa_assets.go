package web

import (
	"net/http"
	"path"
	"path/filepath"
	"strings"
)

func newSPAAssetHandler(files http.FileSystem) http.Handler {
	fileServer := http.FileServer(files)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		file, err := files.Open(strings.TrimPrefix(r.URL.Path, "/"))
		if err == nil {
			_ = file.Close()
			applyStaticAssetCacheHeaders(w, r.URL.Path)
			fileServer.ServeHTTP(w, r)
			return
		}
		if isStaticAssetPath(r.URL.Path) {
			applyStaticAssetCacheHeaders(w, "/index.html")
			http.NotFound(w, r)
			return
		}

		// Only extensionless routes can be client-side application routes.
		applyStaticAssetCacheHeaders(w, "/index.html")
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})
}

func applyStaticAssetCacheHeaders(w http.ResponseWriter, urlPath string) {
	cleaned := path.Clean("/" + strings.TrimSpace(urlPath))
	if strings.HasPrefix(cleaned, "/chunks/") {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		return
	}
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
}

func isStaticAssetPath(reqPath string) bool {
	return strings.HasPrefix(reqPath, "/assets/") || strings.HasPrefix(reqPath, "/chunks/") || filepath.Ext(reqPath) != ""
}
