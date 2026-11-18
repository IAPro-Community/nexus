package store

import (
	"database/sql"
	"fmt"
)

// CostAnalyticsTotals is a read-only aggregation of the canonical observation
// ledger. Cost evidence classes are deliberately NOT merged into a fictional
// bill: provider-reported values and API-equivalent estimates are separate.
//
// Failure observability (F2.5a) is evidence-first as well: InvocationTotal
// mirrors ObservationCount (one invocation per persisted observation), and
// ErrorRate is a derived fraction that stays nil when the scope has no
// observations — a missing signal is never coerced into a fabricated 0.
// Latency/throughput averages are nil unless at least one row actually
// measured that evidence (duration_ms > 0, ttft_ms / output_tokens_per_second
// NOT NULL); NULL evidence means "not measured", never synthetic zero.
type CostAnalyticsTotals struct {
	ObservationCount           int64    `json:"observation_count"`
	InvocationsTotal           int64    `json:"invocations_total"`
	InvocationsFailed          int64    `json:"invocations_failed"`
	ErrorRate                  *float64 `json:"error_rate,omitempty"`
	AvgDurationMS              *float64 `json:"avg_duration_ms,omitempty"`
	AvgTTFTMS                  *float64 `json:"avg_ttft_ms,omitempty"`
	AvgOutputTokensPerSecond   *float64 `json:"avg_output_tokens_per_second,omitempty"`
	KnownCostObservations      int64    `json:"known_cost_observations"`
	UnknownCostObservations    int64    `json:"unknown_cost_observations"`
	KnownTokenObservations     int64    `json:"known_token_observations"`
	ReportedCostObservations   int64    `json:"reported_cost_observations"`
	EquivalentCostObservations int64    `json:"equivalent_cost_observations"`
	InferredCostObservations   int64    `json:"inferred_cost_observations"`
	KnownCacheObservations     int64    `json:"known_cache_observations"`
	CachePromptInputTokens     int64    `json:"cache_prompt_input_tokens"`
	CachePromptReadTokens      int64    `json:"cache_prompt_read_tokens"`
	InputTokens                int64    `json:"input_tokens"`
	CachedInputTokens          int64    `json:"cached_input_tokens"`
	OutputTokens               int64    `json:"output_tokens"`
	ReasoningTokens            int64    `json:"reasoning_tokens"`
	DurationMS                 int64    `json:"duration_ms"`
	ProviderReportedMicroUSD   int64    `json:"provider_reported_microusd"`
	APIEquivalentMicroUSD      int64    `json:"api_equivalent_microusd"`
	InferredMicroUSD           int64    `json:"inferred_microusd"`
	UnclassifiedMicroUSD       int64    `json:"unclassified_microusd"`
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
	Total           CostAnalyticsTotals  `json:"total"`
	ByDay           []DailyCostAnalytics `json:"by_day"`
	ByModel         []ModelCostAnalytics `json:"by_model"`
	ModelsTruncated bool                 `json:"models_truncated"`
}

// costAnalyticsFailurePredicate is the canonical, documented definition of a
// failed invocation on the cost ledger. An observation is a failure when its
// normalized status is neither terminal success ('COMPLETED'), an explicit
// cancellation ('CANCELED'), empty (status was never reported), nor the
// explicit non-evidence marker 'UNKNOWN' — OR when the status carries a
// failure substring (FAIL*, BLOCKED, TIMED_OUT). The substring branches cover
// producer variants such as 'FAILED_TIMEOUT' or 'PROVIDER_BLOCKED'. UNKNOWN is
// deliberately a NON-failure: unclassified evidence must not be turned into a
// failure rate, mirroring the "UNKNOWN ≠ 0" cost-evidence rule. Future
// producers only need their terminal-success statuses added to the NOT IN list.
const costAnalyticsFailurePredicate = `
		UPPER(TRIM(COALESCE(status, ''))) NOT IN ('', 'COMPLETED', 'CANCELED', 'UNKNOWN')
		OR UPPER(TRIM(COALESCE(status, ''))) LIKE '%FAIL%'
		OR UPPER(TRIM(COALESCE(status, ''))) LIKE '%BLOCKED%'
		OR UPPER(TRIM(COALESCE(status, ''))) LIKE '%TIMED_OUT%'`

