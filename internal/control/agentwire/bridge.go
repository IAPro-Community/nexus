package agentwire

import (
 "context"
 "encoding/json"
 "errors"
 "fmt"
 "strings"
 "sync"
 "sync/atomic"
)

// WireSignal is an observation emitted by a provider protocol. A command
// acknowledgement and a complete agent turn are deliberately separate.
type WireSignal struct {
 State string
 Kind string
 Summary string
 InteractionID string
 Options []PermissionChoice
}
type PermissionChoice struct {
 ID string `json:"option_id"`
 Name string `json:"name"`
}
type permissionWait struct {
 options map[string]bool
 selected chan string
 submitted bool
}

// Bridge adapts bidirectional ACP and Pi's JSONL RPC to one session-scoped
// callback contract. The bridge does not own a terminal, permission policy,
// authentication, or shared Nexus runtime state.
type Bridge struct {
 mode string
 acp *ACP
 pi *PiRPC
 cwd string
 sessionID string
 ctx context.Context
 cancel context.CancelFunc
 signals func(WireSignal)
 output func(string)
 mu sync.Mutex
 pending map[string]*permissionWait
 next atomic.Uint64
 busy bool
 ready chan struct{}
}

func NewBridge(mode string, input interface{ Read([]byte)(int,error) }, output interface{ Write([]byte)(int,error) }, cwd string, signals func(WireSignal), terminalOutput func(string)) (*Bridge,error) {
 if cwd=="" {return nil,errors.New("agentwire: workspace required")}
 b:=&Bridge{mode:mode,cwd:cwd,signals:signals,output:terminalOutput,pending:make(map[string]*permissionWait),ready:make(chan struct{})}
 switch mode {
 case "acp": b.acp=NewACP(input,output)
 case "pi_rpc": b.pi=NewPiRPC(input,output)
 default: return nil,fmt.Errorf("agentwire: unsupported protocol %q",mode)
 }
 return b,nil
}
func(b *Bridge)emit(s WireSignal){if b.signals!=nil{b.signals(s)}}
func(b *Bridge)print(s string){if b.output!=nil && s!=""{b.output(s)}}

// Start must be called after the child process is running. ACP initialization
// is bounded by the caller's context; Pi has no initialization handshake.
func(b *Bridge)Start(ctx context.Context)error{
 b.ctx,b.cancel=context.WithCancel(ctx)
 if b.acp!=nil {
  b.acp.OnRequest(b.request)
  b.acp.OnNotification(b.notification)
  go func(){if err:=b.acp.Pump(b.ctx);err!=nil&&b.ctx.Err()==nil{b.emit(WireSignal{State:"error",Summary:"ACP stream terminated"})}}()
  handshake,cancelHandshake:=context.WithTimeout(b.ctx,20*1000000000)
  defer cancelHandshake()
  reply,err:=b.acp.Initialize(handshake,1)
  if err!=nil {b.cancel();return fmt.Errorf("ACP initialize: %w",err)}
  if reply.ProtocolVersion!=1 {b.cancel();return fmt.Errorf("ACP version mismatch %d",reply.ProtocolVersion)}
  id,err:=b.acp.NewSession(handshake,b.cwd)
  if err!=nil {b.cancel();return fmt.Errorf("ACP new session: %w",err)}
  b.mu.Lock();b.sessionID=id;b.mu.Unlock()
 } else {
  b.pi.OnEvent(b.piEvent)
  go func(){if err:=b.pi.Pump(b.ctx);err!=nil&&b.ctx.Err()==nil{b.emit(WireSignal{State:"error",Summary:"RPC stream terminated"})}}()
 }
 close(b.ready)
 b.emit(WireSignal{State:"idle",Summary:"Ready for a prompt"})
 return nil
}
func(b *Bridge)SessionID()string{b.mu.Lock();defer b.mu.Unlock();return b.sessionID}
func(b *Bridge)Stop(){
 if b.cancel!=nil{b.cancel()}
 b.mu.Lock()
 for _,p:=range b.pending {select {case p.selected<- "":default:}}
 b.mu.Unlock()
 if b.acp!=nil{b.acp.Close()}
 if b.pi!=nil{b.pi.Close()}
}

