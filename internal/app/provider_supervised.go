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
	"github.com/IAPro-Community/nexus/internal/control/protocol"
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

// executeProviderSupervised launches the provider through the same persistent
// SessionHost used by Web/Desktop. This opt-in path gives terminal-only users
// slash control, detach/attach, handoff and generation tracking without
// requiring the Web or Desktop surface.
func executeProviderSupervised(provName, explicitProfile string, args []string) error {
	if strings.TrimSpace(provName) == "" {
		return fmt.Errorf("provider is required for supervised launch")
	}

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

	candidateNames := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		candidateNames = append(candidateNames, candidate.Name)
	}
	cfg, _ := config.LoadConfig()
	selected := profile.ResolveDefault(provName, explicitProfile, candidateNames, cfg.Defaults[provName], func() (string, error) {
		selector := scheduler.NewSelector(cfg, quotaEngine, cooldown.NewTracker())
		result, selectErr := selector.SelectBestProfile(context.Background(), provName, mustWorkingDirectory(), candidates, accounts, nil)
		if selectErr == nil && result != nil && result.SelectedProfile != nil {
			return result.SelectedProfile.Name, nil
		}
		return "", selectErr
	})
	if !profileInCandidates(candidates, selected) {
		return fmt.Errorf("profile %q is not configured for provider %s", selected, provName)
	}

	workspace := mustWorkingDirectory()
	lead := nexus.InteractiveLeadBinding{}
	n := nexus.Default()
	if st, projectErr := n.OpenProject(); projectErr == nil {
		if canonical, canonicalErr := store.CanonicalPath(workspace); canonicalErr == nil {
			project, lookupErr := st.GetProjectByPath(canonical)
			if errors.Is(lookupErr, store.ErrNotFound) {
				project, lookupErr = st.CreateProject(store.Project{
					Name:          filepath.Base(canonical),
					CanonicalPath: canonical,
				})
			}
			if lookupErr == nil {
				_ = st.TouchProject(project.ID)
				if agent, leadErr := n.PrepareInteractiveLead(context.Background(), project.ID, provName, selected); leadErr == nil {
					lead = nexus.InteractiveLeadBinding{AgentID: agent.ID, ProjectID: project.ID, ProjectName: project.Name}
				}
			}
		}
	}
	session, err := launcher.Default().Launch(context.Background(), launcher.LaunchOptions{
		ProviderID:  provName,
		ProfileID:   selected,
		Workspace:   workspace,
		AgentID:     lead.AgentID,
		ProjectID:   lead.ProjectID,
		ProjectName: lead.ProjectName,
		Args:        args,
		Standalone:  false,
		Timeout:     15 * time.Second,
		Labels: map[string]string{
			"nexus.interactive_lead": fmt.Sprintf("%t", lead.AgentID != ""),
			"nexus.delegation_mode":  string(nexus.DelegationAuto),
		},
	})
	if err != nil {
		releaseInteractiveLeadReservation(n, lead.AgentID)
		if errors.Is(err, codex.ErrTUIBusy) {
			return fmt.Errorf("%s profile %s is busy (profile lock held by another session); stop the existing session or retry shortly", provName, selected)
		}
		return fmt.Errorf("supervised %s launch failed: %w", provName, err)
	}
	if lead.AgentID != "" {
		if err := n.BindInteractiveLeadRuntime(context.Background(), lead.AgentID, session); err != nil {
			if client, clientErr := protocol.NewClient(session.RuntimeID); clientErr == nil {
				_ = client.Stop()
				_ = client.Close()
			}
			releaseInteractiveLeadReservation(n, lead.AgentID)
			return fmt.Errorf("bind interactive Lead Agent: %w", err)
		}
	}
	fmt.Fprintf(os.Stderr, "⚡ [nexus] supervised %s:%s runtime=%s\n", provName, selected, session.RuntimeID)
	if lead.AgentID == "" {
		return attachRuntime(session.RuntimeID)
	}
	return attachInteractiveLeadRuntime(session.RuntimeID, n, *session, lead.AgentID)
}

func releaseInteractiveLeadReservation(n *nexus.Nexus, agentID string) {
	if n == nil || strings.TrimSpace(agentID) == "" {
		return
	}
	if st, err := n.OpenProject(); err == nil {
		_ = st.ReleaseInteractiveLeadReservation(agentID)
	}
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
