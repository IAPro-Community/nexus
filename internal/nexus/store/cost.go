package store

import (
	"database/sql"
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
