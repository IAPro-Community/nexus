package handoff

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func composeFixture() (MaestroComposePreviewRequest, json.RawMessage) {
	content := "Create an API, add verification and review before release."
	sum := sha256.Sum256([]byte(content))
	req := MaestroComposePreviewRequest{
		ArtifactID: "artifact-1", Version: 2, Content: content,
		SHA256: hex.EncodeToString(sum[:]), ApprovedModelUse: true,
	}
	raw := fmt.Sprintf(`{"schemaVersion":1,"authority":"MAESTRO","source":"maestro-semantic-planner","providerId":"codex","reviewOnly":true,"executable":false,"requiresApproval":true,"artifact":{"id":"artifact-1","version":2,"sha256":"%s"},"context":{"contentDigest":"%s"},"validation":{"valid":true,"blockers":[],"warnings":[]},"tasks":[{"id":"api","title":"Implement API","objective":"Build and validate API","dependsOn":[],"acceptanceCriteria":["API rejects invalid inputs"]},{"id":"verify","title":"Test API","objective":"Verify API behavior","dependsOn":["api"],"acceptanceCriteria":["Contract tests pass"]}]}`, req.SHA256, strings.Repeat("a", 64))
	return req, json.RawMessage(raw)
}

func TestMaestroComposeCapabilityFailsClosed(t *testing.T) {
	var init maestroBridgeInit
	init.ProtocolVersion = 1
	if _, err := maestroComposeMethod(init); err == nil {
		t.Fatal("missing compose capability accepted")
	}
	init.Capabilities.ProjectCompose = true
	if method, err := maestroComposeMethod(init); err != nil || method != "project.compose" {
		t.Fatalf("unexpected method %q: %v", method, err)
	}
	init.ProtocolVersion = 2
	if _, err := maestroComposeMethod(init); err == nil {
		t.Fatal("incompatible protocol accepted")
	}
}

func TestMaestroComposePreviewAcceptsVerifiedReadOnlyDAG(t *testing.T) {
	req, raw := composeFixture()
	preview, err := decodeMaestroComposePreview(raw, req)
	if err != nil {
		t.Fatal(err)
	}
	if !preview.Validation.Valid || len(preview.Tasks) != 2 || preview.Executable {
		t.Fatalf("unexpected compose preview: %+v", preview)
	}
}

func TestMaestroComposePreviewRejectsForgedModelApprovalAndDAG(t *testing.T) {
	req, raw := composeFixture()
	for _, change := range []struct{ old, replacement string }{
		{`"providerId":"codex"`, `"providerId":"opencode"`},
		{`"executable":false`, `"executable":true`},
		{`"reviewOnly":true`, `"reviewOnly":false`},
		{`"requiresApproval":true`, `"requiresApproval":false`},
		{`"source":"maestro-semantic-planner"`, `"source":"nexus"`},
		{`"id":"artifact-1"`, `"id":"other"`},
		{`"dependsOn":["api"]`, `"dependsOn":["verify"]`},
		{`"dependsOn":["api"]`, `"dependsOn":["missing"]`},
		{`"dependsOn":["api"]`, `"dependsOn":["api","api"]`},
		{`"valid":true,"blockers":[]`, `"valid":true,"blockers":[{"code":"UNRESOLVED"}]`},
		{`"API rejects invalid inputs"`, `"   "`},
	} {
		bad := strings.Replace(string(raw), change.old, change.replacement, 1)
		if _, err := decodeMaestroComposePreview(json.RawMessage(bad), req); err == nil {
			t.Fatalf("accepted invalid compose response %s", change.replacement)
		}
	}
}

func TestMaestroComposeRequiresExplicitConsentBeforeExternalCall(t *testing.T) {
	req, _ := composeFixture()
	for _, bad := range []MaestroComposePreviewRequest{
		{ArtifactID: req.ArtifactID, Version: req.Version, Content: req.Content, SHA256: req.SHA256},
		{ArtifactID: "unsafe/path", Version: req.Version, Content: req.Content, SHA256: req.SHA256, ApprovedModelUse: true},
		{ArtifactID: req.ArtifactID, Version: req.Version, Content: req.Content, SHA256: strings.Repeat("0", 64), ApprovedModelUse: true},
		{ArtifactID: req.ArtifactID, Version: req.Version, Content: req.Content, SHA256: req.SHA256, Model: "../unsafe", ApprovedModelUse: true},
	} {
		if _, err := FetchMaestroComposePreview(t.Context(), "/not-used", bad); err == nil {
			t.Fatalf("unsafe composition request accepted: %+v", bad)
		}
	}
}
