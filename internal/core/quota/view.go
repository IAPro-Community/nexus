package quota

import (
	"fmt"
	"sort"
	"strings"
	"time"

	sdkquota "github.com/IAPro-Community/IAPro-SDK/quota"
	"github.com/IAPro-Community/nexus/internal/core/model"
)

// QuotaView is the omnibus view model for quota display and scoring.
// Consumed by TUI, Web API, CLI, and Scheduler.
type QuotaView struct {
	Provider     string       `json:"provider"`
	Profile      string       `json:"profile"`
	Account      string       `json:"account"`
	Plan         string       `json:"plan"`
	Status       string       `json:"status"`
	Source       string       `json:"source"`
	ModelGroups  []ModelGroup `json:"model_groups"`
	FetchedAt    time.Time    `json:"fetched_at"`
	Available    bool         `json:"available"`
	AvailReasons AvailReasons `json:"avail_reasons"`
}

// AvailReasons holds structured reasons for availability status.
type AvailReasons struct {
	ExhaustedWindows []string `json:"exhausted_windows,omitempty"`
	RateLimited      bool     `json:"rate_limited,omitempty"`
	UnknownQuota     bool     `json:"unknown_quota,omitempty"`
	AuthRequired     bool     `json:"auth_required,omitempty"`
	AllOK            bool     `json:"all_ok,omitempty"`
}

// ModelGroup clusters usage windows by model family.
// A provider like AGY exposes two groups (Gemini, Claude/GPT).
// Simpler providers have a single unnamed group.
type ModelGroup = sdkquota.ModelGroup

// Window is a display-ready usage window within a model group.
type Window = sdkquota.Window

// Bottleneck returns the minimum remaining percentage across ALL windows and groups.
// The second return value identifies the bottleneck window kind.
// This is the tightest window for warnings ("Gemini 5h 0%"), not the account score.
// Prefer BestGroupRemaining for scheduling / quota_remaining.
func (qv *QuotaView) Bottleneck() (float64, string) {
	shared := sdkView(*qv)
	return shared.Bottleneck()
}

// GroupRemaining returns the capacity of one model group: min of its known
// windows (5h and weekly stack in the same pool). ok is false when the group
// has no scorable windows. Windows whose reset time is already past are ignored
// (the quota window has rolled over; stale used% must not look like exhaustion).
// BestGroupRemaining returns the best usable pool capacity: max of each
// group's GroupRemaining. Independent families (AGY Gemini vs Claude) do not
// cancel each other; a single-pool provider (Codex) collapses to min(5h, weekly).
func (qv *QuotaView) BestGroupRemaining() (remaining float64, ok bool) {
	shared := sdkView(*qv)
	return shared.BestGroupRemaining()
}

// UsableGroupCount returns how many model groups still have capacity (> 0).
func (qv *QuotaView) UsableGroupCount() int {
	shared := sdkView(*qv)
	return shared.UsableGroupCount()
}

// TotalGroupRemaining sums each group's GroupRemaining (zeros included when
// the group has scorable windows). Used with UsableGroupCount for scheduling.
func (qv *QuotaView) TotalGroupRemaining() (total float64, groups int) {
	shared := sdkView(*qv)
	return shared.TotalGroupRemaining()
}

// EffectiveCapacityScore encodes "most available" for scheduling:
// primary = usable group count, secondary = average group remaining.
// Formula: usableGroups*100 + totalGroupRemaining/max(numGroups,1)
// so AGY with 2 live families always beats AGY with 1 live family at 100%.
func (qv *QuotaView) EffectiveCapacityScore() (score float64, ok bool) {
	total, groups := qv.TotalGroupRemaining()
	if groups == 0 {
		return 0, false
	}
	avg := total / float64(groups)
	return float64(qv.UsableGroupCount())*100.0 + avg, true
}

