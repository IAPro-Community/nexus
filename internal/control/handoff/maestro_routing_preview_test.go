package handoff

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func routingPreviewFixture() (MaestroRoutePreviewRequest, json.RawMessage) {
	content := "Implement front-end with AGY and API with OpenCode"
	sum := sha256.Sum256([]byte(content))
	req := MaestroRoutePreviewRequest{
		ArtifactID: "prompt-42", Version: 2, Content: content,
		SHA256: hex.EncodeToString(sum[:]),
		Proposal: json.RawMessage(`{"tasks":[{"id":"a","title":"Frontend","objective":"Build frontend","acceptanceCriteria":["Frontend works"]},{"id":"b","title":"Backend","objective":"Build backend","acceptanceCriteria":["Backend works"]}]}`),
		Assignments: []MaestroWorkAssignment{
			{TaskID:"a", ProviderID:"agy", Model:"frontend-model", Sandbox:"enabled", WorktreePath:"/repo/agy"},
			{TaskID:"b", ProviderID:"opencode", Sandbox:"none", WorktreePath:"/repo/opencode"},
		},
	}
	body := fmt.Sprintf(`{"schemaVersion":1,"authority":"MAESTRO","previewOnly":true,"executable":false,"approvalRequired":true,"ready":true,"plan":{"authority":"MAESTRO","artifact":{"id":%q,"version":%d,"sha256":%q}},"assignments":[{"taskId":"a","providerId":"agy","model":"frontend-model","status":"READY_FOR_REVIEW"},{"taskId":"b","providerId":"opencode","model":"default","status":"READY_FOR_REVIEW"}],"blockers":[]}`, req.ArtifactID, req.Version, req.SHA256)
	return req, json.RawMessage(body)
}

func TestMaestroRoutePreviewNegotiatesCapability(t *testing.T) {
	var init maestroBridgeInit
	init.ProtocolVersion = 1
	if _, err := maestroRoutePreviewMethod(init); err == nil {
		t.Fatal("older Maestro cannot be accepted as supporting routePreview")
	}
	init.Capabilities.PlansRoutePreview = true
	method, err := maestroRoutePreviewMethod(init)
	if err != nil || method != "plans.routePreview" {
		t.Fatalf("unexpected method %q: %v", method, err)
	}
	init.ProtocolVersion = 2
	if _, err := maestroRoutePreviewMethod(init); err == nil {
		t.Fatal("unsupported protocol must fail closed")
	}
}

func TestMaestroRoutePreviewVerifiesExactProviderSelectionAndProvenance(t *testing.T) {
	req, raw := routingPreviewFixture()
	preview, err := decodeMaestroRoutePreview(raw, req)
	if err != nil { t.Fatal(err) }
	if !preview.Ready || len(preview.Assignments) != 2 ||
		preview.Assignments[0].ProviderID != "agy" || preview.Assignments[1].ProviderID != "opencode" {
		t.Fatalf("provider routing changed: %+v", preview)
	}
	for _, replacement := range []string{
		`"providerId":"codex"`,
		`"model":"other-model"`,
		`"executable":true`,
		`"authority":"NEXUS"`,
		`"sha256":"00000000"`,
		`"previewOnly":false`,
		`"approvalRequired":false`,
	} {
		bad := string(raw)
		switch replacement {
		case `"providerId":"codex"`:
			bad = strings.Replace(bad, `"providerId":"agy"`, replacement, 1)
		case `"model":"other-model"`:
			bad = strings.Replace(bad, `"model":"frontend-model"`, replacement, 1)
		case `"executable":true`:
			bad = strings.Replace(bad, `"executable":false`, replacement, 1)
		case `"authority":"NEXUS"`:
			bad = strings.Replace(bad, `"authority":"MAESTRO"`, replacement, 1)
		case `"sha256":"00000000"`:
			bad = strings.Replace(bad, `"sha256":"`+req.SHA256+`"`, replacement, 1)
		case `"previewOnly":false`:
			bad = strings.Replace(bad, `"previewOnly":true`, replacement, 1)
		case `"approvalRequired":false`:
			bad = strings.Replace(bad, `"approvalRequired":true`, replacement, 1)
		}
		if _, err := decodeMaestroRoutePreview(json.RawMessage(bad), req); err == nil {
			t.Errorf("accepted changed Maestro route preview: %s", replacement)
		}
	}
}

func TestMaestroRoutePreviewRefusesMalformedArtifactsWithoutCallingExternalTool(t *testing.T) {
	req, _ := routingPreviewFixture()
	for _, bad := range []MaestroRoutePreviewRequest{
		{ArtifactID: "../bad", Version: req.Version, Content: req.Content, SHA256: req.SHA256, Proposal:req.Proposal, Assignments:req.Assignments},
		{ArtifactID: req.ArtifactID, Version:req.Version, Content:req.Content, SHA256:strings.Repeat("0",64), Proposal:req.Proposal, Assignments:req.Assignments},
		{ArtifactID:req.ArtifactID, Version:req.Version, Content:req.Content, SHA256:req.SHA256, Proposal:req.Proposal, Assignments:[]MaestroWorkAssignment{req.Assignments[0], req.Assignments[0]}},
	} {
		if _, err := FetchMaestroRoutePreview(t.Context(), "/unused", bad); err == nil {
			t.Fatal("malformed local artifact was not rejected")
		}
	}
}
