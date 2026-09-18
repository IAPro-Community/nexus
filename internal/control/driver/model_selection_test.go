package driver

import (
	"testing"

	"github.com/IAPro-Community/nexus/internal/core/model"
)

func TestResolveLaunchModelPreservesExplicitModel(t *testing.T) {
	got := ResolveLaunchModel("agy", "gpt-custom", model.UsageSnapshot{})
	if got != "gpt-custom" {
		t.Fatalf("explicit model=%q, want gpt-custom", got)
	}
}

func TestResolveLaunchModelPrefersAvailableGeminiPool(t *testing.T) {
	remaining := 42.0
	usage := model.UsageSnapshot{Windows: []model.UsageWindow{
		{Group: "gemini", Kind: "weekly", RemainingPercent: &remaining},
		{Group: "claude_gpt", Kind: "weekly", RemainingPercent: &remaining},
	}}
	if got := ResolveLaunchModel("agy", "", usage); got != DefaultAGYGeminiModel {
		t.Fatalf("model=%q, want %q", got, DefaultAGYGeminiModel)
	}
}

func TestResolveLaunchModelFallsBackToClaudeWhenGeminiExhausted(t *testing.T) {
	zero, full := 0.0, 100.0
	usage := model.UsageSnapshot{Windows: []model.UsageWindow{
		{Group: "gemini", Kind: "weekly", RemainingPercent: &zero},
		{Group: "claude_gpt", Kind: "weekly", RemainingPercent: &full},
	}}
	if got := ResolveLaunchModel("agy", "", usage); got != DefaultAGYClaudeModel {
		t.Fatalf("model=%q, want %q", got, DefaultAGYClaudeModel)
	}
}

func TestResolveLaunchModelDefaultsToGeminiWhenQuotaIsUnknown(t *testing.T) {
	if got := ResolveLaunchModel("agy", "", model.UsageSnapshot{}); got != DefaultAGYGeminiModel {
		t.Fatalf("model=%q, want %q", got, DefaultAGYGeminiModel)
	}
}
