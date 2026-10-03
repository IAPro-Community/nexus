package nexus

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/IAPro-Community/nexus/internal/control/computer"
	"github.com/IAPro-Community/nexus/internal/core/security"
)

const maxComputerCoordinationOutputBytes = 64 << 10

type computerCoordinationSnapshot struct {
	ID                 string              `json:"id"`
	App                computer.AppInfo    `json:"app"`
	Window             computer.WindowInfo `json:"window"`
	Elements           []computer.Element  `json:"elements,omitempty"`
	TreeText           string              `json:"tree_text,omitempty"`
	ScreenshotCaptured bool                `json:"screenshot_captured,omitempty"`
	ScreenshotRef      string              `json:"screenshot_ref,omitempty"`
	CapturedAt         string              `json:"captured_at"`
	Truncated          bool                `json:"truncated,omitempty"`
}

type computerCoordinationActionResult struct {
	Snapshot     computerCoordinationSnapshot `json:"snapshot"`
	Verification computer.ActionVerification  `json:"verification"`
}

func decodeComputerCoordinationPayload(raw json.RawMessage, out any) error {
	if len(raw) == 0 {
		return fmt.Errorf("computer coordination tool_payload is required")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		return fmt.Errorf("decode computer coordination tool_payload: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err == nil {
		return fmt.Errorf("computer coordination tool_payload contains trailing JSON")
	}
	return nil
}

func validateComputerCoordinationPayloadSecrets(raw json.RawMessage) error {
	text := string(raw)
	if security.Redact(text) != text {
		return fmt.Errorf("computer coordination tool_payload appears to contain secret material")
	}
	return nil
}

func computerSnapshotCoordinationOutput(snapshot *computer.Snapshot) (string, error) {
	if snapshot == nil {
		return "", fmt.Errorf("computer snapshot is unavailable")
	}
	projected := projectComputerSnapshotForCoordination(*snapshot)
	return encodeBoundedComputerCoordination(projected)
}

func computerActionCoordinationOutput(result *computer.ActionResult) (string, error) {
	if result == nil {
		return "", fmt.Errorf("computer action result is unavailable")
	}
	projected := computerCoordinationActionResult{
		Snapshot:     projectComputerSnapshotForCoordination(result.Snapshot),
		Verification: result.Verification,
	}
	return encodeBoundedComputerCoordination(projected)
}

func projectComputerSnapshotForCoordination(snapshot computer.Snapshot) computerCoordinationSnapshot {
	const maxElements = 64
	projected := computerCoordinationSnapshot{
		ID:                 snapshot.ID,
		App:                snapshot.App,
		Window:             snapshot.Window,
		TreeText:           boundComputerCoordinationText(snapshot.TreeText, 8192),
		ScreenshotCaptured: strings.TrimSpace(snapshot.Screenshot) != "",
		CapturedAt:         snapshot.CapturedAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"),
	}
	if projected.ScreenshotCaptured {
		projected.ScreenshotRef = filepath.Base(snapshot.Screenshot)
	}
	limit := len(snapshot.Elements)
	if limit > maxElements {
		limit = maxElements
		projected.Truncated = true
	}
	projected.Elements = make([]computer.Element, 0, limit)
	for i := 0; i < limit; i++ {
		element := snapshot.Elements[i]
		element.Name = boundComputerCoordinationText(element.Name, 256)
		element.Value = boundComputerCoordinationText(element.Value, 512)
		if len(element.Actions) > 12 {
			element.Actions = append([]string(nil), element.Actions[:12]...)
			projected.Truncated = true
		} else {
			element.Actions = append([]string(nil), element.Actions...)
		}
		projected.Elements = append(projected.Elements, element)
	}
	if projected.TreeText != snapshot.TreeText {
		projected.Truncated = true
	}
	return projected
}

func encodeBoundedComputerCoordination(value any) (string, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("encode computer coordination result: %w", err)
	}
	if len(raw) <= maxComputerCoordinationOutputBytes {
		return string(raw), nil
	}

	// The semantic projection should normally stay under the ceiling. If a
	// provider still returns unexpectedly large content, fail closed rather
	// than silently handing an unbounded context blob to the next Agent turn.
	return "", fmt.Errorf(
		"computer coordination result exceeds %d bytes after semantic projection",
		maxComputerCoordinationOutputBytes,
	)
}

func boundComputerCoordinationText(value string, maxBytes int) string {
	if maxBytes <= 0 || len(value) <= maxBytes {
		return value
	}
	candidate := value[:maxBytes]
	for !strings.ValidUTF8(candidate) && len(candidate) > 0 {
		candidate = candidate[:len(candidate)-1]
	}
	return candidate + "…"
}
