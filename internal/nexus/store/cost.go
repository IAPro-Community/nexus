package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/ids"
)

const (
	CostKindProviderReported     = "PROVIDER_REPORTED"
	CostKindAPIEquivalentEstimate = "API_EQUIVALENT_ESTIMATE"
	CostKindInferred             = "INFERRED"
	CostKindUnknown              = "UNKNOWN"
)

// CostObservation is one local, attribution-safe execution observation.
// Nil token/cost pointers mean the source did not provide that evidence.
// They must never be coerced to zero.
type CostObservation struct {
	ID                  string    `json:"id"`
	ProjectID           string    `json:"project_id,omitempty"`
	MissionRunID        string    `json:"mission_run_id,omitempty"`
	PackageID           string    `json:"package_id,omitempty"`
	InvocationID        string    `json:"invocation_id,omitempty"`
	AgentID             string    `json:"agent_id,omitempty"`
	RuntimeGenerationID string    `json:"runtime_generation_id,omitempty"`
	RuntimeID           string    `json:"runtime_id,omitempty"`
	Provider            string    `json:"provider,omitempty"`
	Profile             string    `json:"profile,omitempty"`
	Model               string    `json:"model,omitempty"`
	Branch              string    `json:"branch,omitempty"`
	WorkspaceKind       string    `json:"workspace_kind,omitempty"`
	InputTokens         *int64    `json:"input_tokens,omitempty"`
	CachedInputTokens   *int64    `json:"cached_input_tokens,omitempty"`
	OutputTokens        *int64    `json:"output_tokens,omitempty"`
	ReasoningTokens     *int64    `json:"reasoning_tokens,omitempty"`
	DurationMS          int64     `json:"duration_ms"`
	CostMicroUSD        *int64    `json:"cost_microusd,omitempty"`
	CostKind            string    `json:"cost_kind"`
	Source              string    `json:"source"`
	Status              string    `json:"status,omitempty"`
	AccountScope        string    `json:"account_scope,omitempty"`
	ObservedAt          time.Time `json:"observed_at"`
}

type CostObservationFilter struct {
	ProjectID    string
	MissionRunID string
	AgentID      string
	Provider     string
	Profile      string
	Since        time.Time
	Limit        int
}

type CostPreferences struct {
	Enabled                   bool      `json:"enabled"`
	WindowDays                int       `json:"window_days"`
	SoftLimitMicroUSD         int64     `json:"soft_limit_microusd"`
	HardLimitMicroUSD         int64     `json:"hard_limit_microusd"`
	EmergencyLimitMicroUSD    int64     `json:"emergency_limit_microusd"`
	AnomalyMultiplier         float64   `json:"anomaly_multiplier"`
	AnomalyMinimumSamples     int       `json:"anomaly_minimum_samples"`
	AnomalyMinCostMicroUSD    int64     `json:"anomaly_min_cost_microusd"`
	UpdatedAt                 time.Time `json:"updated_at"`
}

func DefaultCostPreferences() CostPreferences {
	return CostPreferences{
		Enabled:                true,
		WindowDays:             7,
		AnomalyMultiplier:      3.0,
		AnomalyMinimumSamples:  5,
		AnomalyMinCostMicroUSD: 100000,
		UpdatedAt:              time.Now().UTC(),
	}
}

func ValidateCostPreferences(p CostPreferences) error {
	if p.WindowDays < 1 || p.WindowDays > 365 {
		return fmt.Errorf("window_days must be between 1 and 365")
	}
	if p.SoftLimitMicroUSD < 0 || p.HardLimitMicroUSD < 0 || p.EmergencyLimitMicroUSD < 0 {
		return fmt.Errorf("cost limits must not be negative")
	}
	limits := []struct {
		name  string
		value int64
	}{
		{"soft_limit_microusd", p.SoftLimitMicroUSD},
		{"hard_limit_microusd", p.HardLimitMicroUSD},
		{"emergency_limit_microusd", p.EmergencyLimitMicroUSD},
	}
	var previous int64
	var previousName string
	for _, limit := range limits {
		if limit.value == 0 {
			continue
		}
		if previous > 0 && limit.value <= previous {
			return fmt.Errorf("%s must be greater than %s", limit.name, previousName)
		}
		previous = limit.value
		previousName = limit.name
	}
	if p.AnomalyMultiplier < 1 {
		return fmt.Errorf("anomaly_multiplier must be at least 1")
	}
	if p.AnomalyMinimumSamples < 3 || p.AnomalyMinimumSamples > 1000 {
		return fmt.Errorf("anomaly_minimum_samples must be between 3 and 1000")
	}
	if p.AnomalyMinCostMicroUSD < 0 {
		return fmt.Errorf("anomaly_min_cost_microusd must not be negative")
	}
	return nil
}

