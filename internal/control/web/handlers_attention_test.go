package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/IAPro-Community/nexus/internal/nexus/runner"
)

func TestAttentionFilterFromRequest(t *testing.T) {
	request := httptest.NewRequest("GET", "/api/v1/attention?project_id=%20project-a%20&mission_id=mission-b", nil)
	filter := attentionFilterFromRequest(request)
	if filter.ProjectID != "project-a" || filter.MissionID != "mission-b" {
		t.Fatalf("unexpected attention filter: %+v", filter)
	}
}

func TestAttentionEndpointRequiresAuthenticationAndReturnsFilteredProjection(t *testing.T) {
	client, server, _ := csrfClient(t)
	unauthenticated := &http.Client{}
	request, err := http.NewRequest(http.MethodGet, server.URL()+"/api/v1/attention", nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := unauthenticated.Do(request)
	if err != nil {
		t.Fatalf("unauthenticated attention request: %v", err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthenticated attention request must return 401, got %d", response.StatusCode)
	}

	filteredRequest, err := http.NewRequest(
		http.MethodGet,
		server.URL()+"/api/v1/attention?project_id=missing-project&mission_id=missing-mission",
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	filteredResponse, err := client.Do(filteredRequest)
	if err != nil {
		t.Fatalf("authenticated attention request: %v", err)
	}
	defer filteredResponse.Body.Close()
	if filteredResponse.StatusCode != http.StatusOK {
		t.Fatalf("authenticated attention request must return 200, got %d", filteredResponse.StatusCode)
	}
	var group runner.AttentionGroup
	if err := json.NewDecoder(filteredResponse.Body).Decode(&group); err != nil {
		t.Fatalf("decode filtered attention group: %v", err)
	}
	if len(group.AllItems) != 0 || group.TotalNeeds != 0 {
		t.Fatalf("non-matching project/mission filters returned attention: %+v", group)
	}
}
