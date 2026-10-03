package host

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/events"
	"github.com/IAPro-Community/nexus/internal/control/protocol"
	"github.com/IAPro-Community/nexus/internal/control/registry"
	nexusruntime "github.com/IAPro-Community/nexus/internal/runtime"
)

func waitForHostEndpoint(t *testing.T, runtimeID string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := protocol.WaitForEndpoint(ctx, runtimeID, time.Second); err != nil {
		t.Fatalf("SessionHost endpoint did not become ready: %v", err)
	}
}

func waitForTerminalReady(t *testing.T, client *protocol.Client, initial []byte) {
	t.Helper()
	if strings.Contains(string(initial), "NEXUS_TEST_READY") {
		return
	}
	timeout := 20 * time.Second
	if deadline, ok := t.Deadline(); ok {
		remaining := time.Until(deadline)
		if remaining < timeout && remaining > 2*time.Second {
			timeout = remaining - time.Second
		}
	}
	if err := client.RawConn().SetReadDeadline(time.Now().Add(timeout)); err != nil {
		t.Fatalf("set terminal readiness deadline: %v", err)
	}
	var output strings.Builder
	for !strings.Contains(output.String(), "NEXUS_TEST_READY") {
		chunk, err := client.Reader().ReadString('\n')
		if err != nil {
			t.Fatalf("terminal readiness marker not observed: %v; output=%q", err, output.String())
		}
		output.WriteString(chunk)
	}
	if err := client.ClearDeadline(); err != nil {
		t.Fatalf("clear terminal readiness deadline: %v", err)
	}
}

func TestMain(m *testing.M) {
	for _, arg := range os.Args[1:] {
		if arg == "-nexus-echo-helper" {
			_, _ = os.Stdout.WriteString("NEXUS_TEST_READY\n")
			_, _ = io.Copy(os.Stdout, os.Stdin)
			os.Exit(0)
		}
	}
	testDir, err := os.MkdirTemp("", "ai-control-host-test-*")
	if err == nil {
		defer os.RemoveAll(testDir)
		_ = os.Setenv("AI_MANAGER_DATA_DIR", testDir)
		_ = os.Setenv("AI_CLI_DATA_DIR", testDir)
	}
	registry.ResetDefaultRegistryForTest()
	code := m.Run()
	registry.ResetDefaultRegistryForTest()
	os.Exit(code)
}

func TestSlashRouterInterceptionAndEscape(t *testing.T) {
	sess := registry.RuntimeSession{
		RuntimeID:         "rt-test-01",
		ProviderID:        "codex",
		ProfileID:         "work",
		ProviderSessionID: "sess-abc-123",
		Workspace:         "/workspace/project",
		State:             registry.StateRunning,
		ControlLevel:      registry.ControlLevelAPI,
	}

	// 1. Normal prompt (should forward untouched)
	res1 := RouteSlashCommand("Refactor this file\n", sess)
	if res1.Intercepted || res1.ForwardToProcess != "Refactor this file\n" {
		t.Errorf("expected normal input to be forwarded, got %+v", res1)
	}

	// 2. /nexus status (should be intercepted)
	res2 := RouteSlashCommand("/nexus status\n", sess)
	if !res2.Intercepted || res2.ForwardToProcess != "" || !strings.Contains(res2.Response, "rt-test-01") {
		t.Errorf("expected /nexus status to be intercepted, got %+v", res2)
	}

	// 3. /ai help (should be intercepted)
	res3 := RouteSlashCommand("/nexus help\n", sess)
	if !res3.Intercepted || !strings.Contains(res3.Response, "UNIVERSAL SLASH COMMANDS") {
		t.Errorf("expected /ai help to show guide, got %+v", res3)
	}

	// 4. //nexus escaped command (should unescape and forward to provider)
	res4 := RouteSlashCommand("//nexus tell me a story\n", sess)
	if res4.Intercepted || res4.ForwardToProcess != "/nexus tell me a story\n" {
		t.Errorf("expected //nexus to be unescaped to /nexus, got %+v", res4)
	}

	// 5. /nexus detach
	res5 := RouteSlashCommand("/nexus detach", sess)
	if !res5.Intercepted || res5.Action != "detach" {
		t.Errorf("expected /nexus detach to produce action detach, got %+v", res5)
	}

	// 6. /nexus stop
	res6 := RouteSlashCommand("/nexus stop", sess)
	if !res6.Intercepted || res6.Action != "stop" {
		t.Errorf("expected /nexus stop to produce action stop, got %+v", res6)
	}
}

