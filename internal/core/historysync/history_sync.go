package historysync

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/provider/spec"
	"github.com/IAPro-Community/nexus/internal/core/security"
)

// ErrActiveSQLiteSession indicates a local profile directory currently holds
// open SQLite WAL/SHM/journal sidecar files and must not be replaced yet.
var ErrActiveSQLiteSession = errors.New("active sqlite WAL session present; deferring directory link")

// SyncProviderHistory synchronizes non-credential conversation history artifacts
// for link-mode providers (AGY, Claude) according to each provider's declarative
// HistorySyncPolicy.
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

	var errs []error
	for _, rel := range s.HistorySyncPolicy.SharedPaths {
		if isForbiddenCredentialPath(rel, s.HistorySyncPolicy.CredentialPaths) {
			return fmt.Errorf("refusing to sync forbidden credential path %q for %s", rel, s.ID)
		}
		sharedPath := filepath.Join(sharedRoot, filepath.FromSlash(rel))
		profilePath := filepath.Join(profileRoot, filepath.FromSlash(rel))
		if err := MergeAndLinkSharedPath(sharedPath, profilePath); err != nil {
			errs = append(errs, err)
		}
	}

	if s.ID == "agy" {
		invalidateStaleAGYSummaryCaches(sharedRoot, profileRoot)
	}
	return errors.Join(errs...)
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

	// Never remove or merge a local SQLite directory while WAL/SHM/journal files
	// show active handles on disk.
	if srcLErr == nil && srcLInfo.IsDir() && hasActiveSQLiteSidecars(profilePath) {
		return fmt.Errorf("%w: %s", ErrActiveSQLiteSession, profilePath)
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

func isSQLiteSidecar(name string) bool {
	lower := strings.ToLower(strings.TrimSpace(name))
	for _, suffix := range []string{
		".db-wal", ".db-shm", ".db-journal",
		".sqlite-wal", ".sqlite-shm", ".sqlite-journal",
		".sqlite3-wal", ".sqlite3-shm", ".sqlite3-journal",
	} {
		if strings.HasSuffix(lower, suffix) {
			return true
		}
	}
	return false
}

func hasActiveSQLiteSidecars(dir string) bool {
	info, err := os.Lstat(dir)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return false
	}
	found := false
	_ = filepath.WalkDir(dir, func(p string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.IsDir() && isSQLiteSidecar(d.Name()) {
			found = true
			return filepath.SkipAll
		}
		return nil
	})
	return found
}

func invalidateStaleAGYSummaryCaches(sharedRoot, profileRoot string) {
	cliSources := []string{
		filepath.Join(sharedRoot, "antigravity-cli", "conversation_summaries.db"),
		filepath.Join(sharedRoot, "antigravity-cli", "conversation_summaries.db-wal"),
		filepath.Join(sharedRoot, "antigravity-cli", "annotations"),
		filepath.Join(sharedRoot, "antigravity-cli", "conversations"),
	}
	invalidateIfStale(
		filepath.Join(profileRoot, "antigravity-cli", "jetbox_summaries_proto.pb"),
		filepath.Join(profileRoot, "antigravity-cli", "conversations"),
		cliSources,
	)

	ideSources := []string{
		filepath.Join(sharedRoot, "antigravity", "conversations"),
		filepath.Join(sharedRoot, "antigravity", "annotations"),
	}
	invalidateIfStale(
		filepath.Join(profileRoot, "antigravity", "agyhub_summaries_proto.pb"),
		filepath.Join(profileRoot, "antigravity", "conversations"),
		ideSources,
	)
}

func invalidateIfStale(cacheFile, localGuardDir string, sourcePaths []string) {
	cacheInfo, err := os.Stat(cacheFile)
	if err != nil || !cacheInfo.Mode().IsRegular() {
		return
	}
	if hasActiveSQLiteSidecars(localGuardDir) {
		return
	}
	latest := latestModTime(sourcePaths)
	if !latest.IsZero() && cacheInfo.ModTime().Before(latest) {
		_ = os.Remove(cacheFile)
	}
}

func latestModTime(paths []string) time.Time {
	var latest time.Time
	update := func(t time.Time) {
		if t.After(latest) {
			latest = t
		}
	}
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			continue
		}
		update(info.ModTime())
		if info.IsDir() {
			entries, err := os.ReadDir(p)
			if err != nil {
				continue
			}
			for _, entry := range entries {
				if eInfo, err := entry.Info(); err == nil {
					update(eInfo.ModTime())
				}
			}
		}
	}
	return latest
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
	case ".jsonl", ".json", ".db", ".sqlite", ".sqlite3", ".pbtxt", ".pb", ".txt", ".log":
		return true
	default:
		return false
	}
}

