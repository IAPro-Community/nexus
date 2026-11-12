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

// AttentionOperation distinguishes terminal responses from provider choices.
type AttentionOperation string

const (
	AttentionOperationResponse AttentionOperation = "response"
	AttentionOperationChoice   AttentionOperation = "choice"
)

// AttentionSubmissionStatus records transport delivery, not provider settlement.
type AttentionSubmissionStatus string

const (
	AttentionSubmissionPending   AttentionSubmissionStatus = "PENDING"
	AttentionSubmissionSubmitted AttentionSubmissionStatus = "SUBMITTED"
	AttentionSubmissionUncertain AttentionSubmissionStatus = "UNCERTAIN"
)

// AttentionSubmission stores only a digest of the operation, never its input.
// It stays in the private registry so a reconnect or restart cannot resend it.
type AttentionSubmission struct {
	Expectation AttentionResponseExpectation `json:"expectation"`
	Operation   AttentionOperation           `json:"operation"`
	Digest      string                       `json:"digest"`
	Status      AttentionSubmissionStatus    `json:"status"`
}

// AttentionSubmissionClaim grants transport dispatch only to the first caller.
type AttentionSubmissionClaim struct {
	Dispatch bool
	Status   AttentionSubmissionStatus
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
	if err := validateAttentionEpisode(s, expected); err != nil {
		return err
	}
	if s.AttentionSubmittedRevision == s.AttentionRevision {
		return conflict("response was already submitted")
	}
	if s.State != StateWaiting && s.State != StateApproval {
		return conflict("runtime is no longer waiting")
	}
	return nil
}

func validateAttentionEpisode(s RuntimeSession, expected AttentionResponseExpectation) error {
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
	return nil
}

// ClaimAttentionResponse is cross-process serialized by Registry.saveLocked.
// Only one HTTP client may submit a revision. An uncertain transport delivery
// deliberately retains the claim: a retry could type into the next question.
func (r *Registry) ClaimAttentionResponse(runtimeID string, expected AttentionResponseExpectation) error {
	_, err := r.claimAttention(runtimeID, expected, "", AttentionOperationResponse, "")
	return err
}

// ClaimAttentionChoice atomically consumes a structured option selection.
func (r *Registry) ClaimAttentionChoice(runtimeID string, expected AttentionResponseExpectation, optionID string) error {
	_, err := r.claimAttention(runtimeID, expected, optionID, AttentionOperationChoice, "")
	return err
}

// ClaimAttentionOperation reserves a revision or returns its persisted outcome
// for the exact same operation. Only a new claim may open the input transport.
func (r *Registry) ClaimAttentionOperation(runtimeID string, expected AttentionResponseExpectation, operation AttentionOperation, digest, optionID string) (AttentionSubmissionClaim, error) {
	if len(digest) != 64 {
		return AttentionSubmissionClaim{}, fmt.Errorf("%w: invalid operation digest", ErrAttentionConflict)
	}
	return r.claimAttention(runtimeID, expected, optionID, operation, digest)
}

func (r *Registry) claimAttention(runtimeID string, expected AttentionResponseExpectation, optionID string, operation AttentionOperation, digest string) (AttentionSubmissionClaim, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var claim AttentionSubmissionClaim
	var conflict error
	err := r.saveLocked(func(fresh map[string]RuntimeSession) {
		s, ok := fresh[runtimeID]
		if !ok {
			conflict = fmt.Errorf("%w: runtime not found", ErrAttentionConflict)
			return
		}
		if digest != "" && s.AttentionSubmittedRevision == expected.Revision {
			previous := s.AttentionSubmission
			if s.AttentionRevision != expected.Revision ||
				fmt.Sprintf("%d:%d", s.HostGeneration, s.ProviderGeneration) != expected.Generation ||
				previous == nil || previous.Expectation != expected || previous.Operation != operation || previous.Digest != digest {
				conflict = fmt.Errorf("%w: revision was claimed by a different operation", ErrAttentionConflict)
				return
			}
			claim.Status = previous.Status
			return
		}
		switch operation {
		case AttentionOperationChoice:
			conflict = ValidateAttentionChoice(s, expected, optionID)
		case AttentionOperationResponse:
			conflict = ValidateAttentionResponse(s, expected)
		default:
			conflict = fmt.Errorf("%w: unknown operation", ErrAttentionConflict)
		}
		if conflict != nil {
			return
		}
		s.AttentionSubmittedRevision = expected.Revision
		if digest != "" {
			s.AttentionSubmission = &AttentionSubmission{
				Expectation: expected, Operation: operation, Digest: digest, Status: AttentionSubmissionPending,
			}
		} else {
			s.AttentionSubmission = nil
		}
		s.UpdatedAt = time.Now()
		fresh[runtimeID] = s
		claim = AttentionSubmissionClaim{Dispatch: true, Status: AttentionSubmissionPending}
	})
	if err != nil {
		return AttentionSubmissionClaim{}, err
	}
	return claim, conflict
}

// CompleteAttentionOperation persists the transport outcome without changing
// the provider's attention state. A later episode cannot be overwritten.
func (r *Registry) CompleteAttentionOperation(runtimeID string, expected AttentionResponseExpectation, digest string, status AttentionSubmissionStatus) error {
	if status != AttentionSubmissionSubmitted && status != AttentionSubmissionUncertain {
		return errors.New("invalid attention delivery outcome")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	var conflict error
	err := r.saveLocked(func(fresh map[string]RuntimeSession) {
		s, ok := fresh[runtimeID]
		if !ok || s.AttentionSubmission == nil || s.AttentionSubmission.Expectation != expected || s.AttentionSubmission.Digest != digest {
			conflict = fmt.Errorf("%w: operation changed before delivery was recorded", ErrAttentionConflict)
			return
		}
		submission := *s.AttentionSubmission
		if submission.Status != AttentionSubmissionPending {
			if submission.Status != status {
				conflict = fmt.Errorf("%w: delivery outcome is already final", ErrAttentionConflict)
			}
			return
		}
		submission.Status = status
		s.AttentionSubmission = &submission
		s.UpdatedAt = time.Now()
		fresh[runtimeID] = s
	})
	if err != nil {
		return err
	}
	return conflict
}
