package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

func TestSPAAssetHandlerReturnsNotFoundForMissingChunks(t *testing.T) {
	handler := newSPAAssetHandler(http.FS(fstest.MapFS{
		"index.html":               &fstest.MapFile{Data: []byte("<html>nexus app</html>")},
		"bundle.js":                &fstest.MapFile{Data: []byte("console.log('app');")},
		"chunks/chunk-ABC123.js":   &fstest.MapFile{Data: []byte("export const x = 1;")},
		"assets/BrowserSurface.js": &fstest.MapFile{Data: []byte("export {}; ")},
	}))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/assets/BrowserSurface-old.js", nil))
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("missing chunk status = %d, want 404", recorder.Code)
	}
	if recorder.Body.String() == "<html>nexus app</html>" {
		t.Fatal("missing chunk returned the SPA document")
	}

	missingSplitChunk := httptest.NewRecorder()
	handler.ServeHTTP(missingSplitChunk, httptest.NewRequest(http.MethodGet, "/chunks/does-not-exist.js", nil))
	if missingSplitChunk.Code != http.StatusNotFound {
		t.Fatalf("missing split chunk status = %d, want 404", missingSplitChunk.Code)
	}

	chunkRec := httptest.NewRecorder()
	handler.ServeHTTP(chunkRec, httptest.NewRequest(http.MethodGet, "/chunks/chunk-ABC123.js", nil))
	if chunkRec.Code != http.StatusOK || chunkRec.Header().Get("Cache-Control") != "public, max-age=31536000, immutable" {
		t.Fatalf("chunk cache header = %q status=%d", chunkRec.Header().Get("Cache-Control"), chunkRec.Code)
	}

	entryRec := httptest.NewRecorder()
	handler.ServeHTTP(entryRec, httptest.NewRequest(http.MethodGet, "/bundle.js", nil))
	if entryRec.Code != http.StatusOK || entryRec.Header().Get("Cache-Control") != "no-cache, no-store, must-revalidate" {
		t.Fatalf("bundle.js cache header = %q status=%d", entryRec.Header().Get("Cache-Control"), entryRec.Code)
	}

	spa := httptest.NewRecorder()
	handler.ServeHTTP(spa, httptest.NewRequest(http.MethodGet, "/p/project-1/workspace", nil))
	if spa.Code != http.StatusOK || spa.Body.String() != "<html>nexus app</html>" {
		t.Fatalf("SPA route response = %d %q, want 200 with index.html", spa.Code, spa.Body.String())
	}
	if spa.Header().Get("Cache-Control") != "no-cache, no-store, must-revalidate" {
		t.Fatalf("SPA index cache header = %q", spa.Header().Get("Cache-Control"))
	}
}
