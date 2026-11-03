package nexus

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

type CostEvidenceState string

const (
	CostEvidenceEmpty           CostEvidenceState = "EMPTY"
	CostEvidenceAttributionOnly CostEvidenceState = "ATTRIBUTION_ONLY"
	CostEvidencePartial         CostEvidenceState = "PARTIAL"
	CostEvidenceMeasured        CostEvidenceState = "MEASURED"
)

type CostBudgetLevel string

const (
	CostBudgetDisabled  CostBudgetLevel = "DISABLED"
	CostBudgetUnknown   CostBudgetLevel = "UNKNOWN"
	CostBudgetOK        CostBudgetLevel = "OK"
	CostBudgetSoft      CostBudgetLevel = "SOFT"
	CostBudgetHard      CostBudgetLevel = "HARD"
	CostBudgetEmergency CostBudgetLevel = "EMERGENCY"
)

type CostBudgetAssessment struct {
	Level                CostBudgetLevel   `json:"level"`
	// KnownCostMicroUSD remains for clients consuming the legacy field, but
	// now contains ONLY provider-reported values (never API estimates).
	KnownCostMicroUSD        int64             `json:"known_cost_microusd"`
	ReportedCostMicroUSD     int64             `json:"reported_cost_microusd"`
	APIEquivalentMicroUSD    int64             `json:"api_equivalent_microusd"`
	InferredMicroUSD         int64             `json:"inferred_microusd"`
	UnclassifiedMicroUSD     int64             `json:"unclassified_microusd"`
	ReportedCostObservations int64             `json:"reported_cost_observations"`
	UnverifiedObservations   int64             `json:"unverified_observations"`
	Evidence                 CostEvidenceState `json:"evidence"`
	ActiveLimitMicroUSD  int64             `json:"active_limit_microusd,omitempty"`
	RequiresConfirmation bool              `json:"requires_confirmation"`
	Message              string            `json:"message"`
}

type CostAnomalyStatus string

const (
	CostAnomalyUnknown      CostAnomalyStatus = "UNKNOWN"
	CostAnomalyInsufficient CostAnomalyStatus = "INSUFFICIENT"
	CostAnomalyNormal       CostAnomalyStatus = "NORMAL"
	CostAnomalyTriggered    CostAnomalyStatus = "ANOMALY"
)

type CostAnomalyAssessment struct {
	Status                 CostAnomalyStatus `json:"status"`
	Triggered              bool              `json:"triggered"`
	Provider               string            `json:"provider,omitempty"`
	Profile                string            `json:"profile,omitempty"`
	CostKind               string            `json:"cost_kind,omitempty"`
	LatestCostMicroUSD     int64             `json:"latest_cost_microusd,omitempty"`
	BaselineMedianMicroUSD int64             `json:"baseline_median_microusd,omitempty"`
	ThresholdMicroUSD      int64             `json:"threshold_microusd,omitempty"`
	HistoricalSamples      int               `json:"historical_samples"`
	ObservedAt             time.Time         `json:"observed_at,omitempty"`
	Message                string            `json:"message"`
}

type CostSummary struct {
	Since         time.Time                     `json:"since"`
	GeneratedAt   time.Time                     `json:"generated_at"`
	ProjectID     string                        `json:"project_id,omitempty"`
	Aggregate     store.CostAggregate           `json:"aggregate"`
	ByProvider    []store.ProviderCostAggregate `json:"by_provider"`
	CostEvidence  CostEvidenceState             `json:"cost_evidence"`
	TokenEvidence CostEvidenceState             `json:"token_evidence"`
	Preferences   store.CostPreferences         `json:"preferences"`
	Budget        CostBudgetAssessment          `json:"budget"`
	Anomaly       CostAnomalyAssessment         `json:"anomaly"`
}

type CostApplicationService struct {
	st *store.Store
}

func NewCostApplicationService(st *store.Store) *CostApplicationService {
	return &CostApplicationService{st: st}
}

func (s *CostApplicationService) requireStore() (*store.Store, error) {
	if s == nil || s.st == nil {
		return nil, errors.New("cost ledger unavailable")
	}
	return s.st, nil
}

