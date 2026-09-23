package web

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/nexus"
	"github.com/IAPro-Community/nexus/internal/nexus/runner"
)

// handleMissionSchedules creates/lists/cancels durable Mission schedules.
func (h *NexusHandler) handleMissionSchedules(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		items, err := h.nexus.ListMissionSchedules(r.Context(), strings.TrimSpace(r.URL.Query().Get("project_id")))
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, items)
	case http.MethodPost:
		var body struct {
			PlanID                string                  `json:"plan_id"`
			PlanRevision          int                     `json:"plan_revision"`
			Mode                  string                  `json:"mode"`
			ScheduledFor          string                  `json:"scheduled_for"`
			AfterRunID            string                  `json:"after_run_id"`
			AgentID               string                  `json:"agent_id"`
			CancelID              string                  `json:"cancel_id"`
			CostConfirmationToken string                  `json:"cost_confirmation_token"`
			Contract              runner.AutonomyContract `json:"contract"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid body")
			return
		}
		if body.CancelID != "" {
			if err := h.nexus.CancelMissionSchedule(r.Context(), body.CancelID); err != nil {
				writeError(w, http.StatusConflict, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, map[string]bool{"canceled": true})
			return
		}
		contract := body.Contract
		if contract.MaxRetries == 0 {
			contract = runner.DefaultAutonomyContract()
		}
		var when *time.Time
		if strings.TrimSpace(body.ScheduledFor) != "" {
			parsed, err := time.Parse(time.RFC3339, body.ScheduledFor)
			if err != nil {
				writeError(w, http.StatusBadRequest, "scheduled_for must be RFC3339")
				return
			}
			when = &parsed
		}
		item, err := h.nexus.ScheduleMissionWithOptions(
			r.Context(),
			body.PlanID,
			body.PlanRevision,
			body.Mode,
			when,
			body.AfterRunID,
			body.AgentID,
			contract,
			nexus.MissionAdmissionOptions{CostConfirmationToken: body.CostConfirmationToken},
		)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, item)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