func TestRingBufferBounding(t *testing.T) {
	rb := NewRingBuffer(10)
	_, _ = rb.Write([]byte("12345"))
	if string(rb.Bytes()) != "12345" {
		t.Errorf("expected '12345', got %q", string(rb.Bytes()))
	}

	// Overflow buffer
	_, _ = rb.Write([]byte("67890ABCDE"))
	if string(rb.Bytes()) != "67890ABCDE" {
		t.Errorf("expected '67890ABCDE', got %q", string(rb.Bytes()))
	}
}

func TestSessionHostLifecycle(t *testing.T) {
	runtimeID := "rt-host-test"
	sess := registry.RuntimeSession{
		RuntimeID:    runtimeID,
		ProviderID:   "test",
		ProfileID:    "default",
		Workspace:    os.TempDir(),
		State:        registry.StateStarting,
		ControlLevel: registry.ControlLevelTerminal,
	}

	sh, err := NewSessionHost(Config{
		Session: sess,
		Binary:  testTerminalBinary,
		Args:    testTerminalArgs(),
		Env:     os.Environ(),
		Cwd:     os.TempDir(),
	})
	if err != nil {
		t.Fatalf("failed to create SessionHost: %v", err)
	}

	if err := sh.Start(); err != nil {
		t.Fatalf("failed to start SessionHost: %v", err)
	}
	defer sh.Stop()
	started := events.DefaultBus().GetHistory(runtimeID, 10)
	if !containsEventType(started, events.EventProcessStarted) || !containsEventType(started, events.EventRuntimeStarted) {
		t.Fatalf("runtime start lifecycle events missing: %+v", started)
	}

	waitForHostEndpoint(t, runtimeID)

	client, err := protocol.NewClient(runtimeID)
	if err != nil {
		t.Fatalf("failed to connect to SessionHost: %v", err)
	}
	defer client.Close()

	if err := client.Ping(); err != nil {
		t.Errorf("ping failed: %v", err)
	}

	st, err := client.Status()
	if err != nil {
		t.Fatalf("status query failed: %v", err)
	}
	if st.RuntimeID != runtimeID || st.State != "RUNNING" {
		t.Errorf("unexpected status: %+v", st)
	}

	// Send Stop command
	if err := client.Stop(); err != nil {
		t.Errorf("stop request failed: %v", err)
	}
	sh.Wait()
	stopped := events.DefaultBus().GetHistory(runtimeID, 10)
	if !containsEventType(stopped, events.EventProcessExited) ||
		(!containsEventType(stopped, events.EventRuntimeStopped) && !containsEventType(stopped, events.EventRuntimeFailed)) {
		t.Fatalf("runtime stop lifecycle events missing: %+v", stopped)
	}
}

func TestSessionHostTerminateWaitsForProcessFinalization(t *testing.T) {
	runtimeID := "rt-host-terminate-finalize"
	sess := registry.RuntimeSession{
		RuntimeID:    runtimeID,
		ProviderID:   "test",
		ProfileID:    "default",
		Workspace:    os.TempDir(),
		State:        registry.StateStarting,
		ControlLevel: registry.ControlLevelTerminal,
	}
	sh, err := NewSessionHost(Config{
		Session: sess,
		Binary:  testTerminalBinary,
		Args:    testTerminalArgs(),
		Env:     os.Environ(),
		Cwd:     os.TempDir(),
	})
	if err != nil {
		t.Fatalf("failed to create SessionHost: %v", err)
	}
	if err := sh.Start(); err != nil {
		t.Fatalf("failed to start SessionHost: %v", err)
	}
	pid := sh.termBackend.PID()

	if err := sh.Terminate(); err != nil {
		t.Fatalf("Terminate returned before completing process cleanup: %v", err)
	}
	select {
	case <-sh.doneChan:
	default:
		t.Fatal("Terminate returned before host finalization completed")
	}
	if sh.session.State != registry.StateStopped {
		t.Fatalf("explicit termination state = %s, want %s", sh.session.State, registry.StateStopped)
	}
	if pid > 0 && registry.IsProcessAlive(pid) {
		t.Fatalf("terminated provider process %d is still alive", pid)
	}
}

