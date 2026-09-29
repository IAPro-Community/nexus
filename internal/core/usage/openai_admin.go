package usage

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/model"
)

const openAIAdminUsageTimeout = 10 * time.Second

var (
	openAIAdminBaseURL    = "https://api.openai.com/v1"
	openAIAdminHTTPClient = &http.Client{Timeout: openAIAdminUsageTimeout}
	openAIAdminNow        = time.Now
)

// OpenAIAdminFilter explicitly attributes organization API usage to a managed
// Nexus profile. At least one filter is required so organization-wide usage is
// never silently assigned to one Codex subscription account.
type OpenAIAdminFilter struct {
	ProjectIDs []string
	UserIDs    []string
	APIKeyIDs  []string
}

func (f OpenAIAdminFilter) Valid() bool {
	return len(nonEmpty(f.ProjectIDs)) > 0 || len(nonEmpty(f.UserIDs)) > 0 || len(nonEmpty(f.APIKeyIDs)) > 0
}

// CollectOpenAIAdminUsage reads month-to-date API consumption and attributable
// cost using an OpenAI Admin API key. The returned API channel is independent
// from the profile's SUBSCRIPTION quota.
func CollectOpenAIAdminUsage(ctx context.Context, adminKey string, filter OpenAIAdminFilter, scope model.AccountScope) (model.ResourceChannelState, error) {
	state := model.ResourceChannelState{
		Channel:      model.ResourceChannelAPI,
		Status:       model.UsageUnknown,
		Source:       model.SourceNone,
		Confidence:   model.UsageConfidenceUnknown,
		AccountScope: scope,
	}
	adminKey = strings.TrimSpace(adminKey)
	if adminKey == "" {
		return state, fmt.Errorf("OpenAI Admin API key is not configured")
	}
	if !filter.Valid() {
		return state, fmt.Errorf("OpenAI API usage binding requires project_id, user_id or api_key_id")
	}
	now := openAIAdminNow().UTC()
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	consumption, err := fetchOpenAICompletionsUsage(ctx, adminKey, filter, start, now)
	if err != nil {
		return state, err
	}
	state.Status = model.UsageLive
	state.Source = model.SourceOfficialAPI
	state.Confidence = model.UsageConfidenceHigh
	state.ObservedAt = now
	state.Consumption = &consumption

	// Costs can be safely attributed through project filters. The costs API
	// does not provide the same user-level attribution contract as completions,
	// so do not broaden a user/api-key binding into organization-wide cost.
	projects := nonEmpty(filter.ProjectIDs)
	if len(projects) > 0 {
		if cost, costErr := fetchOpenAICosts(ctx, adminKey, projects, start, now); costErr == nil {
			state.Cost = &cost
			if len(projects) == 1 {
				if budget, budgetErr := fetchOpenAIProjectSpendLimit(ctx, adminKey, projects[0]); budgetErr == nil && budget > 0 {
					state.Cost.Budget = &budget
					remaining := budget
					if state.Cost.Amount != nil {
						remaining -= *state.Cost.Amount
					}
					if remaining < 0 {
						remaining = 0
					}
					state.Cost.BudgetRemaining = &remaining
				}
			}
		}
	}
	return state, nil
}

type openAIUsagePage struct {
	Data []struct {
		Results []struct {
			InputTokens       float64 `json:"input_tokens"`
			InputCachedTokens float64 `json:"input_cached_tokens"`
			OutputTokens      float64 `json:"output_tokens"`
			NumModelRequests  float64 `json:"num_model_requests"`
		} `json:"results"`
	} `json:"data"`
	HasMore  bool   `json:"has_more"`
	NextPage string `json:"next_page"`
}

type openAICostPage struct {
	Data []struct {
		Results []struct {
			Amount struct {
				Currency string  `json:"currency"`
				Value    float64 `json:"value"`
			} `json:"amount"`
		} `json:"results"`
	} `json:"data"`
	HasMore  bool   `json:"has_more"`
	NextPage string `json:"next_page"`
}

