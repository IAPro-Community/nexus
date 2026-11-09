package host

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/agentwire"
	"github.com/IAPro-Community/nexus/internal/control/protocol"
	"github.com/IAPro-Community/nexus/internal/control/registry"
)

// TestMockWirePermissionPipelineProcess is a deterministic ACP child process
// used to verify the real Bridge → SessionHost → Registry event path.
func TestMockWirePermissionPipelineProcess(t *testing.T) {
	if os.Getenv("NEXUS_WIRE_PERMISSION_CHILD") != "1" {
		return
	}
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 0, 4096), 1<<20)
	writer := bufio.NewWriter(os.Stdout)
	write := func(frame any) bool {
		if json.NewEncoder(writer).Encode(frame) != nil {
			return false
		}
		return writer.Flush() == nil
	}
	var promptID json.RawMessage
	for scanner.Scan() {
		var frame struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
		}
		if json.Unmarshal(scanner.Bytes(), &frame) != nil {
			os.Exit(3)
		}
		switch frame.Method {
		case "initialize":
			if !write(map[string]any{"jsonrpc": "2.0", "id": frame.ID, "result": map[string]any{"protocolVersion": 1}}) {
				os.Exit(4)
			}
		case "session/new":
			if !write(map[string]any{"jsonrpc": "2.0", "id": frame.ID, "result": map[string]string{"sessionId": "permission-session"}}) {
				os.Exit(4)
			}
		case "session/prompt":
			promptID = append(json.RawMessage(nil), frame.ID...)
			if !write(map[string]any{
				"jsonrpc": "2.0", "id": "provider-permission-1", "method": "session/request_permission",
				"params": map[string]any{
					"sessionId": "permission-session",
					"title":     "Approve the same structured action?",
					"options":   []map[string]string{{"optionId": "allow_once", "name": "Allow once"}},
				},
			}) {
				os.Exit(4)
			}
		default:
			if frame.Method != "" || string(frame.ID) != `"provider-permission-1"` {
				continue
			}
			var response struct {
				Result struct {
					Outcome struct {
						Outcome string `json:"outcome"`
					} `json:"outcome"`
				} `json:"result"`
			}
			if json.Unmarshal(scanner.Bytes(), &response) != nil || response.Result.Outcome.Outcome != "selected" {
				continue
			}
			if !write(map[string]any{
				"jsonrpc": "2.0", "method": "session/update",
				"params": map[string]any{
					"sessionId": "permission-session",
					"update":    map[string]string{"sessionUpdate": "tool_call"},
				},
			}) {
				os.Exit(4)
			}
			if err := os.WriteFile(os.Getenv("NEXUS_WIRE_ACK_MARKER"), []byte("acknowledged"), 0o600); err != nil {
				os.Exit(5)
			}
			releaseMarker := os.Getenv("NEXUS_WIRE_RELEASE_MARKER")
			deadline := time.Now().Add(5 * time.Second)
			for {
				if _, err := os.Stat(releaseMarker); err == nil {
					break
				}
				if time.Now().After(deadline) {
					os.Exit(6)
				}
				time.Sleep(5 * time.Millisecond)
			}
			if !write(map[string]any{"jsonrpc": "2.0", "id": promptID, "result": map[string]string{"stopReason": "end_turn"}}) {
				os.Exit(4)
			}
		}
	}
	os.Exit(0)
}

