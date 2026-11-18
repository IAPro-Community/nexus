package store

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/ids"
)

const (
	CostKindProviderReported      = "PROVIDER_REPORTED"
	CostKindAPIEquivalentEstimate = "API_EQUIVALENT_ESTIMATE"
	CostKindInferred              = "INFERRED"
	CostKindUnknown               = "UNKNOWN"
)

// CostObservation is one local, attribution-safe execution observation.
// Nil token/cost pointers mean the source did not provide that evidence.
// They must never be coerced to zero. The same holds for TTFTMS and
// OutputTokensPerSecond: nil means not measured, never synthetic zero.
type CostObservation struct {
	ID                    string    `json:"id"`
	ProjectID             string    `json:"project_id,omitempty"`
	MissionRunID          string    `json:"mission_run_id,omitempty"`
	PackageID             string    `json:"package_id,omitempty"`
	InvocationID          string    `json:"invocation_id,omitempty"`
	AgentID               string    `json:"agent_id,omitempty"`
	RuntimeGenerationID   string    `json:"runtime_generation_id,omitempty"`
	RuntimeID             string    `json:"runtime_id,omitempty"`
	Provider              string    `json:"provider,omitempty"`
	Profile               string    `json:"profile,omitempty"`
	Model                 string    `json:"model,omitempty"`
	Branch                string    `json:"branch,omitempty"`
	WorkspaceKind         string    `json:"workspace_kind,omitempty"`
	InputTokens           *int64    `json:"input_tokens,omitempty"`
	CachedInputTokens     *int64    `json:"cached_input_tokens,omitempty"`
	OutputTokens          *int64    `json:"output_tokens,omitempty"`
	ReasoningTokens       *int64    `json:"reasoning_tokens,omitempty"`
	DurationMS            int64     `json:"duration_ms"`
	CostMicroUSD          *int64    `json:"cost_microusd,omitempty"`
	CostKind              string    `json:"cost_kind"`
	Source                string    `json:"source"`
	Status                string    `json:"status,omitempty"`
	AccountScope          string    `json:"account_scope,omitempty"`
	SessionID             string    `json:"session_id,omitempty"`
	TTFTMS                *int64    `json:"ttft_ms,omitempty"`
	OutputTokensPerSecond *float64  `json:"output_tokens_per_second,omitempty"`
	ObservedAt            time.Time `json:"observed_at"`
}

type CostObservationFilter struct {
	ProjectID    string
	MissionRunID string
	AgentID      string
	Provider     string
	Profile      string
	SessionID    string
	Since        time.Time
	Limit        int
}

