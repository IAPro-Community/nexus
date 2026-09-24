package web

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: CheckOrigin,
}

type TerminalMessage struct {
	Type string `json:"type"` // "output", "input", "resize", "lease", "promotion", "error"
	Data string `json:"data,omitempty"`
	Rows int    `json:"rows,omitempty"`
	Cols int    `json:"cols,omitempty"`
	Role string `json:"role,omitempty"` // "CONTROL" or "VIEW_ONLY"
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

func (h *TerminalHub) HandleWebSocket(w http.ResponseWriter, r *http.Request, agentID string, runtimeID string) {
	if h.auth != nil && h.auth.AuthenticateRequest(r) == nil {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}

	reg := h.registry
	sess, exists := reg.Get(runtimeID)
	if !exists {
		writeError(w, http.StatusNotFound, "runtime not found")
		return
	}
	if !sess.HostLive() {
		// Do not upgrade: a "connected" WebSocket into a dead host leaves the
		// UI on a black xterm with no recover action (especially window chrome).
		writeError(w, http.StatusNotFound, "runtime host is not running")
		return
	}
	if err := registry.ValidateAttachCompatibility(sess); err != nil {
		// Preserve the process for explicit user cleanup, but make the runtime
		// non-attachable and actionable instead of opening a silent black TTY.
		_ = reg.UpdateState(runtimeID, registry.StateStale)
		writeError(w, http.StatusConflict, "runtime is incompatible with the current Nexus binary; recreate the session with nexus codex --yolo")
		return
	}

	// Dial the control endpoint before upgrading. A live PID/STARTING row is
	// not enough on Windows when the named pipe was never created; prefer HTTP
	// 404 over a WebSocket error overlay ("Runtime do Agente desconectado").
	client, err := protocol.NewClient(runtimeID)
	if err != nil {
		writeError(w, http.StatusNotFound, "runtime host is not running ("+err.Error()+")")
		return
	}

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		_ = client.Close()
		return
	}
	defer ws.Close()
	defer client.Close()

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
		_ = sendAttachedCommandWithRetry(rawConn, 3)
	}
	_ = safeWriteJSON(TerminalMessage{
		Type: "lease",
		Role: effectiveRole,
	})

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
	if strings.EqualFold(string(webInteractiveDelegationMode(sess)), string(nexus.DelegationOff)) {
		shadow.SetEnabled(false)
	}
	startPromotion := func(prompt string, fallbackToLead bool) {
		accepted, startErr := startPromotedMission(h.nexus, sess, prompt)
		if startErr != nil {
			if fallbackToLead {
				// The held prompt is still in the provider TUI. Submit it
				// unchanged to the Lead instead of consuming it into a failed
				// heuristic Mission promotion.
				_ = sendAttachedCommand(rawConn, protocol.CmdInput, protocol.InputPayload{Data: "\r"})
				_ = safeWriteJSON(TerminalMessage{
					Type: "promotion",
					Role: "FALLBACK",
					Data: fmt.Sprintf("Mission não admitida (%v); continuando com o Lead.", startErr),
				})
				return
			}
			// Explicit/confirmed Mission requests never silently degrade to
			// Direct. Clear the held input and show the exact admission blocker.
			_ = sendAttachedCommand(rawConn, protocol.CmdInput, protocol.InputPayload{Data: clearLineSequence(sess.ProviderID)})
			_ = safeWriteJSON(TerminalMessage{
				Type: "promotion",
				Role: "BLOCKED",
				Data: fmt.Sprintf("Mission solicitada, mas a admissão foi bloqueada: %v", startErr),
			})
			return
		}
		_ = sendAttachedCommand(rawConn, protocol.CmdInput, protocol.InputPayload{Data: clearLineSequence(sess.ProviderID)})
		_ = safeWriteJSON(TerminalMessage{
			Type: "promotion",
			Role: "PROMOTED",
			Data: fmt.Sprintf("Mission %s iniciada.", accepted.RunID),
		})
		go waitPromotedMission(h.nexus, sess, prompt, accepted)
	}

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
							if len(pending) > 0 && pending[0] != '{' {
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
			if broker.IsWriter(agentID, ws) && pendingPromotion != "" {
				if strings.EqualFold(strings.TrimSpace(msg.Data), "y") || strings.EqualFold(strings.TrimSpace(msg.Data), "yes") {
					startPromotion(pendingPromotion, false)
				} else {
					_ = sendAttachedCommand(rawConn, protocol.CmdInput, protocol.InputPayload{Data: "\r"})
				}
				pendingPromotion = ""
			}

		case "input":
			// Authoritative writer check on Agent level (§45, A6).
			// SessionHost streamAttachedInput is line-oriented for RPC framing;
			// raw keystrokes have no trailing \n, so they must be sent as CmdInput
			// JSON frames (newline-terminated) or they buffer forever and typing
			// appears dead.
			if broker.IsWriter(agentID, ws) && msg.Data != "" {
				if pendingPromotion != "" {
					// ASK consumes the next answer outside the provider PTY. The
					// original line is retained only until this explicit choice.
					answer := strings.TrimSpace(msg.Data)
					if strings.EqualFold(answer, "y") || strings.EqualFold(answer, "yes") {
						startPromotion(pendingPromotion, false)
					} else {
						_ = sendAttachedCommand(rawConn, protocol.CmdInput, protocol.InputPayload{Data: "\r"})
					}
					pendingPromotion = ""
					continue
				}
				if h.nexus == nil || sess.ProjectID == "" || sess.AgentID == "" || !shadow.Enabled() {
					_ = sendAttachedCommand(rawConn, protocol.CmdInput, protocol.InputPayload{Data: msg.Data})
					break
				}
				event := shadow.Consume([]byte(msg.Data))
				if len(event.Forward) > 0 {
					_ = sendAttachedCommand(rawConn, protocol.CmdInput, protocol.InputPayload{Data: string(event.Forward)})
				}
				if event.Submit {
					for _, line := range event.Submissions {
						if promotion.IsImmediateSubmitLine(line) {
							_ = sendAttachedCommand(rawConn, protocol.CmdInput, protocol.InputPayload{Data: "\r"})
							continue
						}
						decideCtx, cancelDecision := context.WithTimeout(r.Context(), promotion.InteractiveDecisionTimeout)
						decision, decideErr := promotion.NewPromoter(h.nexus).Decide(decideCtx, sess.ProjectID, line, webInteractiveDelegationMode(sess))
						cancelDecision()
						if decideErr != nil || decision.Action == promotion.KeepWithLead {
							_ = sendAttachedCommand(rawConn, protocol.CmdInput, protocol.InputPayload{Data: "\r"})
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

func webInteractiveDelegationMode(sess registry.RuntimeSession) nexus.DelegationMode {
	mode := nexus.DelegationMode(sess.Labels["nexus.delegation_mode"])
	if override := strings.TrimSpace(os.Getenv("NEXUS_DELEGATION_MODE")); override != "" {
		mode = nexus.DelegationMode(override)
	}
	return nexus.NormalizeDelegationMode(mode)
}

func clearLineSequence(providerID string) string {
	if provider, ok := providerspec.Get(providerID); ok && provider.ClearLine != "" {
		return provider.ClearLine
	}
	return "\x15"
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
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return false
	}
	var probe map[string]json.RawMessage
	if json.Unmarshal(trimmed, &probe) != nil {
		return false
	}
	if _, hasCommand := probe["command"]; hasCommand {
		return true
	}
	if _, hasOK := probe["ok"]; hasOK {
		return true
	}
	return false
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
