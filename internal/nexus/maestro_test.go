package nexus

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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
	bin := testutil.WriteFakeCommand(t, t.TempDir(), "maestro", []testutil.FakeCommandResponse{
		{Args: []string{"advise", "--json"}, ExitCode: 2},
	}, 2, "")
	c := &MaestroClient{status: MaestroStatus{Available: true, Mode: MaestroAssist, Capabilities: &MaestroCapability{Version: "1.2.3"}}, maestroBin: bin}
	resp, err := c.GetAdvice(AdviceContext{ProjectID: "p1"}, "ship product")
	if err == nil {
		t.Fatal("expected advise contract failure")
	}
	if !resp.Degraded || len(resp.Required)+len(resp.Recommended)+len(resp.Optional) != 0 {
		t.Fatalf("expected empty degraded response, got %+v", resp)
	}
}

func TestSyncScriptInvocationSupportsShellAndPowerShell(t *testing.T) {
	lookup := func(name string) (string, error) {
		switch name {
		case "bash":
			return "/tooling/bash", nil
		case "pwsh":
			return "/tooling/pwsh", nil
		default:
			return "", os.ErrNotExist
		}
	}

	sh := filepath.Join(t.TempDir(), "sync-skills.sh")
	binary, args, err := syncScriptInvocation(sh, "--dry-run", lookup)
	if err != nil {
		t.Fatal(err)
	}
	if binary != "/tooling/bash" || len(args) != 2 || args[0] != sh || args[1] != "--dry-run" {
		t.Fatalf("unexpected shell invocation: binary=%q args=%v", binary, args)
	}

	ps := filepath.Join(t.TempDir(), "sync-skills.ps1")
	binary, args, err = syncScriptInvocation(ps, "--apply", lookup)
	if err != nil {
		t.Fatal(err)
	}
	if binary != "/tooling/pwsh" {
		t.Fatalf("unexpected PowerShell binary: %q", binary)
	}
	want := []string{"-NoProfile", "-NonInteractive", "-File", ps, "--apply"}
	if strings.Join(args, "\x00") != strings.Join(want, "\x00") {
		t.Fatalf("unexpected PowerShell args: got=%v want=%v", args, want)
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
	bin := testutil.WriteFakeCommand(t, t.TempDir(), "maestro", []testutil.FakeCommandResponse{
		{
			Args:      []string{"verify", "--json"},
			Stdout:    "{\"version\":\"1.0.0\",\"lifecycle\":\"VERIFY\",\"passed\":true,\"required_actions\":[],\"explanation\":\"all required checks passed\"}\n",
			ReadStdin: true,
		},
	}, 2, "")
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
	bin := testutil.WriteFakeCommand(t, t.TempDir(), "maestro", nil, 0, marker)
	c := &MaestroClient{
		status:     MaestroStatus{Available: true, Mode: MaestroAssist},
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

func TestMaestroCapabilitiesUsesExplicitSkillsListContract(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("NEXUS_DATA_DIR", t.TempDir())
	bin := testutil.WriteFakeCommand(t, t.TempDir(), "maestro", []testutil.FakeCommandResponse{
		{
			Args:   []string{"capabilities", "--json"},
			Stdout: "{\"version\":\"1.0.0\",\"modes\":[\"OFF\",\"ASSIST\",\"ORCHESTRATE\"],\"skills\":[],\"gates\":[\"review\"],\"processes\":[\"verification\"]}\n",
		},
		{
			Args:   []string{"skills", "list", "--json"},
			Stdout: "{\"skills\":[{\"id\":\"security-review\",\"name\":\"Security Review\",\"description\":\"Review security\",\"version\":\"1.2.0\",\"risk\":\"medium\",\"triggers\":[\"security\"]}]}\n",
		},
		{Args: []string{"version"}, Stdout: "1.0.0\n"},
		{Args: []string{"--version"}, Stdout: "1.0.0\n"},
	}, 2, "")
	client := &MaestroClient{maestroBin: bin}
	capabilities, err := client.queryCapabilities()
	if err != nil {
		t.Fatal(err)
	}
	if capabilities.Version != "1.0.0" || len(capabilities.Gates) != 1 || len(capabilities.Processes) != 1 {
		t.Fatalf("capabilities contract was not preserved: %+v", capabilities)
	}
	if len(capabilities.Skills) != 1 || capabilities.Skills[0].ID != "security-review" {
		t.Fatalf("skills list contract was not consumed: %+v", capabilities.Skills)
	}
	if capabilities.Skills[0].Version != "1.2.0" || capabilities.Skills[0].Triggers[0] != "security" {
		t.Fatalf("skills metadata was lost: %+v", capabilities.Skills[0])
	}
}
