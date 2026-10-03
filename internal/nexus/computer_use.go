package nexus

import (
	"context"
	"fmt"
	"strings"

	"github.com/IAPro-Community/nexus/internal/control/computer"
	coreflags "github.com/IAPro-Community/nexus/internal/core/flags"
	"github.com/IAPro-Community/nexus/internal/core/security"
	"github.com/IAPro-Community/nexus/internal/nexus/runner"
)

var computerProviderFactory = func() computer.Provider {
	return computer.NewLocalProvider()
}

type ComputerUseStatus struct {
	FeatureEnabled bool                      `json:"feature_enabled"`
	Capabilities   computer.Capabilities     `json:"capabilities"`
	Permissions    computer.PermissionStatus `json:"permissions"`
}

func computerPolicyFromContract(contract runner.AutonomyContract) computer.Policy {
	return computer.Policy{
		AllowComputerUse:      contract.AllowComputerUse,
		AllowScreenCapture:    contract.AllowScreenCapture,
		AllowKeyboardInput:    contract.AllowKeyboardInput,
		AllowPointerInput:     contract.AllowPointerInput,
		AllowWindowManagement: contract.AllowWindowManagement,
	}
}

func (n *Nexus) ComputerUseStatus(ctx context.Context) (ComputerUseStatus, error) {
	provider := computerProviderFactory()
	capabilities, err := provider.Capabilities(ctx)
	if err != nil {
		return ComputerUseStatus{}, err
	}
	permissions, err := provider.Permissions(ctx)
	if err != nil {
		return ComputerUseStatus{}, err
	}
	return ComputerUseStatus{
		FeatureEnabled: coreflags.Enabled(coreflags.ComputerUse),
		Capabilities:   capabilities,
		Permissions:    permissions,
	}, nil
}

func (n *Nexus) ComputerListApps(
	ctx context.Context,
	contract runner.AutonomyContract,
) ([]computer.AppInfo, error) {
	if !coreflags.Enabled(coreflags.ComputerUse) {
		return nil, fmt.Errorf("computer use is disabled by feature flag")
	}
	policy := computerPolicyFromContract(contract)
	if err := policy.ValidateObservation(); err != nil {
		return nil, err
	}
	provider := computerProviderFactory()
	capabilities, _, err := computerProviderReady(ctx, provider)
	if err != nil {
		return nil, err
	}
	if !capabilities.ListApps {
		return nil, fmt.Errorf("computer-use provider does not support application discovery")
	}
	apps, err := provider.ListApps(ctx)
	if err != nil {
		return nil, fmt.Errorf("list computer applications: %w", err)
	}
	filtered := make([]computer.AppInfo, 0, len(apps))
	for _, app := range apps {
		if policy.ValidateApp(app.Name) != nil {
			continue
		}
		app.Name = security.Redact(strings.TrimSpace(app.Name))
		app.BundleID = security.Redact(strings.TrimSpace(app.BundleID))
		filtered = append(filtered, app)
	}
	return filtered, nil
}

func (n *Nexus) ComputerListWindows(
	ctx context.Context,
	contract runner.AutonomyContract,
	app string,
) ([]computer.WindowInfo, error) {
	if !coreflags.Enabled(coreflags.ComputerUse) {
		return nil, fmt.Errorf("computer use is disabled by feature flag")
	}
	app = strings.TrimSpace(app)
	policy := computerPolicyFromContract(contract)
	if err := policy.ValidateApp(app); err != nil {
		return nil, err
	}
	provider := computerProviderFactory()
	capabilities, _, err := computerProviderReady(ctx, provider)
	if err != nil {
		return nil, err
	}
	if !capabilities.ListWindows {
		return nil, fmt.Errorf("computer-use provider does not support window discovery")
	}
	windows, err := provider.ListWindows(ctx, app)
	if err != nil {
		return nil, fmt.Errorf("list computer windows: %w", err)
	}
	for i := range windows {
		windows[i].Title = security.Redact(strings.TrimSpace(windows[i].Title))
	}
	return windows, nil
}

