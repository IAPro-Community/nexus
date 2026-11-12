package web

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

// handleProjectAgentGroups manages durable, project-scoped Agent groups.
// Collection: GET/POST /api/v1/projects/{project}/agent-groups
// Member/order update and delete: PUT/DELETE /api/v1/projects/{project}/agent-groups/{group}
func (h *NexusHandler) handleProjectAgentGroups(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/projects/"), "/"), "/")
	if len(parts) < 2 || parts[0] == "" || parts[1] != "agent-groups" || len(parts) > 3 {
		writeError(w, http.StatusNotFound, "invalid agent group path")
		return
	}
	projectID := parts[0]
	st, err := h.nexus.OpenProject()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	if _, err := st.GetProject(projectID); err != nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}
	if len(parts) == 2 {
		switch r.Method {
		case http.MethodGet:
			groups, err := st.ListAgentGroups(projectID)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, groups)
		case http.MethodPost:
			var body struct {
				Name     string `json:"name"`
				Task     string `json:"task"`
				Worktree string `json:"worktree"`
				Feature  string `json:"feature"`
			}
			if err := json.NewDecoder(io.LimitReader(r.Body, 16<<10)).Decode(&body); err != nil {
				writeError(w, http.StatusBadRequest, "invalid body")
				return
			}
			group, err := st.CreateAgentGroup(projectID, body.Name, body.Task, body.Worktree, body.Feature)
			if err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
			writeJSON(w, http.StatusCreated, group)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
		return
	}
	groupID := parts[2]
	switch r.Method {
	case http.MethodPut:
		var body struct {
			Name     string   `json:"name"`
			Task     string   `json:"task"`
			Worktree string   `json:"worktree"`
			Feature  string   `json:"feature"`
			AgentIDs []string `json:"agent_ids"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid body")
			return
		}
		group, err := st.UpdateAgentGroup(projectID, store.AgentGroup{
			ID: groupID, Name: body.Name, Task: body.Task, Worktree: body.Worktree,
			Feature: body.Feature, AgentIDs: body.AgentIDs,
		})
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, group)
	case http.MethodDelete:
		if err := st.DeleteAgentGroup(projectID, groupID); err != nil {
			writeError(w, http.StatusNotFound, "agent group not found")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
