package host

import (
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
	"sync/atomic"
	"syscall"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/agentwire"
	"github.com/IAPro-Community/nexus/internal/control/events"
	"github.com/IAPro-Community/nexus/internal/control/protocol"
	"github.com/IAPro-Community/nexus/internal/control/registry"
	"github.com/IAPro-Community/nexus/internal/control/terminal"
	"github.com/IAPro-Community/nexus/internal/core/classifier"
	"github.com/IAPro-Community/nexus/internal/core/provider/adapters/codex"
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

	// terminalBackend is an internal test seam. Production callers
	// leave it nil and receive the platform backend from terminal.NewBackend.
	terminalBackend terminal.Backend
	// auditBus is an internal test seam. Production hosts publish to the shared
	// event bus, whose Nexus recorder persists lease transitions to SQLite.
	auditBus *events.Bus
	// SessionIDDiscoverer optionally probes the provider's own session store
	// for the live provider session id of terminal runtimes that cannot report
	// it over a structured channel. It must be a read-only observer: no PTY
	// writes, no profile mutations, no contention with the interactive TUI.
	// Production wiring resolves it from the provider driver; tests may inject
	// a stub. Nil disables discovery (structured backends do not need it).
	SessionIDDiscoverer SessionIDDiscovererFunc
}

// SessionHost manages a single supervised process runtime and its IPC listener.
type SessionHost struct {
	mu                sync.RWMutex
	session           registry.RuntimeSession
	registry          *registry.Registry
	cfg               Config
	cmd               *exec.Cmd
	termBackend       terminal.Backend
	ringBuffer        *RingBuffer
	fanout            *BoundedFanout
	listener          net.Listener
	clients           map[net.Conn]bool
	clientIDs         map[net.Conn]string
	clientActors      map[net.Conn]string
	clientSequence    atomic.Uint64
	activeWriter      net.Conn
	activeWriterEpoch uint64
	auditBus          *events.Bus
	stopChan          chan struct{}
	doneChan          chan struct{}
	streamDone        chan struct{}
	detector          *AttentionDetector
	quotaWatcher      *QuotaLimitWatcher
	shellInput        shellPipeInputNormalizer
	terminalInput     *nexusruntime.TerminalOutputScrubber
	terminalMouse     *nexusruntime.TerminalMouseState
	stopOnce          sync.Once
	doneOnce          sync.Once
	codexTUILock      interface{ Release() error }
	wireReducer       *agentwire.Reducer
	wireEpoch         uint64
	wireSequence      atomic.Uint64
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
	protocolMode := cfg.Session.Labels["nexus.agent_protocol"]
	structuredMode := protocolMode == "acp" || protocolMode == "pi_rpc"
	if termBackend == nil && !structuredMode {
		termBackend = terminal.NewBackendForProvider(cfg.Session.ProviderID)
	}

	sh := &SessionHost{
		session:      cfg.Session,
		registry:     cfg.Registry,
		cfg:          cfg,
		cmd:          cmd,
		termBackend:  termBackend,
		ringBuffer:   NewRingBuffer(128 * 1024), // 128 KB terminal history
		fanout:       NewBoundedFanout(256),
		clients:      make(map[net.Conn]bool),
		clientIDs:    make(map[net.Conn]string),
		clientActors: make(map[net.Conn]string),
		auditBus:     cfg.auditBus,
		stopChan:     make(chan struct{}),
		doneChan:     make(chan struct{}),
		streamDone:   make(chan struct{}),
	}
	if sh.auditBus == nil {
		sh.auditBus = events.DefaultBus()
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
	sh.detector.SetControlPolicy(cfg.Session.ControlLevel, structuredMode, cfg.Session.AgentID)

	// Quota pattern watcher (read-only): accelerates the quota evidence check
	// when provider limit language shows up in the output stream. It never
	// terminates or mutates the session by itself.
	sh.quotaWatcher = NewQuotaLimitWatcher(
		cfg.Session.RuntimeID,
		cfg.Session.ProviderID,
		cfg.Session.ProfileID,
		cfg.Session.LineageID,
		time.Now(),
		sh.quotaPatternPublisher(),
	)

	if termBackend == nil && structuredMode {
		sh.wireReducer = agentwire.NewReducer()
		sh.wireEpoch = uint64(time.Now().UnixNano())
		// The wire backend is the sole owner of this provider child process.
		// It publishes state via the existing detector / runtime registry.
		sh.termBackend = newStructuredBackend(protocolMode, cfg.Cwd,
			func(signal agentwire.WireSignal) {
				sh.observeStructuredSignal(signal, protocolMode)
			},
			func(sessionID string) {
				_ = sh.registry.UpdateProviderSessionID(sh.session.RuntimeID, sessionID)
				sh.mu.Lock()
				sh.session.ProviderSessionID = sessionID
				sh.mu.Unlock()
			},
		)
	}
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

	// Fence every attention event to this exact SessionHost incarnation.
	// Install the identity BEFORE the output reader can publish callbacks.
	generation := time.Now().UnixNano()
	sh.detector.SetRuntimeGeneration(generation, generation)

	// Drain terminal output independently and let lifecycle finalization wait for
	// the reader to observe EOF. This prevents Windows cleanup from racing live
	// ConPTY/pipe handles after the provider process has already exited.
	go func() {
		defer close(sh.streamDone)
		sh.streamReader(sh.termBackend)
	}()

	// Update session details
	sh.session.PID = sh.termBackend.PID()
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

	// Session continuity discovery (PTY runtimes). Structured backends learn
	// the provider session id from the provider protocol; terminal runtimes
	// only get it from the provider's own session store. The watcher is a
	// read-only observer and never touches the input path.
	if sh.cfg.SessionIDDiscoverer != nil {
		go sh.discoverProviderSessionID(time.Now())
	}

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
						sh.observeOutputChunk(chunk)
					} else if len(pending) > MaxFrameSize {
						chunk := pending
						pending = nil
						sh.ringBuffer.Write(chunk)
						sh.broadcast(chunk)
						sh.observeOutputChunk(chunk)
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
				sh.observeOutputChunk(line)
			}
		}
		if err != nil {
			_ = scrubber.Flush()
			flushFiltered()
			if len(pending) > 0 && !isHostProtocolControlFrame(pending) {
				sh.ringBuffer.Write(pending)
				sh.broadcast(pending)
				sh.observeOutputChunk(pending)
			}
			break
		}
	}
}

