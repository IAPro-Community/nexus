package profile

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/model"
	coreprovider "github.com/IAPro-Community/nexus/internal/core/provider"
	"github.com/IAPro-Community/nexus/internal/core/provider/adapters/agy"
	"github.com/IAPro-Community/nexus/internal/core/provider/adapters/claude"
	"github.com/IAPro-Community/nexus/internal/core/provider/adapters/codex"
	"github.com/IAPro-Community/nexus/internal/core/provider/adapters/cursor"
	"github.com/IAPro-Community/nexus/internal/core/provider/adapters/gemini"
	"github.com/IAPro-Community/nexus/internal/core/provider/adapters/opencode"
	"github.com/IAPro-Community/nexus/internal/core/provider/authcopy"
)

const (
	HostSessionImported             = "IMPORTED"
	HostSessionAlreadyAuthenticated = "ALREADY_AUTHENTICATED"
	HostSessionNoSource             = "NO_SESSION"
	HostSessionLoginRequired        = "LOGIN_REQUIRED"
	HostSessionUnsupported          = "UNSUPPORTED"
	HostSessionFailed               = "FAILED"
	HostSessionNotInstalled         = "NOT_INSTALLED"
)

// HostSessionImportResult contains only profile metadata and a stable status;
// it never includes account identity or credential material.
type HostSessionImportResult struct {
	Provider string `json:"provider"`
	Profile  string `json:"profile,omitempty"`
	Status   string `json:"status"`
}

var hostSessionImportMu sync.Mutex

// ImportHostSessions attempts to import file-backed sessions for every
// installed provider that implements HostAuthImporter. It registers profiles
// only when a valid host session is actually available.
func ImportHostSessions(ctx context.Context) ([]HostSessionImportResult, error) {
	hostSessionImportMu.Lock()
	defer hostSessionImportMu.Unlock()

	profiles, err := List()
	if err != nil {
		return nil, err
	}
	providers := []coreprovider.Provider{
		codex.New(), claude.New(), gemini.New(), agy.New(), opencode.New(), cursor.New(),
	}
	results := make([]HostSessionImportResult, 0, len(providers))
	for _, adapter := range providers {
		result, updatedProfiles := importHostSession(ctx, adapter, profiles)
		results = append(results, result)
		profiles = updatedProfiles
	}
	return results, nil
}

func importHostSession(
	ctx context.Context,
	adapter coreprovider.Provider,
	profiles []model.Profile,
) (HostSessionImportResult, []model.Profile) {
	providerID := string(adapter.ID())
	result := HostSessionImportResult{Provider: providerID}

	providerCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	detection := adapter.Detect(providerCtx)
	if !detection.Installed {
		result.Status = HostSessionNotInstalled
		return result, profiles
	}

	importer, canImport := adapter.(coreprovider.HostAuthImporter)
	authProvider, canInspect := adapter.(coreprovider.AuthProvider)
	if !canImport || !canInspect {
		result.Status = HostSessionUnsupported
		return result, profiles
	}

	existing := make([]model.Profile, 0)
	for _, candidate := range profiles {
		if candidate.Provider == providerID && !candidate.Disabled {
			if authProvider.InspectAuth(providerCtx, candidate).Authenticated {
				result.Profile = candidate.Name
				result.Status = HostSessionAlreadyAuthenticated
				return result, profiles
			}
			existing = append(existing, candidate)
		}
	}

	var target model.Profile
	created := false
	if len(existing) > 0 {
		target = preferredHostImportProfile(providerID, existing)
	} else {
		name := newHostImportProfileName(providerID, profiles)
		var err error
		target, err = Create(providerID, name)
		if err != nil {
			result.Status = HostSessionFailed
			return result, profiles
		}
		created = true
	}

	imported, err := importer.ImportHostAuth(providerCtx, target)
	if errors.Is(err, authcopy.ErrDestinationExists) && !created {
		// Keep an existing invalid/pending profile intact. A separate profile
		// lets a valid host session be imported without replacing its credentials.
		name := newHostImportProfileName(providerID, profiles)
		target, err = Create(providerID, name)
		if err == nil {
			created = true
			imported, err = importer.ImportHostAuth(providerCtx, target)
		}
	}
	if err != nil {
		if created {
			_ = Delete(providerID, target.Name)
		}
		result.Status = HostSessionFailed
		return result, profiles
	}
	if !imported {
		if created {
			_ = Delete(providerID, target.Name)
			result.Status = HostSessionNoSource
			return result, profiles
		}
		result.Profile = target.Name
		result.Status = HostSessionLoginRequired
		return result, profiles
	}

	account := authProvider.InspectAuth(providerCtx, target)
	if !account.Authenticated {
		result.Status = HostSessionFailed
		if created {
			_ = Delete(providerID, target.Name)
			return result, profiles
		}
		result.Profile = target.Name
		result.Status = HostSessionLoginRequired
		return result, profiles
	}
	identity := account.Email
	if providerID == "codex" && account.ExternalAccountID != "" {
		identity = account.ExternalAccountID
	}
	if _, err := AccountScope(providerID, target.Name, identity); err != nil {
		if created {
			_ = Delete(providerID, target.Name)
		}
		result.Status = HostSessionFailed
		return result, profiles
	}
	setDefaultIfUnavailable(providerID, target, existing, authProvider, providerCtx)

	result.Profile = target.Name
	result.Status = HostSessionImported
	if created {
		profiles = append(profiles, target)
		sort.Slice(profiles, func(i, j int) bool {
			if profiles[i].Provider == profiles[j].Provider {
				return profiles[i].Name < profiles[j].Name
			}
			return profiles[i].Provider < profiles[j].Provider
		})
	}
	return result, profiles
}

func preferredHostImportProfile(providerID string, profiles []model.Profile) model.Profile {
	cfg, _ := config.LoadConfig()
	defaultName := cfg.Defaults[providerID]
	for _, candidate := range profiles {
		if candidate.Name == defaultName {
			return candidate
		}
	}
	return profiles[0]
}

func newHostImportProfileName(providerID string, profiles []model.Profile) string {
	hasProviderProfile := false
	for _, candidate := range profiles {
		if candidate.Provider == providerID {
			hasProviderProfile = true
		}
	}
	if !hasProviderProfile && !Exists(providerID, providerID) {
		return providerID
	}
	for suffix := 1; ; suffix++ {
		name := "host-import"
		if suffix > 1 {
			name = fmt.Sprintf("host-import-%d", suffix)
		}
		if !Exists(providerID, name) {
			return name
		}
	}
}

func setDefaultIfUnavailable(
	providerID string,
	target model.Profile,
	existing []model.Profile,
	authProvider coreprovider.AuthProvider,
	ctx context.Context,
) {
	cfg, err := config.LoadConfig()
	if err != nil {
		return
	}
	defaultName := cfg.Defaults[providerID]
	if defaultName == "" {
		_ = config.SetDefaultProfile(providerID, target.Name)
		return
	}
	for _, candidate := range existing {
		if candidate.Name == defaultName && authProvider.InspectAuth(ctx, candidate).Authenticated {
			return
		}
	}
	if defaultName != target.Name {
		_ = config.SetDefaultProfile(providerID, target.Name)
	}
}
