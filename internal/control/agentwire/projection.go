package agentwire

// ObserveWireSignal projects a verified protocol-frame notification into the
// shared runtime/turn/intervention reducer. PTY narration never enters here.
// A command acknowledgement and an operator keystroke are not evidence that
// a provider resolved an intervention.
func (r *Reducer) ObserveWireSignal(runtimeID string, epoch, sequence uint64, signal WireSignal) (Snapshot, error) {
 observation := Observation{
  RuntimeID: runtimeID,
  Epoch: epoch,
  Sequence: sequence,
  Source: "structured_provider",
  Evidence: Signalled,
  Turn: TurnUnknown,
 }
 switch signal.State {
 case "blocked":
  if signal.InteractionID != "" {
   kind := Question
   if signal.Kind == "permission" { kind = Approval }
   if signal.Kind == "auth" { kind = Authentication }
   observation.Request = &Intervention{
    ID: signal.InteractionID, Kind: kind, Text: signal.Summary,
   }
  }
 case "working":
  observation.Turn = TurnRunning
 case "idle":
  observation.Turn = TurnIdle
 case "done":
  // ACP prompt completion and Pi agent_settled are protocol-owned terminal
  // turn events. Only these can close all pending requests in this adapter.
  observation.Turn = TurnSettled
  observation.Evidence = Authoritative
 case "error":
  observation.Turn = TurnError
 default:
  // Do not manufacture a state transition for unknown protocol events.
 }
 return r.Apply(observation)
}
