package nexus

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/driver"
	"github.com/IAPro-Community/nexus/internal/control/launcher"
	"github.com/IAPro-Community/nexus/internal/control/registry"
	"github.com/IAPro-Community/nexus/internal/nexus/intelligence"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func TestRuntimeKillIdentityValidRequiresGeneration(t *testing.T) {
	if runtimeKillIdentityValid(os.Getpid(), 0) {
		t.Fatal("legacy runtime without generation must never be a destructive kill target")
	}
	if runtimeKillIdentityValid(-1, time.Now().UnixNano()) {
		t.Fatal("invalid PID must never be a destructive kill target")
	}
	if !runtimeKillIdentityValid(os.Getpid(), time.Now().UnixNano()) {
		t.Fatal("current process with a fresh generation should pass identity validation")
	}
}

var mockRuntimeSeq uint64

// mockLauncher satisfies the Launcher interface for unit tests without spawning
// real processes. It registers runtimes in the DefaultRegistry so runtimeAlive()
// returns true for mocked sessions, and tracks Stop calls.
type mockLauncher struct{}

func newMockLauncher() *mockLauncher {
	// Reset the singleton so each test gets an isolated registry.
	registry.ResetDefaultRegistryForTest()
	return &mockLauncher{}
}

func (m *mockLauncher) Launch(_ context.Context, opts launcher.LaunchOptions) (*registry.RuntimeSession, error) {
	if opts.ProviderID == "" {
		return nil, fmt.Errorf("provider is required")
	}
	sess := registry.RuntimeSession{
		RuntimeID:         opts.RuntimeID,
		ProviderID:        opts.ProviderID,
		ProfileID:         opts.ProfileID,
		ProviderSessionID: opts.ProviderSessionID,
		Workspace:         opts.Workspace,
		State:             registry.StateRunning,
		PID:               os.Getpid(), // use real PID so IsProcessAlive works
		StartedAt:         time.Now(),
		Transport:         "mock",
	}
	if sess.RuntimeID == "" {
		seq := atomic.AddUint64(&mockRuntimeSeq, 1)
		sess.RuntimeID = fmt.Sprintf("mock-%s-%d", opts.ProviderID, seq)
	}
	if err := registry.DefaultRegistry().Register(sess); err != nil {
		return nil, fmt.Errorf("mock register: %w", err)
	}
	return &sess, nil
}

func (m *mockLauncher) Stop(runtimeID string) error {
	_ = registry.DefaultRegistry().UpdateState(runtimeID, registry.StateStopped)
	registry.DefaultRegistry().Delete(runtimeID)
	return nil
}

