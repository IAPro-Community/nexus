package web

import (
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/browser"
)

// FSEntry represents a directory entry on the host filesystem.
type FSEntry struct {
	Name        string   `json:"name"`
	Path        string   `json:"path"`
	IsDir       bool     `json:"is_dir"`
	IsGit       bool     `json:"is_git"`
	Tech        []string `json:"tech"`
	ModTime     string   `json:"mod_time"`
	SizeBytes   int64    `json:"size_bytes"`
	ChildCount  int      `json:"child_count,omitempty"`
	Permissions string   `json:"permissions"`
}

// FSBookmark represents a quick-access folder in the OS.
type FSBookmark struct {
	Label string `json:"label"`
	Path  string `json:"path"`
	Icon  string `json:"icon"` // "home", "folder", "desktop", "documents", "root"
}

// FSBrowseResponse is returned by GET /api/v1/fs/browse
type FSBrowseResponse struct {
	CurrentPath string       `json:"current_path"`
	ParentPath  string       `json:"parent_path"`
	Breadcrumbs []string     `json:"breadcrumbs"`
	Entries     []FSEntry    `json:"entries"`
	Bookmarks   []FSBookmark `json:"bookmarks"`
	IsGit       bool         `json:"is_git"`
	GitBranch   string       `json:"git_branch,omitempty"`
	Tech        []string     `json:"tech,omitempty"`
}

// FSScanResult represents a discovered Git repository on the host OS.
type FSScanResult struct {
	Name       string   `json:"name"`
	Path       string   `json:"path"`
	Branch     string   `json:"branch"`
	Tech       []string `json:"tech"`
	ModTime    string   `json:"mod_time"`
	IsImported bool     `json:"is_imported"`
}

// FSInspectResponse is returned by GET /api/v1/fs/inspect
type FSInspectResponse struct {
	Path          string   `json:"path"`
	Exists        bool     `json:"exists"`
	IsDir         bool     `json:"is_dir"`
	IsGit         bool     `json:"is_git"`
	GitBranch     string   `json:"git_branch,omitempty"`
	GitRemote     string   `json:"git_remote,omitempty"`
	SuggestedName string   `json:"suggested_name"`
	Tech          []string `json:"tech"`
}

func expandHomePath(path string) string {
	if path == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			return home
		}
		return path
	}
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, strings.TrimPrefix(path, "~/"))
		}
	}
	return path
}

// detectTech inspects the files in a directory and returns detected technologies/frameworks.
func detectTech(dir string) []string {
	tech := make([]string, 0)
	hasFile := func(name string) bool {
		_, err := os.Stat(filepath.Join(dir, name))
		return err == nil
	}

	if hasFile("package.json") {
		tech = append(tech, "Node.js")
	}
	if hasFile("tsconfig.json") {
		tech = append(tech, "TypeScript")
	}
	if hasFile("go.mod") {
		tech = append(tech, "Go")
	}
	if hasFile("Cargo.toml") {
		tech = append(tech, "Rust")
	}
	if hasFile("pyproject.toml") || hasFile("requirements.txt") || hasFile("Pipfile") {
		tech = append(tech, "Python")
	}
	if hasFile("Dockerfile") || hasFile("docker-compose.yml") {
		tech = append(tech, "Docker")
	}
	if hasFile("pom.xml") || hasFile("build.gradle") {
		tech = append(tech, "Java")
	}
	if hasFile("Makefile") {
		tech = append(tech, "Make")
	}
	return tech
}

// getGitBranch returns the current git branch of a directory if available.
func getGitBranch(dir string) string {
	headPath := filepath.Join(dir, ".git", "HEAD")
	data, err := os.ReadFile(headPath)
	if err != nil {
		return ""
	}
	content := strings.TrimSpace(string(data))
	if strings.HasPrefix(content, "ref: refs/heads/") {
		return strings.TrimPrefix(content, "ref: refs/heads/")
	}
	if len(content) > 7 {
		return content[:7]
	}
	return content
}

