package quota

import (
	"testing"

	"github.com/IAPro-Community/nexus/internal/core/model"
)

func TestNormalizePercentEvidenceUsedToRemaining(t *testing.T) {
	tests := []struct {
		name      string
		value     float64
		remaining float64
		used      float64
	}{
		{name: "codex unused is fully available", value: 0, remaining: 100, used: 0},
		{name: "partial used", value: 30, remaining: 70, used: 30},
		{name: "fully used is exhausted", value: 100, remaining: 0, used: 100},
		{name: "provider overflow clamps", value: 120, remaining: 0, used: 100},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			remaining, used, ok := NormalizePercentEvidence(tc.value, model.UsageSemanticsUsedPercent)
			if !ok || remaining != tc.remaining || used != tc.used {
				t.Fatalf("NormalizePercentEvidence(%v USED_PERCENT)=(%v,%v,%v), want (%v,%v,true)", tc.value, remaining, used, ok, tc.remaining, tc.used)
			}
		})
	}
}

func TestNormalizePercentEvidenceRemainingStaysRemaining(t *testing.T) {
	remaining, used, ok := NormalizePercentEvidence(100, model.UsageSemanticsRemainingPercent)
	if !ok || remaining != 100 || used != 0 {
		t.Fatalf("100%% remaining must stay 100%% remaining, got remaining=%v used=%v ok=%v", remaining, used, ok)
	}

	remaining, used, ok = NormalizePercentEvidence(40, model.UsageSemanticsRemainingPercent)
	if !ok || remaining != 40 || used != 60 {
		t.Fatalf("40%% remaining must normalize to remaining=40 used=60, got remaining=%v used=%v ok=%v", remaining, used, ok)
	}
}

func TestNormalizePercentEvidenceRejectsNonPercentMetrics(t *testing.T) {
	if _, _, ok := NormalizePercentEvidence(42, model.UsageSemanticsUsedAbsolute); ok {
		t.Fatal("absolute evidence must not be silently interpreted as a percentage")
	}
}
