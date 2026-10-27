package driver

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/IAPro-Community/nexus/internal/control/registry"
	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/runtime"
)

// PreviewHarnessDriver supports supervised interactive PTY launch for new
// harnesses. It deliberately reports structured control as UNSUPPORTED until
// the agentwire clients are connected to SessionHost lifecycle and permissions.
// Never launch ACP/JSONL into a PTY (protocol output is not a terminal).
type PreviewHarnessDriver struct{ id string }

func NewPreviewHarnessDriver(id string) *PreviewHarnessDriver {
	switch id {
	case "pi", "omp", "mimo", "dsh":
		return &PreviewHarnessDriver{id: id}
	}
	panic("unrecognized preview harness")
}
func (d *PreviewHarnessDriver) ProviderID() string { return d.id }
func (d *PreviewHarnessDriver) Detect(ctx context.Context) (model.DetectionResult, error) {
	binary, err := runtime.LookPath(d.id)
	if err != nil {
		return model.DetectionResult{Installed: false, Error: d.id + " executable not found"}, nil
	}
	output, err := runtime.RunCommandCapture(ctx, binary, []string{"--version"}, os.Environ(), "")
	if err != nil {
		return model.DetectionResult{Installed: false, BinaryPath: binary, Error: strings.TrimSpace(output)}, nil
	}
	lines := strings.Split(strings.TrimSpace(output), "\n")
	return model.DetectionResult{Installed: true, BinaryPath: binary, Version: lines[0]}, nil
}
func (d *PreviewHarnessDriver) EffectiveCaps(ctx context.Context, _ model.Profile) EffectiveCapabilities {
	detected, _ := d.Detect(ctx)
	evidence := CapabilityEvidence{Status: CapabilitySupported, ProviderVersion: detected.Version, Mechanism: "supervised PTY / ConPTY", Tested: false}
	no := CapabilityEvidence{Status: CapabilityUnsupported, ProviderVersion: detected.Version, Reason: "not connected to SessionHost structured control", Tested: false}
	terminal := evidence
	if d.id == "dsh" {
		terminal = CapabilityEvidence{Status: CapabilityUnsupported, Reason: "DSH uses ACP/SDK; interactive TUI launch has not been validated", Tested: false}
	}
	return EffectiveCapabilities{
		Process:          evidence,
		Terminal:         terminal,
		Attach:           evidence,
		StructuredEvents: no,
		Approvals:        no,
		Sessions: CapabilityEvidence{
			Status: CapabilityPartial, Reason: "session semantics require provider E2E validation", Tested: false,
		},
		Resume: CapabilityEvidence{
			Status: CapabilityPartial, ProviderVersion: detected.Version,
			Reason: "CLI resume syntax documented; requires live session E2E", Tested: false,
		},
		Fork:         no,
		SubmitPrompt: no,
		CancelTurn:   no,
		Headless:     no,
		SlashControl: no,
		ControlLevel: registry.ControlLevelTerminal,
	}
}
func (d *PreviewHarnessDriver) Capabilities(ctx context.Context, p model.Profile) ControlCapabilities {
	return d.EffectiveCaps(ctx, p).ToBooleanCaps()
}
func (d *PreviewHarnessDriver) BuildCommand(ctx context.Context, p model.Profile, args []string) (string, []string, []string, error) {
	if d.id == "dsh" {
		return "", nil, nil, fmt.Errorf("dsh requires ACP controlled execution; not safe to present as interactive PTY")
	}
	bin, err := runtime.LookPath(d.id)
	if err != nil {
		return "", nil, nil, err
	}
	home, err := config.ProfileHome(d.id, p.Name)
	if err != nil {
		return "", nil, nil, err
	}
	if err := os.MkdirAll(home, 0700); err != nil {
		return "", nil, nil, err
	}
	env := append([]string{}, os.Environ()...)
	env = append(env, "HOME="+home, "USERPROFILE="+home, "AI_PROVIDER="+d.id, "AI_PROFILE="+p.Name)
	if d.id == "pi" || d.id == "omp" {
		env = append(env, "PI_CODING_AGENT_DIR="+filepath.Join(home, ".pi", "agent"))
	}
	if d.id == "mimo" {
		env = append(env,
			"XDG_CONFIG_HOME="+filepath.Join(home, ".config"),
			"XDG_DATA_HOME="+filepath.Join(home, ".local", "share"),
			"XDG_CACHE_HOME="+filepath.Join(home, ".cache"))
	}
	return bin, args, env, nil
}
func (d *PreviewHarnessDriver) CanResume(ctx context.Context, p model.Profile, sessionID string) (bool, string) {
	if strings.TrimSpace(sessionID) == "" {
		return false, "missing session identifier"
	}
	if d.id == "dsh" {
		return false, "DSH needs structured ACP resume"
	}
	return true, "CLI resume syntax is available but not E2E-verified"
}
func (d *PreviewHarnessDriver) BuildResumeArgs(ctx context.Context, p model.Profile, sessionID string) ([]string, error) {
	if ok, _ := d.CanResume(ctx, p, sessionID); !ok {
		return nil, fmt.Errorf("missing or unsupported %s session", d.id)
	}
	switch d.id {
	case "pi", "mimo":
		return []string{"--session", sessionID}, nil
	case "omp":
		return []string{"--resume", sessionID}, nil
	}
	return nil, fmt.Errorf("%s resume unsupported", d.id)
}
func (d *PreviewHarnessDriver) BuildKickoffArgs(ctx context.Context, p model.Profile, prompt string) ([]string, error) {
	switch d.id {
	case "pi", "omp":
		return []string{"-p", prompt}, nil
	case "mimo":
		return []string{"run", prompt}, nil
	}
	return nil, fmt.Errorf("%s structured kickoff not certified", d.id)
}
