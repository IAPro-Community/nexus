package profile

import "github.com/IAPro-Community/nexus/internal/core/historysync"

// SyncProviderHistory synchronizes non-credential conversation history artifacts
// for link-mode providers (AGY, Claude, Gemini, OpenCode, Cursor).
func SyncProviderHistory(providerID, profileName, profileHome string) error {
	return historysync.SyncProviderHistory(providerID, profileName, profileHome)
}

// MergeAndLinkSharedPath safely links a profile history path to the shared store,
// non-destructively merging any pre-existing local profile content first.
func MergeAndLinkSharedPath(sharedPath, profilePath string) error {
	return historysync.MergeAndLinkSharedPath(sharedPath, profilePath)
}
