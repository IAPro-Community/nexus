package handoff

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func maestroMemoryFixture(items, page string) []byte {
	return []byte(fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"result":{"schemaVersion":1,"authority":"maestro","items":%s,"page":%s}}`, items, page))
}

func TestDecodeMaestroMemorySearchReadOnlyContract(t *testing.T) {
	items := `[{"id":"obs_0123456789abcdef","timestamp":"2026-10-29T18:00:00Z","type":"decision","summary":"Use isolated worktrees","verified":false,"scope":{"level":"branch","repositoryId":"repo_0123","branch":"release/v1.0.0"}}]`
	page := `{"offset":0,"limit":20,"hasMore":false}`
	got, err := decodeMaestroMemorySearch(maestroMemoryFixture(items, page))
	if err != nil {
		t.Fatalf("read-only Maestro observation rejected: %v", err)
	}
	if got.Authority != "maestro" || len(got.Items) != 1 || got.Items[0].ID != "obs_0123456789abcdef" {
		t.Fatalf("unexpected Maestro read model: %+v", got)
	}
	if got.Items[0].Scope.Branch != "release/v1.0.0" {
		t.Fatal("branch scope was not preserved")
	}
	empty, err := decodeMaestroMemorySearch(maestroMemoryFixture(`[]`, page))
	if err != nil || empty.Items == nil || len(empty.Items) != 0 {
		t.Fatalf("valid empty query must return a non-nil list: %+v %v", empty, err)
	}
}

func TestDecodeMaestroMemorySearchFailsClosedOnInvalidResponse(t *testing.T) {
	page := `{"offset":0,"limit":20,"hasMore":false}`
	validItem := `[{"id":"obs_0123456789abcdef","timestamp":"2026-10-29","type":"decision","summary":"ok","scope":{"level":"repository","repositoryId":"repo_1"}}]`
	cases := [][]byte{
		[]byte("not-json"),
		[]byte(`{"jsonrpc":"2.0","id":1,"error":{"code":-32601}}`),
		[]byte(`{"jsonrpc":"2.0","id":2,"result":{"schemaVersion":1,"authority":"maestro","items":[],"page":{"limit":20}}}`),
		[]byte(`{"jsonrpc":"2.0","id":1,"result":{"schemaVersion":2,"authority":"maestro","items":[],"page":{"limit":20}}}`),
		[]byte(`{"jsonrpc":"2.0","id":1,"result":{"schemaVersion":1,"authority":"nexus","items":[],"page":{"limit":20}}}`),
		maestroMemoryFixture(validItem, `{"offset":0,"limit":51}`),
		maestroMemoryFixture(`[{"id":"../../escape","timestamp":"2026-10-29","type":"decision","summary":"bad","scope":{"level":"repository","repositoryId":"repo_1"}}]`, page),
		maestroMemoryFixture(`[{"id":"obs_0123456789abcdef","timestamp":"2026-10-29","type":"decision","summary":"bad","scope":{"level":"repository"}}]`, page),
		maestroMemoryFixture(`[{"id":"obs_0123456789abcdef","timestamp":"2026-10-29","type":"decision","summary":"bad","scope":{"level":"repository","repositoryId":"repo_1"}}]`, `{"offset":-1,"limit":20}`),
	}
	for _, input := range cases {
		if _, err := decodeMaestroMemorySearch(input); err == nil {
			t.Fatalf("unsafe Maestro response accepted: %s", input)
		}
	}
	if _, err := decodeMaestroMemorySearch([]byte(strings.Repeat("x", maestroMemoryMaxResponse+1))); err == nil {
		t.Fatal("oversized Maestro memory response accepted")
	}
}

func TestMaestroMemoryValidationsHaveNoNetworkOrStorageSideEffects(t *testing.T) {
	for _, tc := range []struct {
		workspace, task, query string
		limit                  int
	}{
		{"", "", "", 20},
		{"/safe", "../../other", "", 20},
		{"/safe", "", strings.Repeat("x", 201), 20},
		{"/safe", "", "", -1},
		{"/safe", "", "", 51},
	} {
		if _, err := ReadMaestroMemory(t.Context(), tc.workspace, tc.task, tc.query, tc.limit); err == nil {
			t.Fatalf("expected preflight validation failure: %+v", tc)
		}
	}
}

func TestMaestroMemoryMethodNegotiation(t *testing.T) {
	var init maestroBridgeInit
	init.ProtocolVersion = maestroBridgeProtocolVersion
	if _, err := maestroMemoryMethod(init); err == nil {
		t.Fatal("memory.search must not run without a declared read capability")
	}
	init.Capabilities.Memory.Read = true
	if got, err := maestroMemoryMethod(init); err != nil || got != "memory.search" {
		t.Fatalf("valid read capability rejected: %q, %v", got, err)
	}
	init.ProtocolVersion++
	if _, err := maestroMemoryMethod(init); err == nil {
		t.Fatal("incompatible Bridge version must fail closed")
	}
}

func validMaestroMemoryWriteRequest() MaestroMemoryWriteRequest {
	return MaestroMemoryWriteRequest{
		Confirmed:      true,
		Reason:         "Explicit operator-approved project observation",
		IdempotencyKey: "nexus-memory-write-001",
		Type:           "decision",
		Summary:        "Keep Maestro as the canonical semantic memory owner",
		ScopeLevel:     "branch",
	}
}

func TestDecodeMaestroMemoryWriteResultContract(t *testing.T) {
	for _, status := range []string{"RECORDED", "REPLAYED"} {
		raw := json.RawMessage(fmt.Sprintf(`{"schemaVersion":1,"authority":"maestro","status":%q,"id":"obs_0123456789abcdef","scope":{"level":"branch","repositoryId":"repo_1","branch":"release/v1.0.0"}}`, status))
		got, err := decodeMaestroMemoryWriteResult(raw)
		if err != nil {
			t.Fatalf("valid %s result rejected: %v", status, err)
		}
		if got.SchemaVersion != 1 || got.Authority != "maestro" || got.Status != status ||
			got.ID != "obs_0123456789abcdef" || got.Scope.RepositoryID != "repo_1" {
			t.Fatalf("unexpected Maestro write acknowledgement: %+v", got)
		}
	}
}

func TestDecodeMaestroMemoryWriteResultAcceptsCanonicalPromotedScope(t *testing.T) {
	raw := json.RawMessage(`{"schemaVersion":1,"authority":"maestro","status":"REPLAYED","id":"obs_0123456789abcdef","scope":{"level":"branch","repositoryId":"repo_1","branch":"release/v1.0.0","promoted":true}}`)
	got, err := decodeMaestroMemoryWriteResult(raw)
	if err != nil {
		t.Fatalf("canonical promoted scope rejected: %v", err)
	}
	if !got.Scope.Promoted {
		t.Fatal("canonical promoted scope was not preserved")
	}
}

func TestDecodeMaestroMemoryWriteResultFailsClosedOnInvalidResponse(t *testing.T) {
	cases := []json.RawMessage{
		json.RawMessage(`not-json`),
		json.RawMessage(`{"schemaVersion":2,"authority":"maestro","status":"RECORDED","id":"obs_0123456789abcdef","scope":{"level":"repository","repositoryId":"repo_1"}}`),
		json.RawMessage(`{"schemaVersion":1,"authority":"nexus","status":"RECORDED","id":"obs_0123456789abcdef","scope":{"level":"repository","repositoryId":"repo_1"}}`),
		json.RawMessage(`{"schemaVersion":1,"authority":"maestro","status":"PENDING","id":"obs_0123456789abcdef","scope":{"level":"repository","repositoryId":"repo_1"}}`),
		json.RawMessage(`{"schemaVersion":1,"authority":"maestro","status":"RECORDED","id":"../../escape","scope":{"level":"repository","repositoryId":"repo_1"}}`),
		json.RawMessage(`{"schemaVersion":1,"authority":"maestro","status":"RECORDED","id":"obs_0123456789abcdef","scope":{"level":"global","repositoryId":"repo_1"}}`),
		json.RawMessage(`{"schemaVersion":1,"authority":"maestro","status":"RECORDED","id":"obs_0123456789abcdef","scope":{"level":"repository"}}`),
		json.RawMessage(`{"schemaVersion":1,"authority":"maestro","status":"RECORDED","id":"obs_0123456789abcdef","scope":{"level":"branch","repositoryId":"repo_1"}}`),
		json.RawMessage(`{"schemaVersion":1,"authority":"maestro","status":"RECORDED","id":"obs_0123456789abcdef","scope":{"level":"workspace","repositoryId":"repo_1"}}`),
		json.RawMessage(`{"schemaVersion":1,"authority":"maestro","status":"RECORDED","id":"obs_0123456789abcdef","scope":{"level":"task","repositoryId":"repo_1"}}`),
		json.RawMessage(`{"schemaVersion":1,"authority":"maestro","status":"RECORDED","id":"obs_0123456789abcdef","scope":{"level":"repository","repositoryId":"repo_1"},"unexpected":true}`),
		json.RawMessage(`{"schemaVersion":1,"authority":"maestro","status":"RECORDED","id":"obs_0123456789abcdef","scope":{"level":"repository","repositoryId":"repo_1"}} {}`),
	}
	for _, input := range cases {
		if _, err := decodeMaestroMemoryWriteResult(input); err == nil {
			t.Fatalf("invalid Maestro memory write result accepted: %s", input)
		}
	}
	if _, err := decodeMaestroMemoryWriteResult(json.RawMessage(strings.Repeat("x", maestroMemoryMaxResponse+1))); err == nil {
		t.Fatal("oversized Maestro memory write result accepted")
	}
}

func TestMaestroMemoryWritePreflightValidationHasNoExternalSideEffects(t *testing.T) {
	valid := validMaestroMemoryWriteRequest()
	cases := []struct {
		name   string
		mutate func(*MaestroMemoryWriteRequest)
	}{
		{name: "not confirmed", mutate: func(r *MaestroMemoryWriteRequest) { r.Confirmed = false }},
		{name: "short reason", mutate: func(r *MaestroMemoryWriteRequest) { r.Reason = "short" }},
		{name: "long reason", mutate: func(r *MaestroMemoryWriteRequest) { r.Reason = strings.Repeat("r", 201) }},
		{name: "invalid idempotency key", mutate: func(r *MaestroMemoryWriteRequest) { r.IdempotencyKey = "not/valid" }},
		{name: "unknown type", mutate: func(r *MaestroMemoryWriteRequest) { r.Type = "unknown" }},
		{name: "empty summary", mutate: func(r *MaestroMemoryWriteRequest) { r.Summary = "  " }},
		{name: "long summary", mutate: func(r *MaestroMemoryWriteRequest) { r.Summary = strings.Repeat("s", 501) }},
		{name: "long details", mutate: func(r *MaestroMemoryWriteRequest) { r.Details = strings.Repeat("d", 4001) }},
		{name: "unknown scope", mutate: func(r *MaestroMemoryWriteRequest) { r.ScopeLevel = "global" }},
		{name: "invalid task", mutate: func(r *MaestroMemoryWriteRequest) { r.TaskID = "not/valid" }},
		{name: "task scope without task", mutate: func(r *MaestroMemoryWriteRequest) { r.ScopeLevel = "task" }},
		{name: "too many tags", mutate: func(r *MaestroMemoryWriteRequest) { r.Tags = make([]string, 17) }},
		{name: "empty tag", mutate: func(r *MaestroMemoryWriteRequest) { r.Tags = []string{""} }},
		{name: "too many files", mutate: func(r *MaestroMemoryWriteRequest) { r.Files = make([]string, 17) }},
		{name: "empty file", mutate: func(r *MaestroMemoryWriteRequest) { r.Files = []string{""} }},
		{name: "different project", mutate: func(r *MaestroMemoryWriteRequest) { r.ProjectPath = "/different" }},
	}
	// Point PATH at an empty directory: if preflight ever regresses, the
	// public client reaches process discovery instead of failing locally.
	t.Setenv("PATH", t.TempDir())
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			request := valid
			if tc.mutate != nil {
				tc.mutate(&request)
			}
			_, err := WriteMaestroMemory(t.Context(), "/workspace", request)
			if tc.name == "different project" {
				if err == nil || !strings.Contains(err.Error(), "must equal workspace") {
					t.Fatalf("projectPath mismatch must be rejected with the workspace error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("invalid Maestro memory write request accepted")
			}
			if strings.Contains(err.Error(), "CLI unavailable") {
				t.Fatalf("invalid request reached external process discovery: %v", err)
			}
		})
	}
	if err := validateMaestroMemoryWriteRequest("/workspace", valid); err != nil {
		t.Fatalf("valid Maestro memory write request rejected: %v", err)
	}
}

func TestMaestroMemoryWriteMethodNegotiationFailsClosed(t *testing.T) {
	var init maestroMemoryWriteInit
	init.ProtocolVersion = maestroBridgeProtocolVersion
	if _, err := maestroMemoryWriteMethod(init); err == nil {
		t.Fatal("memory.record must not run without a declared write capability")
	}
	init.Capabilities.Memory.Write = true
	if got, err := maestroMemoryWriteMethod(init); err != nil || got != "memory.record" {
		t.Fatalf("valid write capability rejected: %q, %v", got, err)
	}
	init.ProtocolVersion++
	if _, err := maestroMemoryWriteMethod(init); err == nil {
		t.Fatal("incompatible Bridge version must fail closed")
	}
}

func TestWriteMaestroMemoryUsesConsentedBridgeTransport(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix shell transport fixture; Windows Bridge contract is checked by Windows Go CI")
	}
	binDir := t.TempDir()
	bin := filepath.Join(binDir, "orquestrador-maestro")
	requestLog := filepath.Join(binDir, "request.json")
	initReply := `{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":1,"capabilities":{"memory":{"read":true,"write":true}}}}`
	writeReply := `{"jsonrpc":"2.0","id":2,"result":{"schemaVersion":1,"authority":"maestro","status":"RECORDED","id":"obs_0123456789abcdef","scope":{"level":"branch","repositoryId":"repo_1","branch":"release/v1.0.0"}}}`
	script := "#!/bin/sh\n" +
		"IFS= read -r request || exit 12\n" +
		"printf '%s\\n' '" + initReply + "'\n" +
		"IFS= read -r request || exit 13\n" +
		"printf '%s' \"$request\" > \"$NEXUS_MAESTRO_MEMORY_REQUEST\"\n" +
		"printf '%s\\n' '" + writeReply + "'\n"
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("NEXUS_MAESTRO_MEMORY_REQUEST", requestLog)

	got, err := WriteMaestroMemory(t.Context(), t.TempDir(), validMaestroMemoryWriteRequest())
	if err != nil {
		t.Fatalf("consented Maestro memory write failed: %v", err)
	}
	if got.Status != "RECORDED" || got.ID != "obs_0123456789abcdef" || got.Scope.Level != "branch" {
		t.Fatalf("unexpected Maestro memory acknowledgement: %+v", got)
	}

	raw, err := os.ReadFile(requestLog)
	if err != nil {
		t.Fatalf("Bridge request was not sent: %v", err)
	}
	var request struct {
		JSONRPC string         `json:"jsonrpc"`
		ID      int            `json:"id"`
		Method  string         `json:"method"`
		Params  map[string]any `json:"params"`
	}
	if err := json.Unmarshal(raw, &request); err != nil {
		t.Fatalf("invalid Bridge request: %v", err)
	}
	if request.JSONRPC != "2.0" || request.ID != 2 || request.Method != "memory.record" {
		t.Fatalf("unexpected Bridge request identity: %+v", request)
	}
	if request.Params["confirmed"] != true ||
		request.Params["idempotencyKey"] != "nexus-memory-write-001" ||
		request.Params["type"] != "decision" ||
		request.Params["scopeLevel"] != "branch" {
		t.Fatalf("unexpected memory.record params: %+v", request.Params)
	}
	if len(request.Params) != 6 {
		t.Fatalf("memory.record included unexpected parameters: %+v", request.Params)
	}
}

func TestWriteMaestroMemoryDoesNotCallRecordWithoutWriteCapability(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix shell transport fixture; Windows Bridge contract is checked by Windows Go CI")
	}
	binDir := t.TempDir()
	bin := filepath.Join(binDir, "orquestrador-maestro")
	requestLog := filepath.Join(binDir, "request.json")
	initReply := `{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":1,"capabilities":{"memory":{"read":true,"write":false}}}}`
	script := "#!/bin/sh\n" +
		"IFS= read -r request || exit 12\n" +
		"printf '%s\\n' '" + initReply + "'\n" +
		"IFS= read -r request || exit 0\n" +
		"printf '%s' \"$request\" > \"$NEXUS_MAESTRO_MEMORY_REQUEST\"\n"
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("NEXUS_MAESTRO_MEMORY_REQUEST", requestLog)

	_, err := WriteMaestroMemory(t.Context(), t.TempDir(), validMaestroMemoryWriteRequest())
	if err == nil || !strings.Contains(err.Error(), "no canonical memory write capability") {
		t.Fatalf("write without capability must fail closed: %v", err)
	}
	if _, statErr := os.Stat(requestLog); !os.IsNotExist(statErr) {
		t.Fatalf("memory.record was sent without write capability: %v", statErr)
	}
}
