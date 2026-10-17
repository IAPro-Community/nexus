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
	if hits.Load() != 2 {
		t.Fatalf("cache should avoid repeat requests, got %d", hits.Load())
	}
	now = now.Add(6 * time.Minute)
	snap = cache.snapshot(context.Background())
	if len(snap) != 2 || hits.Load() != 4 {
		t.Fatalf("cache did not refresh: %+v, %d", snap, hits.Load())
	}
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

func TestPublicStatusCanceledRequestDoesNotPoisonCache(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		_, _ = w.Write([]byte(`{"status":{"indicator":"none","description":"All systems operational"}}`))
	}))
	defer srv.Close()
	clock := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	cache := &publicStatusCache{
		client:  srv.Client(),
		clock:   func() time.Time { return clock },
		sources: []publicStatusSource{{"openai", srv.URL}},
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	aborted := cache.snapshot(ctx)
	if len(aborted) != 1 || aborted[0].Indicator != "UNKNOWN" || !aborted[0].Stale {
		t.Fatalf("canceled request should be unknown, got %+v", aborted)
	}
	if hits.Load() != 0 {
		t.Fatalf("canceled request should not probe public services: %d", hits.Load())
	}
	// A new real request must not inherit the canceled request's UNKNOWN TTL.
	recovered := cache.snapshot(context.Background())
	if len(recovered) != 1 || recovered[0].Indicator != "none" || recovered[0].Stale {
		t.Fatalf("healthy follow-up poisoned by canceled request: %+v", recovered)
	}
	if hits.Load() != 1 {
		t.Fatalf("expected one actual probe, got %d", hits.Load())
	}
}

func TestPublicStatusMidFlightCancellationAllowsNextProbe(t *testing.T) {
	started := make(chan struct{})
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			close(started)
			<-r.Context().Done()
			return
		}
		_, _ = w.Write([]byte(`{"status":{"indicator":"none"}}`))
	}))
	defer srv.Close()
	now := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	cache := &publicStatusCache{
		client:  srv.Client(),
		clock:   func() time.Time { return now },
		sources: []publicStatusSource{{"openai", srv.URL}},
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result := make(chan []PublicServiceStatus, 1)
	go func() { result <- cache.snapshot(ctx) }()
	select {
	case <-started:
		cancel()
	case <-time.After(3 * time.Second):
		t.Fatal("first probe did not start")
	}
	select {
	case aborted := <-result:
		if len(aborted) != 1 || !aborted[0].Stale {
			t.Fatalf("aborted probe should be marked stale: %+v", aborted)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("canceled request did not complete")
	}
	next := cache.snapshot(context.Background())
	if len(next) != 1 || next[0].Indicator != "none" || next[0].Stale || calls.Load() != 2 {
		t.Fatalf("canceled request poisoned following probe: %+v (calls=%d)", next, calls.Load())
	}
}
