package web

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/IAPro-Community/nexus/internal/nexus"
)

// handlePlanParallelChallenge POST /api/v1/plans/{id}/parallel-challenge
//
// Evaluates a merge-free parallel challenge over the completed MissionRuns of
// one plan. Criteria are declared by the caller (the endpoint never invents
// them); each participant is one completed run with its own isolated
// workspace. The response always carries auto_merge_allowed=false with the
// explicit policy: the challenge informs, the human merges.
func (h *NexusHandler) handlePlanParallelChallenge(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	planID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/v1/plans/"), "/parallel-challenge")
	if strings.TrimSpace(planID) == "" {
		writeError(w, http.StatusBadRequest, "plan id is required")
		return
	}
	var req struct {
		Criteria []nexus.AgentCompareCriterion `json:"criteria"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid parallel-challenge request: "+err.Error())
		return
	}
	result, err := h.nexus.EvaluatePlanChallenge(r.Context(), planID, req.Criteria)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"result": result})
}
