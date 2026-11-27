package host

// IPC transport for the SessionHost: listener lifecycle, client accept loop,
// bounded line framing, attached-input streaming and the JSON-RPC command
// switch. Everything here runs against the same SessionHost state and the same
// locking discipline as host.go; no protocol, payload or route changed when the
// file was split.

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/events"
	"github.com/IAPro-Community/nexus/internal/control/protocol"
	"github.com/IAPro-Community/nexus/internal/control/registry"
	"github.com/IAPro-Community/nexus/internal/core/security"
	"github.com/IAPro-Community/nexus/internal/profile"
	nexusruntime "github.com/IAPro-Community/nexus/internal/runtime"
)

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
	var auditEvents []events.Event
	var newAttachment bool
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
		if !sh.clients[conn] {
			sh.clients[conn] = true
			newAttachment = true
			sh.bindTerminalClientIdentityLocked(conn, req.Payload)
			clientID := sh.terminalClientIDLocked(conn)
			if sh.activeWriter == nil {
				sh.activeWriter = conn
				auditEvents = append(auditEvents, sh.terminalLeaseEventLocked(events.EventTerminalLeaseGranted, clientID, "initial_attach"))
			}
		}
		history := sh.ringBuffer.Bytes()
		resp, _ = protocol.NewResponse(string(history))

	case protocol.CmdDetach:
		clientID := sh.clientIDs[conn]
		if sh.clients[conn] {
			delete(sh.clients, conn)
			sh.fanout.RemoveClient(conn)
		}
		if sh.activeWriter == conn {
			sh.activeWriter = nil
			auditEvents = append(auditEvents, sh.terminalLeaseEventLocked(events.EventTerminalLeaseReleased, clientID, "detach"))
		}
		delete(sh.clientIDs, conn)
		delete(sh.clientActors, conn)
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
				var auditEvent *events.Event
				forwardWrite, auditEvent = sh.collectAttachedInputLocked(conn, []byte(p.Data))
				if auditEvent != nil {
					auditEvents = append(auditEvents, *auditEvent)
				}
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

	case protocol.CmdChooseInteraction:
		var p protocol.ChooseInteractionPayload
		if req.Payload == nil || json.Unmarshal(req.Payload, &p) != nil || p.InteractionID == "" || p.OptionID == "" {
			resp = protocol.NewErrorResponse("INTERACTION_AND_OPTION_REQUIRED")
			break
		}
		chooser, ok := sh.termBackend.(interface {
			ChooseInteraction(string, string) error
		})
		if !ok {
			resp = protocol.NewErrorResponse("STRUCTURED_INTERACTION_UNAVAILABLE")
			break
		}
		if err := chooser.ChooseInteraction(p.InteractionID, p.OptionID); err != nil {
			resp = protocol.NewErrorResponse("INTERACTION_CHOICE_REJECTED")
			break
		}
		resp = protocol.NewControlResponse(protocol.ControlResult{
			OK: true, Code: "INTERACTION_CHOICE_SUBMITTED", RuntimeID: sh.session.RuntimeID,
			Action: string(protocol.CmdChooseInteraction), State: string(sh.session.State),
			Message: "provider choice submitted",
		})

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
		previousWriter := sh.activeWriter
		granted, reason := sh.applyTerminalLeaseLocked(conn, req.Payload)
		if !granted {
			resp = protocol.NewErrorResponse(reason)
			break
		}
		clientID := sh.terminalClientIDLocked(conn)
		if previousWriter != conn {
			if previousWriter != nil {
				previousClientID := sh.terminalClientIDLocked(previousWriter)
				auditEvents = append(auditEvents, sh.terminalLeaseEventLocked(events.EventTerminalLeaseRevoked, previousClientID, "takeover"))
			}
			auditEvents = append(auditEvents, sh.terminalLeaseEventLocked(events.EventTerminalLeaseGranted, clientID, "explicit_takeover"))
		}
		resp, _ = protocol.NewResponse(reason)

	case protocol.CmdLeaseRelease:
		if sh.activeWriter == conn {
			sh.activeWriter = nil
			auditEvents = append(auditEvents, sh.terminalLeaseEventLocked(events.EventTerminalLeaseReleased, sh.terminalClientIDLocked(conn), "explicit_release"))
		}
		resp, _ = protocol.NewResponse("lease_released")

	default:
		resp = protocol.NewErrorResponse(fmt.Sprintf("unknown command %q", req.Command))
	}
	if resp.CorrelationID == "" {
		resp.CorrelationID = req.ID
	}
	if newAttachment {
		// Queue the attach acknowledgment before the connection can receive
		// raw PTY output. The client still expects a JSON frame at this point.
		data, _ := json.Marshal(resp)
		sh.fanout.AddClient(conn, append(data, '\n'))
		skipResponse = true
	}

	_, isAttached := sh.clients[conn]
	if isAttached && req.Command == protocol.CmdResize {
		skipResponse = true
	}
	sh.mu.Unlock()
	sh.publishTerminalAuditEvents(auditEvents)

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