func (s *Store) GetCostPreferences() (CostPreferences, error) {
	row := s.db.QueryRow(`
		SELECT enabled, window_days, soft_limit_microusd, hard_limit_microusd,
		       emergency_limit_microusd, anomaly_multiplier, anomaly_minimum_samples,
		       anomaly_min_cost_microusd, updated_at
		FROM cost_preferences WHERE id = 1
	`)
	var p CostPreferences
	var enabled int
	var updated string
	err := row.Scan(
		&enabled, &p.WindowDays, &p.SoftLimitMicroUSD, &p.HardLimitMicroUSD,
		&p.EmergencyLimitMicroUSD, &p.AnomalyMultiplier, &p.AnomalyMinimumSamples,
		&p.AnomalyMinCostMicroUSD, &updated,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return DefaultCostPreferences(), nil
	}
	if err != nil {
		return CostPreferences{}, err
	}
	p.Enabled = enabled == 1
	p.UpdatedAt, err = parseStoreTime(updated)
	if err != nil {
		return CostPreferences{}, fmt.Errorf("corrupt cost preferences updated_at: %w", err)
	}
	return p, nil
}

func (s *Store) PutCostPreferences(p CostPreferences) error {
	if err := ValidateCostPreferences(p); err != nil {
		return err
	}
	if p.UpdatedAt.IsZero() {
		p.UpdatedAt = time.Now().UTC()
	}
	enabled := 0
	if p.Enabled {
		enabled = 1
	}
	_, err := s.db.Exec(`
		INSERT INTO cost_preferences(
			id, enabled, window_days, soft_limit_microusd, hard_limit_microusd,
			emergency_limit_microusd, anomaly_multiplier, anomaly_minimum_samples,
			anomaly_min_cost_microusd, updated_at
		) VALUES(1, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			enabled = excluded.enabled,
			window_days = excluded.window_days,
			soft_limit_microusd = excluded.soft_limit_microusd,
			hard_limit_microusd = excluded.hard_limit_microusd,
			emergency_limit_microusd = excluded.emergency_limit_microusd,
			anomaly_multiplier = excluded.anomaly_multiplier,
			anomaly_minimum_samples = excluded.anomaly_minimum_samples,
			anomaly_min_cost_microusd = excluded.anomaly_min_cost_microusd,
			updated_at = excluded.updated_at
	`,
		enabled, p.WindowDays, p.SoftLimitMicroUSD, p.HardLimitMicroUSD,
		p.EmergencyLimitMicroUSD, p.AnomalyMultiplier, p.AnomalyMinimumSamples,
		p.AnomalyMinCostMicroUSD, formatStoreTime(p.UpdatedAt),
	)
	return err
}

func validateCostObservation(obs CostObservation) error {
	for name, value := range map[string]*int64{
		"input_tokens": obs.InputTokens,
		"cached_input_tokens": obs.CachedInputTokens,
		"output_tokens": obs.OutputTokens,
		"reasoning_tokens": obs.ReasoningTokens,
		"cost_microusd": obs.CostMicroUSD,
	} {
		if value != nil && *value < 0 {
			return fmt.Errorf("%s must not be negative", name)
		}
	}
	if obs.DurationMS < 0 {
		return fmt.Errorf("duration_ms must not be negative")
	}
	switch strings.ToUpper(strings.TrimSpace(obs.CostKind)) {
	case CostKindProviderReported, CostKindAPIEquivalentEstimate, CostKindInferred, CostKindUnknown:
	default:
		return fmt.Errorf("invalid cost kind %q", obs.CostKind)
	}
	return nil
}

