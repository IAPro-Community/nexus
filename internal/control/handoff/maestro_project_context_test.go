package handoff

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func maestroVerifiedContextJSON(content string) json.RawMessage {
	sum := sha256.Sum256([]byte(content))
	return json.RawMessage(fmt.Sprintf(
		`{"content":%q,"used":%d,"manifest":{"version":1,"contentDigest":%q}}`,
		content, len(content), hex.EncodeToString(sum[:]),
	))
}

func maestroProjectionJSON(content string) json.RawMessage {
	return json.RawMessage(fmt.Sprintf(
		`{"schemaVersion":1,"authority":"MAESTRO","project":{"id":"prj_123","name":"example","known":true,"status":"idle"},"context":%s}`,
		maestroVerifiedContextJSON(content),
	))
}

func TestMaestroBridgeNegotiatesCanonicalContext(t *testing.T) {
	var init maestroBridgeInit
	init.ProtocolVersion = 1
	init.Capabilities.ProjectContext = true
	init.Capabilities.ContextBrief = true
	method, err := maestroContextMethod(init)
	if err != nil || method != "project.context" {
		t.Fatalf("project.context must be preferred: %q: %v", method, err)
	}
	init.Capabilities.ProjectContext = false
	method, err = maestroContextMethod(init)
	if err != nil || method != "context.brief" {
		t.Fatalf("explicit legacy context.brief must remain supported: %q: %v", method, err)
	}
	init.Capabilities.ContextBrief = false
	if _, err = maestroContextMethod(init); err == nil {
		t.Fatal("missing capabilities must fail closed")
	}
	init.ProtocolVersion = 2
	init.Capabilities.ProjectContext = true
	if _, err = maestroContextMethod(init); err == nil {
		t.Fatal("unknown protocol must not be accepted")
	}
}

func TestMaestroCanonicalProjectContextDecode(t *testing.T) {
	content := "# Project\\nCanonical state"
	got, err := decodeMaestroProjectResult("project.context", maestroProjectionJSON(content))
	if err != nil {
		t.Fatal(err)
	}
	if got.Context != content || got.Method != "project.context" ||
		got.Project == nil || got.Project.ID != "prj_123" || !got.Project.Known {
		t.Fatalf("unexpected canonical project projection: %+v", got)
	}
	legacy, err := decodeMaestroProjectResult("context.brief", maestroVerifiedContextJSON(content))
	if err != nil || legacy.Context != content || legacy.Project != nil {
		t.Fatalf("legacy context must not fabricate project identity: %+v %v", legacy, err)
	}
}

func TestMaestroCanonicalProjectContextRejectsForgedEvidence(t *testing.T) {
	content := "# Valid context"
	for _, bad := range []json.RawMessage{
		json.RawMessage(`{"schemaVersion":2,"authority":"MAESTRO","project":{"known":true,"id":"prj_123"},"context":{}}`),
		json.RawMessage(`{"schemaVersion":1,"authority":"NEXUS","project":{"known":true,"id":"prj_123"},"context":{}}`),
		json.RawMessage(`{"schemaVersion":1,"authority":"MAESTRO","project":{"known":true},"context":{}}`),
		json.RawMessage(`{"schemaVersion":1,"authority":"MAESTRO","context":{}}`),
		json.RawMessage(`{"schemaVersion":1,"authority":"MAESTRO","project":{"known":true,"id":"prj_123"},"context":{"content":"forged","used":6,"manifest":{"version":1,"contentDigest":"000"}}}`),
	} {
		if _, err := decodeMaestroProjectResult("project.context", bad); err == nil {
			t.Fatalf("forged projection accepted: %s", bad)
		}
	}
	good := maestroProjectionJSON(content)
	if _, err := decodeMaestroProjectResult("fake.local.memory", good); err == nil {
		t.Fatal("local semantic fallback must not be accepted")
	}
}

func TestMaestroBridgeReplyChecksIdentityAndSize(t *testing.T) {
	makeReply := func(id int, raw string) string {
		return fmt.Sprintf(`{"jsonrpc":"2.0","id":%d,"result":%s}`+"\n", id, raw)
	}
	good := makeReply(2, string(maestroProjectionJSON("# Project")))
	result, err := readMaestroBridgeReply(bufio.NewReader(strings.NewReader(good)), 2)
	if err != nil || len(result) == 0 {
		t.Fatalf("canonical bridge response rejected: %v", err)
	}
	cases := []string{
		makeReply(1, "{}"),
		`{"jsonrpc":"1.0","id":2,"result":{}}` + "\n",
		`{"jsonrpc":"2.0","id":2,"error":{"code":-32601}}` + "\n",
		makeReply(2, "null"),
		"not-json\n",
		`{"jsonrpc":"2.0","id":2,"result":` + strings.Repeat("x", maestroBridgeReplyLimit) + "\n",
	}
	for _, input := range cases {
		if _, err := readMaestroBridgeReply(bufio.NewReader(strings.NewReader(input)), 2); err == nil {
			t.Fatalf("invalid Bridge response accepted (length=%d)", len(input))
		}
	}
	// Oversized replies must not consume memory beyond the configured limit.
	oversized := bytes.Repeat([]byte("x"), maestroBridgeReplyLimit+1)
	if _, err := readMaestroBridgeReply(bufio.NewReader(bytes.NewReader(append(oversized, '\n'))), 2); err == nil {
		t.Fatal("oversized Bridge response accepted")
	}
}
