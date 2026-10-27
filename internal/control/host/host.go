package host

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/events"
	"github.com/IAPro-Community/nexus/internal/control/protocol"
	"github.com/IAPro-Community/nexus/internal/control/registry"
	"github.com/IAPro-Community/nexus/internal/control/terminal"
	"github.com/IAPro-Community/nexus/internal/core/classifier"
	"github.com/IAPro-Community/nexus/internal/core/provider/adapters/codex"
	"github.com/IAPro-Community/nexus/internal/core/security"
	"github.com/IAPro-Community/nexus/internal/profile"
	nexusruntime "github.com/IAPro-Community/nexus/internal/runtime"
)

const MaxFrameSize = 64 * 1024 // 64 KB max frame

// Config configures a SessionHost instance.
type Config struct {
	Session     registry.RuntimeSession
	Registry    *registry.Registry
	Binary      string
	Args        []string
	Env         []string
	Cwd         string
	InitialRows int
	InitialCols int

	// terminalBackend is an internal lifecycle test seam. Production callers
	// leave it nil and receive the platform backend from terminal.NewBackend.
	terminalBackend terminal.Backend
}

// SessionHost manages a single supervised process runtime and its IPC listener.
type SessionHost struct {
	mu            sync.RWMutex
	session       registry.RuntimeSession
	registry      *registry.Registry
	cfg           Config
	cmd           *exec.Cmd
	termBackend   terminal.Backend
	ringBuffer    *RingBuffer
	fanout        *BoundedFanout
	listener      net.Listener
	clients       map[net.Conn]bool
	activeWriter  net.Conn
	stopChan      chan struct{}
	doneChan      chan struct{}
	streamDone    chan struct{}
	detector      *AttentionDetector
	shellInput    shellPipeInputNormalizer
	terminalInput *nexusruntime.TerminalOutputScrubber
	terminalMouse *nexusruntime.TerminalMouseState
	stopOnce      sync.Once
	doneOnce      sync.Once
	codexTUILock  interface{ Release() error }
}

// NewSessionHost creates a new SessionHost for a given runtime.
func NewSessionHost(cfg Config) (*SessionHost, error) {
	if cfg.Session.RuntimeID == "" {
		return nil, fmt.Errorf("runtime ID is required")
	}
	if cfg.Registry == nil {
		cfg.Registry = registry.DefaultRegistry()
	}

	cmd := exec.Command(cfg.Binary, cfg.Args...)
	cmd.Dir = cfg.Cwd
	// The embedded terminal is xterm.js: provider TUIs started by the user
	// (for example `codex` typed in a project shell) need the same compatibility
	// defaults as supervised provider runtimes.
	cmd.Env = nexusruntime.ApplyEmbeddedTerminalCompat(cfg.Env)

	termBackend := cfg.terminalBackend
	if termBackend == nil {
		termBackend = terminal.NewBackendForProvider(cfg.Session.ProviderID)
	}

	sh := &SessionHost{
		session:     cfg.Session,
		registry:    cfg.Registry,
		cfg:         cfg,
		cmd:         cmd,
		termBackend: termBackend,
		ringBuffer:  NewRingBuffer(128 * 1024), // 128 KB terminal history
		fanout:      NewBoundedFanout(256),
		clients:     make(map[net.Conn]bool),
		stopChan:    make(chan struct{}),
		doneChan:    make(chan struct{}),
		streamDone:  make(chan struct{}),
	}

	sh.detector = NewAttentionDetectorWithProject(
		cfg.Session.RuntimeID,
		cfg.Session.ProviderID,
		cfg.Session.ProfileID,
		cfg.Cwd,
		cfg.Session.ProjectID,
		cfg.Session.ProjectName,
		func(reason, context, dynamicTitle string, state registry.RuntimeState) {
			sh.mu.Lock()
			sh.session.State = state
			sh.session.AttentionReason = reason
			sh.session.AttentionContext = context
			sh.session.DynamicTitle = dynamicTitle
			sh.session.Title = dynamicTitle
			if sh.detector != nil {
				sh.session.PromptKind = sh.detector.lastPrompt
				sh.session.AttentionKind = sh.detector.lastKind
				sh.session.AttentionFingerprint = sh.detector.lastFingerprint
			}
			sh.mu.Unlock()
		},
	)
	sh.detector.SetRegistry(cfg.Registry)
	// PTY regex is TERMINAL fallback only. EVENTS/CONTROL_API (future structured
	// adapters) must not scrape stdout for agent attention.
	sh.detector.SetControlPolicy(cfg.Session.ControlLevel, false, cfg.Session.AgentID)

	return sh, nil
}

