package nexus

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/computer"
	"github.com/IAPro-Community/nexus/internal/nexus/runner"
)

type fakeComputerProvider struct {
	capabilities computer.Capabilities
	permissions  computer.PermissionStatus
	snapshot     computer.Snapshot
	action       computer.ActionResult
	snapshotErr  error
	actionErr    error
	snapshotCalls int
	actionCalls   int
}

func (p *fakeComputerProvider) Capabilities(context.Context) (computer.Capabilities, error) {
	return p.capabilities, nil
}

func (p *fakeComputerProvider) Permissions(context.Context) (computer.PermissionStatus, error) {
	return p.permissions, nil
}

func (p *fakeComputerProvider) ListApps(context.Context) ([]computer.AppInfo, error) {
	return nil, nil
}

func (p *fakeComputerProvider) ListWindows(context.Context, string) ([]computer.WindowInfo, error) {
	return nil, nil
}

func (p *fakeComputerProvider) Snapshot(context.Context, computer.SnapshotRequest) (computer.Snapshot, error) {
	p.snapshotCalls++
	return p.snapshot, p.snapshotErr
}

func (p *fakeComputerProvider) Act(context.Context, computer.ActionRequest) (computer.ActionResult, error) {
	p.actionCalls++
	return p.action, p.actionErr
}

func withFakeComputerProvider(t *testing.T, provider computer.Provider) {
	t.Helper()
	previous := computerProviderFactory
	computerProviderFactory = func() computer.Provider { return provider }
	t.Cleanup(func() { computerProviderFactory = previous })
}

func availableComputerProvider() *fakeComputerProvider {
	return &fakeComputerProvider{
		capabilities: computer.Capabilities{
			Provider:          "fake",
			ProtocolVersion:   1,
			Platform:          "test",
			AccessibilityTree: true,
			Screenshot:        true,
			Click:             true,
			SetValue:          true,
			WindowManagement:  true,
		},
		permissions: computer.PermissionStatus{
			Available:        true,
			Accessibility:   "AVAILABLE",
			ScreenCapture:   "AVAILABLE",
			InputMonitoring: "NOT_REQUIRED",
		},
	}
}

func TestComputerSnapshotFailsClosedByDefault(t *testing.T) {
	t.Setenv("NEXUS_ENABLE_COMPUTER_USE", "")
	t.Setenv("NEXUS_DISABLE_COMPUTER_USE", "")
	provider := availableComputerProvider()
	withFakeComputerProvider(t, provider)

	n := &Nexus{}
	_, err := n.ComputerSnapshot(context.Background(), runner.AutonomyContract{
		AllowComputerUse:   true,
		AllowScreenCapture: true,
	}, computer.SnapshotRequest{App: "editor", Screenshot: true})
	if err == nil || !strings.Contains(err.Error(), "disabled by feature flag") {
		t.Fatalf("computer use must default off, got %v", err)
	}
	if provider.snapshotCalls != 0 {
		t.Fatal("disabled computer use must not call native provider")
	}
}

func TestComputerSnapshotRequiresContractAndCapability(t *testing.T) {
	t.Setenv("NEXUS_ENABLE_COMPUTER_USE", "1")
	provider := availableComputerProvider()
	withFakeComputerProvider(t, provider)

	n := &Nexus{}
	_, err := n.ComputerSnapshot(context.Background(), runner.AutonomyContract{}, computer.SnapshotRequest{
		App: "editor",
	})
	if err == nil || !strings.Contains(err.Error(), "not authorized") {
		t.Fatalf("missing contract permission must fail closed, got %v", err)
	}

	provider.capabilities.Screenshot = false
	_, err = n.ComputerSnapshot(context.Background(), runner.AutonomyContract{
		AllowComputerUse:   true,
		AllowScreenCapture: true,
	}, computer.SnapshotRequest{App: "editor", Screenshot: true})
	if err == nil || !strings.Contains(err.Error(), "does not support screenshots") {
		t.Fatalf("missing screenshot capability must fail closed, got %v", err)
	}
}

