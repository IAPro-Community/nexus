package web

import (
    "bufio"
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "net/http/cookiejar"
    "os"
    "path/filepath"
    "strings"
    "testing"
    "time"

    "github.com/IAPro-Community/nexus/internal/control/host"
    "github.com/IAPro-Community/nexus/internal/control/protocol"
    "github.com/IAPro-Community/nexus/internal/control/registry"
)

// A real child process speaking ACP JSON-RPC, not an in-memory registry state
// fixture. It requests an identical permission on each of two independent
// provider sessions and completes only after the Nexus response is received.
func TestACPProcessPermissionFixture(t *testing.T) {
    if os.Getenv("NEXUS_ACP_E2E_CHILD") != "1" { return }
    sessionID := os.Getenv("NEXUS_ACP_E2E_SESSION")
    in := bufio.NewScanner(os.Stdin)
    out := bufio.NewWriter(os.Stdout)
    send := func(message any) {
        if err := json.NewEncoder(out).Encode(message); err != nil { os.Exit(2) }
        if err := out.Flush(); err != nil { os.Exit(2) }
    }
    var promptID json.RawMessage
    for in.Scan() {
        var message struct {
            ID json.RawMessage `json:"id"`
            Method string `json:"method"`
            Result json.RawMessage `json:"result"`
        }
        if err := json.Unmarshal(in.Bytes(), &message); err != nil { os.Exit(3) }
        switch message.Method {
        case "initialize":
            send(map[string]any{"jsonrpc":"2.0", "id":message.ID,
                "result":map[string]any{"protocolVersion":1}})
        case "session/new":
            send(map[string]any{"jsonrpc":"2.0", "id":message.ID,
                "result":map[string]string{"sessionId":sessionID}})
        case "session/prompt":
            promptID = append([]byte(nil), message.ID...)
            send(map[string]any{"jsonrpc":"2.0", "id":"request-permission",
                "method":"session/request_permission",
                "params":map[string]any{
                    "sessionId":sessionID, "title":"Allow the same harmless action?",
                    "options":[]map[string]string{
                        {"optionId":"deny", "name":"Deny"},
                        {"optionId":"allow_once", "name":"Allow once"},
                    },
                },
            })
        default:
            if string(message.ID) != `"request-permission"` || len(promptID)==0 { continue }
            var result struct {
                Outcome struct {
                    Outcome string `json:"outcome"`
                    OptionID string `json:"optionId"`
                } `json:"outcome"`
            }
            if json.Unmarshal(message.Result,&result)!=nil || result.Outcome.Outcome!="selected" {
                os.Exit(4)
            }
            send(map[string]any{"jsonrpc":"2.0", "id":promptID,
                "result":map[string]string{"stopReason":"end_turn"}})
            promptID=nil
        }
    }
    os.Exit(0)
}

