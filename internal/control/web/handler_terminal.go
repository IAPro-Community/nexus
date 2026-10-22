package web

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/events"
	"github.com/IAPro-Community/nexus/internal/control/protocol"
	"github.com/IAPro-Community/nexus/internal/control/registry"
	providerspec "github.com/IAPro-Community/nexus/internal/core/provider/spec"
	"github.com/IAPro-Community/nexus/internal/nexus"
	"github.com/IAPro-Community/nexus/internal/nexus/promotion"
	nexusruntime "github.com/IAPro-Community/nexus/internal/runtime"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: CheckOrigin,
}

type TerminalMessage struct {
	Type  string `json:"type"` // "output", "input", "resize", "lease", "promotion", "trace", "error"
	Data  string `json:"data,omitempty"`
	Rows  int    `json:"rows,omitempty"`
	Cols  int    `json:"cols,omitempty"`
	Role  string `json:"role,omitempty"` // lease: CONTROL/VIEW_ONLY; promotion: ASK/PROMOTED/FALLBACK/BLOCKED
	Stage string `json:"stage,omitempty"`
	Size  int    `json:"size,omitempty"`
}

func traceDocsCaptureTerminalStage(stage string) {
	if os.Getenv("NEXUS_DOCS_CAPTURE") == "1" {
		log.Printf("nexus_docs_capture_terminal_stage=%s", stage)
	}
}

type TerminalHub struct {
	auth     *AuthManager
	registry *registry.Registry
	nexus    *nexus.Nexus
}

func (h *TerminalHub) SetNexus(n *nexus.Nexus) { h.nexus = n }

func NewTerminalHub(auth *AuthManager, registries ...*registry.Registry) *TerminalHub {
	reg := registry.DefaultRegistry()
	if len(registries) > 0 && registries[0] != nil {
		reg = registries[0]
	}
	return &TerminalHub{
		auth:     auth,
		registry: reg,
	}
}

// attachDialBudget bounds how long HandleWebSocket waits for the runtime's
// control pipe before answering 404. The launch handshake routinely needs
// multiple attempts over a couple of seconds to reach the pipe on a loaded
// machine, so a single short budget turned a live-but-slow host into a
// spurious 404 ("Unexpected response code: 404") that broke the terminal
// reconnect. The budget stays comfortably inside the frontend's
// TERMINAL_CONNECT_TIMEOUT_MS (30s) and the capture's lease timer (35s).
const attachDialBudget = 15 * time.Second

// dialAttachControl dials the runtime control pipe with the same patience the
// launch handshake uses: short per-attempt deadlines retried inside one bounded
// budget. Fast-fail stays intact because HostLive re-checks the host process
// (and its generation) after every attempt — a genuinely dead runtime still
// 404s within a single ~500ms attempt instead of burning the whole budget.
func dialAttachControl(sess registry.RuntimeSession, runtimeID string) (*protocol.Client, error) {
	return dialAttachControlBudget(sess, runtimeID, attachDialBudget)
}

func dialAttachControlBudget(
	sess registry.RuntimeSession,
	runtimeID string,
	budget time.Duration,
) (*protocol.Client, error) {
	started := time.Now()
	deadline := started.Add(budget)
	attempt := 0
	for {
		attempt++
		attemptCtx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		client, err := protocol.NewClientContext(attemptCtx, runtimeID)
		cancel()
		if err == nil {
			if os.Getenv("NEXUS_DOCS_CAPTURE") == "1" {
				log.Printf(
					"nexus_docs_capture_diagnostic=pipe_dial_ready attempts=%d elapsed=%s",
					attempt,
					time.Since(started),
				)
			}
			return client, nil
		}
		if os.Getenv("NEXUS_DOCS_CAPTURE") == "1" {
			log.Printf("nexus_docs_capture_diagnostic=pipe_dial_attempt attempt=%d err=%q", attempt, err.Error())
		}
		if !sess.HostLive() {
			return nil, err
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("control pipe did not accept within %s (%d attempts): %w", budget, attempt, err)
		}
		<-time.After(30 * time.Millisecond)
	}
}

