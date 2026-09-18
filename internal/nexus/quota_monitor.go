package nexus

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/events"
	"github.com/IAPro-Community/nexus/internal/control/notify"
	"github.com/IAPro-Community/nexus/internal/control/registry"
	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/core/quota"
)

const (
	DefaultLowQuotaThreshold = 0.30 // 30% remaining
	DefaultQuotaStepDelta    = 0.05 // 5% minimum drop to trigger consecutive notification
)

// QuotaNotificationAction represents an emitted alert action for an account.
type QuotaNotificationAction struct {
	Provider                string             `json:"provider"`
	Profile                 string             `json:"profile"`
	DisplayName             string             `json:"display_name"`
	Kind                    string             `json:"kind"` // "QUOTA_LOW" | "QUOTA_EXHAUSTED"
	RemainingPercent        float64            `json:"remaining_percent"`
	Group                   string             `json:"group,omitempty"`
	Window                  string             `json:"window,omitempty"`
	DataStatus              string             `json:"data_status,omitempty"`
	DataAge                 string             `json:"data_age,omitempty"`
	ResetDesc               string             `json:"reset_desc,omitempty"`
	Title                   string             `json:"title"`
	Body                    string             `json:"body"`
	RecommendedProvider     string             `json:"recommended_provider,omitempty"`
	RecommendedProfile      string             `json:"recommended_profile,omitempty"`
	RecommendedDisplayName  string             `json:"recommended_display_name,omitempty"`
	RecommendedQuotaPercent float64            `json:"recommended_quota_percent,omitempty"`
	AffectedRuntimeID       string             `json:"affected_runtime_id,omitempty"`
	DeliveredAt             time.Time          `json:"delivered_at"`
	AccountScope            model.AccountScope `json:"account_scope,omitempty"`
}

// AccountQuotaState tracks the notification lifecycle per provider account.
type AccountQuotaState struct {
	LastNotifiedRatio float64
	Exhausted         bool
	LastChecked       time.Time
	ResetCycle        string
}

// QuotaDropMonitor tracks quota decaimento and emits step-down alerts down to 0%,
// suppressing any further notifications once 0% is reached until quota is renewed.
type QuotaDropMonitor struct {
	mu               sync.Mutex
	states           map[string]*AccountQuotaState
	notifier         notify.Notifier
	eventBus         *events.Bus
	lowThreshold     float64
	stepDelta        float64
	statePath        string
	notifierDegraded bool
}

var (
	defaultQuotaDropMonitor     *QuotaDropMonitor
	defaultQuotaDropMonitorOnce sync.Once
)

// DefaultQuotaDropMonitor returns the global singleton monitor.
func DefaultQuotaDropMonitor() *QuotaDropMonitor {
	defaultQuotaDropMonitorOnce.Do(func() {
		defaultQuotaDropMonitor = NewPersistentQuotaDropMonitor(notify.Default(), events.DefaultBus(), DefaultLowQuotaThreshold, DefaultQuotaStepDelta)
	})
	return defaultQuotaDropMonitor
}

func NewPersistentQuotaDropMonitor(notifier notify.Notifier, bus *events.Bus, lowThreshold, stepDelta float64) *QuotaDropMonitor {
	m := NewQuotaDropMonitor(notifier, bus, lowThreshold, stepDelta)
	if dir, err := config.StateDir(); err == nil {
		_ = os.MkdirAll(dir, 0700)
		m.statePath = filepath.Join(dir, "quota-monitor-state.json")
		m.loadState()
	}
	return m
}

// NewQuotaDropMonitor creates a new monitor instance.
func NewQuotaDropMonitor(notifier notify.Notifier, bus *events.Bus, lowThreshold, stepDelta float64) *QuotaDropMonitor {
	if lowThreshold <= 0 || lowThreshold > 1 {
		lowThreshold = DefaultLowQuotaThreshold
	}
	if stepDelta <= 0 || stepDelta > 1 {
		stepDelta = DefaultQuotaStepDelta
	}
	return &QuotaDropMonitor{
		states:       make(map[string]*AccountQuotaState),
		notifier:     notifier,
		eventBus:     bus,
		lowThreshold: lowThreshold,
		stepDelta:    stepDelta,
	}
}

