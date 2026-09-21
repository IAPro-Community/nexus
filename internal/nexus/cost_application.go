package nexus

import (
	"context"
	"errors"
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
	KnownCostMicroUSD    int64             `json:"known_cost_microusd"`
	Evidence             CostEvidenceState `json:"evidence"`
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
	Status              CostAnomalyStatus `json:"status"`
	Triggered           bool              `json:"triggered"`
	Provider            string            `json:"provider,omitempty"`
	Profile             string            `json:"profile,omitempty"`
	CostKind            string            `json:"cost_kind,omitempty"`
	LatestCostMicroUSD  int64             `json:"latest_cost_microusd,omitempty"`
	BaselineMedianMicroUSD int64          `json:"baseline_median_microusd,omitempty"`
	ThresholdMicroUSD   int64             `json:"threshold_microusd,omitempty"`
	HistoricalSamples   int               `json:"historical_samples"`
	ObservedAt          time.Time         `json:"observed_at,omitempty"`
	Message             string            `json:"message"`
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
		Budget:        evaluateCostBudget(aggregate, prefs, costEvidence),
		Anomaly:       evaluateCostAnomaly(observations, prefs),
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

func evaluateCostBudget(aggregate store.CostAggregate, prefs store.CostPreferences, evidence CostEvidenceState) CostBudgetAssessment {
	assessment := CostBudgetAssessment{
		Level:             CostBudgetDisabled,
		KnownCostMicroUSD: aggregate.CostMicroUSD,
		Evidence:          evidence,
		Message:           "No local cost budget is configured.",
	}
	if !prefs.Enabled || (prefs.SoftLimitMicroUSD == 0 && prefs.HardLimitMicroUSD == 0 && prefs.EmergencyLimitMicroUSD == 0) {
		return assessment
	}

	known := aggregate.CostMicroUSD
	setExceeded := func(level CostBudgetLevel, limit int64, confirmation bool, message string) CostBudgetAssessment {
		assessment.Level = level
		assessment.ActiveLimitMicroUSD = limit
		assessment.RequiresConfirmation = confirmation
		assessment.Message = message
		return assessment
	}

	if prefs.EmergencyLimitMicroUSD > 0 && known >= prefs.EmergencyLimitMicroUSD {
		return setExceeded(CostBudgetEmergency, prefs.EmergencyLimitMicroUSD, true, "Known spend reached the emergency budget. New expensive work requires explicit confirmation.")
	}
	if prefs.HardLimitMicroUSD > 0 && known >= prefs.HardLimitMicroUSD {
		return setExceeded(CostBudgetHard, prefs.HardLimitMicroUSD, true, "Known spend reached the hard budget. New expensive work requires explicit confirmation.")
	}
	if prefs.SoftLimitMicroUSD > 0 && known >= prefs.SoftLimitMicroUSD {
		return setExceeded(CostBudgetSoft, prefs.SoftLimitMicroUSD, false, "Known spend reached the soft budget.")
	}
	if evidence != CostEvidenceMeasured {
		assessment.Level = CostBudgetUnknown
		assessment.RequiresConfirmation = prefs.HardLimitMicroUSD > 0 || prefs.EmergencyLimitMicroUSD > 0
		assessment.Message = "Budget headroom cannot be proven because some execution cost is unknown."
		if assessment.RequiresConfirmation {
			assessment.Message += " New expensive work requires explicit confirmation under uncertainty."
		}
		return assessment
	}
	assessment.Level = CostBudgetOK
	assessment.Message = "Measured spend is below configured budget thresholds."
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
