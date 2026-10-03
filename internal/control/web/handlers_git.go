package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/IAPro-Community/nexus/internal/executil"
	coreflags "github.com/IAPro-Community/nexus/internal/core/flags"
	"github.com/IAPro-Community/nexus/internal/nexus"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

// ProjectGitBranchesResponse is returned by GET /api/v1/projects/:id/git/branches
type ProjectGitBranchesResponse struct {
	ProjectID      string   `json:"project_id"`
	CanonicalPath  string   `json:"canonical_path"`
	CurrentBranch  string   `json:"current_branch"`
	DefaultBranch  string   `json:"default_branch"`
	Branches       []string `json:"branches"`
	RemoteBranches []string `json:"remote_branches"`
	IsClean        bool     `json:"is_clean"`
	ModifiedCount  int      `json:"modified_count"`
}

// handleProjectGitBranches GET /api/v1/projects/{projectID}/git/branches
func (h *NexusHandler) handleProjectGitBranches(w http.ResponseWriter, r *http.Request) {
	st, err := h.nexus.OpenProject()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	projectID := projectIDFromPath(r.URL.Path)
	if projectID == "" {
		writeError(w, http.StatusNotFound, "missing project id")
		return
	}
	proj, err := st.GetProject(projectID)
	if err != nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}

	dir := proj.CanonicalPath
	current := getGitBranch(dir)
	if current == "" {
		current = proj.DefaultBranch
	}
	if current == "" {
		current = "main"
	}

	// 1. List local branches
	var branches []string
	cmdList := exec.Command("git", "branch", "--list", "--no-color")
	cmdList.Dir = dir
	executil.HideConsoleWindow(cmdList)
	if out, err := cmdList.Output(); err == nil {
		lines := strings.Split(string(out), "\n")
		for _, l := range lines {
			name := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(l), "*"))
			if name != "" && !strings.Contains(name, "->") {
				branches = append(branches, name)
			}
		}
	}
	if len(branches) == 0 {
		branches = []string{current}
	}

	// 2. List remote branches
	var remoteBranches []string
	cmdRemote := exec.Command("git", "branch", "-r", "--no-color")
	cmdRemote.Dir = dir
	executil.HideConsoleWindow(cmdRemote)
	if out, err := cmdRemote.Output(); err == nil {
		lines := strings.Split(string(out), "\n")
		for _, l := range lines {
			name := strings.TrimSpace(l)
			if name != "" && !strings.Contains(name, "->") {
				remoteBranches = append(remoteBranches, name)
			}
		}
	}

	// 3. Status check for uncommitted changes
	isClean := true
	modifiedCount := 0
	cmdStatus := exec.Command("git", "status", "--porcelain")
	cmdStatus.Dir = dir
	executil.HideConsoleWindow(cmdStatus)
	if out, err := cmdStatus.Output(); err == nil {
		trimmed := strings.TrimSpace(string(out))
		if trimmed != "" {
			lines := strings.Split(trimmed, "\n")
			modifiedCount = len(lines)
			isClean = false
		}
	}

	resp := ProjectGitBranchesResponse{
		ProjectID:      proj.ID,
		CanonicalPath:  proj.CanonicalPath,
		CurrentBranch:  current,
		DefaultBranch:  proj.DefaultBranch,
		Branches:       branches,
		RemoteBranches: remoteBranches,
		IsClean:        isClean,
		ModifiedCount:  modifiedCount,
	}
	writeJSON(w, http.StatusOK, resp)
}