// observeStructuredSignal applies native events through the same evidence
// reducer for each runtime. Non-authoritative progress cannot silently clear a
// pending approval, and stale provider generations are isolated by epoch.
func (sh *SessionHost) observeStructuredSignal(signal agentwire.WireSignal, source string) {
	sh.mu.RLock()
	state := sh.session.State
	runtimeID := sh.session.RuntimeID
	sh.mu.RUnlock()
	if state == registry.StateStopping || state == registry.StateStopped || state == registry.StateFailed {
		return
	}
	if sh.wireReducer != nil {
		observation, err := sh.wireReducer.ObserveWireSignal(
			runtimeID, sh.wireEpoch, sh.wireSequence.Add(1), signal,
		)
		if err != nil {
			slog.Warn("invalid structured event", "runtime_id", runtimeID, "error", err)
			return
		}
		// A generic tool/progress event is not proof that the provider accepted
		// an outstanding answer. A correlated response-submitted signal may move
		// the Registry card to working, but only settlement clears the request.
		if len(observation.Attention) != 0 && (signal.State == "working" || signal.State == "idle") &&
			signal.Kind != agentwire.WireSignalResponseSubmitted {
			return
		}
	}
	if signal.Kind == agentwire.WireSignalResponseSubmitted {
		source += "_response_submitted"
	}
	options := make([]registry.AttentionOption, 0, len(signal.Options))
	for _, option := range signal.Options {
		options = append(options, registry.AttentionOption{ID: option.ID, Name: option.Name})
	}
	sh.detector.ApplyProviderSignalWithOptions(signal.State, signal.Kind, signal.Summary, source, signal.InteractionID, options)
}

// isHostProtocolControlFrame detects Nexus Control RPC lines that must never
// enter the ring buffer / fanout (e.g. lease_acquire echoed into the PTY).
func isHostProtocolControlFrame(data []byte) bool {
	return protocol.IsWireControlFrame(data)
}

func (sh *SessionHost) broadcast(data []byte) {
	sh.fanout.Broadcast(data)
}

// errFrameTooLarge is returned when an IPC frame exceeds MaxFrameSize.
var errFrameTooLarge = errors.New("IPC frame exceeds maximum allowed size")

