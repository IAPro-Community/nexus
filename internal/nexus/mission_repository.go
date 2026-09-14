package nexus

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/IAPro-Community/nexus/internal/nexus/runner"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

// storeRunRepository adapts the product SQLite store to the runner's durable
// repository contract. Lease columns are authoritative; payload JSON carries
// the rest of the immutable/auditable mission state.
type storeRunRepository struct {
	st *store.Store
}

func newStoreRunRepository(st *store.Store) runner.RunRepository {
	return &storeRunRepository{st: st}
}

func (r *storeRunRepository) SaveRun(ctx context.Context, run *runner.MissionRun) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if r.st == nil {
		return fmt.Errorf("nexus store unavailable")
	}
	if run == nil || run.ID == "" {
		return fmt.Errorf("mission run is required")
	}
	payload, err := json.Marshal(run)
	if err != nil {
		return fmt.Errorf("marshal mission run: %w", err)
	}
	return r.st.UpsertMissionRun(store.MissionRunRecord{
		ID:             run.ID,
		PlanID:         run.PlanID,
		ProjectID:      run.ProjectID,
		State:          string(run.State),
		PayloadJSON:    string(payload),
		LeaseOwner:     run.LeaseOwner,
		LeaseToken:     run.LeaseToken,
		LeaseExpiresAt: run.LeaseExpiresAt,
		HeartbeatAt:    run.HeartbeatAt,
		CreatedAt:      run.StartedAt,
		UpdatedAt:      run.UpdatedAt,
	})
}

func (r *storeRunRepository) GetRun(ctx context.Context, id string) (*runner.MissionRun, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	rec, err := r.st.GetMissionRun(id)
	if err != nil {
		return nil, mapRunStoreError(err)
	}
	return decodeMissionRun(rec)
}

func (r *storeRunRepository) ListRuns(ctx context.Context) ([]*runner.MissionRun, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	recs, err := r.st.ListMissionRuns()
	if err != nil {
		return nil, err
	}
	out := make([]*runner.MissionRun, 0, len(recs))
	for i := range recs {
		run, err := decodeMissionRun(&recs[i])
		if err != nil {
			return nil, err
		}
		out = append(out, run)
	}
	return out, nil
}

func (r *storeRunRepository) AcquireLease(ctx context.Context, id, owner string, ttl time.Duration) (*runner.MissionRun, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	rec, err := r.st.AcquireMissionLease(id, owner, ttl)
	if err != nil {
		if isLeaseError(err) {
			return nil, runner.ErrLeaseHeld
		}
		return nil, mapRunStoreError(err)
	}
	return decodeMissionRun(rec)
}

func (r *storeRunRepository) RenewLease(ctx context.Context, id, owner, token string, ttl time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := r.st.RenewMissionLease(id, owner, token, ttl); err != nil {
		if isLeaseError(err) {
			return runner.ErrLeaseHeld
		}
		return mapRunStoreError(err)
	}
	return nil
}

func (r *storeRunRepository) ReleaseLease(ctx context.Context, id, owner, token string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := r.st.ReleaseMissionLease(id, owner, token); err != nil {
		if isLeaseError(err) {
			return runner.ErrLeaseHeld
		}
		return mapRunStoreError(err)
	}
	return nil
}

func (r *storeRunRepository) CommitInterventionResolution(ctx context.Context, run *runner.MissionRun, resolution *runner.InterventionResolution) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if r.st == nil || run == nil || resolution == nil {
		return fmt.Errorf("mission intervention commit requires store, run and resolution")
	}
	payload, err := json.Marshal(run)
	if err != nil {
		return fmt.Errorf("marshal mission run intervention resolution: %w", err)
	}
	return r.st.CommitMissionInterventionResolution(store.MissionRunRecord{
		ID: run.ID, PlanID: run.PlanID, ProjectID: run.ProjectID, State: string(run.State), PayloadJSON: string(payload),
		LeaseOwner: run.LeaseOwner, LeaseToken: run.LeaseToken, LeaseExpiresAt: run.LeaseExpiresAt, HeartbeatAt: run.HeartbeatAt,
		CreatedAt: run.StartedAt, UpdatedAt: run.UpdatedAt,
	}, store.MissionInterventionResolutionRecord{
		RunID: run.ID, InterventionID: resolution.InterventionID, Version: resolution.Version,
		OptionID: resolution.OptionID, IdempotencyKey: resolution.IdempotencyKey, ResolvedAt: resolution.ResolvedAt,
		ResolvedBy: resolution.ResolvedBy, ResumeStatus: "PENDING",
	})
}