func (h *TerminalHub) HandleWebSocket(w http.ResponseWriter, r *http.Request, agentID string, runtimeID string) {
	traceDocsCaptureTerminalStage("request")
	if h.auth != nil && h.auth.AuthenticateRequest(r) == nil {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	traceDocsCaptureTerminalStage("authenticated")

	reg := h.registry
	sess, exists := reg.Get(runtimeID)
	if !exists {
		traceDocsCaptureTerminalStage("runtime-missing")
		writeError(w, http.StatusNotFound, "runtime not found")
		return
	}
	traceDocsCaptureTerminalStage("runtime-found")
	if !sess.HostLive() {
		// Do not upgrade: a "connected" WebSocket into a dead host leaves the
		// UI on a black xterm with no recover action (especially window chrome).
		writeError(w, http.StatusNotFound, "runtime host is not running")
		return
	}
	traceDocsCaptureTerminalStage("host-live")

	// Dial the control endpoint BEFORE the attach validation. A live
	// PID/STARTING row is not enough on Windows when the named pipe was never
	// created or its host is gone; prefer a fast HTTP 404 over a WebSocket
	// error overlay ("Runtime do Agente desconectado"). Dialing first keeps
	// the cheap liveness proof ahead of the multi-second binary identity
	// checks, so a dead host fails fast instead of burning the client's lease
	// budget on validation.
	traceDocsCaptureTerminalStage("pipe-dial-start")
	client, err := dialAttachControl(sess, runtimeID)
	if err != nil {
		traceDocsCaptureTerminalStage("pipe-dial-failed")
		if os.Getenv("NEXUS_DOCS_CAPTURE") == "1" {
			log.Printf("nexus_docs_capture_diagnostic=pipe_dial_error %s", err.Error())
		}
		writeError(w, http.StatusNotFound, "runtime host is not running ("+err.Error()+")")
		return
	}
	traceDocsCaptureTerminalStage("pipe-connected")

	compatibilityStarted := time.Now()
	traceDocsCaptureTerminalStage("attach-validation-start")
	if err := registry.ValidateAttachCompatibility(sess); err != nil {
		// Preserve the process for explicit user cleanup, but make the runtime
		// non-attachable and actionable instead of opening a silent black TTY.
		_ = client.Close()
		_ = reg.UpdateState(runtimeID, registry.StateStale)
		writeError(w, http.StatusConflict, "runtime is incompatible with the current Nexus binary; recreate the session with nexus codex --yolo")
		return
	}
	if os.Getenv("NEXUS_DOCS_CAPTURE") == "1" {
		log.Printf("nexus_docs_capture_diagnostic=attach_validation_elapsed=%s", time.Since(compatibilityStarted))
	}
	traceDocsCaptureTerminalStage("attach-validation-finished")
	traceDocsCaptureTerminalStage("attach-compatible")

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		_ = client.Close()
		return
	}
	defer ws.Close()
	defer client.Close()
	traceDocsCaptureTerminalStage("websocket-upgraded")

	var wsMu sync.Mutex
	safeWriteJSON := func(v any) error {
		wsMu.Lock()
		defer wsMu.Unlock()
		return ws.WriteJSON(v)
	}

	broker := DefaultBroker()
	hasRuntime := runtimeID != ""
	role := broker.Attach(agentID, ws, hasRuntime, &wsMu)
	defer broker.Detach(agentID, ws)

	stopChan := make(chan struct{})
	var stopOnce sync.Once
	stop := func() {
		stopOnce.Do(func() {
			close(stopChan)
			_ = ws.Close()
		})
	}
	defer stop()

	// Attach to host
	traceDocsCaptureTerminalStage("attach-command-start")
	resp, err := client.Send(protocol.CmdAttach, nil)
	if err != nil {
		// Same as NewClient: attach timeout is not proof the child exited.
		_ = safeWriteJSON(TerminalMessage{
			Type: "error",
			Data: "Failed to attach: runtime host is no longer responding (" + err.Error() + ").",
		})
		return
	}
	_ = client.ClearDeadline()
	traceDocsCaptureTerminalStage("attach-command-complete")

	rawConn := client.RawConn()
	resizeRequests := make(chan [2]int, 1)
	go func() {
		for {
			select {
			case <-stopChan:
				return
			case size := <-resizeRequests:
				rpcClient, err := protocol.NewClient(runtimeID)
				if err == nil {
					_ = rpcClient.Resize(size[0], size[1])
					_ = rpcClient.Close()
				}
			}
		}
	}()

	// Keep the SessionHost writer authority aligned with the Agent-level
	// broker. The command is deliberately sent over this connection *after*
	// Attach so SessionHost can bind the lease to the exact streaming client,
	// not to an unrelated RPC connection.
	//
	// Never demote CONTROL when the host write fails: the broker already
	// granted the seat; the client can retry lease_acquire. Demoting here
	// left the UI read-only even though no other writer held the lease.
	effectiveRole := role
	if role == "CONTROL" {
		traceDocsCaptureTerminalStage("lease-acquire-pipe-start")
		_ = sendAttachedCommandWithRetry(rawConn, 3)
		traceDocsCaptureTerminalStage("lease-acquire-pipe-finished")
	}
	traceDocsCaptureTerminalStage("lease-websocket-write-start")
	if err := safeWriteJSON(TerminalMessage{
		Type: "lease",
		Role: effectiveRole,
	}); err != nil {
		traceDocsCaptureTerminalStage("lease-write-failed")
	} else {
		traceDocsCaptureTerminalStage("lease-sent")
	}
	if nexusruntime.TerminalTraceEnabled() {
		_ = safeWriteJSON(TerminalMessage{Type: "trace_config", Data: "enabled"})
	}

	if ch := DefaultBroker().WatchRuntimeChanged(agentID); ch != nil {
		go func() {
			select {
			case <-ch:
				stop()
			case <-stopChan:
			}
		}()
	}

	// Initial ring buffer history. Older hosts may have echoed lease_acquire
	// into the PTY (and thus the ring); scrub so attach never floods xterm.
	var history string
	if json.Unmarshal(resp.Data, &history) == nil && history != "" {
		if cleaned := scrubProtocolFramesFromOutput(history); cleaned != "" {
			_ = safeWriteJSON(TerminalMessage{
				Type: "output",
				Data: cleaned,
			})
		}
	}

	// Send initial dynamic title and attention if available
	if sess, ok := reg.Get(runtimeID); ok {
		if sess.DynamicTitle != "" {
			_ = safeWriteJSON(TerminalMessage{
				Type: "title",
				Data: sess.DynamicTitle,
			})
		}
		if sess.AttentionReason != "" && sess.AttentionContext != "" && sess.PromptKind != "" && sess.PromptKind != "none" {
			_ = safeWriteJSON(map[string]any{
				"type":             "attention",
				"runtime_id":       runtimeID,
				"attention_reason": sess.AttentionReason,
				"attention_kind":   sess.AttentionKind,
				"prompt_kind":      sess.PromptKind,
				"fingerprint":      sess.AttentionFingerprint,
				"context":          sess.AttentionContext,
				"project_id":       sess.ProjectID,
				"project_name":     sess.ProjectName,
				"dynamic_title":    sess.DynamicTitle,
			})
		}
	}

	// Subscribe to event bus for real-time attention and title frames
	eventCh, unsub := events.DefaultBus().Subscribe(runtimeID)
	defer unsub()
	go func() {
		for {
			select {
			case <-stopChan:
				return
			case ev, ok := <-eventCh:
				if !ok {
					return
				}
				if ev.Type == events.EventApprovalRequired || ev.Type == events.EventToolFinished {
					_ = safeWriteJSON(map[string]any{
						"type":             "attention",
						"runtime_id":       ev.RuntimeID,
						"attention_reason": ev.Data["attention_reason"],
						"attention_kind":   ev.Data["attention_kind"],
						"prompt_kind":      ev.Data["prompt_kind"],
						"fingerprint":      ev.Data["fingerprint"],
						"context":          ev.Data["context"],
						"project_id":       ev.Data["project_id"],
						"project_name":     ev.Data["project_name"],
						"dynamic_title":    ev.Data["dynamic_title"],
						"summary":          ev.Summary,
					})
				}
				if dt, ok := ev.Data["dynamic_title"].(string); ok && dt != "" {
					_ = safeWriteJSON(TerminalMessage{
						Type: "title",
						Data: dt,
					})
				}
			}
		}
	}()

	shadow := promotion.NewLineShadow()
	pendingPromotion := ""
	// Explicitly enabled Web Lead interception supports user-owned read-only
	// delegation even when autonomous Mission promotion is OFF. Ordinary Web
	// terminal sessions without this label remain byte-transparent.
	interactiveLead := webInteractiveLeadEnabled(sess)
	if !interactiveLead {
		shadow.SetEnabled(false)
	}
	startPromotion := func(prompt string, fallbackToLead bool) {
		accepted, startErr := startPromotedMission(h.nexus, sess, prompt)
		if startErr != nil {
			safeAdmissionFallback := fallbackToLead && accepted != nil && accepted.FallbackSafe
			if safeAdmissionFallback {
				if strings.TrimSpace(accepted.PlanID) != "" {
					_ = h.nexus.DeleteWorkPlan(context.Background(), accepted.PlanID)
				}
				// The held prompt is still in the provider TUI. Submit it
				// unchanged to the Lead exactly once.
				_ = writeAttachedTerminalInput(rawConn, "\r")
				_ = safeWriteJSON(TerminalMessage{
					Type: "promotion",
					Role: "FALLBACK",
					Data: fmt.Sprintf("Mission não admitida (%v); continuando com o Lead.", startErr),
				})
				return
			}

			// Explicit/confirmed Mission requests never silently degrade to
			// Direct. Untyped/internal failures are conservative too: without
			// proof that no Mission crossed the admission boundary, do not risk
			// duplicate execution.
			_ = writeAttachedTerminalInput(rawConn, clearLineSequence(sess.ProviderID))
			detail := fmt.Sprintf("Promoção para Mission falhou antes da confirmação de admissão: %v. Fallback Direct desativado por segurança.", startErr)
			if accepted != nil && accepted.AdmissionBlocked {
				detail = fmt.Sprintf("Mission solicitada, mas a admissão foi bloqueada (etapa %s): %v", accepted.AdmissionStage, startErr)
				if strings.TrimSpace(accepted.PlanID) != "" {
					detail = fmt.Sprintf("Mission solicitada, mas a admissão foi bloqueada (plan %s, etapa %s): %v", accepted.PlanID, accepted.AdmissionStage, startErr)
				}
			} else if accepted != nil && strings.TrimSpace(accepted.AdmissionStage) != "" {
				detail = fmt.Sprintf("Mission solicitada, mas a preparação falhou (etapa %s): %v. Fallback Direct desativado por segurança.", accepted.AdmissionStage, startErr)
				if strings.TrimSpace(accepted.PlanID) != "" {
					detail = fmt.Sprintf("Mission solicitada, mas a preparação falhou (plan %s, etapa %s): %v. Fallback Direct desativado por segurança.", accepted.PlanID, accepted.AdmissionStage, startErr)
				}
			}
			_ = safeWriteJSON(TerminalMessage{
				Type: "promotion",
				Role: "BLOCKED",
				Data: detail,
			})
			return
		}
		_ = writeAttachedTerminalInput(rawConn, clearLineSequence(sess.ProviderID))
		_ = safeWriteJSON(TerminalMessage{
			Type: "promotion",
			Role: "PROMOTED",
			Data: fmt.Sprintf("Mission %s iniciada.", accepted.RunID),
		})
		go waitPromotedMission(h.nexus, sess, prompt, accepted)
	}

	// Only the explicitly enabled interactive Lead needs a Web-side input
	// shadow/sequence coalescer. For direct provider input, do not add a second
	// stateful filter: SessionHost is the final PTY input authority. Stacking
	// timers across WebSocket and host caused delayed, reordered Escape/arrow
	// fragments that could interrupt an active Codex session.
	inputMu := sync.Mutex{}
	processTerminalInput := func(data string) {
		if data == "" {
			return
		}
		if h.nexus == nil || sess.ProjectID == "" || sess.AgentID == "" || !shadow.Enabled() {
			_ = writeAttachedTerminalInput(rawConn, data)
			return
		}
		event := shadow.Consume([]byte(data))
		if len(event.Forward) > 0 {
			_ = writeAttachedTerminalInput(rawConn, string(event.Forward))
		}
		if !event.Submit {
			return
		}
		for _, line := range event.Submissions {
			if promotion.IsImmediateSubmitLine(line) {
				_ = writeAttachedTerminalInput(rawConn, "\r")
				continue
			}
			if webInteractiveDelegationMode(sess) == nexus.DelegationOff {
				provider, delegated, delegateErr := h.nexus.ResolveExplicitReadOnlyProviderDelegate(line)
				if delegateErr != nil || !delegated {
					_ = writeAttachedTerminalInput(rawConn, "\r")
					continue
				}
				_ = writeAttachedTerminalInput(rawConn, clearLineSequence(sess.ProviderID))
				_ = safeWriteJSON(TerminalMessage{
					Type: "promotion", Role: "DELEGATED",
					Data: fmt.Sprintf("Delegando revisão explícita para %s sem Mission.", provider),
				})
				go runPromotedSingleTurnDelegate(h.nexus, sess, line, provider)
				continue
			}
			decideCtx, cancelDecision := context.WithTimeout(r.Context(), promotion.InteractiveDecisionTimeout)
			decision, decideErr := promotion.NewPromoter(h.nexus).Decide(decideCtx, sess.ProjectID, line, webInteractiveDelegationMode(sess))
			cancelDecision()
			if decideErr != nil || decision.Action == promotion.KeepWithLead {
				_ = writeAttachedTerminalInput(rawConn, "\r")
				continue
			}
			if decision.Action == promotion.Delegate {
				_ = writeAttachedTerminalInput(rawConn, clearLineSequence(sess.ProviderID))
				_ = safeWriteJSON(TerminalMessage{Type: "promotion", Role: "DELEGATED", Data: fmt.Sprintf("Delegando um turno para %s sem criar Mission.", decision.Intent.TaskRequirements.PreferProvider)})
				go runPromotedSingleTurnDelegate(h.nexus, sess, line, decision.Intent.TaskRequirements.PreferProvider)
				continue
			}
			if decision.Action == promotion.Ask {
				pendingPromotion = line
				_ = safeWriteJSON(TerminalMessage{Type: "promotion", Data: line, Role: "ASK"})
				continue
			}
			startPromotion(line, !decision.ExplicitMissionRequest)
		}
	}
	inputStream := nexusruntime.NewTerminalInputScrubber(terminalInputCommandWriter(func(data []byte) (int, error) {
		nexusruntime.TraceTerminalBytes(runtimeID, "web.scrubber.output", data)
		inputMu.Lock()
		defer inputMu.Unlock()
		processTerminalInput(string(data))
		return len(data), nil
	}))
	// Never flush incomplete Escape/CSI input into a live provider when this
	// browser detaches. A late timer-generated byte can corrupt the next turn.
	defer func() { _ = inputStream.Close() }()

	// 1. Pump stdout from child process to browser.
	// Buffer across reads so protocol frames (request or response) that
	// span chunk boundaries never leak into the visible terminal.
	go func() {
		buf := make([]byte, 2048)
		var pending []byte
		for {
			select {
			case <-stopChan:
				return
			default:
				_ = rawConn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
				n, err := rawConn.Read(buf)
				if n > 0 {
					pending = append(pending, buf[:n]...)
					for {
						idx := bytes.IndexByte(pending, '\n')
						if idx < 0 {
							// Flush non-protocol partial data that cannot be a
							// control frame (does not start with '{').
							if len(pending) > 0 && !protocol.MayBeWireControlPrefix(pending) {
								_ = safeWriteJSON(TerminalMessage{
									Type: "output",
									Data: string(pending),
								})
								pending = nil
							} else if len(pending) > MaxProtocolPending {
								// Oversized incomplete JSON: show as output rather than stall.
								_ = safeWriteJSON(TerminalMessage{
									Type: "output",
									Data: string(pending),
								})
								pending = nil
							}
							break
						}
						line := pending[:idx+1]
						pending = pending[idx+1:]
						if isProtocolControlFrame(line) {
							continue
						}
						_ = safeWriteJSON(TerminalMessage{
							Type: "output",
							Data: string(line),
						})
					}
				}
				if err != nil && !strings.Contains(err.Error(), "timeout") {
					if len(pending) > 0 && !isProtocolControlFrame(pending) {
						_ = safeWriteJSON(TerminalMessage{
							Type: "output",
							Data: string(pending),
						})
					}
					stop()
					return
				}
			}
		}
	}()

	// 2. Read WebSocket frames from browser
	for {
		var msg TerminalMessage
		if err := ws.ReadJSON(&msg); err != nil {
			stop()
			return
		}

		switch msg.Type {
		case "promotion_answer":
			if broker.IsWriter(agentID, ws) {
				inputMu.Lock()
				if pendingPromotion != "" {
					if strings.EqualFold(strings.TrimSpace(msg.Data), "y") || strings.EqualFold(strings.TrimSpace(msg.Data), "yes") || strings.EqualFold(strings.TrimSpace(msg.Data), "s") || strings.EqualFold(strings.TrimSpace(msg.Data), "sim") {
						startPromotion(pendingPromotion, false)
					} else {
						_ = writeAttachedTerminalInput(rawConn, "\r")
					}
					pendingPromotion = ""
				}
				inputMu.Unlock()
			}

		case "input":
			// Only explicit interactive Lead promotion uses Web-side buffering.
			// Ordinary Codex input is forwarded unmodified to the host PTY.
			if broker.IsWriter(agentID, ws) && msg.Data != "" {
				nexusruntime.TraceTerminalBytes(runtimeID, "web.websocket.input", []byte(msg.Data))
				inputMu.Lock()
				if pendingPromotion != "" {
					answer := strings.TrimSpace(msg.Data)
					if strings.EqualFold(answer, "y") || strings.EqualFold(answer, "yes") || strings.EqualFold(answer, "s") || strings.EqualFold(answer, "sim") {
						startPromotion(pendingPromotion, false)
					} else {
						_ = writeAttachedTerminalInput(rawConn, "\r")
					}
					pendingPromotion = ""
					inputMu.Unlock()
					continue
				}
				inputMu.Unlock()
				dispatchWebTerminalInput(msg.Data, shadow.Enabled(),
					func(raw string) {
						inputMu.Lock()
						defer inputMu.Unlock()
						processTerminalInput(raw)
					},
					func(raw string) { _, _ = inputStream.Write([]byte(raw)) },
				)
			}

		case "trace":
			if broker.IsWriter(agentID, ws) && nexusruntime.TerminalTraceEnabled() {
				switch msg.Stage {
				case "xterm.ondata", "xterm.filtered":
					nexusruntime.TraceTerminalShape(runtimeID, "web."+msg.Stage, msg.Data, msg.Size)
				}
			}

		case "resize":
			if msg.Rows > 0 && msg.Cols > 0 {
				size := [2]int{msg.Rows, msg.Cols}
				select {
				case resizeRequests <- size:
				default:
					// Keep only the newest dimensions while the RPC worker is busy.
					select {
					case <-resizeRequests:
					default:
					}
					select {
					case resizeRequests <- size:
					default:
					}
				}
			}

		case "lease_acquire":
			if broker.TakeControl(agentID, ws) {
				_ = safeWriteJSON(TerminalMessage{
					Type: "lease",
					Role: "CONTROL",
				})
				_ = sendAttachedCommandWithRetry(rawConn, 3)
			}

		case "lease_release":
			broker.ReleaseControl(agentID, ws)
			_ = safeWriteJSON(TerminalMessage{
				Type: "lease",
				Role: "VIEW_ONLY",
			})
			_ = sendAttachedCommand(rawConn, protocol.CmdLeaseRelease, nil)
		}
	}
}

