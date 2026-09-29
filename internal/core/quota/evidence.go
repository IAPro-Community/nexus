package quota

import "github.com/IAPro-Community/nexus/internal/core/model"

// NormalizePercentEvidence converts provider-native percent semantics into the
// canonical pair used by Nexus: remaining and used percentages. This is the
// single arithmetic boundary for providers that report either "used" or
// "remaining" percentages.
func NormalizePercentEvidence(value float64, semantics model.UsageSemantics) (remaining, used float64, ok bool) {
	value = clampEvidencePercent(value)
	switch semantics {
	case model.UsageSemanticsUsedPercent:
		used = value
		remaining = 100.0 - value
		return remaining, used, true
	case model.UsageSemanticsRemainingPercent:
		remaining = value
		used = 100.0 - value
		return remaining, used, true
	default:
		return 0, 0, false
	}
}

func clampEvidencePercent(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}
