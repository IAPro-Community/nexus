package store

import (
	"fmt"
)

// CostAnalyticsTotals is a read-only aggregation of the canonical observation
// ledger. Cost evidence classes are deliberately NOT merged into a fictional
// bill: provider-reported values and API-equivalent estimates are separate.
type CostAnalyticsTotals struct {
	ObservationCount         int64 `json:"observation_count"`
	KnownCostObservations    int64 `json:"known_cost_observations"`
	UnknownCostObservations  int64 `json:"unknown_cost_observations"`
	KnownTokenObservations   int64 `json:"known_token_observations"`
	ReportedCostObservations  int64 `json:"reported_cost_observations"`
	EquivalentCostObservations int64 `json:"equivalent_cost_observations"`
	InferredCostObservations  int64 `json:"inferred_cost_observations"`
	InputTokens              int64 `json:"input_tokens"`
	CachedInputTokens        int64 `json:"cached_input_tokens"`
	OutputTokens             int64 `json:"output_tokens"`
	ReasoningTokens          int64 `json:"reasoning_tokens"`
	DurationMS               int64 `json:"duration_ms"`
	ProviderReportedMicroUSD int64 `json:"provider_reported_microusd"`
	APIEquivalentMicroUSD    int64 `json:"api_equivalent_microusd"`
	InferredMicroUSD         int64 `json:"inferred_microusd"`
	UnclassifiedMicroUSD     int64 `json:"unclassified_microusd"`
}

type DailyCostAnalytics struct {
	Date string `json:"date"`
	CostAnalyticsTotals
}

type ModelCostAnalytics struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
	CostAnalyticsTotals
}

type CostAnalyticsView struct {
	Total            CostAnalyticsTotals  `json:"total"`
	ByDay            []DailyCostAnalytics `json:"by_day"`
	ByModel          []ModelCostAnalytics `json:"by_model"`
	ModelsTruncated  bool                 `json:"models_truncated"`
}

// All aggregates use the same filter and cost evidence semantics. A NULL
// cost/token remains unknown: COALESCE is only used *after* the known/unknown
// observation counts are recorded.
const costAnalyticsColumns = `
	COUNT(*),
	COALESCE(SUM(CASE WHEN cost_microusd IS NOT NULL THEN 1 ELSE 0 END),0),
	COALESCE(SUM(CASE WHEN cost_microusd IS NULL THEN 1 ELSE 0 END),0),
	COALESCE(SUM(CASE WHEN input_tokens IS NOT NULL OR cached_input_tokens IS NOT NULL OR output_tokens IS NOT NULL OR reasoning_tokens IS NOT NULL THEN 1 ELSE 0 END),0),
	COALESCE(SUM(CASE WHEN cost_microusd IS NOT NULL AND cost_kind='PROVIDER_REPORTED' THEN 1 ELSE 0 END),0),
	COALESCE(SUM(CASE WHEN cost_microusd IS NOT NULL AND cost_kind='API_EQUIVALENT_ESTIMATE' THEN 1 ELSE 0 END),0),
	COALESCE(SUM(CASE WHEN cost_microusd IS NOT NULL AND cost_kind='INFERRED' THEN 1 ELSE 0 END),0),
	COALESCE(SUM(input_tokens),0),
	COALESCE(SUM(cached_input_tokens),0),
	COALESCE(SUM(output_tokens),0),
	COALESCE(SUM(reasoning_tokens),0),
	COALESCE(SUM(duration_ms),0),
	COALESCE(SUM(CASE WHEN cost_kind='PROVIDER_REPORTED' THEN cost_microusd END),0),
	COALESCE(SUM(CASE WHEN cost_kind='API_EQUIVALENT_ESTIMATE' THEN cost_microusd END),0),
	COALESCE(SUM(CASE WHEN cost_kind='INFERRED' THEN cost_microusd END),0),
	COALESCE(SUM(CASE WHEN cost_kind NOT IN ('PROVIDER_REPORTED','API_EQUIVALENT_ESTIMATE','INFERRED') THEN cost_microusd END),0)`

type analyticsScanner interface {
	Scan(dest ...any) error
}

func scanCostAnalyticsTotals(scanner analyticsScanner, dest *CostAnalyticsTotals) error {
	return scanner.Scan(
		&dest.ObservationCount, &dest.KnownCostObservations, &dest.UnknownCostObservations,
		&dest.KnownTokenObservations, &dest.ReportedCostObservations,
		&dest.EquivalentCostObservations, &dest.InferredCostObservations,
		&dest.InputTokens, &dest.CachedInputTokens,
		&dest.OutputTokens, &dest.ReasoningTokens, &dest.DurationMS,
		&dest.ProviderReportedMicroUSD, &dest.APIEquivalentMicroUSD,
		&dest.InferredMicroUSD, &dest.UnclassifiedMicroUSD,
	)
}

