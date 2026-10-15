package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/nexus"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func TestExternalClaudeRequiresExplicitConsentAndKeepsCostSeparate(t *testing.T) {
	nexus.ResetDefaultForTest()
	t.Setenv("NEXUS_DATA_DIR", t.TempDir())
	t.Setenv("AI_CLI_STATE_DIR", t.TempDir())
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // os.UserHomeDir on Windows ignores HOME
	t.Cleanup(nexus.ResetDefaultForTest)
	client, srv, csrf := csrfClient(t)
	dir := filepath.Join(home, ".claude", "projects", "sample")
	if err := os.MkdirAll(dir, 0700); err != nil { t.Fatal(err) }
	now := time.Now().UTC().Format(time.RFC3339Nano)
	line := []byte("{\"type\":\"assistant\",\"timestamp\":\""+now+"\",\"message\":{\"id\":\"message-1\",\"model\":\"claude-test\",\"usage\":{\"input_tokens\":10,\"output_tokens\":5}}}\n")
	if err := os.WriteFile(filepath.Join(dir, "sample.jsonl"),line,0600);err!=nil{t.Fatal(err)}
	post := func(body string) *http.Response {
		t.Helper()
		req,err:=http.NewRequest(http.MethodPost,srv.URL()+"/api/v1/cost/external-claude",bytes.NewBufferString(body))
		if err!=nil{t.Fatal(err)}
		req.Header.Set("Content-Type","application/json")
		req.Header.Set("X-CSRF-Token",csrf)
		resp,err:=client.Do(req)
		if err!=nil{t.Fatal(err)}
		return resp
	}
	without:=post("{\"consent\":false}")
	without.Body.Close()
	if without.StatusCode!=http.StatusBadRequest{t.Fatalf("must require consent, got %d",without.StatusCode)}
	pre,err:=client.Get(srv.URL()+"/api/v1/cost/external-claude")
	if err!=nil{t.Fatal(err)}
	var initial store.ExternalClaudeSummary
	if err:=json.NewDecoder(pre.Body).Decode(&initial);err!=nil{t.Fatal(err)}
	pre.Body.Close()
	if initial.Messages!=0{t.Fatalf("GET triggered implicit scan: %+v",initial)}
	yes:=post("{\"consent\":true}")
	if yes.StatusCode!=http.StatusOK{yes.Body.Close();t.Fatalf("explicit scan status %d",yes.StatusCode)}
	var result struct {
		Scan struct{ Messages int `json:"messages"` } `json:"scan"`
		Summary store.ExternalClaudeSummary `json:"summary"`
	}
	if err:=json.NewDecoder(yes.Body).Decode(&result);err!=nil{t.Fatal(err)}
	yes.Body.Close()
	if result.Scan.Messages!=1||result.Summary.Messages!=1||!result.Summary.Unattributed {
		t.Fatalf("external scan contract invalid: %+v",result)
	}
	repeat:=post("{\"consent\":true}")
	repeat.Body.Close()
	if repeat.StatusCode!=http.StatusOK{t.Fatalf("repeat scan status %d",repeat.StatusCode)}
	summary,err:=nexus.Default().Store().SummarizeExternalClaude(t.Context(),7)
	if err!=nil||summary.Messages!=1{t.Fatalf("double-counted transcript: %+v %v",summary,err)}
	agg,err:=nexus.Default().Store().SummarizeCostObservations(store.CostObservationFilter{})
	if err!=nil||agg.ObservationCount!=0{t.Fatalf("ledger contaminated: %+v %v",agg,err)}
}

func TestExternalClaudeConcurrentReadAndDeleteRejectsConflict(t *testing.T) {
	nexus.ResetDefaultForTest()
	t.Setenv("NEXUS_DATA_DIR",t.TempDir())
	t.Setenv("AI_CLI_STATE_DIR",t.TempDir())
	t.Cleanup(nexus.ResetDefaultForTest)
	client,srv,csrf:=csrfClient(t)
	externalClaudeOperationMu.Lock()
	defer externalClaudeOperationMu.Unlock()
	for _,method:=range []string{http.MethodPost,http.MethodDelete}{
		body:=""
		if method==http.MethodPost{body="{\"consent\":true}"}
		req,err:=http.NewRequest(method,srv.URL()+"/api/v1/cost/external-claude",bytes.NewBufferString(body))
		if err!=nil{t.Fatal(err)}
		req.Header.Set("Content-Type","application/json")
		req.Header.Set("X-CSRF-Token",csrf)
		resp,err:=client.Do(req)
		if err!=nil{t.Fatal(err)}
		resp.Body.Close()
		if resp.StatusCode!=http.StatusConflict{
			t.Fatalf("%s must not race existing scan/clear, got %d",method,resp.StatusCode)
		}
	}
}
