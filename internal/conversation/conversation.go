package conversation

import (
	"context"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/core/provider/adapters/agy"
	"github.com/IAPro-Community/nexus/internal/core/provider/adapters/claude"
	"github.com/IAPro-Community/nexus/internal/core/provider/adapters/codex"
	"github.com/IAPro-Community/nexus/internal/core/provider/adapters/gemini"
	"github.com/IAPro-Community/nexus/internal/core/provider/adapters/opencode"
	"github.com/IAPro-Community/nexus/internal/core/session"
	"github.com/IAPro-Community/nexus/internal/profile"
)

type Conversation struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	Provider     string    `json:"provider"`
	Profile      string    `json:"profile,omitempty"`
	Workspace    string    `json:"workspace"`
	LastModified time.Time `json:"last_modified"`
}

// ListRecent returns unique recent conversations across all providers and profile stores.
func ListRecent(limit int, workspaceFilter string) []Conversation {
	ctx := context.Background()
	var allSessions []model.Session

	profs, _ := profile.List()

	// 1. Query for each configured profile
	codexAdapter := codex.New()
	agyAdapter := agy.New()
	claudeAdapter := claude.New()
	opencodeAdapter := opencode.New()
	geminiAdapter := gemini.New()

	for _, p := range profs {
		switch p.Provider {
		case "codex":
			if list, err := codexAdapter.ListConversations(ctx, p, workspaceFilter); err == nil {
				allSessions = append(allSessions, list...)
			}
		case "agy":
			if list, err := agyAdapter.ListConversations(ctx, p, workspaceFilter); err == nil {
				allSessions = append(allSessions, list...)
			}
		case "claude":
			if list, err := claudeAdapter.ListConversations(ctx, p, workspaceFilter); err == nil {
				allSessions = append(allSessions, list...)
			}
		case "opencode":
			if list, err := opencodeAdapter.ListConversations(ctx, p, workspaceFilter); err == nil {
				allSessions = append(allSessions, list...)
			}
		case "gemini":
			if list, err := geminiAdapter.ListConversations(ctx, p, workspaceFilter); err == nil {
				allSessions = append(allSessions, list...)
			}
		}
	}

	// 2. Query host-level default directories as fallback
	dummyProf := model.Profile{Name: "default"}
	if list, err := codexAdapter.ListConversations(ctx, dummyProf, workspaceFilter); err == nil {
		allSessions = append(allSessions, list...)
	}
	if list, err := agyAdapter.ListConversations(ctx, dummyProf, workspaceFilter); err == nil {
		allSessions = append(allSessions, list...)
	}
	if list, err := claudeAdapter.ListConversations(ctx, dummyProf, workspaceFilter); err == nil {
		allSessions = append(allSessions, list...)
	}
	if list, err := opencodeAdapter.ListConversations(ctx, dummyProf, workspaceFilter); err == nil {
		allSessions = append(allSessions, list...)
	}
	if list, err := geminiAdapter.ListConversations(ctx, dummyProf, workspaceFilter); err == nil {
		allSessions = append(allSessions, list...)
	}

	store := session.NewStore()
	aggregated := store.Aggregate(allSessions, workspaceFilter)

	var result []Conversation
	for _, s := range aggregated {
		result = append(result, Conversation{
			ID:           s.ID,
			Title:        s.Title,
			Provider:     s.ProviderID,
			Profile:      s.ProfileID,
			Workspace:    s.Workspace,
			LastModified: s.UpdatedAt,
		})
	}

	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}
	return result
}
