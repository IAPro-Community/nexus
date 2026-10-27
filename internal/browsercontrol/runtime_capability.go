package browsercontrol

import (
 "bufio"
 "bytes"
 "context"
 "crypto/rand"
 "crypto/sha256"
 "crypto/subtle"
 "encoding/hex"
 "encoding/json"
 "errors"
 "fmt"
 "io"
 "net"
 "net/http"
 "os"
 "path/filepath"
 "strconv"
 "strings"
 "time"

 "github.com/IAPro-Community/nexus/internal/control/registry"
 "github.com/IAPro-Community/nexus/internal/core/config"
)

type RuntimeCapability struct {
 RuntimeID string `json:"runtime_id"`
 ProjectID string `json:"project_id"`
 ExpiresAt time.Time `json:"expires_at"`
 Digest string `json:"digest"`
}
type localEndpoint struct {URL string `json:"url"`; PID int `json:"pid"`}
func browserDataDir()(string,error){dir,err:=config.DataDir();if err!=nil{return "",err};return filepath.Join(dir,"browser-capabilities"),nil}
func safeRuntimeID(id string)bool {
 if id==""||len(id)>160||filepath.Base(id)!=id||strings.ContainsAny(id,"/\\:"){return false}
 for _,r:=range id{if !(r>='a'&&r<='z'||r>='A'&&r<='Z'||r>='0'&&r<='9'||r=='-'||r=='_'){return false}}
 return true
}
func capFile(id string)(string,error){if !safeRuntimeID(id){return "",errors.New("invalid runtime id")};dir,err:=browserDataDir();if err!=nil{return "",err};return filepath.Join(dir,"runtime-"+id+".json"),nil}
func IssueRuntimeCapability(runtimeID,projectID string)(string,error){
 if strings.TrimSpace(projectID)==""||len(projectID)>200{return "",errors.New("browser project scope required")}
 path,err:=capFile(runtimeID);if err!=nil{return "",err}
 secret:=make([]byte,32);if _,err=rand.Read(secret);err!=nil{return "",err}
 token:="nbr_"+runtimeID+"."+hex.EncodeToString(secret)
 digest:=sha256.Sum256([]byte(token))
 cap:=RuntimeCapability{RuntimeID:runtimeID,ProjectID:projectID,ExpiresAt:time.Now().Add(24*time.Hour),Digest:hex.EncodeToString(digest[:])}
 data,err:=json.Marshal(cap);if err!=nil{return "",err}
 if err=os.MkdirAll(filepath.Dir(path),0700);err!=nil{return "",err}
 if err=os.Chmod(filepath.Dir(path),0700);err!=nil{return "",err}
 f,err:=os.OpenFile(path,os.O_WRONLY|os.O_CREATE|os.O_TRUNC,0600);if err!=nil{return "",err}
 _,writeErr:=f.Write(data);closeErr:=f.Close();if writeErr!=nil{return "",writeErr};if closeErr!=nil{return "",closeErr}
 return token,nil
}
func RevokeRuntimeCapability(runtimeID string){if path,err:=capFile(runtimeID);err==nil{_ = os.Remove(path)}}
func ResolveRuntimeCapability(token string)(RuntimeCapability,bool){
 if !strings.HasPrefix(token,"nbr_"){return RuntimeCapability{},false}
 parts:=strings.SplitN(strings.TrimPrefix(token,"nbr_"),".",2)
 if len(parts)!=2||len(parts[1])!=64{return RuntimeCapability{},false}
 path,err:=capFile(parts[0]);if err!=nil{return RuntimeCapability{},false}
 data,err:=os.ReadFile(path);if err!=nil{return RuntimeCapability{},false}
 var cap RuntimeCapability
 if json.Unmarshal(data,&cap)!=nil||cap.RuntimeID!=parts[0]||cap.ProjectID==""||time.Now().After(cap.ExpiresAt){return RuntimeCapability{},false}
 actual:=sha256.Sum256([]byte(token));expected,err:=hex.DecodeString(cap.Digest)
 if err!=nil||subtle.ConstantTimeCompare(expected,actual[:])!=1{return RuntimeCapability{},false}
 sess,ok:=registry.DefaultRegistry().Get(cap.RuntimeID)
 if !ok||(sess.ProjectID!=""&&sess.ProjectID!=cap.ProjectID)||!sess.HostLive(){return RuntimeCapability{},false}
 return cap,true
}
func endpointFile()(string,error){dir,err:=browserDataDir();if err!=nil{return "",err};return filepath.Join(dir,"core-endpoint.json"),nil}
func PublishBrowserEndpoint(port,pid int)error{
 if port<1||port>65535||pid<1{return errors.New("invalid core endpoint")}
 path,err:=endpointFile();if err!=nil{return err}
 if err=os.MkdirAll(filepath.Dir(path),0700);err!=nil{return err}
 if err=os.Chmod(filepath.Dir(path),0700);err!=nil{return err}
 b,err:=json.Marshal(localEndpoint{URL:"http://127.0.0.1:"+strconv.Itoa(port)+"/api/v1/browser/mcp",PID:pid});if err!=nil{return err}
 return os.WriteFile(path,b,0600)
}
func UnpublishBrowserEndpoint(pid int){path,err:=endpointFile();if err!=nil{return};b,err:=os.ReadFile(path);if err!=nil{return};var ep localEndpoint;if json.Unmarshal(b,&ep)==nil&&ep.PID==pid{_ = os.Remove(path)}}
func DiscoverBrowserEndpoint()string{
 path,err:=endpointFile();if err!=nil{return ""};b,err:=os.ReadFile(path);if err!=nil{return ""};var ep localEndpoint
 if json.Unmarshal(b,&ep)!=nil{return ""}
 hp:=strings.TrimPrefix(ep.URL,"http://127.0.0.1:")
 if !strings.HasSuffix(hp,"/api/v1/browser/mcp"){return ""}
 port:=strings.TrimSuffix(hp,"/api/v1/browser/mcp");n,err:=strconv.Atoi(port)
 if err!=nil||n<1||n>65535{return ""}
 client:=&http.Client{Timeout:350*time.Millisecond}
 resp,err:=client.Get("http://127.0.0.1:"+port+"/api/v1/health");if err!=nil{return ""};_ =resp.Body.Close();if resp.StatusCode!=http.StatusOK{return ""}
 return ep.URL
}