func TestTwoSupervisedACPProcessesAndTwoAuthenticatedHTTPClients(t *testing.T) {
    dir:=t.TempDir()
    t.Setenv("AI_MANAGER_DATA_DIR",dir)
    t.Setenv("AI_CLI_DATA_DIR",dir)
    reg:=registry.DefaultRegistry()

    launch:=func(runtimeID,projectID,sessionID string)*host.SessionHost{
        t.Helper()
        workspace:=filepath.Join(dir,projectID)
        if err:=os.MkdirAll(workspace,0700);err!=nil{t.Fatal(err)}
        s:=registry.RuntimeSession{
            RuntimeID:runtimeID,ProviderID:"opencode",ProfileID:"default",
            Workspace:workspace,ProjectID:projectID,ProjectName:projectID,
            State:registry.StateStarting,StartedAt:time.Now(),
            ControlLevel:registry.ControlLevelEvents,
            Labels:map[string]string{"nexus.agent_protocol":"acp"},
        }
        h,err:=host.NewSessionHost(host.Config{
            Session:s, Registry:reg, Binary:os.Args[0],
            Args:[]string{"-test.run=^TestACPProcessPermissionFixture$"},
            Env:append(os.Environ(),"NEXUS_ACP_E2E_CHILD=1","NEXUS_ACP_E2E_SESSION="+sessionID),
            Cwd:workspace,
        })
        if err!=nil{t.Fatal(err)}
        if err:=h.Start();err!=nil{t.Fatalf("start %s: %v",runtimeID,err)}
        return h
    }
    alpha:=launch("acp-http-agent-alpha","project-alpha","provider-alpha")
    defer func(){_ = alpha.Terminate();alpha.Wait()}()
    beta:=launch("acp-http-agent-beta","project-beta","provider-beta")
    defer func(){_ = beta.Terminate();beta.Wait()}()

    srv,err:=NewServer(ServerOptions{Host:"127.0.0.1",Port:0,NoOpen:true})
    if err!=nil{t.Fatal(err)}
    go func(){_ = srv.Start()}()
    defer srv.Shutdown(context.Background())

    client:=func()*http.Client {
        t.Helper()
        jar,err:=cookiejar.New(nil)
        if err!=nil{t.Fatal(err)}
        c:=&http.Client{Jar:jar,Timeout:3*time.Second}
        res,err:=c.Post(srv.URL()+"/api/v1/auth/bootstrap","application/json",
            strings.NewReader(fmt.Sprintf(`{"token":%q}`,srv.bootstrap)))
        if err!=nil{t.Fatal(err)}
        defer res.Body.Close()
        if res.StatusCode!=http.StatusOK{t.Fatalf("bootstrap status %d",res.StatusCode)}
        return c
    }
    desktop:=client()
    mobile:=client()
    read:=func(c *http.Client)map[string]registry.RuntimeSession{
        t.Helper()
        response,err:=c.Get(srv.URL()+"/api/v1/runtimes")
        if err!=nil{t.Fatal(err)}
        defer response.Body.Close()
        if response.StatusCode!=http.StatusOK{t.Fatalf("runtime API status %d",response.StatusCode)}
        var sessions []registry.RuntimeSession
        if err:=json.NewDecoder(response.Body).Decode(&sessions);err!=nil{t.Fatal(err)}
        result:=make(map[string]registry.RuntimeSession)
        for _,s:=range sessions{result[s.RuntimeID]=s}
        return result
    }
    submit:=func(id string){
        t.Helper()
        c,err:=protocol.NewClient(id)
        if err!=nil{t.Fatal(err)}
        defer c.Close()
        if err:=c.Ping();err!=nil{t.Fatal(err)}
        if _,err:=c.RawConn().Write([]byte("ask safely\r\n"));err!=nil{t.Fatal(err)}
    }
    submit("acp-http-agent-alpha")
    submit("acp-http-agent-beta")

    wait:=func(condition func()bool){
        t.Helper()
        deadline:=time.Now().Add(5*time.Second)
        for time.Now().Before(deadline){
            if condition(){return}
            time.Sleep(25*time.Millisecond)
        }
        t.Fatal("timed out waiting for process-backed attention transition")
    }
    wait(func()bool {
        a:=read(desktop)
        b:=read(mobile)
        alpha1,okA:=a["acp-http-agent-alpha"]
        beta1,okB:=b["acp-http-agent-beta"]
        if !okA||!okB{return false}
        return alpha1.AttentionKind=="needs_user"&&beta1.AttentionKind=="needs_user"
    })
    first:=read(desktop)
    a,b:=first["acp-http-agent-alpha"],first["acp-http-agent-beta"]
    if a.AttentionContext!=b.AttentionContext || a.AttentionFingerprint==b.AttentionFingerprint {
        t.Fatalf("same question merged across live process boundary: %q %q",a.AttentionFingerprint,b.AttentionFingerprint)
    }

    // Respond through the actual runtime IPC, not Registry.UpdateAttentionMeta.
    c,err:=protocol.NewClient("acp-http-agent-alpha")
    if err!=nil{t.Fatal(err)}
    if err:=c.Ping();err!=nil{t.Fatal(err)}
    if _,err:=c.RawConn().Write([]byte("/permit req-1 deny\r\n"));err!=nil{t.Fatal(err)}
    _=c.Close()
    wait(func()bool{
        state:=read(mobile)
        return state["acp-http-agent-alpha"].AttentionKind=="completed" &&
            state["acp-http-agent-beta"].AttentionKind=="needs_user"
    })
    if err:=beta.Terminate();err!=nil{t.Fatal(err)}
    beta.Wait()
    wait(func()bool{
        state:=read(desktop)
        return state["acp-http-agent-beta"].AttentionKind=="idle"
    })
}
