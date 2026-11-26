package historysync

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/provider/spec"
	"github.com/IAPro-Community/nexus/internal/core/security"
)

// SyncProviderHistory synchronizes non-credential conversation history artifacts
// for link-mode providers (AGY, Claude, Gemini, OpenCode, Cursor) according to
// each provider's declarative HistorySyncPolicy.
func SyncProviderHistory(providerID, profileName, profileHome string) error {
	s, ok := spec.Get(providerID)
	if !ok || s.HistorySyncPolicy.Mode != spec.HistorySyncLink {
		return nil
	}
	if strings.TrimSpace(profileHome) == "" {
		return fmt.Errorf("profile home is required for history sync")
	}

	sharedBase := security.FindHostHome()
	if sharedBase == "" || filepath.Clean(sharedBase) == filepath.Clean(profileHome) {
		dataDir, err := config.DataDir()
		if err != nil {
			return err
		}
		sharedBase = filepath.Join(dataDir, "shared-history", s.ID)
	}

	sharedRoot := sharedBase
	profileRoot := profileHome
	if sub := strings.TrimSpace(s.HistorySyncPolicy.RootSubdir); sub != "" {
		sharedRoot = filepath.Join(sharedBase, filepath.FromSlash(sub))
		profileRoot = filepath.Join(profileHome, filepath.FromSlash(sub))
	}

	var firstErr error
	for _, rel := range s.HistorySyncPolicy.SharedPaths {
		if isForbiddenCredentialPath(rel, s.HistorySyncPolicy.CredentialPaths) {
			return fmt.Errorf("refusing to sync forbidden credential path %q for %s", rel, s.ID)
		}
		sharedPath := filepath.Join(sharedRoot, filepath.FromSlash(rel))
		profilePath := filepath.Join(profileRoot, filepath.FromSlash(rel))
		if err := MergeAndLinkSharedPath(sharedPath, profilePath); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// MergeAndLinkSharedPath safely links profilePath -> sharedPath, merging any
// pre-existing local profile data into sharedPath before replacing it with a link.
func MergeAndLinkSharedPath(sharedPath, profilePath string) error {
	if isSymlinkTo(profilePath, sharedPath) {
		return nil
	}

	dstInfo, dstErr := os.Stat(sharedPath)
	srcLInfo, srcLErr := os.Lstat(profilePath)

	// Already a symlink pointing elsewhere: replace link without touching old target.
	if srcLErr == nil && srcLInfo.Mode()&os.ModeSymlink != 0 {
		_ = os.Remove(profilePath)
		srcLErr = os.ErrNotExist
	}

	if os.IsNotExist(dstErr) {
		if srcLErr == nil {
			if err := mergePathInto(profilePath, sharedPath); err != nil {
				return err
			}
			if err := os.RemoveAll(profilePath); err != nil {
				return err
			}
			return security.SafeLinkOrCopy(sharedPath, profilePath)
		}
		if !looksLikeFilePath(sharedPath) {
			if err := os.MkdirAll(sharedPath, 0700); err != nil {
				return err
			}
			return security.SafeLinkOrCopy(sharedPath, profilePath)
		}
		return nil
	}
	if dstErr != nil {
		return dstErr
	}

	if srcLErr == nil {
		if srcLInfo.IsDir() != dstInfo.IsDir() {
			return fmt.Errorf("cannot merge %s into %s: file/directory type mismatch", profilePath, sharedPath)
		}
		if err := mergePathInto(profilePath, sharedPath); err != nil {
			return err
		}
		if err := os.RemoveAll(profilePath); err != nil {
			return err
		}
	}
	return security.SafeLinkOrCopy(sharedPath, profilePath)
}

func isForbiddenCredentialPath(candidate string, forbidden []string) bool {
	cand := cleanSlashPath(candidate)
	if cand == "" || cand == "." {
		return true
	}
	for _, cred := range forbidden {
		c := cleanSlashPath(cred)
		if c == "" {
			continue
		}
		if cand == c || strings.HasPrefix(cand, c+"/") || strings.HasPrefix(c, cand+"/") {
			return true
		}
	}
	return false
}

func cleanSlashPath(p string) string {
	cleaned := path.Clean(strings.TrimSpace(strings.ReplaceAll(p, "\\", "/")))
	cleaned = strings.TrimPrefix(cleaned, "./")
	return strings.TrimSuffix(cleaned, "/")
}

func isSymlinkTo(linkPath, targetPath string) bool {
	info, err := os.Lstat(linkPath)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		return false
	}
	resolved, err := os.Readlink(linkPath)
	if err != nil {
		return false
	}
	if !filepath.IsAbs(resolved) {
		resolved = filepath.Join(filepath.Dir(linkPath), resolved)
	}
	return filepath.Clean(resolved) == filepath.Clean(targetPath)
}

func looksLikeFilePath(p string) bool {
	ext := strings.ToLower(filepath.Ext(filepath.Base(p)))
	switch ext {
	case ".jsonl", ".json", ".db", ".sqlite", ".sqlite3", ".pbtxt", ".txt", ".log":
		return true
	default:
		return false
	}
}

func mergePathInto(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	if srcInfo.IsDir() {
		if err := os.MkdirAll(dst, 0700); err != nil {
			return err
		}
		entries, err := os.ReadDir(src)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if err := mergePathInto(filepath.Join(src, entry.Name()), filepath.Join(dst, entry.Name())); err != nil {
				return err
			}
		}
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0700); err != nil {
		return err
	}
	dstInfo, dstErr := os.Stat(dst)
	if os.IsNotExist(dstErr) {
		return copyFileAtomic(src, dst)
	}
	if dstErr != nil {
		return dstErr
	}
	if strings.HasSuffix(strings.ToLower(src), ".jsonl") {
		return unionMergeJSONL(src, dst)
	}
	if srcInfo.ModTime().After(dstInfo.ModTime()) && srcInfo.Size() > 0 {
		return copyFileAtomic(src, dst)
	}
	return nil
}

func copyFileAtomic(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	tmp, err := os.CreateTemp(filepath.Dir(dst), ".historysync-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if err := tmp.Chmod(0600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := io.Copy(tmp, in); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, dst)
}

func unionMergeJSONL(src, dst string) error {
	seen := make(map[string]struct{})
	var lines []string

	readLines := func(path string) error {
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if line == "" {
				continue
			}
			if _, ok := seen[line]; !ok {
				seen[line] = struct{}{}
				lines = append(lines, line)
			}
		}
		return sc.Err()
	}

	if err := readLines(dst); err != nil {
		return err
	}
	if err := readLines(src); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(filepath.Dir(dst), ".historysync-jsonl-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0600); err != nil {
		_ = tmp.Close()
		return err
	}
	for _, line := range lines {
		if _, err := fmt.Fprintln(tmp, line); err != nil {
			_ = tmp.Close()
			return err
		}
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, dst)
}