// getGitRemote returns the origin remote URL if available.
func getGitRemote(dir string) string {
	configPath := filepath.Join(dir, ".git", "config")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return ""
	}
	lines := strings.Split(string(data), "\n")
	inOrigin := false
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if strings.HasPrefix(trimmed, "[remote \"origin\"]") {
			inOrigin = true
			continue
		}
		if inOrigin {
			if strings.HasPrefix(trimmed, "[") {
				break
			}
			if strings.HasPrefix(trimmed, "url = ") {
				return redactGitRemote(strings.TrimPrefix(trimmed, "url = "))
			}
		}
	}
	return ""
}

func redactGitRemote(remote string) string {
	parsed, err := url.Parse(remote)
	if err != nil || parsed.User == nil {
		return remote
	}
	parsed.User = nil
	return parsed.String()
}

// getOSBookmarks returns convenient OS bookmark paths.
func getOSBookmarks() []FSBookmark {
	bookmarks := make([]FSBookmark, 0)

	homeDir, _ := os.UserHomeDir()
	if homeDir != "" {
		bookmarks = append(bookmarks, FSBookmark{Label: "Home (~)", Path: homeDir, Icon: "home"})

		desktop := filepath.Join(homeDir, "Desktop")
		if _, err := os.Stat(desktop); err == nil {
			bookmarks = append(bookmarks, FSBookmark{Label: "Desktop", Path: desktop, Icon: "desktop"})
		}

		docs := filepath.Join(homeDir, "Documents")
		if _, err := os.Stat(docs); err == nil {
			bookmarks = append(bookmarks, FSBookmark{Label: "Documents", Path: docs, Icon: "documents"})
		}

		userProjects := filepath.Join(homeDir, "projetos")
		if _, err := os.Stat(userProjects); err == nil {
			bookmarks = append(bookmarks, FSBookmark{Label: "~/projetos", Path: userProjects, Icon: "folder"})
		}

		workspace := filepath.Join(homeDir, "workspace")
		if _, err := os.Stat(workspace); err == nil {
			bookmarks = append(bookmarks, FSBookmark{Label: "~/workspace", Path: workspace, Icon: "folder"})
		}
	}

	// Root /projetos if exists on host
	if _, err := os.Stat("/projetos"); err == nil {
		bookmarks = append(bookmarks, FSBookmark{Label: "/projetos", Path: "/projetos", Icon: "folder"})
	}

	// Root filesystem
	bookmarks = append(bookmarks, FSBookmark{Label: "Root (/)", Path: "/", Icon: "root"})

	// The picker can only browse allowed roots, so never surface a bookmark
	// the filesystem policy would reject on click. On Windows "Root (/)" would
	// resolve to the drive root, which is outside allowed roots.
	allowed := make([]FSBookmark, 0, len(bookmarks))
	for _, bookmark := range bookmarks {
		if isWithinAllowedRoots(bookmark.Path) {
			allowed = append(allowed, bookmark)
		}
	}

	return allowed
}

