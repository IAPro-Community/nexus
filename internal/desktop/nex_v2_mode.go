package desktop

import (
	"errors"

	"github.com/IAPro-Community/nexus/internal/nexus"
)

const (
	// HUDPresenterLegacy identifies the existing Wails/React Guardian HUD surface.
	HUDPresenterLegacy = "legacy"
	// HUDPresenterNexV2 identifies the experimental Nex 2.0 desktop shell surface.
	HUDPresenterNexV2 = "nex_v2"
)

// ErrHUDPresenterConflict is returned when both legacy and Nex 2.0 presenters
// are requested simultaneously.
var ErrHUDPresenterConflict = errors.New("legacy HUD and Nex 2.0 HUD presenters are mutually exclusive")

// NexV2DesktopEnabled reports whether the experimental Nex 2.0 desktop shell
// is explicitly enabled. It fails closed (false) unless both the global and
// NEX_V2 feature switches are set to "true".
func NexV2DesktopEnabled() bool {
	return nexus.AdvancedFeatureEnabled(nexus.FeatureNexV2)
}

// ActiveHUDPresenter returns the single desktop presenter selected for this
// process. Rollback to the legacy surface requires only unsetting the NEX_V2
// flag and restarting nexus-hud; persisted legacy window state remains intact.
func ActiveHUDPresenter() string {
	if NexV2DesktopEnabled() {
		return HUDPresenterNexV2
	}
	return HUDPresenterLegacy
}

// ValidateHUDPresenterMutualExclusion enforces that legacy HUD and Nex 2.0 HUD
// cannot be active simultaneously inside the same host session.
func ValidateHUDPresenterMutualExclusion(legacyActive, nexV2Active bool) error {
	if legacyActive && nexV2Active {
		return ErrHUDPresenterConflict
	}
	return nil
}
