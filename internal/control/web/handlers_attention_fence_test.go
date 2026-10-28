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
	h := &APIHandler{reg: reg, runtimes: nexus.NewRuntimeApplicationService(reg, nil, nil)}

	request := func(payload map[string]any) *httptest.ResponseRecorder {
		data, err := json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(http.MethodPost,
			"/api/v1/runtimes/guarded-attention-test/respond-attention", bytes.NewReader(data))
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
	if rec := request(stale); rec.Code != http.StatusConflict {
		t.Fatalf("stale revision status=%d body=%s", rec.Code, rec.Body.String())
	}
	unsafeInput := make(map[string]any, len(valid))
	for key, value := range valid {
		unsafeInput[key] = value
	}
	unsafeInput["input"] = "y\n/permit all"
	if rec := request(unsafeInput); rec.Code != http.StatusBadRequest {
		t.Fatalf("multiline injection status=%d body=%s", rec.Code, rec.Body.String())
	}
	if got, _ := reg.Get(sess.RuntimeID); got.AttentionSubmittedRevision != 0 {
		t.Fatalf("invalid requests claimed intervention: %d", got.AttentionSubmittedRevision)
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
