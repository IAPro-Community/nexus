package nexus

import (
	"context"
	"strings"
	"testing"

	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func TestExecuteAgentPromptFailsClosedForManagedRemoteBinding(t *testing.T) {
	t.Setenv("NEXUS_ENABLE_REMOTE_EXECUTION", "1")

	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	projectDir := t.TempDir()
	project, err := st.CreateProject(store.Project{
		Name:          "Remote Mission Guard",
		CanonicalPath: projectDir,
	})
	if err != nil {
		t.Fatal(err)
	}
	agent, err := st.CreateAgent(store.Agent{
		ProjectID: project.ID,
		Name:      "Mission Remote",
	})
	if err != nil {
		t.Fatal(err)
	}
	revision, err := st.AddRevision(agent.ID, AgentConfig{
		Provider: "fake",
		Profile:  "default",
	}.ConfigJSON())
	if err != nil {
		t.Fatal(err)
	}
	agent.CurrentRevisionID = revision.ID
	if err := st.UpdateAgent(agent); err != nil {
		t.Fatal(err)
	}
	if err := st.SetProjectSetting(project.ID, managedExecutionTargetSettingKey, ManagedExecutionTargetBinding{
		TargetID:      "devbox",
		ApprovalToken: "approved",
		MachineID:     "machine-devbox",
	}); err != nil {
		t.Fatal(err)
	}

	result, err := n.executeAgentPrompt(
		context.Background(),
		agent.ID,
		projectDir,
		"make a change",
		agentPromptPolicy{},
	)
	if result != nil {
		t.Fatalf("managed remote Mission guard must fail before launching, got %+v", result)
	}
	if err == nil || !strings.Contains(err.Error(), "managed remote Mission execution is not enabled") {
		t.Fatalf("expected explicit managed remote Mission guard, got %v", err)
	}
}