// handleFSBrowse GET /api/v1/fs/browse?path=...
func (h *NexusHandler) handleFSBrowse(w http.ResponseWriter, r *http.Request) {
	if !h.hostFilesystemEnabled {
		writeError(w, http.StatusForbidden, "host filesystem operations are available only on loopback")
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	targetPath := r.URL.Query().Get("path")
	if strings.TrimSpace(targetPath) == "" {
		targetPath = defaultBrowsePath()
	}

	// Clean and normalize target path
	if strings.HasPrefix(targetPath, "~") {
		targetPath = expandHomePath(targetPath)
	}

	absPath, err := filepath.Abs(filepath.Clean(targetPath))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if !isWithinAllowedRoots(absPath) {
		writeError(w, http.StatusForbidden, "path is outside allowed directories")
		return
	}

	info, err := os.Stat(absPath)
	if err != nil {
		writeError(w, http.StatusNotFound, "path not found: "+err.Error())
		return
	}

	if !info.IsDir() {
		absPath = filepath.Dir(absPath)
	}

	if !isWithinAllowedRoots(absPath) {
		writeError(w, http.StatusForbidden, "path is outside allowed directories")
		return
	}

	entries, err := os.ReadDir(absPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "cannot read directory: "+err.Error())
		return
	}

	var results []FSEntry
	includeEntryMetadata := r.URL.Query().Get("details") == "full"
	for _, entry := range entries {
		// Skip hidden files/dirs unless needed (skip .git internal directory itself, but keep flag)
		if strings.HasPrefix(entry.Name(), ".") && entry.Name() != ".git" {
			continue
		}

		fullPath := filepath.Join(absPath, entry.Name())
		isDir := entry.IsDir()
		isGit := false
		var tech []string
		var entryInfo os.FileInfo

		if includeEntryMetadata {
			entryInfo, err = entry.Info()
			if err != nil {
				continue
			}
		}

		if isDir && includeEntryMetadata {
			// Check if child is a Git repo
			if _, gErr := os.Stat(filepath.Join(fullPath, ".git")); gErr == nil {
				isGit = true
			}
			tech = detectTech(fullPath)
		}

		results = append(results, FSEntry{
			Name:  entry.Name(),
			Path:  fullPath,
			IsDir: isDir,
			IsGit: isGit,
			Tech:  tech,
		})
		if entryInfo != nil {
			results[len(results)-1].ModTime = entryInfo.ModTime().Format(time.RFC3339)
			results[len(results)-1].SizeBytes = entryInfo.Size()
			results[len(results)-1].Permissions = entryInfo.Mode().String()
		}
	}

	// Sort: Directories first, then alphabetical
	sort.Slice(results, func(i, j int) bool {
		if results[i].IsDir != results[j].IsDir {
			return results[i].IsDir // directories first
		}
		return strings.ToLower(results[i].Name) < strings.ToLower(results[j].Name)
	})

	// Check if current directory itself is a Git repository
	currentIsGit := false
	currentBranch := ""
	if _, gErr := os.Stat(filepath.Join(absPath, ".git")); gErr == nil {
		currentIsGit = true
		currentBranch = getGitBranch(absPath)
	}

	parent := filepath.Dir(absPath)
	if parent == absPath {
		parent = ""
	}

	// Build breadcrumbs
	cleanParts := strings.Split(filepath.ToSlash(absPath), "/")
	var breadcrumbs []string
	curr := ""
	for _, p := range cleanParts {
		if p == "" {
			if curr == "" {
				curr = "/"
				breadcrumbs = append(breadcrumbs, "/")
			}
			continue
		}
		curr = filepath.Join(curr, p)
		breadcrumbs = append(breadcrumbs, curr)
	}

	resp := FSBrowseResponse{
		CurrentPath: absPath,
		ParentPath:  parent,
		Breadcrumbs: breadcrumbs,
		Entries:     results,
		Bookmarks:   getOSBookmarks(),
		IsGit:       currentIsGit,
		GitBranch:   currentBranch,
		Tech:        detectTech(absPath),
	}

	writeJSON(w, http.StatusOK, resp)
}