// All aggregates use the same filter and cost evidence semantics. A NULL
// cost/token remains unknown: COALESCE is only used *after* the known/unknown
// observation counts are recorded. The AVG expressions intentionally keep NULL
// when no row measured the evidence; the Go scan turns that into a nil pointer
// instead of a fabricated zero.
const costAnalyticsColumns = `
	COUNT(*),
	COALESCE(SUM(CASE WHEN cost_microusd IS NOT NULL THEN 1 ELSE 0 END),0),
	COALESCE(SUM(CASE WHEN cost_microusd IS NULL THEN 1 ELSE 0 END),0),
	COALESCE(SUM(CASE WHEN input_tokens IS NOT NULL OR cached_input_tokens IS NOT NULL OR output_tokens IS NOT NULL OR reasoning_tokens IS NOT NULL THEN 1 ELSE 0 END),0),
	COALESCE(SUM(CASE WHEN cost_microusd IS NOT NULL AND cost_kind='PROVIDER_REPORTED' THEN 1 ELSE 0 END),0),
	COALESCE(SUM(CASE WHEN cost_microusd IS NOT NULL AND cost_kind='API_EQUIVALENT_ESTIMATE' THEN 1 ELSE 0 END),0),
	COALESCE(SUM(CASE WHEN cost_microusd IS NOT NULL AND cost_kind='INFERRED' THEN 1 ELSE 0 END),0),
	COALESCE(SUM(CASE WHEN input_tokens IS NOT NULL AND cached_input_tokens IS NOT NULL THEN 1 ELSE 0 END),0),
	COALESCE(SUM(CASE WHEN input_tokens IS NOT NULL AND cached_input_tokens IS NOT NULL THEN input_tokens ELSE 0 END),0),
	COALESCE(SUM(CASE WHEN input_tokens IS NOT NULL AND cached_input_tokens IS NOT NULL THEN cached_input_tokens ELSE 0 END),0),
	COALESCE(SUM(input_tokens),0),
	COALESCE(SUM(cached_input_tokens),0),
	COALESCE(SUM(output_tokens),0),
	COALESCE(SUM(reasoning_tokens),0),
	COALESCE(SUM(duration_ms),0),
	COALESCE(SUM(CASE WHEN cost_kind='PROVIDER_REPORTED' THEN cost_microusd END),0),
	COALESCE(SUM(CASE WHEN cost_kind='API_EQUIVALENT_ESTIMATE' THEN cost_microusd END),0),
	COALESCE(SUM(CASE WHEN cost_kind='INFERRED' THEN cost_microusd END),0),
	COALESCE(SUM(CASE WHEN cost_kind NOT IN ('PROVIDER_REPORTED','API_EQUIVALENT_ESTIMATE','INFERRED') THEN cost_microusd END),0),
	COALESCE(SUM(CASE WHEN ` + costAnalyticsFailurePredicate + ` THEN 1 ELSE 0 END),0),
	AVG(CASE WHEN duration_ms > 0 THEN duration_ms END),
	AVG(CASE WHEN ttft_ms IS NOT NULL THEN ttft_ms END),
	AVG(CASE WHEN output_tokens_per_second IS NOT NULL THEN output_tokens_per_second END)`

type analyticsScanner interface {
	Scan(dest ...any) error
}

