package provider

import (
	"context"

	"github.com/IAPro-Community/nexus/internal/core/model"
)

// Provider is the primary contract for integrating an AI coding CLI.
type Provider interface {
	ID() model.ProviderID
	Name() string
	Detect(ctx context.Context) model.DetectionResult
	Capabilities() model.Capabilities
	Prepare(ctx context.Context, p model.Profile) error
	Run(ctx context.Context, p model.Profile, args []string) (model.Failure, error)
}

// AuthProvider provides login, logout, and identity inspection.
type AuthProvider interface {
	Login(ctx context.Context, p model.Profile) error
	Logout(ctx context.Context, p model.Profile) error
	InspectAuth(ctx context.Context, p model.Profile) model.AccountInfo
}

// UsageProvider retrieves live or local usage metrics.
type UsageProvider interface {
	GetUsage(ctx context.Context, p model.Profile) model.UsageSnapshot
}

// ConversationProvider lists previous sessions and resumes them.
type ConversationProvider interface {
	ListConversations(ctx context.Context, p model.Profile, workspace string) ([]model.Session, error)
	Resume(ctx context.Context, p model.Profile, sessionID string, args []string) (model.Failure, error)
}

// ErrorClassifier classifies provider-specific errors.
type ErrorClassifier interface {
	ClassifyError(err error, output string) model.Failure
}
