package nexus

import "strings"

type NexSpendPolicy struct {
	DailyCapMicroUSD  int64 `json:"daily_cap_microusd"`
	AllowPaidFallback bool  `json:"allow_paid_fallback"`
}

type NexSpendDecision struct {
	Allowed bool   `json:"allowed"`
	Code    string `json:"code"`
	Reason  string `json:"reason,omitempty"`
}

// EvaluateNexSpendGuard enforces fail-closed spend limits and denies silent
// automatic paid model fallback unless explicitly enabled by user policy.
func EvaluateNexSpendGuard(cost *HUDCostSummary, policy NexSpendPolicy, usingFallback bool) NexSpendDecision {
	if usingFallback && !policy.AllowPaidFallback {
		return NexSpendDecision{
			Allowed: false,
			Code:    "paid_fallback_denied",
			Reason:  "Automatic paid model fallback is disabled by policy.",
		}
	}
	if cost != nil {
		if strings.EqualFold(string(cost.BudgetLevel), "EXCEEDED") {
			return NexSpendDecision{
				Allowed: false,
				Code:    "spend_cap_exceeded",
				Reason:  "Configured spend budget is exceeded.",
			}
		}
		if policy.DailyCapMicroUSD > 0 && cost.KnownMicroUSD >= policy.DailyCapMicroUSD {
			return NexSpendDecision{
				Allowed: false,
				Code:    "spend_cap_exceeded",
				Reason:  "Daily Nex spend cap reached.",
			}
		}
	}
	return NexSpendDecision{
		Allowed: true,
		Code:    "allowed",
	}
}