func (sh *SessionHost) processAttachedInput(conn net.Conn, data []byte) {
	nexusruntime.TraceTerminalBytes(sh.session.RuntimeID, "host.raw_attach.input", data)
	sh.mu.Lock()
	forward, auditEvent := sh.collectAttachedInputLocked(conn, data)
	forward = sh.normalizeShellPipeInputLocked(forward)
	sh.mu.Unlock()
	if auditEvent != nil {
		sh.publishTerminalAuditEvents([]events.Event{*auditEvent})
	}

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
	return err
}

func (sh *SessionHost) normalizeShellPipeInputLocked(data []byte) []byte {
	if runtime.GOOS != "windows" || !strings.EqualFold(sh.session.ProviderID, "shell") || sh.termBackend.SupportsRawMode() {
		return data
	}
	return sh.shellInput.Normalize(data)
}

// terminalClientIDLocked returns a process-local opaque connection ID. It is
// stored in event summaries so lease changes remain traceable after the
// bounded in-memory event history is gone, without logging auth/session data.
func (sh *SessionHost) terminalClientIDLocked(conn net.Conn) string {
	if sh.clientIDs == nil {
		sh.clientIDs = make(map[net.Conn]string)
	}
	if clientID := sh.clientIDs[conn]; clientID != "" {
		return clientID
	}
	clientID := fmt.Sprintf("host-%d-client-%d", sh.session.HostGeneration, sh.clientSequence.Add(1))
	sh.clientIDs[conn] = clientID
	return clientID
}

func validTerminalAuditIdentity(value string) bool {
	if len(value) == 0 || len(value) > 96 {
		return false
	}
	for _, char := range value {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '-' || char == '_' {
			continue
		}
		return false
	}
	return true
}

func (sh *SessionHost) bindTerminalClientIdentityLocked(conn net.Conn, payload json.RawMessage) {
	var identity protocol.TerminalClientPayload
	if len(payload) == 0 || json.Unmarshal(payload, &identity) != nil ||
		!validTerminalAuditIdentity(identity.ClientID) || !validTerminalAuditIdentity(identity.ActorID) {
		return
	}
	if sh.clientIDs == nil {
		sh.clientIDs = make(map[net.Conn]string)
	}
	if sh.clientActors == nil {
		sh.clientActors = make(map[net.Conn]string)
	}
	sh.clientIDs[conn] = identity.ClientID
	sh.clientActors[conn] = identity.ActorID
}

// terminalLeaseEventLocked creates a scoped audit record while the host state
// lock is held. Callers publish it only after releasing sh.mu because the
// recorder may synchronously write to durable storage.
func (sh *SessionHost) terminalLeaseEventLocked(kind events.EventType, clientID, reason string) events.Event {
	verb := map[events.EventType]string{
		events.EventTerminalLeaseGranted:  "granted",
		events.EventTerminalLeaseRevoked:  "revoked",
		events.EventTerminalLeaseReleased: "released",
	}[kind]
	data := map[string]any{
		"client_id":        clientID,
		"actor_id":         sh.clientActorsForEventLocked(clientID),
		"lease_transition": verb,
		"reason":           reason,
		"host_generation":  sh.session.HostGeneration,
	}
	if sh.session.ProjectID != "" {
		data["project_id"] = sh.session.ProjectID
	}
	if sh.session.AgentID != "" {
		data["agent_id"] = sh.session.AgentID
	}
	actorID := sh.clientActorsForEventLocked(clientID)
	summary := fmt.Sprintf("Terminal lease %s client=%s actor=%s generation=%d reason=%s", verb, clientID, actorID, sh.session.HostGeneration, reason)
	return events.NewEventWithCorrelation(
		sh.session.LineageID, sh.session.RuntimeID, sh.session.ProviderID,
		sh.session.ProfileID, kind, summary, data,
	)
}

func (sh *SessionHost) clientActorsForEventLocked(clientID string) string {
	for conn, candidate := range sh.clientIDs {
		if candidate == clientID {
			if actorID := sh.clientActors[conn]; actorID != "" {
				return actorID
			}
			return "local"
		}
	}
	return "unknown"
}

func (sh *SessionHost) publishTerminalAuditEvents(auditEvents []events.Event) {
	if sh.auditBus == nil {
		return
	}
	for _, auditEvent := range auditEvents {
		sh.auditBus.Publish(auditEvent)
	}
}