// Start launches the supervised process and begins listening for IPC control connections.
func (sh *SessionHost) Start() error {
	sh.mu.Lock()
	defer sh.mu.Unlock()
	setStage := func(stage registry.StartupStage, fault registry.StartupFault) {
		sh.session.StartupStage = stage
		sh.session.StageChangedAt = time.Now()
		sh.session.LastFault = fault
		_ = sh.registry.UpdateStartupStage(sh.session.RuntimeID, stage, fault)
	}
	setStage(registry.StartupHostStarting, "")
	setStage(registry.StartupIPCBinding, "")

	// Bind IPC before starting the provider so a bind failure is reported as
	// IPC_BIND_FAILED instead of being hidden behind a generic handshake timeout.
	l, err := protocol.Listen(sh.session.RuntimeID)
	if err != nil {
		setStage(registry.StartupIPCBinding, registry.StartupFaultIPCBindFailed)
		sh.completeStartFailureLocked()
		return fmt.Errorf("failed to create control endpoint: %w", err)
	}
	sh.listener = l
	setStage(registry.StartupIPCBound, "")
	go sh.serveIPC(l)
	setStage(registry.StartupProtocolReady, "")

	// 1. Start process with terminal backend
	rows := sh.cfg.InitialRows
	cols := sh.cfg.InitialCols
	if rows <= 0 {
		rows = 24
	}
	if cols <= 0 {
		cols = 80
	}

	prepareCmd(sh.cmd)
	setStage(registry.StartupTerminalStarting, "")
	setStage(registry.StartupProviderStarting, "")

	if strings.EqualFold(sh.session.ProviderID, "codex") {
		home := codex.EnvCODEXHome(sh.cfg.Env)
		if home == "" {
			setStage(registry.StartupProviderStarting, registry.StartupFaultProcessSupervision)
			sh.closeListenerLocked("startup-codex-home-missing")
			sh.completeStartFailureLocked()
			return fmt.Errorf("CODEX_HOME is required to start a supervised Codex session")
		}
		// Keep sh.mu held so Stop/Terminate cannot race listener teardown while
		// this runtime acquires its shared profile lease.
		lock, ok, err := codex.TryAcquireTUILease(home)
		if err != nil {
			setStage(registry.StartupProviderStarting, registry.StartupFaultProcessSupervision)
			sh.closeListenerLocked("startup-codex-tui-lease-error")
			sh.completeStartFailureLocked()
			return fmt.Errorf("failed to acquire Codex TUI lease: %w", err)
		}
		if !ok {
			setStage(registry.StartupProviderStarting, registry.StartupFaultProcessSupervision)
			sh.closeListenerLocked("startup-codex-tui-lease-busy")
			sh.completeStartFailureLocked()
			return fmt.Errorf("failed to acquire Codex TUI lease: %w", codex.ErrTUIBusy)
		}
		sh.codexTUILock = lock
	}

	if err := sh.termBackend.Start(sh.cmd, rows, cols); err != nil {
		setStage(registry.StartupProviderStarting, registry.StartupFaultConPTYStartFailed)
		sh.releaseCodexTUILockLocked()
		sh.completeStartFailureLocked()
		return fmt.Errorf("failed to start terminal backend: %w", err)
	}
	if err := sh.termBackend.Supervise(); err != nil {
		setStage(registry.StartupProviderStarting, registry.StartupFaultProcessSupervision)
		_ = sh.termBackend.Kill()
		_ = sh.termBackend.Wait()
		sh.releaseCodexTUILockLocked()
		sh.completeStartFailureLocked()
		return fmt.Errorf("failed to supervise provider process: %w", err)
	}
	// SessionHost is the final input authority. Raw attach streams are byte
	// streams, not message transports, so upstream write boundaries cannot be
	// trusted. Coalesce CSI/SS3 and terminal metadata one last time immediately
	// before the provider PTY for both CLI and Web paths.
	sh.terminalMouse = &nexusruntime.TerminalMouseState{}
	sh.terminalInput = nexusruntime.NewTerminalInputScrubberWithMouseState(
		nexusruntime.NewTerminalTracingWriter(sh.termBackend, sh.session.RuntimeID, "host.scrubber.output"),
		sh.terminalMouse,
	)

	setStage(registry.StartupTerminalReady, "")

	// Drain terminal output independently and let lifecycle finalization wait for
	// the reader to observe EOF. This prevents Windows cleanup from racing live
	// ConPTY/pipe handles after the provider process has already exited.
	go func() {
		defer close(sh.streamDone)
		sh.streamReader(sh.termBackend)
	}()

	// Update session details
	sh.session.PID = sh.termBackend.PID()
	generation := time.Now().UnixNano()
	sh.session.ProviderGeneration = generation
	sh.session.HostPID = os.Getpid()
	sh.session.HostGeneration = generation
	if sh.session.HostBinaryPath == "" {
		sh.session.HostBinaryPath, _ = os.Executable()
	}
	if sh.session.HostBinaryFingerprint == "" {
		sh.session.HostBinaryFingerprint, _ = registry.BinaryFingerprint(sh.session.HostBinaryPath)
	}
	sh.session.State = registry.StateRunning
	setStage(registry.StartupRunning, "")
	sh.session.ControlEndpoint = protocol.EndpointPath(sh.session.RuntimeID)

	// Persist in Registry
	_ = sh.registry.Register(sh.session)

	// Emit Process Started event
	eventData := sh.lifecycleEventData(map[string]any{
		"pid": sh.session.PID, "host_pid": sh.session.HostPID, "endpoint": sh.session.ControlEndpoint,
	})
	events.DefaultBus().Publish(events.NewEventWithCorrelation(
		sh.session.LineageID,
		sh.session.RuntimeID,
		sh.session.ProviderID,
		sh.session.ProfileID,
		events.EventProcessStarted,
		fmt.Sprintf("Started supervised %s runtime (PID %d, Host PID %d)", sh.session.ProviderID, sh.session.PID, sh.session.HostPID),
		eventData,
	))
	events.DefaultBus().Publish(events.NewEventWithCorrelation(
		sh.session.LineageID,
		sh.session.RuntimeID,
		sh.session.ProviderID,
		sh.session.ProfileID,
		events.EventRuntimeStarted,
		fmt.Sprintf("Runtime started (PID %d)", sh.session.PID),
		eventData,
	))

	// Monitor process termination.
	go sh.waitProcess()

	return nil
}

