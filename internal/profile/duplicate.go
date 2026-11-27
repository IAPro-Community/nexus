package profile

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

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
	if !info.Authenticated {
		return ""
	}
	prov := strings.ToLower(strings.TrimSpace(providerName))
	email := strings.ToLower(strings.TrimSpace(info.Email))
	extID := strings.ToLower(strings.TrimSpace(info.ExternalAccountID))
	if prov == "codex" || prov == "cursor" {
		if extID != "" {
			return extID
		}
		return email
	}
	if email != "" {
		return email
	}
	return extID
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
func FindDuplicateAccountGroups(ctx context.Context, profiles []model.Profile) []DuplicateAccountGroup {
	if ctx == nil {
		ctx = context.Background()
	}
	type groupKey struct {
		provider string
		identity string
	}
	grouped := map[groupKey]*DuplicateAccountGroup{}
	for _, p := range profiles {
		if p.Disabled {
			continue
		}
		info, _ := GetAccountInfoContext(ctx, p.Provider, p.Name)
		identity := CanonicalAccountIdentity(p.Provider, info)
		if identity == "" {
			continue
		}
		key := groupKey{provider: p.Provider, identity: identity}
		grp, ok := grouped[key]
		if !ok {
			grp = &DuplicateAccountGroup{
				Provider:     p.Provider,
				Identity:     identity,
				AccountLabel: accountDisplayLabel(info),
			}
			grouped[key] = grp
		}
		if grp.AccountLabel == "" {
			grp.AccountLabel = accountDisplayLabel(info)
		}
		grp.Profiles = append(grp.Profiles, p.Name)
	}
	var out []DuplicateAccountGroup
	for _, grp := range grouped {
		if len(grp.Profiles) > 1 {
			sort.Strings(grp.Profiles)
			out = append(out, *grp)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Provider != out[j].Provider {
			return out[i].Provider < out[j].Provider
		}
		return out[i].AccountLabel < out[j].AccountLabel
	})
	return out
}

var dedupeMu sync.Mutex

// DuplicateCleanupResult describes automatic deduplication performed for a
// provider account shared across multiple active profiles.
type DuplicateCleanupResult struct {
	Provider        string   `json:"provider"`
	Identity        string   `json:"identity"`
	AccountLabel    string   `json:"account_label,omitempty"`
	KeptProfile     string   `json:"kept_profile"`
	RemovedProfiles []string `json:"removed_profiles,omitempty"`
	SkippedLocked   []string `json:"skipped_locked,omitempty"`
}

// HasDuplicateAccounts checks in-memory account info for multiple non-disabled
// profiles of the same provider sharing one canonical account identity.
func HasDuplicateAccounts(profiles []model.Profile, accounts map[string]model.AccountInfo) bool {
	type key struct {
		provider string
		identity string
	}
	seen := make(map[key]struct{}, len(profiles))
	for _, p := range profiles {
		if p.Disabled {
			continue
		}
		info, ok := accounts[p.Provider+":"+p.Name]
		if !ok {
			continue
		}
		ident := CanonicalAccountIdentity(p.Provider, info)
		if ident == "" {
			continue
		}
		k := key{provider: p.Provider, identity: ident}
		if _, dup := seen[k]; dup {
			return true
		}
		seen[k] = struct{}{}
	}
	return false
}

func isAuthExpiredCached(provider, profileName string) bool {
	snap := GetUsageSnapshot(provider, profileName)
	return snap.Status == model.UsageError && strings.EqualFold(strings.TrimSpace(snap.DiagnosticCode), "AUTH_REQUIRED")
}

