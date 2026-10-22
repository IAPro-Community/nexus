package store

import (
	"context"
	"testing"
	"time"
)

func TestExternalClaudeAtomicDedupAndSeparatedCost(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	msg := ExternalClaudeMessage{MessageHash: "hash-a", Model: "claude-test", InputTokens: 10, OutputTokens: 5, ObservedAt: now}
	if err := s.SaveExternalClaudeBatch(ctx, "LOCAL_DEFAULT_UNVERIFIED", "file-hash", 100, []ExternalClaudeMessage{msg}); err != nil {
		t.Fatal(err)
	}
	offset, err := s.ExternalClaudeOffset(ctx, "LOCAL_DEFAULT_UNVERIFIED", "file-hash")
	if err != nil || offset != 100 {
		t.Fatalf("offset: %d %v", offset, err)
	}
	msg.InputTokens = 20
	if err := s.SaveExternalClaudeBatch(ctx, "LOCAL_DEFAULT_UNVERIFIED", "file-hash", 160, []ExternalClaudeMessage{msg}); err != nil {
		t.Fatal(err)
	}
	summary, err := s.SummarizeExternalClaude(ctx, 7)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Messages != 1 || summary.InputTokens != 20 || !summary.Unattributed {
		t.Fatalf("idempotent external usage: %+v", summary)
	}
	agg, err := s.SummarizeCostObservations(CostObservationFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if agg.ObservationCount != 0 {
		t.Fatalf("external transcripts must not contaminate ledger: %+v", agg)
	}
	bad := msg
	bad.InputTokens = -1
	if err := s.SaveExternalClaudeBatch(ctx, "LOCAL_DEFAULT_UNVERIFIED", "file-hash", 200, []ExternalClaudeMessage{bad}); err == nil {
		t.Fatal("invalid batch accepted")
	}
	offset, _ = s.ExternalClaudeOffset(ctx, "LOCAL_DEFAULT_UNVERIFIED", "file-hash")
	if offset != 160 {
		t.Fatalf("rolled back offset advanced to %d", offset)
	}
	if err := s.SaveExternalClaudeBatch(ctx, "wrong-account", "file-hash", 250, nil); err == nil {
		t.Fatal("arbitrary scope accepted")
	}
}

func TestExternalClaudeClearForgetsOffsetsAndMessages(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	now := time.Now().UTC()
	msg := ExternalClaudeMessage{MessageHash: "hashed-id", Model: "claude", InputTokens: 100, ObservedAt: now}
	if err := s.SaveExternalClaudeBatch(ctx, "LOCAL_DEFAULT_UNVERIFIED", "file-hash", 125, []ExternalClaudeMessage{msg}); err != nil {
		t.Fatal(err)
	}
	if err := s.ClearExternalClaude(ctx); err != nil {
		t.Fatal(err)
	}
	summary, err := s.SummarizeExternalClaude(ctx, 7)
	if err != nil || summary.Messages != 0 {
		t.Fatalf("external observations not cleared: %+v, %v", summary, err)
	}
	offset, err := s.ExternalClaudeOffset(ctx, "LOCAL_DEFAULT_UNVERIFIED", "file-hash")
	if err != nil || offset != 0 {
		t.Fatalf("external offsets not cleared: %d, %v", offset, err)
	}
}

func TestPruneExternalClaudeRetainsNewerEvidenceOnly(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	now := time.Now().UTC()
	old := ExternalClaudeMessage{
		MessageHash: "old-hash", Model: "claude", InputTokens: 10,
		ObservedAt: now.Add(-120 * 24 * time.Hour),
	}
	recent := ExternalClaudeMessage{
		MessageHash: "new-hash", Model: "claude", InputTokens: 20,
		ObservedAt: now,
	}
	if err := s.SaveExternalClaudeBatch(ctx, "LOCAL_DEFAULT_UNVERIFIED", "old-file", 50, []ExternalClaudeMessage{old}); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveExternalClaudeBatch(ctx, "LOCAL_DEFAULT_UNVERIFIED", "recent-file", 100, []ExternalClaudeMessage{recent}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.DB().Exec("UPDATE external_claude_offsets SET updated_at=? WHERE file_hash=?", formatStoreTime(now.Add(-120*24*time.Hour)), "old-file"); err != nil {
		t.Fatal(err)
	}
	if err := s.PruneExternalClaude(ctx, now.Add(-90*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	summary, err := s.SummarizeExternalClaude(ctx, 365)
	if err != nil || summary.Messages != 1 || summary.InputTokens != 20 {
		t.Fatalf("prune: %+v %v", summary, err)
	}
	oldOffset, err := s.ExternalClaudeOffset(ctx, "LOCAL_DEFAULT_UNVERIFIED", "old-file")
	if err != nil || oldOffset != 0 {
		t.Fatalf("inactive offset persisted: %d %v", oldOffset, err)
	}
	newOffset, err := s.ExternalClaudeOffset(ctx, "LOCAL_DEFAULT_UNVERIFIED", "recent-file")
	if err != nil || newOffset != 100 {
		t.Fatalf("recent offset lost: %d %v", newOffset, err)
	}
}

func TestExternalClaudeOlderDuplicateCannotOverwriteNewerObservation(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	newer := ExternalClaudeMessage{
		MessageHash: "same-message", Model: "claude-new",
		InputTokens: 99, OutputTokens: 15, ObservedAt: now,
	}
	older := ExternalClaudeMessage{
		MessageHash: "same-message", Model: "claude-old",
		InputTokens: 5, OutputTokens: 2, ObservedAt: now.Add(-time.Hour),
	}
	if err := s.SaveExternalClaudeBatch(ctx, "LOCAL_DEFAULT_UNVERIFIED", "file-new", 100,
		[]ExternalClaudeMessage{newer}); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveExternalClaudeBatch(ctx, "LOCAL_DEFAULT_UNVERIFIED", "file-old", 100,
		[]ExternalClaudeMessage{older}); err != nil {
		t.Fatal(err)
	}
	out, err := s.SummarizeExternalClaude(ctx, 7)
	if err != nil || out.Messages != 1 || out.InputTokens != 99 || out.OutputTokens != 15 {
		t.Fatalf("older duplicate replaced newer evidence: %+v, %v", out, err)
	}
}
