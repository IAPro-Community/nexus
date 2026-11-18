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

func TestMaestroSubmissionUsageDecode(t *testing.T) {
	int64Ptr := func(v int64) *int64 { return &v }
	validUsage := MaestroSubmissionUsage{
		SchemaVersion:   1,
		Evidence:        "RUN_STORE_OBSERVED",
		TokensInput:     int64Ptr(1200),
		TokensOutput:    int64Ptr(300),
		TokensCached:    nil,
		TokensReasoning: int64Ptr(64),
		DurationMS:      int64Ptr(45000),
	}
	withUsage := func(usage MaestroSubmissionUsage) MaestroSubmissionStatus {
		row := MaestroSubmissionStatus{
			SchemaVersion: 1, Authority: "MAESTRO", Owner: "MAESTRO",
			SubmissionID: "submission-test-001", Status: "COMPLETED",
			TaskCount: 1, Providers: []string{"opencode"},
			ResultEvidence: "RUN_STORE_OBSERVED",
		}
		row.Usage = &usage
		return row
	}
	decodeRow := func(row MaestroSubmissionStatus) error {
		data, err := json.Marshal(row)
		if err != nil {
			t.Fatal(err)
		}
		decoded, err := decodeMaestroSubmissionRow(data)
		if err != nil {
			return err
		}
		if decoded.Usage == nil || decoded.Usage.TokensInput == nil || *decoded.Usage.TokensInput != 1200 {
			t.Fatalf("measured usage lost in decode: %+v", decoded.Usage)
		}
		if decoded.Usage.TokensCached != nil {
			t.Fatalf("unmeasured usage coerced away from nil: %+v", decoded.Usage.TokensCached)
		}
		return nil
	}
	if err := decodeRow(withUsage(validUsage)); err != nil {
		t.Fatalf("valid measured usage rejected: %v", err)
	}
	tests := map[string]func(*MaestroSubmissionUsage){
		"wrong schema version":  func(u *MaestroSubmissionUsage) { u.SchemaVersion = 2 },
		"forged evidence":       func(u *MaestroSubmissionUsage) { u.Evidence = "INFERRED" },
		"empty evidence":        func(u *MaestroSubmissionUsage) { u.Evidence = "" },
		"negative tokensInput":  func(u *MaestroSubmissionUsage) { u.TokensInput = int64Ptr(-1) },
		"negative tokensOutput": func(u *MaestroSubmissionUsage) { u.TokensOutput = int64Ptr(-5) },
		"negative duration":     func(u *MaestroSubmissionUsage) { u.DurationMS = int64Ptr(-1000) },
		"zero schema version":   func(u *MaestroSubmissionUsage) { u.SchemaVersion = 0 },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			usage := validUsage
			mutate(&usage)
			if err := decodeRow(withUsage(usage)); err == nil {
				t.Fatal("invalid measured usage accepted")
			}
		})
	}
	t.Run("unknown usage field fails closed", func(t *testing.T) {
		row := withUsage(validUsage)
		data, err := json.Marshal(row)
		if err != nil {
			t.Fatal(err)
		}
		forged := strings.Replace(string(data),
			`"usage":{"schemaVersion":1`,
			`"usage":{"tokensTotal":9999,"schemaVersion":1`, 1)
		if forged == string(data) {
			t.Fatal("test mutation did not apply")
		}
		if _, err := decodeMaestroSubmissionRow(json.RawMessage(forged)); err == nil {
			t.Fatal("unknown usage field accepted")
		}
	})
	t.Run("usage omitted stays nil", func(t *testing.T) {
		row := MaestroSubmissionStatus{
			SchemaVersion: 1, Authority: "MAESTRO", Owner: "MAESTRO",
			SubmissionID: "submission-test-001", Status: "DISPATCHED",
			TaskCount: 1, Providers: []string{"opencode"},
			ResultEvidence: "RUN_STORE_OBSERVED",
		}
		data, err := json.Marshal(row)
		if err != nil {
			t.Fatal(err)
		}
		decoded, err := decodeMaestroSubmissionRow(data)
		if err != nil || decoded.Usage != nil {
			t.Fatalf("absent usage must decode as nil: %+v %v", decoded.Usage, err)
		}
	})
	t.Run("snapshot rejects invalid usage", func(t *testing.T) {
		validCursor := "s1:" + strings.Repeat("a", 64)
		snapshot := MaestroSubmissionSnapshot{
			SchemaVersion: 1, Authority: "MAESTRO", Owner: "MAESTRO",
			Delivery: "SNAPSHOT_POLL", Replayable: false,
			ProjectID: "project-test-001", Cursor: validCursor,
			Changed: true, Limit: 2, Offset: 0, Total: 1,
			Submissions: []MaestroSubmissionStatus{withUsage(func() MaestroSubmissionUsage {
				usage := validUsage
				usage.Evidence = "SELF_REPORTED"
				return usage
			}())},
		}
		data, err := json.Marshal(snapshot)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := decodeMaestroSubmissionSnapshot(data, "", 2, 0); err == nil {
			t.Fatal("snapshot with invalid usage accepted")
		}
	})
}