func (s *CostApplicationService) Summary(ctx context.Context, projectID string, days int) (CostSummary, error) {
	if err := ctx.Err(); err != nil {
		return CostSummary{}, err
	}
	st, err := s.requireStore()
	if err != nil {
		return CostSummary{}, err
	}
	prefs, err := st.GetCostPreferences()
	if err != nil {
		return CostSummary{}, err
	}
	if days <= 0 {
		days = prefs.WindowDays
	}
	if days > 365 {
		days = 365
	}
	now := time.Now().UTC()
	since := now.Add(-time.Duration(days) * 24 * time.Hour)
	filter := store.CostObservationFilter{
		ProjectID: strings.TrimSpace(projectID),
		Since:     since,
	}
	aggregate, err := st.SummarizeCostObservations(filter)
	if err != nil {
		return CostSummary{}, err
	}
	byProvider, err := st.SummarizeCostByProvider(filter)
	if err != nil {
		return CostSummary{}, err
	}
	costBreakdown, err := st.SummarizeCostEvidence(filter)
	if err != nil {
		return CostSummary{}, err
	}
	costEvidence := evidenceState(aggregate.ObservationCount, aggregate.KnownCostObservations)
	tokenEvidence := evidenceState(aggregate.ObservationCount, aggregate.KnownTokenObservations)
	observations, err := st.ListCostObservations(store.CostObservationFilter{
		ProjectID: filter.ProjectID,
		Since:     filter.Since,
		Limit:     1000,
	})
	if err != nil {
		return CostSummary{}, err
	}
	return CostSummary{
		Since:         since,
		GeneratedAt:   now,
		ProjectID:     filter.ProjectID,
		Aggregate:     aggregate,
		ByProvider:    byProvider,
		CostEvidence:  costEvidence,
		TokenEvidence: tokenEvidence,
		Preferences:   prefs,
		Budget:        evaluateCostBudget(costBreakdown, prefs),
		Anomaly:       evaluateCostAnomaly(observations, prefs),
	}, nil
}

// CostAnalyticsReport is a read-only breakdown over the existing cost ledger.
// Observations are not invoices, and API-equivalent estimates are never added
// to provider-reported amounts as if they were a single payable balance.
type CostAnalyticsReport struct {
	Since       time.Time `json:"since"`
	GeneratedAt time.Time `json:"generated_at"`
	Days        int       `json:"days"`
	ProjectID   string    `json:"project_id,omitempty"`
	Source      string    `json:"source"`
	store.CostAnalyticsView
}

func (s *CostApplicationService) Analytics(ctx context.Context, projectID string, days int) (CostAnalyticsReport, error) {
	if err := ctx.Err(); err != nil {
		return CostAnalyticsReport{}, err
	}
	if days < 1 || days > 365 {
		return CostAnalyticsReport{}, fmt.Errorf("days must be between 1 and 365")
	}
	st, err := s.requireStore()
	if err != nil {
		return CostAnalyticsReport{}, err
	}
	now := time.Now().UTC()
	projectID = strings.TrimSpace(projectID)
	since := now.Add(-time.Duration(days) * 24 * time.Hour)
	view, err := st.SummarizeCostAnalytics(store.CostObservationFilter{
		ProjectID: projectID,
		Since:     since,
	}, 25)
	if err != nil {
		return CostAnalyticsReport{}, err
	}
	return CostAnalyticsReport{
		Since: since, GeneratedAt: now, Days: days,
		ProjectID: projectID, Source: "LOCAL_COST_OBSERVATIONS",
		CostAnalyticsView: view,
	}, nil
}

func (s *CostApplicationService) Preferences(ctx context.Context) (store.CostPreferences, error) {
	if err := ctx.Err(); err != nil {
		return store.CostPreferences{}, err
	}
	st, err := s.requireStore()
	if err != nil {
		return store.CostPreferences{}, err
	}
	return st.GetCostPreferences()
}

