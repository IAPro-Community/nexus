package desktop

import (
	"errors"
	"testing"
)

func TestNexV2DesktopFlagDefaultsOnWithOptOutAndMutualExclusion(t *testing.T) {
	t.Setenv("NEXUS_FEATURES_ENABLED", "")
	t.Setenv("NEXUS_FEATURE_NEX_V2", "")
	if !NexV2DesktopEnabled() {
		t.Fatal("Nex 2.0 desktop flag must default on")
	}
	if got := ActiveHUDPresenter(); got != HUDPresenterNexV2 {
		t.Fatalf("default presenter=%q, want %q", got, HUDPresenterNexV2)
	}

	t.Setenv("NEXUS_FEATURES_ENABLED", "false")
	if !NexV2DesktopEnabled() {
		t.Fatal("NEX_V2 must remain enabled independently of NEXUS_FEATURES_ENABLED")
	}

	t.Setenv("NEXUS_FEATURE_NEX_V2", "false")
	if NexV2DesktopEnabled() {
		t.Fatal("NEXUS_FEATURE_NEX_V2=false must roll back to legacy HUD")
	}
	if got := ActiveHUDPresenter(); got != HUDPresenterLegacy {
		t.Fatalf("opt-out presenter=%q, want %q", got, HUDPresenterLegacy)
	}

	t.Setenv("NEXUS_FEATURE_NEX_V2", "true")
	if !NexV2DesktopEnabled() {
		t.Fatal("explicit NEX_V2=true must keep Nex 2.0 presenter enabled")
	}
	if got := ActiveHUDPresenter(); got != HUDPresenterNexV2 {
		t.Fatalf("enabled presenter=%q, want %q", got, HUDPresenterNexV2)
	}

	if err := ValidateHUDPresenterMutualExclusion(true, true); !errors.Is(err, ErrHUDPresenterConflict) {
		t.Fatalf("expected mutual exclusion conflict, got %v", err)
	}
	if err := ValidateHUDPresenterMutualExclusion(true, false); err != nil {
		t.Fatalf("unexpected error for legacy-only: %v", err)
	}
	if err := ValidateHUDPresenterMutualExclusion(false, true); err != nil {
		t.Fatalf("unexpected error for nex_v2-only: %v", err)
	}
}