func TestStructuredACPAttentionReachesRegistryAndSettlesOnlyItsRuntime(t *testing.T) {
	root := t.TempDir()
	reg := registry.NewRegistry(filepath.Join(root, "runtimes.json"))
	type runtimeHarness struct {
		id            string
		host          *SessionHost
		ackMarker     string
		releaseMarker string
	}
	harnesses := make([]runtimeHarness, 0, 2)
	for _, id := range []string{"structured-a3-alpha", "structured-a3-beta"} {
		cwd := filepath.Join(root, id)
		if err := os.MkdirAll(cwd, 0o700); err != nil {
			t.Fatal(err)
		}
		ackMarker := filepath.Join(root, id+"-ack")
		releaseMarker := filepath.Join(root, id+"-release")
		session := registry.RuntimeSession{
			RuntimeID: id, ProviderID: "opencode", ProfileID: "default", Workspace: cwd,
			State: registry.StateStarting, ControlLevel: registry.ControlLevelEvents,
			ProjectID: "project-a3", ProjectName: "A3 integration",
			AgentID: "agent-" + strings.TrimPrefix(id, "structured-a3-"),
			Labels:  map[string]string{"nexus.agent_protocol": "acp"},
		}
		host, err := NewSessionHost(Config{
			Session: session, Registry: reg, Binary: os.Args[0],
			Args: []string{"-test.run=^TestMockWirePermissionPipelineProcess$"},
			Env: append(os.Environ(), "NEXUS_WIRE_PERMISSION_CHILD=1",
				"NEXUS_WIRE_ACK_MARKER="+ackMarker, "NEXUS_WIRE_RELEASE_MARKER="+releaseMarker),
			Cwd: cwd,
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := host.Start(); err != nil {
			t.Fatal(err)
		}
		harnesses = append(harnesses, runtimeHarness{id: id, host: host, ackMarker: ackMarker, releaseMarker: releaseMarker})
	}
	defer func() {
		for _, harness := range harnesses {
			_ = harness.host.Terminate()
			harness.host.Wait()
		}
	}()

	for _, harness := range harnesses {
		backend, ok := harness.host.termBackend.(*structuredBackend)
		if !ok {
			t.Fatalf("runtime %s has backend %T, want structured ACP", harness.id, harness.host.termBackend)
		}
		if _, err := backend.Write([]byte("continue safely\r")); err != nil {
			t.Fatalf("submit prompt for %s: %v", harness.id, err)
		}
	}

	waitFor := func(label string, check func() bool) {
		t.Helper()
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) {
			if check() {
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
		t.Fatalf("timed out waiting for %s", label)
	}
	for _, harness := range harnesses {
		waitFor("structured permission in registry for "+harness.id, func() bool {
			session, ok := reg.Get(harness.id)
			return ok && session.AttentionKind == AttentionNeedsUser && session.AttentionInteractionID == "req-1"
		})
		session, _ := reg.Get(harness.id)
		if session.AttentionReason != "APPROVAL" || session.PromptKind != "choice" || session.AttentionSource != "AUTHORITATIVE" || len(session.AttentionOptions) != 1 || session.AttentionOptions[0].ID != "allow_once" {
			t.Fatalf("structured source/provenance was not projected for %s: %+v", harness.id, session)
		}
		if session.AttentionFingerprint == "" || session.AttentionRevision == 0 || session.HostGeneration == 0 || session.ProviderGeneration == 0 {
			t.Fatalf("interaction was not bound to durable runtime identity for %s: %+v", harness.id, session)
		}
	}
	alpha, _ := reg.Get(harnesses[0].id)
	beta, _ := reg.Get(harnesses[1].id)
	if alpha.AttentionFingerprint == beta.AttentionFingerprint {
		t.Fatalf("identical interactions from separate runtimes collided: alpha=%+v beta=%+v", alpha, beta)
	}

	alphaBackend := harnesses[0].host.termBackend.(*structuredBackend)
	alphaSnapshot, ok := harnesses[0].host.wireReducer.Get(harnesses[0].id)
	if !ok || len(alphaSnapshot.Attention) != 1 {
		t.Fatalf("alpha reducer did not retain its structured question: %+v", alphaSnapshot)
	}
	if err := alphaBackend.ChooseInteraction("req-1", "unknown_option"); err == nil {
		t.Fatal("provider bridge accepted an option outside its exact choice set")
	}
	client, err := protocol.NewClient(harnesses[0].id)
	if err != nil {
		t.Fatal(err)
	}
	choice, err := client.Send(protocol.CmdChooseInteraction, protocol.ChooseInteractionPayload{
		InteractionID: "req-1", OptionID: "allow_once",
	})
	_ = client.Close()
	if err != nil || !choice.OK || choice.Code != "INTERACTION_CHOICE_SUBMITTED" {
		t.Fatalf("structured choice IPC failed: response=%+v error=%v", choice, err)
	}
	waitFor("provider permission acknowledgement", func() bool {
		_, err := os.Stat(harnesses[0].ackMarker)
		return err == nil
	})
	waitFor("non-terminal provider progress event", func() bool {
		current, ok := harnesses[0].host.wireReducer.Get(harnesses[0].id)
		return ok && current.Sequence > alphaSnapshot.Sequence && current.Turn == agentwire.TurnRunning
	})
	progressSnapshot, _ := harnesses[0].host.wireReducer.Get(harnesses[0].id)
	if len(progressSnapshot.Attention) != 1 || progressSnapshot.Evidence != agentwire.Signalled { //nolint:misspell // Canonical evidence enum.
		t.Fatalf("provider progress was treated as authoritative settlement: %+v", progressSnapshot)
	}
	alpha, _ = reg.Get(harnesses[0].id)
	beta, _ = reg.Get(harnesses[1].id)
	if alpha.AttentionKind != AttentionWorking || alpha.AttentionInteractionID != "req-1" ||
		alpha.AttentionReason != "WORKING" || alpha.AttentionSource != "SIGNALLED" || beta.AttentionKind != AttentionNeedsUser { //nolint:misspell // Canonical Registry enum.
		t.Fatalf("correlated response must become working but not completed, without crossing runtimes: alpha=%+v beta=%+v", alpha, beta)
	}
	if progressSnapshot.Attention[0].Status != "SUBMITTED" {
		t.Fatalf("correlated request did not record its submitted status: %+v", progressSnapshot)
	}
	if err := os.WriteFile(harnesses[0].releaseMarker, []byte("provider settled"), 0o600); err != nil {
		t.Fatal(err)
	}
	waitFor("authoritative settlement in alpha", func() bool {
		session, ok := reg.Get(harnesses[0].id)
		return ok && session.AttentionKind == AttentionCompleted
	})
	alpha, _ = reg.Get(harnesses[0].id)
	beta, _ = reg.Get(harnesses[1].id)
	if beta.AttentionKind != AttentionNeedsUser || beta.AttentionInteractionID != "req-1" || beta.AttentionSource != "AUTHORITATIVE" {
		t.Fatalf("settling alpha mutated beta's identical interaction: alpha=%+v beta=%+v", alpha, beta)
	}
	alphaSnapshot, _ = harnesses[0].host.wireReducer.Get(harnesses[0].id)
	betaSnapshot, _ := harnesses[1].host.wireReducer.Get(harnesses[1].id)
	if len(alphaSnapshot.Attention) != 0 || len(betaSnapshot.Attention) != 1 {
		t.Fatalf("structured reducer crossed runtime boundary: alpha=%+v beta=%+v", alphaSnapshot, betaSnapshot)
	}
	if alphaSnapshot.Turn != agentwire.TurnSettled || alphaSnapshot.Evidence != agentwire.Authoritative {
		t.Fatalf("provider turn result was not authoritative: %+v", alphaSnapshot)
	}
	if alpha.AttentionSource != "AUTHORITATIVE" {
		t.Fatalf("authoritative settlement lost its provenance: %+v", alpha)
	}
}