// dispatchWebTerminalInput keeps DIRECT/STICKY_LEAD bytes out of the extra
// Web-side coalescer. Explicit interactive Lead promotion is the only path
// allowed to hold/interpret input before the SessionHost scrubber.
func dispatchWebTerminalInput(data string, interactiveLead bool, direct, promotion func(string)) {
	if data == "" {
		return
	}
	if interactiveLead {
		promotion(data)
		return
	}
	direct(data)
}

type terminalInputCommandWriter func([]byte) (int, error)

func (w terminalInputCommandWriter) Write(data []byte) (int, error) {
	return w(data)
}

func webInteractiveLeadEnabled(sess registry.RuntimeSession) bool {
	return strings.EqualFold(strings.TrimSpace(sess.Labels["nexus.interactive_lead"]), "true")
}

func webInteractiveDelegationMode(sess registry.RuntimeSession) nexus.DelegationMode {
	mode := nexus.DelegationOff
	if configured := strings.TrimSpace(sess.Labels["nexus.delegation_mode"]); configured != "" {
		mode = nexus.DelegationMode(configured)
	}
	if override := strings.TrimSpace(os.Getenv("NEXUS_DELEGATION_MODE")); override != "" {
		mode = nexus.DelegationMode(override)
	}
	return nexus.NormalizeInteractiveDelegationMode(mode)
}

