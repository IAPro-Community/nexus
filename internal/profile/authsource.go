package profile

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
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
	"github.com/IAPro-Community/nexus/internal/core/provider/adapters/zcode"
)

const (
	authSourceFileName = "auth-source.json"
	// authLinkOptOutFile records an explicit logout. Without it, a profile with
	// no credential would silently re-adopt the host login the next time quota
	// is requested, undoing the user's decision.
	authLinkOptOutFile = "auth-link.opt-out"
)

// AuthSource reports how a profile obtained its provider credential, if the
// provenance was recorded. It never exposes the credential itself.
func AuthSource(provider, name string) (model.AuthSource, bool) {
	var src model.AuthSource
	root, err := config.ProfileRoot(provider, name)
	if err != nil || root == "" {
		return src, false
	}
	data, err := os.ReadFile(filepath.Join(root, authSourceFileName))
	if err != nil || len(data) == 0 {
		return src, false
	}
	if err := json.Unmarshal(data, &src); err != nil {
		return model.AuthSource{}, false
	}
	if src.Origin == "" {
		return model.AuthSource{}, false
	}
	return src, true
}

// ClearAuthSource forgets the provenance after the credential is removed or
// replaced, so the UI stops claiming the profile is linked to the host login.
func ClearAuthSource(provider, name string) error {
	root, err := config.ProfileRoot(provider, name)
	if err != nil || root == "" {
		return err
	}
	err = os.Remove(filepath.Join(root, authSourceFileName))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// HostAuthAccountID exposes the host CLI account identity for divergence
// checks, without leaking the credential. Empty when the provider has no
// host-managed login.
func HostAuthAccountID(provider string) string {
	if provider != "codex" {
		return ""
	}
	return codex.HostAuthAccountID()
}

// SuppressHostAuthLink records that the user explicitly logged this profile out,
// so automatic host-login adoption stays off until they log in again.
func SuppressHostAuthLink(provider, name string) error {
	root, err := config.ProfileRoot(provider, name)
	if err != nil || root == "" {
		return err
	}
	return os.WriteFile(filepath.Join(root, authLinkOptOutFile), []byte("explicit logout\n"), 0600)
}

func hostAuthLinkSuppressed(provider, name string) bool {
	root, err := config.ProfileRoot(provider, name)
	if err != nil || root == "" {
		return false
	}
	_, statErr := os.Stat(filepath.Join(root, authLinkOptOutFile))
	return statErr == nil
}

// AdoptHostAuth copies the host CLI login into a profile that has no
// credential of its own and records where it came from.
//
// The credential is never overwritten and provenance is always recorded.
func AdoptHostAuth(ctx context.Context, p model.Profile) (model.AuthSource, bool) {
	var src model.AuthSource
	if hostAuthLinkSuppressed(p.Provider, p.Name) {
		return src, false
	}
	if p.Provider == "codex" {
		return codex.AdoptHostAuth(ctx, p)
	}

	var adapter coreprovider.Provider
	switch p.Provider {
	case "claude":
		adapter = claude.New()
	case "gemini":
		adapter = gemini.New()
	case "agy":
		adapter = agy.New()
	case "opencode":
		adapter = opencode.New()
	case "cursor":
		adapter = cursor.New()
	case "zcode":
		adapter = zcode.New()
	default:
		return src, false
	}

	importer, canImport := adapter.(coreprovider.HostAuthImporter)
	authProv, canInspect := adapter.(coreprovider.AuthProvider)
	if !canImport || !canInspect {
		return src, false
	}

	imported, err := importer.ImportHostAuth(ctx, p)
	if err != nil || !imported {
		return src, false
	}

	acc := authProv.InspectAuth(ctx, p)
	if !acc.Authenticated {
		return src, false
	}

	plan := acc.Plan
	if plan == "" {
		plan = adapter.Name()
	}
	src = model.AuthSource{
		Origin:   model.AuthSourceOriginHost,
		LinkedAt: time.Now().UTC(),
		Plan:     plan,
	}
	root, err := config.ProfileRoot(p.Provider, p.Name)
	if err == nil && root != "" {
		if data, err := json.MarshalIndent(src, "", "  "); err == nil {
			_ = os.WriteFile(filepath.Join(root, authSourceFileName), append(data, '\n'), 0600)
		}
	}
	return src, true
}

// EnsureHostCredential links the host CLI login into a profile that has no
// credential of its own, unless the user explicitly logged it out.
//
// It is called before a quota read, where the intent to use the account is
// explicit. The credential is never overwritten and provenance is always
// recorded, so every surface can explain where the account came from.
func EnsureHostCredential(provider, name string) (model.AuthSource, bool) {
	if provider != "codex" {
		return model.AuthSource{}, false
	}
	if hostAuthLinkSuppressed(provider, name) {
		return model.AuthSource{}, false
	}
	return AdoptHostAuth(context.Background(), model.Profile{Provider: provider, Name: name})
}
