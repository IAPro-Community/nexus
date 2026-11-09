package web

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/IAPro-Community/nexus/internal/nexus"
)

func (h *NexusHandler) handleComparisonCampaigns(w http.ResponseWriter, r *http.Request) {
	if h.comparisonCampaigns == nil {
		writeError(w, http.StatusServiceUnavailable, "comparison campaign service unavailable")
		return
	}
	switch r.Method {
	case http.MethodGet:
		projectID := strings.TrimSpace(r.URL.Query().Get("project_id"))
		if projectID == "" {
			writeError(w, http.StatusBadRequest, "project_id is required")
			return
		}
		items, err := h.comparisonCampaigns.List(r.Context(), projectID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, items)
	case http.MethodPost:
		var body nexus.CreateComparisonCampaignRequest
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid body")
			return
		}
		view, err := h.comparisonCampaigns.Create(r.Context(), body)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, view)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *NexusHandler) handleComparisonCampaignDetail(w http.ResponseWriter, r *http.Request) {
	if h.comparisonCampaigns == nil {
		writeError(w, http.StatusServiceUnavailable, "comparison campaign service unavailable")
		return
	}
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/comparison-campaigns/"), "/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		writeError(w, http.StatusNotFound, "comparison campaign not found")
		return
	}
	if len(parts) == 1 && r.Method == http.MethodGet {
		view, err := h.comparisonCampaigns.Get(r.Context(), parts[0])
		if err != nil {
			writeError(w, http.StatusNotFound, "comparison campaign not found")
			return
		}
		writeJSON(w, http.StatusOK, view)
		return
	}
	if len(parts) == 2 && parts[1] == "launch" && r.Method == http.MethodPost {
		var body nexus.LaunchComparisonCampaignRequest
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid body")
			return
		}
		view, err := h.comparisonCampaigns.Launch(r.Context(), parts[0], body)
		if err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, view)
		return
	}
	writeError(w, http.StatusMethodNotAllowed, "method not allowed")
}

func (h *NexusHandler) handleComparisons(w http.ResponseWriter, r *http.Request) {
	if h.comparisons == nil {
		writeError(w, http.StatusServiceUnavailable, "comparison service unavailable")
		return
	}
	switch r.Method {
	case http.MethodGet:
		projectID := strings.TrimSpace(r.URL.Query().Get("project_id"))
		if projectID == "" {
			writeError(w, http.StatusBadRequest, "project_id is required")
			return
		}
		items, err := h.comparisons.List(r.Context(), projectID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, items)
	case http.MethodPost:
		var body struct {
			ProjectID string `json:"project_id"`
			RunAID    string `json:"run_a_id"`
			RunBID    string `json:"run_b_id"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid body")
			return
		}
		view, err := h.comparisons.Create(r.Context(), body.ProjectID, body.RunAID, body.RunBID)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, view)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *NexusHandler) handleComparisonDetail(w http.ResponseWriter, r *http.Request) {
	if h.comparisons == nil {
		writeError(w, http.StatusServiceUnavailable, "comparison service unavailable")
		return
	}
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/comparisons/"), "/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		writeError(w, http.StatusNotFound, "comparison not found")
		return
	}
	if len(parts) == 1 && r.Method == http.MethodGet {
		view, err := h.comparisons.Get(r.Context(), parts[0])
		if err != nil {
			writeError(w, http.StatusNotFound, "comparison not found")
			return
		}
		writeJSON(w, http.StatusOK, view)
		return
	}
	if len(parts) == 2 && parts[1] == "decision" && r.Method == http.MethodPost {
		var body struct {
			Decision string `json:"decision"`
			Reason   string `json:"reason"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid body")
			return
		}
		view, err := h.comparisons.Decide(r.Context(), parts[0], body.Decision, body.Reason)
		if err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, view)
		return
	}
	writeError(w, http.StatusMethodNotAllowed, "method not allowed")
}
