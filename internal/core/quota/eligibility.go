package quota

import (
	"strings"

	"github.com/kivervinicius/ai-cli/internal/core/model"
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
	Eligible   bool              `json:"eligible"`
	Reason     EligibilityReason `json:"reason"`
	Confidence string            `json:"confidence"`
	Message    string            `json:"message,omitempty"`
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
	confidence := confidenceOf(in.View.Status)
	out := Eligibility{Confidence: confidence}

	if !in.Authenticated {
		out.Reason = ReasonNotAuthenticated
		out.Message = "Conta não autenticada"
		return out
	}
	if in.NetworkError {
		out.Reason = ReasonNetworkError
		out.Message = "Erro de rede ao consultar o provedor"
		return out
	}

	health := strings.ToLower(strings.TrimSpace(in.Health))
	switch health {
	case strings.ToLower(string(model.HealthAuthRequired)), "auth_required":
		out.Reason = ReasonAuthRequired
		out.Message = "Autenticação necessária"
		return out
	case strings.ToLower(string(model.HealthUnavailable)), "unavailable", "unhealthy":
		out.Reason = ReasonUnavailable
		out.Message = "Provedor indisponível"
		return out
	}

	if in.InCooldown {
		out.Reason = ReasonCooldown
		out.Message = "Conta em cooldown de rate limit"
		return out
	}

	status := strings.ToUpper(strings.TrimSpace(in.View.Status))
	if status == string(model.UsageRateLimited) || in.View.AvailReasons.RateLimited {
		out.Reason = ReasonRateLimited
		out.Confidence = "RATE_LIMITED"
		out.Message = "Conta rate-limited"
		return out
	}

	// Availability is evaluated before UNKNOWN so an explicitly unavailable
	// account is rejected as exhausted/unavailable instead of "quota unknown".
	if !in.View.Available || len(in.View.AvailReasons.ExhaustedWindows) > 0 {
		if status == string(model.UsageUnknown) || in.View.AvailReasons.UnknownQuota {
			out.Reason = ReasonUnknownQuota
			out.Confidence = "UNKNOWN"
			out.Message = "Quota UNKNOWN — inelegível para alocação automática"
			if in.ExplicitSelection {
				// Still reject when Available=false; explicit only relaxes
				// UNKNOWN-but-available accounts.
				if in.View.Available {
					out.Eligible = true
					out.Message = "Quota não aferida; seleção explícita permitida"
				}
			}
			return out
		}
		out.Reason = ReasonExhausted
		out.Message = "resource unavailable"
		return out
	}

	if status == string(model.UsageUnknown) || in.View.AvailReasons.UnknownQuota {
		out.Reason = ReasonUnknownQuota
		out.Confidence = "UNKNOWN"
		out.Message = "Quota UNKNOWN — inelegível para alocação automática"
		if in.ExplicitSelection {
			out.Eligible = true
			out.Message = "Quota não aferida; seleção explícita permitida"
		}
		return out
	}
	if status == string(model.UsageUnsupported) {
		out.Reason = ReasonUnsupportedQuota
		out.Confidence = "UNSUPPORTED"
		out.Message = "Quota não suportada pelo provedor"
		if in.ExplicitSelection {
			out.Eligible = true
		}
		return out
	}

	out.Eligible = true
	out.Reason = ReasonOK
	return out
}

func confidenceOf(status string) string {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case string(model.UsageLive):
		return "LIVE"
	case string(model.UsageCached):
		return "CACHED"
	case string(model.UsageEstimated):
		return "ESTIMATED"
	case string(model.UsageUnsupported):
		return "UNSUPPORTED"
	case string(model.UsageRateLimited):
		return "RATE_LIMITED"
	default:
		return "UNKNOWN"
	}
}