func TestComputerActRejectsSensitiveControlBeforeMutation(t *testing.T) {
	t.Setenv("NEXUS_ENABLE_COMPUTER_USE", "1")
	provider := availableComputerProvider()
	provider.snapshot = computer.Snapshot{
		ID: "before",
		Elements: []computer.Element{
			{ID: "password-box", Index: 3, Role: "Edit", Name: "Password", Sensitive: true},
		},
		CapturedAt: time.Now().UTC(),
	}
	withFakeComputerProvider(t, provider)

	n := &Nexus{}
	_, err := n.ComputerAct(context.Background(), runner.AutonomyContract{
		AllowComputerUse:   true,
		AllowKeyboardInput: true,
	}, computer.ActionRequest{
		Kind:      computer.ActionSetValue,
		App:       "editor",
		ElementID: "password-box",
		Text:      "secret",
	})
	if err == nil || !strings.Contains(err.Error(), "sensitive/password") {
		t.Fatalf("sensitive control mutation must fail closed, got %v", err)
	}
	if provider.snapshotCalls != 1 {
		t.Fatalf("expected one semantic preflight snapshot, got %d", provider.snapshotCalls)
	}
	if provider.actionCalls != 0 {
		t.Fatal("sensitive mutation reached native provider")
	}
}

func TestComputerActRequiresSpecificPermissionAndCapability(t *testing.T) {
	t.Setenv("NEXUS_ENABLE_COMPUTER_USE", "1")
	provider := availableComputerProvider()
	withFakeComputerProvider(t, provider)
	n := &Nexus{}

	_, err := n.ComputerAct(context.Background(), runner.AutonomyContract{
		AllowComputerUse: true,
	}, computer.ActionRequest{
		Kind: computer.ActionClick,
		App: "editor",
		X: intPtr(10),
		Y: intPtr(20),
	})
	if err == nil || !strings.Contains(err.Error(), "pointer input is not authorized") {
		t.Fatalf("click without pointer permission must fail closed, got %v", err)
	}

	provider.capabilities.Click = false
	_, err = n.ComputerAct(context.Background(), runner.AutonomyContract{
		AllowComputerUse:  true,
		AllowPointerInput: true,
	}, computer.ActionRequest{
		Kind: computer.ActionClick,
		App: "editor",
		X: intPtr(10),
		Y: intPtr(20),
	})
	if err == nil || !strings.Contains(err.Error(), "does not support action CLICK") {
		t.Fatalf("click without capability must fail closed, got %v", err)
	}
}

func TestComputerSnapshotRedactsSecretsAndSensitiveValues(t *testing.T) {
	t.Setenv("NEXUS_ENABLE_COMPUTER_USE", "1")
	provider := availableComputerProvider()
	secret := "sk-proj-1234567890abcdef1234567890abcdef1234567890"
	provider.snapshot = computer.Snapshot{
		ID: "snapshot-1",
		App: computer.AppInfo{Name: "Editor " + secret, PID: 42},
		Window: computer.WindowInfo{Title: "token=" + secret, Width: 800, Height: 600},
		Elements: []computer.Element{
			{ID: "normal", Index: 1, Name: "api_key=" + secret, Value: secret},
			{ID: "password", Index: 2, Name: "Password", Value: "plain-password", Sensitive: true},
		},
		TreeText:   "authorization: Bearer " + secret,
		CapturedAt: time.Now().UTC(),
	}
	withFakeComputerProvider(t, provider)

	n := &Nexus{}
	snapshot, err := n.ComputerSnapshot(context.Background(), runner.AutonomyContract{
		AllowComputerUse: true,
	}, computer.SnapshotRequest{App: "editor"})
	if err != nil {
		t.Fatal(err)
	}
	raw := snapshot.App.Name + snapshot.Window.Title + snapshot.Elements[0].Name +
		snapshot.Elements[0].Value + snapshot.Elements[1].Value + snapshot.TreeText
	if strings.Contains(raw, secret) || strings.Contains(raw, "plain-password") {
		t.Fatalf("computer snapshot leaked sensitive content: %q", raw)
	}
	if snapshot.Elements[1].Value != "" {
		t.Fatalf("sensitive element value must be empty, got %q", snapshot.Elements[1].Value)
	}
}

func TestComputerActPropagatesProviderFailureWithoutMutationEvidence(t *testing.T) {
	t.Setenv("NEXUS_ENABLE_COMPUTER_USE", "1")
	provider := availableComputerProvider()
	provider.actionErr = errors.New("native action failed")
	withFakeComputerProvider(t, provider)

	n := &Nexus{}
	_, err := n.ComputerAct(context.Background(), runner.AutonomyContract{
		AllowComputerUse:       true,
		AllowWindowManagement:  true,
	}, computer.ActionRequest{
		Kind: computer.ActionFocusWindow,
		App:  "editor",
	})
	if err == nil || !strings.Contains(err.Error(), "native action failed") {
		t.Fatalf("provider action failure was hidden: %v", err)
	}
}

func intPtr(value int) *int { return &value }