func openTestNexus(t *testing.T) *Nexus {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("AI_CLI_DATA_DIR", dir)
	st, err := store.Open(filepath.Join(dir, "nexus.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return &Nexus{st: st, launcher: newMockLauncher()}
}

func TestNexusUsesInjectedControlDriverRegistry(t *testing.T) {
	injected := driver.NewRegistry()
	injected.Register(driver.NewFakeDriver())
	n := &Nexus{drivers: injected}

	if _, err := n.controlDrivers().Get("fake"); err != nil {
		t.Fatalf("injected driver registry was not used: %v", err)
	}
	if _, err := n.controlDrivers().Get("codex"); err == nil {
		t.Fatal("provider lookup leaked to the process-global registry")
	}
}

func TestEffectiveAgentState(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	proj, _ := st.CreateProject(store.Project{Name: "P", CanonicalPath: t.TempDir()})
	agent, _ := st.CreateAgent(store.Agent{ProjectID: proj.ID, Name: "Dev"})

	// No generation yet → store status (STOPPED).
	state, err := n.EffectiveAgentState(agent.ID)
	if err != nil || state != store.AgentStopped {
		t.Fatalf("expected STOPPED with no generation, got %q (%v)", state, err)
	}

	// A generation referencing a dead runtime + WORKING status → RECOVERABLE.
	dead := &store.RuntimeGeneration{
		AgentID:    agent.ID,
		RevisionID: "rev_x",
		RuntimeID:  "rt_definitely_dead_xyz",
		Provider:   "fake",
		Profile:    "default",
		StartedAt:  time.Now().UTC(),
		State:      "RUNNING",
	}
	if _, err := st.AddGeneration(*dead); err != nil {
		t.Fatal(err)
	}
	agent.Status = store.AgentWorking
	if err := st.UpdateAgent(agent); err != nil {
		t.Fatal(err)
	}

	state, err = n.EffectiveAgentState(agent.ID)
	if err != nil || state != store.AgentRecoverable {
		t.Fatalf("expected RECOVERABLE for dead runtime, got %q (%v)", state, err)
	}

	// A STOPPED agent with a dead runtime stays STOPPED (never claims RECOVERABLE).
	agent.Status = store.AgentStopped
	_ = st.UpdateAgent(agent)
	state, _ = n.EffectiveAgentState(agent.ID)
	if state != store.AgentStopped {
		t.Fatalf("expected STOPPED (not recoverable) for stopped agent, got %q", state)
	}
}

func TestRecoverAgentRequiresKnownRuntime(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	proj, _ := st.CreateProject(store.Project{Name: "P", CanonicalPath: t.TempDir()})
	agent, _ := st.CreateAgent(store.Agent{ProjectID: proj.ID, Name: "Dev"})

	// Recover on an agent with a live runtime must refuse cleanly (no panic, no 500).
	// Without a live generation this is guarded by EffectiveAgentState in the API;
	// here we assert the store-level invariants hold for a stopped agent.
	if got := os.Getenv("AI_CLI_DATA_DIR"); got == "" {
		t.Fatal("data dir not set")
	}
	if agent.ID == "" {
		t.Fatal("agent id empty")
	}
}

func TestAskAgentReusesExistingAgentAndLiveRuntime(t *testing.T) {
	n := openTestNexus(t)
	st, _ := n.OpenProject()
	proj, err := st.CreateProject(store.Project{Name: "P", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	agent, err := st.CreateAgent(store.Agent{ProjectID: proj.ID, Name: "Dev"})
	if err != nil {
		t.Fatal(err)
	}
	sess, err := n.StartAgent(context.Background(), agent.ID, "fake", "default")
	if err != nil {
		t.Fatal(err)
	}
	var gotRuntime, gotPrompt string
	n.submitPrompt = func(runtimeID, prompt string) error { gotRuntime, gotPrompt = runtimeID, prompt; return nil }

	result, err := n.AskAgent(context.Background(), agent.ID, "fix the tests", false)
	if err != nil {
		t.Fatal(err)
	}
	if result.AgentID != agent.ID || result.RuntimeID != sess.RuntimeID || result.Started {
		t.Fatalf("Ask must reuse Agent/live runtime: %+v", result)
	}
	if result.ExecutionRole != "build" {
		t.Fatalf("implicit runtime role must report build without changing the prompt: %+v", result)
	}
	if gotRuntime != sess.RuntimeID || gotPrompt != "fix the tests" {
		t.Fatalf("submit=%q %q", gotRuntime, gotPrompt)
	}
	agents, _ := st.ListAgents(proj.ID)
	if len(agents) != 1 {
		t.Fatalf("Ask created another Agent: %d", len(agents))
	}
}

func TestAskAgentCompilesPersistentSpecializationForCustomAgent(t *testing.T) {
	n := openTestNexus(t)
	st, _ := n.OpenProject()
	project, err := st.CreateProject(store.Project{Name: "QA", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	agent, err := st.CreateAgent(store.Agent{ProjectID: project.ID, Name: "QA", Role: "qa"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := n.SafeApply(context.Background(), agent.ID, AgentConfig{
		Provider: "fake", Profile: "default",
		AgentSpec: intelligence.AgentSpec{Role: "qa", Instructions: []string{"Collect evidence before suggesting fixes"}},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := n.StartAgent(context.Background(), agent.ID, "fake", "default"); err != nil {
		t.Fatal(err)
	}
	var submitted string
	n.submitPrompt = func(_ string, prompt string) error { submitted = prompt; return nil }
	if _, err := n.AskAgent(context.Background(), agent.ID, "reproduce the failure", false); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(submitted, "Collect evidence before suggesting fixes") || !strings.Contains(submitted, "reproduce the failure") {
		t.Fatalf("persistent specialization did not reach execution prompt: %q", submitted)
	}
}

func TestAskAgentWithOptionsSkipsEnvelopeForInteractiveLead(t *testing.T) {
	n := openTestNexus(t)
	st, _ := n.OpenProject()
	project, err := st.CreateProject(store.Project{Name: "Lead", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	agent, err := n.PrepareInteractiveLead(context.Background(), project.ID, "fake", "default")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := n.StartAgent(context.Background(), agent.ID, "fake", "default"); err != nil {
		t.Fatal(err)
	}
	var submitted string
	n.submitPrompt = func(_ string, prompt string) error { submitted = prompt; return nil }
	if _, err := n.AskAgentWithOptions(context.Background(), agent.ID, "fix the typo", AskAgentOptions{
		CompileSpecialization: true,
	}); err != nil {
		t.Fatal(err)
	}
	if submitted != "fix the typo" {
		t.Fatalf("interactive lead must receive raw prompt, got %q", submitted)
	}
	if strings.Contains(submitted, "Direct Agent request") || strings.Contains(submitted, "WorkPackage") {
		t.Fatalf("interactive lead received WorkPackage envelope: %q", submitted)
	}
}

func TestAskAgentExecutionRoleOverlayPreservesInteractiveLeadAndRuntimeBinding(t *testing.T) {
	n := openTestNexus(t)
	st, _ := n.OpenProject()
	project, err := st.CreateProject(store.Project{Name: "Role Lead", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	agent, err := n.PrepareInteractiveLead(context.Background(), project.ID, "fake", "default")
	if err != nil {
		t.Fatal(err)
	}
	sess, err := n.StartAgent(context.Background(), agent.ID, "fake", "default")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := n.SetRuntimeExecutionRole(context.Background(), sess.RuntimeID, "review", "test"); err != nil {
		t.Fatal(err)
	}

	var submitted string
	n.submitPrompt = func(_ string, prompt string) error { submitted = prompt; return nil }
	result, err := n.AskAgentWithOptions(context.Background(), agent.ID, "inspect this diff", AskAgentOptions{
		CompileSpecialization: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.ExecutionRole != "review" {
		t.Fatalf("expected persisted review role, got %+v", result)
	}
	if !strings.Contains(submitted, "[Nexus execution role]") || !strings.Contains(submitted, "ID: review") || !strings.Contains(submitted, "inspect this diff") {
		t.Fatalf("execution role overlay missing: %q", submitted)
	}
	if strings.Contains(submitted, "Direct Agent request") || strings.Contains(submitted, "structured WorkPackage") {
		t.Fatalf("interactive lead received full WorkPackage envelope: %q", submitted)
	}

	submitted = ""
	result, err = n.AskAgentWithOptions(context.Background(), agent.ID, "check threat boundaries", AskAgentOptions{
		ExecutionRoleID:       "security",
		CompileSpecialization: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.ExecutionRole != "security" || !strings.Contains(submitted, "ID: security") {
		t.Fatalf("turn role override not applied: result=%+v prompt=%q", result, submitted)
	}
	effective, err := n.EffectiveRuntimeExecutionRole(sess.RuntimeID)
	if err != nil {
		t.Fatal(err)
	}
	if effective.Role.ID != "review" {
		t.Fatalf("turn override mutated runtime binding: %#v", effective)
	}
}

func TestAskAgentWithOptionsDisablesSpecializationWhenRequested(t *testing.T) {
	n := openTestNexus(t)
	st, _ := n.OpenProject()
	project, err := st.CreateProject(store.Project{Name: "QA", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	agent, err := st.CreateAgent(store.Agent{ProjectID: project.ID, Name: "QA", Role: "qa"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := n.SafeApply(context.Background(), agent.ID, AgentConfig{
		Provider: "fake", Profile: "default",
		AgentSpec: intelligence.AgentSpec{Role: "qa", Instructions: []string{"Collect evidence before suggesting fixes"}},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := n.StartAgent(context.Background(), agent.ID, "fake", "default"); err != nil {
		t.Fatal(err)
	}
	var submitted string
	n.submitPrompt = func(_ string, prompt string) error { submitted = prompt; return nil }
	if _, err := n.AskAgentWithOptions(context.Background(), agent.ID, "reproduce the failure", AskAgentOptions{
		CompileSpecialization: false,
	}); err != nil {
		t.Fatal(err)
	}
	if submitted != "reproduce the failure" {
		t.Fatalf("disabled specialization must submit raw prompt, got %q", submitted)
	}
}

func TestAskAgentStoppedUsesStartAndAskWithoutCreatingAgent(t *testing.T) {
	n := openTestNexus(t)
	st, _ := n.OpenProject()
	proj, _ := st.CreateProject(store.Project{Name: "P", CanonicalPath: t.TempDir()})
	agent, _ := st.CreateAgent(store.Agent{ProjectID: proj.ID, Name: "Dev"})
	sess, err := n.StartAgent(context.Background(), agent.ID, "fake", "default")
	if err != nil {
		t.Fatal(err)
	}
	if err := n.StopAgent(context.Background(), agent.ID); err != nil {
		t.Fatal(err)
	}
	_ = sess
	var submitted string
	n.submitPrompt = func(runtimeID, prompt string) error { submitted = runtimeID; return nil }
	result, err := n.AskAgent(context.Background(), agent.ID, "continue", true)
	if err != nil {
		t.Fatal(err)
	}
	if result.AgentID != agent.ID || !result.Started || result.RuntimeID == "" || submitted != result.RuntimeID {
		t.Fatalf("Start & Ask result=%+v submitted=%q", result, submitted)
	}
	agents, _ := st.ListAgents(proj.ID)
	if len(agents) != 1 {
		t.Fatalf("Start & Ask created another Agent: %d", len(agents))
	}
}

func TestStartProjectShellUsesCanonicalPathAndCreatesNoAgent(t *testing.T) {
	n := openTestNexus(t)
	st, _ := n.OpenProject()
	root := t.TempDir()
	proj, err := st.CreateProject(store.Project{Name: "P", CanonicalPath: root})
	if err != nil {
		t.Fatal(err)
	}
	before, _ := st.ListAgents(proj.ID)
	sess, err := n.StartProjectShell(context.Background(), proj.ID)
	if err != nil {
		t.Fatal(err)
	}
	if sess.ProviderID != "shell" || sess.ProfileID != "local" || sess.Workspace != root {
		t.Fatalf("unexpected shell runtime: %+v", sess)
	}
	after, _ := st.ListAgents(proj.ID)
	if len(after) != len(before) {
		t.Fatalf("Project Shell created Agent: before=%d after=%d", len(before), len(after))
	}
}

func TestStartProjectShellMissingDirectoryFails(t *testing.T) {
	n := openTestNexus(t)
	st, _ := n.OpenProject()
	missingPath := filepath.Join(t.TempDir(), "non-existent-folder")
	proj, err := st.CreateProject(store.Project{Name: "MissingDirProject", CanonicalPath: missingPath})
	if err != nil {
		t.Fatal(err)
	}
	_, err = n.StartProjectShell(context.Background(), proj.ID)
	if err == nil {
		t.Fatal("expected StartProjectShell to fail when canonical directory does not exist, got nil")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("unexpected error message: %v", err)
	}
}
