package spec

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCanonicalQuotaFixtures(t *testing.T) {
	t.Run("agy tsv", func(t *testing.T) {
		raw := readFixture(t, "quota/agy/v1.tsv")
		if !strings.Contains(string(raw), "Weekly Limit Remaining") || !strings.Contains(string(raw), "Five Hour Limit Remaining") {
			t.Fatalf("agy fixture missing window labels:\n%s", raw)
		}
	})
	t.Run("claude oauth usage", func(t *testing.T) {
		var payload map[string]any
		if err := json.Unmarshal(readFixture(t, "quota/claude/oauth_usage_v1.json"), &payload); err != nil {
			t.Fatal(err)
		}
		for _, key := range []string{"five_hour", "seven_day", "seven_day_sonnet"} {
			period, ok := payload[key].(map[string]any)
			if !ok {
				t.Fatalf("claude fixture missing %s object", key)
			}
			if _, ok := period["utilization"]; !ok {
				t.Fatalf("claude %s missing utilization", key)
			}
		}
		assertByteEqual(t, "quota/claude/oauth_usage_v1.json", filepath.Join("..", "adapters", "claude", "testdata", "oauth_usage_v1.json"))
	})
	t.Run("codex rateLimits/read", func(t *testing.T) {
		var payload struct {
			RateLimits struct {
				Primary struct {
					UsedPercent float64 `json:"usedPercent"`
				} `json:"primary"`
			} `json:"rateLimits"`
		}
		if err := json.Unmarshal(readFixture(t, "quota/codex/rate_limits_read_v1.json"), &payload); err != nil {
			t.Fatal(err)
		}
		if payload.RateLimits.Primary.UsedPercent <= 0 {
			t.Fatalf("codex fixture usedPercent=%v", payload.RateLimits.Primary.UsedPercent)
		}
	})
}

func readFixture(t *testing.T, rel string) []byte {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", rel))
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func assertByteEqual(t *testing.T, canonicalRel, adapterRel string) {
	t.Helper()
	canonical, err := os.ReadFile(filepath.Join("testdata", canonicalRel))
	if err != nil {
		t.Fatal(err)
	}
	adapter, err := os.ReadFile(adapterRel)
	if err != nil {
		t.Fatal(err)
	}
	if string(canonical) != string(adapter) {
		t.Fatalf("%s drifted from canonical testdata/%s", adapterRel, canonicalRel)
	}
}
