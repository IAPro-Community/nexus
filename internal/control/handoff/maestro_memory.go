package handoff

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/security"
	"github.com/IAPro-Community/nexus/internal/executil"
	"github.com/IAPro-Community/nexus/internal/runtime"
)

const (
	maestroMemoryMaxResponse = 256 * 1024
	maestroMemoryMaxLimit    = 50
)

var maestroMemoryTaskID = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,128}$`)
var maestroMemoryObservationID = regexp.MustCompile(`^obs_[a-f0-9]{16}$`)

// MaestroMemoryReadResult is an ephemeral read model. It is NOT a second
// semantic memory store; nothing here is cached or persisted in Nexus.
type MaestroMemoryReadResult struct {
	SchemaVersion int                        `json:"schemaVersion"`
	Authority     string                     `json:"authority"`
	Items         []MaestroMemoryObservation  `json:"items"`
	Page          MaestroMemoryPage           `json:"page"`
}

type MaestroMemoryObservation struct {
	ID        string                  `json:"id"`
	Timestamp string                  `json:"timestamp"`
	Type      string                  `json:"type"`
	Summary   string                  `json:"summary"`
	Verified  bool                    `json:"verified"`
	Scope     MaestroMemoryScope      `json:"scope"`
}

type MaestroMemoryScope struct {
	Level        string `json:"level"`
	RepositoryID string `json:"repositoryId"`
	Branch       string `json:"branch,omitempty"`
	WorkspaceID  string `json:"workspaceId,omitempty"`
	TaskID       string `json:"taskId,omitempty"`
	HeadCommit   string `json:"headCommit,omitempty"`
}

type MaestroMemoryPage struct {
	Offset  int  `json:"offset"`
	Limit   int  `json:"limit"`
	HasMore bool `json:"hasMore"`
}

type maestroMemoryRPCReply struct {
	JSONRPC string                 `json:"jsonrpc"`
	ID      int                    `json:"id"`
	Result  *MaestroMemoryReadResult `json:"result"`
	Error   *struct {
		Code int `json:"code"`
	} `json:"error"`
}

// ReadMaestroMemory sends a bounded, project-scoped request to the canonical
// Maestro Bridge. A missing/older Bridge fails explicitly; do not fall back to
// the legacy Nexus IntelligenceMemory store. The caller supplies an authorized
// workspace, never a raw memory directory or repository identity.
func ReadMaestroMemory(parent context.Context, workspace, taskID, search string, limit int) (MaestroMemoryReadResult, error) {
	var zero MaestroMemoryReadResult
	if strings.TrimSpace(workspace) == "" {
		return zero, errors.New("Maestro memory workspace is required")
	}
	if taskID != "" && !maestroMemoryTaskID.MatchString(taskID) {
		return zero, errors.New("invalid Maestro memory task identifier")
	}
	if len(search) > 200 {
		return zero, errors.New("Maestro memory query exceeds 200 characters")
	}
	if limit == 0 {
		limit = 20
	}
	if limit < 1 || limit > maestroMemoryMaxLimit {
		return zero, errors.New("Maestro memory limit must be between 1 and 50")
	}
	binary, err := runtime.LookPath("orquestrador-maestro")
	if err != nil {
		return zero, fmt.Errorf("Maestro memory unavailable: %w", err)
	}
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()
	params := map[string]any{"limit": limit, "offset": 0}
	if taskID != "" {
		params["taskId"] = taskID
	}
	if search != "" {
		params["search"] = security.Redact(search)
	}
	request, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": "memory.search", "params": params,
	})
	if err != nil {
		return zero, err
	}
	cmd := executil.CommandContext(ctx, binary, "bridge", "--stdio", "--project-path", workspace)
	cmd.Stdin = bytes.NewReader(append(request, '\n'))
	out, err := cmd.Output()
	if err != nil {
		return zero, fmt.Errorf("Maestro memory request failed: %w", err)
	}
	return decodeMaestroMemorySearch(out)
}

func decodeMaestroMemorySearch(raw []byte) (MaestroMemoryReadResult, error) {
	var zero MaestroMemoryReadResult
	if len(raw) > maestroMemoryMaxResponse {
		return zero, errors.New("Maestro memory response exceeds safety budget")
	}
	var reply maestroMemoryRPCReply
	if err := json.Unmarshal(bytes.TrimSpace(raw), &reply); err != nil {
		return zero, fmt.Errorf("invalid Maestro memory response: %w", err)
	}
	if reply.JSONRPC != "2.0" || reply.ID != 1 {
		return zero, errors.New("Maestro memory response identity mismatch")
	}
	if reply.Error != nil {
		return zero, fmt.Errorf("Maestro memory method unavailable (%d)", reply.Error.Code)
	}
	if reply.Result == nil || reply.Result.SchemaVersion != 1 || reply.Result.Authority != "maestro" {
		return zero, errors.New("unsupported or unverified Maestro memory contract")
	}
	result := *reply.Result
	if result.Page.Limit < 1 || result.Page.Limit > maestroMemoryMaxLimit ||
		result.Page.Offset < 0 || len(result.Items) > result.Page.Limit {
		return zero, errors.New("Maestro memory pagination invalid")
	}
	for i := range result.Items {
		item := &result.Items[i]
		if !maestroMemoryObservationID.MatchString(item.ID) ||
			item.Type == "" || len(item.Summary) > 500 ||
			item.Scope.RepositoryID == "" || item.Scope.Level == "" ||
			item.Timestamp == "" {
			return zero, errors.New("invalid Maestro memory observation")
		}
		item.Summary = security.Redact(item.Summary)
	}
	if result.Items == nil {
		result.Items = []MaestroMemoryObservation{}
	}
	return result, nil
}