// Reset clears state for testing or system reload.
func (m *QuotaDropMonitor) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.states = make(map[string]*AccountQuotaState)
	m.persistLocked()
}

func (m *QuotaDropMonitor) loadState() {
	if m.statePath == "" {
		return
	}
	data, err := os.ReadFile(m.statePath)
	if err != nil {
		return
	}
	var states map[string]*AccountQuotaState
	if json.Unmarshal(data, &states) == nil && states != nil {
		m.states = states
	}
}

func (m *QuotaDropMonitor) persistLocked() {
	if m.statePath == "" {
		return
	}
	data, err := json.Marshal(m.states)
	if err != nil {
		return
	}
	tmp := fmt.Sprintf("%s.tmp.%d", m.statePath, time.Now().UnixNano())
	if err := os.WriteFile(tmp, data, 0600); err == nil {
		_ = os.Rename(tmp, m.statePath)
	}
}

// CheckAccount inspects an account and emits a notification if quota dropped below thresholds.
func (m *QuotaDropMonitor) CheckAccount(acc ProviderAccount) *QuotaNotificationAction {
	return m.CheckAccountWithPool(acc, nil)
}

// CheckAccountWithPool inspects an account with awareness of the broader resource pool
// to suggest the best failover candidate and identify any active runtime.
func (m *QuotaDropMonitor) CheckAccountWithPool(acc ProviderAccount, pool []ProviderAccount) *QuotaNotificationAction {
	// Consumption alerts require a trustworthy observation. Estimated and
	// unknown snapshots are intentionally silent; the resident service owns the
	// separate degraded-capacity event.
	if acc.QuotaView == nil || !isTrustworthyQuotaStatus(acc.QuotaView.Status) {
		return nil
	}
	displayName := acc.DisplayName
	if displayName == "" {
		displayName = fmt.Sprintf("%s (%s)", acc.Provider, acc.Profile)
	}

	for _, group := range acc.QuotaView.ModelGroups {
		for _, window := range group.Windows {
			if window.Kind == "unknown" || window.Kind == "" {
				continue
			}
			if action := m.checkWindow(acc, group.Key, window, displayName, pool); action != nil {
				return action
			}
		}
	}
	// Preserve compatibility with callers that only provide the aggregate
	// account fields, while all real QuotaView windows use independent keys.
	if acc.QuotaTotal > 0 {
		remaining := acc.QuotaRemaining / acc.QuotaTotal
		window := quotaWindow{kind: "account", group: "account", remaining: remaining}
		return m.checkWindow(acc, window.group, window.toWindow(), displayName, pool)
	}
	return nil
}

func isTrustworthyQuotaStatus(status string) bool {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "LIVE", "CACHED", "OK", "RATE_LIMITED":
		return true
	default:
		return false
	}
}

type quotaWindow struct {
	kind, group string
	remaining   float64
}

func (w quotaWindow) toWindow() quota.Window {
	return quota.Window{Kind: w.kind, Status: "LIVE", Remaining: w.remaining * 100}
}

