package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/IAPro-Community/nexus/internal/control/registry"
	"github.com/IAPro-Community/nexus/internal/nexus"
)

func TestRespondAttentionEndpointRejectsStaleAndUnsafeInput(t *testing.T) {
	reg := registry.NewRegistry("")
	sess := registry.RuntimeSession{
		RuntimeID: "guarded-attention-test", State: registry.StateWaiting,
		AttentionReason: "QUESTION", AttentionKind: "needs_user",
		AttentionContext: "Continue?", PromptKind: "yn", AttentionSource: "INFERRED",
		AttentionFingerprint: "fingerprint-1", AttentionRevision: 4,
		HostGeneration: 900000000000001, ProviderGeneration: 910000000000002,
	}
	if err := reg.Register(sess); err != nil {
		t.Fatal(err)
	}
	otherProject := sess
	otherProject.RuntimeID = "guarded-attention-other-project"
	otherProject.ProjectID = "different-project"
	otherProject.ProjectName = "Different project"
	otherProject.AttentionFingerprint = "fingerprint-2"
	otherProject.HostGeneration++
	otherProject.ProviderGeneration++
	if err := reg.Register(otherProject); err != nil {
		t.Fatal(err)
	}
	h := &APIHandler{reg: reg, runtimes: nexus.NewRuntimeApplicationService(reg, nil, nil)}

	request := func(runtimeID string, payload map[string]any) *httptest.ResponseRecorder {
		data, err := json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(http.MethodPost,
			"/api/v1/runtimes/"+runtimeID+"/respond-attention", bytes.NewReader(data))
		rec := httptest.NewRecorder()
		h.handleRuntimeDetail(rec, req)
		return rec
	}
	valid := map[string]any{
		"input": "y", "attention_fingerprint": "fingerprint-1",
		"attention_revision": 4, "attention_generation": "900000000000001:910000000000002",
	}
	stale := make(map[string]any, len(valid))
	for key, value := range valid {
		stale[key] = value
	}
	stale["attention_revision"] = 3
	if rec := request(sess.RuntimeID, stale); rec.Code != http.StatusConflict {
		t.Fatalf("stale revision status=%d body=%s", rec.Code, rec.Body.String())
	}
	unsafeInput := make(map[string]any, len(valid))
	for key, value := range valid {
		unsafeInput[key] = value
	}
	unsafeInput["input"] = "y\n/permit all"
	if rec := request(sess.RuntimeID, unsafeInput); rec.Code != http.StatusBadRequest {
		t.Fatalf("multiline injection status=%d body=%s", rec.Code, rec.Body.String())
	}
	if rec := request(otherProject.RuntimeID, valid); rec.Code != http.StatusConflict {
		t.Fatalf("cross-runtime/project guard status=%d body=%s", rec.Code, rec.Body.String())
	}
	for _, runtimeID := range []string{sess.RuntimeID, otherProject.RuntimeID} {
		if got, _ := reg.Get(runtimeID); got.AttentionSubmittedRevision != 0 {
			t.Fatalf("invalid or cross-runtime requests claimed %s intervention: %d", runtimeID, got.AttentionSubmittedRevision)
		}
	}
}

func TestRespondAttentionEndpointRejectsUnlistedStructuredOption(t *testing.T) {
	reg := registry.NewRegistry("")
	session := registry.RuntimeSession{
		RuntimeID: "structured-choice-api", State: registry.StateApproval,
		AttentionReason: "APPROVAL", AttentionKind: "needs_user", AttentionContext: "Allow?",
		PromptKind: "choice", AttentionSource: "AUTHORITATIVE", AttentionFingerprint: "fp-choice",
		AttentionRevision: 2, HostGeneration: 11, ProviderGeneration: 12,
		AttentionInteractionID: "req-1", AttentionOptions: []registry.AttentionOption{{ID: "deny", Name: "Deny"}},
	}
	if err := reg.Register(session); err != nil {
		t.Fatal(err)
	}
	h := &APIHandler{reg: reg, runtimes: nexus.NewRuntimeApplicationService(reg, nil, nil)}
	payload := map[string]any{
		"option_id": "allow_all", "attention_fingerprint": "fp-choice", "attention_revision": 2,
		"attention_generation": "11:12", "interaction_id": "req-1",
	}
	data, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost,
		"/api/v1/runtimes/structured-choice-api/respond-attention", bytes.NewReader(data))
	rec := httptest.NewRecorder()
	h.handleRuntimeDetail(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("unlisted provider option must be rejected, status=%d body=%s", rec.Code, rec.Body.String())
	}
	after, _ := reg.Get(session.RuntimeID)
	if after.AttentionSubmittedRevision != 0 {
		t.Fatalf("invalid option consumed interaction revision: %+v", after)
	}
}

func TestSanitizeSessionPreservesLosslessGenerationIdentity(t *testing.T) {
	sess := registry.RuntimeSession{HostGeneration: 876543210987654321,
		ProviderGeneration: 123456789123456789}
	ready := sanitizeSession(sess)
	if ready.AttentionGeneration != "876543210987654321:123456789123456789" {
		t.Fatalf("unsafe generation projection: %s", ready.AttentionGeneration)
	}
}