// completeStartFailureLocked leaves a failed startup in the same terminal
// lifecycle shape as an exited child: resources are closed, Wait unblocks and
// later Stop/Terminate calls remain idempotent. The caller holds sh.mu.
func (sh *SessionHost) completeStartFailureLocked() {
	sh.session.State = registry.StateFailed
	_ = sh.registry.UpdateState(sh.session.RuntimeID, registry.StateFailed)
	sh.stopOnce.Do(func() { close(sh.stopChan) })
	sh.closeListenerLocked("startup-failure")
	sh.fanout.Close()
	_ = sh.termBackend.Close()
	sh.doneOnce.Do(func() { close(sh.doneChan) })
}

func (sh *SessionHost) streamReader(r io.Reader) {
	buf := make([]byte, 4096)
	var pending []byte
	var filtered bytes.Buffer
	scrubber := nexusruntime.NewTerminalOutputScrubberTrackingMouse(&filtered, sh.terminalMouse)
	flushFiltered := func() {
		if filtered.Len() == 0 {
			return
		}
		pending = append(pending, filtered.Bytes()...)
		filtered.Reset()
	}
	for {
		n, err := r.Read(buf)
		if n > 0 {
			_, _ = scrubber.Write(buf[:n])
			flushFiltered()
			for {
				idx := bytes.IndexByte(pending, '\n')
				if idx < 0 {
					// Flush non-JSON partials so interactive output stays live.
					if len(pending) > 0 && !protocol.MayBeWireControlPrefix(pending) {
						chunk := pending
						pending = nil
						sh.ringBuffer.Write(chunk)
						sh.broadcast(chunk)
						if sh.detector != nil {
							sh.detector.ProcessChunk(chunk)
						}
					} else if len(pending) > MaxFrameSize {
						chunk := pending
						pending = nil
						sh.ringBuffer.Write(chunk)
						sh.broadcast(chunk)
						if sh.detector != nil {
							sh.detector.ProcessChunk(chunk)
						}
					}
					break
				}
				line := pending[:idx+1]
				pending = pending[idx+1:]
				if isHostProtocolControlFrame(line) {
					continue
				}
				sh.ringBuffer.Write(line)
				sh.broadcast(line)
				if sh.detector != nil {
					sh.detector.ProcessChunk(line)
				}
			}
		}
		if err != nil {
			_ = scrubber.Flush()
			flushFiltered()
			if len(pending) > 0 && !isHostProtocolControlFrame(pending) {
				sh.ringBuffer.Write(pending)
				sh.broadcast(pending)
				if sh.detector != nil {
					sh.detector.ProcessChunk(pending)
				}
			}
			break
		}
	}
}

// isHostProtocolControlFrame detects Nexus Control RPC lines that must never
// enter the ring buffer / fanout (e.g. lease_acquire echoed into the PTY).
func isHostProtocolControlFrame(data []byte) bool {
	return protocol.IsWireControlFrame(data)
}

func (sh *SessionHost) broadcast(data []byte) {
	sh.fanout.Broadcast(data)
}

func (sh *SessionHost) serveIPC(listener net.Listener) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-sh.stopChan:
				return
			default:
			}
			if errors.Is(err, net.ErrClosed) {
				return
			}
			// A transient Accept error must NOT retire the listener: the pipe
			// keeps existing in the Windows object namespace while the accept
			// loop is gone, so clients see FILE_NOT_FOUND/BUSY and time out
			// even though the host process is alive. Retry until the listener
			// is closed or the runtime stops.
			slog.Info(
				"nexus ipc accept retry",
				"runtime_id", sh.session.RuntimeID,
				"error", err.Error(),
			)
			select {
			case <-sh.stopChan:
				return
			case <-time.After(50 * time.Millisecond):
			}
			continue
		}
		go sh.handleClient(conn)
	}
}

// closeListenerLocked closes the IPC listener under sh.mu and records why. A
// closed listener is the only state where a live host answers dial timeouts,
// so every close path must leave an auditable reason in the host log.
func (sh *SessionHost) closeListenerLocked(reason string) {
	if sh.listener != nil {
		_ = sh.listener.Close()
		sh.listener = nil
	}
	slog.Info(
		"nexus ipc listener closed",
		"runtime_id", sh.session.RuntimeID,
		"reason", reason,
	)
}

func (sh *SessionHost) handleClient(conn net.Conn) {
	reader := bufio.NewReaderSize(conn, MaxFrameSize)

	// Read initial RPC command frames
	for {
		line, err := readBoundedLine(reader, MaxFrameSize)
		if err != nil {
			if err == errFrameTooLarge {
				sh.broadcast([]byte("\r\n[Nexus Control] Error: oversized IPC frame rejected\r\n"))
			}
			sh.removeClient(conn)
			_ = conn.Close()
			return
		}

		if len(line) > MaxFrameSize {
			_ = conn.Close()
			sh.removeClient(conn)
			return
		}

		var req protocol.Request
		if err := json.Unmarshal(line, &req); err != nil {
			// If not a JSON command, process raw input if writer
			sh.processAttachedInput(conn, line)
			continue
		}

		// Handle RPC request
		isAttach := req.Command == protocol.CmdAttach
		sh.handleRPCRequest(conn, req)

		// If this was an Attach command, switch connection to continuous raw streaming mode
		if isAttach {
			_ = conn.SetDeadline(time.Time{})
			sh.streamAttachedInput(conn, reader)
			return
		}
	}
}

// errFrameTooLarge is returned when an IPC frame exceeds MaxFrameSize.
var errFrameTooLarge = errors.New("IPC frame exceeds maximum allowed size")

