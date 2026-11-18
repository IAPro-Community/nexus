package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// The usage import endpoint is an observational receipt import: it takes no
// parameters, stays behind routeProject authentication + CSRF, and maps
// Bridge unavailability to a generic 503 without leaking diagnostics.
func TestMaestroUsageImportRouteContract(t *testing.T) {
	client, srv, csrf := csrfClient(t)
	base := srv.URL()

	projDir := t.TempDir()
	body, _ := json.Marshal(map[string]string{"name": "Usage import", "path": projDir})
	req, _ := http.NewRequest(http.MethodPost, base+"/api/v1/projects", bytes.NewReader(body))
	req.Header.Set("X-CSRF-Token", csrf)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	var proj struct {
		ID string `json:"id"`
	}
	if resp.StatusCode != http.StatusCreated {
		resp.Body.Close()
		t.Fatalf("expected 201 for project create, got %d", resp.StatusCode)
	}
	_ = json.NewDecoder(resp.Body).Decode(&proj)
	resp.Body.Close()
	if proj.ID == "" {
		t.Fatal("missing project id")
	}
	path := base + "/api/v1/projects/" + proj.ID + "/maestro/submissions/import-usage"

	// Mutations require the session CSRF token.
	noCSRF, err := client.Post(path, "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	noCSRF.Body.Close()
	if noCSRF.StatusCode != http.StatusForbidden {
		t.Fatalf("import-usage without CSRF: expected 403, got %d", noCSRF.StatusCode)
	}

	// The endpoint is POST-only even though the path is not a known project
	// detail suffix: a GET must not fall through to handleProjectDetail.
	getResp, err := client.Get(path)
	if err != nil {
		t.Fatal(err)
	}
	getResp.Body.Close()
	if getResp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("GET import-usage: expected 405, got %d", getResp.StatusCode)
	}

	// Parameterless contract: any payload beyond empty/{ } is rejected before
	// the Bridge is consulted.
	for _, raw := range []string{`{"force":true}`, `[1,2,3]`, `{}{}`} {
		req, _ = http.NewRequest(http.MethodPost, path, strings.NewReader(raw))
		req.Header.Set("X-CSRF-Token", csrf)
		resp, err = client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("import-usage with body %s: expected 400, got %d", raw, resp.StatusCode)
		}
	}

	// Valid call: either the Bridge is unavailable (generic 503, no leaks) or
	// the import succeeded with pure counters — never fabricated evidence.
	req, _ = http.NewRequest(http.MethodPost, path, nil)
	req.Header.Set("X-CSRF-Token", csrf)
	resp, err = client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusServiceUnavailable:
		// Canonical path when the Maestro Bridge cannot be reached.
	case http.StatusOK:
		var result struct {
			Imported  int `json:"imported"`
			Skipped   int `json:"skipped"`
			Conflicts int `json:"conflicts"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("invalid import result payload: %v", err)
		}
		if result.Imported < 0 || result.Skipped < 0 || result.Conflicts < 0 {
			t.Fatalf("negative counters are impossible: %+v", result)
		}
	default:
		t.Fatalf("import-usage valid call: unexpected status %d", resp.StatusCode)
	}
}
