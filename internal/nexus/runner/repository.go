package runner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
)

var ErrRunNotFound = errors.New("mission run not found")
var ErrLeaseHeld = errors.New("mission run lease held by another worker")

// RunRepository is the durable authority for MissionRun state and worker leases.
type RunRepository interface {
	SaveRun(context.Context, *MissionRun) error
	GetRun(context.Context, string) (*MissionRun, error)
	ListRuns(context.Context) ([]*MissionRun, error)
	AcquireLease(context.Context, string, string, time.Duration) (*MissionRun, error)
	RenewLease(context.Context, string, string, string, time.Duration) error
	ReleaseLease(context.Context, string, string, string) error
}

// InterventionCommitter gives the application transaction a single durable
// boundary for the run payload, canonical resolution and resume outbox intent.
// Production SQLite and the memory test repository both implement it.
type InterventionCommitter interface {
	CommitInterventionResolution(context.Context, *MissionRun, *InterventionResolution) error
}

// MemoryRunRepository is test/dev only. Production Nexus injects the SQLite-backed adapter.
type MemoryRunRepository struct {
	mu          sync.Mutex
	runs        map[string]*MissionRun
	capsules    map[string]*ContextCapsule
	receipts    map[string]*WorkReceipt
	resolutions map[string]*InterventionResolution
}

func NewMemoryRunRepository() *MemoryRunRepository {
	return &MemoryRunRepository{runs: map[string]*MissionRun{}, capsules: map[string]*ContextCapsule{}, receipts: map[string]*WorkReceipt{}, resolutions: map[string]*InterventionResolution{}}
}

func cloneRun(in *MissionRun) *MissionRun {
	if in == nil {
		return nil
	}
	b, _ := json.Marshal(in)
	var out MissionRun
	_ = json.Unmarshal(b, &out)
	return &out
}
func (m *MemoryRunRepository) SaveRun(ctx context.Context, run *MissionRun) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if run == nil || run.ID == "" {
		return fmt.Errorf("mission run is required")
	}
	if current, ok := m.runs[run.ID]; ok {
		if current.LeaseToken != "" && current.LeaseToken != run.LeaseToken {
			return ErrLeaseHeld
		}
		copy := cloneRun(run)
		// Lease metadata is maintained only by Acquire/Renew/Release and must
		// never move backwards because a worker saved an older in-memory copy.
		copy.LeaseOwner = current.LeaseOwner
		copy.LeaseToken = current.LeaseToken
		copy.LeaseExpiresAt = current.LeaseExpiresAt
		copy.HeartbeatAt = current.HeartbeatAt
		m.runs[run.ID] = copy
		return nil
	}
	m.runs[run.ID] = cloneRun(run)
	return nil
}
func (m *MemoryRunRepository) GetRun(ctx context.Context, id string) (*MissionRun, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.runs[id]
	if !ok {
		return nil, ErrRunNotFound
	}
	return cloneRun(r), nil
}
func (m *MemoryRunRepository) ListRuns(ctx context.Context) ([]*MissionRun, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*MissionRun, 0, len(m.runs))
	for _, r := range m.runs {
		out = append(out, cloneRun(r))
	}
	return out, nil
}
func (m *MemoryRunRepository) AcquireLease(ctx context.Context, id, owner string, ttl time.Duration) (*MissionRun, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.runs[id]
	if !ok {
		return nil, ErrRunNotFound
	}
	now := time.Now().UTC()
	if r.LeaseOwner != "" && r.LeaseExpiresAt != nil && r.LeaseExpiresAt.After(now) {
		return nil, ErrLeaseHeld
	}
	token := fmt.Sprintf("lease-%s-%d", owner, now.UnixNano())
	expires := now.Add(ttl)
	r.LeaseOwner, r.LeaseToken, r.LeaseExpiresAt, r.HeartbeatAt = owner, token, &expires, &now
	return cloneRun(r), nil
}
func (m *MemoryRunRepository) RenewLease(ctx context.Context, id, owner, token string, ttl time.Duration) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.runs[id]
	if !ok {
		return ErrRunNotFound
	}
	if r.LeaseOwner != owner || r.LeaseToken != token {
		return ErrLeaseHeld
	}
	now := time.Now().UTC()
	expires := now.Add(ttl)
	r.HeartbeatAt, r.LeaseExpiresAt = &now, &expires
	return nil
}
func (m *MemoryRunRepository) ReleaseLease(ctx context.Context, id, owner, token string) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.runs[id]
	if !ok {
		return ErrRunNotFound
	}
	if r.LeaseOwner != owner || r.LeaseToken != token {
		return ErrLeaseHeld
	}
	r.LeaseOwner, r.LeaseToken, r.LeaseExpiresAt = "", "", nil
	return nil
}