func (m *QuotaDropMonitor) checkWindow(acc ProviderAccount, group string, window quota.Window, displayName string, pool []ProviderAccount) *QuotaNotificationAction {
	ratio := window.Remaining / 100
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	// ResetDesc is display text and commonly changes on every refresh (for
	// example, a countdown from "2h 00m" to "1h 59m"). It must not be part of
	// the state identity, otherwise the same low-quota condition bypasses the
	// anti-spam guard on every polling cycle.
	key := fmt.Sprintf("%s:%s:%s", accountKey(acc), group, window.Kind)
	now := time.Now()
	m.mu.Lock()
	st := m.states[key]
	if st == nil {
		st = &AccountQuotaState{LastNotifiedRatio: 1}
		m.states[key] = st
	}
	st.LastChecked = now
	if ratio > m.lowThreshold || ratio >= st.LastNotifiedRatio+0.15 {
		st.Exhausted = false
		st.LastNotifiedRatio = ratio
		m.persistLocked()
		m.mu.Unlock()
		return nil
	}
	resetDesc := window.ResetDesc
	pct := math.Round(ratio * 100)
	if ratio <= 0.001 || acc.RateLimited {
		if st.Exhausted {
			m.mu.Unlock()
			return nil
		}
		st.Exhausted = true
		st.LastNotifiedRatio = 0
		m.persistLocked()
		m.mu.Unlock()
		action := &QuotaNotificationAction{Provider: acc.Provider, Profile: acc.Profile, AccountScope: acc.Scope, DisplayName: displayName, Group: group, Window: window.Kind, DataStatus: acc.QuotaView.Status, DataAge: quota.FormatFreshness(acc.QuotaView.FetchedAt), Kind: string(events.EventQuotaExhausted), RemainingPercent: 0, ResetDesc: resetDesc, Title: fmt.Sprintf("Nexus · Quota Esgotada: %s", displayName), DeliveredAt: now}
		if alt := findBestAlternative(acc, pool); alt != nil {
			action.RecommendedProvider = alt.Account.Provider
			action.RecommendedProfile = alt.Account.Profile
			action.RecommendedDisplayName = alt.Account.DisplayName
			if alt.Account.QuotaTotal > 0 {
				action.RecommendedQuotaPercent = math.Round((alt.Account.QuotaRemaining / alt.Account.QuotaTotal) * 100)
			}
		}
		action.AffectedRuntimeID = findActiveRuntime(acc.Provider, acc.Profile, acc.Scope)
		action.Body = fmt.Sprintf("A quota de %s (%s · %s) atingiu 0%%.", displayName, group, window.Kind)
		if action.RecommendedDisplayName != "" {
			action.Body += fmt.Sprintf(" Recomendado continuar com %s.", action.RecommendedDisplayName)
		}
		m.dispatch(action)
		return action
	}
	if ratio <= m.lowThreshold {
		crossed := st.LastNotifiedRatio > m.lowThreshold || ratio <= st.LastNotifiedRatio-m.stepDelta
		for _, milestone := range []float64{0.20, 0.10, 0.05} {
			if st.LastNotifiedRatio > milestone && ratio <= milestone {
				crossed = true
			}
		}
		if crossed {
			st.LastNotifiedRatio = ratio
			st.Exhausted = false
			m.persistLocked()
			m.mu.Unlock()
			action := &QuotaNotificationAction{Provider: acc.Provider, Profile: acc.Profile, AccountScope: acc.Scope, DisplayName: displayName, Group: group, Window: window.Kind, DataStatus: acc.QuotaView.Status, DataAge: quota.FormatFreshness(acc.QuotaView.FetchedAt), Kind: string(events.EventQuotaLow), RemainingPercent: pct, ResetDesc: resetDesc, Title: fmt.Sprintf("Nexus · Quota Baixa: %s", displayName), DeliveredAt: now}
			action.Body = fmt.Sprintf("A quota de %s (%s · %s) caiu para %.0f%%.", displayName, group, window.Kind, pct)
			m.dispatch(action)
			return action
		}
	}
	m.mu.Unlock()
	return nil
}

// CheckAccounts processes a slice of accounts and returns all triggered actions.
func (m *QuotaDropMonitor) CheckAccounts(accounts []ProviderAccount) []QuotaNotificationAction {
	var actions []QuotaNotificationAction
	for _, acc := range accounts {
		if act := m.CheckAccountWithPool(acc, accounts); act != nil {
			actions = append(actions, *act)
		}
	}
	return actions
}

func accountKey(acc ProviderAccount) string {
	if acc.Scope.Verifiable() {
		return acc.Scope.Key()
	}
	return fmt.Sprintf("legacy/%s/%s", acc.Provider, acc.Profile)
}

