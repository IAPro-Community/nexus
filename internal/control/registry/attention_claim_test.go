package registry

import (
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func waitingAttention() AttentionUpdate {
	return AttentionUpdate{
		State: StateWaiting, Reason: "QUESTION", Kind: "needs_user",
		Context: "Continue?", PromptKind: "yn", Fingerprint: "fingerprint-a",
		Source: "INFERRED",
	}
}

func newAttentionOperationRegistry(t *testing.T, path string) (*Registry, AttentionResponseExpectation) {
	t.Helper()
	reg := NewRegistry(path)
	if err := reg.Register(RuntimeSession{RuntimeID: "operation-runtime", State: StateWaiting,
		HostGeneration: 44, ProviderGeneration: 55}); err != nil {
		t.Fatal(err)
	}
	if err := reg.UpdateAttentionMeta("operation-runtime", waitingAttention()); err != nil {
		t.Fatal(err)
	}
	return reg, AttentionResponseExpectation{Fingerprint: "fingerprint-a", Revision: 1, Generation: "44:55"}
}

func TestAttentionOperationRetryReturnsPersistedOutcome(t *testing.T) {
	path := filepath.Join(t.TempDir(), "runtimes.json")
	reg, expected := newAttentionOperationRegistry(t, path)
	digest := strings.Repeat("a", 64)
	claim, err := reg.ClaimAttentionOperation("operation-runtime", expected, AttentionOperationResponse, digest, "")
	if err != nil || !claim.Dispatch || claim.Status != AttentionSubmissionPending {
		t.Fatalf("initial claim=%+v err=%v", claim, err)
	}
	claim, err = reg.ClaimAttentionOperation("operation-runtime", expected, AttentionOperationResponse, digest, "")
	if err != nil || claim.Dispatch || claim.Status != AttentionSubmissionPending {
		t.Fatalf("concurrent retry=%+v err=%v", claim, err)
	}
	if err := reg.CompleteAttentionOperation("operation-runtime", expected, digest, AttentionSubmissionSubmitted); err != nil {
		t.Fatal(err)
	}
	copy, _ := reg.Get("operation-runtime")
	copy.AttentionSubmission.Status = AttentionSubmissionUncertain
	original, _ := reg.Get("operation-runtime")
	if original.AttentionSubmission.Status != AttentionSubmissionSubmitted {
		t.Fatal("returned runtime must not expose mutable submission state")
	}
	restarted := NewRegistry(path)
	claim, err = restarted.ClaimAttentionOperation("operation-runtime", expected, AttentionOperationResponse, digest, "")
	if err != nil || claim.Dispatch || claim.Status != AttentionSubmissionSubmitted {
		t.Fatalf("retry after restart=%+v err=%v", claim, err)
	}
	if _, err := restarted.ClaimAttentionOperation("operation-runtime", expected, AttentionOperationResponse, strings.Repeat("b", 64), ""); !errors.Is(err, ErrAttentionConflict) {
		t.Fatalf("different operation must conflict, got %v", err)
	}
	if err := restarted.CompleteAttentionOperation("operation-runtime", expected, digest, AttentionSubmissionUncertain); !errors.Is(err, ErrAttentionConflict) {
		t.Fatalf("final submitted outcome must not be downgraded, got %v", err)
	}
	if err := restarted.UpdateAttentionMeta("operation-runtime", AttentionUpdate{State: StateRunning, Kind: "working"}); err != nil {
		t.Fatal(err)
	}
	claim, err = restarted.ClaimAttentionOperation("operation-runtime", expected, AttentionOperationResponse, digest, "")
	if err != nil || claim.Dispatch || claim.Status != AttentionSubmissionSubmitted {
		t.Fatalf("same operation after provider progressed=%+v err=%v", claim, err)
	}
	if err := restarted.UpdateAttentionMeta("operation-runtime", waitingAttention()); err != nil {
		t.Fatal(err)
	}
	if _, err := restarted.ClaimAttentionOperation("operation-runtime", expected, AttentionOperationResponse, digest, ""); !errors.Is(err, ErrAttentionConflict) {
		t.Fatalf("old revision must not replay into next prompt, got %v", err)
	}
}

func TestAttentionOperationConcurrentClaimsDispatchOnceAcrossInstances(t *testing.T) {
	path := filepath.Join(t.TempDir(), "runtimes.json")
	_, expected := newAttentionOperationRegistry(t, path)
	digest := strings.Repeat("c", 64)
	const clients = 12
	results := make(chan AttentionSubmissionClaim, clients)
	errorsSeen := make(chan error, clients)
	var work sync.WaitGroup
	for range clients {
		work.Add(1)
		go func() {
			defer work.Done()
			claim, err := NewRegistry(path).ClaimAttentionOperation("operation-runtime", expected, AttentionOperationResponse, digest, "")
			results <- claim
			errorsSeen <- err
		}()
	}
	work.Wait()
	close(results)
	close(errorsSeen)
	for err := range errorsSeen {
		if err != nil {
			t.Fatalf("same operation concurrent claim: %v", err)
		}
	}
	dispatches := 0
	for claim := range results {
		if claim.Dispatch {
			dispatches++
		}
	}
	if dispatches != 1 {
		t.Fatalf("dispatches=%d, want exactly one", dispatches)
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
		name     string
		change   func(*RuntimeSession)
		expected AttentionResponseExpectation
	}{
		{"old generation", func(s *RuntimeSession) {}, AttentionResponseExpectation{
			Fingerprint: "fingerprint-a", Revision: 1, Generation: "old-host:old-provider",
		}},
		{"native signal", func(s *RuntimeSession) { s.AttentionSource = "SIGNALLED" }, AttentionResponseExpectation{ //nolint:misspell // Matches the canonical evidence enum.
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

func TestStructuredChoiceRequiresExactOptionAndClaimsOnce(t *testing.T) {
	reg := NewRegistry(filepath.Join(t.TempDir(), "runtimes.json"))
	session := RuntimeSession{RuntimeID: "choice-runtime", State: StateApproval,
		HostGeneration: 44, ProviderGeneration: 55}
	if err := reg.Register(session); err != nil {
		t.Fatal(err)
	}
	update := AttentionUpdate{
		State: StateApproval, Reason: "APPROVAL", Kind: "needs_user", Context: "Allow?",
		PromptKind: "choice", Fingerprint: "choice-fingerprint", Source: "AUTHORITATIVE",
		InteractionID: "req-7", Options: []AttentionOption{{ID: "allow_once", Name: "Allow once"}, {ID: "deny", Name: "Deny"}},
	}
	if err := reg.UpdateAttentionMeta(session.RuntimeID, update); err != nil {
		t.Fatal(err)
	}
	stored, _ := reg.Get(session.RuntimeID)
	expectation := AttentionResponseExpectation{
		Fingerprint: stored.AttentionFingerprint, Revision: stored.AttentionRevision,
		Generation: "44:55", InteractionID: "req-7",
	}
	if err := reg.ClaimAttentionChoice(session.RuntimeID, expectation, "allow_everything"); !errors.Is(err, ErrAttentionConflict) {
		t.Fatalf("unknown option must be rejected without claim, got %v", err)
	}
	stored, _ = reg.Get(session.RuntimeID)
	if stored.AttentionSubmittedRevision != 0 {
		t.Fatal("unknown option consumed the current interaction")
	}
	if err := reg.ClaimAttentionChoice(session.RuntimeID, expectation, "allow_once"); err != nil {
		t.Fatalf("current provider option should be accepted: %v", err)
	}
	if err := reg.ClaimAttentionChoice(session.RuntimeID, expectation, "deny"); !errors.Is(err, ErrAttentionConflict) {
		t.Fatalf("second option must not be accepted after one-shot claim, got %v", err)
	}
}
