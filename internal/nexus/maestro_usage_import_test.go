package nexus

import (
	"strings"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/handoff"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func maestroUsageTestStore(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(t.TempDir() + "/nexus.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func maestroUsageRow(submissionID, status string, usage *handoff.MaestroSubmissionUsage, updatedAt string) handoff.MaestroSubmissionStatus {
	return handoff.MaestroSubmissionStatus{
		SchemaVersion: 1, Authority: "MAESTRO", Owner: "MAESTRO",
		SubmissionID: submissionID, Status: status,
		RunID: "run-" + submissionID, ManifestID: "manifest-" + submissionID,
		TaskCount:      1,
		Providers:      []string{"opencode"},
		CreatedAt:      updatedAt,
		UpdatedAt:      updatedAt,
		ResultEvidence: "RUN_STORE_OBSERVED",
		Usage:          usage,
	}
}

func maestroUsage(t *testing.T) *handoff.MaestroSubmissionUsage {
	t.Helper()
	in, out, cached, reasoning, duration := int64(1200), int64(300), int64(900), int64(64), int64(45000)
	return &handoff.MaestroSubmissionUsage{
		SchemaVersion:   1,
		Evidence:        "RUN_STORE_OBSERVED",
		TokensInput:     &in,
		TokensOutput:    &out,
		TokensCached:    &cached,
		TokensReasoning: &reasoning,
		DurationMS:      &duration,
	}
}

func TestImportMaestroUsageRowsRecordsTerminalUsageOnly(t *testing.T) {
	st := maestroUsageTestStore(t)
	observed := time.Now().UTC().Truncate(time.Second)
	timestamp := observed.Format(time.RFC3339Nano)
	longProvider := strings.Repeat("p", 80)
	rows := []handoff.MaestroSubmissionStatus{
		maestroUsageRow("submission-complete-01", "COMPLETED", maestroUsage(t), timestamp),
		// Partial evidence: only output measured; unmeasured stays absent.
		func() handoff.MaestroSubmissionStatus {
			usage := maestroUsage(t)
			usage.TokensInput, usage.TokensCached, usage.TokensReasoning, usage.DurationMS = nil, nil, nil, nil
			usage.TokensOutput = ptrInt64(100)
			return maestroUsageRow("submission-failed-02", "FAILED", usage, timestamp)
		}(),
		// Terminal without usage evidence: no line, never a synthetic zero.
		maestroUsageRow("submission-nousage-03", "BLOCKED", nil, timestamp),
		// Non-terminal rows are never imported even with usage.
		maestroUsageRow("submission-pending-04", "DISPATCHED", maestroUsage(t), timestamp),
		// Unattributable observation time is skipped fail-closed.
		maestroUsageRow("submission-badtime-05", "CANCELLED", maestroUsage(t), "not-a-timestamp"), //nolint:misspell // Canonical Maestro Bridge status.
		// Provider identity is joined and bounded.
		func() handoff.MaestroSubmissionStatus {
			row := maestroUsageRow("submission-longprov-06", "COMPLETED", maestroUsage(t), timestamp)
			row.Providers = []string{longProvider}
			row.TaskCount = 1
			return row
		}(),
	}

	result, err := importMaestroUsageRows(st, "project-1", rows, map[string]bool{})
	if err != nil {
		t.Fatalf("import failed: %v", err)
	}
	if result.Imported != 3 || result.Skipped != 3 || result.Conflicts != 0 {
		t.Fatalf("unexpected first pass counts: %+v", result)
	}

	observations, err := st.ListCostObservations(store.CostObservationFilter{ProjectID: "project-1", Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	if len(observations) != 3 {
		t.Fatalf("expected 3 recorded observations, got %d", len(observations))
	}
	byInvocation := map[string]store.CostObservation{}
	for _, obs := range observations {
		byInvocation[obs.InvocationID] = obs
	}

	complete := byInvocation["submission-complete-01"]
	if complete.ID == "" || complete.Source != MaestroUsageImportSource ||
		complete.MissionRunID != "submission-complete-01" || complete.ProjectID != "project-1" ||
		complete.Status != "COMPLETED" || complete.CostKind != store.CostKindUnknown {
		t.Fatalf("wrong attribution for completed submission: %+v", complete)
	}
	if complete.Provider != "opencode" || complete.Model != "" || complete.Profile != "" ||
		complete.Branch != "" || complete.WorkspaceKind != "" || complete.AccountScope != "" {
		t.Fatalf("wrong provider/attribution scope: %+v", complete)
	}
	if complete.CostMicroUSD != nil {
		t.Fatalf("usage is not cost evidence: cost was invented (%d)", *complete.CostMicroUSD)
	}
	if complete.InputTokens == nil || *complete.InputTokens != 1200 ||
		complete.OutputTokens == nil || *complete.OutputTokens != 300 ||
		complete.CachedInputTokens == nil || *complete.CachedInputTokens != 900 ||
		complete.ReasoningTokens == nil || *complete.ReasoningTokens != 64 {
		t.Fatalf("measured tokens lost or coerced: %+v", complete)
	}
	if complete.DurationMS != 45000 {
		t.Fatalf("measured duration lost: %d", complete.DurationMS)
	}
	if !complete.ObservedAt.Equal(observed) {
		t.Fatalf("observed_at must come from the bridge row, got %v want %v", complete.ObservedAt, observed)
	}

	partial := byInvocation["submission-failed-02"]
	if partial.InputTokens != nil || partial.CachedInputTokens != nil || partial.ReasoningTokens != nil {
		t.Fatalf("unmeasured usage coerced to zero: %+v", partial)
	}
	if partial.OutputTokens == nil || *partial.OutputTokens != 100 {
		t.Fatalf("measured output tokens lost: %+v", partial)
	}
	if partial.DurationMS != 0 {
		t.Fatalf("unmeasured duration must not be synthesized: %d", partial.DurationMS)
	}

	long := byInvocation["submission-longprov-06"]
	if long.Provider != strings.Repeat("p", 64) {
		t.Fatalf("provider join not truncated to 64: %d", len(long.Provider))
	}
}

func ptrInt64(v int64) *int64 { return &v }

func TestImportMaestroUsageRowsIsIdempotentAcrossPasses(t *testing.T) {
	st := maestroUsageTestStore(t)
	timestamp := time.Now().UTC().Format(time.RFC3339Nano)
	rows := []handoff.MaestroSubmissionStatus{
		maestroUsageRow("submission-replay-01", "COMPLETED", maestroUsage(t), timestamp),
		maestroUsageRow("submission-replay-02", "FAILED", nil, timestamp),
	}

	first, err := importMaestroUsageRows(st, "project-1", rows, map[string]bool{})
	if err != nil || first.Imported != 1 || first.Skipped != 1 {
		t.Fatalf("unexpected first pass: %+v %v", first, err)
	}

	// Second pass over the same snapshot (replay/restart): the already
	// recorded submission is counted as skipped, never imported twice.
	known := maestroUsageKnownInvocations(st, "project-1")
	second, err := importMaestroUsageRows(st, "project-1", rows, known)
	if err != nil || second.Imported != 0 || second.Skipped != 2 || second.Conflicts != 0 {
		t.Fatalf("unexpected replay pass: %+v %v", second, err)
	}
	if !known["submission-replay-01"] {
		t.Fatal("known invocation index missing recorded submission")
	}
}

func TestImportMaestroUsageRowsCountsConflictsWithoutAborting(t *testing.T) {
	st := maestroUsageTestStore(t)
	timestamp := time.Now().UTC().Format(time.RFC3339Nano)
	// Pre-record diverging immutable evidence under the same logical
	// (source, invocation_id). The ledger must never overwrite it.
	conflicting := maestroUsageRow("submission-conflict-01", "COMPLETED", maestroUsage(t), timestamp)
	if _, err := st.RecordCostObservation(store.CostObservation{
		ProjectID:    "project-1",
		MissionRunID: conflicting.SubmissionID,
		InvocationID: conflicting.SubmissionID,
		Provider:     "opencode",
		InputTokens:  ptrInt64(999),
		DurationMS:   1,
		CostKind:     store.CostKindUnknown,
		Source:       MaestroUsageImportSource,
		Status:       "COMPLETED",
		ObservedAt:   time.Now().UTC(),
	}); err != nil {
		t.Fatalf("seed conflicting observation: %v", err)
	}
	rows := []handoff.MaestroSubmissionStatus{
		conflicting,
		maestroUsageRow("submission-after-02", "COMPLETED", maestroUsage(t), timestamp),
	}
	result, err := importMaestroUsageRows(st, "project-1", rows, nil)
	if err != nil {
		t.Fatalf("conflict must not abort the batch: %v", err)
	}
	if result.Conflicts != 1 || result.Imported != 1 || result.Skipped != 0 {
		t.Fatalf("unexpected conflict pass counts: %+v", result)
	}
	// The conflicting evidence stays untouched.
	observations, err := st.ListCostObservations(store.CostObservationFilter{ProjectID: "project-1", Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	for _, obs := range observations {
		if obs.InvocationID == "submission-conflict-01" && (obs.InputTokens == nil || *obs.InputTokens != 999) {
			t.Fatalf("conflicting observation was overwritten: %+v", obs)
		}
	}
}

func TestImportMaestroUsageRowsRejectsRowsWithoutTimestamp(t *testing.T) {
	st := maestroUsageTestStore(t)
	rows := []handoff.MaestroSubmissionStatus{
		maestroUsageRow("submission-notime-01", "COMPLETED", maestroUsage(t), ""),
	}
	result, err := importMaestroUsageRows(st, "project-1", rows, map[string]bool{})
	if err != nil {
		t.Fatalf("unattributable row must be skipped, not fatal: %v", err)
	}
	if result.Imported != 0 || result.Skipped != 1 {
		t.Fatalf("unexpected counts: %+v", result)
	}
	observations, _ := st.ListCostObservations(store.CostObservationFilter{ProjectID: "project-1"})
	if len(observations) != 0 {
		t.Fatalf("no line may exist without an honest observed_at: %+v", observations)
	}
}
