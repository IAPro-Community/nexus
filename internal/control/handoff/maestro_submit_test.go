package handoff

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestMaestroSubmitCapabilityFailClosed(t *testing.T) {
	var init maestroBridgeInit
	init.ProtocolVersion = maestroBridgeProtocolVersion
	if _, err := maestroSubmitMethod(init); err == nil {
		t.Fatal("missing runsSubmit capability accepted")
	}
	init.Capabilities.RunsSubmit = true
	if method, err := maestroSubmitMethod(init); err != nil || method != "runs.submit" {
		t.Fatalf("valid submit capability rejected: %q %v", method, err)
	}
	init.ProtocolVersion++
	if _, err := maestroSubmitMethod(init); err == nil {
		t.Fatal("incompatible protocol accepted")
	}
	init.ProtocolVersion = maestroBridgeProtocolVersion
	init.Capabilities.RunsSubmit = false
	init.Capabilities.RunsSubmissionsCancel = true
	if method, err := maestroSubmissionCancelMethod(init); err != nil || method != "runs.submissions.cancel" {
		t.Fatalf("valid cancel capability rejected: %q %v", method, err)
	}
	if _, err := maestroSubmitMethod(init); err == nil {
		t.Fatal("submit capability must be negotiated independently")
	}
}

func maestroValidAck() MaestroRunAck {
	return MaestroRunAck{
		SchemaVersion:        1,
		SubmissionID:         "submission-nexus-001",
		RequestDigest:        strings.Repeat("a", 64),
		ManifestID:           "manifest-nexus-001",
		ManifestDigest:       strings.Repeat("b", 64),
		ApprovalDigest:       strings.Repeat("c", 64),
		Status:               "PENDING_WORKER",
		Replayed:             false,
		WorkspaceFingerprint: strings.Repeat("d", 64),
		Branch:               "main",
		HeadCommit:           strings.Repeat("e", 40),
	}
}

func TestMaestroRunAckValidation(t *testing.T) {
	valid := maestroValidAck()
	data, err := json.Marshal(valid)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := decodeMaestroRunAck(data); err != nil {
		t.Fatalf("valid ACK rejected: %v", err)
	}
	// Terminal replay of the same logical submission stays decodable.
	terminal := maestroValidAck()
	terminal.Status = "COMPLETED"
	terminal.Replayed = true
	data, err = json.Marshal(terminal)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := decodeMaestroRunAck(data); err != nil {
		t.Fatalf("terminal replay ACK rejected: %v", err)
	}
	tests := map[string]func(*MaestroRunAck){
		"unqueued status":     func(a *MaestroRunAck) { a.Status = "EXECUTING" },
		"forged authority id": func(a *MaestroRunAck) { a.SubmissionID = "../escape" },
		"short digest":        func(a *MaestroRunAck) { a.RequestDigest = "abc" },
		"missing approval":    func(a *MaestroRunAck) { a.ApprovalDigest = "" },
		"path in branch":      func(a *MaestroRunAck) { a.Branch = "../escape" },
		"unknown schema":      func(a *MaestroRunAck) { a.SchemaVersion = 2 },
		"foreign fingerprint": func(a *MaestroRunAck) { a.WorkspaceFingerprint = strings.Repeat("z", 63) },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			copy := maestroValidAck()
			mutate(&copy)
			data, err := json.Marshal(copy)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := decodeMaestroRunAck(data); err == nil {
				t.Fatal("invalid ACK was accepted")
			}
		})
	}
	// Unknown fields fail closed: a silently extended Bridge contract must
	// not be silently trusted by the Nexus client.
	extended := maestroValidAck()
	data, err = json.Marshal(struct {
		MaestroRunAck
		Extra string `json:"providerstdout"`
	}{extended, "leak"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := decodeMaestroRunAck(data); err == nil {
		t.Fatal("unknown ACK fields were accepted")
	}
}

func maestroSubmitFixture(t *testing.T, script string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("Unix shell transport fixture; Windows Bridge contract is checked by Windows Go CI")
	}
	binDir := t.TempDir()
	bin := filepath.Join(binDir, "orquestrador-maestro")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\n"+script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func maestroSubmitRequest() MaestroRunSubmitRequest {
	return MaestroRunSubmitRequest{
		ArtifactID: "artifact-001", Version: 1,
		Content: "approved body", SHA256: strings.Repeat("a", 64),
		ApprovedModelUse: true,
		Plan:             json.RawMessage(`{"tasks":[]}`),
		Assignments:      []MaestroWorkAssignment{{TaskID: "task-1", ProviderID: "opencode", Sandbox: "none"}},
	}
}

