package registry

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// ErrAttentionConflict means the presented intervention is no longer the
// current, actionable episode or has already been submitted by another client.
var ErrAttentionConflict = errors.New("attention response is stale or already submitted")

// AttentionResponseExpectation binds a quick reply to the exact runtime
// generation and durable attention revision displayed to its operator.
type AttentionResponseExpectation struct {
	Fingerprint   string `json:"attention_fingerprint"`
	Revision      uint64 `json:"attention_revision"`
	Generation    string `json:"attention_generation"`
	InteractionID string `json:"interaction_id,omitempty"`
}

func ValidateAttentionResponse(s RuntimeSession, expected AttentionResponseExpectation) error {
	conflict := func(why string) error { return fmt.Errorf("%w: %s", ErrAttentionConflict, why) }
	if expected.Revision == 0 || expected.Fingerprint == "" ||
		(s.ProviderGeneration <= 0 && s.HostGeneration <= 0) || expected.Generation == "" {
		return conflict("missing intervention or process-generation precondition")
	}
	if s.AttentionRevision != expected.Revision || s.AttentionFingerprint != expected.Fingerprint ||
		fmt.Sprintf("%d:%d", s.HostGeneration, s.ProviderGeneration) != expected.Generation ||
		s.AttentionInteractionID != expected.InteractionID {
		return conflict("intervention identity changed")
	}
	if s.AttentionSubmittedRevision == s.AttentionRevision {
		return conflict("response was already submitted")
	}
	if s.State != StateWaiting && s.State != StateApproval {
		return conflict("runtime is no longer waiting")
	}
	// PTY prompts are inferred, not permission protocol transactions. OSC and
	// structured provider-owned native_status blockers are terminal-view only
	// until a correlated provider response adapter has been negotiated.
	if s.AttentionSource != "INFERRED" || s.AttentionKind != "needs_user" ||
		s.AttentionReason != "QUESTION" || strings.TrimSpace(s.AttentionContext) == "" {
		return conflict("intervention does not permit a typed quick response")
	}
	switch s.PromptKind {
	case "yn", "choice", "free_text":
		return nil
	default:
		return conflict("prompt kind must be handled by its native terminal")
	}
}

// ClaimAttentionResponse is cross-process serialized by Registry.saveLocked.
// Only one HTTP client may submit a revision. An uncertain transport delivery
// deliberately retains the claim: a retry could type into the next question.
func (r *Registry) ClaimAttentionResponse(runtimeID string, expected AttentionResponseExpectation) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var conflict error
	err := r.saveLocked(func(fresh map[string]RuntimeSession) {
		s, ok := fresh[runtimeID]
		if !ok {
			conflict = fmt.Errorf("%w: runtime not found", ErrAttentionConflict)
			return
		}
		if conflict = ValidateAttentionResponse(s, expected); conflict != nil {
			return
		}
		s.AttentionSubmittedRevision = expected.Revision
		s.UpdatedAt = time.Now()
		fresh[runtimeID] = s
	})
	if err != nil {
		return err
	}
	return conflict
}
