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


func TestComputerActTypeTextRequiresSafeTargetAndBoundedText(t *testing.T) {
	t.Setenv("NEXUS_ENABLE_COMPUTER_USE", "1")
	provider := availableComputerProvider()
	provider.capabilities.TypeText = true
	provider.snapshot = computer.Snapshot{
		ID: "before",
		Elements: []computer.Element{
			{ID: "editor-box", Index: 1, Role: "Edit", Name: "Editor"},
		},
		CapturedAt: time.Now().UTC(),
	}
	provider.action = computer.ActionResult{
		Snapshot: computer.Snapshot{ID: "after", CapturedAt: time.Now().UTC()},
		Verification: computer.ActionVerification{State: "OBSERVED"},
	}
	withFakeComputerProvider(t, provider)

	n := &Nexus{}
	_, err := n.ComputerAct(context.Background(), runner.AutonomyContract{
		AllowComputerUse:   true,
		AllowKeyboardInput: true,
	}, computer.ActionRequest{
		Kind:      computer.ActionTypeText,
		App:       "editor",
		ElementID: "editor-box",
		Text:      "hello",
	})
	if err != nil {
		t.Fatalf("safe TYPE_TEXT rejected: %v", err)
	}
	if provider.snapshotCalls != 1 || provider.actionCalls != 1 {
		t.Fatalf("TYPE_TEXT preflight/action calls = %d/%d", provider.snapshotCalls, provider.actionCalls)
	}

	_, err = n.ComputerAct(context.Background(), runner.AutonomyContract{
		AllowComputerUse:   true,
		AllowKeyboardInput: true,
	}, computer.ActionRequest{
		Kind:      computer.ActionTypeText,
		App:       "editor",
		ElementID: "editor-box",
		Text:      strings.Repeat("x", 8193),
	})
	if err == nil || !strings.Contains(err.Error(), "8192") {
		t.Fatalf("oversized TYPE_TEXT must fail before provider call, got %v", err)
	}
	if provider.actionCalls != 1 {
		t.Fatal("oversized TYPE_TEXT reached native provider")
	}
}

func TestComputerActScrollAndDragRequireStructuredPointerInput(t *testing.T) {
	t.Setenv("NEXUS_ENABLE_COMPUTER_USE", "1")
	provider := availableComputerProvider()
	provider.capabilities.Scroll = true
	provider.capabilities.Drag = true
	provider.action = computer.ActionResult{
		Snapshot: computer.Snapshot{ID: "after", CapturedAt: time.Now().UTC()},
		Verification: computer.ActionVerification{State: "OBSERVED"},
	}
	withFakeComputerProvider(t, provider)
	n := &Nexus{}
	contract := runner.AutonomyContract{
		AllowComputerUse:  true,
		AllowPointerInput: true,
	}

	if _, err := n.ComputerAct(context.Background(), contract, computer.ActionRequest{
		Kind:      computer.ActionScroll,
		App:       "editor",
		Direction: "diagonal",
	}); err == nil || !strings.Contains(err.Error(), "UP, DOWN, LEFT or RIGHT") {
		t.Fatalf("invalid scroll direction must fail closed, got %v", err)
	}
	if provider.actionCalls != 0 {
		t.Fatal("invalid scroll reached native provider")
	}

	if _, err := n.ComputerAct(context.Background(), contract, computer.ActionRequest{
		Kind:      computer.ActionScroll,
		App:       "editor",
		Direction: "down",
	}); err != nil {
		t.Fatalf("valid scroll rejected: %v", err)
	}

	if _, err := n.ComputerAct(context.Background(), contract, computer.ActionRequest{
		Kind: computer.ActionDrag,
		App:  "editor",
		X:    intPtr(10),
		Y:    intPtr(20),
		ToX:  intPtr(30),
		ToY:  intPtr(40),
	}); err != nil {
		t.Fatalf("valid drag rejected: %v", err)
	}
	if provider.actionCalls != 2 {
		t.Fatalf("expected scroll + drag provider calls, got %d", provider.actionCalls)
	}
}


func TestComputerListAppsFiltersBlockedTargets(t *testing.T) {
	t.Setenv("NEXUS_ENABLE_COMPUTER_USE", "1")
	provider := availableComputerProvider()
	provider.capabilities.ListApps = true
	providerApps := []computer.AppInfo{
		{Name: "code", PID: 10},
		{Name: "1Password", PID: 20},
		{Name: "notepad", PID: 30},
	}
	fake := &computerListAppsFake{
		fakeComputerProvider: provider,
		apps:                 providerApps,
	}
	withFakeComputerProvider(t, fake)

	n := &Nexus{}
	apps, err := n.ComputerListApps(context.Background(), runner.AutonomyContract{
		AllowComputerUse: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(apps) != 2 || apps[0].Name != "code" || apps[1].Name != "notepad" {
		t.Fatalf("blocked apps were not filtered: %+v", apps)
	}
}

type computerListAppsFake struct {
	*fakeComputerProvider
	apps    []computer.AppInfo
	windows []computer.WindowInfo
}

func (p *computerListAppsFake) ListApps(context.Context) ([]computer.AppInfo, error) {
	return append([]computer.AppInfo(nil), p.apps...), nil
}

func (p *computerListAppsFake) ListWindows(context.Context, string) ([]computer.WindowInfo, error) {
	return append([]computer.WindowInfo(nil), p.windows...), nil
}

func TestComputerListWindowsRequiresAllowedApp(t *testing.T) {
	t.Setenv("NEXUS_ENABLE_COMPUTER_USE", "1")
	provider := availableComputerProvider()
	provider.capabilities.ListWindows = true
	fake := &computerListAppsFake{
		fakeComputerProvider: provider,
		windows: []computer.WindowInfo{{ID: "1", Title: "Editor", Width: 800, Height: 600}},
	}
	withFakeComputerProvider(t, fake)

	n := &Nexus{}
	if _, err := n.ComputerListWindows(context.Background(), runner.AutonomyContract{
		AllowComputerUse: true,
	}, "Bitwarden"); err == nil {
		t.Fatal("blocked app window discovery must fail closed")
	}
	windows, err := n.ComputerListWindows(context.Background(), runner.AutonomyContract{
		AllowComputerUse: true,
	}, "code")
	if err != nil {
		t.Fatal(err)
	}
	if len(windows) != 1 || windows[0].Title != "Editor" {
		t.Fatalf("unexpected windows: %+v", windows)
	}
}