func fetchOpenAICompletionsUsage(ctx context.Context, adminKey string, filter OpenAIAdminFilter, start, end time.Time) (model.ConsumptionState, error) {
	var inputTokens, cachedTokens, outputTokens, requests float64
	page := ""
	for pageCount := 0; pageCount < 32; pageCount++ {
		values := baseOpenAIUsageQuery(start, end)
		for _, value := range nonEmpty(filter.ProjectIDs) {
			values.Add("project_ids", value)
		}
		for _, value := range nonEmpty(filter.UserIDs) {
			values.Add("user_ids", value)
		}
		for _, value := range nonEmpty(filter.APIKeyIDs) {
			values.Add("api_key_ids", value)
		}
		if page != "" {
			values.Set("page", page)
		}
		var response openAIUsagePage
		if err := getOpenAIAdminJSON(ctx, adminKey, "/organization/usage/completions", values, &response); err != nil {
			return model.ConsumptionState{}, err
		}
		for _, bucket := range response.Data {
			for _, result := range bucket.Results {
				inputTokens += result.InputTokens
				cachedTokens += result.InputCachedTokens
				outputTokens += result.OutputTokens
				requests += result.NumModelRequests
			}
		}
		if !response.HasMore || strings.TrimSpace(response.NextPage) == "" {
			return model.ConsumptionState{
				InputTokens:  &inputTokens,
				CachedTokens: &cachedTokens,
				OutputTokens: &outputTokens,
				Requests:     &requests,
			}, nil
		}
		page = response.NextPage
	}
	return model.ConsumptionState{}, fmt.Errorf("OpenAI usage pagination exceeded safety bound")
}

func fetchOpenAICosts(ctx context.Context, adminKey string, projectIDs []string, start, end time.Time) (model.CostState, error) {
	var amount float64
	currency := ""
	page := ""
	for pageCount := 0; pageCount < 32; pageCount++ {
		values := baseOpenAIUsageQuery(start, end)
		for _, projectID := range nonEmpty(projectIDs) {
			values.Add("project_ids", projectID)
		}
		if page != "" {
			values.Set("page", page)
		}
		var response openAICostPage
		if err := getOpenAIAdminJSON(ctx, adminKey, "/organization/costs", values, &response); err != nil {
			return model.CostState{}, err
		}
		for _, bucket := range response.Data {
			for _, result := range bucket.Results {
				amount += result.Amount.Value
				observedCurrency := strings.ToUpper(strings.TrimSpace(result.Amount.Currency))
				if currency == "" {
					currency = observedCurrency
				} else if observedCurrency != "" && currency != observedCurrency {
					return model.CostState{}, fmt.Errorf("OpenAI costs returned mixed currencies")
				}
			}
		}
		if !response.HasMore || strings.TrimSpace(response.NextPage) == "" {
			return model.CostState{Amount: &amount, Currency: currency}, nil
		}
		page = response.NextPage
	}
	return model.CostState{}, fmt.Errorf("OpenAI cost pagination exceeded safety bound")
}

func fetchOpenAIProjectSpendLimit(ctx context.Context, adminKey, projectID string) (float64, error) {
	var response struct {
		ThresholdAmount float64 `json:"threshold_amount"`
		Currency        string  `json:"currency"`
		Interval        string  `json:"interval"`
	}
	path := "/organization/projects/" + url.PathEscape(strings.TrimSpace(projectID)) + "/spend_limit"
	if err := getOpenAIAdminJSON(ctx, adminKey, path, nil, &response); err != nil {
		return 0, err
	}
	if response.ThresholdAmount <= 0 {
		return 0, nil
	}
	// Spend limit threshold_amount is expressed in cents by the Admin API.
	return response.ThresholdAmount / 100.0, nil
}

func baseOpenAIUsageQuery(start, end time.Time) url.Values {
	values := url.Values{}
	values.Set("start_time", strconv.FormatInt(start.Unix(), 10))
	values.Set("end_time", strconv.FormatInt(end.Unix(), 10))
	values.Set("bucket_width", "1d")
	values.Set("limit", "31")
	return values
}

func getOpenAIAdminJSON(ctx context.Context, adminKey, path string, query url.Values, destination any) error {
	endpoint := strings.TrimRight(openAIAdminBaseURL, "/") + path
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Bearer "+adminKey)
	request.Header.Set("Accept", "application/json")
	response, err := openAIAdminHTTPClient.Do(request)
	if err != nil {
		return fmt.Errorf("OpenAI Admin API: %w", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 4<<20))
	if err != nil {
		return err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("OpenAI Admin API returned HTTP %d", response.StatusCode)
	}
	if err := json.Unmarshal(body, destination); err != nil {
		return fmt.Errorf("decode OpenAI Admin API response: %w", err)
	}
	return nil
}

func nonEmpty(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
