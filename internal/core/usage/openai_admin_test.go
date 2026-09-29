package usage

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/model"
)

func TestCollectOpenAIAdminUsageKeepsAPISeparateFromSubscription(t *testing.T) {
	scope := model.AccountScope{ProviderID: "codex", ProfileID: "work", AccountID: "chatgpt-1", IdentityVersion: "1"}
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.URL.Path+"?"+r.URL.RawQuery)
		if got := r.Header.Get("Authorization"); got != "Bearer admin-test" {
			t.Fatalf("authorization=%q", got)
		}
		switch r.URL.Path {
		case "/organization/usage/completions":
			if got := r.URL.Query()["project_ids"]; len(got) != 1 || got[0] != "proj_1" {
				t.Fatalf("project_ids=%v", got)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":[{"results":[{"input_tokens":1000,"input_cached_tokens":400,"output_tokens":500,"num_model_requests":5}]}],"has_more":false}`))
		case "/organization/costs":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":[{"results":[{"amount":{"currency":"usd","value":12.5}}]}],"has_more":false}`))
		case "/organization/projects/proj_1/spend_limit":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"threshold_amount":10000,"currency":"usd","interval":"month"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	prevBase, prevClient, prevNow := openAIAdminBaseURL, openAIAdminHTTPClient, openAIAdminNow
	openAIAdminBaseURL = server.URL
	openAIAdminHTTPClient = server.Client()
	openAIAdminNow = func() time.Time { return time.Date(2026, 9, 29, 18, 0, 0, 0, time.UTC) }
	t.Cleanup(func() {
		openAIAdminBaseURL, openAIAdminHTTPClient, openAIAdminNow = prevBase, prevClient, prevNow
	})

	state, err := CollectOpenAIAdminUsage(context.Background(), "admin-test", OpenAIAdminFilter{ProjectIDs: []string{"proj_1"}}, scope)
	if err != nil {
		t.Fatal(err)
	}
	if state.Channel != model.ResourceChannelAPI || state.Status != model.UsageLive {
		t.Fatalf("state=%+v", state)
	}
	if state.Quota != nil {
		t.Fatal("API consumption/budget must never become Codex subscription quota")
	}
	if state.Consumption == nil || state.Consumption.InputTokens == nil || *state.Consumption.InputTokens != 1000 ||
		state.Consumption.CachedTokens == nil || *state.Consumption.CachedTokens != 400 ||
		state.Consumption.OutputTokens == nil || *state.Consumption.OutputTokens != 500 ||
		state.Consumption.Requests == nil || *state.Consumption.Requests != 5 {
		t.Fatalf("consumption=%+v", state.Consumption)
	}
	if state.Cost == nil || state.Cost.Amount == nil || *state.Cost.Amount != 12.5 || state.Cost.Currency != "USD" {
		t.Fatalf("cost=%+v", state.Cost)
	}
	if state.Cost.Budget == nil || *state.Cost.Budget != 100 || state.Cost.BudgetRemaining == nil || *state.Cost.BudgetRemaining != 87.5 {
		t.Fatalf("budget=%+v", state.Cost)
	}
	if state.AccountScope.Key() != scope.Key() {
		t.Fatalf("scope=%+v", state.AccountScope)
	}
	if len(calls) != 3 {
		t.Fatalf("calls=%v", calls)
	}
}

func TestCollectOpenAIAdminUsageRequiresExplicitAttribution(t *testing.T) {
	_, err := CollectOpenAIAdminUsage(context.Background(), "admin-test", OpenAIAdminFilter{}, model.AccountScope{})
	if err == nil || !strings.Contains(err.Error(), "requires project_id") {
		t.Fatalf("err=%v", err)
	}
}
