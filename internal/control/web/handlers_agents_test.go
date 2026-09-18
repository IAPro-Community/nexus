package web

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWriteAgentConflictClassifiesRequiredResourceSelection(t *testing.T) {
	recorder := httptest.NewRecorder()
	writeAgentConflict(recorder, &testError{message: "REQUIRED_RESOURCE_SELECTION: choose a resource"})
	if recorder.Code != 409 {
		t.Fatalf("status=%d, want 409", recorder.Code)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, `"code":"REQUIRED_RESOURCE_SELECTION"`) {
		t.Fatalf("body=%s, missing stable resource-selection code", body)
	}
}

type testError struct{ message string }

func (e *testError) Error() string { return e.message }