// handleFSInspect GET /api/v1/fs/inspect?path=...
func (h *NexusHandler) handleFSInspect(w http.ResponseWriter, r *http.Request) {
	if !h.hostFilesystemEnabled {
		writeError(w, http.StatusForbidden, "host filesystem operations are available only on loopback")
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	targetPath := r.URL.Query().Get("path")
	if strings.TrimSpace(targetPath) == "" {
		writeError(w, http.StatusBadRequest, "path is required")
		return
	}

	if strings.HasPrefix(targetPath, "~") {
		targetPath = expandHomePath(targetPath)
	}

	absPath, err := filepath.Abs(filepath.Clean(targetPath))
	if err != nil {
		writeJSON(w, http.StatusOK, FSInspectResponse{Path: targetPath, Exists: false})
		return
	}

	info, err := os.Stat(absPath)
	if err != nil {
		writeJSON(w, http.StatusOK, FSInspectResponse{Path: absPath, Exists: false})
		return
	}

	isGit := false
	branch := ""
	remote := ""
	if info.IsDir() {
		if _, gErr := os.Stat(filepath.Join(absPath, ".git")); gErr == nil {
			isGit = true
			branch = getGitBranch(absPath)
			remote = getGitRemote(absPath)
		}
	}

	name := filepath.Base(absPath)
	if name == "." || name == "/" || name == "" {
		name = "Project"
	}

	resp := FSInspectResponse{
		Path:          absPath,
		Exists:        true,
		IsDir:         info.IsDir(),
		IsGit:         isGit,
		GitBranch:     branch,
		GitRemote:     remote,
		SuggestedName: name,
		Tech:          detectTech(absPath),
	}

	writeJSON(w, http.StatusOK, resp)
}

// handleFSScan GET /api/v1/fs/scan?root=...
func (h *NexusHandler) handleFSScan(w http.ResponseWriter, r *http.Request) {
	if !h.hostFilesystemEnabled {
		writeError(w, http.StatusForbidden, "host filesystem operations are available only on loopback")
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	st, err := h.nexus.OpenProject()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	existingProjects, _ := st.ListProjects()
	importedMap := make(map[string]bool)
	for _, p := range existingProjects {
		importedMap[filepath.Clean(p.CanonicalPath)] = true
	}

	var scanRoots []string
	customRoot := r.URL.Query().Get("root")
	if customRoot != "" {
		if strings.HasPrefix(customRoot, "~") {
			customRoot = expandHomePath(customRoot)
		}
		if abs, err := filepath.Abs(filepath.Clean(customRoot)); err == nil {
			scanRoots = append(scanRoots, abs)
		}
	} else {
		home, _ := os.UserHomeDir()
		if home != "" {
			scanRoots = append(scanRoots, home)
			scanRoots = append(scanRoots, filepath.Join(home, "projetos"))
			scanRoots = append(scanRoots, filepath.Join(home, "workspace"))
			scanRoots = append(scanRoots, filepath.Join(home, "dev"))
			scanRoots = append(scanRoots, filepath.Join(home, "src"))
			scanRoots = append(scanRoots, filepath.Join(home, "Desktop"))
		}
		if _, err := os.Stat("/projetos"); err == nil {
			scanRoots = append(scanRoots, "/projetos")
		}
	}

	var discovered []FSScanResult
	seen := make(map[string]bool)

	// Scan with max depth of 3
	for _, root := range scanRoots {
		if _, err := os.Stat(root); err != nil {
			continue
		}

		_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if !d.IsDir() {
				return nil
			}

			// Don't descend into node_modules, .git, vendor, cache, tmp
			name := d.Name()
			if name == ".git" {
				// Found a git repo!
				repoDir := filepath.Dir(path)
				if !seen[repoDir] {
					seen[repoDir] = true
					info, _ := d.Info()
					modTime := ""
					if info != nil {
						modTime = info.ModTime().Format(time.RFC3339)
					}
					discovered = append(discovered, FSScanResult{
						Name:       filepath.Base(repoDir),
						Path:       repoDir,
						Branch:     getGitBranch(repoDir),
						Tech:       detectTech(repoDir),
						ModTime:    modTime,
						IsImported: importedMap[filepath.Clean(repoDir)],
					})
				}
				return filepath.SkipDir
			}

			if strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" || name == "dist" || name == "build" || name == "target" {
				return filepath.SkipDir
			}

			// Limit depth from root to at most 3
			rel, _ := filepath.Rel(root, path)
			if strings.Count(rel, string(filepath.Separator)) >= 3 {
				return filepath.SkipDir
			}

			return nil
		})
	}

	// Sort: non-imported first, then alphabetical
	sort.Slice(discovered, func(i, j int) bool {
		if discovered[i].IsImported != discovered[j].IsImported {
			return !discovered[i].IsImported
		}
		return strings.ToLower(discovered[i].Name) < strings.ToLower(discovered[j].Name)
	})

	writeJSON(w, http.StatusOK, discovered)
}

