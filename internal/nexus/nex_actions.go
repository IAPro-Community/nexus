package nexus

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/security"
)

const (
	NexActionStatusProposed    = "PROPOSED"
	NexActionStatusDenied      = "DENIED"
	NexActionStatusExecuted    = "EXECUTED"
	NexActionStatusFailed      = "FAILED"
	NexActionStatusUnavailable = "UNAVAILABLE"

	nexActionMaxReceipts = 200
)

var (
	ErrNexActionForbidden = errors.New("nex action forbidden")
	ErrNexActionUnknown   = errors.New("nex action unknown")
	ErrNexReceiptUnknown  = errors.New("nex action receipt unknown")
)

type NexActionSpec struct {
	ID                   string `json:"id"`
	Risk                 string `json:"risk"`
	RequiresConfirmation bool   `json:"requires_confirmation"`
	Owner                string `json:"owner"`
}

var nexActionCatalog = []NexActionSpec{
	{ID: "read_hud_summary", Risk: "read", RequiresConfirmation: false, Owner: "nexus.hud"},
	{ID: "read_model_catalog", Risk: "read", RequiresConfirmation: false, Owner: "nexus.model_watch"},
	{ID: "ack_capacity_alert", Risk: "mutate", RequiresConfirmation: true, Owner: "nexus.capacity"},
	{ID: "snooze_capacity_alert", Risk: "mutate", RequiresConfirmation: true, Owner: "nexus.capacity"},
	{ID: "open_nexus_surface", Risk: "navigation", RequiresConfirmation: true, Owner: "nexus.ui"},
	{ID: "delegate_agent_task", Risk: "delegate", RequiresConfirmation: true, Owner: "nexus.agents"},
}

var nexForbiddenActions = map[string]struct{}{
	"shell": {}, "exec": {}, "push": {}, "deploy": {}, "delete_repo": {}, "paid_fallback": {},
}

type NexActionReceipt struct {
	ReceiptID   string            `json:"receipt_id"`
	ActionID    string            `json:"action_id"`
	Risk        string            `json:"risk"`
	Owner       string            `json:"owner"`
	Status      string            `json:"status"`
	Summary     string            `json:"summary,omitempty"`
	Params      map[string]string `json:"params,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	CompletedAt *time.Time        `json:"completed_at,omitempty"`
}

// NexActionExecutor runs one allowlisted action through its canonical owner
// service and returns the honest resulting status plus a summary.
type NexActionExecutor func(spec NexActionSpec, params map[string]string) (status, summary string)

type nexActionDisk struct {
	Receipts []NexActionReceipt `json:"receipts"`
}

type NexActionService struct {
	mu       sync.Mutex
	filePath string
	receipts []NexActionReceipt
}

func NewNexActionService(filePath string) *NexActionService {
	if strings.TrimSpace(filePath) == "" {
		dataDir, err := config.DataDir()
		if err != nil || strings.TrimSpace(dataDir) == "" {
			dataDir = filepath.Join(os.TempDir(), "iapro-nexus")
		}
		filePath = filepath.Join(dataDir, "nex-action-receipts.json")
	}
	s := &NexActionService{filePath: filePath, receipts: []NexActionReceipt{}}
	if raw, err := os.ReadFile(filePath); err == nil {
		var disk nexActionDisk
		if json.Unmarshal(raw, &disk) == nil && disk.Receipts != nil {
			s.receipts = disk.Receipts
		}
	}
	return s
}

func (s *NexActionService) Catalog() []NexActionSpec {
	out := make([]NexActionSpec, len(nexActionCatalog))
	copy(out, nexActionCatalog)
	return out
}

func (s *NexActionService) Receipts() []NexActionReceipt {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]NexActionReceipt, len(s.receipts))
	copy(out, s.receipts)
	return out
}

func (s *NexActionService) saveLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.filePath), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(nexActionDisk{Receipts: s.receipts}, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.filePath + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.filePath)
}

func redactParams(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		k = trimValidUTF8(security.Redact(k), 64)
		v = trimValidUTF8(security.Redact(v), 300)
		if k != "" {
			out[k] = v
		}
	}
	return out
}

func lookupNexActionSpec(id string) (NexActionSpec, error) {
	id = strings.ToLower(strings.TrimSpace(id))
	if _, bad := nexForbiddenActions[id]; bad {
		return NexActionSpec{}, ErrNexActionForbidden
	}
	for _, spec := range nexActionCatalog {
		if spec.ID == id {
			return spec, nil
		}
	}
	return NexActionSpec{}, ErrNexActionUnknown
}

func (s *NexActionService) Propose(actionID string, params map[string]string, now time.Time) (NexActionReceipt, error) {
	spec, err := lookupNexActionSpec(actionID)
	if err != nil {
		return NexActionReceipt{}, err
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return NexActionReceipt{}, err
	}
	r := NexActionReceipt{
		ReceiptID: "rcpt-" + hex.EncodeToString(b[:]),
		ActionID:  spec.ID,
		Risk:      spec.Risk,
		Owner:     spec.Owner,
		Status:    NexActionStatusProposed,
		Params:    redactParams(params),
		CreatedAt: now.UTC(),
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.receipts = append(s.receipts, r)
	if len(s.receipts) > nexActionMaxReceipts {
		s.receipts = append([]NexActionReceipt{}, s.receipts[len(s.receipts)-nexActionMaxReceipts:]...)
	}
	if err := s.saveLocked(); err != nil {
		return NexActionReceipt{}, err
	}
	return r, nil
}

// Confirm is single-use: only a PROPOSED receipt can be decided. A repeated
// confirmation returns the already-final receipt without executing again.
func (s *NexActionService) Confirm(receiptID string, approved bool, exec NexActionExecutor, now time.Time) (NexActionReceipt, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	idx := -1
	for i := range s.receipts {
		if s.receipts[i].ReceiptID == strings.TrimSpace(receiptID) {
			idx = i
			break
		}
	}
	if idx < 0 {
		return NexActionReceipt{}, ErrNexReceiptUnknown
	}
	r := s.receipts[idx]
	if r.Status != NexActionStatusProposed {
		return r, nil
	}
	spec, err := lookupNexActionSpec(r.ActionID)
	if err != nil {
		return r, err
	}
	done := now.UTC()
	switch {
	case !approved:
		r.Status = NexActionStatusDenied
		r.Summary = "denied by user"
	case exec == nil:
		r.Status = NexActionStatusUnavailable
		r.Summary = "no canonical executor configured"
	default:
		status, summary := exec(spec, r.Params)
		switch status {
		case NexActionStatusExecuted, NexActionStatusFailed, NexActionStatusUnavailable:
			r.Status = status
		default:
			r.Status = NexActionStatusFailed
		}
		r.Summary = trimValidUTF8(security.Redact(summary), 300)
	}
	r.CompletedAt = &done
	s.receipts[idx] = r
	if err := s.saveLocked(); err != nil {
		return r, fmt.Errorf("persist receipt: %w", err)
	}
	return r, nil
}