// EffectiveCapacityRatio normalizes the internal multi-pool ranking signal
// to 0-1. It is not literal remaining quota and must not back public
// quota_remaining fields; use BestGroupRemaining for that contract.
func (qv *QuotaView) EffectiveCapacityRatio() (ratio float64, ok bool) {
	score, ok := qv.EffectiveCapacityScore()
	if !ok {
		return 0, false
	}
	_, groups := qv.TotalGroupRemaining()
	if groups == 0 {
		return 0, false
	}
	maxScore := float64(groups)*100.0 + 100.0
	if maxScore <= 0 {
		return 0, false
	}
	ratio = score / maxScore
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	return ratio, true
}

// CompactGroupSummary lists each group's capacity for compact UIs
// ("Gemini 0% · Claude 100%" or "5h 70% · weekly 95%" for a single pool).
func (qv *QuotaView) CompactGroupSummary() string {
	if qv == nil || len(qv.ModelGroups) == 0 {
		return ""
	}
	if qv.HasMultipleGroups() {
		parts := make([]string, 0, len(qv.ModelGroups))
		for _, g := range qv.ModelGroups {
			rem, ok := g.GroupRemaining()
			if !ok {
				continue
			}
			parts = append(parts, fmt.Sprintf("%s %.0f%%", shortGroupLabel(g), rem))
		}
		return strings.Join(parts, " · ")
	}
	g := qv.ModelGroups[0]
	parts := make([]string, 0, len(g.Windows))
	for _, w := range g.Windows {
		if w.Kind == "unknown" {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s %.0f%%", shortWindowLabel(w.Kind), w.Remaining))
	}
	return strings.Join(parts, " · ")
}

func shortGroupLabel(g ModelGroup) string {
	key := g.Key
	if key == "" {
		key = strings.ToLower(g.Name)
	}
	switch {
	case key == "gemini" || strings.Contains(key, "gemini"):
		return "Gemini"
	case key == "claude_gpt" || strings.Contains(key, "claude") || strings.Contains(key, "gpt"):
		return "Claude"
	case strings.TrimSpace(g.Name) != "":
		return g.Name
	default:
		return "Quota"
	}
}

func shortWindowLabel(kind string) string {
	switch kind {
	case "5h", "daily", "claude_5h", "claude_five_hour":
		return "5h"
	case "weekly", "claude_weekly":
		return "weekly"
	default:
		return kind
	}
}

// AllWindows returns a flat list of all windows across all groups.
// Used by scheduler for multi-window bottleneck scoring.
func (qv *QuotaView) AllWindows() []Window {
	var all []Window
	for _, g := range qv.ModelGroups {
		all = append(all, g.Windows...)
	}
	return all
}

// HasMultipleGroups returns true when the provider exposes separate model families.
func (qv *QuotaView) HasMultipleGroups() bool {
	return len(qv.ModelGroups) > 1
}

// IsAvailable returns true if at least one model group can accept work RIGHT NOW.
// A caller selecting a specific model must still verify that model's group.
// An agent is unavailable when every known quota pool is exhausted or when
// rate-limited. This matters for providers such as AGY, which exposes separate
// Gemini and Claude/GPT capacity pools.
// UNKNOWN quota is a hard gate for automatic allocation / Mission routing —
// missing evidence is never treated as capacity.
func (qv *QuotaView) IsAvailable() bool {
	return qv.Available
}

// AvailabilityLabel returns a human-readable availability status.
func (qv *QuotaView) AvailabilityLabel() string {
	if qv.AvailReasons.UnknownQuota {
		return "SEM DADOS"
	}
	if qv.Available {
		return "DISPONIVEL"
	}
	if qv.AvailReasons.RateLimited {
		return "RATE LIMITED"
	}
	if len(qv.AvailReasons.ExhaustedWindows) > 0 {
		return "QUOTA ESGOTADA"
	}
	if qv.AvailReasons.AuthRequired {
		return "SEM AUTENTICACAO"
	}
	return "INDISPONIVEL"
}

// ComputeAvailability analyzes all windows and sets Available + AvailReasons.
func (qv *QuotaView) ComputeAvailability() {
	qv.computeAvailabilityAt(time.Now())
}

func (qv *QuotaView) computeAvailabilityAt(now time.Time) {
	applySDKAvailability(qv, now)
}

// RenderBar renders a progress bar for a given remaining percentage.
func RenderBar(remaining float64, status string, width int) string {
	var pctPtr *float64
	if statusHasMeasuredPercent(status) {
		pctPtr = &remaining
	}
	return RenderProgressBar(model.UsageStatus(status), pctPtr, width)
}

// RenderBarWithPercent renders a progress bar with a percentage label.
func RenderBarWithPercent(remaining float64, status string, width int) string {
	bar := RenderBar(remaining, status, width)
	if statusHasMeasuredPercent(status) {
		return fmt.Sprintf("%s %2.0f%%", bar, remaining)
	}
	return bar
}

// statusHasMeasuredPercent reports whether a status is backed by a real
// measurement. RATE_LIMITED qualifies: the official provider read supplies exact
// percentages alongside the block.
func statusHasMeasuredPercent(status string) bool {
	switch model.UsageStatus(status) {
	case model.UsageLive, model.UsageCached, model.UsageEstimated, model.UsageRateLimited:
		return true
	default:
		return false
	}
}

// WindowLabel returns the display label for a window kind.
func WindowLabel(kind string) string {
	switch kind {
	case "5h", "daily":
		return "Limite 5 Horas"
	case "weekly":
		return "Limite Semanal"
	case "claude_5h", "claude_five_hour":
		return "Limite 5 Horas"
	case "claude_weekly":
		return "Limite Semanal"
	default:
		return kind
	}
}

// GroupDisplayName returns the display name for a model group.
func GroupDisplayName(groupKey string) string {
	switch groupKey {
	case "gemini":
		return "Gemini Models"
	case "claude_gpt":
		return "Claude & GPT Models"
	case "":
		return ""
	default:
		return groupKey
	}
}

// sortGroupOrder defines a stable display order for model groups.
var sortGroupOrder = map[string]int{
	"":           0,
	"gemini":     1,
	"claude_gpt": 2,
}

// SortModelGroups orders model groups for consistent display.
func SortModelGroups(groups []ModelGroup) {
	sort.Slice(groups, func(i, j int) bool {
		oi, okI := sortGroupOrder[groupSortKey(groups[i])]
		oj, okJ := sortGroupOrder[groupSortKey(groups[j])]
		if okI && okJ {
			return oi < oj
		}
		if okI {
			return true
		}
		if okJ {
			return false
		}
		return groups[i].Name < groups[j].Name
	})
}

func groupSortKey(g ModelGroup) string {
	if g.Key != "" {
		return g.Key
	}
	name := strings.ToLower(g.Name)
	switch {
	case strings.Contains(name, "gemini"):
		return "gemini"
	case strings.Contains(name, "claude") || strings.Contains(name, "gpt"):
		return "claude_gpt"
	default:
		return g.Name
	}
}

// GroupKeys returns a sorted slice of unique group keys from windows.
func GroupKeys(windows []model.UsageWindow) []string {
	seen := make(map[string]bool)
	for _, w := range windows {
		seen[w.Group] = true
	}
	var keys []string
	for k := range seen {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		oi, okI := sortGroupOrder[keys[i]]
		oj, okJ := sortGroupOrder[keys[j]]
		if okI && okJ {
			return oi < oj
		}
		if okI {
			return true
		}
		if okJ {
			return false
		}
		return keys[i] < keys[j]
	})
	return keys
}

// NormalizeGroupLabel removes trailing whitespace and normalizes group names for display.
func NormalizeGroupLabel(s string) string {
	return strings.TrimSpace(s)
}
