package handoff

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode/utf16"
	"unicode/utf8"

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

var maestroMemoryObservationTypes = map[string]bool{
	"decision":       true,
	"discovery":      true,
	"problem":        true,
	"attempt":        true,
	"failure":        true,
	"implementation": true,
	"verification":   true,
	"risk":           true,
	"dependency":     true,
	"environment":    true,
	"workaround":     true,
}

var maestroMemoryScopeLevels = map[string]bool{
	"repository": true,
	"branch":     true,
	"workspace":  true,
	"task":       true,
}

// MaestroMemoryReadResult is an ephemeral read model. It is NOT a second
// semantic memory store; nothing here is cached or persisted in Nexus.
type MaestroMemoryReadResult struct {
	SchemaVersion int                        `json:"schemaVersion"`
	Authority     string                     `json:"authority"`
	Items         []MaestroMemoryObservation `json:"items"`
	Page          MaestroMemoryPage          `json:"page"`
}

type MaestroMemoryObservation struct {
	ID        string             `json:"id"`
	Timestamp string             `json:"timestamp"`
	Type      string             `json:"type"`
	Summary   string             `json:"summary"`
	Verified  bool               `json:"verified"`
	Scope     MaestroMemoryScope `json:"scope"`
}

type MaestroMemoryScope struct {
	Level        string `json:"level"`
	RepositoryID string `json:"repositoryId"`
	Branch       string `json:"branch,omitempty"`
	WorkspaceID  string `json:"workspaceId,omitempty"`
	TaskID       string `json:"taskId,omitempty"`
	HeadCommit   string `json:"headCommit,omitempty"`
	Promoted     bool   `json:"promoted,omitempty"`
}

// MaestroMemoryWriteRequest is one explicit, operator-confirmed write to the
// canonical Maestro memory. Nexus does not derive scope, verify observations,
// persist requests, or retain a local idempotency cache.
type MaestroMemoryWriteRequest struct {
	Confirmed      bool
	Reason         string
	IdempotencyKey string
	Type           string
	Summary        string
	Details        string
	ScopeLevel     string
	TaskID         string
	Tags           []string
	Files          []string
	ProjectPath    string
}

// MaestroMemoryWriteResult is the canonical acknowledgement returned by
// Maestro. It contains no observation body and is safe to return ephemerally.
type MaestroMemoryWriteResult struct {
	SchemaVersion int                `json:"schemaVersion"`
	Authority     string             `json:"authority"`
	Status        string             `json:"status"`
	ID            string             `json:"id"`
	Scope         MaestroMemoryScope `json:"scope"`
}

type MaestroMemoryPage struct {
	Offset  int  `json:"offset"`
	Limit   int  `json:"limit"`
	HasMore bool `json:"hasMore"`
}

type maestroMemoryWriteInit struct {
	ProtocolVersion int `json:"protocolVersion"`
	Capabilities    struct {
		Memory struct {
			Write bool `json:"write"`
		} `json:"memory"`
	} `json:"capabilities"`
}

type maestroMemoryRPCReply struct {
	JSONRPC string                   `json:"jsonrpc"`
	ID      int                      `json:"id"`
	Result  *MaestroMemoryReadResult `json:"result"`
	Error   *struct {
		Code int `json:"code"`
	} `json:"error"`
}

// maestroMemoryMethod fails closed on older Maestro Bridge versions.
func maestroMemoryMethod(init maestroBridgeInit) (string, error) {
	if init.ProtocolVersion != maestroBridgeProtocolVersion {
		return "", fmt.Errorf("unsupported Maestro Bridge protocol version %d", init.ProtocolVersion)
	}
	if !init.Capabilities.Memory.Read {
		return "", errors.New("maestro Bridge has no scoped memory read capability")
	}
	return "memory.search", nil
}

