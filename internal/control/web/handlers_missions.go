package web

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/kivervinicius/ai-cli/internal/nexus"
	"github.com/kivervinicius/ai-cli/internal/nexus/store"
)

// handleMissionsList GET /api/v1/projects/{id}/missions — list missions for a project.
func (h *NexusHandler) handleMissionsList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	projectID := projectIDFromPath(r.URL.Path)
	if projectID == "" {
		writeError(w, http.StatusNotFound, "missing project id")
		return
	}
	missions, err := h.missions.List(r.Context(), projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"missions": missions})
}

// handleMissionCreate POST /api/v1/projects/{id}/missions — create a mission.
func (h *NexusHandler) handleMissionCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	projectID := projectIDFromPath(r.URL.Path)
	if projectID == "" {
		writeError(w, http.StatusNotFound, "missing project id")
		return
	}
	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Goal        string `json:"goal"`
		Scope       string `json:"scope"`
		RiskLevel   string `json:"risk_level"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	m := &store.Mission{
		ProjectID:   projectID,
		Name:        body.Name,
		Description: body.Description,
		Goal:        body.Goal,
		Scope:       body.Scope,
		RiskLevel:   body.RiskLevel,
	}
	if _, err := h.missions.Create(r.Context(), m); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, m)
}

// handleMissionDetail GET/PATCH/DELETE /api/v1/missions/{id}
func (h *NexusHandler) handleMissionDetail(w http.ResponseWriter, r *http.Request) {
	missionID := missionIDFromPath(r.URL.Path)
	if missionID == "" {
		writeError(w, http.StatusNotFound, "missing mission id")
		return
	}

	switch r.Method {
	case http.MethodGet:
		detail, err := h.missions.Detail(r.Context(), missionID)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"mission":     detail.Mission,
			"tasks":       detail.Tasks,
			"assignments": detail.Assignments,
			"stats":       detail.Stats,
		})

	case http.MethodPatch:
		var body struct {
			Name        *string `json:"name"`
			Description *string `json:"description"`
			Status      *string `json:"status"`
			Goal        *string `json:"goal"`
			Scope       *string `json:"scope"`
			RiskLevel   *string `json:"risk_level"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid body")
			return
		}
		m, err := h.missions.Update(r.Context(), missionID, nexus.MissionPatch{
			Name: body.Name, Description: body.Description, Status: body.Status,
			Goal: body.Goal, Scope: body.Scope, RiskLevel: body.RiskLevel,
		})
		if err != nil {
			if errors.Is(err, nexus.ErrTerminalMissionStatusPatch) {
				writeError(w, http.StatusConflict, err.Error())
				return
			}
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, m)

	case http.MethodDelete:
		if err := h.missions.Delete(r.Context(), missionID); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleMissionTaskCreate POST /api/v1/missions/{id}/tasks — add a task to a mission.
func (h *NexusHandler) handleMissionTaskCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	missionID := missionIDFromPath(r.URL.Path)
	if missionID == "" {
		writeError(w, http.StatusNotFound, "missing mission id")
		return
	}
	var body struct {
		Name         string `json:"name"`
		Description  string `json:"description"`
		Kind         string `json:"kind"`
		Priority     int    `json:"priority"`
		Dependencies string `json:"dependencies"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	t := &store.MissionTask{
		MissionID:    missionID,
		Name:         body.Name,
		Description:  body.Description,
		Kind:         body.Kind,
		Priority:     body.Priority,
		Dependencies: body.Dependencies,
	}
	if err := h.missions.CreateTask(r.Context(), t); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

// handleMissionAssign POST /api/v1/missions/{id}/assign — assign an agent to a task.
func (h *NexusHandler) handleMissionAssign(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	missionID := missionIDFromPath(r.URL.Path)
	if missionID == "" {
		writeError(w, http.StatusNotFound, "missing mission id")
		return
	}
	var body struct {
		TaskID  string `json:"task_id"`
		AgentID string `json:"agent_id"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	a := &store.MissionAssignment{
		MissionID: missionID,
		TaskID:    body.TaskID,
		AgentID:   body.AgentID,
	}
	if err := h.missions.Assign(r.Context(), a); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, a)
}

func projectIDFromPath(path string) string {
	parts := strings.Split(strings.TrimPrefix(path, "/api/v1/projects/"), "/")
	if len(parts) >= 1 && strings.HasSuffix(path, "/agents") {
		return parts[0]
	}
	if len(parts) >= 1 {
		return parts[0]
	}
	return ""
}

func agentIDFromPath(path string) string {
	parts := strings.Split(strings.TrimPrefix(path, "/api/v1/agents/"), "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}

func missionIDFromPath(path string) string {
	// Handle /api/v1/missions/{id} and /api/v1/missions/{id}/tasks and /api/v1/missions/{id}/assign
	parts := strings.Split(strings.TrimPrefix(path, "/api/v1/missions/"), "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}

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
