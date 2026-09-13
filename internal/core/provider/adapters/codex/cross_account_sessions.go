package codex

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/kivervinicius/ai-cli/internal/core/config"
	"github.com/kivervinicius/ai-cli/internal/core/security"
)

const (
	crossAccountMarkerFile = "nexus-cross-account-sessions.jsonl"
	// crossAccountHostSeedCap limits how many host rollouts we import so a
	// multi-hundred host store does not flood the profile picker.
	crossAccountHostSeedCap = 40
	// crossAccountSeedBudget bounds how long Prepare may spend walking sibling/
	// host rollouts. Launch must open the TUI immediately; resume seeding is
	// best-effort and must never look like a hang.
	crossAccountSeedBudget = 200 * time.Millisecond
)

type crossAccountSessionRecord struct {
	SessionID     string    `json:"session_id"`
	SourceProfile string    `json:"source_profile,omitempty"`
	SourcePath    string    `json:"source_path,omitempty"`
	AdoptedAt     time.Time `json:"adopted_at"`
}

type sessionIndexEntry struct {
	ID         string `json:"id"`
	ThreadName string `json:"thread_name"`
	UpdatedAt  string `json:"updated_at"`
	CWD        string `json:"cwd,omitempty"`
}

// sessionCandidateRoots returns Codex session trees to search for resume
// artifacts: sibling Nexus profiles first, then the host store. excludeHome
// skips the destination profile's own sessions tree.
func sessionCandidateRoots(excludeHome string) []string {
	var roots []string
	seen := map[string]bool{}
	add := func(dir string) {
		dir = strings.TrimSpace(dir)
		if dir == "" {
			return
		}
		real, err := filepath.EvalSymlinks(dir)
		if err != nil {
			real = dir
		}
		if excludeHome != "" {
			if config.FilesystemPathsEquivalent(real, excludeHome) || config.FilesystemPathWithin(excludeHome, real) {
				return
			}
			exclSessions := filepath.Join(excludeHome, "sessions")
			if config.FilesystemPathsEquivalent(real, exclSessions) || config.FilesystemPathWithin(exclSessions, real) {
				return
			}
		}
		if seen[real] {
			return
		}
		if fi, err := os.Stat(real); err != nil || !fi.IsDir() {
			return
		}
		seen[real] = true
		roots = append(roots, real)
	}

	if data, err := config.DataDir(); err == nil {
		codexRoot := filepath.Join(data, "profiles", "codex")
		entries, _ := os.ReadDir(codexRoot)
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			home := filepath.Join(codexRoot, e.Name(), "home")
			add(filepath.Join(home, "sessions"))
			add(filepath.Join(home, ".codex", "sessions"))
		}
	}
	if host := security.FindHostHome(); host != "" {
		add(filepath.Join(host, ".codex", "sessions"))
	}
	return roots
}

func hardlinkOrCopy(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0700); err != nil {
		return err
	}
	if _, err := os.Lstat(dst); err == nil {
		return nil
	}
	// Copy only: hardlinking would share an inode across accounts so a resume
	// in one profile could mutate/truncate another account's rollout history.
	return copyFile(src, dst)
}

