package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func installTestMemory(value string) IntelligenceMemory {
	return IntelligenceMemory{
		ID: "imem_test", Key: "quality.testing", Value: value,
		Source: "explicit", UpdatedAt: "2026-10-29T12:00:00Z",
	}
}

func prepareInstallConfig(t *testing.T, old []IntelligenceMemory) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("NEXUS_CONFIG_DIR", dir)
	cfg := NewDefaultConfig()
	cfg.Language = "es"
	cfg.Defaults["codex"] = "work"
	cfg.IntelligenceMemories = old
	if err := SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}
	return filepath.Join(dir, "config.json")
}

func TestInstallMemoryCASPreservesUnrelatedAndUnknownConfigSettings(t *testing.T) {
	before := []IntelligenceMemory{installTestMemory("go test ./...")}
	configPath := prepareInstallConfig(t, before)
	original, err := os.ReadFile(configPath)
	if err != nil { t.Fatal(err) }
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(original, &raw); err != nil { t.Fatal(err) }
	raw["future_feature_options"] = json.RawMessage(`{"enabled":true,"unknown_nested":["x",3]}`)
	input, err := json.Marshal(raw)
	if err != nil { t.Fatal(err) }
	if err := os.WriteFile(configPath, input, 0600); err != nil { t.Fatal(err) }

	replacement := []IntelligenceMemory{installTestMemory("go test -race ./...")}
	receipt, err := ReplaceInstallIntelligenceMemoriesIfUnchanged(before, replacement)
	if err != nil { t.Fatal(err) }
	if len(receipt.SHA256) != 64 || receipt.Bytes < 1 {
		t.Fatalf("missing post-rename checksum: %+v", receipt)
	}
	after, err := os.ReadFile(configPath)
	if err != nil { t.Fatal(err) }
	var output map[string]json.RawMessage
	if err := json.Unmarshal(after, &output); err != nil { t.Fatal(err) }
	for name, old := range raw {
		if name == "intelligence_memories" { continue }
		var x, y any
		if err := json.Unmarshal(old, &x); err != nil { t.Fatal(err) }
		if err := json.Unmarshal(output[name], &y); err != nil { t.Fatal(err) }
		ax, _ := json.Marshal(x)
		ay, _ := json.Marshal(y)
		if string(ax) != string(ay) {
			t.Fatalf("unrelated config setting %q was modified: %s vs %s", name, ax, ay)
		}
	}
	cfg, err := LoadConfig()
	if err != nil { t.Fatal(err) }
	if !sameInstallMemorySet(cfg.IntelligenceMemories, replacement) ||
		cfg.Language != "es" || cfg.Defaults["codex"] != "work" {
		t.Fatalf("install CAS lost normal config fields: %+v", cfg)
	}
}

func TestInstallMemoryCASRejectsStaleSnapshotWithoutWriting(t *testing.T) {
	old := []IntelligenceMemory{installTestMemory("old")}
	path := prepareInstallConfig(t, old)
	latest := []IntelligenceMemory{installTestMemory("changed by user")}
	if _, err := ReplaceInstallIntelligenceMemoriesIfUnchanged(old, latest); err != nil {
		t.Fatal(err)
	}
	start, err := os.ReadFile(path)
	if err != nil { t.Fatal(err) }
	if _, err := ReplaceInstallIntelligenceMemoriesIfUnchanged(old, []IntelligenceMemory{installTestMemory("stale restore")}); err == nil ||
		!strings.Contains(err.Error(), "CAS precondition failed") {
		t.Fatalf("stale INSTALL restore was accepted: %v", err)
	}
	end, err := os.ReadFile(path)
	if err != nil { t.Fatal(err) }
	if string(start) != string(end) {
		t.Fatal("failed CAS modified config.json")
	}
}

func TestInstallMemoryCASOnlyOneConcurrentInProcessWriterWins(t *testing.T) {
	original := []IntelligenceMemory{installTestMemory("starting")}
	prepareInstallConfig(t, original)
	var wg sync.WaitGroup
	outcomes := make(chan error, 2)
	for _, value := range []string{"new-one", "new-two"} {
		wg.Add(1)
		go func(value string) {
			defer wg.Done()
			_, err := ReplaceInstallIntelligenceMemoriesIfUnchanged(original,
				[]IntelligenceMemory{installTestMemory(value)})
			outcomes <- err
		}(value)
	}
	wg.Wait()
	close(outcomes)
	success, failure := 0, 0
	for err := range outcomes {
		if err == nil { success++ } else if strings.Contains(err.Error(), "CAS precondition failed") {
			failure++
		} else { t.Fatalf("unexpected concurrency failure: %v", err) }
	}
	if success != 1 || failure != 1 {
		t.Fatalf("expected one winner and one stale retry: success=%d fail=%d", success, failure)
	}
}

func TestInstallMemoryCASRejectsMissingConfigSymlinksAndMalformedData(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NEXUS_CONFIG_DIR", dir)
	if _, err := ReplaceInstallIntelligenceMemoriesIfUnchanged(nil,
		[]IntelligenceMemory{installTestMemory("new")}); err == nil {
		t.Fatal("CAS unexpectedly created a missing config.json")
	}
	target := filepath.Join(dir, "target.json")
	if err := os.WriteFile(target, []byte(`{"intelligence_memories":[]}`), 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "config.json")
	if err := os.Symlink(target, link); err == nil {
		if _, err := ReplaceInstallIntelligenceMemoriesIfUnchanged(nil, nil); err == nil {
			t.Fatal("CAS followed config.json symlink")
		}
		if err := os.Remove(link); err != nil { t.Fatal(err) }
	}
	if err := os.WriteFile(link, []byte("{"), 0600); err != nil { t.Fatal(err) }
	if _, err := ReplaceInstallIntelligenceMemoriesIfUnchanged(nil, nil); err == nil {
		t.Fatal("CAS accepted malformed config JSON")
	}
}

func TestInstallMemoryCASRejectsDuplicateKeysIDsAndMissingValues(t *testing.T) {
	old := []IntelligenceMemory{installTestMemory("original")}
	path := prepareInstallConfig(t, old)
	before, err := os.ReadFile(path)
	if err != nil { t.Fatal(err) }
	cases := [][]IntelligenceMemory{
		{old[0], old[0]},
		{old[0], {ID: "other", Key: old[0].Key, Value: "x"}},
		{{ID: "", Key: "quality.testing", Value: "x"}},
		{{ID: "imem_test", Key: "quality.testing", Value: ""}},
	}
	for _, replacement := range cases {
		if _, err := ReplaceInstallIntelligenceMemoriesIfUnchanged(old, replacement); err == nil {
			t.Fatal("invalid memory replacement accepted")
		}
	}
	after, err := os.ReadFile(path)
	if err != nil { t.Fatal(err) }
	if string(before) != string(after) {
		t.Fatal("invalid memory replacement changed configuration")
	}
}

func TestInstallMemoryCASEmptyReplacement(t *testing.T) {
	old := []IntelligenceMemory{installTestMemory("original")}
	configPath := prepareInstallConfig(t, old)
	if _, err := ReplaceInstallIntelligenceMemoriesIfUnchanged(old, nil); err != nil {
		t.Fatal(err)
	}
	bytes, err := os.ReadFile(configPath)
	if err != nil { t.Fatal(err) }
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(bytes, &raw); err != nil { t.Fatal(err) }
	if string(raw["intelligence_memories"]) != "[]" {
		t.Fatalf("empty restore did not persist an empty array: %s", raw["intelligence_memories"])
	}
}
