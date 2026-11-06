package handoff

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestMaestroSubmissionPollCapabilityFailClosed(t *testing.T) {
	var init maestroBridgeInit
	init.ProtocolVersion = maestroBridgeProtocolVersion
	if _, err := maestroSubmissionPollMethod(init); err == nil {
		t.Fatal("unsupported Maestro capability accepted")
	}
	init.Capabilities.RunsSubmissionsPoll = true
	if method, err := maestroSubmissionPollMethod(init); err != nil || method != "runs.submissions.poll" {
		t.Fatalf("valid polling capability rejected: %q %v", method, err)
	}
	init.ProtocolVersion++
	if _, err := maestroSubmissionPollMethod(init); err == nil {
		t.Fatal("incompatible protocol accepted")
	}
}

func TestMaestroSubmissionSnapshotValidation(t *testing.T) {
	const cursor = "s1:" + "" // replaced by a valid 64-char digest below
	validCursor := cursor + strings.Repeat("a", 64)
	row := MaestroSubmissionStatus{
		SchemaVersion: 1, Authority: "MAESTRO", Owner: "MAESTRO",
		SubmissionID: "submission-test-001", Status: "DISPATCHED",
		TaskCount: 1, Providers: []string{"opencode"},
		ResultEvidence: "RUN_STORE_OBSERVED",
	}
	valid := MaestroSubmissionSnapshot{
		SchemaVersion: 1, Authority: "MAESTRO", Owner: "MAESTRO",
		Delivery: "SNAPSHOT_POLL", Replayable: false,
		ProjectID: "project-test-001", Cursor: validCursor,
		Changed: true, Limit: 2, Offset: 0, Total: 1,
		Submissions: []MaestroSubmissionStatus{row},
	}
	encode := func(input MaestroSubmissionSnapshot) json.RawMessage {
		data, err := json.Marshal(input)
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	if _, err := decodeMaestroSubmissionSnapshot(encode(valid), "", 2, 0); err != nil {
		t.Fatalf("valid snapshot rejected: %v", err)
	}
	tests := map[string]func(*MaestroSubmissionSnapshot){
		"forged authority":         func(s *MaestroSubmissionSnapshot) { s.Authority = "NEXUS" },
		"unsupported event replay": func(s *MaestroSubmissionSnapshot) { s.Replayable = true },
		"missing cursor":           func(s *MaestroSubmissionSnapshot) { s.Cursor = "" },
		"contradictory no-change":  func(s *MaestroSubmissionSnapshot) { s.Changed = false },
		"duplicate rows":           func(s *MaestroSubmissionSnapshot) { s.Submissions = append(s.Submissions, row) },
		"unknown status":           func(s *MaestroSubmissionSnapshot) { s.Submissions[0].Status = "SUCCESS" },
		"untrusted evidence":       func(s *MaestroSubmissionSnapshot) { s.Submissions[0].ResultEvidence = "INFERRED" },
		"incorrect page":           func(s *MaestroSubmissionSnapshot) { s.Limit = 4 },
		"missing pagination":       func(s *MaestroSubmissionSnapshot) { s.Total = 10 },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			copy := valid
			copy.Submissions = append([]MaestroSubmissionStatus(nil), valid.Submissions...)
			mutate(&copy)
			if _, err := decodeMaestroSubmissionSnapshot(encode(copy), "", 2, 0); err == nil {
				t.Fatal("invalid snapshot was accepted")
			}
		})
	}
	unchanged := valid
	unchanged.Changed = false
	unchanged.Submissions = nil
	if _, err := decodeMaestroSubmissionSnapshot(encode(unchanged), validCursor, 2, 0); err != nil {
		t.Fatalf("identical snapshot cursor rejected: %v", err)
	}
}
