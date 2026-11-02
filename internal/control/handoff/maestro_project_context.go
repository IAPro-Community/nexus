package handoff

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/security"
	"github.com/IAPro-Community/nexus/internal/executil"
	"github.com/IAPro-Community/nexus/internal/runtime"
)

const maestroBridgeProtocolVersion = 1
const maestroBridgeReplyLimit = 128 * 1024

// MaestroProjectIdentity is an ephemeral projection from Maestro, not a
// persisted copy of its project registry. A legacy Bridge has no identity.
type MaestroProjectIdentity struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Known  bool   `json:"known"`
	Status string `json:"status"`
}

// MaestroProjectContext is a bounded, verified read from a single project
// Bridge process. It is not a semantic memory store or a composition engine.
type MaestroProjectContext struct {
	Context string
	Method  string
	Project *MaestroProjectIdentity
}

type maestroBridgeEnvelope struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int            `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *struct {
		Code int `json:"code"`
	} `json:"error"`
}

type maestroBridgeInit struct {
	ProtocolVersion int `json:"protocolVersion"`
	Capabilities    struct {
		ProjectContext bool `json:"projectContext"`
		ContextBrief   bool `json:"contextBrief"`
	} `json:"capabilities"`
}

func maestroContextMethod(init maestroBridgeInit) (string, error) {
	if init.ProtocolVersion != maestroBridgeProtocolVersion {
		return "", fmt.Errorf("unsupported Maestro Bridge protocol version %d", init.ProtocolVersion)
	}
	if init.Capabilities.ProjectContext {
		return "project.context", nil
	}
	if init.Capabilities.ContextBrief {
		return "context.brief", nil
	}
	return "", errors.New("Maestro Bridge does not provide canonical project context")
}

// decodeMaestroProjectResult accepts only the versioned Maestro projection.
// For older Maestro installations, context.brief is explicitly negotiated and
// the client intentionally does NOT fabricate project identity.
func decodeMaestroProjectResult(method string, result json.RawMessage) (MaestroProjectContext, error) {
	if method != "project.context" && method != "context.brief" {
		return MaestroProjectContext{}, errors.New("unsupported Maestro context method")
	}
	payload := result
	var project *MaestroProjectIdentity
	if method == "project.context" {
		var projected struct {
			SchemaVersion int                     `json:"schemaVersion"`
			Authority     string                  `json:"authority"`
			Project       *MaestroProjectIdentity `json:"project"`
			Context       json.RawMessage         `json:"context"`
		}
		if err := json.Unmarshal(result, &projected); err != nil ||
			projected.SchemaVersion != 1 || projected.Authority != "MAESTRO" ||
			projected.Project == nil || len(projected.Context) == 0 {
			return MaestroProjectContext{}, errors.New("invalid canonical Maestro project projection")
		}
		if projected.Project.Known && strings.TrimSpace(projected.Project.ID) == "" {
			return MaestroProjectContext{}, errors.New("known Maestro project has no identity")
		}
		project = projected.Project
		payload = projected.Context
	}
	// Reuse the existing digest, UTF-16 budget, version and redaction verifier.
	// A locally constructed envelope only adapts the verified result to the
	// older decoder; it never changes the content or its digest.
	wrapped, err := json.Marshal(struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      int             `json:"id"`
		Result  json.RawMessage `json:"result"`
	}{JSONRPC: "2.0", ID: 1, Result: payload})
	if err != nil {
		return MaestroProjectContext{}, err
	}
	brief, err := decodeMaestroContextBrief(wrapped)
	if err != nil {
		return MaestroProjectContext{}, err
	}
	return MaestroProjectContext{Context: brief, Method: method, Project: project}, nil
}

func readMaestroBridgeReply(reader *bufio.Reader, expectedID int) (json.RawMessage, error) {
	// ReadString alone is unbounded. The supplied reader must be a limited
	// reader, and each response is also individually size checked.
	line, err := reader.ReadBytes('\n')
	if err != nil {
		return nil, fmt.Errorf("Maestro Bridge response incomplete: %w", err)
	}
	if len(line) > maestroBridgeReplyLimit {
		return nil, errors.New("Maestro Bridge response exceeds limit")
	}
	var reply maestroBridgeEnvelope
	if err := json.Unmarshal(line, &reply); err != nil {
		return nil, errors.New("invalid Maestro Bridge JSON response")
	}
	if reply.JSONRPC != "2.0" || reply.ID == nil || *reply.ID != expectedID {
		return nil, errors.New("Maestro Bridge response identity mismatch")
	}
	if reply.Error != nil {
		return nil, fmt.Errorf("Maestro Bridge method unavailable (%d)", reply.Error.Code)
	}
	if len(reply.Result) == 0 || string(reply.Result) == "null" {
		return nil, errors.New("empty Maestro Bridge result")
	}
	return reply.Result, nil
}

// FetchMaestroProjectContext reads canonical project context from the external
// Maestro executable. Both calls use ONE project-scoped Bridge process. There
// is no fallback to Nexus semantic memory, provider calls, or a cloud service.
func FetchMaestroProjectContext(parent context.Context, workspace, task string) (MaestroProjectContext, error) {
	if strings.TrimSpace(workspace) == "" {
		return MaestroProjectContext{}, errors.New("Maestro project workspace is required")
	}
	binary, err := runtime.LookPath("orquestrador-maestro")
	if err != nil {
		return MaestroProjectContext{}, fmt.Errorf("Maestro CLI unavailable: %w", err)
	}

	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()
	cmd := executil.CommandContext(ctx, binary, "bridge", "--stdio", "--project-path", workspace)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return MaestroProjectContext{}, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return MaestroProjectContext{}, err
	}
	if err := cmd.Start(); err != nil {
		return MaestroProjectContext{}, fmt.Errorf("could not start Maestro Bridge: %w", err)
	}
	waited := false
	defer func() {
		_ = stdin.Close()
		if !waited {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	}()

	// Include the buffer read-ahead in the bounded stream budget.
	reader := bufio.NewReader(io.LimitReader(stdout, 2*maestroBridgeReplyLimit+1))
	send := func(id int, method string, params map[string]any) (json.RawMessage, error) {
		request := struct {
			JSONRPC string         `json:"jsonrpc"`
			ID      int            `json:"id"`
			Method  string         `json:"method"`
			Params  map[string]any `json:"params"`
		}{JSONRPC: "2.0", ID: id, Method: method, Params: params}
		if err := json.NewEncoder(stdin).Encode(request); err != nil {
			return nil, fmt.Errorf("Maestro Bridge request failed: %w", err)
		}
		return readMaestroBridgeReply(reader, id)
	}

	initResult, err := send(1, "initialize", map[string]any{"protocolVersion": maestroBridgeProtocolVersion})
	if err != nil {
		return MaestroProjectContext{}, err
	}
	var init maestroBridgeInit
	if err := json.Unmarshal(initResult, &init); err != nil {
		return MaestroProjectContext{}, errors.New("invalid Maestro Bridge initialize result")
	}
	method, err := maestroContextMethod(init)
	if err != nil {
		return MaestroProjectContext{}, err
	}
	params := map[string]any{"projectPath": workspace, "maxChars": maestroContextBudget}
	if task != "" {
		runes := []rune(security.Redact(task))
		if len(runes) > 400 {
			runes = runes[:400]
		}
		params["task"] = string(runes)
	}
	result, err := send(2, method, params)
	if err != nil {
		return MaestroProjectContext{}, err
	}
	projectContext, err := decodeMaestroProjectResult(method, result)
	if err != nil {
		return MaestroProjectContext{}, err
	}
	if err := stdin.Close(); err != nil {
		return MaestroProjectContext{}, fmt.Errorf("Maestro Bridge stdin closure failed: %w", err)
	}
	if err := cmd.Wait(); err != nil {
		waited = true
		return MaestroProjectContext{}, fmt.Errorf("Maestro Bridge exited unsuccessfully: %w", err)
	}
	waited = true
	return projectContext, nil
}