func (m *MemoryRunRepository) CommitInterventionResolution(ctx context.Context, run *MissionRun, resolution *InterventionResolution) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if run == nil || resolution == nil || run.ID == "" || resolution.IdempotencyKey == "" {
		return fmt.Errorf("mission intervention resolution is required")
	}
	current, ok := m.runs[run.ID]
	if !ok {
		return ErrRunNotFound
	}
	if current.LeaseToken != "" && current.LeaseToken != run.LeaseToken {
		return ErrLeaseHeld
	}
	if existing := m.resolutions[resolution.IdempotencyKey]; existing != nil {
		return nil
	}
	m.resolutions[resolution.IdempotencyKey] = cloneResolution(resolution)
	copy := cloneRun(run)
	copy.LeaseOwner, copy.LeaseToken, copy.LeaseExpiresAt, copy.HeartbeatAt = current.LeaseOwner, current.LeaseToken, current.LeaseExpiresAt, current.HeartbeatAt
	m.runs[run.ID] = copy
	return nil
}

func cloneResolution(in *InterventionResolution) *InterventionResolution {
	if in == nil {
		return nil
	}
	b, _ := json.Marshal(in)
	var out InterventionResolution
	_ = json.Unmarshal(b, &out)
	return &out
}

func contextErr(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	return ctx.Err()
}

// EvidenceRepository persists typed Step evidence independently from the run
// payload so audit/recovery does not require parsing provider transcripts.
type EvidenceRepository interface {
	SaveContextCapsule(context.Context, *ContextCapsule) error
	GetContextCapsule(context.Context, string, string) (*ContextCapsule, error)
	SaveWorkReceipt(context.Context, *WorkReceipt) error
	GetWorkReceipt(context.Context, string, string) (*WorkReceipt, error)
}

func evidenceKey(runID, stepID string) string { return runID + "\x00" + stepID }

func cloneCapsule(in *ContextCapsule) *ContextCapsule {
	if in == nil {
		return nil
	}
	b, _ := json.Marshal(in)
	var out ContextCapsule
	_ = json.Unmarshal(b, &out)
	return &out
}
func cloneReceipt(in *WorkReceipt) *WorkReceipt {
	if in == nil {
		return nil
	}
	b, _ := json.Marshal(in)
	var out WorkReceipt
	_ = json.Unmarshal(b, &out)
	return &out
}
func (m *MemoryRunRepository) SaveContextCapsule(ctx context.Context, capsule *ContextCapsule) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	if capsule == nil || capsule.RunID == "" || capsule.Step.ID == "" {
		return fmt.Errorf("context capsule is required")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.capsules[evidenceKey(capsule.RunID, capsule.Step.ID)] = cloneCapsule(capsule)
	return nil
}
func (m *MemoryRunRepository) GetContextCapsule(ctx context.Context, runID, stepID string) (*ContextCapsule, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	value, ok := m.capsules[evidenceKey(runID, stepID)]
	if !ok {
		return nil, ErrRunNotFound
	}
	return cloneCapsule(value), nil
}
func (m *MemoryRunRepository) SaveWorkReceipt(ctx context.Context, receipt *WorkReceipt) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	if receipt == nil || receipt.RunID == "" || receipt.StepID == "" {
		return fmt.Errorf("work receipt is required")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	key := evidenceKey(receipt.RunID, receipt.StepID)
	if existing, ok := m.receipts[key]; ok && existing.ID != "" {
		receipt.ID = existing.ID
	}
	m.receipts[key] = cloneReceipt(receipt)
	return nil
}
func (m *MemoryRunRepository) GetWorkReceipt(ctx context.Context, runID, stepID string) (*WorkReceipt, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	value, ok := m.receipts[evidenceKey(runID, stepID)]
	if !ok {
		return nil, ErrRunNotFound
	}
	return cloneReceipt(value), nil
}
