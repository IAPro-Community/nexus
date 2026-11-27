package doctor

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/config"
	_ "modernc.org/sqlite"
)

var (
	// Strict allowlist of regenerable compiler/browser build caches inside
	// `<profile>/home/.cache/`. Credential/tool state dirs like `antigravity`,
	// `gh`, and `copilot` are NEVER in this list.
	allowlistedHomeCacheDirs = map[string]struct{}{
		"go-build":               {},
		"go-tmp":                 {},
		"npm":                    {},
		"yarn":                   {},
		"pnpm":                   {},
		"pip":                    {},
		"uv":                     {},
		"node-gyp":               {},
		"golangci-lint":          {},
		"ms-playwright":          {},
		"Cypress":                {},
		"electron":               {},
		"electron-builder":       {},
		"google-chrome-headless": {},
		"node":                   {},
		"bun":                    {},
	}

	// Strict allowlist of regenerable package-manager caches inside
	// `<profile>/home/.local/share/`.
	allowlistedLocalShareCacheDirs = map[string]struct{}{
		"yarn": {},
	}
)

// PruneOptions controls `PruneProfileStorage` safety bounds.
type PruneOptions struct {
	DryRun               bool
	IncludeGhostProfiles bool
	MinTempAge           time.Duration
}

// StorageCandidate describes one allowlisted cache, leaked temp artifact, or
// unreferenced ghost profile eligible for cleanup.
type StorageCandidate struct {
	Kind       string `json:"kind"` // "build_cache" | "temp_leak" | "ghost_profile" | "stale_web_auth"
	Provider   string `json:"provider,omitempty"`
	Profile    string `json:"profile,omitempty"`
	Path       string `json:"path"`
	Bytes      int64  `json:"bytes"`
	Referenced bool   `json:"referenced,omitempty"`
	Removed    bool   `json:"removed"`
	Skipped    string `json:"skipped,omitempty"`
}

// ProfileStorageUsage summarizes real non-symlink disk footprint per profile.
type ProfileStorageUsage struct {
	Provider      string `json:"provider"`
	Profile       string `json:"profile"`
	Path          string `json:"path"`
	TotalBytes    int64  `json:"total_bytes"`
	Reclaimable   int64  `json:"reclaimable_bytes"`
	ValidName     bool   `json:"valid_name"`
	Referenced    bool   `json:"referenced"`
	SymlinksFound int    `json:"symlinks_found"`
	LeakedTempCnt int    `json:"leaked_temp_count"`
}

// StorageReport summarizes profile storage, shared caches, and cleanup candidates.
type StorageReport struct {
	DataDir             string                `json:"data_dir"`
	TotalBytes          int64                 `json:"total_bytes"`
	ReclaimableBytes    int64                 `json:"reclaimable_bytes"`
	FreedBytes          int64                 `json:"freed_bytes"`
	DryRun              bool                  `json:"dry_run"`
	Profiles            []ProfileStorageUsage `json:"profiles"`
	Candidates          []StorageCandidate    `json:"candidates"`
	StaleWebAuthRemoved int                   `json:"stale_web_auth_removed"`
}

// InspectProfileStorage scans managed profiles and `web-auth/` without
// following any symlink (`os.Lstat` only).
func InspectProfileStorage(dataDir string) (StorageReport, error) {
	return PruneProfileStorage(dataDir, PruneOptions{
		DryRun:               true,
		IncludeGhostProfiles: true,
		MinTempAge:           time.Hour,
	})
}