// readBoundedLine reads a newline-terminated frame while never allocating more
// than limit bytes. This prevents a malicious peer from forcing unbounded
// memory growth by omitting the newline (see readBytes-safety requirement).
func readBoundedLine(r *bufio.Reader, limit int) ([]byte, error) {
	var buf []byte
	for {
		b, err := r.ReadByte()
		if err != nil {
			return buf, err
		}
		if b == '\n' {
			return buf, nil
		}
		buf = append(buf, b)
		if len(buf) > limit {
			return buf, errFrameTooLarge
		}
	}
}

func (sh *SessionHost) streamAttachedInput(conn net.Conn, reader *bufio.Reader) {
	_ = conn.SetDeadline(time.Time{})
	// Mix of newline-terminated RPC frames and raw PTY keystrokes on one
	// stream. Raw bytes must be forwarded immediately — interactive shells
	// send single keystrokes with no trailing \n (Enter is usually \r).
	var jsonBuf []byte
	inJSON := false

	flushRaw := func(data []byte) {
		if len(data) == 0 {
			return
		}
		sh.processAttachedInput(conn, data)
	}
	resetJSON := func() {
		jsonBuf = nil
		inJSON = false
	}
	looksLikeProtocol := func(buf []byte) bool {
		return bytes.HasPrefix(buf, []byte(`{"version"`)) ||
			bytes.HasPrefix(buf, []byte(`{"command"`)) ||
			bytes.HasPrefix(buf, []byte(`{"id"`))
	}

	for {
		b, err := reader.ReadByte()
		if err != nil {
			if inJSON && len(jsonBuf) > 0 {
				flushRaw(jsonBuf)
			}
			sh.removeClient(conn)
			_ = conn.Close()
			return
		}

		if !inJSON {
			if b == '{' {
				inJSON = true
				jsonBuf = append(jsonBuf[:0], b)
				continue
			}
			flushRaw([]byte{b})
			continue
		}

		jsonBuf = append(jsonBuf, b)
		if len(jsonBuf) > MaxFrameSize {
			flushRaw(jsonBuf)
			resetJSON()
			continue
		}
		// Enter (\r) mid-buffer means interactive typing, not an RPC frame.
		if b == '\r' {
			flushRaw(jsonBuf)
			resetJSON()
			continue
		}
		// Once long enough, reject buffers that cannot be protocol JSON.
		if len(jsonBuf) >= 11 && !looksLikeProtocol(jsonBuf) {
			flushRaw(jsonBuf)
			resetJSON()
			continue
		}
		if b != '\n' {
			continue
		}

		trimmed := bytes.TrimSpace(jsonBuf)
		var req protocol.Request
		if json.Unmarshal(trimmed, &req) == nil && req.Command != "" {
			sh.handleRPCRequest(conn, req)
		} else {
			flushRaw(jsonBuf)
		}
		resetJSON()
	}
}

