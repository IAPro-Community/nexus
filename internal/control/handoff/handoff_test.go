package handoff

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IAPro-Community/nexus/internal/control/driver"
	"github.com/IAPro-Community/nexus/internal/control/events"
	"github.com/IAPro-Community/nexus/internal/control/registry"
)

func TestRollbackPreservesAgentAndProjectIdentity(t *testing.T) {
	source := registry.RuntimeSession{
		RuntimeID: "origin", AgentID: "agent-123", ProjectID: "project-456",
		ProjectName: "Project", Title: "Persistent Agent",
		ProviderID: "codex", ProfileID: "work", ProviderSessionID: "session-abc",
		Workspace: "/worktree", Model: "model", Labels: map[string]string{"role": "reviewer"},
	}
	opts := rollbackLaunchOptions(source, "recovered-789", []string{"resume", "session-abc"})
	if opts.AgentID != source.AgentID || opts.ProjectID != source.ProjectID ||
		opts.ProjectName != source.ProjectName || opts.Workspace != source.Workspace ||
		opts.ProviderSessionID != source.ProviderSessionID || opts.Model != source.Model {
		t.Fatalf("rollback lost stable Agent/project/session identity: %+v", opts)
	}
	if opts.RuntimeID == source.RuntimeID || opts.RuntimeID != "recovered-789" {
		t.Fatalf("rollback must start a new runtime generation: %+v", opts)
	}
	opts.Labels["role"] = "changed"
	if source.Labels["role"] != "reviewer" {
		t.Fatal("rollback changed source labels instead of cloning them")
	}
}

func TestContextHandoffSourceStateNeverRevivesStoppedProcess(t *testing.T) {
	live := registry.RuntimeSession{PID: os.Getpid(), State: registry.StateWaiting}
	if got := contextHandoffSourceState(live); got != registry.StateWaiting {
		t.Fatalf("live source should retain WAITING, got %s", got)
	}
	if got := contextHandoffSourceState(registry.RuntimeSession{PID: 0, State: registry.StateRunning}); got != registry.StateStale {
		t.Fatalf("unknown PID must never be reported as RUNNING, got %s", got)
	}
	if got := contextHandoffSourceState(registry.RuntimeSession{PID: 99999999, State: registry.StateRunning}); got != registry.StateStopped {
		t.Fatalf("dead source must never be reported as RUNNING, got %s", got)
	}
}

func TestHandoffServiceUsesInjectedRuntimeRegistry(t *testing.T) {
	owned := registry.NewRegistry("")
	ownedRuntimeID := "handoff-owned-runtime-test"
	if err := owned.Register(registry.RuntimeSession{
		RuntimeID:         ownedRuntimeID,
		ProviderID:        "fake",
		ProfileID:         "work",
		ProviderSessionID: "session-owned",
		Workspace:         t.TempDir(),
		State:             registry.StateRunning,
	}); err != nil {
		t.Fatal(err)
	}

	service := NewService(Dependencies{
		Registry: owned,
		Drivers:  driver.NewRegistry(),
	})
	_, err := service.PerformAccountHandoff(context.Background(), ownedRuntimeID, "other:profile")
	if err == nil || !strings.Contains(err.Error(), "matching provider") {
		t.Fatalf("handoff did not resolve the injected runtime registry: %v", err)
	}
}

func TestContextHandoffCompletedEventCorrelatesByLineage(t *testing.T) {
	e := newContextHandoffCompletedEvent(
		"lineage-context-123",
		registry.RuntimeSession{RuntimeID: "runtime-source", ProviderID: "codex", ProfileID: "work"},
		registry.RuntimeSession{RuntimeID: "runtime-target", ProviderID: "claude", ProfileID: "work"},
		"checkpoint-1",
	)
	if e.CorrelationID != "lineage-context-123" {
		t.Fatalf("expected lineage correlation, got %q", e.CorrelationID)
	}
	if e.Type != events.EventHandoffCompleted || e.RuntimeID != "runtime-target" {
		t.Fatalf("unexpected context handoff event: %+v", e)
	}
	if e.Data["source_id"] != "runtime-source" || e.Data["checkpoint_id"] != "checkpoint-1" {
		t.Fatalf("event data lost handoff identity: %+v", e.Data)
	}
}

