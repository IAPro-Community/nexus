package profile

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/core/provider/adapters/codex"
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
	return codex.AdoptHostAuth(context.Background(), model.Profile{Provider: provider, Name: name})
}
