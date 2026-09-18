package intelligence

import (
	"context"
	"testing"
)

func TestCompileExecutionContext_ComposesPersistentAndTaskRolesWithProvenance(t *testing.T) {
	compiled, err := NewNexusEngine(nil).CompileExecutionContext(context.Background(), ExecutionContextRequest{
		Agent: AgentSpec{
			Role:         "senior developer",
			Instructions: []string{"Prefer small reversible changes"},
			Domains:      []string{"software-engineering"},
			Strengths:    []string{"systems-thinking"},
			Tags:         []string{"persistent-specialist"},
		},
		Project:  ProjectContext{ProjectID: "project-1", Facts: map[string]string{"branch": "main"}},
		Task:     WorkPackageContext{Title: "Review release", Goal: "Review the release", Priority: "HIGH", Role: "reviewer", AcceptanceCriteria: []string{"evidence exists"}},
		Guidance: ExecutionGuidance{Enabled: true, Instructions: []string{"Use the configured quality gate"}, Source: "maestro"},
		Runtime:  RuntimeConstraints{Provider: "claude", Model: "sonnet", Workspace: "/workspace", Isolation: "worktree", Capabilities: []string{"submit_prompt"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !containsSection(compiled.Sections, "agent", "senior developer") || !containsSection(compiled.Sections, "task", "reviewer") {
		t.Fatalf("missing role provenance: %#v", compiled.Sections)
	}
	if !containsSection(compiled.Sections, "guidance", "quality gate") {
		t.Fatalf("missing generic guidance provenance: %#v", compiled.Sections)
	}
	if compiled.SystemInstructions == "" || compiled.TaskInstructions == "" || !contains(compiled.SystemInstructions, "Persistent specialization: senior developer") || !contains(compiled.SystemInstructions, "systems-thinking") || !contains(compiled.SystemInstructions, "persistent-specialist") || !contains(compiled.SystemInstructions, "Task role: reviewer") {
		t.Fatalf("compiled context is incomplete: %#v", compiled)
	}
	if !containsSection(compiled.Sections, "runtime", "/workspace") || !containsSection(compiled.Sections, "runtime", "worktree") {
		t.Fatalf("runtime workspace/isolation constraints were not preserved: %#v", compiled.Sections)
	}
}

func TestCompileExecutionContext_MaestroOffDoesNotAddGuidance(t *testing.T) {
	compiled, err := NewNexusEngine(nil).CompileExecutionContext(context.Background(), ExecutionContextRequest{
		Agent: AgentSpec{Role: "qa"},
		Task:  WorkPackageContext{Title: "Direct", Goal: "Run checks", Role: "reviewer"},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, section := range compiled.Sections {
		if section.Source == "maestro" {
			t.Fatalf("Maestro OFF unexpectedly changed Direct context: %#v", compiled.Sections)
		}
	}
}

func TestCompileExecutionContextUsesGenericGuidanceWithoutMaestro(t *testing.T) {
	compiled, err := NewNexusEngine(nil).CompileExecutionContext(context.Background(), ExecutionContextRequest{
		Agent: AgentSpec{Role: "tester"},
		Task:  WorkPackageContext{Title: "Verify", Goal: "Verify the change"},
		Guidance: ExecutionGuidance{
			Enabled:      true,
			Instructions: []string{"Use reproducible checks"},
			Skills:       []string{"verification"},
			Source:       "nexus",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !containsSection(compiled.Sections, "guidance", "reproducible checks") || !containsSection(compiled.Sections, "guidance", "verification") {
		t.Fatalf("generic guidance was not compiled: %#v", compiled.Sections)
	}
	for _, section := range compiled.Sections {
		if section.Source == "maestro" {
			t.Fatalf("generic guidance leaked Maestro provenance: %#v", compiled.Sections)
		}
	}
}

func TestCompileExecutionContext_DevOpsSpecializationDiffersFromQA(t *testing.T) {
	engine := NewNexusEngine(nil)
	qa, err := engine.CompileExecutionContext(context.Background(), ExecutionContextRequest{
		Agent: AgentSpec{Role: "qa", Instructions: []string{"Reproduce failures before suggesting fixes"}, Capabilities: []string{"test", "review"}},
		Task:  WorkPackageContext{Title: "Validate release", Goal: "Validate the release"},
	})
	if err != nil {
		t.Fatal(err)
	}
	devOps, err := engine.CompileExecutionContext(context.Background(), ExecutionContextRequest{
		Agent: AgentSpec{Role: "devops", Instructions: []string{"Protect deployment reliability"}, Capabilities: []string{"deploy", "observe"}},
		Task:  WorkPackageContext{Title: "Validate release", Goal: "Validate the release"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if contains(qa.SystemInstructions, "Protect deployment reliability") || !contains(devOps.SystemInstructions, "Protect deployment reliability") || qa.SystemInstructions == devOps.SystemInstructions {
		t.Fatalf("QA and DevOps specializations were not composed independently: qa=%q devops=%q", qa.SystemInstructions, devOps.SystemInstructions)
	}
}

func containsSection(sections []ContextSection, source, text string) bool {
	for _, section := range sections {
		if section.Source == source && contains(section.Content, text) {
			return true
		}
	}
	return false
}

func contains(value, needle string) bool {
	for i := 0; i+len(needle) <= len(value); i++ {
		if value[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
