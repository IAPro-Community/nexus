package web

import (
	"net/http"
	"strings"

	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

// handleNexConversation exposes operational conversation history to clients
// of the same authenticated local Nexus owner. No Maestro dependency and no
// export of provider credentials. All mutations require CSRF via authMiddleware.
func (h *NexusHandler) handleNexConversation(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.nexus == nil {
		writeError(w, http.StatusServiceUnavailable, "Nex unavailable")
		return
	}
	projectID := strings.TrimSpace(r.URL.Query().Get("project_id"))
	if len(projectID) > 128 {
		writeError(w, http.StatusBadRequest, "invalid project id")
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodDelete {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	st, err := h.nexus.OpenProject()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "Nex conversation unavailable")
		return
	}
	if projectID != "" {
		if _, err := st.GetProject(projectID); err != nil {
			writeError(w, http.StatusNotFound, "project not found")
			return
		}
	}
	switch r.Method {
	case http.MethodGet:
		turns, err := st.ListNexConversation(projectID)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "Nex conversation unavailable")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"turns": turns, "persisted": true})
	case http.MethodDelete:
		if err := st.DeleteNexConversation(projectID); err != nil {
			writeError(w, http.StatusServiceUnavailable, "could not remove Nex conversation")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	}
}

// Keep the schema contract explicit at compilation time. The Nex assistant
// never needs direct access to Composer's semantic planning turn schema.
var _ = store.NexConversationTurn{}
