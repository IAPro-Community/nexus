package nexus

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/quota"
	"github.com/IAPro-Community/nexus/internal/nexus/runner"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
	"github.com/IAPro-Community/nexus/internal/profile"
)

const hudProjectionCacheTTL = 15 * time.Second

const (
	HUDPrimaryAlert    = "ALERT"
	HUDPrimaryMission  = "MISSION"
	HUDPrimaryAgent    = "AGENT"
	HUDPrimaryResource = "RESOURCE"
	HUDPrimaryCapacity = "CAPACITY"
	HUDPrimaryEmpty    = "EMPTY"
)

type HUDWindow struct {
	Group            string     `json:"group,omitempty"`
	Kind             string     `json:"kind"`
	Label            string     `json:"label,omitempty"`
	RemainingPercent *float64   `json:"remaining_percent,omitempty"`
	ResetDescription string     `json:"reset_description,omitempty"`
	ResetAt          *time.Time `json:"reset_at,omitempty"`
	Status           string     `json:"status"`
}

type HUDProvider struct {
	Provider       string      `json:"provider"`
	Profile        string      `json:"profile"`
	DisplayName    string      `json:"display_name,omitempty"`
	Health         string      `json:"health"`
	Availability   string      `json:"availability"`
	CapacityState  string      `json:"capacity_state"`
	Source         string      `json:"source,omitempty"`
	Freshness      string      `json:"freshness"`
	LastUpdated    *time.Time  `json:"last_updated,omitempty"`
	RateLimited    bool        `json:"rate_limited"`
	LimitingWindow *HUDWindow  `json:"limiting_window,omitempty"`
	Windows        []HUDWindow `json:"windows"`
}

type HUDRecommendation struct {
	Provider     string   `json:"provider"`
	Profile      string   `json:"profile"`
	DisplayName  string   `json:"display_name,omitempty"`
	Score        float64  `json:"score"`
	Confidence   string   `json:"confidence"`
	Explanation  string   `json:"explanation"`
	Alternatives []string `json:"alternatives,omitempty"`
	Context      string   `json:"context"`
}

type HUDAgent struct {
	ID        string     `json:"id"`
	ProjectID string     `json:"project_id"`
	Name      string     `json:"name"`
	Role      string     `json:"role,omitempty"`
	Status    string     `json:"status"`
	Provider  string     `json:"provider,omitempty"`
	Profile   string     `json:"profile,omitempty"`
	Model     string     `json:"model,omitempty"`
	RuntimeID string     `json:"runtime_id,omitempty"`
	StartedAt *time.Time `json:"started_at,omitempty"`
}