func TestSessionHostUsesConfiguredRegistryOwnership(t *testing.T) {
	runtimeID := "rt-host-owned-registry"
	owned := registry.NewRegistry(filepath.Join(t.TempDir(), "runtimes.json"))
	sess := registry.RuntimeSession{
		RuntimeID:    runtimeID,
		ProviderID:   "test",
		ProfileID:    "default",
		Workspace:    os.TempDir(),
		State:        registry.StateStarting,
		ControlLevel: registry.ControlLevelTerminal,
	}

	sh, err := NewSessionHost(Config{
		Session:  sess,
		Registry: owned,
		Binary:   testTerminalBinary,
		Args:     testTerminalArgs(),
		Env:      os.Environ(),
		Cwd:      os.TempDir(),
	})
	if err != nil {
		t.Fatalf("failed to create SessionHost: %v", err)
	}
	if err := sh.Start(); err != nil {
		t.Fatalf("failed to start SessionHost: %v", err)
	}

	stored, ok := owned.Get(runtimeID)
	if !ok || stored.State != registry.StateRunning {
		t.Fatalf("configured registry did not receive running runtime: ok=%v session=%+v", ok, stored)
	}
	if _, ok := registry.DefaultRegistry().Get(runtimeID); ok {
		t.Fatal("runtime leaked into the process-global registry")
	}

	if err := sh.Stop(); err != nil {
		t.Fatalf("stop failed: %v", err)
	}
	sh.Wait()
	stored, ok = owned.Get(runtimeID)
	if !ok || (stored.State != registry.StateStopped && stored.State != registry.StateFailed) {
		t.Fatalf("configured registry did not receive terminal runtime: ok=%v session=%+v", ok, stored)
	}
}

func containsEventType(history []events.Event, want events.EventType) bool {
	for _, event := range history {
		if event.Type == want {
			return true
		}
	}
	return false
}

func TestSessionHost_CmdInputNoDeadlock(t *testing.T) {
	runtimeID := "rt-deadlock-test"
	sess := registry.RuntimeSession{
		RuntimeID:    runtimeID,
		ProviderID:   "test",
		ProfileID:    "default",
		Workspace:    os.TempDir(),
		State:        registry.StateStarting,
		ControlLevel: registry.ControlLevelTerminal,
	}

	sh, err := NewSessionHost(Config{
		Session: sess,
		Binary:  testTerminalBinary,
		Args:    testTerminalArgs(),
		Env:     os.Environ(),
		Cwd:     os.TempDir(),
	})
	if err != nil {
		t.Fatalf("failed to create SessionHost: %v", err)
	}

	if err := sh.Start(); err != nil {
		t.Fatalf("failed to start SessionHost: %v", err)
	}
	defer sh.Stop()

	waitForHostEndpoint(t, runtimeID)

	client, err := protocol.NewClient(runtimeID)
	if err != nil {
		t.Fatalf("failed to connect to SessionHost: %v", err)
	}
	defer client.Close()

	// Send CmdInput via RPC with a strict timeout
	done := make(chan error, 1)
	go func() {
		_, err := client.Send(protocol.CmdInput, protocol.InputPayload{Data: "hello\n"})
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("CmdInput failed: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("DEADLOCK DETECTED: CmdInput hung for >2s due to reentrant mutex lock")
	}
}

func TestSessionHostFinalInputBoundaryRecoalescesSplitCSI(t *testing.T) {
	var out bytes.Buffer
	sh := &SessionHost{
		terminalInput: nexusruntime.NewTerminalInputScrubber(&out),
	}

	if _, err := sh.writeTerminalInput([]byte{0x1b}); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); got != "" {
		t.Fatalf("partial ESC leaked before completion: %q", got)
	}
	if _, err := sh.writeTerminalInput([]byte{'['}); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); got != "" {
		t.Fatalf("partial CSI leaked before completion: %q", got)
	}
	if _, err := sh.writeTerminalInput([]byte{'A'}); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); got != "\x1b[A" {
		t.Fatalf("final PTY boundary failed to recoalesce arrow: %q", got)
	}
}

