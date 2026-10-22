package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/launcher"
	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/cooldown"
	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/core/provider/adapters/codex"
	"github.com/IAPro-Community/nexus/internal/core/quota"
	"github.com/IAPro-Community/nexus/internal/core/scheduler"
	"github.com/IAPro-Community/nexus/internal/nexus"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
	"github.com/IAPro-Community/nexus/internal/profile"
)

// hasProviderLaunchFlag recognizes Nexus-owned launch flags before provider
// arguments are normalized. Provider CLIs must never receive these flags.
func hasProviderLaunchFlag(args []string, wanted string) bool {
	for _, arg := range args {
		if strings.EqualFold(strings.TrimSpace(arg), wanted) {
			return true
		}
	}
	return false
}

func removeProviderLaunchFlag(args []string, wanted string) []string {
	out := make([]string, 0, len(args))
	for _, arg := range args {
		if strings.EqualFold(strings.TrimSpace(arg), wanted) {
			continue
		}
		out = append(out, arg)
	}
	return out
}

// supervisedRuntimeLabels keeps runtime supervision, Lead identity and Mission
// authorization separate. The runtime is registered with Lead=false first;
// UpdateInteractiveLeadBinding flips the label only after the persistent Lead
// metadata has been committed successfully. Delegation always starts OFF.
func supervisedRuntimeLabels() map[string]string {
	return map[string]string{
		"nexus.delegation_mode":  string(nexus.DelegationOff),
		"nexus.interactive_lead": "false",
	}
}

// executeProviderSupervised launches the provider through the same persistent
// SessionHost used by Web/Desktop. STICKY_LEAD binds identity/continuity while
// delegation remains OFF, so the provider TTY stays byte-transparent. When the
// binding is implicit it is best-effort; explicit --lead makes binding failure
// visible to the caller.
func executeProviderSupervised(provName, explicitProfile string, args []string, withLead, requireLead bool) error {
	if strings.TrimSpace(provName) == "" {
		return fmt.Errorf("provider is required for supervised launch")
	}
	args = removeProviderLaunchFlag(args, "--lead")
	args = removeProviderLaunchFlag(args, "--no-lead")

	fmt.Fprintf(os.Stderr, "⚡ [nexus] iniciando %s…\n", provName)

	profiles, err := profile.List()
	if err != nil {
		return err
	}
	var candidates []model.Profile
	accounts := make(map[string]model.AccountInfo)
	quotaEngine := quota.NewEngine(quota.DefaultTTL)
	for _, candidate := range profiles {
		if candidate.Provider != provName || candidate.Disabled {
			continue
		}
		candidates = append(candidates, candidate)
		account := profile.GetAccountInfo(provName, candidate.Name)
		snapshot := profile.PeekUsageSnapshot(provName, candidate.Name)
		account.Usage = snapshot
		accounts[candidate.Name] = account
	}
	if len(candidates) == 0 {
		return fmt.Errorf("no enabled profiles configured for provider %s", provName)
	}

	candidates, err = launchableProfiles(provName, explicitProfile, candidates)
	if err != nil {
		return err
	}

	cfg, _ := config.LoadConfig()
	// The TTY default is supervised. The selection must use the same fresh
	// identity-scoped evidence as the direct launcher, not September cache.
	// Explicit profile launches remain immediate.
	refreshCodexLaunchQuota(context.Background(), provName, explicitProfile, candidates, accounts)
	selector := scheduler.NewSelector(cfg, quotaEngine, cooldown.NewTracker())
	workspace := mustWorkingDirectory()
	session, selected, err := launchWithBusyFailover(
		context.Background(),
		launcher.Default(),
		selector,
		provName,
		explicitProfile,
		workspace,
		cfg.Defaults[provName],
		candidates,
		accounts,
		launcher.LaunchOptions{
			Workspace:  workspace,
			Args:       args,
			Standalone: false,
			Timeout:    15 * time.Second,
			Labels:     supervisedRuntimeLabels(),
		},
		os.Stderr,
	)
	if err != nil {
		return busyOrLaunchError(provName, selected, explicitProfile, err)
	}

	if !withLead {
		fmt.Fprintf(os.Stderr, "⚡ [nexus] supervised %s:%s runtime=%s\n", provName, selected, session.RuntimeID)
		return attachRuntime(session.RuntimeID)
	}

	n := nexus.Default()
	lead, err := prepareInteractiveLead(n, workspace, provName, selected)
	if err != nil {
		if requireLead {
			_ = launcher.Default().Stop(session.RuntimeID)
			return fmt.Errorf("prepare interactive Lead: %w", err)
		}
		fmt.Fprintf(os.Stderr, "⚠ [nexus] STICKY_LEAD indisponível (%v); continuando com TTY provider-native.\n", err)
		return attachRuntime(session.RuntimeID)
	}
	if err := n.RuntimeRegistry().UpdateInteractiveLeadBinding(session.RuntimeID, lead.AgentID, lead.ProjectID, lead.ProjectName); err != nil {
		releaseInteractiveLeadReservation(n, lead.AgentID)
		if requireLead {
			_ = launcher.Default().Stop(session.RuntimeID)
			return fmt.Errorf("persist interactive Lead runtime metadata: %w", err)
		}
		fmt.Fprintf(os.Stderr, "⚠ [nexus] não foi possível persistir STICKY_LEAD (%v); continuando com TTY provider-native.\n", err)
		return attachRuntime(session.RuntimeID)
	}
	session.AgentID = lead.AgentID
	session.ProjectID = lead.ProjectID
	session.ProjectName = lead.ProjectName
	if lead.AgentID != "" {
		if err := n.BindInteractiveLeadRuntime(context.Background(), lead.AgentID, session); err != nil {
			_ = n.RuntimeRegistry().UpdateInteractiveLeadBinding(session.RuntimeID, "", "", "")
			releaseInteractiveLeadReservation(n, lead.AgentID)
			if requireLead {
				_ = launcher.Default().Stop(session.RuntimeID)
				return fmt.Errorf("bind interactive Lead Agent: %w", err)
			}
			fmt.Fprintf(os.Stderr, "⚠ [nexus] bind STICKY_LEAD falhou (%v); continuando com TTY provider-native.\n", err)
			return attachRuntime(session.RuntimeID)
		}
	}
	fmt.Fprintf(os.Stderr, "⚡ [nexus] supervised %s:%s runtime=%s mode=STICKY_LEAD delegation=OFF\n", provName, selected, session.RuntimeID)
	if lead.AgentID == "" {
		return attachRuntime(session.RuntimeID)
	}
	return attachInteractiveLeadRuntime(session.RuntimeID, n, *session, lead.AgentID)
}

