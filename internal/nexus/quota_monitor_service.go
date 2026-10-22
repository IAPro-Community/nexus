package nexus

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/events"
	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/core/provider/adapters/codex"
	"github.com/IAPro-Community/nexus/internal/core/quota"
	"github.com/IAPro-Community/nexus/internal/profile"
)

const quotaMonitorInterval = time.Minute
const quotaMonitorLeaseTTL = 2 * quotaMonitorInterval

var quotaLeaseSequence atomic.Uint64

// QuotaMonitorService keeps quota alerts independent from any UI surface.
// A single service belongs to one Nexus process and is stopped with its context.
type QuotaMonitorService struct {
	monitor    *QuotaDropMonitor
	bus        *events.Bus
	mu         sync.Mutex
	ctx        context.Context
	cancel     context.CancelFunc
	started    bool
	wg         sync.WaitGroup
	degraded   map[string]bool
	leasePath  string
	leaseToken string
	leaseTTL   time.Duration
	continuity func(context.Context, string, ContinuityReason)
}

// SetContinuityHandler connects an explicitly observed quota exhaustion to
// a supervised runtime handoff. It does not react to inferred text or
// unsupported/opaque quota observations.
func (s *QuotaMonitorService) SetContinuityHandler(handler func(context.Context, string, ContinuityReason)) {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.continuity = handler
	s.mu.Unlock()
}

func NewQuotaMonitorService(monitor *QuotaDropMonitor, bus *events.Bus) *QuotaMonitorService {
	if monitor == nil {
		monitor = DefaultQuotaDropMonitor()
	}
	if bus == nil {
		bus = events.DefaultBus()
	}
	service := &QuotaMonitorService{
		monitor:    monitor,
		bus:        bus,
		degraded:   make(map[string]bool),
		leaseTTL:   quotaMonitorLeaseTTL,
		leaseToken: fmt.Sprintf("%d-%d-%d", os.Getpid(), time.Now().UnixNano(), quotaLeaseSequence.Add(1)),
	}
	if dir, err := config.StateDir(); err == nil {
		if os.MkdirAll(dir, 0700) == nil {
			service.leasePath = filepath.Join(dir, "quota-monitor-leader.lease")
		}
	}
	return service
}

// Start is idempotent. The first pass runs immediately, then the ticker wakes
// once per minute; it never polls in a tight loop.
func (s *QuotaMonitorService) Start(ctx context.Context) {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return
	}
	s.ctx, s.cancel = context.WithCancel(ctx)
	s.started = true
	s.wg.Add(1)
	s.mu.Unlock()
	go s.loop()
}

func (s *QuotaMonitorService) loop() {
	defer s.wg.Done()
	if s.acquireOrRenewLease() {
		s.check()
	}
	ticker := time.NewTicker(quotaMonitorInterval)
	defer ticker.Stop()
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			if s.acquireOrRenewLease() {
				s.check()
			}
		}
	}
}

func (s *QuotaMonitorService) Stop() {
	s.mu.Lock()
	if !s.started {
		s.mu.Unlock()
		return
	}
	s.cancel()
	s.started = false
	s.mu.Unlock()
	s.wg.Wait()
	s.releaseLease()
}

// acquireOrRenewLease elects one Nexus process as the quota collector. The
// lease is deliberately a small O_EXCL file so it works on Unix and Windows
// without a daemon or a platform-specific lock API. A crashed leader is
// recoverable after leaseTTL; an active leader renews its file timestamp.
func (s *QuotaMonitorService) acquireOrRenewLease() bool {
	s.mu.Lock()
	path, token, ttl := s.leasePath, s.leaseToken, s.leaseTTL
	s.mu.Unlock()
	if path == "" || token == "" || ttl <= 0 {
		s.emitDegraded("lease", "quota collector lease is unavailable")
		return false
	}
	if data, err := os.ReadFile(path); err == nil && strings.TrimSpace(string(data)) == token {
		_ = os.Chtimes(path, time.Now(), time.Now())
		return true
	} else if err == nil {
		if info, statErr := os.Stat(path); statErr == nil && time.Since(info.ModTime()) < ttl {
			return false
		}
		_ = os.Remove(path)
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return false
	}
	_, writeErr := file.WriteString(token + "\n")
	closeErr := file.Close()
	if writeErr != nil || closeErr != nil {
		_ = os.Remove(path)
		return false
	}
	return true
}

func (s *QuotaMonitorService) releaseLease() {
	s.mu.Lock()
	path, token := s.leasePath, s.leaseToken
	s.mu.Unlock()
	if path == "" || token == "" {
		return
	}
	if data, err := os.ReadFile(path); err == nil && strings.TrimSpace(string(data)) == token {
		_ = os.Remove(path)
	}
}

