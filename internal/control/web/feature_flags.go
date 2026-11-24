package web

import (
	"net/http"
	"strings"

	"github.com/IAPro-Community/nexus/internal/nexus"
)

// Optional product capabilities are off in every installation unless both
// the global switch AND the individual switch are explicitly true.
// Never use a UI/localStorage query parameter as authorization to run them.
const featureComposer = "COMPOSER"
const featureFlow = "FLOW"
const featureMissions = "MISSIONS"
const featureParallelChallenge = "PARALLEL_CHALLENGE"
const featureComparisonLab = "COMPARISON_LAB"
const featureVoiceActions = "VOICE_ACTIONS"
const featureNexProactivity = "NEX_PROACTIVITY"
const featureDesignInspect = "DESIGN_INSPECT"
const featureVisualDiff = "VISUAL_DIFF"

func optionalFeatureEnabled(feature string) bool {
	return nexus.AdvancedFeatureEnabled(feature)
}

func optionalFeatureUnavailable(w http.ResponseWriter, feature string) {
	writeError(w, http.StatusNotFound,
		"feature "+strings.ToLower(feature)+" is disabled; enable NEXUS_FEATURES_ENABLED=true and NEXUS_FEATURE_"+feature+"=true explicitly")
}

// The existing execution history stays readable and cancelable when a feature
// is retired. Only NEW mutations require an opt-in. Do not block shutdown.
func optionalFeatureMutationBlocked(w http.ResponseWriter, r *http.Request, feature string) bool {
	if optionalFeatureEnabled(feature) || isReadOnlyMethod(r.Method) {
		return false
	}
	// Only exact shutdown routes may bypass the admission gate.
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if r.Method == http.MethodPost && feature == featureMissions {
		if len(parts) == 5 && parts[0] == "api" && parts[1] == "v1" && parts[2] == "runs" &&
			(parts[4] == "cancel" || parts[4] == "pause") {
			return false
		}
		if len(parts) == 7 && parts[0] == "api" && parts[1] == "v1" && parts[2] == "projects" &&
			parts[4] == "maestro" && parts[5] == "submissions" && parts[6] == "cancel" {
			return false
		}
	}
	optionalFeatureUnavailable(w, feature)
	return true
}

func requireOptionalFeature(feature string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !optionalFeatureEnabled(feature) {
			optionalFeatureUnavailable(w, feature)
			return
		}
		next(w, r)
	}
}

func requireOptionalMutation(feature string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if optionalFeatureMutationBlocked(w, r, feature) {
			return
		}
		next(w, r)
	}
}

func featureForProjectMutation(path string) string {
	switch {
	case strings.HasSuffix(path, "/composer-sessions"),
		strings.HasSuffix(path, "/maestro/compose-preview"):
		return featureComposer
	case strings.HasSuffix(path, "/plans"),
		strings.HasSuffix(path, "/maestro/route-preview"):
		return featureFlow
	case strings.HasSuffix(path, "/maestro/runs"),
		strings.HasSuffix(path, "/maestro/submissions"),
		strings.HasSuffix(path, "/maestro/submissions/cancel"):
		return featureMissions
	}
	return ""
}

// Browser core remains available; advanced operations require explicit opt-in.
func browserAdvancedFeatureGate(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/inspect"):
			if !optionalFeatureEnabled(featureDesignInspect) {
				optionalFeatureUnavailable(w, featureDesignInspect)
				return
			}
		case strings.HasSuffix(r.URL.Path, "/visual-diff"):
			if !optionalFeatureEnabled(featureVisualDiff) {
				optionalFeatureUnavailable(w, featureVisualDiff)
				return
			}
		}
		next(w, r)
	}
}