// PruneProfileStorage reports and (when opts.DryRun is false) removes strictly
// allowlisted regenerable build caches, leaked 0-byte temp files, orphaned
// `.quota-probe-*` directories, dead `web-auth/listen-*.json` files, and
// unreferenced invalid-name ghost profiles. It NEVER follows symlinks and NEVER
// touches `.codex`, `.claude`, `.gemini`, `.cursor`, or `.cache` tool auth.
func PruneProfileStorage(dataDir string, opts PruneOptions) (StorageReport, error) {
	if strings.TrimSpace(dataDir) == "" {
		resolved, err := config.DataDir()
		if err != nil {
			return StorageReport{}, err
		}
		dataDir = resolved
	}
	if opts.MinTempAge <= 0 {
		opts.MinTempAge = time.Hour
	}

	report := StorageReport{
		DataDir:    dataDir,
		DryRun:     opts.DryRun,
		Profiles:   []ProfileStorageUsage{},
		Candidates: []StorageCandidate{},
	}

	profilesRoot := filepath.Join(dataDir, "profiles")
	providerEntries, err := os.ReadDir(profilesRoot)
	if err != nil && !os.IsNotExist(err) {
		return report, err
	}

	now := time.Now()
	for _, provEntry := range providerEntries {
		provName := provEntry.Name()
		provPath := filepath.Join(profilesRoot, provName)
		if info, err := os.Lstat(provPath); err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			continue
		}
		profEntries, err := os.ReadDir(provPath)
		if err != nil {
			continue
		}
		for _, profEntry := range profEntries {
			profName := profEntry.Name()
			profRoot := filepath.Join(provPath, profName)
			info, err := os.Lstat(profRoot)
			if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
				continue
			}

			totalBytes, symlinks := lstatDirSize(profRoot)
			validName := config.ValidateProfileName(profName) == nil
			referenced := false
			if !validName {
				referenced = isGhostProfileReferenced(dataDir, provName, profName)
			}

			usage := ProfileStorageUsage{
				Provider:      provName,
				Profile:       profName,
				Path:          profRoot,
				TotalBytes:    totalBytes,
				ValidName:     validName,
				Referenced:    referenced,
				SymlinksFound: symlinks,
			}

			if !validName {
				cand := StorageCandidate{
					Kind:       "ghost_profile",
					Provider:   provName,
					Profile:    profName,
					Path:       profRoot,
					Bytes:      totalBytes,
					Referenced: referenced,
				}
				switch {
				case referenced:
					cand.Skipped = "referenced in config, runtimes, or database"
				case !opts.IncludeGhostProfiles:
					cand.Skipped = "ghost profile removal disabled"
				default:
					usage.Reclaimable += totalBytes
					report.ReclaimableBytes += totalBytes
					if !opts.DryRun {
						if rmErr := os.RemoveAll(profRoot); rmErr == nil {
							cand.Removed = true
							report.FreedBytes += totalBytes
						} else {
							cand.Skipped = rmErr.Error()
						}
					}
				}
				report.Candidates = append(report.Candidates, cand)
				report.TotalBytes += totalBytes
				report.Profiles = append(report.Profiles, usage)
				continue
			}

			// 1. Leaked 0-byte account-scope.tmp.* files & .quota-probe-* dirs in profile root.
			if rootItems, err := os.ReadDir(profRoot); err == nil {
				for _, item := range rootItems {
					name := item.Name()
					itemPath := filepath.Join(profRoot, name)
					st, lerr := os.Lstat(itemPath)
					if lerr != nil || st.Mode()&os.ModeSymlink != 0 {
						continue
					}
					isTempScope := !st.IsDir() && strings.HasPrefix(name, "account-scope.tmp.") && st.Size() == 0
					isProbeDir := st.IsDir() && strings.HasPrefix(name, ".quota-probe-")
					if !isTempScope && !isProbeDir {
						continue
					}
					if now.Sub(st.ModTime()) < opts.MinTempAge {
						continue
					}
					size := st.Size()
					if st.IsDir() {
						size, _ = lstatDirSize(itemPath)
					}
					usage.LeakedTempCnt++
					usage.Reclaimable += size
					report.ReclaimableBytes += size
					cand := StorageCandidate{
						Kind:     "temp_leak",
						Provider: provName,
						Profile:  profName,
						Path:     itemPath,
						Bytes:    size,
					}
					if !opts.DryRun {
						if rmErr := os.RemoveAll(itemPath); rmErr == nil {
							cand.Removed = true
							report.FreedBytes += size
						} else {
							cand.Skipped = rmErr.Error()
						}
					}
					report.Candidates = append(report.Candidates, cand)
				}
			}

			// 2. Allowlisted caches inside `<profile>/home/.cache/`.
			cacheRoot := filepath.Join(profRoot, "home", ".cache")
			pruneAllowlistedDirChildren(cacheRoot, allowlistedHomeCacheDirs, provName, profName, opts.DryRun, &usage, &report)

			// 3. Allowlisted package-manager cache inside `<profile>/home/.local/share/`.
			localShareRoot := filepath.Join(profRoot, "home", ".local", "share")
			pruneAllowlistedDirChildren(localShareRoot, allowlistedLocalShareCacheDirs, provName, profName, opts.DryRun, &usage, &report)

			report.TotalBytes += totalBytes
			report.Profiles = append(report.Profiles, usage)
		}
	}

	// 4. Stale dead-PID files in `<dataDir>/web-auth/listen-*.json`.
	webAuthDir := filepath.Join(dataDir, "web-auth")
	if authEntries, err := os.ReadDir(webAuthDir); err == nil {
		for _, entry := range authEntries {
			name := entry.Name()
			if !strings.HasPrefix(name, "listen-") || !strings.HasSuffix(name, ".json") {
				continue
			}
			fullPath := filepath.Join(webAuthDir, name)
			st, lerr := os.Lstat(fullPath)
			if lerr != nil || !st.Mode().IsRegular() || st.Mode()&os.ModeSymlink != 0 {
				continue
			}
			raw, rerr := os.ReadFile(fullPath)
			if rerr != nil {
				continue
			}
			var ls struct {
				PID int `json:"pid"`
			}
			dead := false
			if json.Unmarshal(raw, &ls) != nil {
				dead = now.Sub(st.ModTime()) > opts.MinTempAge
			} else if ls.PID > 0 {
				dead = !storageProcessAlive(ls.PID)
			}
			if !dead {
				continue
			}
			report.ReclaimableBytes += st.Size()
			cand := StorageCandidate{
				Kind:  "stale_web_auth",
				Path:  fullPath,
				Bytes: st.Size(),
			}
			if !opts.DryRun {
				if rmErr := os.Remove(fullPath); rmErr == nil {
					cand.Removed = true
					report.FreedBytes += st.Size()
					report.StaleWebAuthRemoved++
				} else {
					cand.Skipped = rmErr.Error()
				}
			}
			report.Candidates = append(report.Candidates, cand)
		}
	}

	sort.Slice(report.Profiles, func(i, j int) bool {
		return report.Profiles[i].TotalBytes > report.Profiles[j].TotalBytes
	})
	sort.Slice(report.Candidates, func(i, j int) bool {
		return report.Candidates[i].Bytes > report.Candidates[j].Bytes
	})
	return report, nil
}

