package nexus

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/events"
	"github.com/IAPro-Community/nexus/internal/control/registry"
	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

type ContinuityReason string

const (
	ContinuityQuotaExhausted      ContinuityReason = "QUOTA_EXHAUSTED"
	ContinuityRateLimited         ContinuityReason = "RATE_LIMITED"
	ContinuityAuthLost            ContinuityReason = "AUTH_LOST"
	ContinuityResourceUnavailable ContinuityReason = "RESOURCE_UNAVAILABLE"
)

type ContinuityDecision struct {
	Reason           ContinuityReason
	SourceRuntimeID  string
	SourceProvider   string
	SourceProfile    string
	TargetProvider   string
	TargetProfile    string
	Mode             string
	CapacityEvidence string
	CreatedAt        time.Time
}

// ContinuityController is independent from Mission routing. It only replaces
// the execution resource of an existing supervised runtime.
type ContinuityController struct {
	nexus *Nexus
}

func NewContinuityController(n *Nexus) *ContinuityController {
	return &ContinuityController{nexus: n}
}

// ContinueRuntime prefers another account on the same provider. If no such
// resource is eligible, it falls back to a context handoff on another provider.
// The existing handoff service remains the authority for checkpoint, quiesce,
// resume, rollback and the no-two-writers invariant.
func (c *ContinuityController) ContinueRuntime(ctx context.Context, runtimeID string, reason ContinuityReason) (*registry.RuntimeSession, ContinuityDecision, error) {
	decision := ContinuityDecision{
		Reason:          reason,
		SourceRuntimeID: strings.TrimSpace(runtimeID),
		CreatedAt:       time.Now().UTC(),
	}
	if c == nil || c.nexus == nil {
		return nil, decision, fmt.Errorf("continuity controller is unavailable")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, decision, err
	}

	reg := c.nexus.runtimeRegistry()
	source, ok := reg.Get(decision.SourceRuntimeID)
	if !ok {
		return nil, decision, fmt.Errorf("source runtime %q not found", decision.SourceRuntimeID)
	}
	decision.SourceProvider = source.ProviderID
	decision.SourceProfile = source.ProfileID
	hops := continuityHopCount(source)
	if hops >= 4 {
		return nil, decision, fmt.Errorf("continuity hop limit reached for runtime %s", source.RuntimeID)
	}

	accounts, err := c.nexus.ListResourcesContext(ctx)
	if err != nil {
		return nil, decision, fmt.Errorf("discover continuity resources: %w", err)
	}
	target, ok := selectContinuityTarget(source, accounts)
	if !ok {
		return nil, decision, fmt.Errorf("no eligible continuity target for %s:%s", source.ProviderID, source.ProfileID)
	}
	decision.TargetProvider = target.Provider
	decision.TargetProfile = target.Profile
	if target.Eligibility != nil {
		decision.CapacityEvidence = target.Eligibility.CapacityEvidence
	}

	if source.Labels == nil {
		source.Labels = map[string]string{}
	}
	source.Labels["nexus.continuity_pending"] = "true"
	source.Labels["nexus.continuity_reason"] = string(reason)
	_ = reg.Register(source)

	bus := events.DefaultBus()
	bus.Publish(events.NewEvent(
		source.RuntimeID,
		source.ProviderID,
		source.ProfileID,
		events.EventQuotaFailoverRequested,
		fmt.Sprintf("Continuity requested for %s:%s -> %s:%s", source.ProviderID, source.ProfileID, target.Provider, target.Profile),
		map[string]any{
			"reason":           string(reason),
			"target_provider":  target.Provider,
			"target_profile":   target.Profile,
			"source_runtime_id": source.RuntimeID,
		},
	))

	runtimes := NewRuntimeApplicationService(reg, c.nexus.launcher, c.nexus.controlDrivers())
	var next *registry.RuntimeSession
	if strings.EqualFold(target.Provider, source.ProviderID) {
		decision.Mode = "ACCOUNT_HANDOFF"
		next, err = runtimes.AccountHandoff(ctx, source.RuntimeID, target.Provider+":"+target.Profile)
	} else {
		decision.Mode = "CONTEXT_HANDOFF"
		next, err = runtimes.ContextHandoff(ctx, source.RuntimeID, target.Provider, target.Profile)
	}
	if err != nil {
		source.Labels["nexus.continuity_pending"] = "false"
		_ = reg.Register(source)
		bus.Publish(events.NewEvent(
			source.RuntimeID,
			source.ProviderID,
			source.ProfileID,
			events.EventQuotaFailoverFailed,
			fmt.Sprintf("Continuity failed for %s:%s: %v", source.ProviderID, source.ProfileID, err),
			map[string]any{
				"reason":          string(reason),
				"target_provider": target.Provider,
				"target_profile":  target.Profile,
			},
		))
		return nil, decision, err
	}

	if next.Labels == nil {
		next.Labels = map[string]string{}
	}
	next.Labels["nexus.continuity_pending"] = "false"
	next.Labels["nexus.continuity_reason"] = string(reason)
	next.Labels["nexus.continuity_replaces_runtime"] = source.RuntimeID
	next.Labels["nexus.continuity_hops"] = strconv.Itoa(hops + 1)
	next.Labels["nexus.continuity_excluded"] = continuityExcludedLabel(source)
	if decision.Mode == "ACCOUNT_HANDOFF" {
		next.Labels["continuity"] = "ACCOUNT_HANDOFF"
	} else {
		next.Labels["continuity"] = "CONTEXT_HANDOFF"
	}
	_ = reg.Register(*next)

	if source.AgentID != "" {
		if persistErr := c.nexus.persistInteractiveContinuityReplacement(ctx, source, next); persistErr != nil {
			return nil, decision, fmt.Errorf("persist continuity replacement: %w", persistErr)
		}
		c.nexus.notifyRuntimeChanged(source.AgentID, source.RuntimeID, next.RuntimeID, next.ProviderID, next.ProfileID, next.Labels["continuity"])
		c.nexus.notifyContinuity(source.AgentID, next.Labels["continuity"])
	}

	bus.Publish(events.NewEventWithCorrelation(
		next.LineageID,
		next.RuntimeID,
		next.ProviderID,
		next.ProfileID,
		events.EventRuntimeReplaced,
		fmt.Sprintf("Runtime replaced %s -> %s", source.RuntimeID, next.RuntimeID),
		map[string]any{
			"old_runtime_id":  source.RuntimeID,
			"new_runtime_id":  next.RuntimeID,
			"reason":          string(reason),
			"continuity_mode": decision.Mode,
			"provider":        next.ProviderID,
			"profile":         next.ProfileID,
			"agent_id":        source.AgentID,
		},
	))
	bus.Publish(events.NewEvent(
		next.RuntimeID,
		next.ProviderID,
		next.ProfileID,
		events.EventQuotaFailoverCompleted,
		fmt.Sprintf("Continuity completed on %s:%s", next.ProviderID, next.ProfileID),
		map[string]any{
			"source_runtime_id": source.RuntimeID,
			"reason":            string(reason),
			"mode":              decision.Mode,
		},
	))

	return next, decision, nil
}

