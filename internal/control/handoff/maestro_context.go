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
	"unicode/utf16"

	"github.com/IAPro-Community/nexus/internal/core/security"
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

// fetchMaestroContextBrief is retained for the handoff call site, but now
// negotiates the canonical project.context method before accepting a bounded
// context.brief result from a compatible older external Maestro installation.
func fetchMaestroContextBrief(parent context.Context, workspace, task string) (string, error) {
	projectContext, err := FetchMaestroProjectContext(parent, workspace, task)
	if err != nil {
		return "", err
	}
	return projectContext.Context, nil
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
