package nexus

import (
	"errors"
	"fmt"
	"os"
)

// The catalog is the only registry of experimental product capabilities.
// Unknown identifiers and omitted environment variables always fail closed.
const (
	FeatureComposer          = "COMPOSER"
	FeatureFlow              = "FLOW"
	FeatureMissions          = "MISSIONS"
	FeatureParallelChallenge = "PARALLEL_CHALLENGE"
	FeatureComparisonLab     = "COMPARISON_LAB"
	FeatureVoiceActions      = "VOICE_ACTIONS"
	FeatureNexProactivity    = "NEX_PROACTIVITY"
	FeatureDesignInspect     = "DESIGN_INSPECT"
	FeatureVisualDiff        = "VISUAL_DIFF"
	FeatureSmartLayouts      = "SMART_LAYOUTS"
	FeatureDevicePreview     = "DEVICE_PREVIEW"
)

var advancedFeatures = [...]string{
	FeatureComposer, FeatureFlow, FeatureMissions, FeatureParallelChallenge,
	FeatureComparisonLab, FeatureVoiceActions, FeatureNexProactivity,
	FeatureDesignInspect, FeatureVisualDiff, FeatureSmartLayouts, FeatureDevicePreview,
}

var ErrAdvancedFeatureDisabled = errors.New("advanced feature disabled")

func AdvancedFeatureEnabled(feature string) bool {
	if os.Getenv("NEXUS_FEATURES_ENABLED") != "true" {
		return false
	}
	for _, known := range advancedFeatures {
		if feature == known {
			return os.Getenv("NEXUS_FEATURE_"+feature) == "true"
		}
	}
	return false
}

func RequireAdvancedFeature(feature string) error {
	if AdvancedFeatureEnabled(feature) {
		return nil
	}
	return fmt.Errorf("%w: %s (set NEXUS_FEATURES_ENABLED=true and NEXUS_FEATURE_%s=true)", ErrAdvancedFeatureDisabled, feature, feature)
}

// EffectiveAdvancedFeatures returns capability values only, never the process
// environment. A new map is returned for each caller.
type EffectiveAdvancedFeatures struct {
	Global   bool            `json:"global"`
	Features map[string]bool `json:"features"`
}

func AdvancedFeatureSnapshot() EffectiveAdvancedFeatures {
	global := os.Getenv("NEXUS_FEATURES_ENABLED") == "true"
	features := make(map[string]bool, len(advancedFeatures))
	for _, feature := range advancedFeatures {
		features[feature] = global && os.Getenv("NEXUS_FEATURE_"+feature) == "true"
	}
	return EffectiveAdvancedFeatures{Global: global, Features: features}
}
