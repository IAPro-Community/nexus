package opencode

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	projection "github.com/IAPro-Community/nexus/internal/core/provider/agentprojection"
)

func baseRequest(t *testing.T, version, roleID, access string) projection.Request {
	t.Helper()
	return projection.Request{
		Provider:        "opencode",
		Profile:         "default",
		ProviderVersion: version,
		ProviderHome:    t.TempDir(),
		Scope:           projection.ScopeNexusProfile,
		RoleID:          roleID,
		Role:            roleID,
		AccessIntent:    access,
		Instructions:    []string{"Keep output concise and evidence-based."},
		Constraints:     []string{"Do not widen scope."},
	}
}

func TestRenderOpenCodeV1ReadOnlyAgent(t *testing.T) {
	req := baseRequest(t, "1.18.30", "review", "READ_ONLY")
	artifacts, err := New().Render(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if len(artifacts) != 1 {
		t.Fatalf("expected one artifact, got %d", len(artifacts))
	}
	got := string(artifacts[0].Content)
	for _, want := range []string{
		"mode: all",
		"permission:",
		"  edit: deny",
		"  bash: deny",
		"nexus-agent-projection: opencode/opencode-v1-agent-md/1",
		"Access intent: READ_ONLY",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("V1 render missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "permissions:") || strings.Contains(got, "action: shell") {
		t.Fatalf("V1 render must not use V2 permission syntax:\n%s", got)
	}
	wantSuffix := filepath.Join(".config", "opencode", "agents", "nexus-review.md")
	if !strings.HasSuffix(artifacts[0].Path, wantSuffix) {
		t.Fatalf("unexpected target path %q, want suffix %q", artifacts[0].Path, wantSuffix)
	}
}

func TestRenderOpenCodeV2ReadOnlyAgent(t *testing.T) {
	req := baseRequest(t, "2.0.3", "investigate", "READ_ONLY")
	artifacts, err := New().Render(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	got := string(artifacts[0].Content)
	for _, want := range []string{
		"mode: all",
		"permissions:",
		"action: edit",
		"action: shell",
		"effect: deny",
		"nexus-agent-projection: opencode/opencode-v2-agent-md/1",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("V2 render missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "  bash:") {
		t.Fatalf("V2 render must not use V1 bash permission syntax:\n%s", got)
	}
}

func TestRenderVerifyOnlyAsksForShell(t *testing.T) {
	for _, version := range []string{"1.18.30", "2.0.0"} {
		t.Run(version, func(t *testing.T) {
			req := baseRequest(t, version, "verify", "VERIFY_ONLY")
			artifacts, err := New().Render(context.Background(), req)
			if err != nil {
				t.Fatal(err)
			}
			got := string(artifacts[0].Content)
			if version[0] == '1' {
				if !strings.Contains(got, "bash: ask") || !strings.Contains(got, "edit: deny") {
					t.Fatalf("V1 verify-only permission mismatch:\n%s", got)
				}
			} else {
				if !strings.Contains(got, "action: shell") || !strings.Contains(got, "effect: ask") {
					t.Fatalf("V2 verify-only permission mismatch:\n%s", got)
				}
			}
		})
	}
}

func TestPlanRejectsNonProfileScopeAndUnknownVersion(t *testing.T) {
	adapter := New()

	req := baseRequest(t, "1.18.30", "review", "READ_ONLY")
	req.Scope = projection.ScopeProject
	if _, err := adapter.Plan(context.Background(), req); err == nil {
		t.Fatal("expected PROJECT scope to be rejected")
	}

	req = baseRequest(t, "unknown", "review", "READ_ONLY")
	if _, err := adapter.Plan(context.Background(), req); err == nil {
		t.Fatal("expected unknown OpenCode version to be rejected")
	}
}

func TestApplyVerifyUpdateAndRemoveLifecycle(t *testing.T) {
	root := t.TempDir()
	adapter := New()
	req := projection.Request{
		Provider:        "opencode",
		Profile:         "default",
		ProviderVersion: "1.18.30",
		ProviderHome:    root,
		Scope:           projection.ScopeNexusProfile,
		RoleID:          "review",
		Role:            "reviewer",
		AccessIntent:    "READ_ONLY",
		Instructions:    []string{"Review the current changes."},
	}

	plan, err := adapter.Plan(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Conflicts) != 0 {
		t.Fatalf("new projection unexpectedly conflicts: %#v", plan.Conflicts)
	}
	receipt, err := adapter.Apply(context.Background(), plan)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Strategy != projection.ActivationNexusOverlay {
		t.Fatalf("projection must remain NEXUS_OVERLAY until native activation is proven, got %s", receipt.Strategy)
	}
	if len(receipt.Artifacts) != 1 {
		t.Fatalf("expected one receipt artifact, got %d", len(receipt.Artifacts))
	}

	verification, err := adapter.Verify(context.Background(), receipt)
	if err != nil {
		t.Fatal(err)
	}
	if !verification.Verified {
		t.Fatalf("expected filesystem verification to pass: %#v", verification)
	}
	if len(verification.Warnings) == 0 {
		t.Fatal("verification must disclose that live OpenCode discovery is not yet proved")
	}

	// Re-planning an unchanged managed file is update-safe.
	replan, err := adapter.Plan(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if len(replan.Conflicts) != 0 {
		t.Fatalf("managed unchanged projection should be update-safe: %#v", replan.Conflicts)
	}

	// Change the generated content while the current file still matches the
	// previous manifest: update is allowed and produces a new hash.
	req.Instructions = []string{"Review correctness, regressions, and missing tests."}
	updatedPlan, err := adapter.Plan(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if len(updatedPlan.Conflicts) != 0 {
		t.Fatalf("managed update unexpectedly conflicts: %#v", updatedPlan.Conflicts)
	}
	updatedReceipt, err := adapter.Apply(context.Background(), updatedPlan)
	if err != nil {
		t.Fatal(err)
	}
	if updatedReceipt.Artifacts[0].ContentHash == receipt.Artifacts[0].ContentHash {
		t.Fatal("expected updated projection content hash to change")
	}

	if err := adapter.Remove(context.Background(), updatedReceipt); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(updatedReceipt.Artifacts[0].Path); !os.IsNotExist(err) {
		t.Fatalf("expected projected agent file to be removed, stat err=%v", err)
	}
}

func TestPlanConflictsWithUnmanagedAndExternallyModifiedFiles(t *testing.T) {
	adapter := New()
	root := t.TempDir()
	req := projection.Request{
		Provider:        "opencode",
		Profile:         "default",
		ProviderVersion: "1.18.30",
		ProviderHome:    root,
		Scope:           projection.ScopeNexusProfile,
		RoleID:          "review",
		Role:            "reviewer",
		AccessIntent:    "READ_ONLY",
	}

	rendered, err := adapter.Render(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	target := rendered[0].Path
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("user owned"), 0o600); err != nil {
		t.Fatal(err)
	}
	plan, err := adapter.Plan(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Conflicts) != 1 || plan.Conflicts[0].Kind != projection.ConflictUnmanaged {
		t.Fatalf("expected unmanaged conflict, got %#v", plan.Conflicts)
	}

	if err := os.Remove(target); err != nil {
		t.Fatal(err)
	}
	plan, err = adapter.Plan(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := adapter.Apply(context.Background(), plan)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("external modification"), 0o600); err != nil {
		t.Fatal(err)
	}

	plan, err = adapter.Plan(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Conflicts) != 1 || plan.Conflicts[0].Kind != projection.ConflictManagedModified {
		t.Fatalf("expected managed-modified conflict, got %#v", plan.Conflicts)
	}
	if err := adapter.Remove(context.Background(), receipt); err == nil {
		t.Fatal("remove must refuse an externally modified managed file")
	}
	if got, err := os.ReadFile(target); err != nil || string(got) != "external modification" {
		t.Fatalf("modified user content must remain untouched: %q err=%v", string(got), err)
	}
}

func TestPlanRejectsSymlinkTarget(t *testing.T) {
	if os.PathSeparator == '\\' {
		t.Skip("symlink creation may require elevated privileges on Windows")
	}
	adapter := New()
	root := t.TempDir()
	req := projection.Request{
		Provider:        "opencode",
		Profile:         "default",
		ProviderVersion: "1.18.30",
		ProviderHome:    root,
		Scope:           projection.ScopeNexusProfile,
		RoleID:          "review",
		Role:            "reviewer",
		AccessIntent:    "READ_ONLY",
	}
	rendered, err := adapter.Render(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	target := rendered[0].Path
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "outside.md")
	if err := os.WriteFile(outside, []byte("outside"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, target); err != nil {
		t.Fatal(err)
	}
	plan, err := adapter.Plan(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Conflicts) != 1 || plan.Conflicts[0].Kind != projection.ConflictUnsafePath {
		t.Fatalf("expected unsafe-path conflict, got %#v", plan.Conflicts)
	}
}

func TestCapabilitiesRemainHonestBeforeLiveProbe(t *testing.T) {
	caps := New().Capabilities(context.Background(), "1.18.30")
	if caps.Definitions.Status != projection.CapabilityPartial || !caps.Definitions.Tested {
		t.Fatalf("definition renderer should be PARTIAL with local tests: %#v", caps.Definitions)
	}
	if caps.PrimaryAgent.Status != projection.CapabilityNotTested || caps.PrimaryAgent.Tested {
		t.Fatalf("primary agent must remain NOT_TESTED until live probe: %#v", caps.PrimaryAgent)
	}
	if caps.ToolPermissions.Status != projection.CapabilityNotTested {
		t.Fatalf("tool permission enforcement must remain NOT_TESTED: %#v", caps.ToolPermissions)
	}
	if caps.ProjectScope.Status != projection.CapabilityUnsupported || caps.UserScope.Status != projection.CapabilityUnsupported {
		t.Fatalf("non-profile scopes must remain unsupported in first milestone: %#v", caps)
	}

	unknown := New().Capabilities(context.Background(), "garbage")
	if unknown.Definitions.Status != projection.CapabilityUnknown {
		t.Fatalf("unknown version must downgrade capability truth: %#v", unknown.Definitions)
	}
}

func TestDiscoveryArgsFollowOpenCodeMajorVersion(t *testing.T) {
	v1, err := DiscoveryArgs("1.18.30")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(v1, " ") != "agent list" {
		t.Fatalf("V1 discovery args = %#v", v1)
	}

	v2, err := DiscoveryArgs("2.0.3")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(v2, " ") != "debug agents" {
		t.Fatalf("V2 discovery args = %#v", v2)
	}

	if _, err := DiscoveryArgs("unknown"); err == nil {
		t.Fatal("unknown OpenCode version must not guess discovery command")
	}
}

func TestAgentNameIsStableAndValidated(t *testing.T) {
	name, err := AgentName("review")
	if err != nil {
		t.Fatal(err)
	}
	if name != "nexus-review" {
		t.Fatalf("agent name = %q", name)
	}
	if _, err := AgentName("../escape"); err == nil {
		t.Fatal("unsafe role id must be rejected")
	}
}

func TestActivationProbeArgsSelectProjectedAgentExplicitly(t *testing.T) {
	args, err := ActivationProbeArgs("2.0.3", "nexus-review", "NEXUS_ACTIVATION_OK_123")
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(args, " ")
	for _, want := range []string{
		"run",
		"--agent nexus-review",
		"NEXUS_ACTIVATION_OK_123",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("activation args %q missing %q", joined, want)
		}
	}
	if strings.Contains(joined, "--auto") {
		t.Fatalf("activation probe must not auto-approve permissions: %q", joined)
	}
}

func TestActivationProbeArgsFailClosed(t *testing.T) {
	if _, err := ActivationProbeArgs("unknown", "nexus-review", "marker"); err == nil {
		t.Fatal("unknown OpenCode version must not guess activation command")
	}
	if _, err := ActivationProbeArgs("2.0.3", "review", "marker"); err == nil {
		t.Fatal("non-projected agent name must be rejected")
	}
	if _, err := ActivationProbeArgs("2.0.3", "nexus-review", ""); err == nil {
		t.Fatal("empty activation marker must be rejected")
	}
}


func TestEnforcementProbeArgsUseJSONAndProjectedAgent(t *testing.T) {
	args, err := EnforcementProbeArgs("1.18.30", "nexus-review", "/workspace/.nexus-probe/canary.txt")
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(args, " ")
	for _, want := range []string{"run", "--format json", "--agent nexus-review", "canary.txt"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("enforcement args missing %q: %#v", want, args)
		}
	}
}

func TestParseEnforcementProbeOutputRequiresPermissionDenial(t *testing.T) {
	denied := `{"type":"tool_use","part":{"tool":"edit","state":{"status":"error","error":"Permission denied by agent policy"}}}`
	evidence := ParseEnforcementProbeOutput(denied)
	if !evidence.WriteAttempted || !evidence.WriteDenied || evidence.WriteCompleted || evidence.Tool != "edit" {
		t.Fatalf("unexpected denial evidence: %#v", evidence)
	}

	completed := `{"type":"tool_use","part":{"tool":"write","state":{"status":"completed"}}}`
	evidence = ParseEnforcementProbeOutput(completed)
	if !evidence.WriteAttempted || evidence.WriteDenied || !evidence.WriteCompleted {
		t.Fatalf("completed write must fail enforcement proof: %#v", evidence)
	}

	unrelated := `{"type":"tool_use","part":{"tool":"edit","state":{"status":"error","error":"file not found"}}}`
	evidence = ParseEnforcementProbeOutput(unrelated)
	if !evidence.WriteAttempted || evidence.WriteDenied {
		t.Fatalf("non-permission error must not prove enforcement: %#v", evidence)
	}
}
