package app

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/core/provider/adapters/codex"
	"github.com/IAPro-Community/nexus/internal/core/quota"
	"github.com/IAPro-Community/nexus/internal/profile"
)

// A bounded preflight is run only when Nexus is making an automatic Codex
// account decision. It is never allowed to delay an explicitly selected
// profile, nor to run an app-server against a live interactive Codex home.
const codexLaunchProbeBudget = 10 * time.Second

func codexLaunchProbeNeeded(snap model.UsageSnapshot, now time.Time) bool {
	if snap.ProviderID != "codex" || len(snap.Windows) == 0 {
		return true
	}
	if quota.CodexEvidenceExpired(snap, now) {
		return true
	}
	// A recent rollout is still weaker evidence than the official quota read.
	// In particular it may predate /status. Do not pay the startup cost when
	// there is already a recent identity-checked official observation.
	return snap.Source != model.SourceOfficialAPI ||
		(snap.Status != model.UsageLive && snap.Status != model.UsageCached && snap.Status != model.UsageRateLimited) ||
		now.Sub(snap.FetchedAt) > quota.DefaultTTL
}

// refreshCodexLaunchQuota gives automatic account selection a single bounded
// opportunity to obtain current official quotas. The HTTP/app-server reader
// checks CODEX_HOME and account_id; this function does not infer identity.
// Failure or an active Codex TUI leaves the candidate's capacity UNKNOWN.
func refreshCodexLaunchQuota(
	ctx context.Context,
	providerName, explicitProfile string,
	candidates []model.Profile,
	accounts map[string]model.AccountInfo,
) {
	if providerName != "codex" || explicitProfile != "" {
		return
	}
	type candidate struct {
		profile model.Profile
		account model.AccountInfo
	}
	var toProbe []candidate
	now := time.Now()
	for _, p := range candidates {
		acc, ok := accounts[p.Name]
		if !ok || !acc.Authenticated || !codexLaunchProbeNeeded(acc.Usage, now) {
			continue
		}
		home, err := config.ProfileHome("codex", p.Name)
		if err != nil || codex.IsTUILocked(home) {
			continue
		}
		toProbe = append(toProbe, candidate{profile: p, account: acc})
	}
	if len(toProbe) == 0 {
		return
	}

	probeCtx, cancel := context.WithTimeout(ctx, codexLaunchProbeBudget)
	defer cancel()
	type result struct {
		name string
		snap model.UsageSnapshot
	}
	results := make(chan result, len(toProbe))
	var wg sync.WaitGroup
	// Keep concurrent probes bounded even for users with many profiles.
	sem := make(chan struct{}, 5)
	for _, candidate := range toProbe {
		wg.Add(1)
		go func(item candidate) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-probeCtx.Done():
				return
			}
			_ = profile.RefreshUsageSnapshotContext(probeCtx, "codex", item.profile.Name)
			if probeCtx.Err() != nil {
				return
			}
			// Re-read through the canonical account-scope/freshness checks.
			snap := profile.GetSanitizedUsageSnapshot("codex", item.profile.Name, item.account.Email)
			results <- result{name: item.profile.Name, snap: snap}
		}(candidate)
	}
	// All probes share the same deadline, not a deadline per account.
	// A canceled probe cannot modify the account map after we return.
	go func() {
		wg.Wait()
		close(results)
	}()
	completed := 0
	verified := 0
	for {
		select {
		case item, ok := <-results:
			if !ok {
				if completed > 0 {
					fmt.Fprintf(os.Stderr, "⚡ [nexus] Codex: %d/%d perfil(is) com quota atual validada\n", verified, len(toProbe))
				}
				return
			}
			acc := accounts[item.name]
			acc.Usage = item.snap
			accounts[item.name] = acc
			completed++
			if item.snap.Source == model.SourceOfficialAPI &&
				!quota.CodexEvidenceExpired(item.snap, time.Now()) &&
				(item.snap.Status == model.UsageLive || item.snap.Status == model.UsageCached ||
					item.snap.Status == model.UsageRateLimited) {
				verified++
			}
		case <-probeCtx.Done():
			fmt.Fprintf(os.Stderr, "⚠ [nexus] consulta de quota Codex encerrada após limite global; contas sem verificação permanecem UNKNOWN\n")
			return
		}
	}
}
