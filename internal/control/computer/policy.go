package computer

import (
	"fmt"
	"strings"
)

type Policy struct {
	AllowComputerUse      bool
	AllowScreenCapture    bool
	AllowKeyboardInput    bool
	AllowPointerInput     bool
	AllowWindowManagement bool
	BlockedApps           []string
}

func (p Policy) ValidateObservation() error {
	if !p.AllowComputerUse {
		return fmt.Errorf("computer use is not authorized")
	}
	return nil
}

func (p Policy) ValidateApp(app string) error {
	if err := p.ValidateObservation(); err != nil {
		return err
	}
	return p.validateApp(app)
}

func (p Policy) ValidateSnapshot(req SnapshotRequest) error {
	if !p.AllowComputerUse {
		return fmt.Errorf("computer use is not authorized")
	}
	if req.Screenshot && !p.AllowScreenCapture {
		return fmt.Errorf("screen capture is not authorized")
	}
	return p.validateApp(req.App)
}

func (p Policy) ValidateAction(req ActionRequest) error {
	if !p.AllowComputerUse {
		return fmt.Errorf("computer use is not authorized")
	}
	if err := p.validateApp(req.App); err != nil {
		return err
	}
	switch req.Kind {
	case ActionClick, ActionScroll, ActionDrag:
		if !p.AllowPointerInput {
			return fmt.Errorf("pointer input is not authorized")
		}
	case ActionSetValue, ActionTypeText, ActionPressKey, ActionHotkey, ActionPasteText:
		if !p.AllowKeyboardInput {
			return fmt.Errorf("keyboard/text input is not authorized")
		}
	case ActionFocusWindow, ActionMoveWindow, ActionResizeWindow:
		if !p.AllowWindowManagement {
			return fmt.Errorf("window management is not authorized")
		}
	default:
		return fmt.Errorf("unsupported computer action %q", req.Kind)
	}
	return nil
}

func (p Policy) validateApp(app string) error {
	needle := strings.ToLower(strings.TrimSpace(app))
	if needle == "" {
		return fmt.Errorf("computer target app is required")
	}
	blocked := p.BlockedApps
	if len(blocked) == 0 {
		blocked = []string{"1password", "bitwarden", "dashlane", "lastpass", "nordpass", "proton pass"}
	}
	for _, candidate := range blocked {
		if candidate = strings.ToLower(strings.TrimSpace(candidate)); candidate != "" && strings.Contains(needle, candidate) {
			return fmt.Errorf("computer target app is blocked by policy")
		}
	}
	return nil
}
