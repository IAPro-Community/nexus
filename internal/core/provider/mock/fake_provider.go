package mock

import (
	"context"

	"github.com/IAPro-Community/nexus/internal/core/model"
)

// FakeProvider implements Provider, AuthProvider, UsageProvider, and ConversationProvider for testing.
type FakeProvider struct {
	ProviderID    model.ProviderID
	ProviderName  string
	Caps          model.Capabilities
	DetectResult  model.DetectionResult
	PrepareErr    error
	RunFailure    model.Failure
	RunErr        error
	Account       model.AccountInfo
	Usage         model.UsageSnapshot
	Conversations []model.Session
	ResumeFailure model.Failure
	ResumeErr     error
}

func (f *FakeProvider) ID() model.ProviderID                               { return f.ProviderID }
func (f *FakeProvider) Name() string                                       { return f.ProviderName }
func (f *FakeProvider) Detect(ctx context.Context) model.DetectionResult   { return f.DetectResult }
func (f *FakeProvider) Capabilities() model.Capabilities                   { return f.Caps }
func (f *FakeProvider) Prepare(ctx context.Context, p model.Profile) error { return f.PrepareErr }
func (f *FakeProvider) Run(ctx context.Context, p model.Profile, args []string) (model.Failure, error) {
	return f.RunFailure, f.RunErr
}
func (f *FakeProvider) Login(ctx context.Context, p model.Profile) error  { return f.RunErr }
func (f *FakeProvider) Logout(ctx context.Context, p model.Profile) error { return nil }
func (f *FakeProvider) InspectAuth(ctx context.Context, p model.Profile) model.AccountInfo {
	return f.Account
}
func (f *FakeProvider) GetUsage(ctx context.Context, p model.Profile) model.UsageSnapshot {
	return f.Usage
}
func (f *FakeProvider) ListConversations(ctx context.Context, p model.Profile, workspace string) ([]model.Session, error) {
	return f.Conversations, nil
}
func (f *FakeProvider) Resume(ctx context.Context, p model.Profile, sessionID string, args []string) (model.Failure, error) {
	return f.ResumeFailure, f.ResumeErr
}
func (f *FakeProvider) ClassifyError(err error, output string) model.Failure {
	return f.RunFailure
}
