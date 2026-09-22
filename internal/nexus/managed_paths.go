package nexus

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// validateManagedPathSegment prevents durable IDs from being reinterpreted as
// filesystem structure when Nexus materializes internal workspaces.
func validateManagedPathSegment(name, value string) error {
	value = strings.TrimSpace(value)
	if value == "" || value == "." || value == ".." ||
		strings.ContainsAny(value, `/\\`) || filepath.Base(value) != value {
		return fmt.Errorf("unsafe managed %s %q", name, value)
	}
	return nil
}

func pathWithinRoot(path, root string) bool {
	absPath, err1 := filepath.Abs(path)
	absRoot, err2 := filepath.Abs(root)
	if err1 != nil || err2 != nil {
		return false
	}
	rel, err := filepath.Rel(absRoot, absPath)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}

func canonicalExistingManagedDir(path string) (string, error) {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", err
	}
	return filepath.Abs(resolved)
}

func rejectManagedSymlink(path, label string) error {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%s must not be a symlink: %s", label, path)
	}
	return nil
}
