package quota

import (
	"strings"
	"time"

	sdkquota "github.com/IAPro-Community/IAPro-SDK/quota"
	"github.com/IAPro-Community/nexus/internal/core/model"
)

// EligibilityReason is a stable machine-readable gate outcome.
type EligibilityReason string

const (
	ReasonOK               EligibilityReason = "OK"
	ReasonUnknownQuota     EligibilityReason = "UNKNOWN_QUOTA"
	ReasonUnsupportedQuota EligibilityReason = "UNSUPPORTED_QUOTA"
	ReasonExhausted        EligibilityReason = "EXHAUSTED"
	ReasonRateLimited      EligibilityReason = "RATE_LIMITED"
	ReasonAuthRequired     EligibilityReason = "AUTH_REQUIRED"
	ReasonUnavailable      EligibilityReason = "UNAVAILABLE"
	ReasonNetworkError     EligibilityReason = "NETWORK_ERROR"
	ReasonCooldown         EligibilityReason = "COOLDOWN"
	ReasonNotAuthenticated EligibilityReason = "NOT_AUTHENTICATED"
)

// Eligibility is the single routing gate shared by CLI scheduler, Mission
// RecommendResources, runtime routing and manual validation.
type Eligibility struct {
	Eligible         bool              `json:"eligible"`
	Reason           EligibilityReason `json:"reason"`
	Confidence       string            `json:"confidence"`
	CapacityEvidence string            `json:"capacity_evidence"` // MEASURED | OPAQUE | UNKNOWN
	Message          string            `json:"message,omitempty"`
}

// EligibilityInput carries the facts needed to evaluate a provider account.
type EligibilityInput struct {
	View              QuotaView
	Authenticated     bool
	Health            string
	NetworkError      bool
	InCooldown        bool
	ExplicitSelection bool
}

// EvaluateEligibility applies hard gates in a single canonical order.
func EvaluateEligibility(in EligibilityInput) Eligibility {
	// The shared SDK gate owns authentication, provider health, cooldown and
	// quota eligibility. Probe its operational gates first with a neutral quota
	// view so Nexus can preserve its established precedence: a stale quota must
	// not hide a current auth/network/provider failure.
	preflight := evaluateSharedEligibility(in, sdkquota.QuotaView{
		Status:    string(model.UsageLive),
		Available: true,
	})
	switch preflight.Reason {
	case sdkquota.ReasonNotAuthenticated, sdkquota.ReasonNetworkError,
		sdkquota.ReasonAuthRequired, sdkquota.ReasonRateLimited,
		sdkquota.ReasonUnavailable, sdkquota.ReasonCooldown:
		return adaptSharedEligibility(in, preflight)
	}

	// Display history may outlive its routing validity by weeks. Only recent,
	// attributable observations can drive automatic capacity decisions. A
	// stale RATE_LIMITED signal must likewise not block an account forever.
	// Existing synthetic/legacy views without FetchedAt remain handled by
	// their status gates below, without inventing a new timestamp.
	if !in.View.FetchedAt.IsZero() && !strings.EqualFold(in.View.Status, string(model.UsageUnsupported)) {
		age := time.Since(in.View.FetchedAt)
		if age < -time.Minute || age > DefaultTTL {
			return Eligibility{
				Reason:           ReasonUnknownQuota,
				Confidence:       "UNKNOWN",
				CapacityEvidence: "UNKNOWN",
				Message:          "Quota desatualizada; atualize antes de selecionar automaticamente",
			}
		}
	}

	view := sdkView(in.View)
	if strings.EqualFold(in.View.Status, string(model.UsageUnsupported)) {
		// UNSUPPORTED is opaque evidence, not exhaustion. Normalize the view so
		// the SDK returns its dedicated unsupported reason; its gate then permits
		// only explicit selection and never treats the missing quota as measured.
		view.Available = true
		view.AvailReasons.ExhaustedWindows = nil
		view.AvailReasons.UnknownQuota = false
		view.AvailReasons.AllOK = false
	}
	return adaptSharedEligibility(in, sdkquota.EvaluateEligibility(sdkquota.EligibilityInput{
		View:              view,
		Authenticated:     in.Authenticated,
		Health:            in.Health,
		NetworkError:      in.NetworkError,
		InCooldown:        in.InCooldown,
		ExplicitSelection: in.ExplicitSelection,
	}))
}

func evaluateSharedEligibility(in EligibilityInput, view sdkquota.QuotaView) sdkquota.Eligibility {
	return sdkquota.EvaluateEligibility(sdkquota.EligibilityInput{
		View:              view,
		Authenticated:     in.Authenticated,
		Health:            in.Health,
		NetworkError:      in.NetworkError,
		InCooldown:        in.InCooldown,
		ExplicitSelection: in.ExplicitSelection,
	})
}

func adaptSharedEligibility(in EligibilityInput, shared sdkquota.Eligibility) Eligibility {
	return Eligibility{
		Eligible:         shared.Eligible,
		Reason:           EligibilityReason(shared.Reason),
		Confidence:       shared.Confidence,
		CapacityEvidence: capacityEvidenceOf(in.View.Status),
		Message:          shared.Message,
	}
}

func capacityEvidenceOf(status string) string {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case string(model.UsageLive), string(model.UsageCached), string(model.UsageRateLimited):
		return "MEASURED"
	case string(model.UsageUnsupported):
		return "OPAQUE"
	default:
		return "UNKNOWN"
	}
}
