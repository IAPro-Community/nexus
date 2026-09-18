package store

import (
	"errors"
	"testing"
)

func TestValidationEvidenceAppendBuildsHashChainAndSupportsReadOnlyQueries(t *testing.T) {
	s := openTestStore(t)
	stream, err := s.CreateValidationEvidenceStream("", "baseline")
	if err != nil {
		t.Fatalf("create stream: %v", err)
	}

	first, err := s.AppendValidationEvidence(ValidationEvidenceAppendRequest{
		StreamID: stream.ID,
		Entry: ValidationEvidenceEntry{
			GitSHA: "abc123", Scenario: "go-test", Outcome: EvidencePass,
			Confidence: EvidenceObserved, EvidenceJSON: `{"command":"go test ./..."}`,
		},
	})
	if err != nil {
		t.Fatalf("append first entry: %v", err)
	}
	if first.Sequence != 1 || first.PreviousHash != "" || first.EntryHash == "" {
		t.Fatalf("unexpected first chain entry: %+v", first)
	}

	second, err := s.AppendValidationEvidence(ValidationEvidenceAppendRequest{
		StreamID:         stream.ID,
		ExpectedSequence: &first.Sequence,
		ExpectedHash:     &first.EntryHash,
		Entry: ValidationEvidenceEntry{
			ID: "entry-second", Scenario: "go-vet", Outcome: EvidencePass,
			Confidence: EvidenceVerified,
		},
	})
	if err != nil {
		t.Fatalf("append second entry: %v", err)
	}
	if second.Sequence != 2 || second.PreviousHash != first.EntryHash {
		t.Fatalf("unexpected second chain entry: %+v", second)
	}

	got, err := s.GetValidationEvidenceEntry(first.ID)
	if err != nil || got.EntryHash != first.EntryHash {
		t.Fatalf("get first entry: got=%+v err=%v", got, err)
	}
	entries, err := s.ListValidationEvidenceEntries(stream.ID, 0)
	if err != nil {
		t.Fatalf("list entries: %v", err)
	}
	if len(entries) != 2 || entries[0].Sequence != 1 || entries[1].Sequence != 2 {
		t.Fatalf("unexpected ordered entries: %+v", entries)
	}
	if err := s.VerifyValidationEvidenceChain(stream.ID); err != nil {
		t.Fatalf("verify chain: %v", err)
	}

	head, err := s.GetValidationEvidenceStream(stream.ID)
	if err != nil {
		t.Fatalf("get stream: %v", err)
	}
	if head.LastSequence != 2 || head.LastHash != second.EntryHash {
		t.Fatalf("unexpected stream head: %+v", head)
	}
}

func TestValidationEvidenceAppendUsesCASAndIdempotencyKey(t *testing.T) {
	s := openTestStore(t)
	stream, err := s.CreateValidationEvidenceStream("", "checks")
	if err != nil {
		t.Fatalf("create stream: %v", err)
	}
	entry := ValidationEvidenceEntry{ID: "stable-entry", Scenario: "format", Outcome: EvidencePass, Confidence: EvidenceVerified}
	first, err := s.AppendValidationEvidence(ValidationEvidenceAppendRequest{StreamID: stream.ID, Entry: entry})
	if err != nil {
		t.Fatalf("append: %v", err)
	}
	retry, err := s.AppendValidationEvidence(ValidationEvidenceAppendRequest{
		StreamID:         stream.ID,
		ExpectedSequence: func() *int64 { v := int64(0); return &v }(),
		ExpectedHash:     func() *string { v := ""; return &v }(),
		Entry:            entry,
	})
	if err != nil {
		t.Fatalf("idempotent retry: %v", err)
	}
	if retry.EntryHash != first.EntryHash || retry.Sequence != first.Sequence {
		t.Fatalf("retry advanced or changed entry: first=%+v retry=%+v", first, retry)
	}

	wrongSequence := int64(0)
	wrongHash := ""
	_, err = s.AppendValidationEvidence(ValidationEvidenceAppendRequest{
		StreamID: stream.ID, ExpectedSequence: &wrongSequence, ExpectedHash: &wrongHash,
		Entry: ValidationEvidenceEntry{Scenario: "stale", Outcome: EvidencePass, Confidence: EvidenceObserved},
	})
	if !errors.Is(err, ErrValidationEvidenceConflict) {
		t.Fatalf("expected CAS conflict, got %v", err)
	}
	entries, err := s.ListValidationEvidenceEntries(stream.ID, 0)
	if err != nil {
		t.Fatalf("list after conflict: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("CAS conflict must not append, got %d entries", len(entries))
	}
}

func TestValidationEvidenceChainDetectsTampering(t *testing.T) {
	s := openTestStore(t)
	stream, err := s.CreateValidationEvidenceStream("", "tamper")
	if err != nil {
		t.Fatalf("create stream: %v", err)
	}
	entry, err := s.AppendValidationEvidence(ValidationEvidenceAppendRequest{
		StreamID: stream.ID,
		Entry:    ValidationEvidenceEntry{Scenario: "test", Outcome: EvidencePass, Confidence: EvidenceObserved},
	})
	if err != nil {
		t.Fatalf("append: %v", err)
	}
	if _, err := s.DB().Exec(`UPDATE validation_evidence_entries SET outcome=? WHERE id=?`, EvidenceFail, entry.ID); err != nil {
		t.Fatalf("tamper fixture: %v", err)
	}
	if err := s.VerifyValidationEvidenceChain(stream.ID); !errors.Is(err, ErrValidationEvidenceIntegrity) {
		t.Fatalf("expected integrity failure, got %v", err)
	}
}

func TestValidationEvidenceRejectsInvalidPayload(t *testing.T) {
	s := openTestStore(t)
	stream, err := s.CreateValidationEvidenceStream("", "invalid")
	if err != nil {
		t.Fatalf("create stream: %v", err)
	}
	_, err = s.AppendValidationEvidence(ValidationEvidenceAppendRequest{StreamID: stream.ID, Entry: ValidationEvidenceEntry{
		Scenario: "bad-json", Outcome: EvidencePass, Confidence: EvidenceObserved, EvidenceJSON: "not-json",
	}})
	if !errors.Is(err, ErrValidationEvidenceInvalid) {
		t.Fatalf("expected invalid payload error, got %v", err)
	}
	_, err = s.AppendValidationEvidence(ValidationEvidenceAppendRequest{StreamID: stream.ID, Entry: ValidationEvidenceEntry{
		Scenario: "missing-outcome", Confidence: EvidenceObserved,
	}})
	if !errors.Is(err, ErrValidationEvidenceInvalid) {
		t.Fatalf("expected missing outcome error, got %v", err)
	}
}
