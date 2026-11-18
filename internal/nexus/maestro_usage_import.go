package nexus

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/handoff"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

// MaestroUsageImportSource is the ledger source attribution for measured
// Maestro submission usage. Together with invocation_id (= submissionId) it
// makes every import idempotent: replays and restarts never double count.
const MaestroUsageImportSource = "MAESTRO_SUBMISSION"

// MaestroUsageImportResult reports what one import pass did. Counts are
// observational only; nothing here executes work or fabricates evidence.
type MaestroUsageImportResult struct {
	Imported  int `json:"imported"`
	Skipped   int `json:"skipped"`
	Conflicts int `json:"conflicts"`
}

// maestroTerminalSubmissionStatus lists the statuses after which the Maestro
// run store considers usage evidence final. Non-terminal rows are never
// imported: their usage may still change.
var maestroTerminalSubmissionStatus = map[string]bool{
	"COMPLETED": true,
	"FAILED":    true,
	"BLOCKED":   true,
	"TIMED_OUT": true,
	"CANCELLED": true, //nolint:misspell // Canonical Maestro Bridge status.
}

const (
	maestroUsageImportPageLimit   = 100
	maestroUsageImportMaxPages    = 100
	maestroUsageImportMaxSeen     = 1000
	maestroUsageImportProviderMax = 64
)

// ImportMaestroUsageObservations fetches the Maestro submission snapshot for
// the project's canonical workspace and records one CostObservation per
// terminal submission that carries measured usage evidence. Cost is never
// invented (CostKind UNKNOWN, no microUSD), unmeasured fields stay NULL and
// the ledger's (source, invocation_id) idempotency prevents double counting.
func (n *Nexus) ImportMaestroUsageObservations(ctx context.Context, project *store.Project) (MaestroUsageImportResult, error) {
	if project == nil || strings.TrimSpace(project.CanonicalPath) == "" {
		return MaestroUsageImportResult{}, errors.New("project canonical workspace is required for Maestro usage import")
	}
	st, err := n.OpenProject()
	if err != nil {
		return MaestroUsageImportResult{}, err
	}
	rows, err := fetchMaestroSubmissionRows(ctx, project.CanonicalPath)
	if err != nil {
		return MaestroUsageImportResult{}, fmt.Errorf("maestro submission snapshot unavailable: %w", err)
	}
	return importMaestroUsageRows(st, project.ID, rows, maestroUsageKnownInvocations(st, project.ID))
}

// fetchMaestroSubmissionRows pages through the Maestro submission snapshot
// with a bounded page budget. It never dispatches work and never fabricates
// pagination: the bridge cursor contract is followed as-is.
func fetchMaestroSubmissionRows(ctx context.Context, workspace string) ([]handoff.MaestroSubmissionStatus, error) {
	rows := make([]handoff.MaestroSubmissionStatus, 0, maestroUsageImportPageLimit)
	cursor, offset := "", 0
	for page := 0; page < maestroUsageImportMaxPages; page++ {
		snapshot, err := handoff.FetchMaestroSubmissionSnapshot(ctx, workspace, cursor,
			maestroUsageImportPageLimit, offset)
		if err != nil {
			return nil, err
		}
		rows = append(rows, snapshot.Submissions...)
		if snapshot.NextOffset == nil {
			return rows, nil
		}
		cursor = snapshot.Cursor
		offset = *snapshot.NextOffset
	}
	return rows, nil
}

// maestroUsageKnownInvocations collects the invocation IDs already recorded
// for this project under the Maestro submission source so a replay is counted
// as skipped, not imported. The ledger's unique index remains the hard
// dedupe guarantee; this map only keeps the counts honest.
func maestroUsageKnownInvocations(st *store.Store, projectID string) map[string]bool {
	known := make(map[string]bool)
	existing, err := st.ListCostObservations(store.CostObservationFilter{
		ProjectID: projectID,
		Limit:     maestroUsageImportMaxSeen,
	})
	if err != nil {
		// Counting aid only: on lookup failure fall back to the ledger's own
		// (source, invocation_id) idempotency instead of aborting the import.
		return known
	}
	for _, obs := range existing {
		if obs.Source == MaestroUsageImportSource && obs.InvocationID != "" {
			known[obs.InvocationID] = true
		}
	}
	return known
}