// ReadMaestroMemory sends a bounded, project-scoped request to the canonical
// Maestro Bridge. A missing/older Bridge fails explicitly; do not fall back to
// the legacy Nexus IntelligenceMemory store. The caller supplies an authorized
// workspace, never a raw memory directory or repository identity.
func ReadMaestroMemory(parent context.Context, workspace, taskID, search string, limit int) (MaestroMemoryReadResult, error) {
	var zero MaestroMemoryReadResult
	if strings.TrimSpace(workspace) == "" {
		return zero, errors.New("maestro memory workspace is required")
	}
	if taskID != "" && !maestroMemoryTaskID.MatchString(taskID) {
		return zero, errors.New("invalid Maestro memory task identifier")
	}
	if len(search) > 200 {
		return zero, errors.New("maestro memory query exceeds 200 characters")
	}
	if limit == 0 {
		limit = 20
	}
	if limit < 1 || limit > maestroMemoryMaxLimit {
		return zero, errors.New("maestro memory limit must be between 1 and 50")
	}
	params := map[string]any{"limit": limit, "offset": 0}
	if taskID != "" {
		params["taskId"] = taskID
	}
	if search != "" {
		params["search"] = security.Redact(search)
	}
	result, _, err := maestroBridgeCall(parent, workspace, 5*time.Second, maestroMemoryMethod, params)
	if err != nil {
		return zero, fmt.Errorf("maestro memory request failed: %w", err)
	}
	// The shared transport verified initialize, protocol version, request IDs,
	// response boundaries and capability before returning this result. Adapt
	// to the existing DTO decoder without another subprocess or memory cache.
	wrapped, err := json.Marshal(struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      int             `json:"id"`
		Result  json.RawMessage `json:"result"`
	}{JSONRPC: "2.0", ID: 1, Result: result})
	if err != nil {
		return zero, err
	}
	return decodeMaestroMemorySearch(wrapped)
}

// maestroMemoryWriteMethod fails closed unless the same Bridge instance that
// will receive memory.record explicitly advertises canonical write support.
func maestroMemoryWriteMethod(init maestroMemoryWriteInit) (string, error) {
	if init.ProtocolVersion != maestroBridgeProtocolVersion {
		return "", fmt.Errorf("unsupported Maestro Bridge protocol version %d", init.ProtocolVersion)
	}
	if !init.Capabilities.Memory.Write {
		return "", errors.New("maestro Bridge has no canonical memory write capability")
	}
	return "memory.record", nil
}

func validateMaestroMemoryWriteRequest(workspace string, request MaestroMemoryWriteRequest) error {
	if strings.TrimSpace(workspace) == "" {
		return errors.New("maestro memory workspace is required")
	}
	if !request.Confirmed {
		return errors.New("explicit operator confirmation is required for Maestro memory write")
	}
	if !utf8.ValidString(request.Reason) || !utf8.ValidString(request.Summary) ||
		!utf8.ValidString(request.Details) {
		return errors.New("Maestro memory write text must be valid UTF-8")
	}
	reasonLength := maestroMemoryUTF16Length(strings.TrimSpace(request.Reason))
	if reasonLength < 8 || maestroMemoryUTF16Length(request.Reason) > 200 {
		return errors.New("Maestro memory reason must be 8 to 200 characters")
	}
	if !maestroMemoryTaskID.MatchString(request.IdempotencyKey) {
		return errors.New("Maestro memory idempotency key must be a stable 1-128 character identifier")
	}
	if !maestroMemoryObservationTypes[request.Type] {
		return errors.New("unknown Maestro memory observation type")
	}
	summaryLength := maestroMemoryUTF16Length(strings.TrimSpace(request.Summary))
	if summaryLength < 1 || maestroMemoryUTF16Length(request.Summary) > 500 {
		return errors.New("Maestro memory summary must be 1 to 500 characters")
	}
	if maestroMemoryUTF16Length(request.Details) > 4000 {
		return errors.New("Maestro memory details must be at most 4000 characters")
	}
	if !maestroMemoryScopeLevels[request.ScopeLevel] {
		return errors.New("Maestro memory scope level must be repository, branch, workspace, or task")
	}
	if request.TaskID != "" && !maestroMemoryTaskID.MatchString(request.TaskID) {
		return errors.New("invalid Maestro memory task identifier")
	}
	if request.ScopeLevel == "task" && request.TaskID == "" {
		return errors.New("Maestro memory task scope requires taskId")
	}
	if err := validateMaestroMemoryWriteItems("tags", request.Tags, 128); err != nil {
		return err
	}
	if err := validateMaestroMemoryWriteItems("files", request.Files, 512); err != nil {
		return err
	}
	if request.ProjectPath != "" {
		workspacePath, err := filepath.Abs(workspace)
		if err != nil {
			return errors.New("invalid Maestro memory workspace path")
		}
		projectPath, err := filepath.Abs(request.ProjectPath)
		if err != nil {
			return errors.New("invalid Maestro memory project path")
		}
		if filepath.Clean(projectPath) != filepath.Clean(workspacePath) {
			return errors.New("Maestro memory project path must equal workspace")
		}
	}
	return nil
}

