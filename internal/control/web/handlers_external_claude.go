package web

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/IAPro-Community/nexus/internal/nexus/externalusage"
)

// handleExternalClaudeUsage is a manual, opt-in operation. No timer, watcher,
// CLI hook or request from the Usage screen ever starts transcript scanning.
// GET returns metadata already stored; POST requires explicit consent and
// scans only ~/.claude/projects of this server's local OS user.
func (h *NexusHandler) handleExternalClaudeUsage(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.nexus == nil || h.nexus.Store() == nil {
		writeError(w, http.StatusServiceUnavailable, "external usage store unavailable")
		return
	}
	st := h.nexus.Store()
	w.Header().Set("Cache-Control", "private, no-store")
	switch r.Method {
	case http.MethodGet:
		days := 7
		if r.URL.Query().Get("days") == "30" { days = 30 }
		summary, err := st.SummarizeExternalClaude(r.Context(), days)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "external usage unavailable")
			return
		}
		writeJSON(w, http.StatusOK, summary)
	case http.MethodPost:
		var payload struct {
			Consent bool `json:"consent"`
		}
		decoder := json.NewDecoder(io.LimitReader(r.Body, 1024))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&payload); err != nil || !payload.Consent {
			writeError(w, http.StatusBadRequest, "explicit one-time consent is required")
			return
		}
		home, err := os.UserHomeDir()
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "local Claude directory unavailable")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()
		result, err := externalusage.ScanClaudeProjects(ctx, st, filepath.Join(home, ".claude", "projects"))
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				writeError(w, http.StatusNotFound, "local Claude projects not found")
			} else {
				writeError(w, http.StatusServiceUnavailable, "local Claude scan unavailable")
			}
			return
		}
		summary, err := st.SummarizeExternalClaude(r.Context(), 7)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "external usage summary unavailable")
			return
		}
		writeJSON(w, http.StatusOK, struct {
			Scan externalusage.ScanReport `json:"scan"`
			Summary any `json:"summary"`
		}{Scan:result, Summary:summary})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
