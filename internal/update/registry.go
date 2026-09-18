package update

import (
	"fmt"
	"os"
	"strings"
)

const (
	// DefaultUpdateRegistryRepo is the canonical binaries-only repository
	// for Nexus channel pointer releases. Closed-beta clients must never
	// touch the private source repository: all manifests and artifacts
	// resolve here unless NEXUS_RELEASE_REPO overrides it (dev/internal).
	DefaultUpdateRegistryRepo = "IAPro-Community/nexus-releases"

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

// ─────────────────────────────────────────────────────────────────────
// ArtifactRegistry: distribution-backend abstraction (Fase 11).
//
// The updater core must never depend semantically on the source-code
// repository. It consumes a signed channel manifest plus artifact URLs,
// wherever they are hosted. GitHub Releases (on a binaries-only repo) is
// one backend; R2/S3/CDN or a first-party service can replace it without
// touching the updater core.
// ─────────────────────────────────────────────────────────────────────

// ArtifactRegistry resolves the signed channel-manifest location for an
// update channel on a specific distribution backend.
type ArtifactRegistry interface {
	// ManifestURL returns the manifest location for channel.
	ManifestURL(channel string) (string, error)
	// BackendName identifies the backend for logs/diagnostics (no secrets).
	BackendName() string
}

// GitHubReleasesRegistry serves channel manifests from GitHub Releases of a
// binaries-only repository. The source repository is never referenced here.
type GitHubReleasesRegistry struct {
	// Repo is owner/name; empty falls back to DefaultUpdateRegistryRepo.
	Repo string
}

// ManifestURL implements ArtifactRegistry.
func (g GitHubReleasesRegistry) ManifestURL(channel string) (string, error) {
	return DefaultManifestURL(g.Repo, channel), nil
}

// BackendName implements ArtifactRegistry.
func (g GitHubReleasesRegistry) BackendName() string {
	repo := strings.Trim(strings.TrimSpace(g.Repo), "/")
	if repo == "" {
		repo = DefaultUpdateRegistryRepo
	}
	return "github-releases:" + repo
}

// StaticURLRegistry serves the channel manifest from a fixed HTTPS URL
// (storage bucket, CDN, or first-party update service).
type StaticURLRegistry struct {
	URL string
}

// ManifestURL implements ArtifactRegistry.
func (s StaticURLRegistry) ManifestURL(_ string) (string, error) {
	u := strings.TrimSpace(s.URL)
	if u == "" {
		return "", fmt.Errorf("static artifact registry URL is empty")
	}
	if !strings.HasPrefix(u, "https://") && !strings.HasPrefix(u, "http://") {
		return "", fmt.Errorf("static artifact registry URL must be http(s): %q", u)
	}
	return u, nil
}

// BackendName implements ArtifactRegistry.
func (s StaticURLRegistry) BackendName() string { return "static-url" }

// ResolveArtifactRegistry picks the distribution backend.
// Precedence: explicit URL argument, then NEXUS_UPDATE_REGISTRY_URL, then
// NEXUS_RELEASE_REPO (GitHub backend), then the default binaries registry.
func ResolveArtifactRegistry(explicit, _ string) ArtifactRegistry {
	if u := strings.TrimSpace(explicit); u != "" {
		return StaticURLRegistry{URL: u}
	}
	if u := strings.TrimSpace(os.Getenv("NEXUS_UPDATE_REGISTRY_URL")); u != "" {
		return StaticURLRegistry{URL: u}
	}
	return GitHubReleasesRegistry{Repo: strings.TrimSpace(os.Getenv("NEXUS_RELEASE_REPO"))}
}

// ResolveRegistryURL picks the update discovery URL.
// Precedence: explicit argument, then NEXUS_UPDATE_REGISTRY_URL, then
// NEXUS_RELEASE_REPO + channel pointer, then the binaries-only default.
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
