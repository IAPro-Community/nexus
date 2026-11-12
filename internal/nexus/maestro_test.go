package nexus

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/IAPro-Community/nexus/internal/testutil"
)

func TestFindMaestroBinPrefersStableAliasOverNodeVersionShims(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell executable fixtures are Unix-only")
	}
	dir := t.TempDir()
	for _, name := range []string{"maestro", "orquestrador-maestro"} {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", dir)
	t.Setenv("HOME", dir)
	t.Setenv("NEXUS_MAESTRO_BIN", "")

	if got, want := findMaestroBin(), filepath.Join(dir, "maestro"); got != want {
		t.Fatalf("findMaestroBin() = %q, want preferred stable alias %q", got, want)
	}
}

func TestNewMaestroClientSkipsAliasWithoutContractAndUsesCompatibleCandidate(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell executable fixtures are Unix-only")
	}
	dir := t.TempDir()
	bad := filepath.Join(dir, "maestro")
	if err := os.WriteFile(bad, []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	good := filepath.Join(dir, "orquestrador-maestro")
	contents := `#!/bin/sh
if [ "$1" = "bridge" ] && [ "$2" = "--stdio" ]; then
  cat >/dev/null
  printf '%s\n' '{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":1,"capabilities":{"contextBrief":true,"projectContext":true,"plansValidate":true,"runsSubmit":false,"memory":{"read":true,"write":false}}}}'
  exit 0
fi
exit 1
`
	if err := os.WriteFile(good, []byte(contents), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	t.Setenv("HOME", dir)
	t.Setenv("NEXUS_MAESTRO_BIN", "")

	client := NewMaestroClient()
	if !client.Status().Available || client.maestroBin != good {
		t.Fatalf("expected compatible candidate after stale alias, got bin=%q status=%+v", client.maestroBin, client.Status())
	}
}

func TestExplicitMaestroBinIsAuthoritative(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell executable fixtures are Unix-only")
	}
	dir := t.TempDir()
	bin := filepath.Join(dir, "selected-maestro")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\nexit 9\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	t.Setenv("HOME", dir)
	t.Setenv("NEXUS_MAESTRO_BIN", bin)

	client := NewMaestroClient()
	if client.maestroBin != bin || client.Status().Available {
		t.Fatalf("explicit incompatible binary must fail closed without another fallback: bin=%q status=%+v", client.maestroBin, client.Status())
	}
}

func TestMaestroBridgeV1ContractInterop(t *testing.T) {
	bin := os.Getenv("NEXUS_MAESTRO_CLI_TEST_PATH")
	if bin == "" {
		t.Skip("set NEXUS_MAESTRO_CLI_TEST_PATH to exercise the real Maestro Bridge v1 contract")
	}
	client := &MaestroClient{maestroBin: bin}
	client.checkAvailability()
	status := client.Status()
	if !status.Available || status.Capabilities == nil || status.Capabilities.Bridge == nil {
		t.Fatalf("real Maestro Bridge v1 capabilities unavailable: %+v", status)
	}
	if status.Capabilities.Version != "bridge-v1" || status.Capabilities.Bridge.ProtocolVersion != 1 {
		t.Fatalf("real Maestro returned unexpected Bridge protocol identity: %+v", status.Capabilities)
	}
	if !status.Capabilities.Bridge.ProjectContext || !status.Capabilities.Bridge.PlansValidate ||
		status.Capabilities.Bridge.RunsSubmit {
		t.Fatalf("real Maestro did not advertise the expected v1 capabilities or falsely enabled execution: %+v", status.Capabilities.Bridge)
	}
}

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
	c := &MaestroClient{status: MaestroStatus{Available: true, Mode: MaestroAssist, Capabilities: &MaestroCapability{Version: "1.2.3", Contracts: MaestroContracts{Advice: "1.0.0"}}}, maestroBin: bin}
	resp, err := c.GetAdvice(AdviceContext{ProjectID: "p1"}, "ship product")
	if err == nil {
		t.Fatal("expected advise contract failure")
	}
	if !resp.Degraded || len(resp.Required)+len(resp.Recommended)+len(resp.Optional) != 0 {
		t.Fatalf("expected empty degraded response, got %+v", resp)
	}
}

