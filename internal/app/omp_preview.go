package app

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/launcher"
	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/profile"
)

// OMP is intentionally a guarded preview until #98 has authenticated
// end-to-end coverage. Do not route it through the regular provider scheduler:
// its account identity, quota and failover contracts are not verified yet.
func executeOMPPreview(explicitProfile string, args []string) error {
	if os.Getenv("NEXUS_EXPERIMENTAL_HARNESSES") != "1" {
		return fmt.Errorf("Oh My Pi preview is disabled; set NEXUS_EXPERIMENTAL_HARNESSES=1 to opt in (see issue #98)")
	}
	if unsupportedOMPLaunchFlag(args) {
		return fmt.Errorf("Oh My Pi preview supports supervised PTY only; --direct, --lead and --yolo are not certified")
	}
	args = removeProviderLaunchFlag(args, "--supervised")
	args = removeProviderLaunchFlag(args, "--no-lead")

	profiles, err := profile.List()
	if err != nil {
		return err
	}
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}
	selected, err := selectOMPPreviewProfile(profiles, explicitProfile, cfg.Defaults["omp"])
	if err != nil {
		return err
	}

	session, err := launcher.Default().Launch(context.Background(), launcher.LaunchOptions{
		ProviderID: "omp",
		ProfileID:  selected,
		Workspace:  mustWorkingDirectory(),
		Args:       args,
		Standalone: false,
		Timeout:    15 * time.Second,
		Labels:     supervisedRuntimeLabels(),
	})
	if err != nil {
		return fmt.Errorf("supervised Oh My Pi preview launch failed: %w", err)
	}
	fmt.Fprintf(os.Stderr, "[nexus] experimental OMP:%s runtime=%s (PTY; structured approvals unverified)\n", selected, session.RuntimeID)
	return attachRuntime(session.RuntimeID)
}

func unsupportedOMPLaunchFlag(args []string) bool {
	for _, arg := range args {
		switch arg {
		case "--direct", "--lead", "--yolo", "-y":
			return true
		}
	}
	return false
}

// Unlike Codex, OMP cannot safely enter auto-account selection until its
// identity-scoped usage evidence is implemented. A default or a single enabled
// profile is deterministic; ambiguity requires explicit user selection.
func selectOMPPreviewProfile(profiles []model.Profile, requested, configuredDefault string) (string, error) {
	if requested == "auto" {
		requested = ""
	}
	var enabled []string
	for _, p := range profiles {
		if p.Provider == "omp" && !p.Disabled {
			enabled = append(enabled, p.Name)
		}
	}
	if len(enabled) == 0 {
		return "", fmt.Errorf("no enabled OMP profiles; run: nexus add omp default --no-login")
	}
	selected := strings.TrimSpace(requested)
	if selected == "" {
		selected = configuredDefault
	}
	if selected == "" && len(enabled) == 1 {
		selected = enabled[0]
	}
	if selected == "" {
		return "", fmt.Errorf("multiple OMP profiles: specify nexus omp:<profile> (automatic quota-based selection is not verified)")
	}
	for _, name := range enabled {
		if name == selected {
			return selected, nil
		}
	}
	return "", fmt.Errorf("OMP profile %q is missing or disabled", selected)
}

func showOMPPreviewHelp() {
	fmt.Println(`Oh My Pi (OMP) — experimental supervised terminal preview
  NEXUS_EXPERIMENTAL_HARNESSES=1 nexus add omp default --no-login
  NEXUS_EXPERIMENTAL_HARNESSES=1 nexus omp:default [native omp flags]
  NEXUS_EXPERIMENTAL_HARNESSES=1 nexus omp default [native omp flags]

Authenticate interactively inside the isolated OMP profile on first launch.
ACP/RPC require an explicit experimental agent_protocol selection in Nexus Web.
No auto account switching, remote approvals or authenticated E2E certification yet.
See https://github.com/IAPro-Community/nexus/issues/98`)
}