func selectContinuityTarget(source registry.RuntimeSession, accounts []ProviderAccount) (ProviderAccount, bool) {
	excluded := continuityExcludedResources(source)
	candidates := make([]ProviderAccount, 0, len(accounts))
	for _, account := range accounts {
		key := strings.ToLower(strings.TrimSpace(account.Provider)) + ":" + strings.ToLower(strings.TrimSpace(account.Profile))
		if _, blocked := excluded[key]; blocked {
			continue
		}
		// Multiple profiles may resolve to the same upstream capacity. Moving
		// from codex:A to codex:B cannot recover a shared exhausted quota.
		if source.AccountScope.Verifiable() && account.Scope.Verifiable() &&
			source.AccountScope.Key() == account.Scope.Key() {
			continue
		}
		candidates = append(candidates, account)
	}

	pick := func(pool []ProviderAccount) (ProviderAccount, bool) {
		if len(pool) == 0 {
			return ProviderAccount{}, false
		}
		rec := RecommendResources(pool, TaskRequirements{TaskKind: "coding"}, PolicyBalanced)
		if rec.Recommended == nil {
			return ProviderAccount{}, false
		}
		return rec.Recommended.Account, true
	}

	sameProvider := make([]ProviderAccount, 0, len(candidates))
	for _, account := range candidates {
		if strings.EqualFold(account.Provider, source.ProviderID) {
			sameProvider = append(sameProvider, account)
		}
	}
	if target, ok := pick(sameProvider); ok {
		return target, true
	}
	return pick(candidates)
}

