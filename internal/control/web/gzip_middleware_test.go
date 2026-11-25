package web

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWithGzipCompressesTextAndSkipsBinaryAndSSE(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/bundle.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")
		_, _ = io.WriteString(w, strings.Repeat("console.log('nexus');", 40))
	})
	mux.HandleFunc("/icon.png", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte{0x89, 'P', 'N', 'G'})
	})
	mux.HandleFunc("/api/v1/events", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: ok\n\n")
	})

	handler := withGzip(mux)

	reqJS := httptest.NewRequest(http.MethodGet, "/bundle.js", nil)
	reqJS.Header.Set("Accept-Encoding", "gzip, deflate")
	recJS := httptest.NewRecorder()
	handler.ServeHTTP(recJS, reqJS)
	if got := recJS.Header().Get("Content-Encoding"); got != "gzip" {
		t.Fatalf("expected Content-Encoding=gzip for JS, got %q", got)
	}
	if !strings.Contains(recJS.Header().Get("Vary"), "Accept-Encoding") {
		t.Fatalf("expected Vary: Accept-Encoding, got %q", recJS.Header().Get("Vary"))
	}
	zr, err := gzip.NewReader(recJS.Body)
	if err != nil {
		t.Fatalf("invalid gzip body: %v", err)
	}
	plain, err := io.ReadAll(zr)
	_ = zr.Close()
	if err != nil || !strings.Contains(string(plain), "console.log('nexus')") {
		t.Fatalf("unexpected decoded JS body: err=%v len=%d", err, len(plain))
	}

	reqPNG := httptest.NewRequest(http.MethodGet, "/icon.png", nil)
	reqPNG.Header.Set("Accept-Encoding", "gzip")
	recPNG := httptest.NewRecorder()
	handler.ServeHTTP(recPNG, reqPNG)
	if got := recPNG.Header().Get("Content-Encoding"); got != "" {
		t.Fatalf("PNG must not be gzipped, got %q", got)
	}

	reqSSE := httptest.NewRequest(http.MethodGet, "/api/v1/events", nil)
	reqSSE.Header.Set("Accept-Encoding", "gzip")
	recSSE := httptest.NewRecorder()
	handler.ServeHTTP(recSSE, reqSSE)
	if got := recSSE.Header().Get("Content-Encoding"); got != "" {
		t.Fatalf("SSE must not be gzipped, got %q", got)
	}
}