func (s *CostApplicationService) PutPreferences(ctx context.Context, prefs store.CostPreferences) (store.CostPreferences, error) {
	if err := ctx.Err(); err != nil {
		return store.CostPreferences{}, err
	}
	st, err := s.requireStore()
	if err != nil {
		return store.CostPreferences{}, err
	}
	if err := store.ValidateCostPreferences(prefs); err != nil {
		return store.CostPreferences{}, err
	}
	prefs.UpdatedAt = time.Now().UTC()
	if err := st.PutCostPreferences(prefs); err != nil {
		return store.CostPreferences{}, err
	}
	return prefs, nil
}

func (s *CostApplicationService) Observations(ctx context.Context, filter store.CostObservationFilter) ([]store.CostObservation, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	st, err := s.requireStore()
	if err != nil {
		return nil, err
	}
	return st.ListCostObservations(filter)
}

func costBudgetConfirmationToken(planID string, revision int, summary CostSummary) string {
	if !summary.Budget.RequiresConfirmation {
		return ""
	}
	fingerprint := fmt.Sprintf(
		"%s|%d|%s|%d|%d|%d|%d|%d|%d|%s|%t|%d|%s",
		strings.TrimSpace(planID),
		revision,
		summary.Budget.Level,
		summary.Budget.ReportedCostMicroUSD,
		summary.Budget.APIEquivalentMicroUSD,
		summary.Budget.InferredMicroUSD,
		summary.Budget.UnclassifiedMicroUSD,
		summary.Budget.ReportedCostObservations,
		summary.Budget.UnverifiedObservations,
		summary.Budget.ActiveLimitMicroUSD,
		summary.Budget.Evidence,
		summary.Preferences.Enabled,
		summary.Preferences.WindowDays,
		summary.Preferences.UpdatedAt.UTC().Format(time.RFC3339Nano),
	)
	digest := sha256.Sum256([]byte(fingerprint))
	return hex.EncodeToString(digest[:])
}

// evaluateCostBudget uses ONLY provider-reported observations as a lower
// bound on spend. Estimates and inferred values are useful for planning, but
// cannot prove that a subscription or API account was actually charged.
// Unreported observations always make remaining headroom unverifiable.
func evaluateCostBudget(totals store.CostAnalyticsTotals, prefs store.CostPreferences) CostBudgetAssessment {
	known := totals.ProviderReportedMicroUSD
	reportedEvidence := evidenceState(totals.ObservationCount, totals.ReportedCostObservations)
	unverified := totals.ObservationCount - totals.ReportedCostObservations
	if unverified < 0 {
		unverified = 0
	}
	assessment := CostBudgetAssessment{
		Level:                    CostBudgetDisabled,
		KnownCostMicroUSD:        known,
		ReportedCostMicroUSD:     known,
		APIEquivalentMicroUSD:    totals.APIEquivalentMicroUSD,
		InferredMicroUSD:         totals.InferredMicroUSD,
		UnclassifiedMicroUSD:     totals.UnclassifiedMicroUSD,
		ReportedCostObservations: totals.ReportedCostObservations,
		UnverifiedObservations:   unverified,
		Evidence:                 reportedEvidence,
		Message:                  "No local cost budget is configured.",
	}
	if !prefs.Enabled || (prefs.SoftLimitMicroUSD == 0 && prefs.HardLimitMicroUSD == 0 && prefs.EmergencyLimitMicroUSD == 0) {
		return assessment
	}

	setExceeded := func(level CostBudgetLevel, limit int64, confirmation bool, message string) CostBudgetAssessment {
		assessment.Level = level
		assessment.ActiveLimitMicroUSD = limit
		assessment.RequiresConfirmation = confirmation
		assessment.Message = message
		return assessment
	}
	// All persisted amounts are nonnegative. Therefore a reported amount can
	// prove a configured threshold exceeded even if other costs are unknown.
	if prefs.EmergencyLimitMicroUSD > 0 && known >= prefs.EmergencyLimitMicroUSD {
		return setExceeded(CostBudgetEmergency, prefs.EmergencyLimitMicroUSD, true,
			"Provider-reported cost reached the emergency threshold. This is not a verified invoice.")
	}
	if prefs.HardLimitMicroUSD > 0 && known >= prefs.HardLimitMicroUSD {
		return setExceeded(CostBudgetHard, prefs.HardLimitMicroUSD, true,
			"Provider-reported cost reached the hard threshold. New expensive work requires confirmation.")
	}
	if prefs.SoftLimitMicroUSD > 0 && known >= prefs.SoftLimitMicroUSD {
		return setExceeded(CostBudgetSoft, prefs.SoftLimitMicroUSD, false,
			"Provider-reported cost reached the soft threshold. Other costs may be unverified.")
	}
	if reportedEvidence != CostEvidenceMeasured {
		assessment.Level = CostBudgetUnknown
		assessment.RequiresConfirmation = prefs.HardLimitMicroUSD > 0 || prefs.EmergencyLimitMicroUSD > 0
		assessment.Message = "Budget headroom is unverified: API-equivalent estimates, inferred amounts or missing cost evidence are not billed spend."
		if assessment.RequiresConfirmation {
			assessment.Message += " New expensive work requires confirmation under uncertainty."
		}
		return assessment
	}
	assessment.Level = CostBudgetOK
	assessment.Message = "All observations have provider-reported costs below configured thresholds; no invoice verification is implied."
	return assessment
}

