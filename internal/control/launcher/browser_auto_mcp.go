package launcher

import (
 "encoding/json"
 "fmt"
 "os"
 "os/exec"
 "path/filepath"
 "strings"

 "github.com/IAPro-Community/nexus/internal/browsercontrol"
 "github.com/IAPro-Community/nexus/internal/core/config"
)

// A single capability injection point shared by Web, Desktop, Mission and CLI.
func configureBrowserMCP(opts LaunchOptions, env []string,args []string)([]string,[]string,error){
 if opts.ProviderID=="shell"||opts.ProviderID=="fake"||strings.TrimSpace(opts.ProjectID)==""{return args,env,nil}
 if enabled,ok:=opts.Options["nexus_browser_enabled"].(bool);ok&&!enabled{return args,env,nil}
 endpoint:=browsercontrol.DiscoverBrowserEndpoint();if endpoint==""{return args,env,nil}
 provider:=strings.ToLower(opts.ProviderID)
 switch provider{case "codex","opencode","agy","claude","gemini","cursor":default:return args,env,nil}
 token,err:=browsercontrol.IssueRuntimeCapability(opts.RuntimeID,opts.ProjectID);if err!=nil{return args,env,fmt.Errorf("provision browser capability: %w",err)}
 cleanup:=func(e error)([]string,[]string,error){browsercontrol.RevokeRuntimeCapability(opts.RuntimeID);return args,env,e}
 env=append(env,"NEXUS_BROWSER_TOKEN="+token,"NEXUS_BROWSER_MCP_URL="+endpoint)
 switch provider{
 case "codex":
  args=append([]string{"-c",fmt.Sprintf(`mcp_servers.nexus-browser.url=%q`,endpoint),"-c",`mcp_servers.nexus-browser.bearer_token_env_var="NEXUS_BROWSER_TOKEN"`},args...)
 case "opencode":
  cfg:=map[string]any{}
  for _,item:=range env{if strings.HasPrefix(item,"OPENCODE_CONFIG_CONTENT="){value:=strings.TrimPrefix(item,"OPENCODE_CONFIG_CONTENT=");if err:=json.Unmarshal([]byte(value),&cfg);err!=nil{return cleanup(fmt.Errorf("existing OPENCODE_CONFIG_CONTENT invalid: %w",err))}}}
  mcp,_:=cfg["mcp"].(map[string]any);if mcp==nil{mcp=map[string]any{};cfg["mcp"]=mcp}
  mcp["nexus-browser"]=map[string]any{"type":"remote","url":endpoint,"enabled":true,"oauth":false,"headers":map[string]any{"Authorization":"Bearer {env:NEXUS_BROWSER_TOKEN}"}}
  out,err:=json.Marshal(cfg);if err!=nil{return cleanup(err)}
  result:=make([]string,0,len(env)+1);for _,item:=range env{if !strings.HasPrefix(item,"OPENCODE_CONFIG_CONTENT="){result=append(result,item)}};env=append(result,"OPENCODE_CONFIG_CONTENT="+string(out))
 case "agy","claude","gemini","cursor":
  bridge:=resolveBrowserBridgeBinary()
  if bridge==""{return cleanup(fmt.Errorf("nexus CLI binary unavailable for %s browser MCP bridge",provider))}
  server:=map[string]any{"command":bridge,"args":[]string{"__browser-mcp-stdio"}}
  switch provider{
  case "claude":
   f,err:=writeBrowserProviderConfig(opts.RuntimeID,"claude.json",map[string]any{"mcpServers":map[string]any{"nexus-browser":server}});if err!=nil{return cleanup(err)}
   args=append([]string{"--mcp-config",f},args...)
  case "gemini":
   // Gemini's explicitly documented per-invocation settings path.
   home:=environmentValue(env,"GEMINI_CLI_HOME")
   if home==""{home=environmentValue(env,"HOME")}
   if home==""{return cleanup(fmt.Errorf("gemini profile has no isolated HOME"))}
   target:=filepath.Join(home,".gemini","settings.json")
   if err:=mergeMCPProviderConfig(target,"mcpServers","nexus-browser",server);err!=nil{return cleanup(err)}
  case "agy":
   home:=environmentValue(env,"HOME");if home==""{return cleanup(fmt.Errorf("agy profile has no isolated HOME"))}
   target:=filepath.Join(home,".gemini","config","mcp_config.json")
   if err:=mergeMCPProviderConfig(target,"mcpServers","nexus-browser",server);err!=nil{return cleanup(err)}
  case "cursor":
   home:=environmentValue(env,"CURSOR_HOME");if home==""{return cleanup(fmt.Errorf("cursor profile has no isolated HOME"))}
   target:=filepath.Join(home,".cursor","mcp.json")
   if err:=mergeMCPProviderConfig(target,"mcpServers","nexus-browser",server);err!=nil{return cleanup(err)}
  }
 }
 return args,env,nil
}
func environmentValue(env []string,key string)string{for i:=len(env)-1;i>=0;i--{if strings.HasPrefix(env[i],key+"="){return strings.TrimPrefix(env[i],key+"=")}};return ""}
func resolveBrowserBridgeBinary()string{
 if p,err:=exec.LookPath("nexus");err==nil{return p}
 if p,err:=exec.LookPath("ai");err==nil{return p}
 self,err:=os.Executable();if err!=nil{return ""}
 base:=strings.ToLower(strings.TrimSuffix(filepath.Base(self),".exe"));if base=="nexus"||base=="ai"{return self}
 return ""
}
func writeBrowserProviderConfig(runtimeID,name string,cfg any)(string,error){
 dirRoot,err:=config.DataDir();if err!=nil{return "",err}
 if runtimeID==""||filepath.Base(runtimeID)!=runtimeID||strings.ContainsAny(runtimeID,"/\\:"){return "",fmt.Errorf("invalid runtime scope")}
 dir:=filepath.Join(dirRoot,"runtime","browser-mcp",runtimeID)
 if err=os.MkdirAll(dir,0700);err!=nil{return "",err}
 f:=filepath.Join(dir,name);body,err:=json.MarshalIndent(cfg,"","  ");if err!=nil{return "",err}
 if err=os.WriteFile(f,body,0600);err!=nil{return "",err}
 return f,nil
}
func mergeMCPProviderConfig(path,field,name string,server any)error{
 if err:=os.MkdirAll(filepath.Dir(path),0700);err!=nil{return err}
 obj:=map[string]any{}
 if b,err:=os.ReadFile(path);err==nil{if err=json.Unmarshal(b,&obj);err!=nil{return fmt.Errorf("existing MCP config %s: %w",path,err)}}else if !os.IsNotExist(err){return err}
 children,_:=obj[field].(map[string]any);if children==nil{children=map[string]any{};obj[field]=children}
 if current,ok:=children[name];ok{b1,_:=json.Marshal(current);b2,_:=json.Marshal(server);if string(b1)!=string(b2){return fmt.Errorf("existing %s MCP server config differs",name)};return nil}
 children[name]=server
 data,err:=json.MarshalIndent(obj,"","  ");if err!=nil{return err}
 tmp:=path+".nexus-tmp";if err=os.WriteFile(tmp,data,0600);err!=nil{return err}
 return os.Rename(tmp,path)
}