func TestMaestroRunSubmitTransportFailureHasNoDomainCode(t *testing.T) {
	maestroSubmitFixture(t, "exit 14\n")
	ack, err := SubmitMaestroRun(context.Background(), t.TempDir(), maestroSubmitRequest())
	if err == nil {
		t.Fatal("unreachable Bridge must fail closed")
	}
	if ack.SubmissionID != "" {
		t.Fatalf("failed admission must not return an ACK: %+v", ack)
	}
	if MaestroDomainCode(err) != "" {
		t.Fatalf("transport failures must not fake a domain code: %v", err)
	}
}

func TestMaestroRunSubmitSurfacesBridgeDomainCode(t *testing.T) {
	maestroSubmitFixture(t,
		"IFS= read -r request || exit 12\n"+
			`printf '%s\n' '{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":1,"capabilities":{"runsSubmit":true}}}'`+"\n"+
			"IFS= read -r request || exit 13\n"+
			`printf '%s\n' '{"jsonrpc":"2.0","id":2,"error":{"code":-32000,"data":{"code":"INVALID_SUBMISSION_PARAMS"}}}'`+"\n")
	ack, err := SubmitMaestroRun(context.Background(), t.TempDir(), maestroSubmitRequest())
	if err == nil {
		t.Fatal("Bridge rejection must fail closed")
	}
	if ack.SubmissionID != "" {
		t.Fatalf("rejected admission must not return an ACK: %+v", ack)
	}
	if got := MaestroDomainCode(err); got != "INVALID_SUBMISSION_PARAMS" {
		t.Fatalf("Bridge domain blocker must be surfaced verbatim: %v", err)
	}
}

func TestMaestroRunSubmitRequestValidation(t *testing.T) {
	valid := maestroSubmitRequest()
	noConsent := valid
	noConsent.ApprovedModelUse = false
	ack, err := SubmitMaestroRun(context.Background(), "/tmp", noConsent)
	if err == nil || ack.SubmissionID != "" || !strings.Contains(err.Error(), "explicitly approved") {
		t.Fatalf("unconsented model use must fail closed: %v", err)
	}
	badTask := valid
	badTask.Assignments = []MaestroWorkAssignment{{TaskID: "../escape", ProviderID: "opencode", Sandbox: "none"}}
	if _, err := SubmitMaestroRun(context.Background(), "/tmp", badTask); err == nil {
		t.Fatal("unsafe task assignment must fail closed")
	}
	oversized := valid
	oversized.Plan = json.RawMessage(strings.Repeat("a", 128*1024+1))
	if _, err := SubmitMaestroRun(context.Background(), "/tmp", oversized); err == nil {
		t.Fatal("oversized plan must fail closed")
	}
	negativeBudget := -1.0
	badBudget := valid
	badBudget.MaxCostUSD = &negativeBudget
	if _, err := SubmitMaestroRun(context.Background(), "/tmp", badBudget); err == nil {
		t.Fatal("negative budget must fail closed")
	}
}

func TestMaestroSubmissionRowValidation(t *testing.T) {
	row := MaestroSubmissionStatus{
		SchemaVersion: 1, Authority: "MAESTRO", Owner: "MAESTRO",
		SubmissionID: "submission-cancel-01", Status: "CANCEL_REQUESTED",
		TaskCount: 2, Providers: []string{"agy", "opencode"},
		ResultEvidence: "RUN_STORE_OBSERVED",
	}
	data, err := json.Marshal(row)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := decodeMaestroSubmissionRow(data); err != nil {
		t.Fatalf("valid row rejected: %v", err)
	}
	forged := row
	forged.ResultEvidence = "INFERRED"
	data, _ = json.Marshal(forged)
	if _, err := decodeMaestroSubmissionRow(data); err == nil {
		t.Fatal("untrusted evidence row accepted")
	}
	if _, err := decodeMaestroSubmissionRow(json.RawMessage(`not json`)); err == nil {
		t.Fatal("malformed row accepted")
	}
	if _, err := CancelMaestroSubmission(context.Background(), "/tmp", "short"); err == nil {
		t.Fatal("unsafe submission id must fail closed")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	if _, err := CancelMaestroSubmission(ctx, "/tmp", "submission-cancel-01"); err == nil {
		t.Fatal("cancellation without a Bridge must fail")
	}
	if _, err := CancelMaestroSubmission(context.Background(), "/tmp", "submission-cancel-01"); err == nil {
		t.Fatal("cancellation without a real Bridge must fail")
	}
	if _, err := CancelMaestroSubmission(context.Background(), "/tmp", "submission-cancel-01"); !errors.Is(err, err) {
		t.Fatal("errors must be returned as-is")
	}
}
