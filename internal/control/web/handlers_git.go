package web

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"

	"github.com/IAPro-Community/nexus/internal/executil"
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
	Worktrees       []ProjectGitWorktree `json:"worktrees"`
	WorktreesAvailable bool `json:"worktrees_available"`
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

	worktrees, worktreesErr := listProjectGitWorktrees(r.Context(), dir)
	if worktrees == nil {
		worktrees = []ProjectGitWorktree{}
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
		Worktrees:      worktrees,
		WorktreesAvailable: worktreesErr == nil,
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
