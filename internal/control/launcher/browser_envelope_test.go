package launcher
import (
 "os"
 "reflect"
 "testing"
)
func TestBrowserLaunchEnvelopeTransportsOnlyScopedRuntimeEnvironment(t *testing.T){
 t.Setenv("NEXUS_DATA_DIR",t.TempDir())
 id:="opencode-browser-handoff-1"
 args:=[]string{"run","test"}
 env:=[]string{"NEXUS_BROWSER_TOKEN=nbr_runtime.test","NEXUS_BROWSER_MCP_URL=http://127.0.0.1:13000/api/v1/browser/mcp","OPENCODE_CONFIG_CONTENT={\"mcp\":{}}","SENSITIVE_OTHER_SERVICE_KEY=must-not-persist"}
 if err:=createLaunchEnvelopeWithBrowser(id,args,false,env);err!=nil{t.Fatal(err)}
 path,err:=launchEnvelopePath(id);if err!=nil{t.Fatal(err)}
 info,err:=os.Stat(path);if err!=nil{t.Fatal(err)}
 if info.Mode().Perm()&0077!=0{t.Fatal("launch envelope permissions are too permissive")}
 recovered,custom,vars,err:=ConsumeLaunchEnvelopeWithBrowser(id)
 if err!=nil||custom||!reflect.DeepEqual(recovered,args){t.Fatalf("reconstructed envelope mismatch: %v %v %v",err,recovered,custom)}
 if vars["NEXUS_BROWSER_TOKEN"]!="nbr_runtime.test"||vars["OPENCODE_CONFIG_CONTENT"]==""{t.Fatal("missing scoped browser environment")}
 if vars["SENSITIVE_OTHER_SERVICE_KEY"]!=""{t.Fatal("unrelated secret transported")}
 if _,err:=os.Stat(path);!os.IsNotExist(err){t.Fatal("one-time envelope was not consumed")}
}
