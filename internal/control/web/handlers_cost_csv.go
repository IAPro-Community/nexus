package web

import (
	"encoding/csv"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

// Export only metadata from the bounded canonical cost ledger. Blank numbers
// mean unknown, never zero. Raw prompts, account scope, IDs and paths are
// intentionally absent from this report.
func (h *NexusHandler) handleCostCSV(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	svc, ok := h.costService()
	if !ok {
		writeError(w, http.StatusServiceUnavailable, "cost ledger unavailable")
		return
	}
	days := 30
	if value := r.URL.Query().Get("days"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > 365 {
			writeError(w, http.StatusBadRequest, "days must be 1-365")
			return
		}
		days = parsed
	}
	rows, err := svc.Observations(r.Context(), store.CostObservationFilter{
		Since: time.Now().UTC().Add(-time.Duration(days)*24*time.Hour),
		Limit: 1000, ProjectID: r.URL.Query().Get("project_id"),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "cost observations unavailable")
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Content-Disposition", "attachment; filename=nexus-cost-observations.csv")
	writer := csv.NewWriter(w)
	_ = writer.Write([]string{
		"observed_at","provider","profile","model","project_id",
		"input_tokens","cached_input_tokens","output_tokens","reasoning_tokens",
		"cost_microusd","cost_kind","source",
	})
	safe := func(value string) string {
		first := strings.TrimLeft(value, " \t\r\n")
		if len(first) > 0 && strings.ContainsAny(first[:1], "=+-@") { return "'" + value }
		return value
	}
	optional := func(v *int64) string {
		if v == nil { return "" }
		return strconv.FormatInt(*v, 10)
	}
	for _, row := range rows {
		_ = writer.Write([]string{
			row.ObservedAt.UTC().Format(time.RFC3339), safe(row.Provider), safe(row.Profile),
			safe(row.Model), safe(row.ProjectID), optional(row.InputTokens),
			optional(row.CachedInputTokens), optional(row.OutputTokens),
			optional(row.ReasoningTokens), optional(row.CostMicroUSD),
			safe(row.CostKind), safe(row.Source),
		})
	}
	writer.Flush()
}
