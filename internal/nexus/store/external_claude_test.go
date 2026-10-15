package store

import (
	"context"
	"testing"
	"time"
)

func TestExternalClaudeAtomicDedupAndSeparatedCost(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	msg := ExternalClaudeMessage{MessageHash:"hash-a", Model:"claude-test",InputTokens:10,OutputTokens:5,ObservedAt:now}
	if err:=s.SaveExternalClaudeBatch(ctx,"LOCAL_DEFAULT_UNVERIFIED","file-hash",100,[]ExternalClaudeMessage{msg});err!=nil{t.Fatal(err)}
	offset,err:=s.ExternalClaudeOffset(ctx,"LOCAL_DEFAULT_UNVERIFIED","file-hash")
	if err!=nil||offset!=100{t.Fatalf("offset: %d %v",offset,err)}
	msg.InputTokens=20
	if err:=s.SaveExternalClaudeBatch(ctx,"LOCAL_DEFAULT_UNVERIFIED","file-hash",160,[]ExternalClaudeMessage{msg});err!=nil{t.Fatal(err)}
	summary,err:=s.SummarizeExternalClaude(ctx,7)
	if err!=nil{t.Fatal(err)}
	if summary.Messages!=1||summary.InputTokens!=20||!summary.Unattributed{
		t.Fatalf("idempotent external usage: %+v",summary)
	}
	agg,err:=s.SummarizeCostObservations(CostObservationFilter{})
	if err!=nil{t.Fatal(err)}
	if agg.ObservationCount!=0{t.Fatalf("external transcripts must not contaminate ledger: %+v",agg)}
	bad:=msg;bad.InputTokens=-1
	if err:=s.SaveExternalClaudeBatch(ctx,"LOCAL_DEFAULT_UNVERIFIED","file-hash",200,[]ExternalClaudeMessage{bad});err==nil{
		t.Fatal("invalid batch accepted")
	}
	offset,_=s.ExternalClaudeOffset(ctx,"LOCAL_DEFAULT_UNVERIFIED","file-hash")
	if offset!=160{t.Fatalf("rolled back offset advanced to %d",offset)}
	if err:=s.SaveExternalClaudeBatch(ctx,"wrong-account","file-hash",250,nil);err==nil{
		t.Fatal("arbitrary scope accepted")
	}
}
