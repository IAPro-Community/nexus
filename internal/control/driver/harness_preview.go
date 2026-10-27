package driver

import (
 "context"
 "fmt"
 "os"
 "path/filepath"
 "strings"

 "github.com/IAPro-Community/nexus/internal/control/registry"
 "github.com/IAPro-Community/nexus/internal/core/config"
 "github.com/IAPro-Community/nexus/internal/core/model"
 "github.com/IAPro-Community/nexus/internal/runtime"
)

// PreviewHarnessDriver supports supervised interactive PTY launch for new
// harnesses. It deliberately reports structured control as UNSUPPORTED until
// the agentwire clients are connected to SessionHost lifecycle and permissions.
// Never launch ACP/JSONL into a PTY (protocol output is not a terminal).
type PreviewHarnessDriver struct{ id string }
func NewPreviewHarnessDriver(id string)*PreviewHarnessDriver{
 switch id {case "pi","omp","mimo","dsh":return &PreviewHarnessDriver{id:id}}
 panic("unrecognized preview harness")
}
func(d *PreviewHarnessDriver)ProviderID()string{return d.id}
func(d *PreviewHarnessDriver)Detect(ctx context.Context)(model.DetectionResult,error){
 binary,err:=runtime.LookPath(d.id)
 if err!=nil{return model.DetectionResult{Installed:false,Error:d.id+" executable not found"},nil}
 text,err:=runtime.RunCommandCapture(ctx,binary,[]string{"--version"},os.Environ(),"")
 if err!=nil{return model.DetectionResult{Installed:false,BinaryPath:binary,Error:strings.TrimSpace(text)},nil}
 lines:=strings.Split(strings.TrimSpace(text),"\n")
 return model.DetectionResult{Installed:true,BinaryPath:binary,Version:lines[0]},nil
}
func(d *PreviewHarnessDriver)EffectiveCaps(ctx context.Context,_ model.Profile)EffectiveCapabilities{
 detected,_:=d.Detect(ctx)
 evidence:=CapabilityEvidence{Status:CapabilitySupported,ProviderVersion:detected.Version,Mechanism:"supervised PTY / ConPTY",Tested:false}
 no:=CapabilityEvidence{Status:CapabilityUnsupported,ProviderVersion:detected.Version,Reason:"not connected to SessionHost structured control",Tested:false}
 terminal:=evidence
 if d.id=="dsh"{
  terminal=CapabilityEvidence{Status:CapabilityUnsupported,Reason:"DSH uses ACP/SDK; interactive TUI launch has not been validated",Tested:false}
 }
 return EffectiveCapabilities{
  Process:evidence,Terminal:terminal,Attach:evidence,
  StructuredEvents:no,Approvals:no,
  Sessions:CapabilityEvidence{Status:CapabilityPartial,Reason:"session semantics require provider E2E validation",Tested:false},
  Resume:no,Fork:no,
  SubmitPrompt:no,CancelTurn:no,Headless:no,SlashControl:no,
  ControlLevel:registry.ControlLevelTerminal,
 }
}
func(d *PreviewHarnessDriver)Capabilities(ctx context.Context,p model.Profile)ControlCapabilities{
 return d.EffectiveCaps(ctx,p).ToBooleanCaps()
}
func(d *PreviewHarnessDriver)BuildCommand(ctx context.Context,p model.Profile,args []string)(string,[]string,[]string,error){
 if d.id=="dsh"{return "",nil,nil,fmt.Errorf("dsh requires ACP controlled execution; not safe to present as interactive PTY")}
 bin,err:=runtime.LookPath(d.id)
 if err!=nil{return "",nil,nil,err}
 home,err:=config.ProfileHome(d.id,p.Name)
 if err!=nil{return "",nil,nil,err}
 if err:=os.MkdirAll(home,0700);err!=nil{return "",nil,nil,err}
 env:=append([]string{},os.Environ()...)
 env=append(env,"HOME="+home,"USERPROFILE="+home,"AI_PROVIDER="+d.id,"AI_PROFILE="+p.Name)
 if d.id=="pi"||d.id=="omp"{
  env=append(env,"PI_CODING_AGENT_DIR="+filepath.Join(home,".pi","agent"))
 }
 if d.id=="mimo"{
  env=append(env,"XDG_CONFIG_HOME="+filepath.Join(home,".config"),"XDG_DATA_HOME="+filepath.Join(home,".local","share"),"XDG_CACHE_HOME="+filepath.Join(home,".cache"))
 }
 return bin,args,env,nil
}
func(d *PreviewHarnessDriver)CanResume(ctx context.Context,p model.Profile,sessionID string)(bool,string){
 return false,"resume not certified for "+d.id
}
func(d *PreviewHarnessDriver)BuildResumeArgs(ctx context.Context,p model.Profile,sessionID string)([]string,error){
 return nil,fmt.Errorf("%s resume is not yet certified",d.id)
}
func(d *PreviewHarnessDriver)BuildKickoffArgs(ctx context.Context,p model.Profile,prompt string)([]string,error){
 return nil,fmt.Errorf("%s structured kickoff is not yet certified",d.id)
}
