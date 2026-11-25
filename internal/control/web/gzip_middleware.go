package web

import (
	"bufio"
	"compress/gzip"
	"errors"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
)

var gzipWriterPool = sync.Pool{
	New: func() any {
		return gzip.NewWriter(io.Discard)
	},
}

// withGzip compresses eligible text/JSON/bundle HTTP responses with gzip when
// the client advertises gzip support. WebSocket upgrades, SSE streams, byte
// ranges, HEAD requests, and already-compressed binary assets pass through untouched.
func withGzip(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		addVaryAcceptEncoding(w.Header())
		if !requestAcceptsGzip(r) {
			next.ServeHTTP(w, r)
			return
		}
		gw := &gzipResponseWriter{ResponseWriter: w, status: http.StatusOK}
		defer gw.close()
		next.ServeHTTP(gw, r)
	})
}

func requestAcceptsGzip(r *http.Request) bool {
	if r.Method == http.MethodHead {
		return false
	}
	if strings.TrimSpace(r.Header.Get("Upgrade")) != "" {
		return false
	}
	if strings.TrimSpace(r.Header.Get("Range")) != "" {
		return false
	}
	if !strings.Contains(strings.ToLower(r.Header.Get("Accept-Encoding")), "gzip") {
		return false
	}
	switch strings.ToLower(filepath.Ext(r.URL.Path)) {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".ico", ".woff", ".woff2", ".gz", ".br", ".zip":
		return false
	}
	return true
}

func addVaryAcceptEncoding(h http.Header) {
	for _, existing := range h.Values("Vary") {
		for _, part := range strings.Split(existing, ",") {
			if strings.EqualFold(strings.TrimSpace(part), "Accept-Encoding") {
				return
			}
		}
	}
	h.Add("Vary", "Accept-Encoding")
}

func statusAllowsBody(status int) bool {
	return status >= 200 && status != http.StatusNoContent && status != http.StatusNotModified
}

type gzipResponseWriter struct {
	http.ResponseWriter
	gz          *gzip.Writer
	status      int
	wroteHeader bool
	bypass      bool
}

func (g *gzipResponseWriter) WriteHeader(code int) {
	if g.wroteHeader {
		return
	}
	g.status = code
	g.wroteHeader = true
	if !statusAllowsBody(code) || g.shouldBypass() {
		g.bypass = true
		g.ResponseWriter.WriteHeader(code)
		return
	}
	g.Header().Del("Content-Length")
	g.Header().Set("Content-Encoding", "gzip")
	g.ResponseWriter.WriteHeader(code)
}

func (g *gzipResponseWriter) Write(p []byte) (int, error) {
	if !g.wroteHeader {
		if g.Header().Get("Content-Type") == "" && len(p) > 0 {
			g.Header().Set("Content-Type", http.DetectContentType(p))
		}
		g.WriteHeader(http.StatusOK)
	}
	if g.bypass {
		return g.ResponseWriter.Write(p)
	}
	if g.gz == nil {
		w := gzipWriterPool.Get().(*gzip.Writer)
		w.Reset(g.ResponseWriter)
		g.gz = w
	}
	return g.gz.Write(p)
}

func (g *gzipResponseWriter) shouldBypass() bool {
	ct := strings.ToLower(g.Header().Get("Content-Type"))
	if strings.HasPrefix(ct, "text/event-stream") {
		return true
	}
	return g.Header().Get("Content-Encoding") != ""
}

func (g *gzipResponseWriter) Flush() {
	if g.gz != nil {
		_ = g.gz.Flush()
	}
	if f, ok := g.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (g *gzipResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	g.bypass = true
	h, ok := g.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("hijacker not supported")
	}
	return h.Hijack()
}

func (g *gzipResponseWriter) Unwrap() http.ResponseWriter {
	return g.ResponseWriter
}

func (g *gzipResponseWriter) close() {
	if g.bypass || !g.wroteHeader || !statusAllowsBody(g.status) {
		return
	}
	if g.gz == nil {
		// Ensure Content-Encoding: gzip advertised at WriteHeader still has a valid gzip trailer.
		w := gzipWriterPool.Get().(*gzip.Writer)
		w.Reset(g.ResponseWriter)
		g.gz = w
	}
	_ = g.gz.Close()
	g.gz.Reset(io.Discard)
	gzipWriterPool.Put(g.gz)
	g.gz = nil
}