func (s *QuotaMonitorService) check() {
	profiles, err := profile.List()
	if err != nil {
		s.emitDegraded("profiles", "unable to enumerate profiles: "+err.Error())
		return
	}
	s.mu.Lock()
	ctx := s.ctx
	s.mu.Unlock()
	if ctx == nil {
		ctx = context.Background()
	}
	for _, p := range profiles {
		if p.Disabled {
			continue
		}
		account := profile.GetAccountInfo(p.Provider, p.Name)
		if !account.Authenticated {
			// Emit a degraded event so the UI and notifications surface
			// which accounts need re-authentication (e.g. expired token).
			s.emitDegraded(p.Provider+":"+p.Name, fmt.Sprintf("account not authenticated: %s", account.Status))
			continue
		}
		var snapshot model.UsageSnapshot
		if profileTUIBusy(p.Provider, p.Name) {
			// Never race an interactive Codex TUI with a live app-server probe.
			snapshot = profile.PeekUsageSnapshot(p.Provider, p.Name)
		} else {
			snapshot = profile.RefreshUsageSnapshotContext(ctx, p.Provider, p.Name)
		}
		// RATE_LIMITED is an accurate reading, not a failed one: the provider
		// told us the account is blocked. Alerting on it is the monitor's job.
		switch snapshot.Status {
		case model.UsageLive, model.UsageCached, model.UsageRateLimited:
		case model.UsageEstimated:
			// Honest deferred quota while a Codex TUI owns the profile home,
			// or rollout evidence from the live session, must not look degraded.
			if len(snapshot.Windows) > 0 {
				break
			}
			fallthrough
		default:
			s.emitDegraded(p.Provider+":"+p.Name, fmt.Sprintf("quota read is %s", snapshot.Status))
			continue
		}
		s.clearDegraded(p.Provider + ":" + p.Name)
		view := quota.BuildQuotaView(snapshot, account.Email, account.Plan)
		// Match GetAccountInfo: Codex scopes by chatgpt_account_id, others by email.
		identity := account.Email
		if p.Provider == "codex" && strings.TrimSpace(account.ExternalAccountID) != "" {
			identity = account.ExternalAccountID
		}
		acc := ProviderAccount{Provider: p.Provider, Profile: p.Name, DisplayName: account.Email, Authenticated: true, QuotaView: &view, QuotaRemaining: 0, QuotaTotal: 1, LastChecked: time.Now()}
		if view.AvailReasons.RateLimited || snapshot.Status == model.UsageRateLimited {
			acc.RateLimited = true
			acc.Available = false
		}
		if scope, scopeErr := profile.AccountScope(p.Provider, p.Name, identity); scopeErr == nil {
			acc.Scope = scope
		}
		if remaining, ok := view.BestGroupRemaining(); ok {
			acc.QuotaRemaining = remaining / 100
		} else if !acc.RateLimited {
			// No trustworthy remaining percentage is not zero remaining.
			// Suppress aggregate fallback notifications and automatic handoff.
			acc.QuotaTotal = 0
		}
		if action := s.monitor.CheckAccount(acc); action != nil &&
			action.AffectedRuntimeID != "" && action.Kind == string(events.EventQuotaExhausted) {
			reason := ContinuityQuotaExhausted
			if acc.RateLimited {
				reason = ContinuityRateLimited
			}
			s.mu.Lock()
			handler := s.continuity
			s.mu.Unlock()
			if handler != nil {
				handler(ctx, action.AffectedRuntimeID, reason)
			}
		}
	}
}

func profileTUIBusy(provider, name string) bool {
	if !strings.EqualFold(strings.TrimSpace(provider), "codex") {
		return false
	}
	home, err := config.ProfileHome("codex", name)
	if err != nil {
		return false
	}
	return codex.IsTUILocked(home)
}

func (s *QuotaMonitorService) emitDegraded(key, reason string) {
	s.mu.Lock()
	if s.degraded[key] {
		s.mu.Unlock()
		return
	}
	s.degraded[key] = true
	s.mu.Unlock()
	s.bus.Publish(events.NewEvent("", "", "", events.EventQuotaMonitorDegraded, reason, map[string]any{"reason": reason, "profile_key": key}))
}

func (s *QuotaMonitorService) clearDegraded(key string) {
	s.mu.Lock()
	was := s.degraded[key]
	if was {
		delete(s.degraded, key)
	}
	s.mu.Unlock()
	if was {
		s.bus.Publish(events.NewEvent("", "", "", events.EventQuotaMonitorRecovered, "quota monitoring recovered", map[string]any{"profile_key": key}))
	}
}