func (s *Store) RecordCostObservation(obs CostObservation) (CostObservation, error) {
	if err := validateCostObservation(obs); err != nil {
		return CostObservation{}, err
	}
	if obs.ID == "" {
		obs.ID = "cost_" + ids.NewRuntimeID()
	}
	if obs.ObservedAt.IsZero() {
		obs.ObservedAt = time.Now().UTC()
	}
	obs.CostKind = strings.ToUpper(strings.TrimSpace(obs.CostKind))
	if obs.Source == "" {
		obs.Source = CostKindUnknown
	}
	_, err := s.db.Exec(`
		INSERT INTO cost_observations(
			id, project_id, mission_run_id, package_id, invocation_id, agent_id,
			runtime_generation_id, runtime_id, provider, profile, model, branch,
			workspace_kind, input_tokens, cached_input_tokens, output_tokens,
			reasoning_tokens, duration_ms, cost_microusd, cost_kind, source,
			status, account_scope, observed_at
		) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
	`,
		obs.ID, obs.ProjectID, obs.MissionRunID, obs.PackageID, obs.InvocationID, obs.AgentID,
		obs.RuntimeGenerationID, obs.RuntimeID, obs.Provider, obs.Profile, obs.Model, obs.Branch,
		obs.WorkspaceKind, obs.InputTokens, obs.CachedInputTokens, obs.OutputTokens,
		obs.ReasoningTokens, obs.DurationMS, obs.CostMicroUSD, obs.CostKind, obs.Source,
		obs.Status, obs.AccountScope, formatStoreTime(obs.ObservedAt),
	)
	if err != nil {
		return CostObservation{}, fmt.Errorf("record cost observation: %w", err)
	}
	return obs, nil
}

func (s *Store) ListCostObservations(filter CostObservationFilter) ([]CostObservation, error) {
	query := `SELECT id,project_id,mission_run_id,package_id,invocation_id,agent_id,
		runtime_generation_id,runtime_id,provider,profile,model,branch,workspace_kind,
		input_tokens,cached_input_tokens,output_tokens,reasoning_tokens,duration_ms,
		cost_microusd,cost_kind,source,status,account_scope,observed_at
		FROM cost_observations`
	var conditions []string
	var args []any
	add := func(condition string, value any) {
		conditions = append(conditions, condition)
		args = append(args, value)
	}
	if strings.TrimSpace(filter.ProjectID) != "" {
		add("project_id = ?", strings.TrimSpace(filter.ProjectID))
	}
	if strings.TrimSpace(filter.MissionRunID) != "" {
		add("mission_run_id = ?", strings.TrimSpace(filter.MissionRunID))
	}
	if strings.TrimSpace(filter.AgentID) != "" {
		add("agent_id = ?", strings.TrimSpace(filter.AgentID))
	}
	if strings.TrimSpace(filter.Provider) != "" {
		add("provider = ?", strings.TrimSpace(filter.Provider))
	}
	if strings.TrimSpace(filter.Profile) != "" {
		add("profile = ?", strings.TrimSpace(filter.Profile))
	}
	if !filter.Since.IsZero() {
		add("observed_at >= ?", formatStoreTime(filter.Since))
	}
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}
	query += " ORDER BY observed_at DESC LIMIT ?"
	args = append(args, limit)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list cost observations: %w", err)
	}
	defer rows.Close()

	out := make([]CostObservation, 0)
	for rows.Next() {
		var obs CostObservation
		var input, cached, output, reasoning, cost sql.NullInt64
		var observed string
		if err := rows.Scan(
			&obs.ID, &obs.ProjectID, &obs.MissionRunID, &obs.PackageID, &obs.InvocationID,
			&obs.AgentID, &obs.RuntimeGenerationID, &obs.RuntimeID, &obs.Provider,
			&obs.Profile, &obs.Model, &obs.Branch, &obs.WorkspaceKind, &input, &cached,
			&output, &reasoning, &obs.DurationMS, &cost, &obs.CostKind, &obs.Source,
			&obs.Status, &obs.AccountScope, &observed,
		); err != nil {
			return nil, fmt.Errorf("scan cost observation: %w", err)
		}
		if input.Valid {
			v := input.Int64
			obs.InputTokens = &v
		}
		if cached.Valid {
			v := cached.Int64
			obs.CachedInputTokens = &v
		}
		if output.Valid {
			v := output.Int64
			obs.OutputTokens = &v
		}
		if reasoning.Valid {
			v := reasoning.Int64
			obs.ReasoningTokens = &v
		}
		if cost.Valid {
			v := cost.Int64
			obs.CostMicroUSD = &v
		}
		parsed, err := parseStoreTime(observed)
		if err != nil {
			return nil, fmt.Errorf("corrupt cost observed_at for %q: %w", obs.ID, err)
		}
		obs.ObservedAt = parsed
		out = append(out, obs)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate cost observations: %w", err)
	}
	return out, nil
}