type HUDMission struct {
	RunID              string    `json:"run_id"`
	ProjectID          string    `json:"project_id"`
	State              string    `json:"state"`
	CurrentStage       string    `json:"current_stage"`
	CurrentTaskID      string    `json:"current_task_id,omitempty"`
	CurrentTaskTitle   string    `json:"current_task_title,omitempty"`
	Provider           string    `json:"provider,omitempty"`
	Profile            string    `json:"profile,omitempty"`
	CompletedPackages  int       `json:"completed_packages"`
	TotalPackages      int       `json:"total_packages"`
	NeedsUser          bool      `json:"needs_user"`
	InterventionReason string    `json:"intervention_reason,omitempty"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type HUDAlert struct {
	Fingerprint      string     `json:"fingerprint"`
	Category         string     `json:"category"`
	Severity         string     `json:"severity"`
	Provider         string     `json:"provider,omitempty"`
	Profile          string     `json:"profile,omitempty"`
	Reason           string     `json:"reason,omitempty"`
	RemainingPercent *float64   `json:"remaining_percent,omitempty"`
	Acked            bool       `json:"acked"`
	SnoozedUntil     *time.Time `json:"snoozed_until,omitempty"`
	LastObservedAt   time.Time  `json:"last_observed_at"`
}

type HUDCostKindTotal struct {
	Kind     string `json:"kind"`
	MicroUSD int64  `json:"microusd"`
	Count    int    `json:"count"`
}

type HUDCostSummary struct {
	Evidence                CostEvidenceState  `json:"evidence"`
	TokenEvidence           CostEvidenceState  `json:"token_evidence"`
	KnownMicroUSD           int64              `json:"known_microusd"`
	UnknownObservationCount int64              `json:"unknown_observation_count"`
	BudgetLevel             CostBudgetLevel    `json:"budget_level"`
	ByKind                  []HUDCostKindTotal `json:"by_kind"`
	GeneratedAt             time.Time          `json:"generated_at,omitempty"`
}

type HUDPrimary struct {
	Kind string `json:"kind"`
	ID   string `json:"id,omitempty"`
}

type HUDSystemState struct {
	Capacity       string `json:"capacity"`
	Resources      string `json:"resources"`
	Agents         string `json:"agents"`
	Missions       string `json:"missions"`
	Cost           string `json:"cost"`
	CacheTTLMillis int64  `json:"cache_ttl_millis"`
}

type HUDState struct {
	GeneratedAt      time.Time             `json:"generated_at"`
	ProjectID        string                `json:"project_id,omitempty"`
	Providers        []HUDProvider         `json:"providers"`
	Recommendation   *HUDRecommendation    `json:"recommendation,omitempty"`
	Agents           []HUDAgent            `json:"agents"`
	Missions         []HUDMission          `json:"missions"`
	Alerts           []HUDAlert            `json:"alerts"`
	RuntimeAttention []HUDRuntimeAttention `json:"runtime_attention"`
	Cost             *HUDCostSummary       `json:"cost,omitempty"`
	Primary          HUDPrimary            `json:"primary"`
	System           HUDSystemState        `json:"system"`
}

type HUDProjectionInput struct {
	GeneratedAt      time.Time
	ProjectID        string
	Capacity         CapacityStatus
	Accounts         []ProviderAccount
	Recommendation   *HUDRecommendation
	Agents           []store.Agent
	AgentDetails     map[string]AgentDetailView
	Runs             []*runner.MissionRun
	Cost             *CostSummary
	CostObservations []store.CostObservation
	System           HUDSystemState
}

type hudCacheEntry struct {
	state     HUDState
	expiresAt time.Time
}

// HUDApplicationService owns only assembly/cache of the read-only HUD
// projection. It delegates every fact and decision to existing Nexus services.
type HUDApplicationService struct {
	nexus    *Nexus
	projects *ProjectApplicationService
	agents   *AgentApplicationService
	runs     *RunApplicationService

	mu    sync.Mutex
	cache map[string]hudCacheEntry
}

func NewHUDApplicationService(n *Nexus) *HUDApplicationService {
	return &HUDApplicationService{
		nexus:    n,
		projects: NewProjectApplicationService(n),
		agents:   NewAgentApplicationService(n),
		runs:     NewRunApplicationService(n),
		cache:    make(map[string]hudCacheEntry),
	}
}

func (s *HUDApplicationService) State(ctx context.Context, projectID string) (HUDState, error) {
	if err := ctx.Err(); err != nil {
		return HUDState{}, err
	}
	projectID = strings.TrimSpace(projectID)
	now := time.Now().UTC()

	s.mu.Lock()
	if cached, ok := s.cache[projectID]; ok && now.Before(cached.expiresAt) {
		state := cached.state
		s.mu.Unlock()
		return state, nil
	}
	s.mu.Unlock()

	st, err := s.nexus.OpenProject()
	if err != nil {
		return HUDState{}, err
	}

	system := HUDSystemState{
		Capacity:       "AVAILABLE",
		Resources:      "AVAILABLE",
		Agents:         "AVAILABLE",
		Missions:       "AVAILABLE",
		Cost:           "AVAILABLE",
		CacheTTLMillis: hudProjectionCacheTTL.Milliseconds(),
	}

	capacity, err := NewCapacityApplicationService(st).Status(ctx)
	if err != nil {
		return HUDState{}, err
	}

	// Do not call ResourceApplicationService.List here: that path performs
	// provider detection/effective-capability probes. HUD refresh must consume
	// already-materialized state and never create a second polling loop.
	system.Resources = "SNAPSHOT_ONLY"
	recommendation := loadHUDRecommendation(st, capacity)
	if recommendation != nil {
		system.Resources = "CANONICAL_EVENT"
	}
	accounts, snapshotErr := loadHUDQuotaSnapshots()
	if snapshotErr != nil {
		system.Resources = "SNAPSHOT_UNAVAILABLE"
	}

	var agents []store.Agent
	agentDetails := map[string]AgentDetailView{}
	if projectID != "" {
		agents, err = s.agents.List(ctx, projectID)
	} else {
		projects, projectErr := s.projects.List(ctx)
		if projectErr != nil {
			err = projectErr
		} else {
			for _, project := range projects {
				projectAgents, listErr := s.agents.List(ctx, project.ID)
				if listErr != nil {
					err = listErr
					break
				}
				agents = append(agents, projectAgents...)
			}
		}
	}
	if err != nil {
		system.Agents = "UNAVAILABLE"
		agents = nil
	} else {
		for _, agent := range agents {
			detail, detailErr := s.agents.Detail(ctx, agent.ID)
			if detailErr == nil {
				agentDetails[agent.ID] = detail
			}
		}
	}

	runs, runsErr := s.runs.List(ctx)
	if runsErr != nil {
		system.Missions = "UNAVAILABLE"
		runs = nil
	}

	costSvc := NewCostApplicationService(st)
	var cost *CostSummary
	var costObservations []store.CostObservation
	if summary, costErr := costSvc.Summary(ctx, projectID, 7); costErr != nil {
		system.Cost = "UNAVAILABLE"
	} else {
		cost = &summary
		costObservations, _ = costSvc.Observations(ctx, store.CostObservationFilter{
			ProjectID: projectID,
			Since:     summary.Since,
			Limit:     1000,
		})
	}

	state := BuildHUDProjection(HUDProjectionInput{
		GeneratedAt:      now,
		ProjectID:        projectID,
		Capacity:         capacity,
		Accounts:         accounts,
		Recommendation:   recommendation,
		Agents:           agents,
		AgentDetails:     agentDetails,
		Runs:             runs,
		Cost:             cost,
		CostObservations: costObservations,
		System:           system,
	})

	s.mu.Lock()
	s.cache[projectID] = hudCacheEntry{state: state, expiresAt: now.Add(hudProjectionCacheTTL)}
	s.mu.Unlock()
	return state, nil
}

func BuildHUDProjection(input HUDProjectionInput) HUDState {
	generatedAt := input.GeneratedAt
	if generatedAt.IsZero() {
		generatedAt = time.Now().UTC()
	}
	state := HUDState{
		GeneratedAt:      generatedAt,
		ProjectID:        strings.TrimSpace(input.ProjectID),
		Providers:        buildHUDProviders(input.Accounts, input.Capacity),
		Agents:           buildHUDAgents(input.Agents, input.AgentDetails),
		Missions:         buildHUDMissions(input.Runs, input.ProjectID),
		Alerts:           buildHUDAlerts(input.Capacity),
		RuntimeAttention: []HUDRuntimeAttention{},
		System:           input.System,
	}
	if state.System.CacheTTLMillis == 0 {
		state.System.CacheTTLMillis = hudProjectionCacheTTL.Milliseconds()
	}
	if input.Recommendation != nil {
		rec := *input.Recommendation
		rec.Alternatives = append([]string(nil), input.Recommendation.Alternatives...)
		state.Recommendation = &rec
	}
	if input.Cost != nil {
		state.Cost = buildHUDCost(*input.Cost, input.CostObservations)
	}
	state.Primary = chooseHUDPrimary(state)
	return state
}

func loadHUDRecommendation(st *store.Store, capacity CapacityStatus) *HUDRecommendation {
	if st == nil {
		return nil
	}
	active := make(map[string]struct{})
	for _, resource := range capacity.Resources {
		switch nonEmptyUpper(resource.State) {
		case "WARNING", "CRITICAL", "EXHAUSTED":
			if strings.TrimSpace(resource.Fingerprint) != "" && !resource.Acked &&
				(resource.SnoozedUntil == nil || resource.SnoozedUntil.Before(time.Now().UTC())) {
				active[resource.Fingerprint] = struct{}{}
			}
		}
	}
	if len(active) == 0 {
		return nil
	}
	events, err := st.ListCapacityEvents(50)
	if err != nil {
		return nil
	}
	for _, event := range events {
		if _, ok := active[event.Fingerprint]; !ok {
			continue
		}
		if strings.TrimSpace(event.PayloadJSON) == "" {
			continue
		}
		var payload struct {
			Provider    string  `json:"recommended_provider"`
			Profile     string  `json:"recommended_profile"`
			DisplayName string  `json:"recommended_display_name"`
			Score       float64 `json:"recommendation_score"`
			Reason      string  `json:"recommendation_reason"`
		}
		if json.Unmarshal([]byte(event.PayloadJSON), &payload) != nil {
			continue
		}
		if strings.TrimSpace(payload.Provider) == "" {
			continue
		}
		return &HUDRecommendation{
			Provider:    payload.Provider,
			Profile:     payload.Profile,
			DisplayName: payload.DisplayName,
			Score:       payload.Score,
			Confidence:  "UNKNOWN",
			Explanation: payload.Reason,
			Context:     "CAPACITY_ALERT",
		}
	}
	return nil
}

func loadHUDQuotaSnapshots() ([]ProviderAccount, error) {
	profiles, err := profile.List()
	if err != nil {
		return nil, err
	}
	accounts := make([]ProviderAccount, 0, len(profiles))
	for _, p := range profiles {
		snapshot := profile.PeekUsageSnapshot(p.Provider, p.Name)
		view := quota.BuildQuotaView(snapshot, snapshot.Account, "")
		accounts = append(accounts, ProviderAccount{
			ID:           p.Provider + ":" + p.Name,
			Provider:     p.Provider,
			Profile:      p.Name,
			DisplayName:  p.Provider + " (" + p.Name + ")",
			Health:       "unknown",
			Available:    view.IsAvailable(),
			AvailReasons: &view.AvailReasons,
			QuotaView:    &view,
			RateLimited:  strings.EqualFold(view.Status, "RATE_LIMITED"),
			NetworkError: strings.EqualFold(view.Status, "ERROR"),
			LastChecked:  view.FetchedAt,
		})
	}
	return accounts, nil
}

func buildHUDProviders(accounts []ProviderAccount, capacity CapacityStatus) []HUDProvider {
	byKey := make(map[string]*HUDProvider)
	keyOf := func(provider, profile string) string {
		return strings.ToLower(strings.TrimSpace(provider)) + "\x00" + strings.ToLower(strings.TrimSpace(profile))
	}
	for _, resource := range capacity.Resources {
		key := keyOf(resource.Provider, resource.Profile)
		item := byKey[key]
		if item == nil {
			item = &HUDProvider{
				Provider:      resource.Provider,
				Profile:       resource.Profile,
				Health:        "UNKNOWN",
				Availability:  "UNKNOWN",
				CapacityState: nonEmptyUpper(resource.State),
				Source:        resource.Source,
				Freshness:     nonEmptyUpper(resource.ObservationStatus),
				Windows:       []HUDWindow{},
			}
			byKey[key] = item
		}
		item.CapacityState = higherCapacityState(item.CapacityState, resource.State)
		if strings.TrimSpace(resource.Source) != "" {
			item.Source = resource.Source
		}
		if strings.TrimSpace(resource.ObservationStatus) != "" {
			item.Freshness = nonEmptyUpper(resource.ObservationStatus)
		}
		remaining := cloneFloat(resource.RemainingPercent)
		item.Windows = append(item.Windows, HUDWindow{
			Group:            resource.QuotaGroup,
			Kind:             resource.WindowKind,
			RemainingPercent: remaining,
			ResetDescription: resource.ResetDescription,
			Status:           nonEmptyUpper(resource.State),
		})
		if !resource.LastObservedAt.IsZero() && (item.LastUpdated == nil || resource.LastObservedAt.After(*item.LastUpdated)) {
			observed := resource.LastObservedAt
			item.LastUpdated = &observed
		}
	}

	for _, account := range accounts {
		key := keyOf(account.Provider, account.Profile)
		item := byKey[key]
		if item == nil {
			item = &HUDProvider{
				Provider:      account.Provider,
				Profile:       account.Profile,
				CapacityState: "UNKNOWN",
				Windows:       []HUDWindow{},
			}
			byKey[key] = item
		}
		item.DisplayName = account.DisplayName
		item.Health = nonEmptyUpper(account.Health)
		item.RateLimited = account.RateLimited

		if account.QuotaView == nil {
			item.Availability = availabilityFromAccount(account, false)
			if item.Freshness == "" {
				item.Freshness = "UNKNOWN"
			}
			continue
		}

		qv := account.QuotaView
		item.Source = qv.Source
		item.Freshness = nonEmptyUpper(qv.Status)
		item.Availability = availabilityFromAccount(account, qv.AvailReasons.UnknownQuota)
		if !qv.FetchedAt.IsZero() {
			fetched := qv.FetchedAt
			item.LastUpdated = &fetched
		}
		item.Windows = item.Windows[:0]
		for _, group := range qv.ModelGroups {
			for _, window := range group.Windows {
				remaining := window.Remaining
				var remainingPtr *float64
				if strings.ToLower(window.Kind) != "unknown" && nonEmptyUpper(window.Status) != "UNKNOWN" {
					remainingPtr = &remaining
				}
				item.Windows = append(item.Windows, HUDWindow{
					Group:            group.Name,
					Kind:             window.Kind,
					Label:            window.Label,
					RemainingPercent: remainingPtr,
					ResetDescription: window.ResetDesc,
					ResetAt:          window.ResetTime,
					Status:           nonEmptyUpper(window.Status),
				})
			}
		}
		if pct, kind := qv.Bottleneck(); kind != "" {
			limiting := HUDWindow{Kind: kind, RemainingPercent: &pct, Status: nonEmptyUpper(qv.Status)}
			for _, window := range item.Windows {
				if window.Kind != kind || window.RemainingPercent == nil {
					continue
				}
				if *window.RemainingPercent == pct {
					copyWindow := window
					limiting = copyWindow
					break
				}
			}
			item.LimitingWindow = &limiting
		}
	}

	out := make([]HUDProvider, 0, len(byKey))
	for _, item := range byKey {
		if item.Health == "" {
			item.Health = "UNKNOWN"
		}
		if item.Availability == "" {
			item.Availability = "UNKNOWN"
		}
		if item.CapacityState == "" {
			item.CapacityState = "UNKNOWN"
		}
		if item.Freshness == "" {
			item.Freshness = "UNKNOWN"
		}
		sort.SliceStable(item.Windows, func(i, j int) bool {
			if item.Windows[i].Group != item.Windows[j].Group {
				return item.Windows[i].Group < item.Windows[j].Group
			}
			return item.Windows[i].Kind < item.Windows[j].Kind
		})
		out = append(out, *item)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Provider != out[j].Provider {
			return out[i].Provider < out[j].Provider
		}
		return out[i].Profile < out[j].Profile
	})
	return out
}

func buildHUDAgents(agents []store.Agent, details map[string]AgentDetailView) []HUDAgent {
	out := make([]HUDAgent, 0, len(agents))
	for _, agent := range agents {
		item := HUDAgent{
			ID:        agent.ID,
			ProjectID: agent.ProjectID,
			Name:      agent.Name,
			Role:      agent.Role,
			Status:    nonEmptyUpper(agent.Status),
		}
		if detail, ok := details[agent.ID]; ok {
			if len(detail.Generations) > 0 {
				generation := detail.Generations[0]
				item.Provider = generation.Provider
				item.Profile = generation.Profile
				item.RuntimeID = generation.RuntimeID
				started := generation.StartedAt
				item.StartedAt = &started
			}
			for _, revision := range detail.Revisions {
				if revision.ID != agent.CurrentRevisionID {
					continue
				}
				if cfg, err := ParseAgentConfig(revision.Config); err == nil {
					item.Model = cfg.Model
					if item.Provider == "" {
						item.Provider = cfg.Provider
					}
					if item.Profile == "" {
						item.Profile = cfg.Profile
					}
				}
				break
			}
		}
		out = append(out, item)
	}
	return out
}

func buildHUDMissions(runs []*runner.MissionRun, projectID string) []HUDMission {
	projectID = strings.TrimSpace(projectID)
	out := make([]HUDMission, 0, len(runs))
	for _, run := range runs {
		if run == nil || (projectID != "" && run.ProjectID != projectID) {
			continue
		}
		projection := BuildMissionStageProjection(run)
		item := HUDMission{
			RunID:            run.ID,
			ProjectID:        run.ProjectID,
			State:            nonEmptyUpper(string(run.State)),
			CurrentStage:     projection.CurrentStage,
			CurrentTaskID:    projection.TaskID,
			CurrentTaskTitle: projection.TaskTitle,
			Provider:         projection.Provider,
			Profile:          projection.Profile,
			TotalPackages:    len(run.PackageRuns),
			UpdatedAt:        run.UpdatedAt,
		}
		for _, pkg := range run.PackageRuns {
			switch pkg.State {
			case runner.StateVerified, runner.StateCompletedVerified:
				item.CompletedPackages++
			}
		}
		if run.NeedsHuman != nil && !run.NeedsHuman.Resolved {
			item.NeedsUser = true
			item.InterventionReason = run.NeedsHuman.ReasonCode
		}
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UpdatedAt.After(out[j].UpdatedAt) })
	return out
}

func buildHUDAlerts(capacity CapacityStatus) []HUDAlert {
	out := make([]HUDAlert, 0)
	now := time.Now().UTC()
	for _, resource := range capacity.Resources {
		severity := capacitySeverity(resource.State)
		if severity == "" || resource.Acked {
			continue
		}
		if resource.SnoozedUntil != nil && resource.SnoozedUntil.After(now) {
			continue
		}
		out = append(out, HUDAlert{
			Fingerprint:      resource.Fingerprint,
			Category:         "CAPACITY",
			Severity:         severity,
			Provider:         resource.Provider,
			Profile:          resource.Profile,
			Reason:           resource.Reason,
			RemainingPercent: cloneFloat(resource.RemainingPercent),
			Acked:            resource.Acked,
			SnoozedUntil:     resource.SnoozedUntil,
			LastObservedAt:   resource.LastObservedAt,
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		left, right := severityRank(out[i].Severity), severityRank(out[j].Severity)
		if left != right {
			return left > right
		}
		return out[i].LastObservedAt.After(out[j].LastObservedAt)
	})
	return out
}

func buildHUDCost(summary CostSummary, observations []store.CostObservation) *HUDCostSummary {
	byKind := map[string]*HUDCostKindTotal{}
	for _, observation := range observations {
		if observation.CostMicroUSD == nil {
			continue
		}
		kind := nonEmptyUpper(observation.CostKind)
		total := byKind[kind]
		if total == nil {
			total = &HUDCostKindTotal{Kind: kind}
			byKind[kind] = total
		}
		total.MicroUSD += *observation.CostMicroUSD
		total.Count++
	}
	kinds := make([]HUDCostKindTotal, 0, len(byKind))
	for _, total := range byKind {
		kinds = append(kinds, *total)
	}
	sort.Slice(kinds, func(i, j int) bool { return kinds[i].Kind < kinds[j].Kind })
	return &HUDCostSummary{
		Evidence:                summary.CostEvidence,
		TokenEvidence:           summary.TokenEvidence,
		KnownMicroUSD:           summary.Aggregate.CostMicroUSD,
		UnknownObservationCount: summary.Aggregate.UnknownCostObservations,
		BudgetLevel:             summary.Budget.Level,
		ByKind:                  kinds,
		GeneratedAt:             summary.GeneratedAt,
	}
}

func chooseHUDPrimary(state HUDState) HUDPrimary {
	for _, alert := range state.Alerts {
		if alert.Severity == "CRITICAL" {
			return HUDPrimary{Kind: HUDPrimaryAlert, ID: alert.Fingerprint}
		}
	}
	for _, mission := range state.Missions {
		if mission.NeedsUser {
			return HUDPrimary{Kind: HUDPrimaryMission, ID: mission.RunID}
		}
	}
	for _, mission := range state.Missions {
		if mission.State == string(runner.StateBlockedNeedsUser) {
			return HUDPrimary{Kind: HUDPrimaryMission, ID: mission.RunID}
		}
	}
	for _, agent := range state.Agents {
		if agent.Status == "WORKING" {
			return HUDPrimary{Kind: HUDPrimaryAgent, ID: agent.ID}
		}
	}
	if state.Recommendation != nil {
		return HUDPrimary{Kind: HUDPrimaryResource, ID: strings.Trim(state.Recommendation.Provider+":"+state.Recommendation.Profile, ":")}
	}
	if len(state.Providers) > 0 {
		return HUDPrimary{Kind: HUDPrimaryCapacity}
	}
	return HUDPrimary{Kind: HUDPrimaryEmpty}
}

func availabilityFromAccount(account ProviderAccount, unknownQuota bool) string {
	if unknownQuota {
		return "UNKNOWN"
	}
	health := strings.ToLower(strings.TrimSpace(account.Health))
	if health == "" || health == "unknown" {
		return "UNKNOWN"
	}
	if account.Available {
		return "AVAILABLE"
	}
	return "UNAVAILABLE"
}

func higherCapacityState(left, right string) string {
	if capacityStateRank(right) > capacityStateRank(left) {
		return nonEmptyUpper(right)
	}
	return nonEmptyUpper(left)
}

func capacityStateRank(state string) int {
	switch nonEmptyUpper(state) {
	case "EXHAUSTED":
		return 5
	case "CRITICAL":
		return 4
	case "WARNING":
		return 3
	case "RECOVERED":
		return 2
	case "HEALTHY":
		return 1
	default:
		return 0
	}
}

func capacitySeverity(state string) string {
	switch nonEmptyUpper(state) {
	case "EXHAUSTED", "CRITICAL":
		return "CRITICAL"
	case "WARNING":
		return "WARNING"
	default:
		return ""
	}
}

func severityRank(severity string) int {
	switch nonEmptyUpper(severity) {
	case "CRITICAL":
		return 3
	case "WARNING":
		return 2
	case "INFO":
		return 1
	default:
		return 0
	}
}

func nonEmptyUpper(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	if value == "" {
		return "UNKNOWN"
	}
	return value
}

func cloneFloat(value *float64) *float64 {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}
