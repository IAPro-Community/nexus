package web

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/IAPro-Community/nexus/internal/control/notify"
)

func recordNexHubFromRequest(r *http.Request) {
	if r == nil {
		return
	}
	hub := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Nexus-Nex-Hub")))
	if hub == "" {
		return
	}
	dndRaw := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Nexus-Nex-Dnd")))
	dnd := dndRaw == "1" || dndRaw == "true"
	switch hub {
	case "active":
		notify.ReportNexHubHeartbeat(true, dnd, "nex_active", 0)
	case "paused":
		notify.ReportNexHubHeartbeat(false, dnd, "nex_paused", 0)
	case "hidden":
		notify.ReportNexHubHeartbeat(false, dnd, "nex_hidden", 0)
	case "offline", "unavailable":
		notify.ReportNexHubHeartbeat(false, dnd, "nex_offline", 0)
	default:
		notify.ReportNexHubHeartbeat(false, dnd, "nex_unavailable", 0)
	}
}

func (h *NexusHandler) handleNexHubStatus(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, notify.CurrentNexHubStatus())
	case http.MethodPost:
		var req struct {
			CanNotify    bool   `json:"can_notify"`
			DoNotDisturb bool   `json:"do_not_disturb"`
			Reason       string `json:"reason"`
		}
		if r.Body != nil {
			if err := json.NewDecoder(io.LimitReader(r.Body, 16<<10)).Decode(&req); err != nil {
				writeError(w, http.StatusBadRequest, "invalid request body")
				return
			}
		}
		status := notify.ReportNexHubHeartbeat(req.CanNotify, req.DoNotDisturb, req.Reason, 0)
		writeJSON(w, http.StatusOK, status)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
