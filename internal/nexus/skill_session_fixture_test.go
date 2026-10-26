package nexus

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/launcher"
	"github.com/IAPro-Community/nexus/internal/control/registry"
	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/nexus/runner"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

type standaloneSkillFixtureLauncher struct {
	inner *launcher.Launcher
}

func (l standaloneSkillFixtureLauncher) Launch(ctx context.Context, opts launcher.LaunchOptions) (*registry.RuntimeSession, error) {
	opts.Standalone = true
	opts.Timeout = 5 * time.Second
	return l.inner.Launch(ctx, opts)
}

func (l standaloneSkillFixtureLauncher) Stop(runtimeID string) error {
	return l.inner.Stop(runtimeID)
}

func TestSelectedSkillsReachAnIsolatedProviderSession(t *testing.T) {
	tests := []struct {
		name           string
		prompt         string
		explicitSkills []string
	}{
		{
			name:           "explicit skill",
			prompt:         "Validate this fixture request and report the checks performed.",
			explicitSkills: []string{"testing"},
		},
		{
			name:   "automatic skills",
			prompt: "Review this code for security issues and verify that the input validation tests cover invalid values.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			hostHome := filepath.Join(root, "host-home")
			binDir := filepath.Join(root, "bin")
			runtimeDir := filepath.Join(root, "runtime")
			capturePath := filepath.Join(root, "provider-args.bin")
			environmentPath := filepath.Join(root, "provider-environment.txt")
			for _, dir := range []string{hostHome, binDir, runtimeDir, filepath.Join(root, "config")} {
				if err := os.MkdirAll(dir, 0o700); err != nil {
					t.Fatalf("create isolated test directory: %v", err)
				}
			}

			t.Setenv("HOME", hostHome)
			t.Setenv("NEXUS_REAL_HOME", hostHome)
			t.Setenv("NEXUS_CONFIG_DIR", filepath.Join(root, "config"))
			t.Setenv("XDG_RUNTIME_DIR", runtimeDir)
			t.Setenv("NEXUS_SKILL_FIXTURE_CAPTURE", capturePath)
			t.Setenv("NEXUS_SKILL_FIXTURE_ENV", environmentPath)
			t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

			fixtureCLI := "#!/bin/sh\n" +
				"if [ \"$1\" = \"--version\" ]; then printf '%s\\n' 'codex fixture 1.0'; exit 0; fi\n" +
				"printf '%s\\0' \"$@\" > \"$NEXUS_SKILL_FIXTURE_CAPTURE\"\n" +
				"printf 'HOME=%s\\nCODEX_HOME=%s\\n' \"$HOME\" \"$CODEX_HOME\" > \"$NEXUS_SKILL_FIXTURE_ENV\"\n" +
				"printf '%s\\n' 'FIXTURE_SKILL_SESSION_COMPLETED'\n" +
				"sleep 1\n"
			if err := os.WriteFile(filepath.Join(binDir, "codex"), []byte(fixtureCLI), 0o700); err != nil {
				t.Fatalf("write fake provider CLI: %v", err)
			}
			resolvedCLI, err := exec.LookPath("codex")
			if err != nil || resolvedCLI != filepath.Join(binDir, "codex") {
				t.Fatalf("runtime would not resolve to the isolated fixture CLI: path=%q err=%v", resolvedCLI, err)
			}

			n := openTestNexus(t)
			n.maestroStatus = func() MaestroStatus { return MaestroStatus{Available: false, Mode: MaestroOff} }
			n.launcher = standaloneSkillFixtureLauncher{inner: launcher.NewLauncher()}
			st, err := n.OpenProject()
			if err != nil {
				t.Fatalf("open isolated Nexus store: %v", err)
			}
			workspace := filepath.Join(root, "workspace")
			if err := os.MkdirAll(workspace, 0o700); err != nil {
				t.Fatalf("create isolated workspace: %v", err)
			}
			project, err := st.CreateProject(store.Project{Name: "Skill session fixture", CanonicalPath: workspace})
			if err != nil {
				t.Fatalf("create fixture project: %v", err)
			}
			agent, err := st.CreateAgent(store.Agent{ProjectID: project.ID, Name: "Fixture Reviewer", Role: "reviewer"})
			if err != nil {
				t.Fatalf("create fixture Agent: %v", err)
			}
			agentConfig := AgentConfig{Provider: "codex", Profile: "skill-fixture", Workspace: workspace}
			revision, err := st.AddRevision(agent.ID, agentConfig.ConfigJSON())
			if err != nil {
				t.Fatalf("persist fixture Agent config: %v", err)
			}
			agent.CurrentRevisionID = revision.ID
			if err := st.UpdateAgent(agent); err != nil {
				t.Fatalf("select fixture Agent revision: %v", err)
			}

			skillIDs := append([]string(nil), tt.explicitSkills...)
			if len(skillIDs) == 0 {
				skillIDs, err = n.ResolveAutomaticSkillIDs(context.Background(), project.ID, tt.prompt)
				if err != nil {
					t.Fatalf("resolve automatic skills: %v", err)
				}
				if len(skillIDs) == 0 || len(skillIDs) > maxAutomaticSkills {
					t.Fatalf("automatic skill selection is empty or exceeds its bound: %v", skillIDs)
				}
			}
			compiled, err := n.CompileAgentPromptForProject(project.ID, tt.prompt, skillIDs)
			if err != nil {
				t.Fatalf("compile selected skill contracts: %v", err)
			}
			if len(compiled.SkillContracts) != len(skillIDs) {
				t.Fatalf("compiled contracts do not match selected skills: selected=%v contracts=%v", skillIDs, compiled.SkillContracts)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			execution, err := n.executeAgentPrompt(ctx, agent.ID, workspace, compiled.CompiledPrompt, agentPromptPolicy{
				Contract: runner.DefaultAutonomyContract(),
				Review:   true,
			})
			if err != nil {
				t.Fatalf("execute isolated fixture session: %v", err)
			}
			if execution.RuntimeID == "" {
				t.Fatal("fixture session did not persist a runtime identity")
			}
			if generation, err := st.CurrentGeneration(agent.ID); err != nil {
				t.Fatalf("load completed runtime generation: %v", err)
			} else if generation.State != "STOPPED" {
				t.Fatalf("completed fixture runtime generation is not stopped: %q", generation.State)
			}
			if persisted, err := st.GetAgent(agent.ID, ""); err != nil {
				t.Fatalf("load fixture Agent state: %v", err)
			} else if persisted.Status != store.AgentStopped {
				t.Fatalf("fixture Agent did not return to STOPPED: %q", persisted.Status)
			}
			if session, ok := registry.DefaultRegistry().Get(execution.RuntimeID); ok && session.IsActive() {
				t.Fatalf("fixture provider runtime remained active after completion: %#v", session)
			}
			if !strings.Contains(execution.Output, "FIXTURE_SKILL_SESSION_COMPLETED") {
				t.Fatalf("fixture output did not reach the Nexus session: %q", execution.Output)
			}
			captured, err := os.ReadFile(capturePath)
			if err != nil {
				t.Fatalf("read fixture CLI argument capture: %v", err)
			}
			args := strings.Split(strings.TrimSuffix(string(captured), "\x00"), "\x00")
			capturedArgs := strings.Join(args, "\x00")
			if !strings.Contains(capturedArgs, "--sandbox\x00read-only") || !strings.Contains(capturedArgs, "--ask-for-approval\x00never") {
				t.Fatalf("fixture Agent did not receive Codex read-only review constraints: %q", args)
			}
			promptFound := false
			for _, arg := range args {
				if arg == compiled.CompiledPrompt {
					promptFound = true
					break
				}
			}
			if !promptFound {
				t.Fatalf("supervised provider session did not receive the compiled prompt; args=%q", args)
			}
			if !strings.Contains(compiled.CompiledPrompt, "Validated Nexus skills:") {
				t.Fatalf("compiled prompt omitted the canonical Nexus skill envelope: %q", compiled.CompiledPrompt)
			}
			for _, skillID := range skillIDs {
				if !strings.Contains(compiled.CompiledPrompt, "- "+skillID+" (source: ") || !strings.Contains(compiled.CompiledPrompt, "--- Skill contract: "+skillID+" ---") {
					t.Errorf("selected skill %q was not carried into the provider prompt: %q", skillID, compiled.CompiledPrompt)
				}
			}

			profileHome, err := config.ProfileHome("codex", "skill-fixture")
			if err != nil {
				t.Fatalf("resolve isolated provider profile: %v", err)
			}
			if _, err := os.Stat(filepath.Join(profileHome, "auth.json")); !os.IsNotExist(err) {
				t.Fatalf("fixture session unexpectedly created or copied provider credentials: %v", err)
			}
			capturedEnvironment, err := os.ReadFile(environmentPath)
			if err != nil {
				t.Fatalf("read fixture CLI environment capture: %v", err)
			}
			expectedHome := "HOME=" + profileHome
			if !strings.Contains(string(capturedEnvironment), expectedHome+"\n") || !strings.Contains(string(capturedEnvironment), "CODEX_HOME="+profileHome+"\n") {
				t.Fatalf("fixture session did not use its isolated provider home: %q", capturedEnvironment)
			}
		})
	}
}