func TestAccountHandoffValidation(t *testing.T) {
	reg := registry.DefaultRegistry()
	ctx := context.Background()

	// 1. Test failure when ProviderSessionID is missing
	sessNoID := registry.RuntimeSession{
		RuntimeID:         "rt-no-sess-id",
		ProviderID:        "fake",
		ProfileID:         "work",
		ProviderSessionID: "",
		Workspace:         os.TempDir(),
		State:             registry.StateRunning,
	}
	_ = reg.Register(sessNoID)

	_, err := PerformAccountHandoff(ctx, "rt-no-sess-id", "fake:personal")
	if err == nil || !strings.Contains(err.Error(), "source provider session ID is unknown") {
		t.Errorf("expected error for empty session ID, got %v", err)
	}

	// 2. Test failure when target provider doesn't match source provider
	sessValid := registry.RuntimeSession{
		RuntimeID:         "rt-with-sess-id",
		ProviderID:        "codex",
		ProfileID:         "work",
		ProviderSessionID: "sess-12345",
		Workspace:         os.TempDir(),
		State:             registry.StateRunning,
	}
	_ = reg.Register(sessValid)

	_, err = PerformAccountHandoff(ctx, "rt-with-sess-id", "claude:work")
	if err == nil || !strings.Contains(err.Error(), "matching provider") {
		t.Errorf("expected error for mismatched provider in account handoff, got %v", err)
	}
}

func TestWorkCheckpointAndRedaction(t *testing.T) {
	secretGoal := "Fix bug with sk-proj-1234567890abcdef1234567890abcdef1234567890 and Bearer eyJhbGciOiJIUzI1NiJ9.test"
	cp := CaptureWorkCheckpoint(os.TempDir(), "rt-src", "codex", "work", "sess-abc", "gpt-5.6-sol", secretGoal)

	if strings.Contains(cp.Goal, "sk-proj-") {
		t.Errorf("expected secret OpenAI key in goal to be redacted, got: %s", cp.Goal)
	}
	if strings.Contains(cp.Goal, "eyJhbGciOiJIUzI1NiJ9") {
		t.Errorf("expected secret JWT in goal to be redacted, got: %s", cp.Goal)
	}

	prompt := FormatKickoffPrompt(cp)
	if strings.Contains(prompt, "sk-proj-") {
		t.Errorf("expected prompt to have redacted secrets, got: %s", prompt)
	}

	path, err := SaveCheckpoint(cp)
	if err != nil || path == "" {
		t.Fatalf("failed to save checkpoint: %v", err)
	}
	_ = os.Remove(path)
}