func adoptSessionIntoProfile(profileName, sessionID string) error {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return fmt.Errorf("empty session id")
	}
	home, err := config.ProfileHome("codex", profileName)
	if err != nil {
		return err
	}
	destRoot := filepath.Join(home, "sessions")
	if err := os.MkdirAll(destRoot, 0700); err != nil {
		return err
	}

	if existing := findRolloutInTree(destRoot, sessionID); existing != "" {
		var idxs []string
		for _, root := range sessionCandidateRoots(home) {
			idxs = append(idxs, indexCandidatesForSessionsRoot(root)...)
		}
		mergeSessionIndexEntry(home, sessionID, idxs...)
		if shouldMarkCrossAccount(home, existing, "") {
			if err := recordCrossAccountSession(home, sessionID, profileNameFromSessionsPath(existing), existing); err != nil {
				return err
			}
		}
		return nil
	}
	// Legacy dual-tree: treat as already present for resume, but prefer canonical.
	if existing := findRolloutInTree(filepath.Join(home, ".codex", "sessions"), sessionID); existing != "" {
		if shouldMarkCrossAccount(home, existing, "") {
			_ = recordCrossAccountSession(home, sessionID, profileNameFromSessionsPath(existing), existing)
		}
		return nil
	}

	candidates := sessionCandidateRoots(home)
	src := ""
	srcRoot := ""
	for _, c := range candidates {
		if p := findRolloutInTree(c, sessionID); p != "" {
			src = p
			srcRoot = c
			break
		}
	}
	if src == "" {
		return fmt.Errorf("session %s not found for adoption", sessionID)
	}

	rel := filepath.Base(src)
	if srcRoot != "" {
		if r, err := filepath.Rel(srcRoot, src); err == nil && r != "" && !strings.HasPrefix(r, "..") {
			rel = r
		}
	}
	dst := filepath.Join(destRoot, rel)
	if err := hardlinkOrCopy(src, dst); err != nil {
		return err
	}
	sourceProfile := profileNameFromSessionsPath(src)
	if shouldMarkCrossAccount(home, src, sourceProfile) {
		if err := recordCrossAccountSession(home, sessionID, sourceProfile, src); err != nil {
			_ = os.Remove(dst)
			return fmt.Errorf("cross-account marker required after adopt: %w", err)
		}
	}
	mergeSessionIndexEntry(home, sessionID, indexCandidatesForSessionsRoot(srcRoot)...)
	return nil
}

func profileNameFromSessionsPath(path string) string {
	parts := strings.Split(filepath.ToSlash(path), "/")
	for i := 0; i < len(parts)-1; i++ {
		if parts[i] == "codex" && i+1 < len(parts) && parts[i+1] != "home" && parts[i+1] != "sessions" {
			return parts[i+1]
		}
	}
	return "host"
}

func indexCandidatesForSessionsRoot(sessionsRoot string) []string {
	parent := filepath.Dir(sessionsRoot) // home or .codex
	return []string{
		filepath.Join(parent, "session_index.jsonl"),
		filepath.Join(filepath.Dir(parent), "session_index.jsonl"),
		filepath.Join(parent, ".codex", "session_index.jsonl"),
	}
}

