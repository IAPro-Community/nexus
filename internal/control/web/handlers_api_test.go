package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteErrorPreservesMessageAndAddsStableCode(t *testing.T) {
	recorder := httptest.NewRecorder()
	writeError(recorder, http.StatusNotFound, "project not found")

	var body APIError
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Error != "project not found" || body.Code != "PROJECT_NOT_FOUND" {
		t.Fatalf("unexpected error response: %#v", body)
	}
}

func TestRuntimeDetailResponsePreservesContractFields(t *testing.T) {
	response := RuntimeDetailResponse{}
	if response.Session.RuntimeID != "" || response.Capabilities != nil {
		t.Fatalf("zero value unexpectedly populated: %#v", response)
	}
}

func TestStableErrorCodeDoesNotTreatUnknownAsZero(t *testing.T) {
	if got := stableErrorCode(http.StatusServiceUnavailable, "quota unavailable"); got != "QUOTA_UNKNOWN" {
		t.Fatalf("code = %q, want QUOTA_UNKNOWN", got)
	}
	if got := stableErrorCode(http.StatusInternalServerError, "something failed"); got != "INTERNAL_ERROR" {
		t.Fatalf("code = %q, want INTERNAL_ERROR", got)
	}
	for _, tc := range []struct {
		message string
		want    string
	}{
		{message: "INTERVENTION_ALREADY_RESOLVED: existing resolution", want: "INTERVENTION_ALREADY_RESOLVED"},
		{message: "STALE_INTERVENTION: version mismatch", want: "STALE_INTERVENTION"},
		{message: "provider dispatch outcome is unknown", want: "UNKNOWN_EXTERNAL_OUTCOME"},
		{message: "INTERVENTION_POLICY_DENIED: unsafe retry", want: "INTERVENTION_POLICY_DENIED"},
	} {
		if got := stableErrorCode(http.StatusConflict, tc.message); got != tc.want {
			t.Errorf("message %q: code = %q, want %q", tc.message, got, tc.want)
		}
	}
}
