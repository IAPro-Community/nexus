package spec

import (
	sdkprovider "github.com/IAPro-Community/IAPro-SDK/providers/spec"
)

func init() {
	// Gemini CLI has been discontinued; remove it from Nexus provider discovery
	// while retaining AGY's "gemini" model/quota group.
	sdkprovider.Unregister("gemini")

	// Extend AGY with conversation metadata/annotation paths and Antigravity IDE
	// ("Other" tab) trees so /resume displays unified titles, flags, and sessions
	// across all authenticated AGY accounts.
	_ = sdkprovider.Extend("agy", func(s *sdkprovider.Spec) {
		s.HistorySyncPolicy.SharedPaths = appendUniqueStrings(
			s.HistorySyncPolicy.SharedPaths,
			"antigravity-cli/annotations",
			"antigravity-cli/implicit",
			"antigravity/conversations",
			"antigravity/brain",
			"antigravity/annotations",
			"antigravity/implicit",
		)
		s.HistorySyncPolicy.CredentialPaths = appendUniqueStrings(
			s.HistorySyncPolicy.CredentialPaths,
			"antigravity-cli/jetski_state.pbtxt",
			"antigravity/antigravity_state.pbtxt",
			"antigravity/installation_id",
		)
	})

	// Multi-account history sharing is currently scoped to AGY, Claude, and Codex.
	// OpenCode's modern SQLite store (opencode.db) mixes account/credential
	// tables and must never be symlinked across profiles.
	_ = sdkprovider.Extend("opencode", func(s *sdkprovider.Spec) {
		s.HistorySyncPolicy.Mode = sdkprovider.HistorySyncNone
		s.HistorySyncPolicy.CredentialPaths = appendUniqueStrings(
			s.HistorySyncPolicy.CredentialPaths,
			"opencode.db",
		)
	})

	_ = sdkprovider.Extend("cursor", func(s *sdkprovider.Spec) {
		s.HistorySyncPolicy = sdkprovider.HistorySyncPolicy{
			Mode:            sdkprovider.HistorySyncNone,
			RootSubdir:      ".cursor",
			CredentialPaths: []string{"auth.json", "cli-config.json"},
		}
	})
}

func appendUniqueStrings(base []string, extra ...string) []string {
	seen := make(map[string]struct{}, len(base)+len(extra))
	out := make([]string, 0, len(base)+len(extra))
	for _, item := range base {
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	for _, item := range extra {
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}