func (n *Nexus) ComputerSnapshot(
	ctx context.Context,
	contract runner.AutonomyContract,
	req computer.SnapshotRequest,
) (*computer.Snapshot, error) {
	if !coreflags.Enabled(coreflags.ComputerUse) {
		return nil, fmt.Errorf("computer use is disabled by feature flag")
	}
	policy := computerPolicyFromContract(contract)
	if err := policy.ValidateSnapshot(req); err != nil {
		return nil, err
	}

	provider := computerProviderFactory()
	capabilities, permissions, err := computerProviderReady(ctx, provider)
	if err != nil {
		return nil, err
	}
	if !capabilities.AccessibilityTree {
		return nil, fmt.Errorf("computer-use provider does not support accessibility snapshots")
	}
	if req.Screenshot && !capabilities.Screenshot {
		return nil, fmt.Errorf("computer-use provider does not support screenshots")
	}
	if req.Screenshot && strings.EqualFold(strings.TrimSpace(permissions.ScreenCapture), "DENIED") {
		return nil, fmt.Errorf("screen capture permission is denied")
	}

	snapshot, err := provider.Snapshot(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("capture computer snapshot: %w", err)
	}
	sanitizeComputerSnapshot(&snapshot)
	return &snapshot, nil
}

func (n *Nexus) ComputerAct(
	ctx context.Context,
	contract runner.AutonomyContract,
	req computer.ActionRequest,
) (*computer.ActionResult, error) {
	if !coreflags.Enabled(coreflags.ComputerUse) {
		return nil, fmt.Errorf("computer use is disabled by feature flag")
	}
	policy := computerPolicyFromContract(contract)
	if err := policy.ValidateAction(req); err != nil {
		return nil, err
	}
	if err := validateComputerActionShape(req); err != nil {
		return nil, err
	}

	provider := computerProviderFactory()
	capabilities, _, err := computerProviderReady(ctx, provider)
	if err != nil {
		return nil, err
	}
	if err := requireComputerActionCapability(capabilities, req.Kind); err != nil {
		return nil, err
	}

	if computerActionWritesElementValue(req.Kind) {
		snapshot, err := provider.Snapshot(ctx, computer.SnapshotRequest{
			App:         req.App,
			WindowID:    req.WindowID,
			WindowIndex: req.WindowIndex,
			Screenshot:  false,
			Restore:     req.Restore,
		})
		if err != nil {
			return nil, fmt.Errorf("inspect computer target before mutation: %w", err)
		}
		element, err := findComputerElement(snapshot.Elements, req.ElementID, req.ElementIndex)
		if err != nil {
			return nil, err
		}
		if element.Sensitive {
			return nil, fmt.Errorf("computer-use refuses to write sensitive/password controls")
		}
	}

	result, err := provider.Act(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("perform computer action: %w", err)
	}
	sanitizeComputerSnapshot(&result.Snapshot)
	result.Verification.Reason = security.Redact(result.Verification.Reason)
	return &result, nil
}

func computerProviderReady(
	ctx context.Context,
	provider computer.Provider,
) (computer.Capabilities, computer.PermissionStatus, error) {
	if provider == nil {
		return computer.Capabilities{}, computer.PermissionStatus{}, fmt.Errorf("computer-use provider is unavailable")
	}
	capabilities, err := provider.Capabilities(ctx)
	if err != nil {
		return computer.Capabilities{}, computer.PermissionStatus{}, fmt.Errorf("read computer-use capabilities: %w", err)
	}
	permissions, err := provider.Permissions(ctx)
	if err != nil {
		return computer.Capabilities{}, computer.PermissionStatus{}, fmt.Errorf("read computer-use permissions: %w", err)
	}
	if !permissions.Available {
		reason := security.Redact(strings.TrimSpace(permissions.Reason))
		if reason == "" {
			reason = "native accessibility provider is unavailable"
		}
		return computer.Capabilities{}, permissions, fmt.Errorf("computer-use provider unavailable: %s", reason)
	}
	return capabilities, permissions, nil
}

