package profile

import "strings"

// ResolveDefault applies the shared profile selection policy:
// explicit profile, best authenticated candidate, configured default, first
// candidate, and finally the conventional "default" profile.
func ResolveDefault(provider, explicit string, candidates []string, configuredDefault string, selectBest func() (string, error)) string {
	_ = provider

	if explicit = strings.TrimSpace(explicit); explicit != "" {
		return explicit
	}
	if selectBest != nil {
		if selected, err := selectBest(); err == nil && containsCandidate(candidates, selected) {
			return selected
		}
	}
	if containsCandidate(candidates, configuredDefault) {
		return configuredDefault
	}
	if len(candidates) > 0 {
		return candidates[0]
	}
	return "default"
}

func containsCandidate(candidates []string, name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	for _, candidate := range candidates {
		if candidate == name {
			return true
		}
	}
	return false
}
