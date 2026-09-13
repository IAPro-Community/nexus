package profile

import (
	"context"
	"strings"

	"github.com/kivervinicius/ai-cli/internal/core/model"
	"github.com/kivervinicius/ai-cli/internal/core/provider/adapters/agy"
	"github.com/kivervinicius/ai-cli/internal/core/provider/adapters/claude"
	"github.com/kivervinicius/ai-cli/internal/core/provider/adapters/codex"
	"github.com/kivervinicius/ai-cli/internal/core/provider/adapters/cursor"
	"github.com/kivervinicius/ai-cli/internal/core/provider/adapters/gemini"
	"github.com/kivervinicius/ai-cli/internal/core/provider/adapters/opencode"
)

type AccountInfo = model.AccountInfo

// GetAccountInfo extracts authentic non-secret account identity, email, and subscription plan.
func GetAccountInfo(providerName, name string) model.AccountInfo {
	ctx := context.Background()
	p := model.Profile{Provider: providerName, Name: name}
	var info model.AccountInfo
	switch providerName {
	case "codex":
		info = codex.New().InspectAuth(ctx, p)
	case "agy":
		info = agy.New().InspectAuth(ctx, p)
	case "claude":
		info = claude.New().InspectAuth(ctx, p)
	case "opencode":
		info = opencode.New().InspectAuth(ctx, p)
	case "gemini":
		info = gemini.New().InspectAuth(ctx, p)
	case "cursor":
		info = cursor.New().InspectAuth(ctx, p)
	default:
		info = model.AccountInfo{
			Status:        "Unknown provider",
			Health:        model.HealthUnknown,
			Authenticated: false,
		}
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
	return info
}
