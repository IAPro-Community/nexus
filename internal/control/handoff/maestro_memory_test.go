package handoff

import (
	"fmt"
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
		limit int
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
