package profile

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/model"
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
	if prov == "codex" {
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