func TestSessionHostFinalInputBoundaryDropsSplitTerminalMetadata(t *testing.T) {
	var out bytes.Buffer
	sh := &SessionHost{
		terminalInput: nexusruntime.NewTerminalInputScrubber(&out),
	}
	for _, part := range [][]byte{{0x1b}, {'['}, {'<', '3', '5', ';'}, {'1', '0', ';', '2', '0', 'M'}} {
		if _, err := sh.writeTerminalInput(part); err != nil {
			t.Fatal(err)
		}
	}
	if got := out.String(); got != "" {
		t.Fatalf("split SGR mouse metadata reached provider PTY: %q", got)
	}
}

func TestSessionHost_CmdInputForwardsTerminalBytesVerbatim(t *testing.T) {
	sh := &SessionHost{
		clients:      make(map[net.Conn]bool),
		activeWriter: nil,
	}
	writer, peer := net.Pipe()
	defer writer.Close()
	defer peer.Close()

	cases := []string{
		"\x1b[3~", "\x1b[C", "\x1b[A", "\x1b[H", "\x1b[F",
		"\x1b[5~", "\x1b[6~", "\x1b", "\x03", "\x1b[1;3u", "Olá 世界",
		"/e[31m literal text", `\e[31m literal text`, "\x1b]0;title\x07",
	}
	for _, input := range cases {
		got := sh.collectAttachedInputLocked(writer, []byte(input))
		if !bytes.Equal(got, []byte(input)) {
			t.Fatalf("input %q was transformed to %q", input, got)
		}
	}
}

func TestSessionHost_CmdInputNeverRoutesSlashCommands(t *testing.T) {
	sh := &SessionHost{clients: make(map[net.Conn]bool)}
	writer, peer := net.Pipe()
	defer writer.Close()
	defer peer.Close()

	input := []byte("/nexus stop\r")
	got := sh.collectAttachedInputLocked(writer, input)
	if !bytes.Equal(got, input) {
		t.Fatalf("slash command input was intercepted: got %q", got)
	}
}