type CostAggregate struct {
	ObservationCount        int64 `json:"observation_count"`
	KnownCostObservations   int64 `json:"known_cost_observations"`
	UnknownCostObservations int64 `json:"unknown_cost_observations"`
	KnownTokenObservations  int64 `json:"known_token_observations"`
	CostMicroUSD            int64 `json:"cost_microusd"`
	InputTokens             int64 `json:"input_tokens"`
	CachedInputTokens       int64 `json:"cached_input_tokens"`
	OutputTokens            int64 `json:"output_tokens"`
	ReasoningTokens         int64 `json:"reasoning_tokens"`
	DurationMS              int64 `json:"duration_ms"`
}

type ProviderCostAggregate struct {
	Provider string `json:"provider"`
	Profile  string `json:"profile"`
	CostAggregate
}

func costFilterClause(filter CostObservationFilter) (string, []any) {
	var conditions []string
	var args []any
	add := func(condition string, value any) {
		conditions = append(conditions, condition)
		args = append(args, value)
	}
	if strings.TrimSpace(filter.ProjectID) != "" {
		add("project_id = ?", strings.TrimSpace(filter.ProjectID))
	}
	if strings.TrimSpace(filter.MissionRunID) != "" {
		add("mission_run_id = ?", strings.TrimSpace(filter.MissionRunID))
	}
	if strings.TrimSpace(filter.AgentID) != "" {
		add("agent_id = ?", strings.TrimSpace(filter.AgentID))
	}
	if strings.TrimSpace(filter.Provider) != "" {
		add("provider = ?", strings.TrimSpace(filter.Provider))
	}
	if strings.TrimSpace(filter.Profile) != "" {
		add("profile = ?", strings.TrimSpace(filter.Profile))
	}
	if !filter.Since.IsZero() {
		add("observed_at >= ?", formatStoreTime(filter.Since))
	}
	if len(conditions) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(conditions, " AND "), args
}

const costAggregateColumns = `
	COUNT(*),
	COALESCE(SUM(CASE WHEN cost_microusd IS NOT NULL THEN 1 ELSE 0 END),0),
	COALESCE(SUM(CASE WHEN cost_microusd IS NULL THEN 1 ELSE 0 END),0),
	COALESCE(SUM(CASE WHEN input_tokens IS NOT NULL OR cached_input_tokens IS NOT NULL OR output_tokens IS NOT NULL OR reasoning_tokens IS NOT NULL THEN 1 ELSE 0 END),0),
	COALESCE(SUM(cost_microusd),0),
	COALESCE(SUM(input_tokens),0),
	COALESCE(SUM(cached_input_tokens),0),
	COALESCE(SUM(output_tokens),0),
	COALESCE(SUM(reasoning_tokens),0),
	COALESCE(SUM(duration_ms),0)`

func (s *Store) SummarizeCostObservations(filter CostObservationFilter) (CostAggregate, error) {
	where, args := costFilterClause(filter)
	row := s.db.QueryRow("SELECT "+costAggregateColumns+" FROM cost_observations"+where, args...)
	var agg CostAggregate
	err := row.Scan(
		&agg.ObservationCount,
		&agg.KnownCostObservations,
		&agg.UnknownCostObservations,
		&agg.KnownTokenObservations,
		&agg.CostMicroUSD,
		&agg.InputTokens,
		&agg.CachedInputTokens,
		&agg.OutputTokens,
		&agg.ReasoningTokens,
		&agg.DurationMS,
	)
	if err != nil {
		return CostAggregate{}, fmt.Errorf("summarize cost observations: %w", err)
	}
	return agg, nil
}

func (s *Store) SummarizeCostByProvider(filter CostObservationFilter) ([]ProviderCostAggregate, error) {
	where, args := costFilterClause(filter)
	rows, err := s.db.Query(
		"SELECT provider,profile,"+costAggregateColumns+" FROM cost_observations"+where+" GROUP BY provider,profile ORDER BY COALESCE(SUM(cost_microusd),0) DESC, provider, profile",
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("summarize cost by provider: %w", err)
	}
	defer rows.Close()
	out := make([]ProviderCostAggregate, 0)
	for rows.Next() {
		var row ProviderCostAggregate
		if err := rows.Scan(
			&row.Provider,
			&row.Profile,
			&row.ObservationCount,
			&row.KnownCostObservations,
			&row.UnknownCostObservations,
			&row.KnownTokenObservations,
			&row.CostMicroUSD,
			&row.InputTokens,
			&row.CachedInputTokens,
			&row.OutputTokens,
			&row.ReasoningTokens,
			&row.DurationMS,
		); err != nil {
			return nil, fmt.Errorf("scan cost provider aggregate: %w", err)
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate cost provider aggregates: %w", err)
	}
	return out, nil
}
