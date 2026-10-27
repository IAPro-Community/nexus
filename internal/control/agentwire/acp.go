package agentwire

import (
 "context"
 "encoding/json"
 "errors"
 "fmt"
 "io"
 "strconv"
 "sync"
 "sync/atomic"
)

type ACP struct {
 wire *JSONLines
 sequence atomic.Uint64
 mu sync.Mutex
 pending map[string]chan rpcReply
 handler func(context.Context,string,json.RawMessage)(any,error)
 notify func(string,json.RawMessage)
 closed bool
 closeOnce sync.Once
 done chan struct{}
}
type rpcReply struct {
 Result json.RawMessage `json:"result"`
 Error *rpcError `json:"error"`
}
type rpcError struct {
 Code int `json:"code"`
 Message string `json:"message"`
}
func(e *rpcError)Error()string{return fmt.Sprintf("ACP error %d: %s",e.Code,e.Message)}
func NewACP(input io.Reader,output io.Writer)*ACP{
 return &ACP{wire:NewJSONLines(input,output),pending:make(map[string]chan rpcReply),done:make(chan struct{})}
}
// Without an explicit policy handler, inbound permissions and filesystem
// requests are always rejected. Never auto-approve an agent's tool calls.
func(a *ACP)OnRequest(f func(context.Context,string,json.RawMessage)(any,error)){
 a.mu.Lock();defer a.mu.Unlock();a.handler=f
}
func(a *ACP)OnNotification(f func(string,json.RawMessage)){
 a.mu.Lock();defer a.mu.Unlock();a.notify=f
}
func(a *ACP)Done()<-chan struct{}{return a.done}
func(a *ACP)Call(ctx context.Context,method string,params any,result any)error{
 if method==""{return errors.New("ACP: empty method")}
 id:=strconv.FormatUint(a.sequence.Add(1),10)
 replyCh:=make(chan rpcReply,1)
 a.mu.Lock()
 if a.closed{a.mu.Unlock();return ErrClosed}
 a.pending[id]=replyCh
 a.mu.Unlock()
 defer func(){a.mu.Lock();delete(a.pending,id);a.mu.Unlock()}()
 if err:=a.wire.Write(map[string]any{"jsonrpc":"2.0","id":id,"method":method,"params":params});err!=nil{return err}
 select{
 case reply:=<-replyCh:
  if reply.Error!=nil{return reply.Error}
  if result==nil||len(reply.Result)==0{return nil}
  return json.Unmarshal(reply.Result,result)
 case <-ctx.Done():return ctx.Err()
 case <-a.done:return ErrClosed
 }
}
func(a *ACP)Notify(method string,params any)error{
 if method==""{return errors.New("ACP: empty method")}
 return a.wire.Write(map[string]any{"jsonrpc":"2.0","method":method,"params":params})
}
// Pump is the sole stdout reader and must start before Call. Provider requests
// execute separately so permission callbacks cannot deadlock a live prompt.
func(a *ACP)Pump(ctx context.Context)error{
 defer a.Close()
 for{
  select{case <-ctx.Done():return ctx.Err();default:}
  frame,err:=a.wire.Read();if err!=nil{return err}
  var e struct{
   Version string `json:"jsonrpc"`
   ID json.RawMessage `json:"id"`
   Method string `json:"method"`
   Params json.RawMessage `json:"params"`
   Result json.RawMessage `json:"result"`
   Error *rpcError `json:"error"`
  }
  if err:=json.Unmarshal(frame,&e);err!=nil{return err}
  if e.Version!="2.0"{return errors.New("ACP: unsupported JSON-RPC version")}
  if e.Method!=""{
   a.mu.Lock();handler:=a.handler;notify:=a.notify;a.mu.Unlock()
   if len(e.ID)>0{
    go func(id json.RawMessage,method string,params json.RawMessage){
     var value any;var callbackErr error
     if handler==nil{callbackErr=&rpcError{Code:-32601,Message:"method unavailable"}}else{value,callbackErr=handler(ctx,method,params)}
     response:=map[string]any{"jsonrpc":"2.0","id":id}
     if callbackErr!=nil{
      if e,ok:=callbackErr.(*rpcError);ok{response["error"]=e}else{response["error"]=rpcError{Code:-32000,Message:"request rejected"}}
     }else{response["result"]=value}
     _=a.wire.Write(response)
    }(e.ID,e.Method,e.Params)
   }else if notify!=nil{notify(e.Method,e.Params)}
   continue
  }
  if len(e.ID)==0{continue}
  key:=string(e.ID)
  if len(key)>0&&key[0]=='"'{
   if id,err:=strconv.Unquote(key);err==nil{key=id}
  }
  a.mu.Lock();ch:=a.pending[key];a.mu.Unlock()
  if ch!=nil{select{case ch<-rpcReply{Result:e.Result,Error:e.Error}:default:}}
 }
}
func(a *ACP)Close(){
 a.closeOnce.Do(func(){
  a.mu.Lock();a.closed=true;a.mu.Unlock()
  a.wire.Close();close(a.done)
 })
}
// InitResult captures negotiated capabilities rather than guessing from a CLI
// name or relying on capabilities declared by an unrelated adapter.
type InitResult struct{
 ProtocolVersion int `json:"protocolVersion"`
 AgentCapabilities json.RawMessage `json:"agentCapabilities"`
 AuthMethods json.RawMessage `json:"authMethods"`
}
func(a *ACP)Initialize(ctx context.Context,protocolVersion int)(InitResult,error){
 var out InitResult
 err:=a.Call(ctx,"initialize",map[string]any{
  "protocolVersion":protocolVersion,
  "clientCapabilities":map[string]any{"fs":map[string]bool{"readTextFile":false,"writeTextFile":false},"terminal":false},
  "clientInfo":map[string]any{"name":"nexus","version":"1.0.0"},
 },&out)
 return out,err
}
func(a *ACP)NewSession(ctx context.Context,cwd string)(string,error){
 if cwd==""{return "",errors.New("ACP: working directory required")}
 var result struct{SessionID string `json:"sessionId"`}
 err:=a.Call(ctx,"session/new",map[string]any{"cwd":cwd,"mcpServers":[]any{}},&result)
 if err!=nil{return "",err}
 if result.SessionID==""{return "",errors.New("ACP: missing sessionId")}
 return result.SessionID,nil
}
func(a *ACP)Prompt(ctx context.Context,sessionID,text string)(json.RawMessage,error){
 if sessionID==""{return nil,errors.New("ACP: missing sessionId")}
 var result json.RawMessage
 err:=a.Call(ctx,"session/prompt",map[string]any{"sessionId":sessionID,"prompt":[]map[string]string{{"type":"text","text":text}}},&result)
 return result,err
}
func(a *ACP)Cancel(sessionID string)error{
 if sessionID==""{return errors.New("ACP: missing sessionId")}
 return a.Notify("session/cancel",map[string]string{"sessionId":sessionID})
}
