package web

import (
	"net/http"
	"strings"
	"testing"
)

func TestComparisonRoutesRequireProjectScopeAndCSRF(t *testing.T) {
	client, srv, csrf := csrfClient(t)
	get, err := client.Get(srv.URL() + "/api/v1/comparisons")
	if err != nil {
		t.Fatal(err)
	}
	_ = get.Body.Close()
	if get.StatusCode != http.StatusBadRequest {
		t.Fatalf("list without project_id status = %d, want %d", get.StatusCode, http.StatusBadRequest)
	}

	request, err := http.NewRequest(http.MethodPost, srv.URL()+"/api/v1/comparisons", strings.NewReader(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	withoutCSRF, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	_ = withoutCSRF.Body.Close()
	if withoutCSRF.StatusCode != http.StatusForbidden {
		t.Fatalf("create without CSRF status = %d, want %d", withoutCSRF.StatusCode, http.StatusForbidden)
	}

	request, err = http.NewRequest(http.MethodPost, srv.URL()+"/api/v1/comparisons", strings.NewReader(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("X-CSRF-Token", csrf)
	withCSRF, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	_ = withCSRF.Body.Close()
	if withCSRF.StatusCode != http.StatusBadRequest {
		t.Fatalf("create with invalid body status = %d, want %d", withCSRF.StatusCode, http.StatusBadRequest)
	}

	campaignList, err := client.Get(srv.URL() + "/api/v1/comparison-campaigns")
	if err != nil {
		t.Fatal(err)
	}
	_ = campaignList.Body.Close()
	if campaignList.StatusCode != http.StatusBadRequest {
		t.Fatalf("campaign list without project_id status = %d, want %d", campaignList.StatusCode, http.StatusBadRequest)
	}

	request, err = http.NewRequest(http.MethodPost, srv.URL()+"/api/v1/comparison-campaigns", strings.NewReader(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	withoutCampaignCSRF, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	_ = withoutCampaignCSRF.Body.Close()
	if withoutCampaignCSRF.StatusCode != http.StatusForbidden {
		t.Fatalf("campaign create without CSRF status = %d, want %d", withoutCampaignCSRF.StatusCode, http.StatusForbidden)
	}
}
