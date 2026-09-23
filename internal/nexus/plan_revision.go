package nexus

import (
	"encoding/json"
	"fmt"

	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

// loadWorkPlanRevision returns the immutable WorkPlan captured for one exact
// revision. Callers must not substitute the mutable work_plans row when an
// approved or scheduled revision has already been selected.
func loadWorkPlanRevision(st *store.Store, planID string, revision int) (*store.WorkPlan, *store.PlanRevision, error) {
	if st == nil {
		return nil, nil, fmt.Errorf("plan store is required")
	}
	if planID == "" || revision <= 0 {
		return nil, nil, fmt.Errorf("plan id and positive revision are required")
	}
	rev, err := st.GetPlanRevision(planID, revision)
	if err != nil {
		return nil, nil, err
	}
	var frozen store.WorkPlan
	if err := json.Unmarshal([]byte(rev.SnapshotJSON), &frozen); err != nil {
		return nil, nil, fmt.Errorf("decode plan revision %s/%d: %w", planID, revision, err)
	}
	if frozen.ID != planID {
		return nil, nil, fmt.Errorf("plan revision snapshot id %q does not match %q", frozen.ID, planID)
	}
	if frozen.CurrentRevision != revision {
		return nil, nil, fmt.Errorf("plan revision snapshot reports revision %d, want %d", frozen.CurrentRevision, revision)
	}
	return &frozen, rev, nil
}
