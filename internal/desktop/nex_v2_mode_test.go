package desktop

import (
	"errors"
	"testing"
)

func TestNexV2DesktopFlagDefaultsOffAndMutualExclusion(t *testing.T) {
	t.Setenv("NEXUS_FEATURES_ENABLED", "")
	t.Setenv("NEXUS_FEATURE_NEX_V2", "")
	if NexV2DesktopEnabled() {
		t.Fatal("Nex 2.0 desktop flag must default off")
	}
	if got := ActiveHUDPresenter(); got != HUDPresenterLegacy {
		t.Fatalf("default presenter=%q, want %q", got, HUDPresenterLegacy)
	}

	t.Setenv("NEXUS_FEATURE_NEX_V2", "true")
	if NexV2DesktopEnabled() {
		t.Fatal("NEX_V2 without global switch must remain disabled")
	}

	t.Setenv("NEXUS_FEATURES_ENABLED", "true")
	t.Setenv("NEXUS_FEATURE_NEX_V2", "1")
	if NexV2DesktopEnabled() {
		t.Fatal("only explicit true is accepted for NEX_V2")
	}

	t.Setenv("NEXUS_FEATURE_NEX_V2", "true")
	if !NexV2DesktopEnabled() {
		t.Fatal("explicit global + NEX_V2 opt-in must enable Nex 2.0 presenter")
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
