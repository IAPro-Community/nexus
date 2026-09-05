package agy

import (
	"math"
	"os"
	"testing"
	"time"

	"github.com/kivervinicius/ai-cli/internal/core/model"
)

func TestAgyClampPercent(t *testing.T) {
	for _, tc := range []struct{ in, want float64 }{{0, 0}, {90.15, 90.15}, {100, 100}, {-5, 0}, {120, 100}} {
		if got := agyClampPercent(tc.in); math.Abs(got-tc.want) > 0.001 {
			t.Errorf("agyClampPercent(%v)=%v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestParseAgyQuotaOutputTreatsPercentAsRemaining(t *testing.T) {
	out := "Gemini Models\tWeekly Limit Remaining\t65.47%\t2026-09-11T04:31:30Z\n" +
		"Gemini Models\tFive Hour Limit Remaining\t0.00%\t2026-09-04T22:50:00Z\n" +
		"Claude and GPT models\tWeekly Limit Remaining\t100.00%\t\n" +
		"Claude and GPT models\tFive Hour Limit Remaining\t100.00%\tQuota available\n"
	windows, ok := parseAgyQuotaOutput(out)
	if !ok {
		t.Fatal("expected parsed windows")
	}
	if len(windows) != 4 {
		t.Fatalf("windows=%d want 4", len(windows))
	}

	byKind := map[string]float64{}
	for _, w := range windows {
		if w.RemainingPercent == nil {
			t.Fatalf("missing remaining for %s", w.Kind)
		}
		byKind[w.Kind] = *w.RemainingPercent
	}
	if byKind["5h"] != 0 {
		t.Fatalf("gemini 5h remaining=%v want 0", byKind["5h"])
	}
	if math.Abs(byKind["weekly"]-65.47) > 0.001 {
		t.Fatalf("gemini weekly remaining=%v want 65.47", byKind["weekly"])
	}
	if byKind["claude_5h"] != 100 {
		t.Fatalf("claude 5h remaining=%v want 100", byKind["claude_5h"])
	}
	if byKind["claude_weekly"] != 100 {
		t.Fatalf("claude weekly remaining=%v want 100", byKind["claude_weekly"])
	}
}

func TestParseAgyQuotaOutputRejectsIncompleteTSV(t *testing.T) {
	out := "Gemini Models\tWeekly Limit Remaining\t65.47%\t2026-09-11T04:31:30Z\n" +
		"Gemini Models\tFive Hour Limit Remaining\t0.00%\t2026-09-04T22:50:00Z\n"
	windows, ok := parseAgyQuotaOutput(out)
	if ok || windows != nil {
		t.Fatalf("incomplete TSV must be rejected, ok=%v windows=%d", ok, len(windows))
	}
}

func TestLegacyAGYQuotaKeepsClaudeAtZero(t *testing.T) {
	dataDir := t.TempDir()
	cfgDir := t.TempDir()
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_CONFIG_DIR", cfgDir)

	root := dataDir + "/profiles/agy/work"
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	content := `{
		"account": "user@example.com",
		"five_hour": { "percent_left": 50, "resets_in": "Refreshes in 1h" },
		"weekly": { "percent_left": 80, "resets_in": "Refreshes in 2d" },
		"claude_five_hour": { "percent_left": 0 },
		"claude_weekly": { "percent_left": 0 }
	}`
	if err := os.WriteFile(root+"/quota.json", []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	snap, ok := New().readCachedQuotaFiles(model.Profile{Provider: "agy", Name: "work"})
	if !ok {
		t.Fatal("expected legacy cache")
	}
	if len(snap.Windows) != 4 {
		t.Fatalf("windows=%d want 4 (Claude 0%% must remain)", len(snap.Windows))
	}
}

func TestFormatAgyResetRelative(t *testing.T) {
	future := time.Now().Add(90 * time.Minute).UTC().Format(time.RFC3339)
	got := formatAgyReset(future)
	if got != "Refreshes in 1h 30m" && got != "Refreshes in 1h 29m" && got != "Refreshes in 1h 31m" {
		t.Fatalf("relative reset=%q", got)
	}
	if formatAgyReset("Quota available") != "Quota available" {
		t.Fatal("passthrough reset text")
	}
}