func mergePathInto(src, dst string) error {
	srcLInfo, err := os.Lstat(src)
	if err != nil {
		return err
	}
	if srcLInfo.Mode()&os.ModeSymlink != 0 {
		if _, err := os.Stat(src); err != nil {
			// Skip dangling symlinks instead of aborting the whole merge.
			return nil
		}
	}
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

	if isSQLiteSidecar(filepath.Base(src)) {
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
	if strings.HasSuffix(strings.ToLower(src), ".pbtxt") {
		return mergeAnnotationPBTXT(src, dst, srcInfo.ModTime().After(dstInfo.ModTime()))
	}
	if strings.EqualFold(filepath.Base(src), "sessions-index.json") {
		// Remove stale Claude sessions-index.json when merging two local project
		// dirs so Claude rebuilds an index covering all merged *.jsonl files.
		if err := os.Remove(dst); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	if srcInfo.ModTime().After(dstInfo.ModTime()) && srcInfo.Size() > 0 {
		return copyFileAtomic(src, dst)
	}
	return nil
}

type pbtxtField struct {
	key        string
	token      string
	hasValue   bool
	stringVal  string
	isQuotedKV bool
}

func mergeAnnotationPBTXT(src, dst string, srcNewer bool) error {
	srcBytes, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	dstBytes, err := os.ReadFile(dst)
	if err != nil {
		return err
	}

	dstFields := parsePBTXTFields(string(dstBytes))
	srcFields := parsePBTXTFields(string(srcBytes))
	if len(dstFields) == 0 && len(srcFields) == 0 {
		if srcNewer && len(srcBytes) > 0 {
			return copyFileAtomic(src, dst)
		}
		return nil
	}

	mergedMap := make(map[string]pbtxtField, len(dstFields)+len(srcFields))
	var order []string
	addOrUpdate := func(f pbtxtField, preferIncoming bool) {
		existing, exists := mergedMap[f.key]
		if !exists {
			mergedMap[f.key] = f
			order = append(order, f.key)
			return
		}
		if f.key == "title" {
			existingNonEmpty := strings.TrimSpace(existing.stringVal) != ""
			incomingNonEmpty := strings.TrimSpace(f.stringVal) != ""
			if !existingNonEmpty && incomingNonEmpty {
				mergedMap[f.key] = f
			} else if existingNonEmpty && incomingNonEmpty && preferIncoming {
				mergedMap[f.key] = f
			}
			return
		}
		if preferIncoming && strings.TrimSpace(f.token) != "" {
			mergedMap[f.key] = f
		}
	}

	for _, f := range dstFields {
		addOrUpdate(f, false)
	}
	for _, f := range srcFields {
		addOrUpdate(f, srcNewer)
	}

	parts := make([]string, 0, len(order))
	if titleField, ok := mergedMap["title"]; ok && strings.TrimSpace(titleField.token) != "" {
		parts = append(parts, titleField.token)
	}
	for _, k := range order {
		if k == "title" {
			continue
		}
		if f, ok := mergedMap[k]; ok && strings.TrimSpace(f.token) != "" {
			parts = append(parts, f.token)
		}
	}
	content := strings.Join(parts, "  ")
	if content == "" {
		return nil
	}
	return writeFileAtomic(dst, []byte(content))
}

func parsePBTXTFields(raw string) []pbtxtField {
	var fields []pbtxtField
	n := len(raw)
	i := 0
	for i < n {
		for i < n && unicode.IsSpace(rune(raw[i])) {
			i++
		}
		if i >= n {
			break
		}
		start := i
		for i < n && !unicode.IsSpace(rune(raw[i])) && raw[i] != ':' {
			i++
		}
		if i >= n || raw[i] != ':' {
			// Skip unkeyed/comment token up to end of line or whitespace.
			for i < n && !unicode.IsSpace(rune(raw[i])) {
				i++
			}
			continue
		}
		key := raw[start:i]
		i++ // consume ':'
		for i < n && (raw[i] == ' ' || raw[i] == '\t') {
			i++
		}
		if i >= n {
			break
		}

		var valEnd int
		var strVal string
		isQuoted := false
		switch raw[i] {
		case '"':
			isQuoted = true
			qStart := i
			i++
			for i < n {
				if raw[i] == '\\' && i+1 < n {
					i += 2
					continue
				}
				if raw[i] == '"' {
					i++
					break
				}
				i++
			}
			valEnd = i
			if unq, err := strconv.Unquote(raw[qStart:valEnd]); err == nil {
				strVal = unq
			} else {
				strVal = strings.Trim(raw[qStart:valEnd], `"`)
			}
		case '{':
			depth := 1
			i++
			for i < n && depth > 0 {
				if raw[i] == '"' {
					i++
					for i < n {
						if raw[i] == '\\' && i+1 < n {
							i += 2
							continue
						}
						if raw[i] == '"' {
							break
						}
						i++
					}
				} else if raw[i] == '{' {
					depth++
				} else if raw[i] == '}' {
					depth--
				}
				i++
			}
			valEnd = i
		default:
			for i < n && !unicode.IsSpace(rune(raw[i])) {
				i++
			}
			valEnd = i
			strVal = raw[i:valEnd]
		}

		fields = append(fields, pbtxtField{
			key:        key,
			token:      raw[start:valEnd],
			hasValue:   true,
			stringVal:  strVal,
			isQuotedKV: isQuoted,
		})
	}
	return fields
}

func writeFileAtomic(dst string, data []byte) error {
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
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, dst)
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
