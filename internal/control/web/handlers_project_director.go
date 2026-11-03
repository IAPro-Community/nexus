package web

import (
    "net/http"

    "github.com/IAPro-Community/nexus/internal/nexus/store"
)

type projectDirectorView struct {
    SchemaVersion int `json:"schema_version"`
    ExecutionOwner string `json:"execution_owner"`
    ProjectID string `json:"project_id"`
    Total int `json:"total"`
    Truncated bool `json:"truncated"`
    Agents []directorProjection `json:"agents"`
}

// handleProjectDirector is a bounded, read-only projection of the EXISTING
// persistent Agents. It does not discover arbitrary host processes, spawn a
// Mission, or query provider homes for native transcripts or credentials.
func (h *NexusHandler) handleProjectDirector(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        writeError(w, http.StatusMethodNotAllowed, "director mutations are unsupported")
        return
    }
    id := projectIDFromPath(r.URL.Path)
    if id == "" {
        writeError(w, http.StatusBadRequest, "project ID required")
        return
    }
    st, err := h.nexus.OpenProject()
    if err != nil {
        writeError(w, http.StatusServiceUnavailable, err.Error())
        return
    }
    if _, err := st.GetProject(id); err != nil {
        writeError(w, http.StatusNotFound, "project not found")
        return
    }
    agents, err := st.ListAgents(id)
    if err != nil {
        writeError(w, http.StatusServiceUnavailable, err.Error())
        return
    }
    constMax := 100
    total := len(agents)
    if len(agents) > constMax { agents = agents[:constMax] }
    views := make([]directorProjection, 0, len(agents))
    for _, agent := range agents {
        view, err := h.directorViewForStoredAgent(st, agent)
        if err != nil {
            writeError(w, http.StatusServiceUnavailable, err.Error())
            return
        }
        views = append(views, view)
    }
    // Never cache volatile process liveness or status derived from a registry.
    w.Header().Set("Cache-Control", "no-store")
    writeJSON(w, http.StatusOK, projectDirectorView{
        SchemaVersion: 1, ExecutionOwner: "NEXUS", ProjectID: id,
        Total: total, Truncated: total > constMax,
        Agents: views,
    })
}

var _ = store.Agent{}
