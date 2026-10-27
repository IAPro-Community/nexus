package agentwire

import (
 "context"
 "encoding/json"
 "errors"
 "io"
 "strconv"
 "sync"
 "sync/atomic"
)
// PiRPC speaks the Pi/Oh My Pi JSONL command protocol, NOT JSON-RPC 2.0.
type PiRPC struct {
 wire *JSONLines
 sequence atomic.Uint64
 mu sync.Mutex
 pending map[string]chan piReply
 onEvent func(string,json.RawMessage)
 done chan struct{}
 once sync.Once
 closed bool
}
type piReply struct{
 Command string `json:"command"`
 Success bool `json:"success"`
 Data json.RawMessage `json:"data"`
}
func NewPiRPC(input io.Reader,output io.Writer)*PiRPC{
 return &PiRPC{wire:NewJSONLines(input,output),pending:make(map[string]chan piReply),done:make(chan struct{})}
}
func(p *PiRPC)OnEvent(f func(string,json.RawMessage)){
 p.mu.Lock();defer p.mu.Unlock();p.onEvent=f
}
func(p *PiRPC)Command(ctx context.Context,command string,fields map[string]any,out *json.RawMessage)error{
 if command==""{return errors.New("Pi RPC: empty command")}
 id:=strconv.FormatUint(p.sequence.Add(1),10)
 ch:=make(chan piReply,1)
 p.mu.Lock();if p.closed{p.mu.Unlock();return ErrClosed}
 p.pending[id]=ch;p.mu.Unlock()
 defer func(){p.mu.Lock();delete(p.pending,id);p.mu.Unlock()}()
 msg:=map[string]any{"id":id,"type":command}
 for k,v:=range fields{
  if k=="id"||k=="type"{return errors.New("Pi RPC: reserved field")}
  msg[k]=v
 }
 if err:=p.wire.Write(msg);err!=nil{return err}
 select{
 case response:=<-ch:
  if !response.Success{return errors.New("Pi RPC: command rejected")}
  if out!=nil{*out=response.Data}
  return nil
 case <-ctx.Done():return ctx.Err()
 case <-p.done:return ErrClosed
 }
}
// agent_end is not an agent-settled signal: callers must observe agent_settled
// if they need a definitive quiescent state after retries and follow-up work.
func(p *PiRPC)Pump(ctx context.Context)error{
 defer p.Close()
 for{
  select{case <-ctx.Done():return ctx.Err();default:}
  raw,err:=p.wire.Read();if err!=nil{return err}
  var env struct{ Type string `json:"type"`;ID string `json:"id"` }
  if err:=json.Unmarshal(raw,&env);err!=nil{return err}
  if env.Type=="response"{
   var r piReply
   if err:=json.Unmarshal(raw,&r);err!=nil{return err}
   p.mu.Lock();ch:=p.pending[env.ID];p.mu.Unlock()
   if ch!=nil{select{case ch<-r:default:}}
   continue
  }
  p.mu.Lock();callback:=p.onEvent;p.mu.Unlock()
  if callback!=nil{callback(env.Type,raw)}
 }
}
func(p *PiRPC)Close(){p.once.Do(func(){p.mu.Lock();p.closed=true;p.mu.Unlock();p.wire.Close();close(p.done)})}