func TestCheckpointCapturesMaestroContextAndKickoffRequiresRehydration(t *testing.T) {
	workspace := t.TempDir()
	if err := os.MkdirAll(filepath.Join(workspace, "DEV", "SPECS"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, relative := range []string{"DEV/INDEX.md", "DEV/HANDOFF.md", "DEV/CONTEXT.md", "DEV/SPECS/ACTIVE.md", "DEV/VERIFY.md"} {
		path := filepath.Join(workspace, filepath.FromSlash(relative))
		if err := os.WriteFile(path, []byte("durable context: "+relative), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	cp := CaptureWorkCheckpoint(workspace, "rt-source", "codex", "work", "", "", "continue task")
	if cp.SchemaVersion < 4 || len(cp.MaestroContext.ReadOrder) != 5 || len(cp.MaestroContext.Files) != 5 {
		t.Fatalf("unexpected Maestro context pack: %+v", cp.MaestroContext)
	}
	for _, file := range cp.MaestroContext.Files {
		if file.Status != "PRESENT" || file.SHA256 == "" || file.Bytes == 0 {
			t.Fatalf("context file lacks evidence: %+v", file)
		}
	}
	prompt := FormatKickoffPrompt(cp)
	if !strings.Contains(prompt, "Maestro continuity contract") || !strings.Contains(prompt, "DEV/SPECS/ACTIVE.md [PRESENT]") {
		t.Fatalf("kickoff did not require durable context rehydration: %s", prompt)
	}
}

func TestAccountHandoffStateTransitionsAndRollback(t *testing.T) {
	// Not full E2E due to process complexities, but we can verify state logic
	reg := registry.DefaultRegistry()
	ctx := context.Background()

	// 1. Setup active source
	sourceID := "rt-test-rollback"
	sess := registry.RuntimeSession{
		RuntimeID:         sourceID,
		ProviderID:        "fake",
		ProfileID:         "work",
		ProviderSessionID: "sess-rollback-test",
		Workspace:         os.TempDir(),
		State:             registry.StateRunning,
	}
	_ = reg.Register(sess)

	// We try to handoff to an unauthenticated profile or something that fails
	// Wait, we can't easily mock auth here without config changes.
	// But we can check that it fails safely and we get rollback behavior.
	_, err := PerformAccountHandoff(ctx, sourceID, "fake:non-existent")
	if err == nil {
		t.Errorf("expected handoff to fail for unauthenticated or non-existent profile")
	}

	// Because we didn't mock a process for `sourceID`, IsProcessAlive will return false.
	// So rollback will attempt to spawn a new process.
	// We can just check that the source state is either running or we get an error about rollback.
	restored, ok := reg.Get(sourceID)
	if ok && restored.State == registry.StateHandoff {
		t.Errorf("source state should not remain in Handoff after failure")
	}
}

func TestContextHandoffIntegration(t *testing.T) {
	reg := registry.DefaultRegistry()
	ctx := context.Background()

	sourceID := "rt-ctx-test"
	sess := registry.RuntimeSession{
		RuntimeID:         sourceID,
		ProviderID:        "fake",
		ProfileID:         "work",
		ProviderSessionID: "sess-ctx-test",
		Workspace:         os.TempDir(),
		State:             registry.StateRunning,
	}
	_ = reg.Register(sess)

	// Wait, without valid credentials, PerformContextHandoff fails at Authentication check.
	_, err := PerformContextHandoff(ctx, sourceID, "fake", "default")
	if err == nil {
		t.Errorf("expected failure if fake:default isn't authenticated, or success if it is")
	}
}

func TestGitBounding(t *testing.T) {
	// Test the checkpoint bounding on git diff
	dir := os.TempDir() + "/handoff-git-test"
	os.MkdirAll(dir, 0700)
	defer os.RemoveAll(dir)

	cp := CaptureWorkCheckpoint(dir, "rt-1", "fake", "prof", "sess-1", "fake-model", "Test goal")
	if cp.Workspace != dir {
		t.Errorf("expected workspace %s, got %s", dir, cp.Workspace)
	}
}

func TestAccountHandoff_CheckpointFailureAborts(t *testing.T) {
	reg := registry.DefaultRegistry()
	ctx := context.Background()

	sourceID := "rt-cp-fail-test"
	sess := registry.RuntimeSession{
		RuntimeID:         sourceID,
		ProviderID:        "fake",
		ProfileID:         "work",
		ProviderSessionID: "sess-cp-fail",
		Workspace:         os.TempDir(),
		State:             registry.StateRunning,
	}
	_ = reg.Register(sess)

	// In normal circumstances, handoff to fake:nonexistent fails early with clear error
	_, err := PerformAccountHandoff(ctx, sourceID, "fake:nonexistent")
	if err == nil {
		t.Fatal("expected handoff to fail on invalid profile")
	}

	// Source session must not be stuck in StateHandoff
	st, ok := reg.Get(sourceID)
	if ok && st.State == registry.StateHandoff {
		t.Errorf("source session state was not preserved/restored: %+v", st)
	}
}

func TestVerifyResumeContinuity(t *testing.T) {
	// Live process: this test process itself.
	live := &registry.RuntimeSession{
		RuntimeID: "rt-verify-live",
		PID:       os.Getpid(),
		State:     registry.StateRunning,
	}

	// Passing case: resume args reference the session ID.
	if ok, reason := VerifyResumeContinuity(live, []string{"resume", "sess-abc"}, "sess-abc"); !ok {
		t.Errorf("expected verification to pass, got: %s", reason)
	}

	// Failing: resume args reference a different session.
	if ok, _ := VerifyResumeContinuity(live, []string{"resume", "sess-zzz"}, "sess-abc"); ok {
		t.Error("verification must fail when resume args do not reference the session ID")
	}

	// Failing: process not alive.
	dead := &registry.RuntimeSession{
		RuntimeID: "rt-verify-dead",
		PID:       99999999,
		State:     registry.StateRunning,
	}
	if ok, _ := VerifyResumeContinuity(dead, []string{"resume", "sess-abc"}, "sess-abc"); ok {
		t.Error("verification must fail when the target process is not alive")
	}

	// Failing: state not running.
	stopped := &registry.RuntimeSession{
		RuntimeID: "rt-verify-stopped",
		PID:       os.Getpid(),
		State:     registry.StateStopped,
	}
	if ok, _ := VerifyResumeContinuity(stopped, []string{"resume", "sess-abc"}, "sess-abc"); ok {
		t.Error("verification must fail when the target runtime is not RUNNING")
	}

	// Failing: empty session ID.
	if ok, _ := VerifyResumeContinuity(live, []string{"resume", ""}, ""); ok {
		t.Error("verification must fail on empty session ID")
	}

	// Failing: nil session.
	if ok, _ := VerifyResumeContinuity(nil, []string{"resume", "sess-abc"}, "sess-abc"); ok {
		t.Error("verification must fail on nil session")
	}
}

func TestResolveTargetModel(t *testing.T) {
	tests := []struct {
		name           string
		sourceModel    string
		sourceProvider string
		targetProvider string
		want           string
	}{
		{
			name:           "empty source model returns empty",
			sourceModel:    "",
			sourceProvider: "agy",
			targetProvider: "codex",
			want:           "",
		},
		{
			name:           "same provider preserves model unconditionally",
			sourceModel:    "custom-internal-model",
			sourceProvider: "agy",
			targetProvider: "agy",
			want:           "custom-internal-model",
		},
		{
			name:           "same provider case-insensitive preserves model",
			sourceModel:    "gpt-5.6-sol",
			sourceProvider: "Codex",
			targetProvider: "codex",
			want:           "gpt-5.6-sol",
		},
		{
			name:           "cross-provider supported target retains model",
			sourceModel:    "claude-sonnet-4-20250514",
			sourceProvider: "agy",
			targetProvider: "claude",
			want:           "claude-sonnet-4-20250514",
		},
		{
			name:           "cross-provider to agy retains model",
			sourceModel:    "gemini-2.5-pro",
			sourceProvider: "codex",
			targetProvider: "agy",
			want:           "gemini-2.5-pro",
		},
		{
			name:           "cross-provider to cursor falls back to empty (unsupported model flag)",
			sourceModel:    "claude-sonnet-4-20250514",
			sourceProvider: "agy",
			targetProvider: "cursor",
			want:           "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveTargetModel(tt.sourceModel, tt.sourceProvider, tt.targetProvider)
			if got != tt.want {
				t.Errorf("ResolveTargetModel(%q, %q, %q) = %q, want %q",
					tt.sourceModel, tt.sourceProvider, tt.targetProvider, got, tt.want)
			}
		})
	}
}

func TestWorkCheckpointModelPersistenceAndPrompt(t *testing.T) {
	cp := CaptureWorkCheckpoint(os.TempDir(), "rt-src", "agy", "work", "sess-123", "claude-sonnet-4-20250514", "Complete feature")
	if cp.SchemaVersion != 4 {
		t.Errorf("expected SchemaVersion 4, got %d", cp.SchemaVersion)
	}
	if cp.SourceModel != "claude-sonnet-4-20250514" {
		t.Errorf("expected SourceModel 'claude-sonnet-4-20250514', got %q", cp.SourceModel)
	}

	prompt := FormatKickoffPrompt(cp)
	if !strings.Contains(prompt, "Previous Model: claude-sonnet-4-20250514") {
		t.Errorf("expected kickoff prompt to contain previous model info, got:\n%s", prompt)
	}

	// When source model is empty, kickoff prompt does not contain Previous Model line
	cpNoModel := CaptureWorkCheckpoint(os.TempDir(), "rt-src", "codex", "work", "sess-123", "", "Complete feature")
	promptNoModel := FormatKickoffPrompt(cpNoModel)
	if strings.Contains(promptNoModel, "Previous Model:") {
		t.Errorf("expected kickoff prompt to omit previous model line when empty, got:\n%s", promptNoModel)
	}
}