func evaluateCostAnomaly(observations []store.CostObservation, prefs store.CostPreferences) CostAnomalyAssessment {
	assessment := CostAnomalyAssessment{
		Status:  CostAnomalyUnknown,
		Message: "No trustworthy cost observation is available for anomaly detection.",
	}
	if !prefs.Enabled {
		assessment.Status = CostAnomalyInsufficient
		assessment.Message = "Cost anomaly detection is disabled with Cost Intelligence."
		return assessment
	}
	var latest *store.CostObservation
	for i := range observations {
		if observations[i].CostMicroUSD != nil {
			latest = &observations[i]
			break
		}
	}
	if latest == nil {
		return assessment
	}

	history := make([]int64, 0, len(observations))
	for i := range observations {
		obs := observations[i]
		if obs.ID == latest.ID || obs.CostMicroUSD == nil {
			continue
		}
		if obs.Provider != latest.Provider || obs.Profile != latest.Profile || obs.CostKind != latest.CostKind {
			continue
		}
		history = append(history, *obs.CostMicroUSD)
	}
	assessment.Provider = latest.Provider
	assessment.Profile = latest.Profile
	assessment.CostKind = latest.CostKind
	assessment.LatestCostMicroUSD = *latest.CostMicroUSD
	assessment.HistoricalSamples = len(history)
	assessment.ObservedAt = latest.ObservedAt
	if len(history) < prefs.AnomalyMinimumSamples {
		assessment.Status = CostAnomalyInsufficient
		assessment.Message = "Not enough comparable historical cost samples to detect an anomaly."
		return assessment
	}
	sort.Slice(history, func(i, j int) bool { return history[i] < history[j] })
	median := history[len(history)/2]
	if len(history)%2 == 0 {
		low, high := history[len(history)/2-1], history[len(history)/2]
		median = low + (high-low)/2
	}
	scaledThreshold := float64(median) * prefs.AnomalyMultiplier
	threshold := int64(math.MaxInt64)
	if scaledThreshold < float64(math.MaxInt64) {
		threshold = int64(scaledThreshold)
	}
	if threshold < prefs.AnomalyMinCostMicroUSD {
		threshold = prefs.AnomalyMinCostMicroUSD
	}
	assessment.BaselineMedianMicroUSD = median
	assessment.ThresholdMicroUSD = threshold
	if *latest.CostMicroUSD > threshold {
		assessment.Status = CostAnomalyTriggered
		assessment.Triggered = true
		assessment.Message = "Latest comparable execution cost exceeds the configured anomaly threshold."
		return assessment
	}
	assessment.Status = CostAnomalyNormal
	assessment.Message = "Latest comparable execution cost is within the observed baseline."
	return assessment
}

func evidenceState(total, known int64) CostEvidenceState {
	switch {
	case total == 0:
		return CostEvidenceEmpty
	case known == 0:
		return CostEvidenceAttributionOnly
	case known < total:
		return CostEvidencePartial
	default:
		return CostEvidenceMeasured
	}
}