func TestSessionHost_SlowObserverDoesNotBlockWriter(t *testing.T) {
	runtimeID := "rt-fanout-test"
	sess := registry.RuntimeSession{
		RuntimeID:    runtimeID,
		ProviderID:   "test",
		ProfileID:    "default",
		Workspace:    os.TempDir(),
		State:        registry.StateStarting,
		ControlLevel: registry.ControlLevelTerminal,
	}

	sh, err := NewSessionHost(Config{
		Session: sess,
		Binary:  testTerminalBinary,
		Args:    testTerminalArgs(),
		Env:     os.Environ(),
		Cwd:     os.TempDir(),
	})
	if err != nil {
		t.Fatalf("failed to create SessionHost: %v", err)
	}

	if err := sh.Start(); err != nil {
		t.Fatalf("failed to start SessionHost: %v", err)
	}
	defer sh.Stop()

	waitForHostEndpoint(t, runtimeID)

	// Writer client
	writer, err := protocol.NewClient(runtimeID)
	if err != nil {
		t.Fatalf("failed to connect writer: %v", err)
	}
	defer writer.Close()
	_, _ = writer.Send(protocol.CmdAttach, nil)

	// Observer client that never reads
	observer, err := protocol.NewClient(runtimeID)
	if err != nil {
		t.Fatalf("failed to connect observer: %v", err)
	}
	defer observer.Close()
	_, _ = observer.Send(protocol.CmdAttach, nil)

	// Send large amount of input through writer
	writeDone := make(chan error, 1)
	go func() {
		for i := 0; i < 50; i++ {
			_, err := writer.RawConn().Write([]byte("echo test line\n"))
			if err != nil {
				writeDone <- err
				return
			}
			time.Sleep(2 * time.Millisecond)
		}
		writeDone <- nil
	}()

	select {
	case err := <-writeDone:
		if err != nil {
			t.Fatalf("writer failed while observer is slow: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("BLOCKED: writer was blocked by slow observer")
	}
}

func TestSessionHost_ListenerFailureTerminatesChild(t *testing.T) {
	runtimeID, cleanup := prepareListenerFailureFixture(t)
	defer cleanup()

	sess := registry.RuntimeSession{
		RuntimeID:    runtimeID,
		ProviderID:   "test",
		ProfileID:    "default",
		Workspace:    os.TempDir(),
		State:        registry.StateStarting,
		ControlLevel: registry.ControlLevelTerminal,
	}

	sh, err := NewSessionHost(Config{
		Session: sess,
		Binary:  testTerminalBinary,
		Args:    testTerminalArgs(),
		Env:     os.Environ(),
		Cwd:     os.TempDir(),
	})
	if err != nil {
		t.Fatalf("failed to create SessionHost: %v", err)
	}

	startErr := sh.Start()
	if startErr == nil {
		_ = sh.Stop()
		t.Fatal("expected Start() to fail due to blocked listener path")
	}

	// Verify child was not orphaned
	pid := sh.session.PID
	if pid > 0 {
		time.Sleep(50 * time.Millisecond)
		if registry.IsProcessAlive(pid) {
			t.Errorf("child process %d is still alive after listener failure", pid)
		}
	}

	// Verify session was marked StateFailed
	if sh.session.State != registry.StateFailed {
		t.Errorf("expected session state to be FAILED, got %s", sh.session.State)
	}
}

func TestSessionHost_ExplicitLeaseAcquireRelease(t *testing.T) {
	runtimeID := "rt-lease-explicit"
	sess := registry.RuntimeSession{
		RuntimeID:    runtimeID,
		ProviderID:   "test",
		ProfileID:    "default",
		Workspace:    os.TempDir(),
		State:        registry.StateStarting,
		ControlLevel: registry.ControlLevelTerminal,
	}

	sh, err := NewSessionHost(Config{
		Session: sess,
		Binary:  testTerminalBinary,
		Args:    testTerminalArgs(),
		Env:     os.Environ(),
		Cwd:     os.TempDir(),
	})
	if err != nil {
		t.Fatalf("failed to create SessionHost: %v", err)
	}

	if err := sh.Start(); err != nil {
		t.Fatalf("failed to start SessionHost: %v", err)
	}
	defer sh.Stop()

	waitForHostEndpoint(t, runtimeID)

	clientA, err := protocol.NewClient(runtimeID)
	if err != nil {
		t.Fatalf("clientA connect failed: %v", err)
	}
	defer clientA.Close()
	if _, err := clientA.Send(protocol.CmdAttach, nil); err != nil {
		t.Fatalf("clientA attach failed: %v", err)
	}

	clientB, err := protocol.NewClient(runtimeID)
	if err != nil {
		t.Fatalf("clientB connect failed: %v", err)
	}
	defer clientB.Close()
	if _, err := clientB.Send(protocol.CmdAttach, nil); err != nil {
		t.Fatalf("clientB attach failed: %v", err)
	}

	// Client B explicitly acquires lease
	resp, err := clientB.Send(protocol.CmdLeaseAcquire, nil)
	if err != nil {
		t.Fatalf("clientB lease_acquire failed: %v", err)
	}
	if !resp.OK {
		t.Fatalf("clientB lease_acquire not OK: %s", resp.Error)
	}

	// Client B releases lease
	resp, err = clientB.Send(protocol.CmdLeaseRelease, nil)
	if err != nil {
		t.Fatalf("clientB lease_release failed: %v", err)
	}
	if !resp.OK {
		t.Fatalf("clientB lease_release not OK: %s", resp.Error)
	}
}

func TestSessionHost_RejectsIncompatibleProtocolVersion(t *testing.T) {
	runtimeID := "rt-version-test"
	sess := registry.RuntimeSession{
		RuntimeID:    runtimeID,
		ProviderID:   "test",
		ProfileID:    "default",
		Workspace:    os.TempDir(),
		State:        registry.StateStarting,
		ControlLevel: registry.ControlLevelTerminal,
	}

	sh, err := NewSessionHost(Config{
		Session: sess,
		Binary:  testTerminalBinary,
		Args:    testTerminalArgs(),
		Env:     os.Environ(),
		Cwd:     os.TempDir(),
	})
	if err != nil {
		t.Fatalf("failed to create SessionHost: %v", err)
	}
	if err := sh.Start(); err != nil {
		t.Fatalf("failed to start SessionHost: %v", err)
	}
	defer sh.Stop()

	waitForHostEndpoint(t, runtimeID)

	client, err := protocol.NewClient(runtimeID)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	// Craft a request with an incompatible protocol version.
	badReq := protocol.Request{
		Version:   99999,
		ID:        "req-bad-version",
		Command:   protocol.CmdPing,
		Timestamp: time.Now(),
	}
	raw, _ := json.Marshal(badReq)
	if _, err := client.RawConn().Write(append(raw, '\n')); err != nil {
		t.Fatalf("failed to write incompatible request: %v", err)
	}

	line, err := client.Reader().ReadBytes('\n')
	if err != nil {
		t.Fatalf("failed to read rejection response: %v", err)
	}
	var resp protocol.Response
	if err := json.Unmarshal(line, &resp); err != nil {
		t.Fatalf("failed to parse rejection response: %v", err)
	}
	if resp.Error != "ERROR_PROTOCOL_VERSION" {
		t.Fatalf("expected ERROR_PROTOCOL_VERSION, got %q", resp.Error)
	}
}

func TestSessionHost_SubmitPromptBypassesSlashRouterWithoutStealingWriterLease(t *testing.T) {
	runtimeID := "rt-submit-prompt-test"
	sess := registry.RuntimeSession{RuntimeID: runtimeID, ProviderID: "test", ProfileID: "default", Workspace: os.TempDir(), State: registry.StateStarting, ControlLevel: registry.ControlLevelTerminal}
	sh, err := NewSessionHost(Config{Session: sess, Binary: interactiveTestTerminalBinary, Args: interactiveTestTerminalArgs(), Env: os.Environ(), Cwd: os.TempDir()})
	if err != nil {
		t.Fatalf("failed to create SessionHost: %v", err)
	}
	if err := sh.Start(); err != nil {
		t.Fatalf("failed to start SessionHost: %v", err)
	}
	defer sh.Stop()
	waitForHostEndpoint(t, runtimeID)

	writer, err := protocol.NewClient(runtimeID)
	if err != nil {
		t.Fatalf("writer connect: %v", err)
	}
	defer writer.Close()
	attachResponse, err := writer.Send(protocol.CmdAttach, nil)
	if err != nil {
		t.Fatalf("attach writer: %v", err)
	}
	waitForTerminalReady(t, writer, attachResponse.Data)

	submitter, err := protocol.NewClient(runtimeID)
	if err != nil {
		t.Fatalf("submitter connect: %v", err)
	}
	defer submitter.Close()
	if err := submitter.SubmitPrompt("/ai status should reach provider literally"); err != nil {
		t.Fatalf("SubmitPrompt: %v", err)
	}

	_ = writer.RawConn().SetReadDeadline(time.Now().Add(2 * time.Second))
	buf := make([]byte, 2048)
	var echoed strings.Builder
	for {
		n, readErr := writer.RawConn().Read(buf)
		if n > 0 {
			echoed.Write(buf[:n])
			if strings.Contains(echoed.String(), "/ai status should reach provider literally") {
				return
			}
		}
		if readErr != nil {
			t.Fatalf("prompt was intercepted or lost: %v; got %q", readErr, echoed.String())
		}
	}
}

func TestSessionHost_AttachedLeaseAcquireDoesNotLeakToPTY(t *testing.T) {
	runtimeID := "rt-lease-no-pty-leak"
	sess := registry.RuntimeSession{
		RuntimeID:    runtimeID,
		ProviderID:   "test",
		ProfileID:    "default",
		Workspace:    os.TempDir(),
		State:        registry.StateStarting,
		ControlLevel: registry.ControlLevelTerminal,
	}
	sh, err := NewSessionHost(Config{Session: sess, Binary: interactiveTestTerminalBinary, Args: interactiveTestTerminalArgs(), Env: os.Environ(), Cwd: os.TempDir()})
	if err != nil {
		t.Fatalf("create SessionHost: %v", err)
	}
	if err := sh.Start(); err != nil {
		t.Fatalf("start SessionHost: %v", err)
	}
	defer sh.Stop()
	waitForHostEndpoint(t, runtimeID)

	client, err := protocol.NewClient(runtimeID)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer client.Close()
	attachResponse, err := client.Send(protocol.CmdAttach, nil)
	if err != nil {
		t.Fatalf("attach: %v", err)
	}
	waitForTerminalReady(t, client, attachResponse.Data)

	req, err := protocol.NewRequest(protocol.CmdLeaseAcquire, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	payload, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if _, err := client.RawConn().Write(append(payload, '\n')); err != nil {
		t.Fatalf("write lease_acquire on attached conn: %v", err)
	}

	// Read the RPC response; it must be a protocol frame, not PTY echo.
	_ = client.RawConn().SetReadDeadline(time.Now().Add(2 * time.Second))
	buf := make([]byte, 4096)
	n, err := client.RawConn().Read(buf)
	if err != nil {
		t.Fatalf("read after lease_acquire: %v", err)
	}
	got := string(buf[:n])
	if strings.Contains(got, "lease_acquire") && !strings.Contains(got, `"ok"`) {
		t.Fatalf("lease_acquire request leaked toward PTY/fanout: %q", got)
	}
	var resp protocol.Response
	if err := json.Unmarshal(bytes.TrimSpace(buf[:n]), &resp); err != nil {
		// May have multiple lines; find the JSON response.
		for _, line := range strings.Split(got, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if json.Unmarshal([]byte(line), &resp) == nil && (resp.OK || resp.Error != "") {
				break
			}
		}
		if !resp.OK && resp.Error == "" {
			t.Fatalf("expected lease response, got %q", got)
		}
	}
	if !resp.OK {
		t.Fatalf("lease_acquire not OK: %s (raw=%q)", resp.Error, got)
	}

	// A subsequent single keystroke (no newline) must still reach the PTY.
	// Interactive terminals send raw bytes without trailing \n.
	marker := "k"
	if _, err := client.RawConn().Write([]byte(marker)); err != nil {
		t.Fatalf("write single keystroke: %v", err)
	}
	deadline := time.Now().Add(2 * time.Second)
	var echoed strings.Builder
	for time.Now().Before(deadline) {
		_ = client.RawConn().SetReadDeadline(time.Now().Add(200 * time.Millisecond))
		n, err := client.RawConn().Read(buf)
		if n > 0 {
			echoed.Write(buf[:n])
			if strings.Contains(echoed.String(), marker) {
				return
			}
		}
		if err != nil && !strings.Contains(err.Error(), "timeout") && !strings.Contains(err.Error(), "deadline") {
			t.Fatalf("read marker echo: %v (so far %q)", err, echoed.String())
		}
	}
	t.Fatalf("PTY did not echo single keystroke after lease_acquire; got %q", echoed.String())
}