// defaultBrowsePath picks the folder picker's starting directory when the
// request does not provide one. The process working directory is only used
// when it already sits inside an allowed root: the web workspace is often
// started from installation or system directories that the filesystem
// policy rejects, and defaulting there would otherwise turn the whole
// picker into a 403. Fall back to the user home directory.
func defaultBrowsePath() string {
	if wd, err := os.Getwd(); err == nil && isWithinAllowedRoots(wd) {
		return wd
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return home
	}
	if wd, err := os.Getwd(); err == nil {
		return wd
	}
	return "/"
}

// isWithinAllowedRoots checks whether path falls inside a safe directory.
// Allowed roots are: user home dir, the OS temporary directory, /tmp on
// non-Windows hosts for compatibility, and any ancestor containing .git or
// AGENTS.md (project workspace markers).
func isWithinAllowedRoots(path string) bool {
	abs, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return false
	}
	// Resolve the nearest existing ancestor before checking policy. A lexical
	// /tmp or home prefix is insufficient when an attacker places a symlink
	// inside that prefix which points at a protected directory.
	realPath, err := resolveExistingAncestor(abs)
	if err != nil {
		return false
	}
	abs = realPath

	if home, err := os.UserHomeDir(); err == nil && pathWithin(home, abs) {
		return true
	}

	if tempDir := os.TempDir(); pathWithin(tempDir, abs) {
		return true
	}
	// On Unix, /tmp may differ from os.TempDir() (e.g. macOS /var/folders).
	// Never hard-require a Unix-only path on Windows.
	if runtime.GOOS != "windows" && pathWithin("/tmp", abs) {
		return true
	}

	// Walk upward looking for project workspace markers
	dir := abs
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return true
		}
		if _, err := os.Stat(filepath.Join(dir, "AGENTS.md")); err == nil {
			return true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return false
}