func TestMaestroAdviceAndVerifyRequireAdvertisedContractVersions(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "called")
	bin := testutil.WriteFakeCommand(t, t.TempDir(), "maestro", nil, 0, marker)
	client := &MaestroClient{
		status:     MaestroStatus{Available: true, Mode: MaestroAssist, Capabilities: &MaestroCapability{Version: "1.1.0"}},
		maestroBin: bin,
	}
	advice, adviceErr := client.GetAdvice(AdviceContext{ProjectID: "project-1"}, "review security")
	if adviceErr == nil || advice == nil || !advice.Degraded {
		t.Fatalf("advice must fail closed without its advertised contract: response=%+v err=%v", advice, adviceErr)
	}
	verification, verifyErr := client.Verify(context.Background(), MaestroVerifyRequest{ProjectID: "project-1"})
	if verifyErr == nil || verification == nil || verification.Passed || !verification.Degraded {
		t.Fatalf("verification must fail closed without its advertised contract: response=%+v err=%v", verification, verifyErr)
	}
	if _, statErr := os.Stat(marker); !os.IsNotExist(statErr) {
		t.Fatalf("Maestro process was invoked without a negotiated contract: %v", statErr)
	}
}

func TestMaestroAdviceRejectsSecretLikeIntentBeforeExecution(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "called")
	bin := testutil.WriteFakeCommand(t, t.TempDir(), "maestro", nil, 0, marker)
	c := &MaestroClient{
		status:     MaestroStatus{Available: true, Mode: MaestroAssist, Capabilities: &MaestroCapability{Contracts: MaestroContracts{Advice: "1.0.0"}}},
		maestroBin: bin,
	}
	resp, err := c.GetAdvice(AdviceContext{ProjectID: "project-1"}, "help with token=super-secret")
	if err == nil {
		t.Fatal("secret-like advice intent must be rejected")
	}
	if resp == nil || !resp.Degraded || resp.Mode != MaestroOff {
		t.Fatalf("unsafe advice must return a degraded response: %+v", resp)
	}
	if _, statErr := os.Stat(marker); !os.IsNotExist(statErr) {
		t.Fatalf("Maestro process was invoked for rejected input: %v", statErr)
	}
}

