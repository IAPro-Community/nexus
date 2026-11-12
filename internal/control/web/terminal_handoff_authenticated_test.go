package web

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/events"
	"github.com/IAPro-Community/nexus/internal/control/host"
	"github.com/IAPro-Community/nexus/internal/control/ids"
	"github.com/IAPro-Community/nexus/internal/control/protocol"
	"github.com/IAPro-Community/nexus/internal/control/registry"
	"github.com/IAPro-Community/nexus/internal/nexus"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
	"github.com/gorilla/websocket"
)

// The supervised child is the current test binary. It writes every accepted
// input line to a file so the integration proof observes process-side effects,
// not just the broker's in-memory lease label.
func init() {
	if os.Getenv("NEXUS_HANDOFF_CHILD") != "1" {
		return
	}
	markerFile := os.Getenv("NEXUS_HANDOFF_MARKER_FILE")
	reader := bufio.NewReader(os.Stdin)
	var line strings.Builder
	for {
		value, err := reader.ReadByte()
		if err != nil {
			os.Exit(0)
		}
		if value == '\r' || value == '\n' {
			if line.Len() > 0 {
				marker := line.String()
				file, openErr := os.OpenFile(markerFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
				if openErr == nil {
					_, _ = fmt.Fprintln(file, marker)
					_ = file.Close()
				}
				_, _ = fmt.Fprintln(os.Stdout, marker)
				line.Reset()
			}
			continue
		}
		line.WriteByte(value)
	}
}

type terminalHandoffWSClient struct {
	conn     *websocket.Conn
	messages chan TerminalMessage
	pongs    chan string
	errors   chan error
}

func dialAuthenticatedTerminal(t *testing.T, target string, session *Session, origin string) *terminalHandoffWSClient {
	t.Helper()
	header := http.Header{}
	header.Set("Origin", origin)
	header.Set("Cookie", (&http.Cookie{Name: sessionCookieName, Value: session.ID}).String())
	conn, response, err := websocket.DefaultDialer.Dial(target, header)
	if err != nil {
		status := 0
		if response != nil {
			status = response.StatusCode
		}
		t.Fatalf("authenticated terminal dial returned HTTP %d: %v", status, err)
	}
	client := &terminalHandoffWSClient{
		conn: conn, messages: make(chan TerminalMessage, 128),
		pongs: make(chan string, 8), errors: make(chan error, 1),
	}
	conn.SetPongHandler(func(data string) error {
		select {
		case client.pongs <- data:
		default:
		}
		return nil
	})
	go func() {
		defer close(client.messages)
		for {
			_, payload, readErr := conn.ReadMessage()
			if readErr != nil {
				select {
				case client.errors <- readErr:
				default:
				}
				return
			}
			var message TerminalMessage
			if json.Unmarshal(payload, &message) == nil {
				select {
				case client.messages <- message:
				default:
					select {
					case client.errors <- fmt.Errorf("terminal message queue overflow"):
					default:
					}
					return
				}
			}
		}
	}()
	return client
}

func (c *terminalHandoffWSClient) close() {
	if c != nil && c.conn != nil {
		_ = c.conn.Close()
	}
}

func (c *terminalHandoffWSClient) awaitLease(t *testing.T, role string) {
	t.Helper()
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	for {
		select {
		case message, ok := <-c.messages:
			if !ok {
				t.Fatalf("terminal websocket closed before lease %s", role)
			}
			if message.Type == "lease" && message.Role == role {
				return
			}
		case err := <-c.errors:
			t.Fatalf("terminal websocket read failed before lease %s: %v", role, err)
		case <-timer.C:
			t.Fatalf("timed out waiting for terminal lease %s", role)
		}
	}
}

func (c *terminalHandoffWSClient) awaitPong(t *testing.T, expected string) {
	t.Helper()
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	for {
		select {
		case received := <-c.pongs:
			if received == expected {
				return
			}
		case err := <-c.errors:
			t.Fatalf("terminal websocket closed before pong barrier: %v", err)
		case <-timer.C:
			t.Fatalf("timed out waiting for terminal input barrier %q", expected)
		}
	}
}

// awaitMarkerContains polls the supervised child's marker file until the
// expected line is appended, without assuming the write ordering relative to
// the WebSocket echo.
func awaitMarkerContains(t *testing.T, path, marker string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		data, err := os.ReadFile(path)
		if err == nil && strings.Contains(string(data), marker) {
			return
		}
		if time.Now().After(deadline) {
			current := ""
			if data, readErr := os.ReadFile(path); readErr == nil {
				current = string(data)
			}
			t.Fatalf("marker %q never reached %s; current content: %q", marker, path, current)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func (c *terminalHandoffWSClient) awaitClosed(t *testing.T) {
	t.Helper()
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	for {
		select {
		case _, ok := <-c.messages:
			if !ok {
				return
			}
		case <-c.errors:
			return
		case <-timer.C:
			t.Fatal("terminal websocket remained open after session revocation")
		}
	}
}

func (c *terminalHandoffWSClient) awaitOutput(t *testing.T, marker string) string {
	t.Helper()
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	var output strings.Builder
	for !strings.Contains(output.String(), marker) {
		select {
		case message, ok := <-c.messages:
			if !ok {
				t.Fatalf("terminal websocket closed before output %q: %q", marker, output.String())
			}
			if message.Type == "output" {
				output.WriteString(message.Data)
			}
		case err := <-c.errors:
			t.Fatalf("terminal websocket read failed before output %q: %v", marker, err)
		case <-timer.C:
			t.Fatalf("timed out waiting for process output %q: %q", marker, output.String())
		}
	}
	return output.String()
}

func TestTerminalHandoffAuthenticatedClientsShareProcessAndPersistIdentity(t *testing.T) {
	root := t.TempDir()
	t.Setenv("NEXUS_DATA_DIR", root)
	t.Setenv("AI_MANAGER_DATA_DIR", root)
	t.Setenv("AI_CLI_DATA_DIR", root)

	srv, err := NewServer(ServerOptions{Host: "127.0.0.1", Port: 0, NoOpen: true})
	if err != nil {
		t.Fatalf("create server: %v", err)
	}
	go func() { _ = srv.Start() }()
	defer srv.Shutdown(context.Background())

	service := nexus.Default()
	st := service.Store()
	if st == nil {
		t.Fatal("Nexus durable store unavailable")
	}
	projectID := "prj_" + ids.NewULID()
	agentID := "agt_" + ids.NewULID()
	runtimeID := "rt_" + ids.NewULID()
	project, err := st.CreateProject(store.Project{
		ID: projectID, Name: "Authenticated Device Handoff", Slug: "handoff-" + strings.ToLower(ids.NewULID()),
		CanonicalPath: root,
	})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	if _, err := st.CreateAgent(store.Agent{ID: agentID, ProjectID: project.ID, Name: "Handoff", Role: "test"}); err != nil {
		t.Fatalf("create agent: %v", err)
	}
	providerBinary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	providerFingerprint, err := registry.BinaryFingerprint(providerBinary)
	if err != nil {
		t.Fatal(err)
	}
	markerFile := filepath.Join(root, "provider-inputs.log")
	workspace := filepath.Join(root, "workspace")
	if err := os.MkdirAll(workspace, 0o700); err != nil {
		t.Fatal(err)
	}
	runtimeSession := registry.RuntimeSession{
		RuntimeID: runtimeID, ProviderID: "test", ProfileID: "default",
		ProjectID: projectID, ProjectName: project.Name, AgentID: agentID,
		Workspace: workspace, State: registry.StateStarting, StartedAt: time.Now(),
		ControlLevel:       registry.ControlLevelTerminal,
		ProviderBinaryPath: providerBinary, ProviderBinaryFingerprint: providerFingerprint,
	}
	process, err := host.NewSessionHost(host.Config{
		Session: runtimeSession, Registry: service.RuntimeRegistry(), Binary: providerBinary,
		Args: []string{"-test.run=^TestTerminalHandoffAuthenticatedChild$"},
		Env:  append(os.Environ(), "NEXUS_HANDOFF_CHILD=1", "NEXUS_HANDOFF_MARKER_FILE="+markerFile),
		Cwd:  workspace,
	})
	if err != nil {
		t.Fatalf("create supervised process: %v", err)
	}
	if err := process.Start(); err != nil {
		t.Fatalf("start supervised process: %v", err)
	}
	defer func() {
		_ = process.Terminate()
		process.Wait()
	}()
	if err := protocol.WaitForEndpoint(context.Background(), runtimeID, 3*time.Second); err != nil {
		t.Fatalf("wait for supervised runtime: %v", err)
	}

	alphaSession, err := srv.auth.CreateSession()
	if err != nil {
		t.Fatalf("create desktop session: %v", err)
	}
	betaSession, err := srv.auth.CreateSession()
	if err != nil {
		t.Fatalf("create mobile session: %v", err)
	}
	if alphaSession.ID == betaSession.ID {
		t.Fatal("expected two distinct authenticated sessions")
	}
	baseURL, err := url.Parse(srv.URL())
	if err != nil {
		t.Fatal(err)
	}
	origin := baseURL.Scheme + "://" + baseURL.Host
	httpClient := &http.Client{Timeout: 5 * time.Second}
	readRuntime := func(session *Session) registry.RuntimeSession {
		t.Helper()
		request, err := http.NewRequest(http.MethodGet, srv.URL()+"/api/v1/runtimes", nil)
		if err != nil {
			t.Fatal(err)
		}
		request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: session.ID})
		response, err := httpClient.Do(request)
		if err != nil {
			t.Fatalf("read runtime for authenticated client: %v", err)
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			t.Fatalf("runtime API returned %d", response.StatusCode)
		}
		var sessions []registry.RuntimeSession
		if err := json.NewDecoder(response.Body).Decode(&sessions); err != nil {
			t.Fatal(err)
		}
		for _, candidate := range sessions {
			if candidate.RuntimeID == runtimeID {
				return candidate
			}
		}
		t.Fatalf("runtime %s not visible to authenticated client", runtimeID)
		return registry.RuntimeSession{}
	}
	alphaRuntime := readRuntime(alphaSession)
	betaRuntime := readRuntime(betaSession)
	if alphaRuntime.PID == 0 || alphaRuntime.HostPID == 0 || alphaRuntime.PID != betaRuntime.PID ||
		alphaRuntime.HostPID != betaRuntime.HostPID || alphaRuntime.HostGeneration != betaRuntime.HostGeneration ||
		alphaRuntime.RuntimeID != betaRuntime.RuntimeID {
		t.Fatalf("authenticated clients did not map to the same live process: alpha=%+v beta=%+v", alphaRuntime, betaRuntime)
	}

	wsURL := "ws" + strings.TrimPrefix(srv.URL(), "http") + "/api/v1/agents/" + agentID + "/terminal?runtime_id=" + runtimeID
	alpha := dialAuthenticatedTerminal(t, wsURL, alphaSession, origin)
	defer alpha.close()
	alpha.awaitLease(t, "CONTROL")
	beta := dialAuthenticatedTerminal(t, wsURL, betaSession, origin)
	defer beta.close()
	beta.awaitLease(t, "VIEW_ONLY")

	if err := beta.conn.WriteJSON(TerminalMessage{Type: "lease_acquire"}); err != nil {
		t.Fatalf("request mobile takeover: %v", err)
	}
	alpha.awaitLease(t, "VIEW_ONLY")
	beta.awaitLease(t, "CONTROL")
	if err := alpha.conn.WriteJSON(TerminalMessage{Type: "input", Data: "DENIED_ALPHA_INPUT\n"}); err != nil {
		t.Fatalf("send read-only client input: %v", err)
	}
	barrier := "alpha-input-processed"
	if err := alpha.conn.WriteControl(websocket.PingMessage, []byte(barrier), time.Now().Add(time.Second)); err != nil {
		t.Fatalf("write read-only input barrier: %v", err)
	}
	alpha.awaitPong(t, barrier)
	if err := beta.conn.WriteJSON(TerminalMessage{Type: "input", Data: "EXECUTED_BETA_ONCE\n"}); err != nil {
		t.Fatalf("send controller input: %v", err)
	}

	processDeadline := time.Now().Add(5 * time.Second)
	var processInput string
	for time.Now().Before(processDeadline) {
		data, readErr := os.ReadFile(markerFile)
		if readErr == nil {
			processInput = string(data)
			if strings.Contains(processInput, "EXECUTED_BETA_ONCE") {
				break
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	if strings.Count(processInput, "EXECUTED_BETA_ONCE") != 1 || strings.Contains(processInput, "DENIED_ALPHA_INPUT") {
		t.Fatalf("provider process received anything other than one leased input: %q", processInput)
	}
	if alphaOutput := alpha.awaitOutput(t, "EXECUTED_BETA_ONCE"); strings.Contains(alphaOutput, "DENIED_ALPHA_INPUT") {
		t.Fatalf("read-only input leaked into shared terminal output: %q", alphaOutput)
	}
	if betaOutput := beta.awaitOutput(t, "EXECUTED_BETA_ONCE"); strings.Contains(betaOutput, "DENIED_ALPHA_INPUT") {
		t.Fatalf("read-only input leaked into controller output: %q", betaOutput)
	}

	stored, err := st.ListEventsMetadata(projectID, agentID, 100)
	if err != nil {
		t.Fatalf("read durable lease events: %v", err)
	}
	expectedActors := map[string]bool{
		terminalAuditActorID(alphaSession.ID, agentID): false,
		terminalAuditActorID(betaSession.ID, agentID):  false,
	}
	grantedClients := map[string]map[string]bool{}
	revokedClients := map[string]string{}
	for _, event := range stored {
		if event.Kind != string(events.EventTerminalLeaseGranted) && event.Kind != string(events.EventTerminalLeaseRevoked) {
			continue
		}
		if strings.Contains(event.Summary, alphaSession.ID) || strings.Contains(event.Summary, betaSession.ID) {
			t.Fatalf("durable lease event exposed bearer session id: %q", event.Summary)
		}
		clientID := summaryIdentity(event.Summary, "client=")
		actorID := summaryIdentity(event.Summary, "actor=")
		if clientID == "" || actorID == "" {
			t.Fatalf("durable lease event omitted client identity: %+v", event)
		}
		if _, known := expectedActors[actorID]; !known {
			t.Fatalf("durable event actor does not match either authenticated session: %+v", event)
		}
		switch event.Kind {
		case string(events.EventTerminalLeaseGranted):
			if grantedClients[actorID] == nil {
				grantedClients[actorID] = make(map[string]bool)
			}
			grantedClients[actorID][clientID] = true
			expectedActors[actorID] = true
		case string(events.EventTerminalLeaseRevoked):
			revokedClients[actorID] = clientID
		}
	}
	for actorID, seen := range expectedActors {
		if !seen || len(grantedClients[actorID]) != 1 {
			t.Fatalf("persisted lease records do not contain one granted connection for actor %s: grants=%v", actorID, grantedClients)
		}
	}
	if revokedClients[terminalAuditActorID(alphaSession.ID, agentID)] == "" ||
		revokedClients[terminalAuditActorID(alphaSession.ID, agentID)] != onlyClientID(grantedClients[terminalAuditActorID(alphaSession.ID, agentID)]) {
		t.Fatalf("desktop's initial lease was not durably tied to its revoked connection: grants=%v revocations=%v", grantedClients, revokedClients)
	}
	if betaClientID := onlyClientID(grantedClients[terminalAuditActorID(betaSession.ID, agentID)]); betaClientID == "" || betaClientID == revokedClients[terminalAuditActorID(alphaSession.ID, agentID)] {
		t.Fatalf("mobile takeover did not record a distinct connection: grants=%v revocations=%v", grantedClients, revokedClients)
	}

	// Revoking a session must tear down its already-upgraded terminal socket,
	// revoke the writer lease, and transfer control to the remaining client.
	srv.auth.RevokeSession(betaSession.ID)
	alpha.awaitLease(t, "CONTROL")
	beta.awaitClosed(t)
	_ = beta.conn.WriteJSON(TerminalMessage{Type: "input", Data: "REVOKED_BETA_INPUT\n"})
	if err := alpha.conn.WriteJSON(TerminalMessage{Type: "input", Data: "ALPHA_AFTER_REVOKE\n"}); err != nil {
		t.Fatalf("remaining authenticated client could not use transferred lease: %v", err)
	}
	alpha.awaitOutput(t, "ALPHA_AFTER_REVOKE")
	// The child appends to the marker file after the host echoes the output,
	// so the echoed frame can win the race against the file write.
	awaitMarkerContains(t, markerFile, "ALPHA_AFTER_REVOKE")
	afterRevoke, err := os.ReadFile(markerFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(afterRevoke), "ALPHA_AFTER_REVOKE") || strings.Contains(string(afterRevoke), "REVOKED_BETA_INPUT") {
		t.Fatalf("process input after revocation was not fenced: %q", afterRevoke)
	}
}

func terminalAuditActorID(sessionID, agentID string) string {
	digest := sha256.Sum256([]byte(agentID + "\x00" + sessionID))
	return "session-" + hex.EncodeToString(digest[:12])
}

func summaryIdentity(summary, key string) string {
	if _, remainder, ok := strings.Cut(summary, key); ok {
		fields := strings.Fields(remainder)
		if len(fields) > 0 {
			return fields[0]
		}
	}
	return ""
}

func onlyClientID(clients map[string]bool) string {
	if len(clients) != 1 {
		return ""
	}
	for clientID := range clients {
		return clientID
	}
	return ""
}

func TestTerminalHandoffAuthenticatedChild(t *testing.T) {}
