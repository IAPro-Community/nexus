package registry

import (
	"errors"
	"path/filepath"
	"testing"
)

func waitingAttention() AttentionUpdate {
	return AttentionUpdate{
		State: StateWaiting, Reason: "QUESTION", Kind: "needs_user",
		Context: "Continue?", PromptKind: "yn", Fingerprint: "fingerprint-a",
		Source: "INFERRED",
	}
}

func TestAttentionRevisionAndOneShotClaimAcrossRegistryInstances(t *testing.T) {
	path := filepath.Join(t.TempDir(), "runtimes.json")
	first := NewRegistry(path)
	if err := first.Register(RuntimeSession{RuntimeID: "rt-a", State: StateWaiting,
		HostGeneration: 876543210987654321, ProviderGeneration: 123456789123456789}); err != nil {
		t.Fatal(err)
	}
	update := waitingAttention()
	if err := first.UpdateAttentionMeta("rt-a", update); err != nil {
		t.Fatal(err)
	}
	s, _ := first.Get("rt-a")
	if s.AttentionRevision != 1 || s.AttentionSource != "INFERRED" {
		t.Fatalf("unexpected initial identity: %+v", s)
	}
	if err := first.UpdateAttentionMeta("rt-a", update); err != nil {
		t.Fatal(err)
	}
	s, _ = first.Get("rt-a")
	if s.AttentionRevision != 1 {
		t.Fatal("poll of same episode incremented revision")
	}
	expectation := AttentionResponseExpectation{
		Fingerprint: "fingerprint-a", Revision: 1,
		Generation: "876543210987654321:123456789123456789",
	}
	if err := first.ClaimAttentionResponse("rt-a", expectation); err != nil {
		t.Fatal(err)
	}
	second := NewRegistry(path)
	if err := second.ClaimAttentionResponse("rt-a", expectation); !errors.Is(err, ErrAttentionConflict) {
		t.Fatalf("second client should be fenced, got %v", err)
	}
	// Same text after a working transition is a NEW intervention.
	if err := second.UpdateAttentionMeta("rt-a", AttentionUpdate{State: StateRunning, Kind: "working"}); err != nil {
		t.Fatal(err)
	}
	if err := second.UpdateAttentionMeta("rt-a", update); err != nil {
		t.Fatal(err)
	}
	s, _ = second.Get("rt-a")
	if s.AttentionRevision != 2 || s.AttentionSubmittedRevision != 1 {
		t.Fatalf("episode fence did not advance: %+v", s)
	}
	expectation.Revision = 2
	if err := first.ClaimAttentionResponse("rt-a", expectation); err != nil {
		t.Fatalf("new episode should be claimable: %v", err)
	}
}

func TestAttentionClaimRejectsStaleGenerationNativeStatusAndUnsupportedResponse(t *testing.T) {
	tests := []struct {
		name      string
		change    func(*RuntimeSession)
		expected  AttentionResponseExpectation
	}{
		{"old generation", func(s *RuntimeSession) {}, AttentionResponseExpectation{
			Fingerprint: "fingerprint-a", Revision: 1, Generation: "old-host:old-provider",
		}},
		{"native signal", func(s *RuntimeSession) { s.AttentionSource = "SIGNALLED" }, AttentionResponseExpectation{
			Fingerprint: "fingerprint-a", Revision: 1, Generation: "44:55",
		}},
		{"structured provider", func(s *RuntimeSession) { s.AttentionSource = "AUTHORITATIVE" }, AttentionResponseExpectation{
			Fingerprint: "fingerprint-a", Revision: 1, Generation: "44:55",
		}},
		{"authentication", func(s *RuntimeSession) { s.AttentionReason = "AUTH" }, AttentionResponseExpectation{
			Fingerprint: "fingerprint-a", Revision: 1, Generation: "44:55",
		}},
		{"stale runtime", func(s *RuntimeSession) { s.State = StateStopped }, AttentionResponseExpectation{
			Fingerprint: "fingerprint-a", Revision: 1, Generation: "44:55",
		}},
		{"missing revision", func(s *RuntimeSession) {}, AttentionResponseExpectation{
			Fingerprint: "fingerprint-a", Generation: "44:55",
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := RuntimeSession{RuntimeID: "rt", State: StateWaiting, AttentionReason: "QUESTION",
				AttentionKind: "needs_user", AttentionContext: "Continue?", PromptKind: "yn",
				AttentionSource: "INFERRED", AttentionFingerprint: "fingerprint-a", AttentionRevision: 1,
				HostGeneration: 44, ProviderGeneration: 55}
			tc.change(&s)
			if err := ValidateAttentionResponse(s, tc.expected); !errors.Is(err, ErrAttentionConflict) {
				t.Fatalf("expected safe 409 for %s, got %v", tc.name, err)
			}
		})
	}
}
