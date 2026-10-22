package web

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/nexus"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func putJSON(t *testing.T, client *http.Client, csrf, url string, body []byte) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("new put request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-Token", csrf)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("put: %v", err)
	}
	return resp
}
func TestCapacityEndpoints(t *testing.T) {
	t.Setenv("AI_CLI_STATE_DIR", t.TempDir())
	client, srv, csrf := csrfClient(t)
	t.Cleanup(func() { nexus.Default().StopCapacityMonitor() })

	postJSON := func(url string, body []byte) *http.Response {
		t.Helper()
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			t.Fatalf("new post request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-CSRF-Token", csrf)
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("post: %v", err)
		}
		return resp
	}

	t.Run("GET /api/v1/capacity/status fresh projects empty not healthy", func(t *testing.T) {
		resp, err := client.Get(srv.URL() + "/api/v1/capacity/status")
		if err != nil {
			t.Fatalf("get status: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		var status map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
			t.Fatalf("decode status: %v", err)
		}
		if status["enabled"] != true {
			t.Errorf("expected enabled default true, got %v", status["enabled"])
		}
		if res, ok := status["resources"].([]any); !ok || len(res) != 0 {
			t.Errorf("expected empty resources (unknown, never healthy), got %v", status["resources"])
		}
	})

	t.Run("GET /api/v1/capacity/history fresh is empty", func(t *testing.T) {
		resp, err := client.Get(srv.URL() + "/api/v1/capacity/history?limit=10")
		if err != nil {
			t.Fatalf("get history: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		var events []any
		if err := json.NewDecoder(resp.Body).Decode(&events); err != nil {
			t.Fatalf("decode history: %v", err)
		}
		if len(events) != 0 {
			t.Errorf("expected 0 events, got %d", len(events))
		}
	})

	t.Run("GET /api/v1/capacity/pace.csv exports only safe headings", func(t *testing.T) {
		resp, err := client.Get(srv.URL() + "/api/v1/capacity/pace.csv")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		if ct := resp.Header.Get("Content-Type"); ct != "text/csv; charset=utf-8" {
			t.Fatalf("incorrect CSV content type: %s", ct)
		}
		if resp.Header.Get("Cache-Control") != "private, no-store" {
			t.Fatal("pace export must disable shared/browser caches")
		}
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.HasPrefix(data, []byte("provider,profile,quota_group,window,")) {
			t.Fatalf("unexpected CSV header: %q", data)
		}
		if bytes.Contains(data, []byte("account_scope")) {
			t.Fatal("scope identifiers must not appear in exported CSV")
		}
	})

	t.Run("PUT /api/v1/capacity/preferences rejects hysteresis inversion", func(t *testing.T) {
		bad, _ := json.Marshal(store.CapacityPreferences{
			WarningThreshold: 5, CriticalThreshold: 10, ExhaustedThreshold: 0,
			RecoveryThreshold: 25,
		})
		resp := putJSON(t, client, csrf, srv.URL()+"/api/v1/capacity/preferences", bad)
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400 for inverted thresholds, got %d", resp.StatusCode)
		}
	})

	t.Run("PUT /api/v1/capacity/preferences applies lifecycle immediately", func(t *testing.T) {
		prefs := store.DefaultCapacityPreferences()
		prefs.BackgroundEnabled = true
		body, _ := json.Marshal(prefs)
		resp := putJSON(t, client, csrf, srv.URL()+"/api/v1/capacity/preferences", body)
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		var saved store.CapacityPreferences
		if err := json.NewDecoder(resp.Body).Decode(&saved); err != nil {
			resp.Body.Close()
			t.Fatalf("decode prefs: %v", err)
		}
		resp.Body.Close()
		if !saved.BackgroundEnabled {
			t.Error("expected background_enabled persisted true")
		}
		if !nexus.Default().CapacityMonitorRunning() {
			t.Fatal("background opt-in must start capacity monitor without restart")
		}

		saved.BackgroundEnabled = false
		body, _ = json.Marshal(saved)
		resp = putJSON(t, client, csrf, srv.URL()+"/api/v1/capacity/preferences", body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("disable expected 200, got %d", resp.StatusCode)
		}
		if nexus.Default().CapacityMonitorRunning() {
			t.Fatal("disabling background monitoring must stop the hosted loop immediately")
		}
	})

	t.Run("POST /api/v1/capacity/ack requires fingerprint", func(t *testing.T) {
		resp := postJSON(srv.URL()+"/api/v1/capacity/ack", []byte(`{}`))
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.StatusCode)
		}
	})

	t.Run("POST /api/v1/capacity/ack records acknowledgement", func(t *testing.T) {
		resp := postJSON(srv.URL()+"/api/v1/capacity/ack", []byte(`{"fingerprint":"alert-abc"}`))
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		var st store.AlertState
		if err := json.NewDecoder(resp.Body).Decode(&st); err != nil {
			t.Fatalf("decode ack: %v", err)
		}
		if !st.Acked || st.Fingerprint != "alert-abc" {
			t.Errorf("expected acked alert-abc, got %+v", st)
		}
	})

	t.Run("POST /api/v1/capacity/snooze rejects past time", func(t *testing.T) {
		past := time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)
		resp := postJSON(srv.URL()+"/api/v1/capacity/snooze",
			[]byte(`{"fingerprint":"alert-abc","until":"`+past+`"}`))
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400 for past snooze, got %d", resp.StatusCode)
		}
	})

	t.Run("POST /api/v1/capacity/snooze records future snooze", func(t *testing.T) {
		future := time.Now().Add(2 * time.Hour).UTC().Format(time.RFC3339)
		resp := postJSON(srv.URL()+"/api/v1/capacity/snooze",
			[]byte(`{"fingerprint":"alert-abc","until":"`+future+`"}`))
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		var st store.AlertState
		if err := json.NewDecoder(resp.Body).Decode(&st); err != nil {
			t.Fatalf("decode snooze: %v", err)
		}
		if st.SnoozedUntil == nil || st.SnoozedUntil.Before(time.Now()) {
			t.Errorf("expected future snoozed_until, got %+v", st)
		}
	})
}