func continuityHopCount(session registry.RuntimeSession) int {
	if session.Labels == nil {
		return 0
	}
	value, _ := strconv.Atoi(strings.TrimSpace(session.Labels["nexus.continuity_hops"]))
	if value < 0 {
		return 0
	}
	return value
}

func continuityExcludedResources(session registry.RuntimeSession) map[string]struct{} {
	out := map[string]struct{}{}
	if session.Labels != nil {
		for _, item := range strings.Split(session.Labels["nexus.continuity_excluded"], ",") {
			item = strings.ToLower(strings.TrimSpace(item))
			if item != "" {
				out[item] = struct{}{}
			}
		}
	}
	current := strings.ToLower(strings.TrimSpace(session.ProviderID)) + ":" + strings.ToLower(strings.TrimSpace(session.ProfileID))
	if current != ":" {
		out[current] = struct{}{}
	}
	return out
}

func continuityExcludedLabel(session registry.RuntimeSession) string {
	items := make([]string, 0)
	for item := range continuityExcludedResources(session) {
		items = append(items, item)
	}
	sort.Strings(items)
	return strings.Join(items, ",")
}

func (n *Nexus) persistInteractiveContinuityReplacement(ctx context.Context, source registry.RuntimeSession, target *registry.RuntimeSession) error {
	if n == nil || target == nil || source.AgentID == "" {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	st, err := n.OpenProject()
	if err != nil {
		return err
	}
	agent, err := st.GetAgent(source.AgentID, "")
	if err != nil {
		return err
	}
	agent, err = n.configureInteractiveLead(st, agent, target.ProviderID, target.ProfileID)
	if err != nil {
		return err
	}
	if oldGen, genErr := st.GenerationByRuntimeID(source.RuntimeID); genErr == nil && oldGen.StoppedAt == nil {
		_ = st.StopGeneration(oldGen.ID, time.Now().UTC())
	}
	continuity := strings.TrimSpace(target.Labels["continuity"])
	if continuity == "" {
		continuity = store.ContinuityNewSession
	}
	if _, genErr := st.GenerationByRuntimeID(target.RuntimeID); genErr != nil {
		if _, addErr := st.AddGeneration(store.RuntimeGeneration{
			AgentID:         agent.ID,
			RevisionID:      agent.CurrentRevisionID,
			RuntimeID:       target.RuntimeID,
			Provider:        target.ProviderID,
			Profile:         target.ProfileID,
			ProviderSession: target.ProviderSessionID,
			Continuity:      continuity,
			StartedAt:       time.Now().UTC(),
			State:           "RUNNING",
		}); addErr != nil {
			return addErr
		}
	}
	agent.Status = store.AgentWorking
	agent.ContinuityStatus = continuity
	now := time.Now().UTC()
	agent.LastStartedAt = &now
	return st.UpdateAgent(agent)
}

func ContinuityReasonForFailure(failure model.Failure) (ContinuityReason, bool) {
	switch failure.Kind {
	case model.FailureQuota:
		return ContinuityQuotaExhausted, true
	case model.FailureRateLimit:
		return ContinuityRateLimited, true
	case model.FailureAuth:
		return ContinuityAuthLost, true
	case model.FailureProvider, model.FailureNetwork:
		return ContinuityResourceUnavailable, true
	default:
		return "", false
	}
}

func (n *Nexus) wireQuotaContinuity(service *QuotaMonitorService) {
	if n == nil || service == nil {
		return
	}
	service.SetContinuityHandler(func(ctx context.Context, runtimeID string, reason ContinuityReason) {
		_, _, _ = NewContinuityController(n).ContinueRuntime(ctx, runtimeID, reason)
	})
}
