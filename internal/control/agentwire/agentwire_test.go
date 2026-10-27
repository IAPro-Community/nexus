package agentwire

import(
 "bytes"
 "context"
 "encoding/json"
 "errors"
 "io"
 "strings"
 "sync"
 "testing"
 "time"
)
func TestJSONLinesBounds(t *testing.T){
 j:=NewJSONLines(strings.NewReader("{\"x\":\"a\\u2028b\"}\n"),io.Discard)
 frame,err:=j.Read()
 if err!=nil||!json.Valid(frame){t.Fatalf("invalid frame: %s %v",frame,err)}
 huge:=NewJSONLines(strings.NewReader(strings.Repeat("x",MaxFrame+1)+"\n"),io.Discard)
 _,err=huge.Read()
 if !errors.Is(err,ErrFrameTooLarge){t.Fatalf("expected frame bound, got %v",err)}
}
type synchronizedBuffer struct{mu sync.Mutex;data bytes.Buffer}
func(s *synchronizedBuffer)Write(p []byte)(int,error){s.mu.Lock();defer s.mu.Unlock();return s.data.Write(p)}
func(s *synchronizedBuffer)Len()int{s.mu.Lock();defer s.mu.Unlock();return s.data.Len()}
func TestACPRequestCorrelation(t *testing.T){
 in,out:=io.Pipe();defer in.Close();defer out.Close()
 written:=&synchronizedBuffer{}
 acp:=NewACP(in,written)
 ctx,cancel:=context.WithCancel(context.Background());defer cancel()
 go func(){_=acp.Pump(ctx)}()
 result:=make(chan error,1)
 go func(){
  var response struct{Count int `json:"count"`}
  err:=acp.Call(ctx,"test/ping",map[string]any{},&response)
  if err==nil&&response.Count!=5{err=errors.New("wrong response")}
  result<-err
 }()
 deadline:=time.After(time.Second)
 for written.Len()==0{
  select{case <-deadline:t.Fatal("ACP request not written");default:time.Sleep(time.Millisecond)}
 }
 _,_=out.Write([]byte("{\"jsonrpc\":\"2.0\",\"id\":\"1\",\"result\":{\"count\":5}}\n"))
 select{case err:=<-result:if err!=nil{t.Fatal(err)}
 case <-time.After(time.Second):t.Fatal("no correlated response")}
}
func TestPiRPCAcknowledgementAndSettledEvent(t *testing.T){
 in,out:=io.Pipe();defer in.Close();defer out.Close()
 written:=&synchronizedBuffer{}
 pi:=NewPiRPC(in,written)
 events:=make(chan string,2)
 pi.OnEvent(func(kind string,_ json.RawMessage){events<-kind})
 ctx,cancel:=context.WithCancel(context.Background());defer cancel()
 go func(){_=pi.Pump(ctx)}()
 complete:=make(chan error,1)
 go func(){complete<-pi.Command(ctx,"prompt",map[string]any{"message":"hello"},nil)}()
 deadline:=time.After(time.Second)
 for written.Len()==0{
  select{case <-deadline:t.Fatal("Pi RPC not written");default:time.Sleep(time.Millisecond)}
 }
 _,_=out.Write([]byte("{\"type\":\"response\",\"id\":\"1\",\"command\":\"prompt\",\"success\":true}\n{\"type\":\"agent_end\"}\n{\"type\":\"agent_settled\"}\n"))
 select{case err:=<-complete:if err!=nil{t.Fatal(err)}
 case <-time.After(time.Second):t.Fatal("Pi command never completed")}
 if <-events!="agent_end"||<-events!="agent_settled"{t.Fatal("lost settled event")}
}
func TestReducerIdentityAndEpoch(t *testing.T){
 r:=NewReducer()
 input:=Observation{RuntimeID:"first",Epoch:1,Sequence:1,Turn:TurnRunning,Evidence:Authoritative,Request:&Intervention{ID:"question1",Kind:Question}}
 got,err:=r.Apply(input)
 if err!=nil||len(got.Attention)!=1{t.Fatalf("initial state: %+v %v",got,err)}
 got,_=r.Apply(input);if len(got.Attention)!=1{t.Fatal("duplicate interaction")}
 second:=input;second.RuntimeID="second"
 got,_=r.Apply(second);if len(got.Attention)!=1{t.Fatal("merged independent runtimes")}
 got,_=r.Apply(Observation{RuntimeID:"first",Epoch:1,Sequence:2,ResolvedID:"question1"})
 if len(got.Attention)!=0{t.Fatal("resolved interaction persisted")}
 got,_=r.Apply(input);if len(got.Attention)!=0{t.Fatal("stale replay")}
 got,_=r.Apply(Observation{RuntimeID:"first",Epoch:2,Sequence:1,Turn:TurnRunning})
 if got.Epoch!=2||got.Sequence!=1{t.Fatal("epoch change not fenced")}
}
