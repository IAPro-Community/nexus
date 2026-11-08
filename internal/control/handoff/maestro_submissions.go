package handoff

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"time"
)

var maestroSubmissionCursor = regexp.MustCompile(`^s1:[a-f0-9]{64}$`)
var maestroSubmissionID = regexp.MustCompile(`^[A-Za-z0-9._:-]{8,128}$`)

// MaestroSubmissionSnapshot is a sanitized, read-only view of canonical Maestro
// submissions. It is a snapshot, not an event stream, scheduler or receipt.
type MaestroSubmissionSnapshot struct {
	SchemaVersion int                       `json:"schemaVersion"`
	Authority     string                    `json:"authority"`
	Owner         string                    `json:"owner"`
	Delivery      string                    `json:"delivery"`
	Replayable    bool                      `json:"replayable"`
	ProjectID     string                    `json:"projectId"`
	Cursor        string                    `json:"cursor"`
	Changed       bool                      `json:"changed"`
	Total         int                       `json:"total"`
	Offset        int                       `json:"offset"`
	Limit         int                       `json:"limit"`
	NextOffset    *int                      `json:"nextOffset"`
	Submissions   []MaestroSubmissionStatus `json:"submissions"`
}

type MaestroSubmissionStatus struct {
	SchemaVersion  int      `json:"schemaVersion"`
	Authority      string   `json:"authority"`
	Owner          string   `json:"owner"`
	SubmissionID   string   `json:"submissionId"`
	Status         string   `json:"status"`
	RunID          string   `json:"runId"`
	ManifestID     string   `json:"manifestId"`
	TaskCount      int      `json:"taskCount"`
	Providers      []string `json:"providers"`
	CreatedAt      string   `json:"createdAt"`
	UpdatedAt      string   `json:"updatedAt"`
	ResultEvidence string   `json:"resultEvidence"`
}

func maestroSubmissionPollMethod(init maestroBridgeInit) (string, error) {
	if init.ProtocolVersion != maestroBridgeProtocolVersion || !init.Capabilities.RunsSubmissionsPoll {
		return "", errors.New("maestro submission snapshot polling unavailable")
	}
	return "runs.submissions.poll", nil
}

func decodeMaestroSubmissionSnapshot(raw json.RawMessage, previous string, limit, offset int) (MaestroSubmissionSnapshot, error) {
	var result MaestroSubmissionSnapshot
	if len(raw) == 0 || len(raw) > maestroBridgeReplyLimit {
		return result, errors.New("maestro submission snapshot response exceeds limit")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&result); err != nil {
		return MaestroSubmissionSnapshot{}, errors.New("invalid Maestro submission snapshot")
	}
	valid := result.SchemaVersion == 1 && result.Authority == "MAESTRO" &&
		result.Owner == "MAESTRO" && result.Delivery == "SNAPSHOT_POLL" &&
		!result.Replayable && maestroSubmissionID.MatchString(result.ProjectID) &&
		maestroSubmissionCursor.MatchString(result.Cursor) &&
		result.Limit == limit && result.Offset == offset &&
		result.Total >= 0 && result.Total <= 1000000 &&
		result.Changed == (result.Cursor != previous) &&
		len(result.Submissions) <= limit
	if !valid {
		return MaestroSubmissionSnapshot{}, errors.New("inconsistent Maestro submission snapshot contract")
	}
	if result.NextOffset != nil {
		if *result.NextOffset != offset+limit || *result.NextOffset >= result.Total {
			return MaestroSubmissionSnapshot{}, errors.New("inconsistent Maestro submission pagination")
		}
	} else if offset+limit < result.Total {
		return MaestroSubmissionSnapshot{}, errors.New("missing Maestro submission pagination")
	}
	if !result.Changed && len(result.Submissions) != 0 {
		return MaestroSubmissionSnapshot{}, errors.New("unchanged Maestro snapshot included rows")
	}
	seen := make(map[string]struct{}, len(result.Submissions))
	statuses := map[string]bool{
		"PENDING_WORKER": true, "CLAIMED": true, "DISPATCHED": true,
		"CANCEL_REQUESTED": true, "RECONCILIATION_REQUIRED": true,
		"COMPLETED": true, "FAILED": true, "BLOCKED": true,
		"TIMED_OUT": true, "CANCELLED": true, //nolint:misspell // Canonical Maestro Bridge status.
	}
	for _, row := range result.Submissions {
		if row.SchemaVersion != 1 || row.Authority != "MAESTRO" || row.Owner != "MAESTRO" ||
			row.ResultEvidence != "RUN_STORE_OBSERVED" ||
			!maestroSubmissionID.MatchString(row.SubmissionID) || !statuses[row.Status] ||
			row.TaskCount < 1 || row.TaskCount > 32 || len(row.Providers) != row.TaskCount {
			return MaestroSubmissionSnapshot{}, errors.New("invalid Maestro submission row")
		}
		if _, exists := seen[row.SubmissionID]; exists {
			return MaestroSubmissionSnapshot{}, errors.New("duplicate Maestro submission row")
		}
		seen[row.SubmissionID] = struct{}{}
	}
	return result, nil
}

// FetchMaestroSubmissionSnapshot uses the canonical external Maestro Bridge.
// No background worker, memory, or execution owner is created in Nexus.
func FetchMaestroSubmissionSnapshot(parent context.Context, workspace, cursor string, limit, offset int) (MaestroSubmissionSnapshot, error) {
	if limit < 1 || limit > 100 || offset < 0 || offset > 1000000 ||
		(cursor != "" && !maestroSubmissionCursor.MatchString(cursor)) {
		return MaestroSubmissionSnapshot{}, errors.New("invalid Maestro snapshot cursor or pagination")
	}
	params := map[string]any{"limit": limit, "offset": offset}
	if cursor != "" {
		params["cursor"] = cursor
	}
	result, method, err := maestroBridgeCall(parent, workspace, 7*time.Second,
		maestroSubmissionPollMethod, params)
	if err != nil {
		return MaestroSubmissionSnapshot{}, fmt.Errorf("maestro submission snapshot unavailable: %w", err)
	}
	if method != "runs.submissions.poll" {
		return MaestroSubmissionSnapshot{}, errors.New("maestro submission method mismatch")
	}
	return decodeMaestroSubmissionSnapshot(result, cursor, limit, offset)
}
