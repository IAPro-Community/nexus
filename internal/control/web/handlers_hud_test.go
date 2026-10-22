package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHUDStateFeatureFlagFailsClosed(t *testing.T) {
	t.Setenv("NEXUS_DISABLE_HUD", "1")
	t.Setenv("NEXUS_ENABLE_HUD", "")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/hud", nil)
	rec := httptest.NewRecorder()
	(&NexusHandler{}).handleHUDState(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200", rec.Code)
	}
	var got hudResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Enabled || got.State != nil || got.Rollout != "experimental" {
		t.Fatalf("unexpected disabled HUD envelope: %+v", got)
	}
}

func TestHUDStateDefaultsOnButDoesNotInventProjection(t *testing.T) {
	t.Setenv("NEXUS_DISABLE_HUD", "")
	t.Setenv("NEXUS_ENABLE_HUD", "")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/hud", nil)
	rec := httptest.NewRecorder()
	(&NexusHandler{}).handleHUDState(rec, req)
	// Enabled read-only projection requires a real HUDApplicationService.
	// Missing service must fail closed, not return an invented empty snapshot.
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d, want 503 while projection is unavailable", rec.Code)
	}
}

func TestHUDStateRejectsMutationMethods(t *testing.T) {
	t.Setenv("NEXUS_DISABLE_HUD", "1")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hud", nil)
	rec := httptest.NewRecorder()
	(&NexusHandler{}).handleHUDState(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d, want 405", rec.Code)
	}
}
