package nexus

import (
	"context"
	"errors"
	"testing"

	"github.com/IAPro-Community/nexus/internal/control/launcher"
	"github.com/IAPro-Community/nexus/internal/control/registry"
	"github.com/IAPro-Community/nexus/internal/nexus/runner"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

type countingMissionLauncher struct {
	launchCalls int
	launchErr   error
}

func (l *countingMissionLauncher) Launch(context.Context, launcher.LaunchOptions) (*registry.RuntimeSession, error) {
	l.launchCalls++
	if l.launchErr != nil {
		return nil, l.launchErr
	}
	return nil, errors.New("fixture launcher must not be reached")
}

func (*countingMissionLauncher) Stop(string) error { return nil }

func TestMissionExecutorClassifiesSetupFailureBeforeLauncherAsNotDispatched(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "Pre-dispatch boundary", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	agent, err := st.CreateAgent(store.Agent{ProjectID: project.ID, Name: "Unconfigured fixture agent"})
	if err != nil {
		t.Fatal(err)
	}
	launch := &countingMissionLauncher{}
	n.launcher = launch

	executor := newNexusPackageExecutor(n).(*nexusPackageExecutor)
	_, executeErr := executor.Execute(context.Background(), &runner.MissionRun{
		ID: "run-pre-dispatch", ProjectID: project.ID,
	}, &runner.PackageRun{
		PackageID: "package-pre-dispatch", AssignedAgent: agent.ID, Workspace: t.TempDir(),
	}, "fixture prompt")
	if executeErr == nil {
		t.Fatal("expected local provider-configuration failure")
	}
	if !errors.Is(executeErr, runner.ErrDispatchNotSent) {
		t.Fatalf("pre-launch setup failure must be proven not sent; got %v", executeErr)
	}
	if launch.launchCalls != 0 {
		t.Fatalf("setup failure crossed the provider launch boundary %d times", launch.launchCalls)
	}
}

func TestMissionExecutorKeepsLauncherFailuresAsUnknownDispatch(t *testing.T) {
	seedCodexExecutionFixture(t)
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "Ambiguous launch boundary", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	agent, err := st.CreateAgent(store.Agent{ProjectID: project.ID, Name: "Configured fixture agent"})
	if err != nil {
		t.Fatal(err)
	}
	revision, err := st.AddRevision(agent.ID, (AgentConfig{Provider: "codex", Profile: "fixture"}).ConfigJSON())
	if err != nil {
		t.Fatal(err)
	}
	agent.CurrentRevisionID = revision.ID
	if err := st.UpdateAgent(agent); err != nil {
		t.Fatal(err)
	}
	launch := &countingMissionLauncher{launchErr: errors.New("fixture launcher handshake failed")}
	n.launcher = launch

	executor := newNexusPackageExecutor(n).(*nexusPackageExecutor)
	_, executeErr := executor.Execute(context.Background(), &runner.MissionRun{
		ID: "run-ambiguous-launch", ProjectID: project.ID,
		Contract: runner.DefaultAutonomyContract(),
	}, &runner.PackageRun{
		PackageID: "package-ambiguous-launch", AssignedAgent: agent.ID, Workspace: t.TempDir(),
	}, "fixture prompt")
	if executeErr == nil {
		t.Fatal("expected launcher failure")
	}
	if errors.Is(executeErr, runner.ErrDispatchNotSent) {
		t.Fatalf("launcher failure must remain ambiguous after entering the launch boundary: %v", executeErr)
	}
	if launch.launchCalls != 1 {
		t.Fatalf("launcher calls=%d, want 1", launch.launchCalls)
	}
}