func (sh *SessionHost) handleRPCRequest(conn net.Conn, req protocol.Request) {
	sh.mu.Lock()

	if req.Version != 0 && req.Version != protocol.ProtocolVersion {
		resp := protocol.NewErrorResponse("ERROR_PROTOCOL_VERSION")
		sh.mu.Unlock()
		data, _ := json.Marshal(resp)
		_, _ = conn.Write(append(data, '\n'))
		return
	}

	var resp protocol.Response
	var resizeRows, resizeCols int
	var promptWrite []byte
	var forwardWrite []byte
	var controlCmd string
	skipResponse := false

	switch req.Command {
	case protocol.CmdPing:
		status := protocol.StatusData{
			RuntimeID:                 sh.session.RuntimeID,
			ProviderID:                sh.session.ProviderID,
			ProfileID:                 sh.session.ProfileID,
			Workspace:                 sh.session.Workspace,
			PID:                       sh.session.PID,
			ProviderGeneration:        sh.session.ProviderGeneration,
			HostPID:                   sh.session.HostPID,
			HostGeneration:            sh.session.HostGeneration,
			HostBinaryPath:            sh.session.HostBinaryPath,
			HostBinaryFingerprint:     sh.session.HostBinaryFingerprint,
			ProviderBinaryPath:        sh.session.ProviderBinaryPath,
			ProviderBinaryFingerprint: sh.session.ProviderBinaryFingerprint,
			State:                     string(sh.session.State),
		}
		resp, _ = protocol.NewResponse(status)
		resp.RuntimeID = status.RuntimeID
		resp.State = status.State
		resp.Code = "PONG"
		resp.Action = string(protocol.CmdPing)
		resp.Message = "runtime ready"

	case protocol.CmdStatus:
		fanoutStats := sh.fanout.Stats()
		quotaView := profile.GetQuotaView(sh.session.ProviderID, sh.session.ProfileID, "", "")
		quotaPercent, _ := quotaView.Bottleneck()
		status := protocol.StatusData{
			RuntimeID:                 sh.session.RuntimeID,
			ProviderID:                sh.session.ProviderID,
			ProfileID:                 sh.session.ProfileID,
			ProviderSessionID:         sh.session.ProviderSessionID,
			Workspace:                 sh.session.Workspace,
			PID:                       sh.session.PID,
			ProviderGeneration:        sh.session.ProviderGeneration,
			HostPID:                   sh.session.HostPID,
			HostGeneration:            sh.session.HostGeneration,
			HostBinaryPath:            sh.session.HostBinaryPath,
			HostBinaryFingerprint:     sh.session.HostBinaryFingerprint,
			ProviderBinaryPath:        sh.session.ProviderBinaryPath,
			ProviderBinaryFingerprint: sh.session.ProviderBinaryFingerprint,
			State:                     string(sh.session.State),
			StartupStage:              string(sh.session.StartupStage),
			StageChangedAt:            sh.session.StageChangedAt,
			LastFault:                 security.Redact(string(sh.session.LastFault)),
			ControlLevel:              string(sh.session.ControlLevel),
			StartedAt:                 sh.session.StartedAt,
			DroppedOutputChunks:       fanoutStats.DroppedChunks,
			AttentionReason:           sh.session.AttentionReason,
			AttentionContext:          security.Redact(sh.session.AttentionContext),
			AttentionKind:             sh.session.AttentionKind,
			PromptKind:                sh.session.PromptKind,
			Continuity:                sh.session.Continuity,
			QuotaStatus:               quotaView.Status,
			QuotaPercent:              quotaPercent,
		}
		resp, _ = protocol.NewResponse(status)
		resp.RuntimeID = status.RuntimeID
		resp.State = status.State
		resp.Code = "STATUS_OK"
		resp.Action = string(protocol.CmdStatus)
		resp.Message = "runtime status"

	case protocol.CmdAttach:
		sh.clients[conn] = true
		sh.fanout.AddClient(conn)
		if sh.activeWriter == nil {
			sh.activeWriter = conn
		}
		history := sh.ringBuffer.Bytes()
		resp, _ = protocol.NewResponse(string(history))

	case protocol.CmdDetach:
		delete(sh.clients, conn)
		sh.fanout.RemoveClient(conn)
		if sh.activeWriter == conn {
			sh.activeWriter = nil
		}
		resp = protocol.NewControlResponse(protocol.ControlResult{OK: true, Code: "DETACHED", RuntimeID: sh.session.RuntimeID, Action: string(protocol.CmdDetach), State: string(sh.session.State), Message: "detached"})

	case protocol.CmdResize:
		if req.Payload != nil {
			var p protocol.ResizePayload
			if json.Unmarshal(req.Payload, &p) == nil && p.Rows > 0 && p.Cols > 0 {
				resizeRows, resizeCols = p.Rows, p.Cols
			}
		}
		resp, _ = protocol.NewResponse("resized")

	case protocol.CmdInput:
		if req.Payload != nil {
			var p protocol.InputPayload
			if json.Unmarshal(req.Payload, &p) == nil && p.Data != "" {
				nexusruntime.TraceTerminalBytes(sh.session.RuntimeID, "host.cmd_input", []byte(p.Data))
				// CmdInput remains byte-transparent at the control layer. The final
				// SessionHost terminal scrubber only preserves sequence atomicity and
				// drops terminal metadata before the provider PTY.
				forwardWrite = sh.collectAttachedInputLocked(conn, []byte(p.Data))
				forwardWrite = sh.normalizeShellPipeInputLocked(forwardWrite)
			}
		}
		resp, _ = protocol.NewResponse("input_received")
		resp.RuntimeID = sh.session.RuntimeID
		resp.Action = string(protocol.CmdInput)
		resp.Code = "INPUT_ACCEPTED"
		resp.Message = "raw input forwarded"

	case protocol.CmdSubmitPrompt:
		var p protocol.SubmitPromptPayload
		if req.Payload == nil || json.Unmarshal(req.Payload, &p) != nil || strings.TrimSpace(p.Prompt) == "" {
			resp = protocol.NewErrorResponse("prompt is required")
			break
		}
		prompt := strings.TrimRight(p.Prompt, "\r\n") + "\r"
		promptWrite = []byte(prompt)
		resp, _ = protocol.NewResponse("prompt_submitted")

	case protocol.CmdStop:
		sh.session.State = registry.StateStopping
		_ = sh.registry.UpdateState(sh.session.RuntimeID, registry.StateStopping)
		go sh.Stop()
		resp = protocol.NewControlResponse(protocol.ControlResult{OK: true, Code: "STOP_REQUESTED", RuntimeID: sh.session.RuntimeID, Action: string(protocol.CmdStop), State: string(registry.StateStopping), Message: "stop requested", CorrelationID: sh.session.LineageID})

	case protocol.CmdTerminate:
		sh.session.State = registry.StateStopping
		_ = sh.registry.UpdateState(sh.session.RuntimeID, registry.StateStopping)
		go sh.Terminate()
		resp = protocol.NewControlResponse(protocol.ControlResult{OK: true, Code: "TERMINATE_REQUESTED", RuntimeID: sh.session.RuntimeID, Action: string(protocol.CmdTerminate), State: string(registry.StateStopping), Message: "terminate requested", CorrelationID: sh.session.LineageID})

	case protocol.CmdEvents:
		limit := 20
		if req.Payload != nil {
			var p protocol.EventsPayload
			if json.Unmarshal(req.Payload, &p) == nil && p.Limit > 0 && p.Limit < 100 {
				limit = p.Limit
			}
		}
		history := events.DefaultBus().GetHistory(sh.session.RuntimeID, limit)
		result := make([]protocol.EventData, 0, len(history))
		for _, event := range history {
			result = append(result, protocol.EventData{ID: event.ID, CorrelationID: event.CorrelationID, RuntimeID: event.RuntimeID, Provider: event.Provider, Profile: event.Profile, Type: string(event.Type), Summary: security.Redact(event.Summary), Timestamp: event.Timestamp})
		}
		resp, _ = protocol.NewResponse(result)
		resp.RuntimeID = sh.session.RuntimeID
		resp.Action = string(protocol.CmdEvents)
		resp.Code = "EVENTS_OK"
		resp.Message = "redacted event history"

	case protocol.CmdUsage:
		quotaView := profile.GetQuotaView(sh.session.ProviderID, sh.session.ProfileID, "", "")
		percentLeft, _ := quotaView.Bottleneck()
		usage := protocol.UsageData{RuntimeID: sh.session.RuntimeID, ProviderID: sh.session.ProviderID, ProfileID: sh.session.ProfileID, Status: quotaView.Status}
		if !quotaView.FetchedAt.IsZero() {
			usage.PercentLeft = percentLeft
			usage.FetchedAtUnix = quotaView.FetchedAt.Unix()
		}
		resp, _ = protocol.NewResponse(usage)
		resp.RuntimeID = sh.session.RuntimeID
		resp.Action = string(protocol.CmdUsage)
		resp.Code = "USAGE_OK"
		resp.Message = "quota snapshot"

	case protocol.CmdHandoff:
		var p protocol.HandoffPayload
		if req.Payload == nil || json.Unmarshal(req.Payload, &p) != nil || strings.TrimSpace(p.TargetProfile) == "" {
			resp = protocol.NewErrorResponse("TARGET_PROFILE_REQUIRED")
			break
		}
		if PerformAccountHandoff == nil {
			resp = protocol.NewErrorResponse("HANDOFF_UNAVAILABLE")
			break
		}
		target := strings.TrimSpace(p.TargetProfile)
		sh.session.State = registry.StateHandoff
		_ = sh.registry.UpdateState(sh.session.RuntimeID, registry.StateHandoff)
		go func() {
			if _, err := PerformAccountHandoff(context.Background(), sh.session.RuntimeID, target); err != nil {
				sh.broadcast([]byte("\r\n[Nexus Control] Handoff failed: " + security.Redact(err.Error()) + "\r\n"))
			}
		}()
		resp = protocol.NewControlResponse(protocol.ControlResult{OK: true, Code: "HANDOFF_REQUESTED", RuntimeID: sh.session.RuntimeID, Action: string(protocol.CmdHandoff), State: string(registry.StateHandoff), Message: "handoff requested", CorrelationID: sh.session.LineageID})

	case protocol.CmdContinue:
		var p protocol.ContinuePayload
		if req.Payload == nil || json.Unmarshal(req.Payload, &p) != nil || strings.TrimSpace(p.TargetProvider) == "" {
			resp = protocol.NewErrorResponse("TARGET_PROVIDER_REQUIRED")
			break
		}
		if PerformContextHandoff == nil {
			resp = protocol.NewErrorResponse("CONTINUE_UNAVAILABLE")
			break
		}
		provider, profile := strings.TrimSpace(p.TargetProvider), strings.TrimSpace(p.TargetProfile)
		sh.session.State = registry.StateHandoff
		_ = sh.registry.UpdateState(sh.session.RuntimeID, registry.StateHandoff)
		go func() {
			if _, err := PerformContextHandoff(context.Background(), sh.session.RuntimeID, provider, profile); err != nil {
				sh.broadcast([]byte("\r\n[Nexus Control] Context handoff failed: " + security.Redact(err.Error()) + "\r\n"))
			}
		}()
		resp = protocol.NewControlResponse(protocol.ControlResult{OK: true, Code: "CONTINUE_REQUESTED", RuntimeID: sh.session.RuntimeID, Action: string(protocol.CmdContinue), State: string(registry.StateHandoff), Message: "context handoff requested", CorrelationID: sh.session.LineageID})

	case protocol.CmdSlash:
		var p protocol.SlashPayload
		if req.Payload == nil || json.Unmarshal(req.Payload, &p) != nil || strings.TrimSpace(p.RawCommand) == "" {
			resp = protocol.NewErrorResponse("RAW_COMMAND_REQUIRED")
			break
		}
		controlCmd = p.RawCommand
		resp = protocol.NewControlResponse(protocol.ControlResult{OK: true, Code: "COMMAND_ACCEPTED", RuntimeID: sh.session.RuntimeID, Action: string(protocol.CmdSlash), State: string(sh.session.State), Message: "explicit control command accepted"})

	case protocol.CmdLeaseAcquire:
		if _, attached := sh.clients[conn]; !attached {
			resp = protocol.NewErrorResponse("lease acquisition requires attached streaming client")
			break
		}
		sh.activeWriter = conn
		resp, _ = protocol.NewResponse("lease_acquired")

	case protocol.CmdLeaseRelease:
		if sh.activeWriter == conn {
			sh.activeWriter = nil
		}
		resp, _ = protocol.NewResponse("lease_released")

	default:
		resp = protocol.NewErrorResponse(fmt.Sprintf("unknown command %q", req.Command))
	}
	if resp.CorrelationID == "" {
		resp.CorrelationID = req.ID
	}

	_, isAttached := sh.clients[conn]
	if isAttached && req.Command == protocol.CmdResize {
		skipResponse = true
	}
	sh.mu.Unlock()

	if resizeRows > 0 && resizeCols > 0 {
		_ = sh.termBackend.Resize(resizeRows, resizeCols)
	}
	if len(forwardWrite) > 0 {
		_ = sh.writeTerminalInput(forwardWrite)
	}
	if len(promptWrite) > 0 {
		_ = sh.writeTerminalInput(promptWrite)
	}
	if controlCmd != "" {
		sh.mu.Lock()
		sh.handleControlCommandLocked(controlCmd)
		sh.mu.Unlock()
	}
	if skipResponse {
		return
	}
	data, _ := json.Marshal(resp)
	_, _ = conn.Write(append(data, '\n'))
}

