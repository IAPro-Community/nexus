package update

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ArtifactTarget defines the type and extraction behavior for an artifact.
type ArtifactTarget string

const (
	// TargetBinary is a single executable binary (no extraction needed).
	TargetBinary ArtifactTarget = "binary"
	// TargetTarGz is a gzipped tar archive (common on Linux/macOS).
	TargetTarGz ArtifactTarget = "tar.gz"
	// TargetZip is a ZIP archive (common on Windows).
	TargetZip ArtifactTarget = "zip"
	// TargetNSIS is a Windows NSIS installer (manual download required).
	TargetNSIS ArtifactTarget = "nsis"
	// TargetDEB is a Debian package (manual install required).
	TargetDEB ArtifactTarget = "deb"
	// TargetRPM is an RPM package (manual install required).
	TargetRPM ArtifactTarget = "rpm"
)

type Artifact struct {
	URL       string         `json:"url"`
	Size      int64          `json:"size"`
	SHA256    string         `json:"sha256"`
	Signature string         `json:"signature,omitempty"`
	Target    ArtifactTarget `json:"target,omitempty"`
}

// ExtractAction returns the recommended extraction action for the artifact type.
// Returns "replace" for self-updateable artifacts, "manual" for package managers.
func (a Artifact) ExtractAction() string {
	switch a.Target {
	case TargetBinary, TargetTarGz, TargetZip:
		return "replace"
	case TargetNSIS, TargetDEB, TargetRPM:
		return "manual"
	default:
		return "replace" // backward compatible default
	}
}

type Manifest struct {
	SchemaVersion     int                 `json:"schema_version"`
	Channel           string              `json:"channel"`
	Version           string              `json:"version"`
	ReleaseDate       string              `json:"release_date"`
	ExpiresAt         string              `json:"expires_at,omitempty"`
	MinNexusVersion   string              `json:"min_nexus_version,omitempty"`
	MinMaestroVersion string              `json:"min_maestro_version,omitempty"`
	KeyID             string              `json:"key_id"`
	Changelog         string              `json:"changelog,omitempty"`
	Artifacts         map[string]Artifact `json:"artifacts,omitempty"`
}

type ManifestPolicy struct {
	Channel        string
	CurrentVersion string
	NexusVersion   string
	MaestroVersion string
	Target         string
	Now            time.Time
}

var (
	ErrManifestExpired       = errors.New("update manifest has expired")
	ErrManifestChannel       = errors.New("update manifest channel is not allowed")
	ErrManifestDowngrade     = errors.New("update manifest would downgrade the installed version")
	ErrManifestCompatibility = errors.New("installed version is incompatible with update manifest")
	ErrManifestTarget        = errors.New("update manifest has no artifact for target")
	ErrManifestArtifact      = errors.New("update manifest artifact is invalid")
	ErrManifestUnsigned      = errors.New("update manifest signature is missing")
)

