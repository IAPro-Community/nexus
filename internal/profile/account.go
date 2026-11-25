package profile

import (
	"context"
	"errors"
	"strings"

	"github.com/IAPro-Community/nexus/internal/core/model"
	coreprovider "github.com/IAPro-Community/nexus/internal/core/provider"
)

type AccountInfo = model.AccountInfo

var ErrAccountStatusUnverified = errors.New("account authentication status could not be verified")

// GetAccountInfo extracts authentic non-secret account identity, email, and subscription plan.
func GetAccountInfo(providerName, name string) model.AccountInfo {
	info, _ := GetAccountInfoContext(context.Background(), providerName, name)
	return info
}

// GetAccountInfoContext extracts account identity while respecting the caller's
// deadline for provider probes that may invoke a local CLI.
func GetAccountInfoContext(ctx context.Context, providerName, name string) (model.AccountInfo, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	p := model.Profile{Provider: providerName, Name: name}
	var info model.AccountInfo
	if adapter, ok := Registry().Get(providerName); ok {
		if authProv, ok := any(adapter).(coreprovider.AuthProvider); ok {
			info = authProv.InspectAuth(ctx, p)
		} else {
			info = model.AccountInfo{
				Status:        "Authentication inspection unsupported",
				Health:        model.HealthUnknown,
				Authenticated: false,
			}
		}
	} else {
		info = model.AccountInfo{
			Status:        "Unknown provider",
			Health:        model.HealthUnknown,
			Authenticated: false,
		}
	}
	if err := ctx.Err(); err != nil {
		info.Status = "Authentication status could not be verified"
		info.Health = model.HealthUnknown
		info.Authenticated = false
		return info, err
	}
	if info.Health == model.HealthUnknown {
		info.Authenticated = false
		return info, ErrAccountStatusUnverified
	}

	// Keep the opaque account boundary attached to every account inspection.
	// Codex scopes by chatgpt_account_id; other providers use email.
	identity := info.Email
	if providerName == "codex" && strings.TrimSpace(info.ExternalAccountID) != "" {
		identity = info.ExternalAccountID
	}
	if scope, err := AccountScope(providerName, name, identity); err == nil {
		info.AccountScope = scope
	}
	return info, nil
}
