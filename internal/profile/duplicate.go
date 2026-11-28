package profile

import (
	"context"
	"fmt"
	"sort"
	"strings"

	sdkproviders "github.com/IAPro-Community/IAPro-SDK/providers"
	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/core/provider/adapters/codex"
)

// DuplicateAccountError is returned when an operation tries to authenticate a
// second profile of the same provider using an account that is already registered.
type DuplicateAccountError struct {
	Provider        string
	Profile         string
	ExistingProfile string
	AccountLabel    string
}

func (e *DuplicateAccountError) Error() string {
	label := strings.TrimSpace(e.AccountLabel)
	if label == "" {
		return fmt.Sprintf("esta conta já está cadastrada em %s:%s; use o perfil existente em vez de duplicar", e.Provider, e.ExistingProfile)
	}
	return fmt.Sprintf("conta %s já está cadastrada em %s:%s; use o perfil existente em vez de duplicar", label, e.Provider, e.ExistingProfile)
}

// DuplicateAccountGroup describes multiple non-disabled profiles of the same
// provider that currently share one canonical account identity.
type DuplicateAccountGroup struct {
	Provider     string
	Identity     string
	AccountLabel string
	Profiles     []string
}

// CanonicalAccountIdentity returns a stable provider-scoped account key used to
// prevent duplicating the same login across multiple profiles of one provider.
func CanonicalAccountIdentity(providerName string, info model.AccountInfo) string {
	return sdkproviders.DuplicateIdentityHint(providerName, info.Email, info.ExternalAccountID, info.Authenticated)
}

func accountDisplayLabel(info model.AccountInfo) string {
	if email := strings.TrimSpace(info.Email); email != "" {
		return email
	}
	return strings.TrimSpace(info.ExternalAccountID)
}

// FindDuplicateAccount reports whether another non-disabled profile for the
// same provider already holds the account identified by info.
func FindDuplicateAccount(ctx context.Context, providerName, excludeProfile string, info model.AccountInfo) (existingProfile, accountLabel string, found bool) {
	if ctx == nil {
		ctx = context.Background()
	}
	targetID := CanonicalAccountIdentity(providerName, info)
	if targetID == "" {
		return "", "", false
	}
	profiles, err := List()
	if err != nil {
		return "", "", false
	}
	cfg, _ := config.LoadConfig()
	defaultName := cfg.Defaults[providerName]

	var matches []model.Profile
	var labels = map[string]string{}
	for _, p := range profiles {
		if p.Provider != providerName || p.Disabled || p.Name == excludeProfile {
			continue
		}
		other, accErr := GetAccountInfoContext(ctx, providerName, p.Name)
		if accErr != nil && !other.Authenticated {
			continue
		}
		if CanonicalAccountIdentity(providerName, other) == targetID {
			matches = append(matches, p)
			labels[p.Name] = accountDisplayLabel(other)
		}
	}
	if len(matches) == 0 {
		return "", "", false
	}
	sort.Slice(matches, func(i, j int) bool {
		iDefault := matches[i].Name == defaultName
		jDefault := matches[j].Name == defaultName
		if iDefault != jDefault {
			return iDefault
		}
		return matches[i].Name < matches[j].Name
	})
	chosen := matches[0].Name
	label := accountDisplayLabel(info)
	if label == "" {
		label = labels[chosen]
	}
	return chosen, label, true
}

// ValidateNoDuplicateAccount verifies that profileName's authenticated account
// does not duplicate another active profile for the same provider.
func ValidateNoDuplicateAccount(ctx context.Context, providerName, profileName string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	info, _ := GetAccountInfoContext(ctx, providerName, profileName)
	existing, label, found := FindDuplicateAccount(ctx, providerName, profileName, info)
	if !found {
		return nil
	}
	return &DuplicateAccountError{
		Provider:        providerName,
		Profile:         profileName,
		ExistingProfile: existing,
		AccountLabel:    label,
	}
}

// FindDuplicateAccountGroups returns groups of active profiles within the same
// provider that currently point to the same authenticated account.
func duplicateHints(ctx context.Context, profiles []model.Profile) []sdkproviders.DuplicateHint {
	hints := make([]sdkproviders.DuplicateHint, 0, len(profiles))
	for _, p := range profiles {
		if p.Disabled || ctx.Err() != nil {
			continue
		}
		info, _ := GetAccountInfoContext(ctx, p.Provider, p.Name)
		hints = append(hints, sdkproviders.DuplicateHint{
			OwnerID: "local", ProviderID: p.Provider, ProfileID: p.Name,
			Identity: CanonicalAccountIdentity(p.Provider, info), AccountLabel: accountDisplayLabel(info),
		})
	}
	return hints
}