func validateMaestroMemoryWriteItems(name string, items []string, maxLength int) error {
	if len(items) > 16 {
		return fmt.Errorf("Maestro memory %s exceeds 16 items", name)
	}
	for _, item := range items {
		if !utf8.ValidString(item) {
			return fmt.Errorf("invalid Maestro memory %s item", name)
		}
		length := maestroMemoryUTF16Length(item)
		if length < 1 || length > maxLength {
			return fmt.Errorf("invalid Maestro memory %s item", name)
		}
	}
	return nil
}

// maestroMemoryUTF16Length matches JavaScript String.length, which is the unit
// used by the canonical Bridge limits.
func maestroMemoryUTF16Length(value string) int {
	return len(utf16.Encode([]rune(value)))
}

func decodeMaestroMemoryWriteResult(raw json.RawMessage) (MaestroMemoryWriteResult, error) {
	var result MaestroMemoryWriteResult
	if len(raw) == 0 || len(raw) > maestroMemoryMaxResponse {
		return result, errors.New("Maestro memory write response exceeds safety budget")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&result); err != nil {
		return MaestroMemoryWriteResult{}, errors.New("invalid Maestro memory write response")
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return MaestroMemoryWriteResult{}, errors.New("invalid Maestro memory write response")
	}
	if result.SchemaVersion != 1 || result.Authority != "maestro" ||
		(result.Status != "RECORDED" && result.Status != "REPLAYED") ||
		!maestroMemoryObservationID.MatchString(result.ID) {
		return MaestroMemoryWriteResult{}, errors.New("invalid Maestro memory write acknowledgement")
	}
	if !maestroMemoryScopeLevels[result.Scope.Level] || result.Scope.RepositoryID == "" {
		return MaestroMemoryWriteResult{}, errors.New("invalid Maestro memory write scope")
	}
	switch result.Scope.Level {
	case "branch":
		if result.Scope.Branch == "" {
			return MaestroMemoryWriteResult{}, errors.New("invalid Maestro memory write scope")
		}
	case "workspace":
		if result.Scope.WorkspaceID == "" {
			return MaestroMemoryWriteResult{}, errors.New("invalid Maestro memory write scope")
		}
	case "task":
		if result.Scope.TaskID == "" {
			return MaestroMemoryWriteResult{}, errors.New("invalid Maestro memory write scope")
		}
	}
	result.Scope.RepositoryID = security.Redact(result.Scope.RepositoryID)
	result.Scope.Branch = security.Redact(result.Scope.Branch)
	result.Scope.WorkspaceID = security.Redact(result.Scope.WorkspaceID)
	result.Scope.TaskID = security.Redact(result.Scope.TaskID)
	result.Scope.HeadCommit = security.Redact(result.Scope.HeadCommit)
	return result, nil
}

func validateMaestroMemoryWriteScope(result MaestroMemoryWriteResult, request MaestroMemoryWriteRequest) error {
	if result.Scope.Level != request.ScopeLevel {
		return errors.New("Maestro memory write scope level does not match request")
	}
	if request.ScopeLevel == "task" && result.Scope.TaskID != security.Redact(request.TaskID) {
		return errors.New("Maestro memory write task scope does not match request")
	}
	return nil
}

