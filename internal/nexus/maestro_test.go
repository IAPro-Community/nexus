package nexus

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/IAPro-Community/nexus/internal/testutil"
)

func TestMaestroCapabilitiesMergeProfileAndCanonicalCatalog(t *testing.T) {
	global := t.TempDir()
	profile := t.TempDir()
	manifest := func(dir string, ids ...string) {
		skills := map[string]map[string]any{}
		for _, id := range ids {
			skills[id] = map[string]any{"name": id, "description": id + " description"}
			if err := os.MkdirAll(filepath.Join(dir, "skills", id), 0700); err != nil {
				t.Fatal(err)
			}
			prompt := "# " + id + "\n\nUse this skill with its complete operating contract."
			if err := os.WriteFile(filepath.Join(dir, "skills", id, "SKILL.md"), []byte(prompt), 0600); err != nil {
				t.Fatal(err)
			}
		}
		data, err := json.Marshal(map[string]any{"skills": skills})
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "SKILLS_MANIFEST.json"), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	manifest(global, "skill-database-migrations", "skill-global-only")
	manifest(profile, "skill-database-migrations", "skill-profile-only")
	client := &MaestroClient{}
	cap, err := client.queryCapabilitiesFromDirs([]string{profile, global})
	if err != nil {
		t.Fatal(err)
	}
	if got := len(cap.Skills); got != 3 {
		t.Fatalf("merged catalog count = %d, want 3", got)
	}
	if cap.Skills[0].ID != "skill-database-migrations" {
		t.Fatalf("skills must be sorted by id, got %+v", cap.Skills)
	}
	if got := cap.Skills[0].Prompt; got == "" || got != "# skill-database-migrations\n\nUse this skill with its complete operating contract." {
		t.Fatalf("merged skill must carry its SKILL.md prompt, got %q", got)
	}
}

func TestMaestroUnavailableNeverFabricatesAdvice(t *testing.T) {
	c := &MaestroClient{status: MaestroStatus{Available: false, Mode: MaestroOff}, maestroBin: ""}
	resp, err := c.GetAdvice(AdviceContext{ProjectID: "p1", Lifecycle: MaestroLifecycleRecovery}, "ship product")
	if err == nil {
		t.Fatal("expected degraded error")
	}
	if !resp.Degraded || len(resp.Required) != 0 || len(resp.Recommended) != 0 || len(resp.Optional) != 0 {
		t.Fatalf("degraded Maestro must not fabricate recommendations: %+v", resp)
	}
	if resp.Lifecycle != MaestroLifecycleRecovery {
		t.Fatalf("lifecycle was not preserved in degraded advice: %+v", resp)
	}
}

func TestMaestroAdviseFailureNeverFallsBackToSyntheticRecommendations(t *testing.T) {
	script := "if [ \"$1\" = \"version\" ] || [ \"$1\" = \"--version\" ]; then echo 1.2.3; exit 0; fi\nexit 2\n"
	bin := testutil.WriteFakeBinaryWithScript(t, t.TempDir(), "maestro", script)
	c := &MaestroClient{status: MaestroStatus{Available: true, Mode: MaestroAssist, Capabilities: &MaestroCapability{Version: "1.2.3"}}, maestroBin: bin}
	resp, err := c.GetAdvice(AdviceContext{ProjectID: "p1"}, "ship product")
	if err == nil {
		t.Fatal("expected advise contract failure")
	}
	if !resp.Degraded || len(resp.Required)+len(resp.Recommended)+len(resp.Optional) != 0 {
		t.Fatalf("expected empty degraded response, got %+v", resp)
	}
}

func TestFindOrquestradorDirUsesPortableDataDirProfiles(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	data := t.TempDir()
	t.Setenv("NEXUS_DATA_DIR", data)
	expected := filepath.Join(data, "profiles", "claude", "work", "home", ".orquestrador")
	if err := os.MkdirAll(expected, 0700); err != nil {
		t.Fatal(err)
	}
	if got := findOrquestradorDir(); got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}
}