// Validate applies local policy before an artifact can be downloaded or installed.
// Empty policy fields intentionally skip that dimension for backwards-compatible
// manifest inspection; callers that install updates should always provide them.
func (m Manifest) Validate(policy ManifestPolicy) error {
	if m.SchemaVersion != 1 {
		return fmt.Errorf("unsupported manifest schema version: %d", m.SchemaVersion)
	}
	if strings.TrimSpace(m.Channel) == "" || strings.TrimSpace(m.Version) == "" {
		return errors.New("manifest channel and version are required")
	}
	if policy.Channel != "" && !strings.EqualFold(policy.Channel, m.Channel) {
		return fmt.Errorf("%w: expected %s, got %s", ErrManifestChannel, policy.Channel, m.Channel)
	}
	now := policy.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if m.ExpiresAt != "" {
		expires, err := time.Parse(time.RFC3339, m.ExpiresAt)
		if err != nil {
			return fmt.Errorf("invalid expires_at: %w", err)
		}
		if !expires.After(now) {
			return ErrManifestExpired
		}
	}
	if policy.CurrentVersion != "" && compareVersions(m.Version, policy.CurrentVersion) < 0 {
		return fmt.Errorf("%w: installed=%s manifest=%s", ErrManifestDowngrade, policy.CurrentVersion, m.Version)
	}
	if policy.NexusVersion != "" && m.MinNexusVersion != "" && compareVersions(policy.NexusVersion, m.MinNexusVersion) < 0 {
		return fmt.Errorf("%w: Nexus %s requires %s", ErrManifestCompatibility, m.MinNexusVersion, policy.NexusVersion)
	}
	if policy.MaestroVersion != "" && m.MinMaestroVersion != "" && compareVersions(policy.MaestroVersion, m.MinMaestroVersion) < 0 {
		return fmt.Errorf("%w: Maestro %s requires %s", ErrManifestCompatibility, m.MinMaestroVersion, policy.MaestroVersion)
	}
	if policy.Target != "" {
		artifact, ok := m.Artifacts[policy.Target]
		if !ok {
			return fmt.Errorf("%w: %s", ErrManifestTarget, policy.Target)
		}
		if artifact.Size < 0 || len(artifact.SHA256) != 64 {
			return fmt.Errorf("%w: %s", ErrManifestArtifact, policy.Target)
		}
		if artifact.Target != "" && artifact.Target != TargetBinary && artifact.Target != TargetTarGz && artifact.Target != TargetZip && artifact.Target != TargetNSIS && artifact.Target != TargetDEB && artifact.Target != TargetRPM {
			return fmt.Errorf("%w: %s has unsupported artifact target %q", ErrManifestArtifact, policy.Target, artifact.Target)
		}
		if _, err := hex.DecodeString(artifact.SHA256); err != nil {
			return fmt.Errorf("%w: %s checksum: %v", ErrManifestArtifact, policy.Target, err)
		}
	}
	return nil
}

func compareVersions(a, b string) int {
	type semver struct {
		major, minor, patch int
		prerelease          string // empty = stable (highest precedence)
	}

	parse := func(v string) semver {
		v = strings.TrimPrefix(strings.TrimSpace(v), "v")
		var s semver
		// Split into core + optional prerelease
		core := v
		if idx := strings.IndexAny(v, "-+"); idx != -1 {
			core = v[:idx]
			s.prerelease = v[idx+1:]
			// Only use the part after '-' (prerelease), ignore build metadata after '+'
			if plusIdx := strings.Index(s.prerelease, "+"); plusIdx != -1 {
				s.prerelease = s.prerelease[:plusIdx]
			}
		}
		segments := strings.Split(core, ".")
		for i := 0; i < len(segments) && i < 3; i++ {
			val, _ := strconv.Atoi(segments[i])
			switch i {
			case 0:
				s.major = val
			case 1:
				s.minor = val
			case 2:
				s.patch = val
			}
		}
		return s
	}

	left, right := parse(a), parse(b)

	// Compare major.minor.patch
	if left.major != right.major {
		if left.major < right.major {
			return -1
		}
		return 1
	}
	if left.minor != right.minor {
		if left.minor < right.minor {
			return -1
		}
		return 1
	}
	if left.patch != right.patch {
		if left.patch < right.patch {
			return -1
		}
		return 1
	}

	// Same major.minor.patch: stable > any prerelease (SemVer 2.0.0 §11)
	if left.prerelease == "" && right.prerelease == "" {
		return 0
	}
	if left.prerelease == "" {
		return 1 // stable > prerelease
	}
	if right.prerelease == "" {
		return -1 // prerelease < stable
	}

	// Both have prerelease: compare dot-separated identifiers numerically
	// then lexically, per SemVer 2.0.0 §11
	leftParts := strings.Split(left.prerelease, ".")
	rightParts := strings.Split(right.prerelease, ".")
	maxLen := len(leftParts)
	if len(rightParts) > maxLen {
		maxLen = len(rightParts)
	}

	for i := 0; i < maxLen; i++ {
		if i >= len(leftParts) {
			return -1 // shorter prerelease has lower precedence
		}
		if i >= len(rightParts) {
			return 1
		}
		lp, rp := leftParts[i], rightParts[i]
		ln, lErr := strconv.Atoi(lp)
		rn, rErr := strconv.Atoi(rp)
		if lErr == nil && rErr == nil {
			// Both numeric: compare numerically
			if ln < rn {
				return -1
			}
			if ln > rn {
				return 1
			}
		} else {
			// At least one is alphanumeric: compare lexically
			if lp < rp {
				return -1
			}
			if lp > rp {
				return 1
			}
		}
	}
	return 0
}
