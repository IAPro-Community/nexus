package handoff

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func validMaestroRunObservation() json.RawMessage {
	return json.RawMessage(`{"schemaVersion":1,"authority":"MAESTRO","executionOwner":"MAESTRO","runId":"run-7","taskId":"task-1","providerId":"agy","status":"running","replayKind":"sanitized-lifecycle-only","liveOutputAvailable":false,"events":[{"id":"event-1","type":"run.started","occurredAt":"2026-11-02T20:00:00Z"}],"nextCursor":"event-1","hasMore":false}`)
}

func TestMaestroRunObservationCapabilityNegotiation(t *testing.T) {
	var init maestroBridgeInit
	init.ProtocolVersion = 1
	if _, err := maestroObservationMethod(init); err == nil {
		t.Fatal("older Maestro cannot be claimed to support run observation")
	}
	init.Capabilities.RunsObserve = true
	method, err := maestroObservationMethod(init)
	if err != nil || method != "runs.observe" {
		t.Fatalf("unexpected method %q: %v", method, err)
	}
	init.ProtocolVersion = 2
	if _, err := maestroObservationMethod(init); err == nil {
		t.Fatal("unknown protocol should be refused")
	}
}

func TestMaestroRunObservationPreservesRunAndSanitizesLifecycle(t *testing.T) {
	got, err := decodeMaestroRunObservation(validMaestroRunObservation(), "run-7", 2)
	if err != nil { t.Fatal(err) }
	if got.ExecutionOwner != "MAESTRO" || got.ProviderID == nil || *got.ProviderID != "agy" ||
		got.RunID != "run-7" || got.Status != "running" || got.LiveOutputAvailable ||
		len(got.Events) != 1 || got.Events[0].Type != "run.started" {
		t.Fatalf("bad Maestro observation %+v", got)
	}
	if strings.Contains(string(validMaestroRunObservation()), "provider.output") {
		t.Fatal("fixture must not contain provider output")
	}
}

func TestMaestroRunObservationRejectsForeignIdentityAndExtraOutput(t *testing.T) {
	original := string(validMaestroRunObservation())
	cases := []string{
		strings.Replace(original, `"executionOwner":"MAESTRO"`, `"executionOwner":"NEXUS"`, 1),
		strings.Replace(original, `"authority":"MAESTRO"`, `"authority":"NEXUS"`, 1),
		strings.Replace(original, `"runId":"run-7"`, `"runId":"run-other"`, 1),
		strings.Replace(original, `"liveOutputAvailable":false`, `"liveOutputAvailable":true`, 1),
		strings.Replace(original, `"type":"run.started"`, `"type":"provider.output"`, 1),
		strings.Replace(original, `"nextCursor":"event-1"`, `"nextCursor":"stale"`, 1),
		strings.Replace(original, `"occurredAt":"2026-11-02T20:00:00Z"`, `"occurredAt":"secret-value"`, 1),
		strings.Replace(original, `"hasMore":false`, `"hasMore":true,"output":"API_KEY=private"`, 1),
		original + `{"unexpected":true}`,
	}
	for i, raw := range cases {
		if _, err := decodeMaestroRunObservation(json.RawMessage(raw), "run-7", 2); err == nil {
			t.Errorf("case %d accepted unsafe run observation", i)
		}
	}
	if _, err := decodeMaestroRunObservation(validMaestroRunObservation(), "run-7", 0); err == nil {
		t.Fatal("oversized response should fail requested limit")
	}
}

func TestMaestroRunObservationRejectsInvalidInputBeforeStartingProcess(t *testing.T) {
	for _, tc := range []struct { runID, cursor string; limit int }{
		{"../other", "", 10},
		{"run-1", "../other", 10},
		{"run-1", "", 0},
		{"run-1", "", 101},
	} {
		_, err := FetchMaestroRunObservation(t.Context(), "/not-a-real-project", tc.runID, tc.cursor, tc.limit)
		if err == nil { t.Fatal(fmt.Sprintf("invalid request accepted: %+v", tc)) }
	}
}