type CostPreferences struct {
	Enabled                bool      `json:"enabled"`
	WindowDays             int       `json:"window_days"`
	SoftLimitMicroUSD      int64     `json:"soft_limit_microusd"`
	HardLimitMicroUSD      int64     `json:"hard_limit_microusd"`
	EmergencyLimitMicroUSD int64     `json:"emergency_limit_microusd"`
	AnomalyMultiplier      float64   `json:"anomaly_multiplier"`
	AnomalyMinimumSamples  int       `json:"anomaly_minimum_samples"`
	AnomalyMinCostMicroUSD int64     `json:"anomaly_min_cost_microusd"`
	UpdatedAt              time.Time `json:"updated_at"`
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
		"input_tokens":        obs.InputTokens,
		"cached_input_tokens": obs.CachedInputTokens,
		"output_tokens":       obs.OutputTokens,
		"reasoning_tokens":    obs.ReasoningTokens,
		"cost_microusd":       obs.CostMicroUSD,
	} {
		if value != nil && *value < 0 {
			return fmt.Errorf("%s must not be negative", name)
		}
	}
	if obs.TTFTMS != nil && *obs.TTFTMS < 0 {
		return fmt.Errorf("ttft_ms must not be negative")
	}
	if obs.OutputTokensPerSecond != nil {
		throughput := *obs.OutputTokensPerSecond
		if math.IsNaN(throughput) || math.IsInf(throughput, 0) {
			return fmt.Errorf("output_tokens_per_second must be finite")
		}
		if throughput < 0 {
			return fmt.Errorf("output_tokens_per_second must not be negative")
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

// ErrCostObservationConflict is returned when a replay of the same logical
// (source, invocation_id) pair carries immutable evidence that differs from
// the already recorded observation. Callers should map it to HTTP 409.
var ErrCostObservationConflict = errors.New("cost observation conflict for source/invocation")

const costObservationInsertColumns = `
		id, project_id, mission_run_id, package_id, invocation_id, agent_id,
		runtime_generation_id, runtime_id, provider, profile, model, branch,
		workspace_kind, input_tokens, cached_input_tokens, output_tokens,
		reasoning_tokens, duration_ms, cost_microusd, cost_kind, source,
		status, account_scope, observed_at, session_id, ttft_ms,
		output_tokens_per_second
	`

func insertCostObservationExec(tx *sql.Tx, obs CostObservation, orIgnore bool) (sql.Result, error) {
	verb := "INSERT INTO"
	if orIgnore {
		verb = "INSERT OR IGNORE INTO"
	}
	query := verb + ` cost_observations(` + costObservationInsertColumns + `)
		VALUES(?,?,?,?,?,?,?,?,?,
			?,?,?,?,?,?,?,?,?,
			?,?,?,?,?,?,?,?,?)
	`
	return tx.Exec(query,
		obs.ID, obs.ProjectID, obs.MissionRunID, obs.PackageID, obs.InvocationID, obs.AgentID,
		obs.RuntimeGenerationID, obs.RuntimeID, obs.Provider, obs.Profile, obs.Model, obs.Branch,
		obs.WorkspaceKind, obs.InputTokens, obs.CachedInputTokens, obs.OutputTokens,
		obs.ReasoningTokens, obs.DurationMS, obs.CostMicroUSD, obs.CostKind, obs.Source,
		obs.Status, obs.AccountScope, formatStoreTime(obs.ObservedAt),
		obs.SessionID, obs.TTFTMS, obs.OutputTokensPerSecond,
	)
}

// costObservationImmutablesEqual compares the fields that must never change
// across replays of the same logical (source, invocation_id) observation:
// attribution, provider identity, token/cost evidence, duration and
// observed_at. Volatile fields (status, id, cost_kind normalization) are
// intentionally excluded so safe retries stay idempotent.
func costObservationImmutablesEqual(existing, replay CostObservation) bool {
	if existing.ProjectID != replay.ProjectID ||
		existing.MissionRunID != replay.MissionRunID ||
		existing.PackageID != replay.PackageID ||
		existing.AgentID != replay.AgentID ||
		existing.Provider != replay.Provider ||
		existing.Profile != replay.Profile ||
		existing.Model != replay.Model ||
		existing.DurationMS != replay.DurationMS ||
		!existing.ObservedAt.Equal(replay.ObservedAt) {
		return false
	}
	int64PtrEqual := func(a, b *int64) bool {
		if a == nil || b == nil {
			return a == nil && b == nil
		}
		return *a == *b
	}
	if !int64PtrEqual(existing.InputTokens, replay.InputTokens) ||
		!int64PtrEqual(existing.CachedInputTokens, replay.CachedInputTokens) ||
		!int64PtrEqual(existing.OutputTokens, replay.OutputTokens) ||
		!int64PtrEqual(existing.ReasoningTokens, replay.ReasoningTokens) ||
		!int64PtrEqual(existing.CostMicroUSD, replay.CostMicroUSD) {
		return false
	}
	return true
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

	tx, err := s.db.Begin()
	if err != nil {
		return CostObservation{}, fmt.Errorf("record cost observation: %w", err)
	}
	defer tx.Rollback()

	// The partial unique index dedupes (source, invocation_id) replays. Rows
	// without that logical key stay free-form and are inserted directly.
	if obs.InvocationID == "" {
		if _, err := insertCostObservationExec(tx, obs, false); err != nil {
			return CostObservation{}, fmt.Errorf("record cost observation: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return CostObservation{}, fmt.Errorf("record cost observation: %w", err)
		}
		return obs, nil
	}

	result, err := insertCostObservationExec(tx, obs, true)
	if err != nil {
		return CostObservation{}, fmt.Errorf("record cost observation: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return CostObservation{}, fmt.Errorf("record cost observation rows affected: %w", err)
	}
	if affected > 0 {
		if err := tx.Commit(); err != nil {
			return CostObservation{}, fmt.Errorf("record cost observation: %w", err)
		}
		return obs, nil
	}

	// Replay of an already recorded (source, invocation_id) pair. Identical
	// immutable evidence returns the committed row unchanged; diverging
	// evidence is a conflict, never an overwrite.
	existing, getErr := getCostObservationByKey(tx, obs.Source, obs.InvocationID)
	if getErr != nil {
		return CostObservation{}, getErr
	}
	if costObservationImmutablesEqual(existing, obs) {
		if err := tx.Commit(); err != nil {
			return CostObservation{}, fmt.Errorf("record cost observation: %w", err)
		}
		return existing, nil
	}
	return CostObservation{}, fmt.Errorf("%w: source=%q invocation=%q", ErrCostObservationConflict, obs.Source, obs.InvocationID)
}

const costObservationSelectColumns = `id,project_id,mission_run_id,package_id,invocation_id,agent_id,
		runtime_generation_id,runtime_id,provider,profile,model,branch,workspace_kind,
		input_tokens,cached_input_tokens,output_tokens,reasoning_tokens,duration_ms,
		cost_microusd,cost_kind,source,status,account_scope,observed_at,
		session_id,ttft_ms,output_tokens_per_second`

// scanCostObservation scans one full cost_observations row. NULL evidence
// stays nil; it is never coerced to a synthetic zero.
func scanCostObservation(scan func(dest ...any) error) (CostObservation, error) {
	var obs CostObservation
	var input, cached, output, reasoning, cost, ttft sql.NullInt64
	var tps sql.NullFloat64
	var observed string
	if err := scan(
		&obs.ID, &obs.ProjectID, &obs.MissionRunID, &obs.PackageID, &obs.InvocationID,
		&obs.AgentID, &obs.RuntimeGenerationID, &obs.RuntimeID, &obs.Provider,
		&obs.Profile, &obs.Model, &obs.Branch, &obs.WorkspaceKind, &input, &cached,
		&output, &reasoning, &obs.DurationMS, &cost, &obs.CostKind, &obs.Source,
		&obs.Status, &obs.AccountScope, &observed,
		&obs.SessionID, &ttft, &tps,
	); err != nil {
		return CostObservation{}, err
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
	if ttft.Valid {
		v := ttft.Int64
		obs.TTFTMS = &v
	}
	if tps.Valid {
		v := tps.Float64
		obs.OutputTokensPerSecond = &v
	}
	parsed, err := parseStoreTime(observed)
	if err != nil {
		return CostObservation{}, fmt.Errorf("corrupt cost observed_at for %q: %w", obs.ID, err)
	}
	obs.ObservedAt = parsed
	return obs, nil
}

func getCostObservationByKey(tx *sql.Tx, source, invocationID string) (CostObservation, error) {
	obs, err := scanCostObservation(func(dest ...any) error {
		return tx.QueryRow(
			`SELECT `+costObservationSelectColumns+`
			FROM cost_observations WHERE source = ? AND invocation_id = ? LIMIT 1`,
			source, invocationID,
		).Scan(dest...)
	})
	if errors.Is(err, sql.ErrNoRows) {
		return CostObservation{}, fmt.Errorf("cost observation lookup: %w", err)
	}
	if err != nil {
		return CostObservation{}, fmt.Errorf("get cost observation by source/invocation: %w", err)
	}
	return obs, nil
}

func (s *Store) ListCostObservations(filter CostObservationFilter) ([]CostObservation, error) {
	query := "SELECT " + costObservationSelectColumns + " FROM cost_observations"
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
	if strings.TrimSpace(filter.SessionID) != "" {
		add("session_id = ?", strings.TrimSpace(filter.SessionID))
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
		obs, err := scanCostObservation(rows.Scan)
		if err != nil {
			return nil, fmt.Errorf("scan cost observation: %w", err)
		}
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
	if strings.TrimSpace(filter.SessionID) != "" {
		add("session_id = ?", strings.TrimSpace(filter.SessionID))
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