func (sh *SessionHost) applyTerminalLeaseLocked(conn net.Conn, payload []byte) (bool, string) {
	var leaseCmd struct {
		Epoch uint64 `json:"epoch"`
	}
	if len(payload) > 0 {
		_ = json.Unmarshal(payload, &leaseCmd)
	}
	if leaseCmd.Epoch == 0 {
		if sh.activeWriterEpoch > 0 {
			return false, "lease_stale"
		}
	} else if leaseCmd.Epoch < sh.activeWriterEpoch ||
		(leaseCmd.Epoch == sh.activeWriterEpoch && sh.activeWriter != conn) {
		return false, "lease_stale"
	}
	sh.activeWriter = conn
	if leaseCmd.Epoch > sh.activeWriterEpoch {
		sh.activeWriterEpoch = leaseCmd.Epoch
	}
	return true, "lease_acquired"
}

func (sh *SessionHost) collectAttachedInputLocked(conn net.Conn, data []byte) ([]byte, *events.Event) {
	// Only active writer (or first attached client) can send input to child process
	if sh.activeWriter != nil && sh.activeWriter != conn {
		nexusruntime.TraceTerminalBytes(sh.session.RuntimeID, "host.input.dropped.lease", data)
		return nil, nil
	}
	var auditEvent *events.Event
	if sh.activeWriter == nil {
		sh.activeWriter = conn
		event := sh.terminalLeaseEventLocked(events.EventTerminalLeaseGranted, sh.terminalClientIDLocked(conn), "implicit_input")
		auditEvent = &event
	}
	// This is deliberately a byte-for-byte copy. Escape, control, UTF-8 and
	// provider-specific keyboard sequences all belong to the child PTY.
	return append([]byte(nil), data...), auditEvent
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
	clientID := sh.clientIDs[conn]
	delete(sh.clients, conn)
	sh.fanout.RemoveClient(conn)
	var auditEvents []events.Event
	if sh.activeWriter == conn {
		sh.activeWriter = nil
		auditEvents = append(auditEvents, sh.terminalLeaseEventLocked(events.EventTerminalLeaseReleased, clientID, "disconnect"))
	}
	delete(sh.clientIDs, conn)
	delete(sh.clientActors, conn)
	sh.mu.Unlock()
	sh.publishTerminalAuditEvents(auditEvents)
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
	// Process exit invalidates outstanding terminal and protocol prompts.
	// Do not leave a stale needs_user notification for a dead runtime.
	sh.session.AttentionReason = ""
	sh.session.AttentionContext = ""
	sh.session.AttentionFingerprint = ""
	sh.session.PromptKind = PromptKindNone
	sh.session.AttentionKind = AttentionIdle
	_ = sh.registry.UpdateAttentionMeta(sh.session.RuntimeID, registry.AttentionUpdate{
		State: state, Reason: "", Context: "", Fingerprint: "",
		PromptKind: PromptKindNone, Kind: AttentionIdle,
		Generation: fmt.Sprintf("%d:%d", sh.session.HostGeneration, sh.session.ProviderGeneration),
	})
	eventData := sh.lifecycleEventData(map[string]any{"state": string(state)})
	lineageID := sh.session.LineageID
	runtimeID := sh.session.RuntimeID
	providerID := sh.session.ProviderID
	profileID := sh.session.ProfileID
	detector := sh.detector
	sh.mu.Unlock()

	if detector != nil {
		detector.Clear()
	}

	if state == registry.StateFailed {
		for key, value := range runtimeFailureEventMetadata(err, sh.ringBuffer.Bytes()) {
			eventData[key] = value
		}
	}
	events.DefaultBus().Publish(events.NewEventWithCorrelation(
		lineageID,
		runtimeID,
		providerID,
		profileID,
		events.EventProcessExited,
		fmt.Sprintf("Process exited (State: %s)", state),
		eventData,
	))
	runtimeEvent := events.EventRuntimeStopped
	if state == registry.StateFailed {
		runtimeEvent = events.EventRuntimeFailed
	}
	events.DefaultBus().Publish(events.NewEventWithCorrelation(
		lineageID,
		runtimeID,
		providerID,
		profileID,
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
	case "codex", "agy", "claude", "opencode":
	default:
		return
	}
	go func() {
		_ = profile.RefreshUsageSnapshot(providerID, profileID)
	}()
}