func decodeMissionRun(rec *store.MissionRunRecord) (*runner.MissionRun, error) {
	if rec == nil {
		return nil, runner.ErrRunNotFound
	}
	var run runner.MissionRun
	if err := json.Unmarshal([]byte(rec.PayloadJSON), &run); err != nil {
		return nil, fmt.Errorf("decode mission run %s: %w", rec.ID, err)
	}
	// Lease columns are written atomically and may be newer than payload_json.
	run.LeaseOwner = rec.LeaseOwner
	run.LeaseToken = rec.LeaseToken
	run.LeaseExpiresAt = rec.LeaseExpiresAt
	run.HeartbeatAt = rec.HeartbeatAt
	if rec.State != "" {
		run.State = runner.State(rec.State)
	}
	if !rec.UpdatedAt.IsZero() {
		run.UpdatedAt = rec.UpdatedAt
	}
	return &run, nil
}

func mapRunStoreError(err error) error {
	if errors.Is(err, sql.ErrNoRows) || errors.Is(err, store.ErrNotFound) {
		return runner.ErrRunNotFound
	}
	return err
}

func isLeaseError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return msg == "mission run lease held" || msg == "mission run lease fencing mismatch"
}

func (r *storeRunRepository) SaveContextCapsule(ctx context.Context, capsule *runner.ContextCapsule) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if capsule == nil {
		return fmt.Errorf("context capsule is required")
	}
	if existing, err := r.st.GetFlowContextCapsule(capsule.RunID, capsule.Step.ID); err == nil && existing.ID != "" {
		capsule.ID = existing.ID
	} else if err != nil && !store.IsFlowEvidenceNotFound(err) {
		return err
	}
	payload, err := json.Marshal(capsule)
	if err != nil {
		return err
	}
	record, err := r.st.UpsertFlowContextCapsule(store.FlowContextCapsuleRecord{ID: capsule.ID, RunID: capsule.RunID, StepID: capsule.Step.ID, FlowRevision: capsule.FlowRevision, ContentJSON: string(payload), CreatedAt: capsule.CreatedAt})
	if err != nil {
		return err
	}
	capsule.ID = record.ID
	return nil
}
func (r *storeRunRepository) GetContextCapsule(ctx context.Context, runID, stepID string) (*runner.ContextCapsule, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	record, err := r.st.GetFlowContextCapsule(runID, stepID)
	if err != nil {
		return nil, err
	}
	var capsule runner.ContextCapsule
	if err := json.Unmarshal([]byte(record.ContentJSON), &capsule); err != nil {
		return nil, fmt.Errorf("decode context capsule: %w", err)
	}
	return &capsule, nil
}
func (r *storeRunRepository) SaveWorkReceipt(ctx context.Context, receipt *runner.WorkReceipt) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if receipt == nil {
		return fmt.Errorf("work receipt is required")
	}
	if existing, err := r.st.GetFlowWorkReceipt(receipt.RunID, receipt.StepID); err == nil && existing.ID != "" {
		receipt.ID = existing.ID
	} else if err != nil && !store.IsFlowEvidenceNotFound(err) {
		return err
	}
	payload, err := json.Marshal(receipt)
	if err != nil {
		return err
	}
	record, err := r.st.UpsertFlowWorkReceipt(store.FlowWorkReceiptRecord{ID: receipt.ID, RunID: receipt.RunID, StepID: receipt.StepID, Status: receipt.Status, ContentJSON: string(payload), CreatedAt: receipt.CompletedAt})
	if err != nil {
		return err
	}
	receipt.ID = record.ID
	return nil
}
func (r *storeRunRepository) GetWorkReceipt(ctx context.Context, runID, stepID string) (*runner.WorkReceipt, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	record, err := r.st.GetFlowWorkReceipt(runID, stepID)
	if err != nil {
		return nil, err
	}
	var receipt runner.WorkReceipt
	if err := json.Unmarshal([]byte(record.ContentJSON), &receipt); err != nil {
		return nil, fmt.Errorf("decode work receipt: %w", err)
	}
	return &receipt, nil
}
