package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleFlowGenerateRejectsUnsafeRequestsBeforeCallingPlanner(t *testing.T) {
	h := &NexusHandler{}
	cases := []struct{
		name,method,body string
		status int
	}{
		{"method",http.MethodGet,"",http.StatusMethodNotAllowed},
		{"malformed",http.MethodPost,"{",http.StatusBadRequest},
		{"empty",http.MethodPost,"{}",http.StatusBadRequest},
		{"no project",http.MethodPost,`{"prompt":"Build a feature"}`,http.StatusBadRequest},
		{"no prompt",http.MethodPost,`{"project_id":"prj_test"}`,http.StatusBadRequest},
		{"oversized",http.MethodPost,`{"project_id":"prj_test","prompt":"`+strings.Repeat("x",32*1024+1)+`"}`,http.StatusRequestEntityTooLarge},
	}
	for _,tt:=range cases{
		t.Run(tt.name,func(t *testing.T){
			req:=httptest.NewRequest(tt.method,"/api/v1/flow/generate",strings.NewReader(tt.body))
			rec:=httptest.NewRecorder()
			h.handleFlowGenerate(rec,req)
			if rec.Code!=tt.status {
				t.Fatalf("unexpected HTTP status: got %d want %d body=%s",rec.Code,tt.status,rec.Body.String())
			}
		})
	}
}