func FindDuplicateAccountGroups(ctx context.Context, profiles []model.Profile) []DuplicateAccountGroup {
	if ctx == nil {
		ctx = context.Background()
	}
	groups := sdkproviders.GroupDuplicateHints(duplicateHints(ctx, profiles))
	out := make([]DuplicateAccountGroup, 0, len(groups))
	for _, g := range groups {
		out = append(out, DuplicateAccountGroup{Provider: g.ProviderID, Identity: g.Identity, AccountLabel: g.AccountLabel, Profiles: g.Profiles})
	}
	return out
}

// DuplicateCleanupResult describes a non-destructive duplicate advisory.
// RemovedProfiles remains for wire compatibility and is always empty here.
type DuplicateCleanupResult struct {
	Provider        string   `json:"provider"`
	Identity        string   `json:"identity"`
	AccountLabel    string   `json:"account_label,omitempty"`
	KeptProfile     string   `json:"kept_profile"`
	RemovedProfiles   []string `json:"removed_profiles,omitempty"`
	SuggestedProfiles []string `json:"suggested_profiles,omitempty"`
	SkippedLocked   []string `json:"skipped_locked,omitempty"`
}

// HasDuplicateAccounts checks in-memory account info for multiple non-disabled
// profiles of the same provider sharing one canonical account identity.
func HasDuplicateAccounts(profiles []model.Profile, accounts map[string]model.AccountInfo) bool {
	hints := make([]sdkproviders.DuplicateHint, 0, len(profiles))
	for _, p := range profiles {
		info, ok := accounts[p.Provider+":"+p.Name]
		if ok {
			hints = append(hints, sdkproviders.DuplicateHint{
				OwnerID: "local", ProviderID: p.Provider, ProfileID: p.Name, Disabled: p.Disabled,
				Identity: CanonicalAccountIdentity(p.Provider, info),
			})
		}
	}
	return len(sdkproviders.GroupDuplicateHints(hints)) > 0
}

func isAuthExpiredCached(provider, profileName string) bool {
	snap := GetUsageSnapshot(provider, profileName)
	return snap.Status == model.UsageError && strings.EqualFold(strings.TrimSpace(snap.DiagnosticCode), "AUTH_REQUIRED")
}

// DeduplicateAccounts is retained as a compatibility entry point. Passive
// duplicate discovery must never delete accounts, credentials or bindings.
// Existing local hints can only recommend a profile for explicit user review.
func DeduplicateAccounts(ctx context.Context) ([]DuplicateCleanupResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	profiles, err := List()
	if err != nil {
		return nil, err
	}
	cfg, cfgErr := config.LoadConfig()
	groups := FindDuplicateAccountGroups(ctx, profiles)
	var results []DuplicateCleanupResult
	for _, group := range groups {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		type candidate struct {
			name    string
			healthy bool
			pinned  bool
			locked  bool
		}
		candidates := make([]candidate, 0, len(group.Profiles))
		for _, name := range group.Profiles {
			info, _ := GetAccountInfoContext(ctx, group.Provider, name)
			if CanonicalAccountIdentity(group.Provider, info) != group.Identity {
				continue
			}
			locked := false
			if group.Provider == "codex" {
				if home, err := Home(group.Provider, name); err == nil {
					locked = codex.IsTUILocked(home)
				}
			}
			candidates = append(candidates, candidate{
				name: name, healthy: info.Authenticated && info.Health != model.HealthAuthRequired && !isAuthExpiredCached(group.Provider, name),
				pinned: cfgErr == nil && cfg.Defaults[group.Provider] == name, locked: locked,
			})
		}
		if len(candidates) < 2 {
			continue
		}
		sort.Slice(candidates, func(i, j int) bool {
			a, b := candidates[i], candidates[j]
			if a.healthy != b.healthy {
				return a.healthy
			}
			if a.pinned != b.pinned {
				return a.pinned
			}
			if a.locked != b.locked {
				return a.locked
			}
			return a.name < b.name
		})
		res := DuplicateCleanupResult{Provider: group.Provider, Identity: group.Identity, AccountLabel: group.AccountLabel, KeptProfile: candidates[0].name}
		for _, c := range candidates[1:] {
			if c.locked {
				res.SkippedLocked = append(res.SkippedLocked, c.name)
			} else {
				res.SuggestedProfiles = append(res.SuggestedProfiles, c.name)
			}
		}
		sort.Strings(res.SuggestedProfiles)
		sort.Strings(res.SkippedLocked)
		results = append(results, res)
	}
	return results, ctx.Err()
}