func clearLineSequence(providerID string) string {
	if provider, ok := providerspec.Get(providerID); ok && provider.ClearLine != "" {
		return provider.ClearLine
	}
	return "\x15"
}

// writeAttachedTerminalInput mirrors the CLI attach path: bytes typed in
// xterm go directly to the attached PTY. CmdInput RPC envelopes on the same
// stream can interfere with the Windows pipe shell/ConPTY input and produce
// an invisible prompt even though the WebSocket remains connected.
// The SessionHost remains the single-writer and input-normalization authority.
func writeAttachedTerminalInput(conn interface{ Write([]byte) (int, error) }, data string) error {
	if data == "" {
		return nil
	}
	written, err := conn.Write([]byte(data))
	if err == nil && written != len(data) {
		return io.ErrShortWrite
	}
	return err
}

func sendAttachedCommand(conn interface{ Write([]byte) (int, error) }, command protocol.CommandType, payload any) error {
	req, err := protocol.NewRequest(command, payload)
	if err != nil {
		return err
	}
	data, err := json.Marshal(req)
	if err != nil {
		return err
	}
	_, err = conn.Write(append(data, '\n'))
	return err
}

func sendAttachedCommandWithRetry(conn interface{ Write([]byte) (int, error) }, _ int) error {
	const attempts = 3
	var err error
	for i := 0; i < attempts; i++ {
		err = sendAttachedCommand(conn, protocol.CmdLeaseAcquire, nil)
		if err == nil {
			return nil
		}
		if i+1 < attempts {
			time.Sleep(40 * time.Millisecond)
		}
	}
	return err
}

// MaxProtocolPending bounds incomplete JSON held while waiting for a newline
// before treating the bytes as terminal output.
const MaxProtocolPending = 64 * 1024

// isProtocolControlFrame reports whether data is a Nexus Control RPC request
// or response that must not be shown in the agent terminal.
func isProtocolControlFrame(data []byte) bool {
	return protocol.IsWireControlFrame(data)
}

// scrubProtocolFramesFromOutput removes complete Nexus Control RPC lines from
// a PTY output blob (ring-buffer replay or mixed chunks). Non-protocol text is
// preserved byte-for-byte, including newlines.
func scrubProtocolFramesFromOutput(data string) string {
	if data == "" {
		return ""
	}
	raw := []byte(data)
	var out bytes.Buffer
	for len(raw) > 0 {
		idx := bytes.IndexByte(raw, '\n')
		if idx < 0 {
			if !isProtocolControlFrame(raw) {
				out.Write(raw)
			}
			break
		}
		line := raw[:idx+1]
		raw = raw[idx+1:]
		if isProtocolControlFrame(line) {
			continue
		}
		out.Write(line)
	}
	return out.String()
}
