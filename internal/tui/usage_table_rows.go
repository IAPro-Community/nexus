package tui

import (
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/conversation"
	"github.com/IAPro-Community/nexus/internal/localization"
)

// Pure account/session filtering, availability ranking and TUI labels.
// Split from usage_table.go without changing the sort, status or quota semantics.
func filterUsageRows(rows []UsageTableRow, query string) []UsageTableRow {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return rows
	}
	filtered := make([]UsageTableRow, 0, len(rows))
	for _, row := range rows {
		search := strings.ToLower(strings.Join([]string{
			row.Provider, row.Profile, row.Account, row.Plan,
			row.Group, row.FiveHour, row.Weekly, row.Status, row.ModelName,
		}, " "))
		if strings.Contains(search, query) {
			filtered = append(filtered, row)
		}
	}
	return filtered
}

func sortUsageRows(rows []UsageTableRow) {
	sort.SliceStable(rows, func(i, j int) bool {
		ai, aj := rowAvailability(rows[i]), rowAvailability(rows[j])
		if ai != aj {
			return ai > aj
		}
		if rows[i].Provider != rows[j].Provider {
			return rows[i].Provider < rows[j].Provider
		}
		if rows[i].Profile != rows[j].Profile {
			return rows[i].Profile < rows[j].Profile
		}
		return rows[i].Group < rows[j].Group
	})
}

func rowAvailability(row UsageTableRow) float64 {
	if row.IsUnconfigured || row.Status == "OFF" {
		return -5
	}
	switch row.Status {
	case "SEM DADOS":
		return -4
	case "RATE LTD", "RATE LIMITED":
		// Below any positive remaining % and below exhausted.
		return -3.5
	case "ESGOTADA":
		// Exhausted sorts below healthy %, above rate-limit / SEM DADOS.
		return -3
	}
	five, fiveOK := parseQuotaPercent(row.FiveHour)
	week, weekOK := parseQuotaPercent(row.Weekly)
	switch {
	case fiveOK && weekOK:
		if five < week {
			return five
		}
		return week
	case fiveOK:
		return five
	case weekOK:
		return week
	case row.Status == "OK", row.Status == "COM QUOTA", row.Status == "ESTIMADA":
		return 50
	default:
		return -2
	}
}

func parseQuotaPercent(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	if s == "" || s == "—" || strings.Contains(strings.ToUpper(s), "SEM") {
		return 0, false
	}
	idx := strings.Index(s, "%")
	if idx <= 0 {
		return 0, false
	}
	token := strings.TrimSpace(s[:idx])
	fields := strings.Fields(token)
	if len(fields) == 0 {
		return 0, false
	}
	n, err := strconv.ParseFloat(strings.TrimPrefix(fields[len(fields)-1], "~"), 64)
	if err != nil {
		return 0, false
	}
	return n, true
}

func filterSessionRows(sess []conversation.Conversation, query string) []conversation.Conversation {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return sess
	}
	var out []conversation.Conversation
	for _, s := range sess {
		search := strings.ToLower(strings.Join([]string{
			s.Provider, s.ID, s.Title, s.Workspace,
		}, " "))
		if strings.Contains(search, query) {
			out = append(out, s)
		}
	}
	return out
}

func formatProviderBadge(p string) string {
	switch strings.ToLower(p) {
	case "codex":
		return "CODEX"
	case "agy":
		return "AGY"
	case "claude":
		return "CLAUDE"
	case "opencode":
		return "OPENCD"
	case "gemini":
		return "GEMINI"
	case "cursor":
		return "CURSOR"
	default:
		s := strings.ToUpper(p)
		if len(s) > 6 {
			s = s[:6]
		}
		return s
	}
}

func formatTimeAgo(t time.Time) string {
	if t.IsZero() {
		return localization.T("tui.unknown")
	}
	diff := time.Since(t)
	if diff < time.Minute {
		return localization.T("tui.now")
	}
	if diff < time.Hour {
		return localization.T("tui.minutes_ago", map[string]any{"Count": int(diff.Minutes())})
	}
	if diff < 24*time.Hour {
		return localization.T("tui.hours_ago", map[string]any{"Count": int(diff.Hours())})
	}
	if diff < 48*time.Hour {
		return localization.T("tui.yesterday")
	}
	return localization.T("tui.days_ago", map[string]any{"Count": int(diff.Hours() / 24)})
}
