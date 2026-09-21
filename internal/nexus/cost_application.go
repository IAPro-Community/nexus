package nexus

import (
	"context"
	"errors"
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

type CostSummary struct {
	Since         time.Time                     `json:"since"`
	GeneratedAt   time.Time                     `json:"generated_at"`
	ProjectID     string                        `json:"project_id,omitempty"`
	Aggregate     store.CostAggregate           `json:"aggregate"`
	ByProvider    []store.ProviderCostAggregate `json:"by_provider"`
	CostEvidence  CostEvidenceState             `json:"cost_evidence"`
	TokenEvidence CostEvidenceState             `json:"token_evidence"`
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
	if days <= 0 {
		days = 7
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
	return CostSummary{
		Since:         since,
		GeneratedAt:   now,
		ProjectID:     filter.ProjectID,
		Aggregate:     aggregate,
		ByProvider:    byProvider,
		CostEvidence:  evidenceState(aggregate.ObservationCount, aggregate.KnownCostObservations),
		TokenEvidence: evidenceState(aggregate.ObservationCount, aggregate.KnownTokenObservations),
	}, nil
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