// dispatch delivers the notification through the OS notifier and the internal event bus.
func (m *QuotaDropMonitor) dispatch(action *QuotaNotificationAction) {
	if m.notifier != nil {
		tag := fmt.Sprintf("quota:%s:%s:%s", action.Provider, action.Profile, action.Kind)
		if err := m.notifier.Notify(notify.Payload{
			Title: action.Title,
			Body:  action.Body,
			Tag:   tag,
		}); err != nil {
			log.Printf("nexus quota native notification unavailable: %v", err)
			m.mu.Lock()
			firstFailure := !m.notifierDegraded
			m.notifierDegraded = true
			m.mu.Unlock()
			if firstFailure && m.eventBus != nil {
				m.eventBus.Publish(events.NewEvent("", action.Provider, action.Profile, events.EventQuotaMonitorDegraded, "native notifications unavailable", map[string]any{"reason": err.Error()}))
			}
		} else {
			m.mu.Lock()
			m.notifierDegraded = false
			m.mu.Unlock()
		}
	}

	if m.eventBus != nil {
		eventType := events.EventQuotaLow
		if action.Kind == string(events.EventQuotaExhausted) {
			eventType = events.EventQuotaExhausted
		}
		data := map[string]any{
			"provider":          action.Provider,
			"profile":           action.Profile,
			"display_name":      action.DisplayName,
			"remaining_percent": action.RemainingPercent,
			"reset_desc":        action.ResetDesc,
			"title":             action.Title,
			"group":             action.Group,
			"window":            action.Window,
			"data_status":       action.DataStatus,
			"data_age":          action.DataAge,
		}
		if action.AccountScope.Verifiable() {
			data["account_scope"] = action.AccountScope
		}
		if action.RecommendedProvider != "" {
			data["recommended_provider"] = action.RecommendedProvider
			data["recommended_profile"] = action.RecommendedProfile
			data["recommended_display_name"] = action.RecommendedDisplayName
			data["recommended_quota_percent"] = action.RecommendedQuotaPercent
		}
		if action.AffectedRuntimeID != "" {
			data["affected_runtime_id"] = action.AffectedRuntimeID
		}

		event := events.NewEvent(
			action.AffectedRuntimeID, // attaches runtimeID if an active runtime is affected
			action.Provider,
			action.Profile,
			eventType,
			action.Body,
			data,
		)
		m.eventBus.Publish(event.WithAccountScope(action.AccountScope))
	}
}

func findBestAlternative(source ProviderAccount, pool []ProviderAccount) *ResourceCandidate {
	candidates := make([]ProviderAccount, 0, len(pool))
	for _, a := range pool {
		if strings.EqualFold(a.Provider, source.Provider) && strings.EqualFold(a.Profile, source.Profile) {
			continue
		}
		if a.RateLimited {
			continue
		}
		if a.QuotaTotal > 0 && a.QuotaRemaining <= 0 {
			continue
		}
		candidates = append(candidates, a)
	}
	if len(candidates) == 0 {
		return nil
	}
	rec := RecommendResources(candidates, TaskRequirements{TaskKind: "coding"}, PolicyBalanced)
	return rec.Recommended
}

func findActiveRuntime(provider, profile string, scope model.AccountScope) string {
	reg := registry.DefaultRegistry()
	if reg == nil {
		return ""
	}
	sessions := reg.List()
	var latestID string
	var latestTime time.Time
	for _, s := range sessions {
		if strings.EqualFold(s.ProviderID, provider) && strings.EqualFold(s.ProfileID, profile) {
			if scope.Verifiable() && s.AccountScope != scope {
				continue
			}
			if s.State == registry.StateRunning || s.State == registry.StateWaiting || s.State == registry.StateStarting {
				if s.StartedAt.After(latestTime) {
					latestTime = s.StartedAt
					latestID = s.RuntimeID
				}
			}
		}
	}
	return latestID
}
