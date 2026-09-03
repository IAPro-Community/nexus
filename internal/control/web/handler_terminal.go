package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/kivervinicius/ai-cli/internal/control/events"
	"github.com/kivervinicius/ai-cli/internal/control/protocol"
	"github.com/kivervinicius/ai-cli/internal/control/registry"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: CheckOrigin,
}

type TerminalMessage struct {
	Type string `json:"type"` // "output", "input", "resize", "lease", "error"
	Data string `json:"data,omitempty"`
	Rows int    `json:"rows,omitempty"`
	Cols int    `json:"cols,omitempty"`
	Role string `json:"role,omitempty"` // "CONTROL" or "VIEW_ONLY"
}

type TerminalHub struct {
	auth *AuthManager
}

// resolveProjectIDForAgent is intentionally best-effort. Terminal attention
// frames must carry verified routing metadata when available, but the
// transport must not invent a project or expose provider/session contents.
var resolveProjectIDForAgent = func(agentID string) string { return "" }

func attentionFrameFromEvent(agentID string, ev events.Event) map[string]any {
	frame := map[string]any{
		"type":               "attention",
		"runtime_id":         ev.RuntimeID,
		"attention_event_id": ev.ID,
		"attention_reason":   ev.Data["attention_reason"],
		"context":            ev.Data["context"],
		"project_name":       ev.Data["project_name"],
		"dynamic_title":      ev.Data["dynamic_title"],
		"summary":            ev.Summary,
		"view":               "work",
	}
	if projectID, ok := ev.Data["project_id"].(string); ok && strings.TrimSpace(projectID) != "" {
		frame["project_id"] = strings.TrimSpace(projectID)
	} else if projectID := strings.TrimSpace(resolveProjectIDForAgent(agentID)); projectID != "" {
		frame["project_id"] = projectID
	}
	if view, ok := ev.Data["view"].(string); ok && strings.TrimSpace(view) != "" {
		frame["view"] = strings.TrimSpace(view)
	}
	return frame
}

func NewTerminalHub(auth *AuthManager) *TerminalHub {
	return &TerminalHub{
		auth: auth,
	}
}

func (h *TerminalHub) HandleWebSocket(w http.ResponseWriter, r *http.Request, agentID string, runtimeID string) {
	if h.auth != nil && h.auth.AuthenticateRequest(r) == nil {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}

	reg := registry.DefaultRegistry()
	_, exists := reg.Get(runtimeID)
	if !exists {
		http.Error(w, "runtime not found", http.StatusNotFound)
		return
	}

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer ws.Close()

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

	// Connect to runtime SessionHost via local IPC
	client, err := protocol.NewClient(runtimeID)
	if err != nil {
		_ = reg.UpdateState(runtimeID, registry.StateStopped)
		_ = safeWriteJSON(TerminalMessage{
			Type: "error",
			Data: "Runtime host is not running (" + err.Error() + "). The process has exited or the socket was closed.",
		})
		return
	}
	defer client.Close()

	// Attach to host
	resp, err := client.Send(protocol.CmdAttach, nil)
	if err != nil {
		_ = reg.UpdateState(runtimeID, registry.StateStopped)
		_ = safeWriteJSON(TerminalMessage{
			Type: "error",
			Data: "Failed to attach: runtime host is no longer responding (" + err.Error() + ").",
		})
		return
	}
	_ = client.ClearDeadline()

	rawConn := client.RawConn()
	stopChan := make(chan struct{})

	// Keep the SessionHost writer authority aligned with the Agent-level
	// broker. The command is deliberately sent over this connection *after*
	// Attach so SessionHost can bind the lease to the exact streaming client,
	// not to an unrelated RPC connection.
	effectiveRole := role
	if role == "CONTROL" {
		if err := sendAttachedCommand(rawConn, protocol.CmdLeaseAcquire, nil); err != nil {
			effectiveRole = "VIEW_ONLY"
		}
	}
	_ = safeWriteJSON(TerminalMessage{
		Type: "lease",
		Role: effectiveRole,
	})

	if ch := DefaultBroker().WatchRuntimeChanged(agentID); ch != nil {
		go func() {
			<-ch
			_ = ws.Close()
		}()
	}

	// Initial ring buffer history
	var history string
	if json.Unmarshal(resp.Data, &history) == nil && history != "" {
		_ = safeWriteJSON(TerminalMessage{
			Type: "output",
			Data: history,
		})
	}

	// Send initial dynamic title and attention if available
	if sess, ok := reg.Get(runtimeID); ok {
		if sess.DynamicTitle != "" {
			_ = safeWriteJSON(TerminalMessage{
				Type: "title",
				Data: sess.DynamicTitle,
			})
		}
		if sess.AttentionReason != "" && sess.AttentionContext != "" {
			_ = safeWriteJSON(map[string]any{
				"type":             "attention",
				"runtime_id":       runtimeID,
				"attention_reason": sess.AttentionReason,
				"context":          sess.AttentionContext,
				"project_name":     sess.ProjectName,
				"dynamic_title":    sess.DynamicTitle,
				"view":             "work",
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
					_ = safeWriteJSON(attentionFrameFromEvent(agentID, ev))
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

	// 1. Pump stdout from child process to browser
	go func() {
		buf := make([]byte, 2048)
		for {
			select {
			case <-stopChan:
				return
			default:
				_ = rawConn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
				n, err := rawConn.Read(buf)
				if n > 0 {
					chunk := buf[:n]
					trimmed := bytes.TrimSpace(chunk)
					if (bytes.HasPrefix(trimmed, []byte("{\"version\":")) && bytes.Contains(trimmed, []byte("\"ok\":"))) || bytes.HasPrefix(trimmed, []byte("{\"ok\":")) {
						// Skip RPC response frame from being displayed in terminal
						continue
					}
					_ = safeWriteJSON(TerminalMessage{
						Type: "output",
						Data: string(chunk),
					})
				}
				if err != nil && !strings.Contains(err.Error(), "timeout") {
					return
				}
			}
		}
	}()

	// 2. Read WebSocket frames from browser
	for {
		var msg TerminalMessage
		if err := ws.ReadJSON(&msg); err != nil {
			close(stopChan)
			return
		}

		switch msg.Type {
		case "input":
			// Authoritative writer check on Agent level (§45, A6)
			if broker.IsWriter(agentID, ws) && msg.Data != "" {
				_, _ = rawConn.Write([]byte(msg.Data))
			}

		case "resize":
			if msg.Rows > 0 && msg.Cols > 0 {
				go func(r, c int) {
					rpcClient, err := protocol.NewClient(runtimeID)
					if err == nil {
						_ = rpcClient.Resize(r, c)
						_ = rpcClient.Close()
					}
				}(msg.Rows, msg.Cols)
			}

		case "lease_acquire":
			if broker.TakeControl(agentID, ws) {
				_ = safeWriteJSON(TerminalMessage{
					Type: "lease",
					Role: "CONTROL",
				})
				_ = sendAttachedCommand(rawConn, protocol.CmdLeaseAcquire, nil)
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
