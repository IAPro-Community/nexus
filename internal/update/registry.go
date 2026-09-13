package update

import (
	"fmt"
	"os"
	"strings"
)

const (
	// DefaultUpdateRegistryRepo is the canonical GitHub owner/name for Nexus
	// channel pointer releases (IAPRO community product identity).
	DefaultUpdateRegistryRepo = "IAPRO/nexus"

	channelStableTag = "channel-stable"
	channelBetaTag   = "channel-beta"
	manifestFileName = "update-manifest.json"
)

// DefaultUpdateRegistryURL is the stable-channel discovery URL when no env
// overrides apply. Prefer ResolveRegistryURL / DefaultManifestURL for channel-aware use.
var DefaultUpdateRegistryURL = DefaultManifestURL(DefaultUpdateRegistryRepo, "stable")

// ChannelPointerTag returns the movable GitHub release tag for a channel.
func ChannelPointerTag(channel string) string {
	if strings.EqualFold(strings.TrimSpace(channel), "beta") {
		return channelBetaTag
	}
	return channelStableTag
}

// DefaultManifestURL builds the GitHub Releases download URL for a channel pointer.
func DefaultManifestURL(repo, channel string) string {
	repo = strings.Trim(strings.TrimSpace(repo), "/")
	if repo == "" {
		repo = DefaultUpdateRegistryRepo
	}
	return fmt.Sprintf(
		"https://github.com/%s/releases/download/%s/%s",
		repo,
		ChannelPointerTag(channel),
		manifestFileName,
	)
}

// ResolveRegistryURL picks the update discovery URL.
// Precedence: explicit argument, then NEXUS_UPDATE_REGISTRY_URL, then
// NEXUS_RELEASE_REPO + channel pointer, then IAPRO/nexus + channel pointer.
func ResolveRegistryURL(explicit, channel string) string {
	if u := strings.TrimSpace(explicit); u != "" {
		return u
	}
	if u := strings.TrimSpace(os.Getenv("NEXUS_UPDATE_REGISTRY_URL")); u != "" {
		return u
	}
	repo := strings.TrimSpace(os.Getenv("NEXUS_RELEASE_REPO"))
	return DefaultManifestURL(repo, channel)
}