// maestroMemoryWriteBridgeCall follows the existing maestroBridgeCall
// transport pattern, but keeps the write capability handshake and the mutation
// in one process. It never falls back to a Nexus-owned memory store.
func maestroMemoryWriteBridgeCall(parent context.Context, workspace string, timeout time.Duration,
	params map[string]any) (json.RawMessage, error) {
	if strings.TrimSpace(workspace) == "" {
		return nil, errors.New("maestro project workspace is required")
	}
	binary, err := runtime.LookPath("orquestrador-maestro")
	if err != nil {
		return nil, fmt.Errorf("maestro CLI unavailable: %w", err)
	}
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()
	cmd := executil.CommandContext(ctx, binary, "bridge", "--stdio", "--project-path", workspace)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("could not start Maestro Bridge: %w", err)
	}
	waited := false
	defer func() {
		_ = stdin.Close()
		if !waited {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	}()
	reader := bufio.NewReader(io.LimitReader(stdout, 2*maestroBridgeReplyLimit+1))
	send := func(id int, method string, args map[string]any) (json.RawMessage, error) {
		request := struct {
			JSONRPC string         `json:"jsonrpc"`
			ID      int            `json:"id"`
			Method  string         `json:"method"`
			Params  map[string]any `json:"params"`
		}{JSONRPC: "2.0", ID: id, Method: method, Params: args}
		payload, err := json.Marshal(request)
		if err != nil {
			return nil, fmt.Errorf("maestro Bridge request encoding: %w", err)
		}
		if len(payload) > maestroBridgeReplyLimit {
			return nil, errors.New("maestro Bridge request exceeds limit")
		}
		if _, err := stdin.Write(append(payload, '\n')); err != nil {
			return nil, fmt.Errorf("maestro Bridge request failed: %w", err)
		}
		return readMaestroBridgeReply(reader, id)
	}
	initResult, err := send(1, "initialize", map[string]any{"protocolVersion": maestroBridgeProtocolVersion})
	if err != nil {
		return nil, err
	}
	var init maestroMemoryWriteInit
	if err := json.Unmarshal(initResult, &init); err != nil {
		return nil, errors.New("invalid Maestro Bridge initialize result")
	}
	method, err := maestroMemoryWriteMethod(init)
	if err != nil {
		return nil, err
	}
	result, err := send(2, method, params)
	if err != nil {
		return nil, err
	}
	if err := stdin.Close(); err != nil {
		return nil, fmt.Errorf("maestro Bridge stdin closure failed: %w", err)
	}
	if err := cmd.Wait(); err != nil {
		waited = true
		return nil, fmt.Errorf("maestro Bridge exited unsuccessfully: %w", err)
	}
	waited = true
	return result, nil
}

// WriteMaestroMemory performs one bounded, explicitly confirmed write through
// Maestro's canonical memory. The returned acknowledgement is ephemeral and
// no observation content, request, or idempotency mapping is stored in Nexus.
func WriteMaestroMemory(parent context.Context, workspace string, request MaestroMemoryWriteRequest) (MaestroMemoryWriteResult, error) {
	if err := validateMaestroMemoryWriteRequest(workspace, request); err != nil {
		return MaestroMemoryWriteResult{}, err
	}
	params := map[string]any{
		"confirmed":      true,
		"reason":         request.Reason,
		"idempotencyKey": request.IdempotencyKey,
		"type":           request.Type,
		"summary":        request.Summary,
		"scopeLevel":     request.ScopeLevel,
	}
	if request.Details != "" {
		params["details"] = request.Details
	}
	if request.TaskID != "" {
		params["taskId"] = request.TaskID
	}
	if len(request.Tags) > 0 {
		params["tags"] = request.Tags
	}
	if len(request.Files) > 0 {
		params["files"] = request.Files
	}
	if request.ProjectPath != "" {
		params["projectPath"] = request.ProjectPath
	}
	result, err := maestroMemoryWriteBridgeCall(parent, workspace, 10*time.Second, params)
	if err != nil {
		return MaestroMemoryWriteResult{}, fmt.Errorf("Maestro memory write failed: %w", err)
	}
	decoded, err := decodeMaestroMemoryWriteResult(result)
	if err != nil {
		return MaestroMemoryWriteResult{}, err
	}
	if err := validateMaestroMemoryWriteScope(decoded, request); err != nil {
		return MaestroMemoryWriteResult{}, err
	}
	return decoded, nil
}

func decodeMaestroMemorySearch(raw []byte) (MaestroMemoryReadResult, error) {
	var zero MaestroMemoryReadResult
	if len(raw) > maestroMemoryMaxResponse {
		return zero, errors.New("maestro memory response exceeds safety budget")
	}
	var reply maestroMemoryRPCReply
	if err := json.Unmarshal(bytes.TrimSpace(raw), &reply); err != nil {
		return zero, fmt.Errorf("invalid Maestro memory response: %w", err)
	}
	if reply.JSONRPC != "2.0" || reply.ID != 1 {
		return zero, errors.New("maestro memory response identity mismatch")
	}
	if reply.Error != nil {
		return zero, fmt.Errorf("maestro memory method unavailable (%d)", reply.Error.Code)
	}
	if reply.Result == nil || reply.Result.SchemaVersion != 1 || reply.Result.Authority != "maestro" {
		return zero, errors.New("unsupported or unverified Maestro memory contract")
	}
	result := *reply.Result
	if result.Page.Limit < 1 || result.Page.Limit > maestroMemoryMaxLimit ||
		result.Page.Offset < 0 || len(result.Items) > result.Page.Limit {
		return zero, errors.New("maestro memory pagination invalid")
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
