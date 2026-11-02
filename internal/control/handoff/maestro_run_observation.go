package handoff

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"time"
)

var maestroRunID = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,128}$`)

type MaestroLifecycleEvent struct {
	ID         string  `json:"id"`
	Type       string  `json:"type"`
	OccurredAt *string `json:"occurredAt"`
}

type MaestroRunObservation struct {
	SchemaVersion       int                     `json:"schemaVersion"`
	Authority           string                  `json:"authority"`
	ExecutionOwner      string                  `json:"executionOwner"`
	RunID               string                  `json:"runId"`
	TaskID              string                  `json:"taskId"`
	ProviderID          *string                 `json:"providerId"`
	Status              string                  `json:"status"`
	ReplayKind          string                  `json:"replayKind"`
	LiveOutputAvailable bool                    `json:"liveOutputAvailable"`
	Events              []MaestroLifecycleEvent `json:"events"`
	NextCursor          *string                 `json:"nextCursor"`
	HasMore             bool                    `json:"hasMore"`
}

var maestroLifecycleEvents = map[string]struct{}{
	"run.created": {}, "run.started": {}, "run.completed": {},
	"run.failed": {}, "run.blocked": {}, "run.cancelled": {}, "run.canceled": {},
	"verification.started": {}, "verification.completed": {}, "verification.failed": {},
	"review.started": {}, "review.completed": {}, "review.failed": {},
	"execution.started": {}, "execution.completed": {}, "execution.failed": {},
	"artifact.created": {},
}

func maestroObservationMethod(init maestroBridgeInit) (string, error) {
	if init.ProtocolVersion != maestroBridgeProtocolVersion || !init.Capabilities.RunsObserve {
		return "", errors.New("external Maestro lacks durable sanitized run observation")
	}
	return "runs.observe", nil
}

func decodeMaestroRunObservation(raw json.RawMessage, runID string, limit int) (MaestroRunObservation, error) {
	var out MaestroRunObservation
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&out); err != nil {
		return MaestroRunObservation{}, errors.New("invalid or unexpected Maestro run observation fields")
	}
	var trailing any
	if err := dec.Decode(&trailing); err != io.EOF {
		return MaestroRunObservation{}, errors.New("unexpected trailing Maestro run result")
	}
	if out.SchemaVersion != 1 || out.Authority != "MAESTRO" || out.ExecutionOwner != "MAESTRO" ||
		out.RunID != runID || !maestroRunID.MatchString(out.TaskID) || out.Status == "" ||
		out.ReplayKind != "sanitized-lifecycle-only" || out.LiveOutputAvailable ||
		len(out.Events) > limit || len(out.Events) > 100 {
		return MaestroRunObservation{}, errors.New("Maestro run observation contract invalid")
	}
	seen := make(map[string]bool, len(out.Events))
	for _, event := range out.Events {
		if !maestroRunID.MatchString(event.ID) || seen[event.ID] {
			return MaestroRunObservation{}, errors.New("Maestro run event cursor invalid")
		}
		seen[event.ID] = true
		if event.OccurredAt != nil {
			if len(*event.OccurredAt) > 64 {
				return MaestroRunObservation{}, errors.New("oversized event time")
			}
			if _, err := time.Parse(time.RFC3339Nano, *event.OccurredAt); err != nil {
				return MaestroRunObservation{}, errors.New("invalid lifecycle event timestamp")
			}
		}
		if _, ok := maestroLifecycleEvents[event.Type]; !ok {
			return MaestroRunObservation{}, errors.New("untrusted Maestro lifecycle event")
		}
	}
	if out.NextCursor != nil && !maestroRunID.MatchString(*out.NextCursor) {
		return MaestroRunObservation{}, errors.New("invalid Maestro lifecycle cursor")
	}
	if len(out.Events) > 0 && (out.NextCursor == nil || *out.NextCursor != out.Events[len(out.Events)-1].ID) {
		return MaestroRunObservation{}, errors.New("run cursor does not match last returned lifecycle event")
	}
	if out.HasMore && len(out.Events) == 0 {
		return MaestroRunObservation{}, errors.New("Maestro run claims additional events without a cursorable page")
	}
	return out, nil
}

// FetchMaestroRunObservation is a scoped, read-only projection from an external
// Maestro. It cannot start, restart, retry or cancel a delegated run.
func FetchMaestroRunObservation(parent context.Context, workspace, runID, afterCursor string, limit int) (MaestroRunObservation, error) {
	if !maestroRunID.MatchString(runID) || (afterCursor != "" && !maestroRunID.MatchString(afterCursor)) ||
		limit < 1 || limit > 100 {
		return MaestroRunObservation{}, errors.New("invalid Maestro run observation input")
	}
	params := map[string]any{"runId": runID, "limit": limit}
	if afterCursor != "" {
		params["afterEventId"] = afterCursor
	}
	result, method, err := maestroBridgeCall(parent, workspace, 10*time.Second, maestroObservationMethod, params)
	if err != nil {
		return MaestroRunObservation{}, fmt.Errorf("Maestro run observation unavailable: %w", err)
	}
	if method != "runs.observe" {
		return MaestroRunObservation{}, errors.New("unexpected Maestro observation method")
	}
	return decodeMaestroRunObservation(result, runID, limit)
}