func pathWithin(root, path string) bool {
	cleanRoot := filepath.Clean(root)
	if resolvedRoot, err := resolveExistingAncestor(cleanRoot); err == nil {
		cleanRoot = resolvedRoot
	}
	rel, err := filepath.Rel(cleanRoot, filepath.Clean(path))
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func resolveExistingAncestor(path string) (string, error) {
	candidate := filepath.Clean(path)
	for {
		if _, err := os.Lstat(candidate); err == nil {
			return filepath.EvalSymlinks(candidate)
		} else if !os.IsNotExist(err) {
			return "", err
		}
		parent := filepath.Dir(candidate)
		if parent == candidate {
			return "", os.ErrNotExist
		}
		candidate = parent
	}
}

// handleFSMkdir POST /api/v1/fs/mkdir
func (h *NexusHandler) handleFSMkdir(w http.ResponseWriter, r *http.Request) {
	if !h.hostFilesystemEnabled {
		writeError(w, http.StatusForbidden, "host filesystem operations are available only on loopback")
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var body struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Path) == "" {
		writeError(w, http.StatusBadRequest, "path is required")
		return
	}

	targetPath := body.Path
	if strings.HasPrefix(targetPath, "~") {
		targetPath = expandHomePath(targetPath)
	}

	absPath, err := filepath.Abs(filepath.Clean(targetPath))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if !isWithinAllowedRoots(absPath) {
		writeError(w, http.StatusForbidden, "path is outside allowed directories")
		return
	}

	if err := os.MkdirAll(absPath, 0755); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create directory: "+err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{"path": absPath, "status": "created"})
}

// handleProjectOpenOS POST /api/v1/projects/{id}/open-os
func (h *NexusHandler) handleProjectOpenOS(w http.ResponseWriter, r *http.Request) {
	if !h.hostFilesystemEnabled {
		writeError(w, http.StatusForbidden, "host filesystem operations are available only on loopback")
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	st, err := h.nexus.OpenProject()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/projects/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		writeError(w, http.StatusNotFound, "missing project id")
		return
	}
	id := parts[0]

	proj, err := st.GetProject(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "project not found: "+err.Error())
		return
	}

	var body struct {
		Action string `json:"action"` // "filemanager", "terminal", "editor"
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}

	path := proj.CanonicalPath
	if path == "" {
		writeError(w, http.StatusBadRequest, "project has no canonical path")
		return
	}

	var cmd *exec.Cmd

	switch body.Action {
	case "filemanager":
		switch runtime.GOOS {
		case "darwin":
			cmd = exec.Command("open", path)
		case "windows":
			cmd = exec.Command("explorer", path)
		default: // linux/unix
			// Resolve real system xdg-open to avoid symlink recursion
			// when running as a browser-helper shim.
			if xdg := browser.ResolveXdgOpen(); xdg != "" {
				cmd = exec.Command(xdg, path)
			} else {
				cmd = exec.Command("xdg-open", path)
			}
		}

	case "terminal":
		term := os.Getenv("TERMINAL")
		if term != "" {
			// Allowlist of known-safe terminal binaries to prevent arbitrary
			// command execution if an attacker influences the process environment.
			allowedTerminals := map[string]bool{
				"x-terminal-emulator": true, "gnome-terminal": true, "konsole": true,
				"xfce4-terminal": true, "alacritty": true, "kitty": true, "xterm": true,
				"iterm2": true, "wezterm": true, "foot": true, "st": true,
				"Terminal.app": true, "wt.exe": true, "cmd.exe": true,
			}
			base := filepath.Base(term)
			if allowedTerminals[base] {
				cmd = exec.Command(term)
			}
		} else if runtime.GOOS == "darwin" {
			cmd = exec.Command("open", "-a", "Terminal", path)
		} else if runtime.GOOS == "windows" {
			cmd = exec.Command("cmd.exe", "/c", "start", "cmd.exe")
		} else {
			// Try standard Linux terminals
			candidates := []string{"x-terminal-emulator", "gnome-terminal", "konsole", "xfce4-terminal", "alacritty", "kitty", "xterm"}
			for _, candidate := range candidates {
				if lp, err := exec.LookPath(candidate); err == nil {
					cmd = exec.Command(lp)
					break
				}
			}
			if cmd == nil {
				cmd = exec.Command("xterm")
			}
		}
		if cmd != nil {
			cmd.Dir = path
		}

	case "editor":
		// Prefer cursor, then code, then $EDITOR
		if lp, err := exec.LookPath("cursor"); err == nil {
			cmd = exec.Command(lp, path)
		} else if lp, err := exec.LookPath("code"); err == nil {
			cmd = exec.Command(lp, path)
		} else if ed := os.Getenv("EDITOR"); ed != "" {
			cmd = exec.Command(ed, path)
		} else {
			// Resolve real system xdg-open to avoid symlink recursion.
			if xdg := browser.ResolveXdgOpen(); xdg != "" {
				cmd = exec.Command(xdg, path)
			} else {
				cmd = exec.Command("xdg-open", path)
			}
		}

	default:
		writeError(w, http.StatusBadRequest, "unknown action: "+body.Action)
		return
	}

	if cmd != nil {
		_ = cmd.Start() // Non-blocking background launch
	}

	writeJSON(w, http.StatusOK, map[string]any{"status": "launched", "action": body.Action, "path": path})
}

// CurrentUserName returns the OS username.
func CurrentUserName() string {
	if u, err := user.Current(); err == nil {
		return u.Username
	}
	return "developer"
}
