package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FilesystemIdentity identifies an existing filesystem object independently of
// textual aliases such as macOS /var and /private/var or Windows short names.
type FilesystemIdentity struct {
	Kind      string `json:"kind"`
	StableKey string `json:"stable_key"`
	Available bool   `json:"available"`
}

// PathRef keeps the user-facing spelling, canonical security path and stable
// identity together. DisplayPath must never be used for authorization.
type PathRef struct {
	DisplayPath   string             `json:"display_path"`
	CanonicalPath string             `json:"canonical_path"`
	Identity      FilesystemIdentity `json:"identity"`
}

// ResolvePathRef resolves a path and, when it exists, records its filesystem
// identity. Non-existing paths remain usable for creation flows but have no
// identity until an object exists on disk.
func ResolvePathRef(p string) (PathRef, error) {
	canonical, err := CanonicalWorkspacePath(p)
	if err != nil {
		return PathRef{}, err
	}
	ref := PathRef{DisplayPath: p, CanonicalPath: canonical}
	if identity, identityErr := filesystemIdentity(canonical); identityErr == nil {
		ref.Identity = identity
	}
	return ref, nil
}

// CanonicalWorkspacePath resolves and canonicalizes any workspace or project path:
// - Verifies non-empty
// - Computes absolute path
// - Cleans path separators and relative segments
// - Resolves filesystem symlinks (e.g. macOS /var -> /private/var)
// - Preserves clean canonical path
func CanonicalWorkspacePath(p string) (string, error) {
	if strings.TrimSpace(p) == "" {
		return "", errors.New("workspace path cannot be empty")
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", fmt.Errorf("resolve workspace path: %w", err)
	}
	clean := filepath.Clean(abs)
	if resolved, err := filepath.EvalSymlinks(clean); err == nil {
		clean = filepath.Clean(resolved)
	}
	return clean, nil
}

// CanonicalExistingWorkspaceDir verifies that the canonical path exists and is a directory.
func CanonicalExistingWorkspaceDir(p string) (string, error) {
	clean, err := CanonicalWorkspacePath(p)
	if err != nil {
		return "", err
	}
	fi, err := os.Stat(clean)
	if err != nil {
		return "", fmt.Errorf("workspace path does not exist: %w", err)
	}
	if !fi.IsDir() {
		return "", fmt.Errorf("workspace path is not a directory: %s", clean)
	}
	return clean, nil
}

// FilesystemPathsEquivalent reports whether two existing paths identify the
// same filesystem object. It falls back to canonical text for paths that do
// not exist yet, which keeps creation flows usable without weakening identity
// checks for existing objects.
func FilesystemPathsEquivalent(left, right string) bool {
	if left == right {
		return true
	}
	leftRef, leftErr := ResolvePathRef(left)
	rightRef, rightErr := ResolvePathRef(right)
	if leftErr != nil || rightErr != nil {
		return filepath.Clean(left) == filepath.Clean(right)
	}
	if leftRef.Identity.Available && rightRef.Identity.Available {
		return leftRef.Identity.Kind == rightRef.Identity.Kind && leftRef.Identity.StableKey == rightRef.Identity.StableKey
	}
	return leftRef.CanonicalPath == rightRef.CanonicalPath
}

// FilesystemPathWithin reports whether path is root itself or a descendant of
// root after platform-specific canonicalization.
func FilesystemPathWithin(root, path string) bool {
	rootRef, rootErr := ResolvePathRef(root)
	pathRef, pathErr := ResolvePathRef(path)
	if rootErr != nil || pathErr != nil {
		return false
	}
	if FilesystemPathsEquivalent(rootRef.CanonicalPath, pathRef.CanonicalPath) {
		return true
	}
	rel, err := filepath.Rel(rootRef.CanonicalPath, pathRef.CanonicalPath)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != "."
}