// DeduplicateAccounts removes redundant non-disabled profiles that point to the
// same canonical account within a provider, migrating defaults and bindings to
// the kept profile and leaving any TUI-locked Codex profiles untouched.
func DeduplicateAccounts(ctx context.Context) ([]DuplicateCleanupResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	dedupeMu.Lock()
	defer dedupeMu.Unlock()

	profiles, err := List()
	if err != nil {
		return nil, err
	}
	cfg, cfgErr := config.LoadConfig()

	type candidate struct {
		profile      model.Profile
		healthy      bool
		isDefault    bool
		locked       bool
		hasEmail     bool
		explicitName bool
		label        string
	}
	type groupKey struct {
		provider string
		identity string
	}
	groups := make(map[groupKey][]candidate)
	for _, p := range profiles {
		if p.Disabled {
			continue
		}
		acc, _ := GetAccountInfoContext(ctx, p.Provider, p.Name)
		identity := CanonicalAccountIdentity(p.Provider, acc)
		if identity == "" {
			continue
		}
		locked := false
		if p.Provider == "codex" {
			if home, homeErr := Home(p.Provider, p.Name); homeErr == nil && codex.IsTUILocked(home) {
				locked = true
			}
		}
		c := candidate{
			profile:      p,
			healthy:      acc.Authenticated && acc.Health != model.HealthAuthRequired && !isAuthExpiredCached(p.Provider, p.Name),
			isDefault:    cfgErr == nil && cfg.Defaults[p.Provider] == p.Name,
			locked:       locked,
			hasEmail:     strings.TrimSpace(acc.Email) != "",
			explicitName: !strings.EqualFold(p.Name, p.Provider) && !strings.EqualFold(p.Name, "default"),
			label:        accountDisplayLabel(acc),
		}
		k := groupKey{provider: p.Provider, identity: identity}
		groups[k] = append(groups[k], c)
	}

	var results []DuplicateCleanupResult
	for k, cands := range groups {
		if len(cands) <= 1 {
			continue
		}
		sort.Slice(cands, func(i, j int) bool {
			if cands[i].healthy != cands[j].healthy {
				return cands[i].healthy
			}
			if cands[i].isDefault != cands[j].isDefault {
				return cands[i].isDefault
			}
			if cands[i].locked != cands[j].locked {
				return cands[i].locked
			}
			if cands[i].hasEmail != cands[j].hasEmail {
				return cands[i].hasEmail
			}
			if cands[i].explicitName != cands[j].explicitName {
				return cands[i].explicitName
			}
			return cands[i].profile.Name < cands[j].profile.Name
		})

		keeper := cands[0]
		label := keeper.label
		if label == "" {
			for _, c := range cands[1:] {
				if c.label != "" {
					label = c.label
					break
				}
			}
		}
		res := DuplicateCleanupResult{
			Provider:     k.provider,
			Identity:     k.identity,
			AccountLabel: label,
			KeptProfile:  keeper.profile.Name,
		}
		for _, dup := range cands[1:] {
			if dup.locked {
				res.SkippedLocked = append(res.SkippedLocked, dup.profile.Name)
				continue
			}
			if cfgErr == nil {
				cfgDirty := false
				if cfg.Defaults != nil && cfg.Defaults[k.provider] == dup.profile.Name {
					cfg.Defaults[k.provider] = keeper.profile.Name
					cfgDirty = true
				}
				for _, bindings := range cfg.Bindings {
					if bindings != nil && bindings[k.provider] == dup.profile.Name {
						bindings[k.provider] = keeper.profile.Name
						cfgDirty = true
					}
				}
				if cfgDirty {
					_ = config.SaveConfig(cfg)
				}
			}
			if delErr := Delete(dup.profile.Provider, dup.profile.Name); delErr == nil {
				res.RemovedProfiles = append(res.RemovedProfiles, dup.profile.Name)
			}
		}
		if len(res.RemovedProfiles) > 0 || len(res.SkippedLocked) > 0 {
			sort.Strings(res.RemovedProfiles)
			sort.Strings(res.SkippedLocked)
			results = append(results, res)
		}
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Provider != results[j].Provider {
			return results[i].Provider < results[j].Provider
		}
		if results[i].AccountLabel != results[j].AccountLabel {
			return results[i].AccountLabel < results[j].AccountLabel
		}
		return results[i].KeptProfile < results[j].KeptProfile
	})
	return results, nil
}