func (sh *SessionHost) processAttachedInput(conn net.Conn, data []byte) {
	nexusruntime.TraceTerminalBytes(sh.session.RuntimeID, "host.raw_attach.input", data)
	sh.mu.Lock()
	forward := sh.collectAttachedInputLocked(conn, data)
	forward = sh.normalizeShellPipeInputLocked(forward)
	sh.mu.Unlock()

	if len(forward) > 0 {
		_ = sh.writeTerminalInput(forward)
	}
}

func (sh *SessionHost) writeTerminalInput(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	sh.mu.RLock()
	input := sh.terminalInput
	backend := sh.termBackend
	sh.mu.RUnlock()
	var err error
	if input != nil {
		_, err = input.Write(data)
	} else {
		nexusruntime.TraceTerminalBytes(sh.session.RuntimeID, "host.pty.write", data)
		_, err = backend.Write(data)
	}
	if err == nil && sh.detector != nil {
		// When AGY/Codex/Claude accepts an actual operator submission,
		// release a previously displayed pending question in the shared
		// registry. Arrow keys and partial input must not clear the HUD.
		sh.detector.ObserveOperatorInput(data)
	}
	return err
}

func (sh *SessionHost) normalizeShellPipeInputLocked(data []byte) []byte {
	if runtime.GOOS != "windows" || !strings.EqualFold(sh.session.ProviderID, "shell") || sh.termBackend.SupportsRawMode() {
		return data
	}
	return sh.shellInput.Normalize(data)
}