func prepareInteractiveLead(n *nexus.Nexus, workspace, provider, profileName string) (nexus.InteractiveLeadBinding, error) {
	if n == nil {
		return nexus.InteractiveLeadBinding{}, errors.New("nexus service is unavailable")
	}
	st, err := n.OpenProject()
	if err != nil {
		return nexus.InteractiveLeadBinding{}, err
	}
	canonical, err := store.CanonicalPath(workspace)
	if err != nil {
		return nexus.InteractiveLeadBinding{}, fmt.Errorf("canonicalize workspace: %w", err)
	}
	project, err := st.GetProjectByPath(canonical)
	if errors.Is(err, store.ErrNotFound) {
		project, err = st.CreateProject(store.Project{
			Name:          filepath.Base(canonical),
			CanonicalPath: canonical,
		})
	}
	if err != nil {
		return nexus.InteractiveLeadBinding{}, fmt.Errorf("resolve project: %w", err)
	}
	if err := st.TouchProject(project.ID); err != nil {
		return nexus.InteractiveLeadBinding{}, fmt.Errorf("touch project: %w", err)
	}
	agent, err := n.PrepareInteractiveLead(context.Background(), project.ID, provider, profileName)
	if err != nil {
		return nexus.InteractiveLeadBinding{}, err
	}
	return nexus.InteractiveLeadBinding{AgentID: agent.ID, ProjectID: project.ID, ProjectName: project.Name}, nil
}

func releaseInteractiveLeadReservation(n *nexus.Nexus, agentID string) {
	if n == nil || strings.TrimSpace(agentID) == "" {
		return
	}
	if st, err := n.OpenProject(); err == nil {
		_ = st.ReleaseInteractiveLeadReservation(agentID)
	}
}

func busyOrLaunchError(provider, profileName, _ string, err error) error {
	if errors.Is(err, codex.ErrTUIBusy) {
		return busyProfileError(provider, profileName)
	}
	if strings.Contains(err.Error(), "no fallback profiles are available") {
		return err
	}
	return fmt.Errorf("supervised %s launch failed: %w", provider, err)
}

func profileInCandidates(candidates []model.Profile, name string) bool {
	for _, candidate := range candidates {
		if candidate.Name == name {
			return true
		}
	}
	return false
}

func mustWorkingDirectory() string {
	cwd, err := os.Getwd()
	if err != nil || strings.TrimSpace(cwd) == "" {
		return "."
	}
	return cwd
}

func launchableProfiles(provName, explicit string, candidates []model.Profile) ([]model.Profile, error) {
	explicit = strings.TrimSpace(explicit)
	if explicit != "" {
		for _, candidate := range candidates {
			if candidate.Name == explicit {
				return []model.Profile{candidate}, nil
			}
		}
		return nil, fmt.Errorf("profile %q not found for provider %s", explicit, provName)
	}
	// Profile identity is shared configuration, not runtime ownership. Multiple
	// supervised invocations may use the same profile concurrently; the Codex
	// adapter's shared TUI lease protects only maintenance/app-server overlap.
	return candidates, nil
}
