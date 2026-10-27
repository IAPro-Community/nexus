package handoff

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/security"
	"github.com/IAPro-Community/nexus/internal/executil"
	"github.com/IAPro-Community/nexus/internal/runtime"
)

const maestroContextBudget = 8000

type maestroContextReply struct {
	JSONRPC string `json:"jsonrpc"`
	ID int `json:"id"`
	Result struct {
		Content string `json:"content"`
		Used int `json:"used"`
		Manifest struct {
			Version int `json:"version"`
			ContentDigest string `json:"contentDigest"`
		} `json:"manifest"`
	} `json:"result"`
	Error *struct {
		Code int `json:"code"`
	} `json:"error"`
}

// A read-only Maestro Bridge request; semantic content is kept solely in
// the ephemeral handoff prompt, not copied into the Nexus checkpoint store.
func fetchMaestroContextBrief(parent context.Context, workspace, task string) (string, error) {
	binary, err := runtime.LookPath("orquestrador-maestro")
	if err != nil {
		return "", fmt.Errorf("Maestro CLI unavailable: %w", err)
	}
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()
	params := map[string]any{"maxChars": maestroContextBudget}
	if task != "" {
		runes := []rune(security.Redact(task))
		if len(runes) > 400 { runes = runes[:400] }
		params["task"] = string(runes)
	}
	request, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": "context.brief", "params": params,
	})
	if err != nil { return "", err }
	cmd := executil.CommandContext(ctx, binary, "bridge", "--stdio", "--project-path", workspace)
	cmd.Stdin = bytes.NewReader(append(request, '\n'))
	out, err := cmd.Output()
	if err != nil { return "", fmt.Errorf("Maestro context request failed: %w", err) }
	return decodeMaestroContextBrief(out, maestroContextBudget)
}

func decodeMaestroContextBrief(raw []byte, maxChars int) (string, error) {
	if len(raw) > 128*1024 { return "", errors.New("Maestro response too large") }
	var response maestroContextReply
	if err := json.Unmarshal(bytes.TrimSpace(raw), &response); err != nil {
		return "", fmt.Errorf("invalid Maestro JSON-RPC response: %w", err)
	}
	if response.JSONRPC != "2.0" || response.ID != 1 {
		return "", errors.New("Maestro response identity mismatch")
	}
	if response.Error != nil {
		return "", fmt.Errorf("Maestro context method unavailable (%d)", response.Error.Code)
	}
	r := response.Result
	if r.Manifest.Version != 1 || r.Content == "" || r.Used < 1 ||
		r.Used > maxChars || len([]rune(r.Content)) > maxChars ||
		len(r.Content) > maxChars*4 {
		return "", errors.New("Maestro context budget or manifest invalid")
	}
	digest := sha256.Sum256([]byte(r.Content))
	if !strings.EqualFold(r.Manifest.ContentDigest, hex.EncodeToString(digest[:])) {
		return "", errors.New("Maestro context digest mismatch")
	}
	return security.Redact(r.Content), nil
}