// seedCrossAccountSessions imports recent sibling/host rollouts into the active
// profile so the native Codex /resume picker can see CrossAccountResume threads.
// It is best-effort and budgeted: a multi-thousand rollout store must not delay
// interactive launch.
func seedCrossAccountSessions(profileName string) error {
	home, err := config.ProfileHome("codex", profileName)
	if err != nil {
		return err
	}
	destRoot := filepath.Join(home, "sessions")
	if err := os.MkdirAll(destRoot, 0700); err != nil {
		return err
	}
	// Profiles that already have local sessions skip the expensive sibling/host
	// walk on the hot launch path. Explicit adopt/resume still seeds on demand.
	if destHasRollouts(destRoot) {
		return nil
	}

	deadline := time.Now().Add(crossAccountSeedBudget)
	cwd, _ := os.Getwd()
	cutoff := time.Now().Add(-rolloutMaxAge)
	type cand struct {
		path       string
		root       string
		modTime    time.Time
		fromHost   bool
		cwdMatch   bool
		sessionID  string
		indexEntry *sessionIndexEntry
	}

	var sibling, host []cand
	seenDest := map[string]bool{}
collect:
	for _, root := range sessionCandidateRoots(home) {
		if time.Now().After(deadline) {
			break
		}
		fromHost := isHostSessionsRoot(root)
		_ = filepath.Walk(root, func(path string, fi os.FileInfo, err error) error {
			if time.Now().After(deadline) {
				return io.EOF
			}
			if err != nil || fi == nil || fi.IsDir() {
				return nil
			}
			name := fi.Name()
			if !strings.HasPrefix(name, "rollout-") || !strings.HasSuffix(name, ".jsonl") {
				return nil
			}
			if fi.ModTime().Before(cutoff) {
				return nil
			}
			sessionID := extractSessionIDFromRolloutName(name)
			if sessionID == "" || seenDest[sessionID] {
				return nil
			}
			seenDest[sessionID] = true
			entry := synthesizeIndexEntry(path, sessionID, fi.ModTime())
			cwdMatch := cwd != "" && entry != nil && entry.CWD != "" &&
				(entry.CWD == cwd || strings.HasPrefix(cwd, entry.CWD) || strings.HasPrefix(entry.CWD, cwd))
			item := cand{
				path: path, root: root, modTime: fi.ModTime(), fromHost: fromHost,
				cwdMatch: cwdMatch, sessionID: sessionID, indexEntry: entry,
			}
			if fromHost {
				host = append(host, item)
			} else {
				sibling = append(sibling, item)
			}
			return nil
		})
		if time.Now().After(deadline) {
			break collect
		}
	}

	sort.SliceStable(sibling, func(i, j int) bool { return sibling[i].modTime.After(sibling[j].modTime) })
	sort.SliceStable(host, func(i, j int) bool {
		if host[i].cwdMatch != host[j].cwdMatch {
			return host[i].cwdMatch
		}
		return host[i].modTime.After(host[j].modTime)
	})
	if len(host) > crossAccountHostSeedCap {
		host = host[:crossAccountHostSeedCap]
	}

	for _, item := range append(sibling, host...) {
		if time.Now().After(deadline) {
			break
		}
		rel := filepath.Base(item.path)
		if r, err := filepath.Rel(item.root, item.path); err == nil && r != "" && !strings.HasPrefix(r, "..") {
			rel = r
		}
		dst := filepath.Join(destRoot, rel)
		if err := hardlinkOrCopy(item.path, dst); err != nil {
			continue
		}
		sourceProfile := profileNameFromSessionsPath(item.path)
		if shouldMarkCrossAccount(home, item.path, sourceProfile) {
			if err := recordCrossAccountSession(home, item.sessionID, sourceProfile, item.path); err != nil {
				_ = os.Remove(dst)
				continue
			}
		}
		merged := false
		for _, idxPath := range indexCandidatesForSessionsRoot(item.root) {
			if mergeSessionIndexFromFile(home, item.sessionID, idxPath) {
				merged = true
				break
			}
		}
		if !merged && item.indexEntry != nil {
			_ = appendSessionIndexEntry(home, *item.indexEntry)
		}
	}
	return nil
}

func destHasRollouts(destRoot string) bool {
	found := false
	_ = filepath.WalkDir(destRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil || d == nil || d.IsDir() {
			return nil
		}
		name := d.Name()
		if strings.HasPrefix(name, "rollout-") && strings.HasSuffix(name, ".jsonl") {
			found = true
			return io.EOF
		}
		return nil
	})
	return found
}

// shouldMarkCrossAccount reports whether an imported rollout must be excluded
// from GetUsage for this profile. Same chatgpt_account_id keeps quota attribution.
func shouldMarkCrossAccount(destHome, sourcePath, sourceProfile string) bool {
	destAcct := readCodexAuthAccountID(filepath.Join(destHome, "auth.json"))
	if rolloutAcct := readRolloutAccountID(sourcePath); destAcct != "" && rolloutAcct != "" {
		return !strings.EqualFold(destAcct, rolloutAcct)
	}
	if sourceProfile == "host" {
		return true
	}
	if sourceProfile == "" {
		// Destination-local path without a verifiable foreign account id:
		// do not suppress quota for native sessions.
		return false
	}
	srcHome := ""
	parts := strings.Split(filepath.ToSlash(sourcePath), "/")
	for i := 0; i < len(parts)-1; i++ {
		if parts[i] == "codex" && i+1 < len(parts) && parts[i+1] != "home" && parts[i+1] != "sessions" {
			srcHome = filepath.Join(strings.Join(parts[:i+2], "/"), "home")
			break
		}
	}
	if srcHome == "" {
		return true
	}
	srcAcct := readCodexAuthAccountID(filepath.Join(srcHome, "auth.json"))
	if destAcct != "" && srcAcct != "" && strings.EqualFold(destAcct, srcAcct) {
		return false
	}
	// Unverifiable or mismatched sibling accounts: fail closed for quota.
	return true
}

