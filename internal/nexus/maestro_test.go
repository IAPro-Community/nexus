package nexus

import (
	"context"
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

func TestMaestroVerifyContractPassesThroughMachineReadableVerdict(t *testing.T) {
	script := `if [ "$1" = "verify" ] && [ "$2" = "--json" ]; then
  cat >/dev/null
  printf '%s\n' '{"version":"1.0.0","lifecycle":"VERIFY","passed":true,"required_actions":[],"explanation":"all required checks passed"}'
  exit 0
fi
exit 2
`
	bin := testutil.WriteFakeBinaryWithScript(t, t.TempDir(), "maestro", script)
	c := &MaestroClient{
		status: MaestroStatus{
			Available: true,
			Mode:      MaestroAssist,
			Capabilities: &MaestroCapability{
				Version: MaestroVersion,
			},
		},
		maestroBin: bin,
	}
	resp, err := c.Verify(context.Background(), MaestroVerifyRequest{
		ProjectID:    "project-1",
		MissionID:    "run-1",
		ChangedPaths: []string{"internal/nexus/maestro.go"},
		Checks:       []string{"go test ./..."},
		EvidenceRefs: []string{"evidence://run-1/tests"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Passed || resp.Degraded || resp.Lifecycle != MaestroLifecycleVerify {
		t.Fatalf("unexpected verify response: %+v", resp)
	}
}

func TestMaestroVerifyUnavailableNeverFabricatesPass(t *testing.T) {
	c := &MaestroClient{status: MaestroStatus{Available: false, Mode: MaestroOff}}
	resp, err := c.Verify(context.Background(), MaestroVerifyRequest{
		ProjectID: "project-1",
		MissionID: "run-1",
	})
	if err == nil {
		t.Fatal("expected degraded verify error")
	}
	if resp.Passed || !resp.Degraded {
		t.Fatalf("unavailable Maestro must fail closed: %+v", resp)
	}
}

func TestMaestroVerifyRejectsSecretLikeInputBeforeExecution(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "called")
	script := "touch " + marker + "\nexit 0\n"
	bin := testutil.WriteFakeBinaryWithScript(t, t.TempDir(), "maestro", script)
	c := &MaestroClient{
		status: MaestroStatus{Available: true, Mode: MaestroAssist},
		maestroBin: bin,
	}
	resp, err := c.Verify(context.Background(), MaestroVerifyRequest{
		ProjectID:    "project-1",
		EvidenceRefs: []string{"token=super-secret"},
	})
	if err == nil {
		t.Fatal("secret-like verify input must be rejected")
	}
	if resp == nil || resp.Passed || !resp.Degraded {
		t.Fatalf("secret rejection must fail closed: %+v", resp)
	}
	if _, statErr := os.Stat(marker); !os.IsNotExist(statErr) {
		t.Fatalf("Maestro process was invoked for rejected input: %v", statErr)
	}
}