func TestMaestroAdviceAcceptsValidEmptyRouterResult(t *testing.T) {
	bin := testutil.WriteFakeCommand(t, t.TempDir(), "maestro", []testutil.FakeCommandResponse{
		{Args: []string{"advise", "--json"}, Stdout: "{\"version\":\"1.1.0\",\"mode\":\"ASSIST\",\"lifecycle\":\"TASK\",\"required\":[],\"recommended\":[],\"optional\":[],\"explanation\":\"no canonical skill matched\"}\n", ReadStdin: true},
	}, 2, "")
	c := &MaestroClient{
		status:     MaestroStatus{Available: true, Mode: MaestroAssist, Capabilities: &MaestroCapability{Version: "1.1.0", Contracts: MaestroContracts{Advice: "1.0.0"}}},
		maestroBin: bin,
	}
	resp, err := c.GetAdvice(AdviceContext{ProjectID: "p1", Lifecycle: MaestroLifecycleTask}, "unmatched intent")
	if err != nil {
		t.Fatal(err)
	}
	if resp.Degraded || resp.Mode != MaestroAssist || resp.Lifecycle != MaestroLifecycleTask || len(resp.Recommended) != 0 {
		t.Fatalf("valid empty routing result must remain non-degraded: %+v", resp)
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
				Version:   MaestroVersion,
				Contracts: MaestroContracts{Verification: "1.0.0"},
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
		status:     MaestroStatus{Available: true, Mode: MaestroAssist, Capabilities: &MaestroCapability{Contracts: MaestroContracts{Verification: "1.0.0"}}},
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

func TestMaestroCapabilitiesNegotiatesBridgeV1Contract(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell executable fixtures are Unix-only")
	}
	t.Setenv("HOME", t.TempDir())
	t.Setenv("NEXUS_DATA_DIR", t.TempDir())
	marker := filepath.Join(t.TempDir(), "bridge-invoked")
	reply := `{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":1,"capabilities":{"projectContext":true,"contextBrief":true,"plansValidate":true,"projectCompose":false,"plansRoutePreview":false,"runsObserve":true,"runsSubmissionsRead":true,"runsSubmissionsPoll":false,"runsSubmissionsCancel":false,"runsSubmit":false,"memory":{"read":true,"write":false}}}}`
	bin := filepath.Join(t.TempDir(), "maestro")
	script := fmt.Sprintf("#!/bin/sh\n: > %s\ncat <<'NEXUS_BRIDGE_REPLY'\n%s\nNEXUS_BRIDGE_REPLY\n", strconv.Quote(marker), reply)
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	client := &MaestroClient{maestroBin: bin}
	capabilities, err := client.queryCapabilities()
	if err != nil {
		t.Fatal(err)
	}
	if _, statErr := os.Stat(marker); statErr != nil {
		t.Fatalf("Bridge handshake did not invoke the Maestro binary: %v", statErr)
	}
	if capabilities.Version != "bridge-v1" {
		t.Fatalf("unexpected Bridge protocol identity: %+v", capabilities)
	}
	if capabilities.Modes[0] != "OFF" || capabilities.Modes[1] != "ASSIST" || len(capabilities.Modes) != 2 {
		t.Fatalf("Bridge v1 must not advertise ORCHESTRATE before a durable worker exists: %+v", capabilities.Modes)
	}
	bridge := capabilities.Bridge
	if bridge == nil || bridge.ProtocolVersion != 1 {
		t.Fatalf("Bridge v1 capabilities were not negotiated: %+v", bridge)
	}
	if !bridge.ProjectContext || !bridge.ContextBrief || !bridge.PlansValidate {
		t.Fatalf("supported Nexus integration methods were lost: %+v", bridge)
	}
	if bridge.RunsSubmit || bridge.SubmissionsPoll || bridge.SubmissionsCancel {
		t.Fatalf("execution submission must stay explicitly false without a durable worker: %+v", bridge)
	}
	if !bridge.MemoryRead || bridge.MemoryWrite {
		t.Fatalf("memory contract was not preserved: %+v", bridge)
	}
	if len(capabilities.Skills) != 0 || len(capabilities.Gates) != 0 || len(capabilities.Processes) != 0 {
		t.Fatalf("Bridge v1 must not invent gates, processes or skills: %+v", capabilities)
	}
}

func TestMaestroVersionAloneDoesNotSatisfyCapabilitiesContract(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("NEXUS_DATA_DIR", t.TempDir())
	bin := testutil.WriteFakeCommand(t, t.TempDir(), "maestro", []testutil.FakeCommandResponse{
		{Args: []string{"version"}, Stdout: "1.1.0\n"},
		{Args: []string{"--version"}, Stdout: "1.1.0\n"},
	}, 2, "")
	client := &MaestroClient{maestroBin: bin}
	client.checkAvailability()
	status := client.Status()
	if status.Available || status.Mode != MaestroOff {
		t.Fatalf("version-only Maestro must remain degraded: %+v", status)
	}
	if !strings.Contains(status.Error, "maestro Bridge v1 unavailable") {
		t.Fatalf("missing explicit degraded reason: %q", status.Error)
	}
}

func TestMaestroUnavailableWithoutExternalExecutable(t *testing.T) {
	// The installed skills catalog may exist independently of the Maestro CLI.
	// It must never mark the mandatory external executor as READY.
	client := &MaestroClient{maestroBin: ""}
	client.checkAvailability()
	status := client.Status()
	if status.Available {
		t.Fatal("Maestro reported available without an executable")
	}
	if status.Mode != MaestroOff {
		t.Fatalf("unavailable Maestro must be OFF, got %q", status.Mode)
	}
	if status.Error == "" {
		t.Fatal("missing executable must produce an explicit error")
	}
}