// handleProjectGitCheckout POST /api/v1/projects/{projectID}/git/checkout
func (h *NexusHandler) handleProjectGitCheckout(w http.ResponseWriter, r *http.Request) {
	st, err := h.nexus.OpenProject()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	projectID := projectIDFromPath(r.URL.Path)
	if projectID == "" {
		writeError(w, http.StatusNotFound, "missing project id")
		return
	}
	proj, err := st.GetProject(projectID)
	if err != nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}

	var body struct {
		Branch string `json:"branch"`
		Create bool   `json:"create"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil || strings.TrimSpace(body.Branch) == "" {
		writeError(w, http.StatusBadRequest, "branch is required")
		return
	}
	targetBranch := strings.TrimSpace(body.Branch)
	// Basic sanitization against flags or illegal chars
	if strings.HasPrefix(targetBranch, "-") || strings.ContainsAny(targetBranch, " ~^:?*[\\") {
		writeError(w, http.StatusBadRequest, "invalid branch name")
		return
	}

	// Safety policy: Check if any agent is actively working in the canonical project tree (A10).
	agents, _ := st.ListAgents(projectID)
	for _, a := range agents {
		eff, _ := h.nexus.EffectiveAgentState(a.ID)
		if eff == store.AgentWorking || eff == store.AgentStarting || eff == store.AgentRecovering {
			isDirectCanonical := true
			if a.CurrentRevisionID != "" {
				if rev, rerr := st.GetRevision(a.CurrentRevisionID); rerr == nil {
					if cfg, perr := nexus.ParseAgentConfig(rev.Config); perr == nil {
						if cfg.Isolation == "worktree" {
							isDirectCanonical = false
						}
					}
				}
			}
			if isDirectCanonical {
				writeError(w, http.StatusConflict, fmt.Sprintf("cannot checkout branch: agent %q (%s) is actively running in the project workspace (stop agent or migrate to worktree isolation first)", a.Name, a.ID))
				return
			}
		}
	}

	dir := proj.CanonicalPath
	var cmd *exec.Cmd
	if body.Create {
		cmd = exec.Command("git", "checkout", "-b", targetBranch)
	} else {
		cmd = exec.Command("git", "checkout", targetBranch)
	}
	cmd.Dir = dir
	executil.HideConsoleWindow(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		sanitizedOut := strings.ReplaceAll(strings.TrimSpace(string(out)), "\n", " ")
		writeError(w, http.StatusBadRequest, fmt.Sprintf("git checkout failed: %s", sanitizedOut))
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success":        true,
		"current_branch": targetBranch,
		"output":         strings.TrimSpace(string(out)),
	})
}


const maxReviewDiffBytes = 1024 * 1024

type ProjectGitChange struct {
	Path     string `json:"path"`
	OldPath  string `json:"old_path,omitempty"`
	Index    string `json:"index_status"`
	Worktree string `json:"worktree_status"`
}

type ProjectGitChangesResponse struct {
	ProjectID     string             `json:"project_id"`
	CurrentBranch string             `json:"current_branch"`
	Head          string             `json:"head"`
	Changes       []ProjectGitChange `json:"changes"`
	Truncated     bool               `json:"truncated"`
}

type ProjectGitDiffResponse struct {
	ProjectID string `json:"project_id"`
	Path      string `json:"path"`
	Staged    bool   `json:"staged"`
	Diff      string `json:"diff"`
	Truncated bool   `json:"truncated"`
}

func (h *NexusHandler) handleProjectGitChanges(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !coreflags.Enabled(coreflags.ReviewCenter) {
		writeError(w, http.StatusNotFound, "review center is disabled")
		return
	}
	st, err := h.nexus.OpenProject()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	projectID := projectIDFromPath(r.URL.Path)
	proj, err := st.GetProject(projectID)
	if err != nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}

	status, truncated, err := runBoundedGit(proj.CanonicalPath, 256*1024, "status", "--porcelain=v1", "-z", "--untracked-files=normal")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	changes := parsePorcelainChanges(status)
	head, _, err := runBoundedGit(proj.CanonicalPath, 4096, "rev-parse", "HEAD")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, ProjectGitChangesResponse{
		ProjectID:     proj.ID,
		CurrentBranch: getGitBranch(proj.CanonicalPath),
		Head:          strings.TrimSpace(head),
		Changes:       changes,
		Truncated:     truncated,
	})
}

func (h *NexusHandler) handleProjectGitDiff(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !coreflags.Enabled(coreflags.ReviewCenter) {
		writeError(w, http.StatusNotFound, "review center is disabled")
		return
	}
	st, err := h.nexus.OpenProject()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	projectID := projectIDFromPath(r.URL.Path)
	proj, err := st.GetProject(projectID)
	if err != nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}
	rel, err := validateReviewPath(r.URL.Query().Get("path"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	staged := r.URL.Query().Get("staged") == "1" || strings.EqualFold(r.URL.Query().Get("staged"), "true")
	args := []string{"diff", "--no-ext-diff", "--no-color", "--unified=3"}
	if staged {
		args = append(args, "--cached")
	}
	args = append(args, "--", rel)
	diff, truncated, err := runBoundedGit(proj.CanonicalPath, maxReviewDiffBytes, args...)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, ProjectGitDiffResponse{
		ProjectID: proj.ID,
		Path:      rel,
		Staged:    staged,
		Diff:      diff,
		Truncated: truncated,
	})
}

func validateReviewPath(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("path is required")
	}
	if strings.ContainsRune(raw, '\x00') || filepath.IsAbs(raw) {
		return "", fmt.Errorf("invalid review path")
	}
	clean := filepath.Clean(raw)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("review path escapes project root")
	}
	return filepath.ToSlash(clean), nil
}

func parsePorcelainChanges(raw string) []ProjectGitChange {
	entries := strings.Split(raw, "\x00")
	out := make([]ProjectGitChange, 0, len(entries))
	for i := 0; i < len(entries); i++ {
		entry := entries[i]
		if len(entry) < 4 {
			continue
		}
		indexStatus := entry[0]
		worktreeStatus := entry[1]
		path := entry[3:]
		if path == "" {
			continue
		}
		change := ProjectGitChange{
			Path:     path,
			Index:    string(indexStatus),
			Worktree: string(worktreeStatus),
		}
		if indexStatus == 'R' || indexStatus == 'C' || worktreeStatus == 'R' || worktreeStatus == 'C' {
			// With porcelain -z Git emits destination first, followed by origin,
			// both NUL terminated and without the human-readable " -> " marker.
			if i+1 < len(entries) {
				i++
				change.OldPath = entries[i]
			}
		}
		out = append(out, change)
	}
	return out
}

type boundedWriter struct {
	buf       bytes.Buffer
	remaining int
	truncated bool
}

func (w *boundedWriter) Write(p []byte) (int, error) {
	original := len(p)
	if w.remaining > 0 {
		keep := len(p)
		if keep > w.remaining {
			keep = w.remaining
		}
		_, _ = w.buf.Write(p[:keep])
		w.remaining -= keep
		if keep < len(p) {
			w.truncated = true
		}
	} else if len(p) > 0 {
		w.truncated = true
	}
	return original, nil
}

func runBoundedGit(dir string, limit int, args ...string) (string, bool, error) {
	if limit <= 0 {
		return "", false, fmt.Errorf("invalid git output limit")
	}
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	executil.HideConsoleWindow(cmd)
	stdout := &boundedWriter{remaining: limit}
	stderr := &boundedWriter{remaining: 16 * 1024}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		message := strings.TrimSpace(stderr.buf.String())
		if message == "" {
			message = err.Error()
		}
		return "", stdout.truncated, fmt.Errorf("git %s failed: %s", strings.Join(args, " "), message)
	}
	return stdout.buf.String(), stdout.truncated, nil
}
