package agentwire

import ("errors";"sync")

// Process liveness, turn lifecycle and human intervention are distinct axes.
// Unknown must never be silently upgraded to complete or approved.
type Evidence string
const(
 Authoritative Evidence="AUTHORITATIVE"
 Signalled Evidence="SIGNALLED"
 Inferred Evidence="INFERRED"
 Unknown Evidence="UNKNOWN"
)
type Turn string
const(
 TurnUnknown Turn="UNKNOWN"
 TurnIdle Turn="IDLE"
 TurnRunning Turn="RUNNING"
 TurnSettled Turn="SETTLED"
 TurnError Turn="ERROR"
 TurnCancelled Turn="CANCELLED"
)
type PendingAction string
const(
 NoAction PendingAction="NONE"
 Question PendingAction="QUESTION"
 Approval PendingAction="APPROVAL"
 Authentication PendingAction="AUTH"
)
type Intervention struct{
 ID string `json:"interaction_id"`
 Kind PendingAction `json:"kind"`
 Status string `json:"status"`
 Text string `json:"text,omitempty"`
}
type Observation struct{
 RuntimeID string `json:"runtime_id"`
 Epoch uint64 `json:"epoch"`
 Sequence uint64 `json:"sequence"`
 Source string `json:"source"`
 Evidence Evidence `json:"evidence"`
 Turn Turn `json:"turn"`
 Request *Intervention `json:"request,omitempty"`
 ResolvedID string `json:"resolved_id,omitempty"`
}
type Snapshot struct{
 RuntimeID string `json:"runtime_id"`
 Epoch uint64 `json:"epoch"`
 Sequence uint64 `json:"sequence"`
 Turn Turn `json:"turn"`
 Evidence Evidence `json:"evidence"`
 Attention []Intervention `json:"attention"`
}
type Reducer struct{mu sync.Mutex;sessions map[string]Snapshot}
func NewReducer()*Reducer{return &Reducer{sessions:make(map[string]Snapshot)}}
// Stale observations and duplicate events are ignored, session restarts use a
// new epoch and independent runtimes never coalesce by question text.
func(r *Reducer)Apply(o Observation)(Snapshot,error){
 if o.RuntimeID==""||o.Epoch==0||o.Sequence==0{return Snapshot{},errors.New("invalid observation identity")}
 r.mu.Lock();defer r.mu.Unlock()
 s,ok:=r.sessions[o.RuntimeID]
 if !ok||o.Epoch>s.Epoch{s=Snapshot{RuntimeID:o.RuntimeID,Epoch:o.Epoch,Turn:TurnUnknown,Evidence:Unknown}}
 if o.Epoch<s.Epoch||o.Sequence<=s.Sequence{return clone(s),nil}
 if o.Turn!=""&&o.Turn!=TurnUnknown{s.Turn=o.Turn;s.Evidence=o.Evidence}
 if o.Request!=nil&&o.Request.ID!=""&&o.Request.Kind!=NoAction{
  present:=false
  for i:=range s.Attention{if s.Attention[i].ID==o.Request.ID{present=true;break}}
  if !present{cp:=*o.Request;cp.Status="PENDING";s.Attention=append(s.Attention,cp)}
 }
 if o.ResolvedID!=""{
  keep:=s.Attention[:0]
  for _,a:=range s.Attention{if a.ID!=o.ResolvedID{keep=append(keep,a)}}
  s.Attention=keep
 }
 if(o.Turn==TurnSettled||o.Turn==TurnCancelled||o.Turn==TurnError)&&o.Evidence==Authoritative{s.Attention=nil}
 s.Sequence=o.Sequence
 r.sessions[o.RuntimeID]=s
 return clone(s),nil
}
func(r *Reducer)Get(id string)(Snapshot,bool){r.mu.Lock();defer r.mu.Unlock();s,ok:=r.sessions[id];return clone(s),ok}
func clone(s Snapshot)Snapshot{s.Attention=append([]Intervention(nil),s.Attention...);return s}
