package driver

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/runtime"
)

// BuildWireCommand selects an explicit provider-native structured entrypoint.
// It is not a transparent conversion of an existing interactive TUI session.
// No flags or model overrides are silently discarded.
func BuildWireCommand(ctx context.Context, d ControlDriver, p model.Profile, mode string, baseArgs []string) (string, []string, []string, error) {
	if len(baseArgs) > 0 {
		return "", nil, nil, fmt.Errorf("structured mode cannot accept interactive launch flags")
	}
	var args []string
	switch mode {
	case "acp":
		switch d.ProviderID() {
		case "opencode", "omp", "mimo", "dsh":
			args = []string{"acp"}
		default:
			return "", nil, nil, fmt.Errorf("provider %s has no declared ACP entrypoint", d.ProviderID())
		}
	case "pi_rpc":
		switch d.ProviderID() {
		case "pi", "omp":
			args = []string{"--mode", "rpc"}
		default:
			return "", nil, nil, fmt.Errorf("provider %s has no declared Pi RPC entrypoint", d.ProviderID())
		}
	default:
		return "", nil, nil, fmt.Errorf("unsupported agent protocol %q", mode)
	}
	if d.ProviderID() != "dsh" {
		bin, _, env, err := d.BuildCommand(ctx, p, nil)
		return bin, args, env, err
	}
	// DSH has no supported interactive PTY in this release. Its ACP launch is
	// allowed only after the user explicitly opts into the experimental harness.
	bin, err := runtime.LookPath("dsh")
	if err != nil {
		return "", nil, nil, err
	}
	home, err := config.ProfileHome("dsh", p.Name)
	if err != nil {
		return "", nil, nil, err
	}
	if err := os.MkdirAll(home, 0700); err != nil {
		return "", nil, nil, err
	}
	env := append([]string{}, os.Environ()...)
	env = append(env, "HOME="+home, "USERPROFILE="+home,
		"XDG_CONFIG_HOME="+filepath.Join(home, ".config"),
		"AI_PROVIDER=dsh", "AI_PROFILE="+strings.TrimSpace(p.Name))
	return bin, args, env, nil
}
