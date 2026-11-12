package nexus

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/events"
	"github.com/IAPro-Community/nexus/internal/control/registry"
	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/core/quota"
	"github.com/IAPro-Community/nexus/internal/profile"
)

// earlyQuotaCheckDelay lets the provider settle its own limit state for a
// moment before the evidence check: probing while the TUI is painting the
// limit banner races the rollout write that carries the authoritative state.
const earlyQuotaCheckDelay = 3 * time.Second

// wireQuotaPatternContinuity consumes fast quota-pattern events produced by
// supervised PTY hosts. The pattern alone never switches a session: this
// subscriber only runs the same evidence-confirmed continuity path the
// 1-minute monitor uses, just earlier. Non-confirmed matches are ignored and
// the monitor remains the authority.
func (n *Nexus) wireQuotaPatternContinuity() {
	if n == nil {
		return
	}
	channel, unsubscribe := events.DefaultBus().Subscribe("*")
	go func() {
		defer unsubscribe()
		for e := range channel {
			if e.Type != events.EventQuotaExhausted {
				continue
			}
			if e.Data == nil || e.Data["nexus.quota_trigger"] != "pty_pattern" {
				continue
			}
			runtimeID, _ := e.Data["affected_runtime_id"].(string)
			provider, _ := e.Data["provider"].(string)
			failureKind, _ := e.Data["failure_kind"].(string)
			if strings.TrimSpace(runtimeID) == "" {
				continue
			}
			go n.earlyQuotaContinuity(runtimeID, provider, failureKind)
		}
	}()
}

// earlyQuotaContinuity verifies quota evidence for a pattern-flagged runtime
// and, only when the evidence confirms exhaustion/rate-limit, runs the same
// continuity path as the scheduled monitor. Evidence is read from the
// runtime's own provider+profile in the registry, never from event payload
// trust.
func (n *Nexus) earlyQuotaContinuity(runtimeID, provider, failureKind string) {
	// Give the provider a moment to settle its limit state.
	timer := time.NewTimer(earlyQuotaCheckDelay)
	defer timer.Stop()
	<-timer.C

	reg := n.runtimeRegistry()
	if reg == nil {
		return
	}
	source, ok := reg.Get(runtimeID)
	if !ok {
		return
	}
	if provider != "" && !strings.EqualFold(source.ProviderID, provider) {
		return
	}
	if continuityPending(source) {
		// A continuity handoff is already in flight for this runtime.
		return
	}
	switch source.State {
	case registry.StateRunning, registry.StateWaiting, registry.StateStarting, registry.StateDetached:
	default:
		return
	}

	view := profile.GetQuotaView(providerOr(source.ProviderID, provider), source.ProfileID, "", "")
	if !quotaEvidenceConfirmsContinuity(view) {
		// Pattern without confirming evidence: the scheduled monitor remains
		// the authority. Publish nothing and change nothing.
		slog.Debug("quota pattern not confirmed by evidence; continuity skipped",
			"runtime_id", runtimeID, "provider", source.ProviderID, "profile", source.ProfileID)
		return
	}
	reason := ContinuityQuotaExhausted
	if strings.EqualFold(strings.TrimSpace(failureKind), string(model.FailureRateLimit)) {
		reason = ContinuityRateLimited
	}
	_, _, _ = NewContinuityController(n).ContinueRuntime(context.Background(), runtimeID, reason)
}

// quotaEvidenceZeroTolerance is how close verified remaining capacity must be
// to zero before a PTY pattern match may authorize continuity (0.5%).
const quotaEvidenceZeroTolerance = 0.005

// quotaEvidenceConfirmsContinuity reports whether an account's quota view is
// strong enough to authorize a continuity switch: an explicit rate-limit
// status, or verified evidence that the best model group is at (near) zero.
// Unknown/stale evidence must never authorize a switch on its own.
func quotaEvidenceConfirmsContinuity(view quota.QuotaView) bool {
	status := strings.ToUpper(strings.TrimSpace(view.Status))
	if status == string(model.UsageRateLimited) {
		return true
	}
	if status != string(model.UsageLive) && status != string(model.UsageEstimated) {
		return false
	}
	remaining, ok := view.BestGroupRemaining()
	if !ok {
		return false
	}
	return remaining <= quotaEvidenceZeroTolerance
}

// providerOr keeps the runtime's own provider as the evidence authority.
func providerOr(runtimeProvider, eventProvider string) string {
	if strings.TrimSpace(runtimeProvider) != "" {
		return runtimeProvider
	}
	return eventProvider
}

// continuityPending reports whether a continuity handoff is already in flight
// for the runtime, so concurrent triggers (scheduled monitor, PTY pattern,
// repeated events) cannot stack handoffs.
func continuityPending(session registry.RuntimeSession) bool {
	return session.Labels != nil && session.Labels["nexus.continuity_pending"] == "true"
}
