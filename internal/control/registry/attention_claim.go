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
	if err := validateAttentionIdentity(s, expected); err != nil {
		return err
	}
	// PTY prompts are inferred, not permission protocol transactions. Structured
	// choices use ValidateAttentionChoice and their exact provider option IDs.
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

// ValidateAttentionChoice only accepts an exact option from the current
// structured provider interaction. It never interprets UI text as a PTY key.
func ValidateAttentionChoice(s RuntimeSession, expected AttentionResponseExpectation, optionID string) error {
	if err := validateAttentionIdentity(s, expected); err != nil {
		return err
	}
	if s.AttentionSource != "AUTHORITATIVE" || s.AttentionKind != "needs_user" ||
		s.PromptKind != "choice" || expected.InteractionID == "" || optionID == "" {
		return fmt.Errorf("%w: intervention is not a structured choice", ErrAttentionConflict)
	}
	for _, option := range s.AttentionOptions {
		if option.ID == optionID {
			return nil
		}
	}
	return fmt.Errorf("%w: option is not part of the current interaction", ErrAttentionConflict)
}

func validateAttentionIdentity(s RuntimeSession, expected AttentionResponseExpectation) error {
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
	return nil
}

// ClaimAttentionResponse is cross-process serialized by Registry.saveLocked.
// Only one HTTP client may submit a revision. An uncertain transport delivery
// deliberately retains the claim: a retry could type into the next question.
func (r *Registry) ClaimAttentionResponse(runtimeID string, expected AttentionResponseExpectation) error {
	return r.claimAttention(runtimeID, expected, "", false)
}

// ClaimAttentionChoice atomically consumes a structured option selection.
func (r *Registry) ClaimAttentionChoice(runtimeID string, expected AttentionResponseExpectation, optionID string) error {
	return r.claimAttention(runtimeID, expected, optionID, true)
}

func (r *Registry) claimAttention(runtimeID string, expected AttentionResponseExpectation, optionID string, choice bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var conflict error
	err := r.saveLocked(func(fresh map[string]RuntimeSession) {
		s, ok := fresh[runtimeID]
		if !ok {
			conflict = fmt.Errorf("%w: runtime not found", ErrAttentionConflict)
			return
		}
		if choice {
			conflict = ValidateAttentionChoice(s, expected, optionID)
		} else {
			conflict = ValidateAttentionResponse(s, expected)
		}
		if conflict != nil {
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
