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
        t.Fatal("explicit global plus individual opt-in must enable feature")
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
    cases := []struct{method,path string; want int}{
        {http.MethodGet, "/api/v1/runs/legacy", http.StatusNoContent},
        {http.MethodPost, "/api/v1/runs/legacy/resume", http.StatusNotFound},
        {http.MethodPost, "/api/v1/runs/legacy/cancel", http.StatusNoContent},
        {http.MethodDelete, "/api/v1/runs/legacy", http.StatusNotFound},
    }
    for _, tt := range cases {
        rec := httptest.NewRecorder()
        handler(rec, httptest.NewRequest(tt.method, tt.path, nil))
        if rec.Code != tt.want { t.Errorf("%s %s: got %d, want %d",tt.method,tt.path,rec.Code,tt.want) }
    }
    if reached != 2 {t.Fatalf("got %d allowed handlers, want 2", reached)}
}

func TestProjectOptionalRoutesAreScoped(t *testing.T) {
    cases := map[string]string{
        "/api/v1/projects/p/agents": "",
        "/api/v1/projects/p/shell": "",
        "/api/v1/projects/p/maestro/compose-preview":featureComposer,
        "/api/v1/projects/p/plans": featureFlow,
        "/api/v1/projects/p/maestro/runs": featureMissions,
    }
    for path,want:=range cases {
        if got:=featureForProjectMutation(path);got!=want{t.Errorf("%s: got %q, want %q",path,got,want)}
    }
}
