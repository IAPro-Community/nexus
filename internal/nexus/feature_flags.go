package nexus

import "os"

// AdvancedFeatureEnabled requires both global and feature-specific opt-ins.
// No default, alternate truthy string, or UI flag can start a managed job.
func AdvancedFeatureEnabled(feature string) bool {
	return os.Getenv("NEXUS_FEATURES_ENABLED") == "true" &&
		os.Getenv("NEXUS_FEATURE_"+feature) == "true"
}