// costAnalyticsScanTargets returns the scan destinations that mirror
// costAnalyticsColumns exactly, including NULL-able averages collected in
// temporary holders so missing evidence stays nil on dest.
func costAnalyticsScanTargets(dest *CostAnalyticsTotals) []any {
	targets := []any{
		&dest.ObservationCount, &dest.KnownCostObservations, &dest.UnknownCostObservations,
		&dest.KnownTokenObservations, &dest.ReportedCostObservations,
		&dest.EquivalentCostObservations, &dest.InferredCostObservations,
		&dest.KnownCacheObservations, &dest.CachePromptInputTokens,
		&dest.CachePromptReadTokens,
		&dest.InputTokens, &dest.CachedInputTokens,
		&dest.OutputTokens, &dest.ReasoningTokens, &dest.DurationMS,
		&dest.ProviderReportedMicroUSD, &dest.APIEquivalentMicroUSD,
		&dest.InferredMicroUSD, &dest.UnclassifiedMicroUSD,
	}
	return targets
}

// scanCostAnalyticsTotals scans one row of costAnalyticsColumns and derives
// the invocation/error observability fields. AVG SQL expressions yield NULL
// when no row measured the evidence, which becomes a nil pointer here — never
// a fabricated zero.
func scanCostAnalyticsTotals(scanner analyticsScanner, dest *CostAnalyticsTotals) error {
	var avgDuration, avgTTFT, avgThroughput sql.NullFloat64
	targets := costAnalyticsScanTargets(dest)
	targets = append(targets, &dest.InvocationsFailed, &avgDuration, &avgTTFT, &avgThroughput)
	if err := scanner.Scan(targets...); err != nil {
		return err
	}
	dest.InvocationsTotal = dest.ObservationCount
	if dest.ObservationCount > 0 {
		rate := float64(dest.InvocationsFailed) / float64(dest.ObservationCount)
		dest.ErrorRate = &rate
	}
	if avgDuration.Valid {
		v := avgDuration.Float64
		dest.AvgDurationMS = &v
	}
	if avgTTFT.Valid {
		v := avgTTFT.Float64
		dest.AvgTTFTMS = &v
	}
	if avgThroughput.Valid {
		v := avgThroughput.Float64
		dest.AvgOutputTokensPerSecond = &v
	}
	return nil
}

// scanPrefixedCostAnalyticsTotals scans optional leading columns (date,
// provider, model) followed by the shared costAnalyticsColumns projection.
func scanPrefixedCostAnalyticsTotals(dest *CostAnalyticsTotals, scan func(dest ...any) error, leading ...any) error {
	return scanCostAnalyticsTotals(prefixedScanner{scan: scan, leading: leading}, dest)
}

type prefixedScanner struct {
	scan    func(dest ...any) error
	leading []any
}

func (p prefixedScanner) Scan(dest ...any) error {
	args := make([]any, 0, len(p.leading)+len(dest))
	args = append(args, p.leading...)
	args = append(args, dest...)
	return p.scan(args...)
}

// SummarizeCostEvidence returns all cost evidence classes in one bounded
// SQL aggregation. This is the canonical source for budget assessment: neither
// estimates nor inferred API equivalents may masquerade as provider charges.
func (s *Store) SummarizeCostEvidence(filter CostObservationFilter) (CostAnalyticsTotals, error) {
	where, args := costFilterClause(filter)
	var totals CostAnalyticsTotals
	if err := scanCostAnalyticsTotals(s.db.QueryRow(
		"SELECT "+costAnalyticsColumns+" FROM cost_observations"+where, args...,
	), &totals); err != nil {
		return CostAnalyticsTotals{}, fmt.Errorf("summarize cost evidence: %w", err)
	}
	return totals, nil
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
	var err error
	view.Total, err = s.SummarizeCostEvidence(filter)
	if err != nil {
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
		if err := scanPrefixedCostAnalyticsTotals(&row.CostAnalyticsTotals, daily.Scan, &row.Date); err != nil {
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
		if err := scanPrefixedCostAnalyticsTotals(&row.CostAnalyticsTotals, models.Scan, &row.Provider, &row.Model); err != nil {
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
