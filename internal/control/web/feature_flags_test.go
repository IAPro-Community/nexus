package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOptionalFeatureFlagsFailClosed(t *testing.T) {
	t.Setenv("NEXUS_FEATURES_ENABLED", "")
	t.Setenv("NEXUS_FEATURE_FLOW", "true")
	if optionalFeatureEnabled(featureFlow) {
		t.Fatal("individual opt-in must not bypass global default false")
	}
	t.Setenv("NEXUS_FEATURES_ENABLED", "true")
	t.Setenv("NEXUS_FEATURE_FLOW", "")
	if optionalFeatureEnabled(featureFlow) {
		t.Fatal("global opt-in alone must not enable features")
	}
	t.Setenv("NEXUS_FEATURE_FLOW", "true")
	if !optionalFeatureEnabled(featureFlow) {
		t.Fatal("global plus individual opt-in must enable feature")
	}
	t.Setenv("NEXUS_FEATURE_FLOW", "1")
	if optionalFeatureEnabled(featureFlow) {
		t.Fatal("only explicit true is accepted")
	}
}

func TestOptionalMutationGatePreservesHistoryAndCancellation(t *testing.T) {
	t.Setenv("NEXUS_FEATURES_ENABLED", "")
	t.Setenv("NEXUS_FEATURE_MISSIONS", "")
	reached := 0
	handler := requireOptionalMutation(featureMissions, func(w http.ResponseWriter, r *http.Request) {
		reached++
		w.WriteHeader(http.StatusNoContent)
	})
	check := func(method, path string, want int) {
		t.Helper()
		rec := httptest.NewRecorder()
		handler(rec, httptest.NewRequest(method, path, nil))
		if rec.Code != want {
			t.Errorf("%s %s: got %d, want %d", method, path, rec.Code, want)
		}
	}
	check(http.MethodGet, "/api/v1/runs/legacy", http.StatusNoContent)
	check(http.MethodPost, "/api/v1/runs/legacy/resume", http.StatusNotFound)
	check(http.MethodPost, "/api/v1/runs/legacy/resume/cancel", http.StatusNotFound)
	check(http.MethodPost, "/api/v1/runs/legacy/cancel", http.StatusNoContent)
	check(http.MethodDelete, "/api/v1/runs/legacy", http.StatusNotFound)
	if reached != 2 {
		t.Fatalf("got %d allowed handlers, want 2", reached)
	}
}

func TestProjectOptionalRoutesAreScoped(t *testing.T) {
	paths := []string{
		"/api/v1/projects/p/agents",
		"/api/v1/projects/p/shell",
		"/api/v1/projects/p/maestro/compose-preview",
		"/api/v1/projects/p/plans",
		"/api/v1/projects/p/maestro/runs",
	}
	want := []string{"", "", featureComposer, featureFlow, featureMissions}
	for i, path := range paths {
		if got := featureForProjectMutation(path); got != want[i] {
			t.Errorf("%s: got %q, want %q", path, got, want[i])
		}
	}
}