// SummarizeCostAnalytics queries already-persisted observations. Daily buckets
// are bounded by the requested 1..365 day window, and the model list has a
// hard cap plus an explicit truncation bit. It never scans provider homes or
// reads session transcripts, credentials, prompts, or tool output.
func (s *Store) SummarizeCostAnalytics(filter CostObservationFilter, maxModels int) (CostAnalyticsView, error) {
	if maxModels < 1 || maxModels > 50 {
		return CostAnalyticsView{}, fmt.Errorf("maxModels must be between 1 and 50")
	}
	where, args := costFilterClause(filter)
	view := CostAnalyticsView{
		ByDay:   make([]DailyCostAnalytics, 0),
		ByModel: make([]ModelCostAnalytics, 0),
	}
	if err := scanCostAnalyticsTotals(s.db.QueryRow(
		"SELECT "+costAnalyticsColumns+" FROM cost_observations"+where, args...,
	), &view.Total); err != nil {
		return CostAnalyticsView{}, fmt.Errorf("summarize cost analytics total: %w", err)
	}
	daily, err := s.db.Query(
		"SELECT substr(observed_at,1,10),"+costAnalyticsColumns+
			" FROM cost_observations"+where+
			" GROUP BY substr(observed_at,1,10) ORDER BY substr(observed_at,1,10) ASC",
		args...,
	)
	if err != nil {
		return CostAnalyticsView{}, fmt.Errorf("summarize cost analytics daily: %w", err)
	}
	for daily.Next() {
		var row DailyCostAnalytics
		if err := daily.Scan(
			&row.Date,
			&row.ObservationCount, &row.KnownCostObservations, &row.UnknownCostObservations,
			&row.KnownTokenObservations, &row.ReportedCostObservations,
			&row.EquivalentCostObservations, &row.InferredCostObservations,
			&row.InputTokens, &row.CachedInputTokens,
			&row.OutputTokens, &row.ReasoningTokens, &row.DurationMS,
			&row.ProviderReportedMicroUSD, &row.APIEquivalentMicroUSD,
			&row.InferredMicroUSD, &row.UnclassifiedMicroUSD,
		); err != nil {
			daily.Close()
			return CostAnalyticsView{}, fmt.Errorf("scan daily cost analytics: %w", err)
		}
		view.ByDay = append(view.ByDay, row)
	}
	if err := daily.Err(); err != nil {
		daily.Close()
		return CostAnalyticsView{}, fmt.Errorf("iterate daily cost analytics: %w", err)
	}
	daily.Close()

	models, err := s.db.Query(
		"SELECT provider,model,"+costAnalyticsColumns+
			" FROM cost_observations"+where+
			" GROUP BY provider,model ORDER BY COUNT(*) DESC,provider,model LIMIT ?",
		append(append([]any(nil), args...), maxModels+1)...,
	)
	if err != nil {
		return CostAnalyticsView{}, fmt.Errorf("summarize cost analytics by model: %w", err)
	}
	for models.Next() {
		var row ModelCostAnalytics
		if err := models.Scan(
			&row.Provider, &row.Model,
			&row.ObservationCount, &row.KnownCostObservations, &row.UnknownCostObservations,
			&row.KnownTokenObservations, &row.ReportedCostObservations,
			&row.EquivalentCostObservations, &row.InferredCostObservations,
			&row.InputTokens, &row.CachedInputTokens,
			&row.OutputTokens, &row.ReasoningTokens, &row.DurationMS,
			&row.ProviderReportedMicroUSD, &row.APIEquivalentMicroUSD,
			&row.InferredMicroUSD, &row.UnclassifiedMicroUSD,
		); err != nil {
			models.Close()
			return CostAnalyticsView{}, fmt.Errorf("scan model cost analytics: %w", err)
		}
		if len(view.ByModel) == maxModels {
			view.ModelsTruncated = true
			break
		}
		view.ByModel = append(view.ByModel, row)
	}
	if err := models.Err(); err != nil {
		models.Close()
		return CostAnalyticsView{}, fmt.Errorf("iterate model cost analytics: %w", err)
	}
	models.Close()
	return view, nil
}
