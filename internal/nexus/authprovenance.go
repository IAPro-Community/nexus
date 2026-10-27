package nexus

import (
	"strings"

	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/profile"
)

// AccountWarningHostChanged is emitted when a profile still uses an adopted
// credential while the host CLI login now belongs to a different account.
// The profile is never re-pointed automatically: silently switching quota
// identity would attribute another account's capacity to this profile.
const AccountWarningHostChanged = "host_account_changed"

// attachCredentialProvenance records how the profile obtained its credential
// and flags divergence from the host login. It never mutates credentials.
func attachCredentialProvenance(acc *ProviderAccount, provider, name string, authenticated bool, externalAccountID string) {
	src, ok := profile.AuthSource(provider, name)
	if !ok {
		return
	}
	acc.AuthSource = &src
	if src.Origin != model.AuthSourceOriginHost || !authenticated {
		return
	}
	hostID := profile.HostAuthAccountID(provider)
	if hostID == "" || strings.TrimSpace(externalAccountID) == "" {
		return
	}
	if !strings.EqualFold(hostID, externalAccountID) {
		acc.AccountWarning = AccountWarningHostChanged
	}
}
