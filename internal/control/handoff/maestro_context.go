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
	"unicode/utf16"

	"github.com/IAPro-Community/nexus/internal/core/security"
	"github.com/IAPro-Community/nexus/internal/executil"
	"github.com/IAPro-Community/nexus/internal/runtime"
)

const maestroContextBudget = 8000

type maestroContextReply struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Result  struct {
		Content  string `json:"content"`
		Used     int    `json:"used"`
		Manifest struct {
			Version       int    `json:"version"`
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
		return "", fmt.Errorf("maestro CLI unavailable: %w", err)
	}
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()
	params := map[string]any{"maxChars": maestroContextBudget}
	if task != "" {
		runes := []rune(security.Redact(task))
		if len(runes) > 400 {
			runes = runes[:400]
		}
		params["task"] = string(runes)
	}
	request, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": "context.brief", "params": params,
	})
	if err != nil {
		return "", err
	}
	cmd := executil.CommandContext(ctx, binary, "bridge", "--stdio", "--project-path", workspace)
	cmd.Stdin = bytes.NewReader(append(request, '\n'))
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("maestro context request failed: %w", err)
	}
	return decodeMaestroContextBrief(out)
}

func decodeMaestroContextBrief(raw []byte) (string, error) {
	if len(raw) > 128*1024 {
		return "", errors.New("maestro response too large")
	}
	var response maestroContextReply
	if err := json.Unmarshal(bytes.TrimSpace(raw), &response); err != nil {
		return "", fmt.Errorf("invalid Maestro JSON-RPC response: %w", err)
	}
	if response.JSONRPC != "2.0" || response.ID != 1 {
		return "", errors.New("maestro response identity mismatch")
	}
	if response.Error != nil {
		return "", fmt.Errorf("maestro context method unavailable (%d)", response.Error.Code)
	}
	r := response.Result
	// Maestro's JavaScript builder counts UTF-16 code units, not UTF-8 bytes
	// or Go runes. Reject a forged budget report before accepting the digest.
	actualUnits := len(utf16.Encode([]rune(r.Content)))
	if r.Manifest.Version != 1 || r.Content == "" || r.Used < 1 ||
		r.Used != actualUnits || r.Used > maestroContextBudget ||
		len(r.Content) > maestroContextBudget*4 {
		return "", errors.New("maestro context budget or manifest invalid")
	}
	digest := sha256.Sum256([]byte(r.Content))
	if !strings.EqualFold(r.Manifest.ContentDigest, hex.EncodeToString(digest[:])) {
		return "", errors.New("maestro context digest mismatch")
	}
	return security.Redact(r.Content), nil
}
