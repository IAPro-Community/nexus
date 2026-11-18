package web

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/nexus"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

// Nex candidate derives only from verified HUD operational evidence.
// Never infer a bug, a conclusion, or permission to execute from a model.
type nexProactiveCandidate struct {
	Fingerprint string    `json:"fingerprint"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Target      string    `json:"target"`
	Evidence    string    `json:"evidence"`
	ObservedAt  time.Time `json:"observed_at"`
}

func nexCandidateFingerprint(projectID, kind, source string) string {
	data := sha256.Sum256([]byte(projectID + "|" + kind + "|" + source))
	return "nex-" + hex.EncodeToString(data[:12])
}

// Propose at most three prioritized, non-sensitive navigations. Candidate IDs
// remain stable while the same underlying event persists, across clients.
func nexProactiveCandidates(hud nexus.HUDState, now time.Time) []nexProactiveCandidate {
	out := []nexProactiveCandidate{}
	for _, mission := range hud.Missions {
		if !mission.NeedsUser || mission.RunID == "" {
			continue
		}
		if hud.ProjectID != "" && mission.ProjectID != hud.ProjectID {
			continue
		}
		out = append(out, nexProactiveCandidate{
			Fingerprint: nexCandidateFingerprint(hud.ProjectID, "decision", mission.RunID+"|"+mission.State+"|"+mission.CurrentTaskID),
			Title:       "Missão aguardando decisão",
			Description: "Uma execução solicita sua intervenção. Abra as missões para revisar.",
			Target:      "missions", Evidence: "mission_needs_user", ObservedAt: mission.UpdatedAt,
		})
	}
	for _, alert := range hud.Alerts {
		if alert.Acked || alert.Fingerprint == "" ||
			(alert.SnoozedUntil != nil && alert.SnoozedUntil.After(now)) {
			continue
		}
		out = append(out, nexProactiveCandidate{
			Fingerprint: nexCandidateFingerprint(hud.ProjectID, "alert", alert.Fingerprint),
			Title:       "Alerta de operação",
			Description: "Existe um alerta não reconhecido. Confira os dados de recursos.",
			Target:      "usage", Evidence: "hud_alert", ObservedAt: alert.LastObservedAt,
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Evidence != out[j].Evidence {
			return out[i].Evidence == "mission_needs_user"
		}
		return out[i].ObservedAt.After(out[j].ObservedAt)
	})
	if len(out) > 3 {
		out = out[:3]
	}
	return out
}

type nexProactivityRequest struct {
	ProjectID   string                          `json:"project_id"`
	Operation   string                          `json:"operation"`
	Settings    store.NexProactivityPreferences `json:"settings"`
	Fingerprint string                          `json:"fingerprint"`
	Minutes     int                             `json:"minutes"`
}

func (h *NexusHandler) handleNexProactivity(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.nexus == nil || h.hud == nil {
		writeError(w, http.StatusServiceUnavailable, "Nex context unavailable")
		return
	}
	st, err := h.nexus.OpenProject()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "Nex preferences unavailable")
		return
	}
	projectID := strings.TrimSpace(r.URL.Query().Get("project_id"))
	var body nexProactivityRequest
	switch r.Method {
	case http.MethodGet:
	case http.MethodPost:
		if err := json.NewDecoder(io.LimitReader(r.Body, 4<<10)).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid Nex proactivity request")
			return
		}
		projectID = strings.TrimSpace(body.ProjectID)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if len(projectID) > 128 {
		writeError(w, http.StatusBadRequest, "invalid project scope")
		return
	}
	if projectID != "" {
		if _, err := st.GetProject(projectID); err != nil {
			writeError(w, http.StatusNotFound, "project not found")
			return
		}
	}
	if r.Method == http.MethodPost && body.Operation == "configure" {
		if err := st.SetNexProactivityPreferences(projectID, body.Settings); err != nil {
			writeError(w, http.StatusBadRequest, "invalid Nex proactivity preferences")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "saved"})
		return
	}
	prefs, err := st.GetNexProactivityPreferences(projectID)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "Nex preferences unavailable")
		return
	}
	if r.Method == http.MethodPost && (body.Operation == "dismiss" || body.Operation == "snooze") {
		status := "DISMISSED"
		if body.Operation == "snooze" {
			status = "SNOOZED"
		}
		if err := st.SetNexSuggestionDecision(projectID, body.Fingerprint, status, body.Minutes, time.Now()); err != nil {
			writeError(w, http.StatusBadRequest, "Nex suggestion decision unavailable")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "saved"})
		return
	}
	if prefs.Mode == "manual" {
		if r.Method == http.MethodPost {
			writeError(w, http.StatusForbidden, "Nex proactivity is disabled")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"settings": prefs, "candidates": []nexProactiveCandidate{},
		})
		return
	}
	now := time.Now()
	if prefs.Mode == "proactive" && store.NexQuietHours(prefs, now) {
		if r.Method == http.MethodPost {
			writeJSON(w, http.StatusOK, map[string]any{"claimed": false})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"settings": prefs, "candidates": []nexProactiveCandidate{}})
		return
	}
	state, err := h.hud.State(r.Context(), projectID)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "Nex projection unavailable")
		return
	}
	candidates := nexProactiveCandidates(state, now)
	if r.Method == http.MethodGet {
		writeJSON(w, http.StatusOK, map[string]any{"settings": prefs, "candidates": candidates})
		return
	}
	if body.Operation != "claim" || prefs.Mode != "proactive" {
		writeError(w, http.StatusBadRequest, "unsupported Nex operation")
		return
	}
	for _, candidate := range candidates {
		if candidate.Fingerprint == body.Fingerprint {
			claimed, claimErr := st.TryClaimNexSuggestion(projectID, candidate.Fingerprint, now)
			if claimErr != nil {
				writeError(w, http.StatusServiceUnavailable, "could not claim Nex suggestion")
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{
				"claimed": claimed, "suggestion": candidate,
			})
			return
		}
	}
	writeError(w, http.StatusConflict, "Nex suggestion is no longer active")
}