func isHostSessionsRoot(root string) bool {
	host := security.FindHostHome()
	if host == "" {
		return false
	}
	hostSessions := filepath.Join(host, ".codex", "sessions")
	return config.FilesystemPathsEquivalent(root, hostSessions) || config.FilesystemPathWithin(hostSessions, root)
}

func synthesizeIndexEntry(rolloutPath, sessionID string, modTime time.Time) *sessionIndexEntry {
	entry := &sessionIndexEntry{
		ID:         sessionID,
		ThreadName: "Codex Thread " + shortSessionTitle(sessionID),
		UpdatedAt:  modTime.UTC().Format(time.RFC3339Nano),
	}
	f, err := os.Open(rolloutPath)
	if err != nil {
		return entry
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for i := 0; scanner.Scan() && i < 20; i++ {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.Contains(line, "session_meta") {
			continue
		}
		var ev struct {
			Timestamp string `json:"timestamp"`
			Type      string `json:"type"`
			Payload   struct {
				ID        string `json:"id"`
				SessionID string `json:"session_id"`
				CWD       string `json:"cwd"`
			} `json:"payload"`
		}
		if json.Unmarshal([]byte(line), &ev) != nil || ev.Type != "session_meta" {
			continue
		}
		// Filename session id is authoritative for resume/picker matching.
		// Only adopt payload ids when the filename did not already carry one.
		if strings.TrimSpace(sessionID) == "" {
			if id := strings.TrimSpace(ev.Payload.ID); id != "" {
				entry.ID = id
			} else if id := strings.TrimSpace(ev.Payload.SessionID); id != "" {
				entry.ID = id
			}
		}
		entry.CWD = strings.TrimSpace(ev.Payload.CWD)
		if ts := strings.TrimSpace(ev.Timestamp); ts != "" {
			entry.UpdatedAt = ts
		}
		break
	}
	return entry
}

func shortSessionTitle(id string) string {
	id = strings.TrimSpace(id)
	if len(id) > 8 {
		return id[:8]
	}
	return id
}

func mergeSessionIndexEntry(home, sessionID string, indexPaths ...string) {
	for _, p := range indexPaths {
		if mergeSessionIndexFromFile(home, sessionID, p) {
			return
		}
	}
}

func mergeSessionIndexFromFile(home, sessionID, indexPath string) bool {
	f, err := os.Open(indexPath)
	if err != nil {
		return false
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var entry sessionIndexEntry
		if json.Unmarshal([]byte(line), &entry) != nil || entry.ID == "" {
			continue
		}
		if entry.ID != sessionID && !strings.Contains(entry.ID, sessionID) && !strings.Contains(sessionID, entry.ID) {
			continue
		}
		_ = appendSessionIndexEntry(home, entry)
		return true
	}
	return false
}

func appendSessionIndexEntry(home string, entry sessionIndexEntry) error {
	if strings.TrimSpace(entry.ID) == "" {
		return fmt.Errorf("empty session index id")
	}
	path := filepath.Join(home, "session_index.jsonl")
	existing, _ := readSessionIndexMap(path)
	if prev, ok := existing[entry.ID]; ok {
		// Keep the newer updated_at when both exist.
		if !indexEntryNewer(entry, prev) {
			return nil
		}
	}
	existing[entry.ID] = entry
	return writeSessionIndexMap(path, existing)
}

func indexEntryNewer(a, b sessionIndexEntry) bool {
	ta, errA := time.Parse(time.RFC3339Nano, a.UpdatedAt)
	if errA != nil {
		ta, _ = time.Parse(time.RFC3339, a.UpdatedAt)
	}
	tb, errB := time.Parse(time.RFC3339Nano, b.UpdatedAt)
	if errB != nil {
		tb, _ = time.Parse(time.RFC3339, b.UpdatedAt)
	}
	if ta.IsZero() {
		return false
	}
	if tb.IsZero() {
		return true
	}
	return ta.After(tb)
}

func readSessionIndexMap(path string) (map[string]sessionIndexEntry, error) {
	out := map[string]sessionIndexEntry{}
	f, err := os.Open(path)
	if err != nil {
		return out, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var entry sessionIndexEntry
		if json.Unmarshal([]byte(line), &entry) != nil || entry.ID == "" {
			continue
		}
		if prev, ok := out[entry.ID]; !ok || indexEntryNewer(entry, prev) {
			out[entry.ID] = entry
		}
	}
	return out, nil
}

func writeSessionIndexMap(path string, entries map[string]sessionIndexEntry) error {
	list := make([]sessionIndexEntry, 0, len(entries))
	for _, e := range entries {
		list = append(list, e)
	}
	sort.SliceStable(list, func(i, j int) bool {
		return indexEntryNewer(list[i], list[j])
	})
	var b strings.Builder
	for _, e := range list {
		raw, err := json.Marshal(e)
		if err != nil {
			continue
		}
		b.Write(raw)
		b.WriteByte('\n')
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(b.String()), 0600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func recordCrossAccountSession(home, sessionID, sourceProfile, sourcePath string) error {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" || home == "" {
		return fmt.Errorf("missing session marker fields")
	}
	path := filepath.Join(home, crossAccountMarkerFile)
	existing := readCrossAccountSessionIDs(home)
	if existing[sessionID] {
		return nil
	}
	rec := crossAccountSessionRecord{
		SessionID:     sessionID,
		SourceProfile: sourceProfile,
		SourcePath:    sourcePath,
		AdoptedAt:     time.Now().UTC(),
	}
	raw, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(raw, '\n'))
	return err
}

func readCrossAccountSessionIDs(home string) map[string]bool {
	out := map[string]bool{}
	path := filepath.Join(home, crossAccountMarkerFile)
	f, err := os.Open(path)
	if err != nil {
		return out
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var rec crossAccountSessionRecord
		if json.Unmarshal([]byte(line), &rec) != nil || rec.SessionID == "" {
			continue
		}
		out[rec.SessionID] = true
	}
	return out
}

// isCrossAccountResumeSession reports whether a rollout under the profile home
// was imported for CrossAccountResume and must not drive quota attribution.
func isCrossAccountResumeSession(home, rolloutPath string) bool {
	if home == "" || rolloutPath == "" {
		return false
	}
	ids := readCrossAccountSessionIDs(home)
	if len(ids) == 0 {
		return false
	}
	sessionID := extractSessionIDFromRolloutName(filepath.Base(rolloutPath))
	if sessionID != "" && ids[sessionID] {
		return true
	}
	for id := range ids {
		if strings.Contains(filepath.Base(rolloutPath), id) {
			return true
		}
	}
	return false
}

// readRolloutAccountID extracts a chatgpt_account_id / account_id from a rollout
// when present. Many rollouts omit it; callers must treat empty as unverifiable.
func readRolloutAccountID(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for i := 0; scanner.Scan() && i < 80; i++ {
		line := scanner.Text()
		if !strings.Contains(line, "account_id") && !strings.Contains(line, "chatgpt_account_id") {
			continue
		}
		var probe map[string]json.RawMessage
		if json.Unmarshal([]byte(line), &probe) != nil {
			continue
		}
		// Flat or nested payload search.
		if id := findAccountIDInJSON([]byte(line)); id != "" {
			return id
		}
	}
	return ""
}

func findAccountIDInJSON(raw []byte) string {
	var any map[string]any
	if json.Unmarshal(raw, &any) != nil {
		return ""
	}
	return findAccountIDInMap(any)
}

func findAccountIDInMap(m map[string]any) string {
	for k, v := range m {
		lk := strings.ToLower(k)
		if lk == "chatgpt_account_id" || lk == "account_id" {
			if s, ok := v.(string); ok {
				s = strings.TrimSpace(s)
				if len(s) >= 8 {
					return s
				}
			}
		}
		switch typed := v.(type) {
		case map[string]any:
			if id := findAccountIDInMap(typed); id != "" {
				return id
			}
		}
	}
	return ""
}