func pruneAllowlistedDirChildren(
	parentDir string,
	allowlist map[string]struct{},
	provider, profileName string,
	dryRun bool,
	usage *ProfileStorageUsage,
	report *StorageReport,
) {
	parentInfo, err := os.Lstat(parentDir)
	if err != nil || !parentInfo.IsDir() || parentInfo.Mode()&os.ModeSymlink != 0 {
		return
	}
	entries, err := os.ReadDir(parentDir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		name := entry.Name()
		if _, allowed := allowlist[name]; !allowed {
			continue
		}
		target := filepath.Join(parentDir, name)
		st, lerr := os.Lstat(target)
		if lerr != nil || st.Mode()&os.ModeSymlink != 0 {
			continue
		}
		var bytes int64
		if st.IsDir() {
			bytes, _ = lstatDirSize(target)
		} else {
			bytes = st.Size()
		}
		usage.Reclaimable += bytes
		report.ReclaimableBytes += bytes
		cand := StorageCandidate{
			Kind:     "build_cache",
			Provider: provider,
			Profile:  profileName,
			Path:     target,
			Bytes:    bytes,
		}
		if !dryRun {
			if rmErr := os.RemoveAll(target); rmErr == nil {
				cand.Removed = true
				report.FreedBytes += bytes
			} else {
				cand.Skipped = rmErr.Error()
			}
		}
		report.Candidates = append(report.Candidates, cand)
	}
}

// lstatDirSize computes real physical file bytes below root WITHOUT following symlinks.
func lstatDirSize(root string) (int64, int) {
	var total int64
	var symlinks int
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		info, lerr := os.Lstat(path)
		if lerr != nil {
			return nil
		}
		if info.Mode()&os.ModeSymlink != 0 {
			symlinks++
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !info.IsDir() {
			total += info.Size()
		}
		return nil
	})
	return total, symlinks
}

func isGhostProfileReferenced(dataDir, provider, name string) bool {
	if cfg, err := config.LoadConfig(); err == nil {
		if cfg.Defaults[provider] == name {
			return true
		}
		for _, binding := range cfg.Bindings {
			if binding[provider] == name {
				return true
			}
		}
	}
	if raw, err := os.ReadFile(filepath.Join(dataDir, "runtimes.json")); err == nil && strings.Contains(string(raw), name) {
		return true
	}
	dbPath := filepath.Join(dataDir, "nexus.db")
	if _, err := os.Stat(dbPath); err == nil {
		db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=ro&_pragma=busy_timeout(2000)", filepath.ToSlash(dbPath)))
		if err == nil {
			defer db.Close()
			rows, qerr := db.Query(`SELECT name FROM sqlite_master WHERE type='table'`)
			if qerr == nil {
				var tables []string
				for rows.Next() {
					var tbl string
					if rows.Scan(&tbl) == nil {
						tables = append(tables, tbl)
					}
				}
				_ = rows.Close()
				for _, tbl := range tables {
					tRows, terr := db.Query(fmt.Sprintf(`SELECT * FROM "%s"`, strings.ReplaceAll(tbl, `"`, `""`)))
					if terr != nil {
						continue
					}
					cols, cerr := tRows.Columns()
					if cerr != nil {
						_ = tRows.Close()
						continue
					}
					vals := make([]any, len(cols))
					ptrs := make([]any, len(cols))
					for i := range vals {
						ptrs[i] = &vals[i]
					}
					for tRows.Next() {
						if tRows.Scan(ptrs...) == nil {
							for _, v := range vals {
								if s, ok := v.(string); ok && strings.Contains(s, name) {
									_ = tRows.Close()
									return true
								}
							}
						}
					}
					_ = tRows.Close()
				}
			}
		}
	}
	return false
}

func storageProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// On Unix os.FindProcess always succeeds; `/proc/<pid>` is authoritative on Linux,
	// and Signal(nil) checks liveness on Unix without mutating the process.
	if _, statErr := os.Stat(fmt.Sprintf("/proc/%d", pid)); statErr == nil {
		return true
	} else if !os.IsNotExist(statErr) {
		return proc.Signal(nil) == nil
	}
	return false
}