func(b *Bridge)Prompt(ctx context.Context,text string)error {
 select {
 case <-b.ready:
 case <-ctx.Done():return ctx.Err()
 }
 if strings.TrimSpace(text)==""{return errors.New("empty prompt")}
 b.mu.Lock()
 if b.busy{b.mu.Unlock();return errors.New("provider turn already running")}
 b.busy=true
 id:=b.sessionID
 b.mu.Unlock()
 defer func(){b.mu.Lock();b.busy=false;b.mu.Unlock()}()
 b.emit(WireSignal{State:"working",Summary:"Agent is working"})
 if b.acp!=nil{
  result,err:=b.acp.Prompt(ctx,id,text)
  if err!=nil {b.emit(WireSignal{State:"error",Summary:"Agent prompt failed"});return err}
  var response struct{StopReason string `json:"stopReason"`}
  _=json.Unmarshal(result,&response)
  switch response.StopReason {
  case "cancelled":b.emit(WireSignal{State:"idle",Summary:"Agent turn cancelled"})
  case "":b.emit(WireSignal{State:"idle",Summary:"Agent response received"})
  default:b.emit(WireSignal{State:"done",Summary:"Agent turn finished"})
  }
  return nil
 }
 // Pi acknowledges the command before running. Only agent_settled confirms
 // that work has stopped, so no COMPLETED event is emitted here.
 if err:=b.pi.Command(ctx,"prompt",map[string]any{"message":text},nil);err!=nil{
  b.emit(WireSignal{State:"error",Summary:"Agent prompt rejected"})
  return err
 }
 return nil
}
func(b *Bridge)request(ctx context.Context,method string,raw json.RawMessage)(any,error){
 if method!="session/request_permission"{return nil,errors.New("ACP operation denied")}
 var req struct{
  SessionID string `json:"sessionId"`
  Title string `json:"title"`
  Options []struct{OptionID string `json:"optionId"`;Name string `json:"name"`}`json:"options"`
 }
 if err:=json.Unmarshal(raw,&req);err!=nil{return nil,err}
 b.mu.Lock()
 if req.SessionID==""||req.SessionID!=b.sessionID||len(req.Options)==0 {b.mu.Unlock();return nil,errors.New("unauthorized permission request")}
 id:=fmt.Sprintf("req-%d",b.next.Add(1))
 p:=&permissionWait{options:make(map[string]bool),selected:make(chan string,1)}
 choices:=make([]PermissionChoice,0,len(req.Options))
 for _,o:=range req.Options {
  if o.OptionID==""{continue}
  p.options[o.OptionID]=true
  choices=append(choices,PermissionChoice{ID:o.OptionID,Name:o.Name})
 }
 if len(choices)==0{b.mu.Unlock();return nil,errors.New("no permission choices")}
 b.pending[id]=p
 b.mu.Unlock()
 defer func(){b.mu.Lock();delete(b.pending,id);b.mu.Unlock()}()
 summary:=strings.TrimSpace(req.Title)
 if summary==""{summary="The agent requests permission"}
 b.emit(WireSignal{State:"blocked",Kind:"permission",Summary:summary,InteractionID:id,Options:choices})
 b.print("Permission "+id+": "+summary+"\n")
 for _,c:=range choices {b.print("  "+c.ID+" — "+c.Name+"\n")}
 b.print("Use /permit "+id+" <option-id> to answer.\n")
 select{
 case option:=<-p.selected:
  if option==""{return map[string]any{"outcome":map[string]string{"outcome":"cancelled"}},nil}
  return map[string]any{"outcome":map[string]string{"outcome":"selected","optionId":option}},nil
 case <-ctx.Done():return map[string]any{"outcome":map[string]string{"outcome":"cancelled"}},nil
 }
}
// Choose is an explicit, correlated action; raw y/Enter is never an approval.
func(b *Bridge)Choose(interactionID,option string)error{
 b.mu.Lock()
 defer b.mu.Unlock()
 p:=b.pending[interactionID]
 if p==nil{return errors.New("stale or unknown permission request")}
 if !p.options[option]{return errors.New("invalid permission option")}
 if p.submitted{return errors.New("permission already submitted")}
 p.submitted=true
 p.selected<-option
 return nil
}
func(b *Bridge)notification(method string,raw json.RawMessage){
 if method!="session/update"{return}
 var p struct{
  SessionID string `json:"sessionId"`
  Update struct{Kind string `json:"sessionUpdate"`;Content struct{Text string `json:"text"`} `json:"content"`} `json:"update"`
 }
 if json.Unmarshal(raw,&p)!=nil{return}
 if p.SessionID!=b.SessionID(){return}
 switch p.Update.Kind{
 case "agent_message_chunk":
  if p.Update.Content.Text!="" {b.print(p.Update.Content.Text)}
 case "tool_call","tool_call_update":
  b.emit(WireSignal{State:"working",Summary:"Agent is using tools"})
 }
}
func(b *Bridge)piEvent(kind string,raw json.RawMessage){
 switch kind{
 case "agent_start":b.emit(WireSignal{State:"working",Summary:"Agent started"})
 case "agent_settled":b.emit(WireSignal{State:"done",Summary:"Agent turn settled"})
 case "agent_end": // Does not mean settled; retry/follow-ups may follow.
 case "message_update":
  var data struct {AssistantMessageEvent struct{Type string `json:"type"`;Delta string `json:"delta"`}`json:"assistantMessageEvent"`}
  if json.Unmarshal(raw,&data)==nil&&data.AssistantMessageEvent.Type=="text_delta" {b.print(data.AssistantMessageEvent.Delta)}
 }
}
