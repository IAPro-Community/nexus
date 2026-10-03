package computer

import "testing"

func TestPolicySeparatesObservationAndMutation(t *testing.T) {
	p := Policy{AllowComputerUse: true, AllowScreenCapture: true}
	if err := p.ValidateSnapshot(SnapshotRequest{App: "editor", Screenshot: true}); err != nil {
		t.Fatalf("authorized snapshot rejected: %v", err)
	}
	if err := p.ValidateAction(ActionRequest{Kind: ActionClick, App: "editor"}); err == nil {
		t.Fatal("pointer input must not be implied by observation permission")
	}
	if err := p.ValidateAction(ActionRequest{Kind: ActionTypeText, App: "editor"}); err == nil {
		t.Fatal("keyboard input must not be implied by observation permission")
	}
}

func TestPolicyBlocksPasswordManagers(t *testing.T) {
	p := Policy{AllowComputerUse: true, AllowScreenCapture: true, AllowPointerInput: true}
	if err := p.ValidateSnapshot(SnapshotRequest{App: "1Password", Screenshot: true}); err == nil {
		t.Fatal("password managers must be blocked by default")
	}
	if err := p.ValidateAction(ActionRequest{Kind: ActionClick, App: "Bitwarden"}); err == nil {
		t.Fatal("password managers must be blocked for actions")
	}
}

func TestPolicyRequiresGranularWindowPermission(t *testing.T) {
	p := Policy{AllowComputerUse: true}
	if err := p.ValidateAction(ActionRequest{Kind: ActionFocusWindow, App: "editor"}); err == nil {
		t.Fatal("window management must require explicit permission")
	}
	p.AllowWindowManagement = true
	if err := p.ValidateAction(ActionRequest{Kind: ActionFocusWindow, App: "editor"}); err != nil {
		t.Fatalf("explicit window permission rejected: %v", err)
	}
}
