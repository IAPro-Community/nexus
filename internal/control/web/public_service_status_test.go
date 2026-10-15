package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestPublicStatusCacheNoAuthAndIndependentFailure(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if r.Header.Get("Authorization") != "" || r.Header.Get("Cookie") != "" {
			t.Error("credential leaked to public service")
		}
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/bad") {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte(`{"status":{"indicator":"minor","description":"A component is degraded"}}`))
	}))
	defer srv.Close()
	now := time.Date(2026, 10, 14, 18, 0, 0, 0, time.UTC)
	cache := &publicStatusCache{
		client: srv.Client(), clock: func() time.Time { return now },
		sources: []publicStatusSource{
			{"codex", srv.URL + "/ok"}, {"claude", srv.URL + "/bad"},
		},
	}
	snap := cache.snapshot(context.Background())
	if len(snap) != 2 || snap[0].Indicator != "minor" || snap[0].Stale ||
		snap[1].Indicator != "UNKNOWN" || !snap[1].Stale {
		t.Fatalf("unexpected independent service states: %+v", snap)
	}
	_ = cache.snapshot(context.Background())
	if hits.Load() != 2 { t.Fatalf("cache should avoid repeat requests, got %d", hits.Load()) }
	now = now.Add(6 * time.Minute)
	snap = cache.snapshot(context.Background())
	if len(snap) != 2 || hits.Load() != 4 { t.Fatalf("cache did not refresh: %+v, %d", snap, hits) }
}

func TestPublicStatusRejectsUnknownIndicator(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"status":{"indicator":"unlimited"}}`))
	}))
	defer srv.Close()
	c := &publicStatusCache{
		client: srv.Client(), clock: time.Now,
		sources: []publicStatusSource{{"codex", srv.URL}},
	}
	got := c.snapshot(context.Background())
	if len(got) != 1 || got[0].Indicator != "UNKNOWN" || !got[0].Stale {
		t.Fatalf("unrecognized status must be unknown: %+v", got)
	}
}
