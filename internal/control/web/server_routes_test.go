package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRegisterRoutesIncludesPublicHealthRoute(t *testing.T) {
	mux := http.NewServeMux()
	server := &Server{}
	registerRoutes(mux, routeDependencies{
		server:       server,
		api:          NewAPIHandler(nil),
		nexusHandler: NewNexusHandler(nil),
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)

	if res.Code == http.StatusNotFound {
		t.Fatalf("health route was not registered: status=%d", res.Code)
	}
}

func TestSystemInfoExposesVersionAndCapabilities(t *testing.T) {
	h := NewAPIHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/system/info", nil)
	res := httptest.NewRecorder()
	h.handleSystemInfo(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("system info status=%d want=%d", res.Code, http.StatusOK)
	}
	var body struct {
		APIVersion    string                    `json:"apiVersion"`
		ServerVersion string                    `json:"serverVersion"`
		Capabilities  map[string]map[string]any `json:"capabilities"`
		Features      map[string]bool           `json:"features"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode system info: %v", err)
	}
	if body.APIVersion != "v1" || body.ServerVersion == "" {
		t.Fatalf("metadata missing version contract: %+v", body)
	}
	if len(body.Capabilities) == 0 {
		t.Fatal("metadata did not expose registered provider capabilities")
	}
	if body.Features == nil {
		t.Fatal("metadata did not expose feature flags")
	}
	if body.Features["review_center"] {
		t.Fatal("review center must remain disabled by default")
	}
}

func TestProjectSubrouteClassification(t *testing.T) {
	cases := map[string]string{
		"/api/v1/projects/proj-1/layout":       "layout",
		"/api/v1/projects/proj-1/agents":       "agents",
		"/api/v1/projects/proj-1/plans":        "plans",
		"/api/v1/projects/proj-1/open-os":      "open-os",
		"/api/v1/projects/proj-1/git/branches": "git-branches",
		"/api/v1/projects/proj-1/git/changes":  "git-changes",
		"/api/v1/projects/proj-1/git/diff":     "git-diff",
		"/api/v1/projects/proj-1/git/checkout": "git-checkout",
		"/api/v1/projects/proj-1":              "detail",
	}
	for path, want := range cases {
		if got := projectSubroute(path); got != want {
			t.Fatalf("path=%s got=%s want=%s", path, got, want)
		}
	}
}
