package agentwire

import (
 "context"
 "encoding/json"
 "io"
 "strings"
 "sync"
 "testing"
 "time"
)

func TestBridgeACPStartAndPrompt(t *testing.T){
 // A fake ACP agent responds to the same JSONL messages as real clients.
 providerRead,clientWrite:=io.Pipe()
 clientRead,providerWrite:=io.Pipe()
 defer providerRead.Close();defer clientWrite.Close()
 defer clientRead.Close();defer providerWrite.Close()
 var outputMu sync.Mutex
 var output strings.Builder
 bridge,err:=NewBridge("acp",clientRead,clientWrite,"/workspace/demo",nil,func(s string){outputMu.Lock();defer outputMu.Unlock();output.WriteString(s)})
 if err!=nil{t.Fatal(err)}
 go func(){
  wire:=NewJSONLines(providerRead,providerWrite)
  for{
   req,err:=wire.Read();if err!=nil{return}
   var v struct{ID string `json:"id"`;Method string `json:"method"`}
   if json.Unmarshal(req,&v)!=nil{return}
   var result any
   switch v.Method{
   case "initialize":result=map[string]any{"protocolVersion":1,"agentCapabilities":map[string]any{}}
   case "session/new":result=map[string]string{"sessionId":"sess-1"}
   case "session/prompt":
    _=wire.Write(map[string]any{"jsonrpc":"2.0","method":"session/update","params":map[string]any{
     "sessionId":"sess-1",
     "update":map[string]any{"sessionUpdate":"agent_message_chunk","content":map[string]string{"text":"hello from provider"}},
    }})
    result=map[string]string{"stopReason":"end_turn"}
   default:result=map[string]any{}
   }
   _=wire.Write(map[string]any{"jsonrpc":"2.0","id":v.ID,"result":result})
  }
 }()
 ctx,cancel:=context.WithTimeout(context.Background(),3*time.Second)
 defer cancel()
 if err:=bridge.Start(ctx);err!=nil{t.Fatal(err)}
 if bridge.SessionID()!="sess-1"{t.Fatal("provider session id not retained")}
 if err:=bridge.Prompt(ctx,"test");err!=nil{t.Fatal(err)}
 outputMu.Lock();got:=output.String();outputMu.Unlock()
 if !strings.Contains(got,"hello from provider"){t.Fatalf("missing provider update: %s",got)}
 bridge.Stop()
}

func TestBridgePermissionRequiresExactOption(t *testing.T){
 b:=&Bridge{pending:map[string]*permissionWait{"req-1":{
  options:map[string]bool{"allow_once":true,"reject_once":true},
  selected:make(chan string,1),
 }}}
 if err:=b.Choose("req-1","admin_override");err==nil{t.Fatal("unadvertised grant accepted")}
 if err:=b.Choose("not-real","allow_once");err==nil{t.Fatal("unknown request accepted")}
 if err:=b.Choose("req-1","reject_once");err!=nil{t.Fatal(err)}
 if err:=b.Choose("req-1","allow_once");err==nil{t.Fatal("repeat permission accepted")}
 if got:=<-b.pending["req-1"].selected;got!="reject_once"{t.Fatalf("response changed: %s",got)}
}

func TestBridgePiSettledNotAck(t *testing.T){
 providerRead,clientWrite:=io.Pipe()
 clientRead,providerWrite:=io.Pipe()
 defer providerRead.Close();defer clientWrite.Close()
 defer clientRead.Close();defer providerWrite.Close()
 signals:=make(chan WireSignal,8)
 bridge,err:=NewBridge("pi_rpc",clientRead,clientWrite,"/workspace/demo",func(s WireSignal){signals<-s},nil)
 if err!=nil{t.Fatal(err)}
 go func(){
  wire:=NewJSONLines(providerRead,providerWrite)
  for{
   msg,err:=wire.Read();if err!=nil{return}
   var p struct{ID string `json:"id"`}
   _=json.Unmarshal(msg,&p)
   _=wire.Write(map[string]any{"type":"response","id":p.ID,"command":"prompt","success":true})
   _=wire.Write(map[string]any{"type":"agent_end"})
   _=wire.Write(map[string]any{"type":"agent_settled"})
  }
 }()
 ctx,cancel:=context.WithTimeout(context.Background(),3*time.Second)
 defer cancel()
 if err:=bridge.Start(ctx);err!=nil{t.Fatal(err)}
 if err:=bridge.Prompt(ctx,"run tests");err!=nil{t.Fatal(err)}
 settled:=false
 for i:=0;i<5;i++{
  select{case v:=<-signals:if v.State=="done"{settled=true}
  case <-time.After(time.Second):t.Fatal("missing provider event")}
  if settled{break}
 }
 if !settled{t.Fatal("agent_settled not forwarded")}
 bridge.Stop()
}