func (sh *SessionHost) collectAttachedInputLocked(conn net.Conn, data []byte) []byte {
	// Only active writer (or first attached client) can send input to child process
	if sh.activeWriter != nil && sh.activeWriter != conn {
		return nil
	}
	if sh.activeWriter == nil {
		sh.activeWriter = conn
	}
	// This is deliberately a byte-for-byte copy. Escape, control, UTF-8 and
	// provider-specific keyboard sequences all belong to the child PTY.
	return append([]byte(nil), data...)
}

func (sh *SessionHost) handleControlCommandLocked(cmd string) {
	route := RouteSlashCommand(cmd, sh.session)
	if route.Intercepted && route.Response != "" {
		sh.broadcast([]byte("\r\n" + route.Response + "\r\n"))
	}

	switch route.Action {
	case "detach":
		for c := range sh.clients {
			_ = c.Close()
		}
		sh.clients = make(map[net.Conn]bool)
		sh.fanout.Close()
		sh.activeWriter = nil
	case "stop":
		go sh.Stop()
	case "handoff":
		if PerformAccountHandoff != nil {
			go func(target string) {
				_, err := PerformAccountHandoff(context.Background(), sh.session.RuntimeID, target)
				if err != nil {
					sh.broadcast([]byte(fmt.Sprintf("\r\n[Nexus Control] Handoff failed: %v\r\n", err)))
				} else {
					sh.broadcast([]byte("\r\n[Nexus Control] Handoff succeeded.\r\n"))
				}
			}(route.ActionArg)
		}
	case "continue":
		if PerformContextHandoff != nil {
			go func(target string) {
				var provider, profile string
				if idx := strings.Index(target, ":"); idx != -1 {
					provider = target[:idx]
					profile = target[idx+1:]
				} else {
					provider = target
				}
				_, err := PerformContextHandoff(context.Background(), sh.session.RuntimeID, provider, profile)
				if err != nil {
					sh.broadcast([]byte(fmt.Sprintf("\r\n[Nexus Control] Context handoff failed: %v\r\n", err)))
				} else {
					sh.broadcast([]byte("\r\n[Nexus Control] Context handoff succeeded.\r\n"))
				}
			}(route.ActionArg)
		}
	}
}

func (sh *SessionHost) removeClient(conn net.Conn) {
	sh.mu.Lock()
	defer sh.mu.Unlock()
	delete(sh.clients, conn)
	sh.fanout.RemoveClient(conn)
	if sh.activeWriter == conn {
		sh.activeWriter = nil
	}
}

func (sh *SessionHost) waitProcess() {
	err := sh.termBackend.Wait()
	<-sh.streamDone

	slog.Info(
		"nexus ipc runtime wait finished",
		"runtime_id", sh.session.RuntimeID,
		"provider_id", sh.session.ProviderID,
		"wait_error", fmt.Sprintf("%v", err),
	)

	// A delayed ESC/CSI timer must not outlive the provider it would write to.
	// Close the stateful input boundary before the terminal backend is torn down.
	if sh.terminalInput != nil {
		_ = sh.terminalInput.Close()
	}

	sh.mu.Lock()
	defer sh.mu.Unlock()

	_ = sh.termBackend.Close()
	sh.releaseCodexTUILockLocked()

	// Tear down IPC surfaces so attached clients disconnect cleanly instead of
	// hanging on a dead runtime.
	sh.closeListenerLocked("runtime-process-exited")
	sh.fanout.Close()
	for c := range sh.clients {
		_ = c.Close()
	}
	sh.clients = make(map[net.Conn]bool)
	sh.activeWriter = nil

	state := registry.StateStopped
	if err != nil {
		select {
		case <-sh.stopChan:
			// A non-zero exit after an explicit stop/terminate is expected and
			// must not be reported as a provider failure.
		default:
			state = registry.StateFailed
		}
	}
	sh.session.State = state
	_ = sh.registry.UpdateState(sh.session.RuntimeID, state)

	eventData := sh.lifecycleEventData(map[string]any{"state": string(state)})
	if state == registry.StateFailed {
		for key, value := range runtimeFailureEventMetadata(err, sh.ringBuffer.Bytes()) {
			eventData[key] = value
		}
	}
	events.DefaultBus().Publish(events.NewEventWithCorrelation(
		sh.session.LineageID,
		sh.session.RuntimeID,
		sh.session.ProviderID,
		sh.session.ProfileID,
		events.EventProcessExited,
		fmt.Sprintf("Process exited (State: %s)", state),
		eventData,
	))
	runtimeEvent := events.EventRuntimeStopped
	if state == registry.StateFailed {
		runtimeEvent = events.EventRuntimeFailed
	}
	events.DefaultBus().Publish(events.NewEventWithCorrelation(
		sh.session.LineageID,
		sh.session.RuntimeID,
		sh.session.ProviderID,
		sh.session.ProfileID,
		runtimeEvent,
		fmt.Sprintf("Runtime ended (State: %s)", state),
		eventData,
	))

	sh.doneOnce.Do(func() { close(sh.doneChan) })
}