func requireComputerActionCapability(capabilities computer.Capabilities, kind computer.ActionKind) error {
	supported := false
	switch kind {
	case computer.ActionClick:
		supported = capabilities.Click
	case computer.ActionSetValue:
		supported = capabilities.SetValue
	case computer.ActionTypeText:
		supported = capabilities.TypeText
	case computer.ActionPressKey:
		supported = capabilities.PressKey
	case computer.ActionHotkey:
		supported = capabilities.Hotkey
	case computer.ActionPasteText:
		supported = capabilities.PasteText
	case computer.ActionScroll:
		supported = capabilities.Scroll
	case computer.ActionDrag:
		supported = capabilities.Drag
	case computer.ActionFocusWindow, computer.ActionMoveWindow, computer.ActionResizeWindow:
		supported = capabilities.WindowManagement
	}
	if !supported {
		return fmt.Errorf("computer-use provider does not support action %s", kind)
	}
	return nil
}

func validateComputerActionShape(req computer.ActionRequest) error {
	hasElement := strings.TrimSpace(req.ElementID) != "" || req.ElementIndex != nil
	if len(req.App) > 512 {
		return fmt.Errorf("computer target app exceeds 512 bytes")
	}
	if len(req.Text) > 8192 {
		return fmt.Errorf("%s text exceeds 8192 bytes", req.Kind)
	}
	switch req.Kind {
	case computer.ActionClick:
		if !hasElement && (req.X == nil || req.Y == nil) {
			return fmt.Errorf("CLICK requires an element or x/y coordinates")
		}
	case computer.ActionSetValue, computer.ActionTypeText, computer.ActionPasteText:
		if !hasElement {
			return fmt.Errorf("%s requires a target element", req.Kind)
		}
	case computer.ActionPressKey:
		if strings.TrimSpace(req.Key) == "" {
			return fmt.Errorf("PRESS_KEY requires key")
		}
	case computer.ActionHotkey:
		if strings.TrimSpace(req.Key) == "" {
			return fmt.Errorf("HOTKEY requires key")
		}
	case computer.ActionScroll:
		switch strings.ToUpper(strings.TrimSpace(req.Direction)) {
		case "UP", "DOWN", "LEFT", "RIGHT":
		default:
			return fmt.Errorf("SCROLL direction must be UP, DOWN, LEFT or RIGHT")
		}
	case computer.ActionDrag:
		if req.X == nil || req.Y == nil || req.ToX == nil || req.ToY == nil {
			return fmt.Errorf("DRAG requires x/y/to_x/to_y")
		}
	case computer.ActionMoveWindow:
		if req.X == nil || req.Y == nil {
			return fmt.Errorf("MOVE_WINDOW requires x/y")
		}
	case computer.ActionResizeWindow:
		if req.Width == nil || req.Height == nil || *req.Width <= 0 || *req.Height <= 0 {
			return fmt.Errorf("RESIZE_WINDOW requires positive width/height")
		}
	case computer.ActionFocusWindow:
	default:
		return fmt.Errorf("unsupported computer action %q", req.Kind)
	}
	return nil
}

func computerActionWritesElementValue(kind computer.ActionKind) bool {
	switch kind {
	case computer.ActionSetValue, computer.ActionTypeText, computer.ActionPasteText:
		return true
	default:
		return false
	}
}

func findComputerElement(
	elements []computer.Element,
	elementID string,
	elementIndex *int,
) (computer.Element, error) {
	elementID = strings.TrimSpace(elementID)
	if elementID != "" {
		for _, element := range elements {
			if element.ID == elementID {
				return element, nil
			}
		}
		return computer.Element{}, fmt.Errorf("computer target element %q is no longer present", elementID)
	}
	if elementIndex != nil {
		for _, element := range elements {
			if element.Index == *elementIndex {
				return element, nil
			}
		}
		return computer.Element{}, fmt.Errorf("computer target element index %d is no longer present", *elementIndex)
	}
	return computer.Element{}, fmt.Errorf("computer target element is required")
}

func sanitizeComputerSnapshot(snapshot *computer.Snapshot) {
	if snapshot == nil {
		return
	}
	snapshot.App.Name = security.Redact(snapshot.App.Name)
	snapshot.App.BundleID = security.Redact(snapshot.App.BundleID)
	snapshot.Window.Title = security.Redact(snapshot.Window.Title)
	snapshot.TreeText = security.Redact(snapshot.TreeText)
	for i := range snapshot.Elements {
		element := &snapshot.Elements[i]
		element.Name = security.Redact(element.Name)
		if element.Sensitive {
			element.Value = ""
		} else {
			element.Value = security.Redact(element.Value)
		}
	}
}