// A provider-native MCP stdio adapter to the same authorized Core browser.
// Secrets remain in the supervised process environment, not in MCP configs.
func ServeBrowserMCPStdio(ctx context.Context,stdin io.Reader,stdout io.Writer)error{
 token:=os.Getenv("NEXUS_BROWSER_TOKEN");endpoint:=os.Getenv("NEXUS_BROWSER_MCP_URL")
 if !strings.HasPrefix(token,"nbr_")||!strings.HasPrefix(endpoint,"http://127.0.0.1:"){return errors.New("nexus browser capability not provisioned")}
 address:=strings.TrimPrefix(endpoint,"http://");host,_,err:=net.SplitHostPort(strings.SplitN(address,"/",2)[0]);if err!=nil||host!="127.0.0.1"{return errors.New("invalid local MCP endpoint")}
 client:=&http.Client{Timeout:20*time.Second}
 scan:=bufio.NewScanner(stdin);scan.Buffer(make([]byte,4096),1024*1024)
 for scan.Scan(){
  line:=append([]byte(nil),scan.Bytes()...);if len(bytes.TrimSpace(line))==0{continue}
  req,err:=http.NewRequestWithContext(ctx,http.MethodPost,endpoint,bytes.NewReader(line));if err!=nil{return err}
  req.Header.Set("Authorization","Bearer "+token);req.Header.Set("Content-Type","application/json");req.Header.Set("Accept","application/json, text/event-stream")
  resp,err:=client.Do(req);if err!=nil{return fmt.Errorf("browser MCP broker unavailable: %w",err)}
  result,readErr:=io.ReadAll(io.LimitReader(resp.Body,2<<20));_ =resp.Body.Close();if readErr!=nil{return readErr}
  var obj struct{ID json.RawMessage `json:"id"`};_ =json.Unmarshal(line,&obj)
  if len(obj.ID)==0{continue}
  if resp.StatusCode!=http.StatusOK{return fmt.Errorf("browser MCP HTTP %d",resp.StatusCode)}
  if len(result)>0{if _,err:=stdout.Write(append(bytes.TrimSpace(result),'\n'));err!=nil{return err}}
 }
 return scan.Err()
}
