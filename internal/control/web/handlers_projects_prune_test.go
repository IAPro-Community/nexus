package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func TestHandleProjectPruneCandidatesListsOnlyRowsWithoutEvidence(t *testing.T) {
	n, handler, implicit := openTestWebProjectHarness(t)
	st, openErr := n.OpenProject()
	if openErr != nil {
		t.Fatal(openErr)
	}
	opened, err := st.CreateProject(store.Project{Name: "Opened", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if err := st.TouchProject(opened.ID); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	handler.handleProjectPruneCandidates(rec, httptest.NewRequest(http.MethodGet, "/api/v1/projects/prune-candidates", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var body struct {
		Candidates []struct {
			ID     string `json:"id"`
			Reason string `json:"reason"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Candidates) != 2 {
		t.Fatalf("expected both projects inspected, got %d", len(body.Candidates))
	}
	reasons := map[string]string{}
	for _, candidate := range body.Candidates {
		reasons[candidate.ID] = candidate.Reason
	}
	if reasons[implicit.ID] == "" {
		t.Fatal("implicit project must be offered as a candidate")
	}
	if reasons[opened.ID] != "" {
		t.Fatalf("opened project must keep its evidence, got %q", reasons[opened.ID])
	}
}

func TestHandleProjectPruneCandidatesRejectsWrites(t *testing.T) {
	_, handler, _ := openTestWebProjectHarness(t)
	rec := httptest.NewRecorder()
	handler.handleProjectPruneCandidates(rec, httptest.NewRequest(http.MethodPost, "/api/v1/projects/prune-candidates", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rec.Code)
	}
}