// importMaestroUsageRows is the pure core of the import. Rows without usage,
// non-terminal rows and rows with an unparseable observation time are skipped
// (no line, never a synthetic zero); ErrCostObservationConflict is counted
// and does not abort the batch; any other store error aborts fail-closed.
// knownInvocations may be nil, in which case dedupe relies on the ledger.
func importMaestroUsageRows(st *store.Store, projectID string, rows []handoff.MaestroSubmissionStatus, knownInvocations map[string]bool) (MaestroUsageImportResult, error) {
	result := MaestroUsageImportResult{}
	for _, row := range rows {
		if row.Usage == nil || !maestroTerminalSubmissionStatus[row.Status] {
			result.Skipped++
			continue
		}
		if knownInvocations[row.SubmissionID] {
			result.Skipped++
			continue
		}
		observedAt, err := parseMaestroSubmissionTime(row.UpdatedAt)
		if err != nil {
			result.Skipped++
			continue
		}
		_, err = st.RecordCostObservation(maestroUsageObservation(projectID, row, observedAt))
		if err != nil {
			if errors.Is(err, store.ErrCostObservationConflict) {
				// A conflicting replay means diverging immutable evidence for
				// the same (source, invocation_id). Never overwrite; count and
				// keep the batch running.
				result.Conflicts++
				if knownInvocations != nil {
					knownInvocations[row.SubmissionID] = true
				}
				continue
			}
			return result, fmt.Errorf("record Maestro usage observation: %w", err)
		}
		if knownInvocations != nil {
			knownInvocations[row.SubmissionID] = true
		}
		result.Imported++
	}
	return result, nil
}

// maestroUsageObservation attributes one measured Maestro submission to the
// cost ledger. Model, profile, account scope, branch and workspace kind stay
// empty (no proof), cost stays UNKNOWN/nil (usage is not cost evidence) and
// nil usage fields are passed through as nil, never coerced to zero.
func maestroUsageObservation(projectID string, row handoff.MaestroSubmissionStatus, observedAt time.Time) store.CostObservation {
	provider := strings.Join(row.Providers, "+")
	if len(provider) > maestroUsageImportProviderMax {
		provider = provider[:maestroUsageImportProviderMax]
	}
	duration := int64(0)
	if row.Usage.DurationMS != nil && *row.Usage.DurationMS > 0 {
		duration = *row.Usage.DurationMS
	}
	return store.CostObservation{
		ProjectID:         projectID,
		MissionRunID:      row.SubmissionID,
		InvocationID:      row.SubmissionID,
		Provider:          provider,
		Model:             "",
		Profile:           "",
		Branch:            "",
		WorkspaceKind:     "",
		InputTokens:       row.Usage.TokensInput,
		CachedInputTokens: row.Usage.TokensCached,
		OutputTokens:      row.Usage.TokensOutput,
		ReasoningTokens:   row.Usage.TokensReasoning,
		DurationMS:        duration,
		CostMicroUSD:      nil,
		CostKind:          store.CostKindUnknown,
		Source:            MaestroUsageImportSource,
		Status:            row.Status,
		AccountScope:      "",
		ObservedAt:        observedAt,
	}
}

// parseMaestroSubmissionTime parses the bridge's ISO-8601 submission
// timestamp. An unparseable timestamp means the observation time cannot be
// attributed honestly; the row is skipped rather than stamped with "now".
func parseMaestroSubmissionTime(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, errors.New("missing Maestro submission timestamp")
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("unparseable Maestro submission timestamp %q: %w", value, err)
	}
	return parsed.UTC(), nil
}
