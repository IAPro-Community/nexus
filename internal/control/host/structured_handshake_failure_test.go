package host

import (
    "bufio"
    "encoding/json"
    "os"
    "os/exec"
    "strings"
    "testing"
)

// An incompatible provider responds to initialize but must never be treated
// as a RUNNING runtime. Start owns child reaping on this failure path.
func TestMockIncompatibleACPAgent(t *testing.T) {
    if os.Getenv("NEXUS_WIRE_INCOMPATIBLE_CHILD") != "1" { return }
    scanner := bufio.NewScanner(os.Stdin)
    if scanner.Scan() {
        var request struct { ID json.RawMessage `json:"id"` }
        if json.Unmarshal(scanner.Bytes(), &request)!=nil { os.Exit(3) }
        _ = json.NewEncoder(os.Stdout).Encode(map[string]any{
            "jsonrpc":"2.0",
            "id":request.ID,
            "result":map[string]any{"protocolVersion": 999},
        })
    }
    for scanner.Scan() {} // parent kills the process after version rejection
    os.Exit(0)
}

func TestStructuredBackendReapsProcessOnFailedHandshake(t *testing.T) {
    b := newStructuredBackend("acp", t.TempDir(), nil, nil)
    child := exec.Command(os.Args[0], "-test.run=^TestMockIncompatibleACPAgent$")
    child.Env = append(os.Environ(), "NEXUS_WIRE_INCOMPATIBLE_CHILD=1")
    if err := b.Start(child, 24, 80); err==nil {
        t.Fatal("incompatible ACP protocol accepted")
    } else if !strings.Contains(err.Error(), "version mismatch") {
        t.Fatalf("unexpected handshake failure: %v", err)
    }
    if child.ProcessState == nil {
        t.Fatal("failed handshake child was not reaped with Wait")
    }
    _ = b.Close()
}