// runtimeFailureEventMetadata enriches failed-runtime lifecycle events with
// bounded, classifier-owned descriptions, never raw terminal output. This is
// diagnostic evidence, not authorization to replay or automatically fail over.
func runtimeFailureEventMetadata(err error, output []byte) map[string]any {
	if err == nil {
		return nil
	}
	failure := classifier.Classify(err, string(output))
	if failure.Kind == "" {
		return nil
	}
	metadata := map[string]any{
		"failure_kind":    string(failure.Kind),
		"failure_message": failure.Message,
	}
	if failure.RetryAfter != nil && *failure.RetryAfter > 0 {
		metadata["retry_after"] = failure.RetryAfter.String()
	}
	return metadata
}

func (sh *SessionHost) releaseCodexTUILockLocked() {
	if sh.codexTUILock != nil {
		_ = sh.codexTUILock.Release()
		sh.codexTUILock = nil
	}
}

// lifecycleEventData adds ownership metadata to runtime/process events when
// the launcher already knows it. It never derives ownership from runtime IDs.
func (sh *SessionHost) lifecycleEventData(data map[string]any) map[string]any {
	result := make(map[string]any, len(data)+2)
	for key, value := range data {
		result[key] = value
	}
	if sh.session.ProjectID != "" {
		result["project_id"] = sh.session.ProjectID
	}
	if sh.session.AgentID != "" {
		result["agent_id"] = sh.session.AgentID
	}
	return result
}

// Wait blocks until the supervised child process terminates.
func (sh *SessionHost) Wait() {
	<-sh.doneChan
}

// Stop gracefully stops the supervised process.
func (sh *SessionHost) Stop() error {
	sh.stopOnce.Do(func() { close(sh.stopChan) })
	sh.mu.Lock()
	provider := sh.session.ProviderID
	profileName := sh.session.ProfileID
	_ = sh.termBackend.Signal(gracefulStopSignal(provider))
	sh.mu.Unlock()

	select {
	case <-sh.doneChan:
		refreshUsageAfterSession(provider, profileName)
		return nil
	case <-time.After(gracefulStopWait(provider)):
		return sh.Terminate()
	}
}

func gracefulStopWait(providerID string) time.Duration {
	if strings.EqualFold(strings.TrimSpace(providerID), "shell") {
		// Interactive shells ignore SIGINT at an idle prompt, so a 3s Ctrl+C
		// wait just stalls the Close dialog. TERM then a short kill is enough.
		return 250 * time.Millisecond
	}
	return 3 * time.Second
}

func gracefulStopSignal(providerID string) os.Signal {
	if strings.EqualFold(strings.TrimSpace(providerID), "shell") {
		return syscall.SIGTERM
	}
	return os.Interrupt
}

// Terminate forcefully kills the supervised process.
func (sh *SessionHost) Terminate() error {
	sh.stopOnce.Do(func() { close(sh.stopChan) })
	sh.mu.Lock()
	provider := sh.session.ProviderID
	profileName := sh.session.ProfileID
	listener := sh.listener
	done := sh.doneChan
	sh.mu.Unlock()

	select {
	case <-done:
		refreshUsageAfterSession(provider, profileName)
		return nil
	default:
	}
	if listener != nil {
		_ = listener.Close()
		slog.Info(
			"nexus ipc listener closed",
			"runtime_id", sh.session.RuntimeID,
			"reason", "stop-requested",
		)
	}
	killErr := sh.termBackend.Kill()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		if killErr != nil {
			return fmt.Errorf("failed to terminate runtime process tree: %w", killErr)
		}
		return fmt.Errorf("timed out waiting for runtime process tree to terminate")
	}
	refreshUsageAfterSession(provider, profileName)
	return nil
}

// refreshUsageAfterSession re-reads provider quota after a session ends so
// live evidence written during the session becomes visible without opening
// `nexus usage` manually. Non-blocking.
func refreshUsageAfterSession(providerID, profileID string) {
	providerID = strings.TrimSpace(providerID)
	profileID = strings.TrimSpace(profileID)
	if providerID == "" || profileID == "" {
		return
	}
	switch strings.ToLower(providerID) {
	case "codex", "agy", "claude", "gemini", "opencode":
	default:
		return
	}
	go func() {
		_ = profile.RefreshUsageSnapshot(providerID, profileID)
	}()
}
