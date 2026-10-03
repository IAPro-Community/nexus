package app

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/buildinfo"
	"github.com/IAPro-Community/nexus/internal/control/launcher"
	"github.com/IAPro-Community/nexus/internal/control/protocol"
	"github.com/IAPro-Community/nexus/internal/control/registry"
	coreflags "github.com/IAPro-Community/nexus/internal/core/flags"
	"github.com/IAPro-Community/nexus/internal/core/security"
)

const remoteBridgeMaxFrameBytes = 256 * 1024

type remoteBridgeLaunchPort interface {
	Launch(context.Context, launcher.LaunchOptions) (*registry.RuntimeSession, error)
}

type remoteBridgeRuntimeClient interface {
	Status() (protocol.StatusData, error)
	Resize(rows, cols int) error
	Send(protocol.CommandType, any) (protocol.Response, error)
	ClearDeadline() error
	RawConn() net.Conn
	Reader() *bufio.Reader
	Close() error
}

type remoteBridgeServer struct {
	launcher      remoteBridgeLaunchPort
	machineID     func() string
	remoteEnabled func() bool
	nexusVersion  string
	osName        string
	arch          string
	resize        func(string, int, int) error
	lookup        func(string) (registry.RuntimeSession, bool)
	dialRuntime   func(string) (remoteBridgeRuntimeClient, error)
}

func newRemoteBridgeServer() remoteBridgeServer {
	return remoteBridgeServer{
		launcher:      launcher.Default(),
		machineID:     registry.LocalMachineID,
		remoteEnabled: func() bool { return coreflags.Enabled(coreflags.RemoteExecution) },
		nexusVersion:  buildinfo.Version,
		osName:        runtime.GOOS,
		arch:          runtime.GOARCH,
		resize:        resizeRemoteBridgeRuntime,
		lookup:        registry.DefaultRegistry().Get,
		dialRuntime: func(runtimeID string) (remoteBridgeRuntimeClient, error) {
			return protocol.NewClient(runtimeID)
		},
	}
}

func remoteBridgeCmd(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: %s __remote-bridge <--stdio|--attach-stdio>", progName())
	}
	server := newRemoteBridgeServer()
	switch args[0] {
	case "--stdio":
		return server.serve(context.Background(), os.Stdin, os.Stdout)
	case "--attach-stdio":
		return server.serveAttach(context.Background(), os.Stdin, os.Stdout)
	default:
		return fmt.Errorf("usage: %s __remote-bridge <--stdio|--attach-stdio>", progName())
	}
}

func (s remoteBridgeServer) serve(ctx context.Context, in io.Reader, out io.Writer) error {
	if in == nil || out == nil {
		return fmt.Errorf("remote bridge stdio is required")
	}
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 64*1024), remoteBridgeMaxFrameBytes)
	encoder := json.NewEncoder(out)
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var request protocol.Request
		if err := decodeStrictRemoteBridgeJSON(line, &request); err != nil {
			if encodeErr := encoder.Encode(remoteBridgeError("", "INVALID_REQUEST", err)); encodeErr != nil {
				return encodeErr
			}
			continue
		}
		response := s.handle(ctx, request)
		if err := encoder.Encode(response); err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("remote bridge input failed: %w", err)
	}
	return nil
}

func (s remoteBridgeServer) handle(ctx context.Context, request protocol.Request) protocol.Response {
	if request.Version != protocol.ProtocolVersion {
		return remoteBridgeError(request.ID, "UNSUPPORTED_PROTOCOL", fmt.Errorf("protocol version %d is unsupported", request.Version))
	}

	switch request.Command {
	case protocol.CmdRemoteBridgeHello:
		var payload protocol.RemoteBridgeHelloPayload
		if err := decodeStrictRemoteBridgeJSON(request.Payload, &payload); err != nil {
			return remoteBridgeError(request.ID, "INVALID_REQUEST", err)
		}
		if err := protocol.ValidateRemoteBridgeHello(payload); err != nil {
			return remoteBridgeError(request.ID, "INVALID_REQUEST", err)
		}
		data := protocol.RemoteBridgeHelloData{
			ProtocolVersion:        protocol.ProtocolVersion,
			MachineID:              s.machineID(),
			OS:                     s.osName,
			Arch:                   s.arch,
			NexusVersion:           s.nexusVersion,
			RemoteExecutionEnabled: s.remoteEnabled(),
			Capabilities: []protocol.RemoteBridgeCapability{
				protocol.RemoteBridgeCapabilitySpawn,
				protocol.RemoteBridgeCapabilityCWD,
				protocol.RemoteBridgeCapabilityPTY,
				protocol.RemoteBridgeCapabilityResize,
				protocol.RemoteBridgeCapabilityLiveness,
			},
		}
		if err := protocol.ValidateRemoteBridgeHelloData(data); err != nil {
			return remoteBridgeError(request.ID, "BRIDGE_NOT_READY", err)
		}
		response, err := protocol.NewResponse(data)
		if err != nil {
			return remoteBridgeError(request.ID, "ENCODE_FAILED", err)
		}
		response.ID = request.ID
		return response

	case protocol.CmdRemoteBridgeStatus:
		if !s.remoteEnabled() {
			return remoteBridgeError(request.ID, "REMOTE_EXECUTION_DISABLED", fmt.Errorf("remote execution is disabled on this host"))
		}
		var payload protocol.RemoteBridgeRuntimePayload
		if err := decodeStrictRemoteBridgeJSON(request.Payload, &payload); err != nil {
			return remoteBridgeError(request.ID, "INVALID_REQUEST", err)
		}
		if err := protocol.ValidateRemoteBridgeRuntime(payload); err != nil {
			return remoteBridgeError(request.ID, "INVALID_REQUEST", err)
		}
		session, client, err := s.openVerifiedRuntime(payload.RuntimeID)
		if err != nil {
			return remoteBridgeError(request.ID, "RUNTIME_UNAVAILABLE", err)
		}
		defer client.Close()
		status, err := client.Status()
		if err != nil {
			return remoteBridgeError(request.ID, "STATUS_FAILED", err)
		}
		response, err := protocol.NewResponse(protocol.RemoteBridgeRuntimeStatusData{
			RuntimeID:    session.RuntimeID,
			MachineID:    firstNonEmptyBridgeValue(session.MachineID, s.machineID()),
			Provider:     session.ProviderID,
			Profile:      session.ProfileID,
			Model:        session.Model,
			Workspace:    session.Workspace,
			State:        status.State,
			ControlLevel: status.ControlLevel,
			Transport:    "nexus-remote-bridge",
		})
		if err != nil {
			return remoteBridgeError(request.ID, "ENCODE_FAILED", err)
		}
		response.ID = request.ID
		response.RuntimeID = session.RuntimeID
		return response

	case protocol.CmdRemoteBridgeResize:
		if !s.remoteEnabled() {
			return remoteBridgeError(request.ID, "REMOTE_EXECUTION_DISABLED", fmt.Errorf("remote execution is disabled on this host"))
		}
		var payload protocol.RemoteBridgeResizePayload
		if err := decodeStrictRemoteBridgeJSON(request.Payload, &payload); err != nil {
			return remoteBridgeError(request.ID, "INVALID_REQUEST", err)
		}
		if err := protocol.ValidateRemoteBridgeResize(payload); err != nil {
			return remoteBridgeError(request.ID, "INVALID_REQUEST", err)
		}
		session, client, err := s.openVerifiedRuntime(payload.RuntimeID)
		if err != nil {
			return remoteBridgeError(request.ID, "RUNTIME_UNAVAILABLE", err)
		}
		defer client.Close()
		if err := client.Resize(payload.Rows, payload.Cols); err != nil {
			return remoteBridgeError(request.ID, "RESIZE_FAILED", err)
		}
		response, err := protocol.NewResponse(map[string]any{
			"runtime_id": session.RuntimeID,
			"rows":       payload.Rows,
			"cols":       payload.Cols,
		})
		if err != nil {
			return remoteBridgeError(request.ID, "ENCODE_FAILED", err)
		}
		response.ID = request.ID
		response.RuntimeID = session.RuntimeID
		return response

	case protocol.CmdRemoteBridgeControl:
		if !s.remoteEnabled() {
			return remoteBridgeError(request.ID, "REMOTE_EXECUTION_DISABLED", fmt.Errorf("remote execution is disabled on this host"))
		}
		var payload protocol.RemoteBridgeControlPayload
		if err := decodeStrictRemoteBridgeJSON(request.Payload, &payload); err != nil {
			return remoteBridgeError(request.ID, "INVALID_REQUEST", err)
		}
		if err := protocol.ValidateRemoteBridgeControl(payload); err != nil {
			return remoteBridgeError(request.ID, "INVALID_REQUEST", err)
		}
		_, client, err := s.openVerifiedRuntime(payload.RuntimeID)
		if err != nil {
			return remoteBridgeError(request.ID, "RUNTIME_UNAVAILABLE", err)
		}
		defer client.Close()
		response, err := client.Send(payload.Command, json.RawMessage(payload.Payload))
		if err != nil {
			return remoteBridgeError(request.ID, "CONTROL_FAILED", err)
		}
		response.ID = request.ID
		response.RuntimeID = payload.RuntimeID
		return response

	case protocol.CmdRemoteBridgeSpawn:
		if !s.remoteEnabled() {
			return remoteBridgeError(request.ID, "REMOTE_EXECUTION_DISABLED", fmt.Errorf("remote execution is disabled on this host"))
		}
		var payload protocol.RemoteBridgeSpawnPayload
		if err := decodeStrictRemoteBridgeJSON(request.Payload, &payload); err != nil {
			return remoteBridgeError(request.ID, "INVALID_REQUEST", err)
		}
		if err := protocol.ValidateRemoteBridgeSpawn(payload); err != nil {
			return remoteBridgeError(request.ID, "INVALID_REQUEST", err)
		}

		if s.lookup != nil {
			if existing, ok := s.lookup(payload.RuntimeID); ok {
				if err := validateRemoteBridgeExistingRuntime(existing, payload); err != nil {
					return remoteBridgeError(request.ID, "RUNTIME_ID_CONFLICT", err)
				}
				if existing.IsActive() && existing.HostLive() {
					return remoteBridgeSpawnResponse(request.ID, existing, firstNonEmptyBridgeValue(existing.MachineID, s.machineID()))
				}
				return remoteBridgeError(request.ID, "RUNTIME_ID_CONFLICT", fmt.Errorf("runtime %q already exists but is not live", payload.RuntimeID))
			}
		}

		session, err := s.launcher.Launch(ctx, launcher.LaunchOptions{
			RuntimeID:  payload.RuntimeID,
			AgentID:    payload.AgentID,
			ProjectID:  payload.ProjectID,
			ProviderID: payload.Provider,
			ProfileID:  payload.Profile,
			Model:      payload.Model,
			Workspace:  payload.Workspace,
			Args:       append([]string(nil), payload.Args...),
			Timeout:    15 * time.Second,
		})
		if err != nil {
			return remoteBridgeError(request.ID, "LAUNCH_FAILED", err)
		}
		if session == nil {
			return remoteBridgeError(request.ID, "LAUNCH_FAILED", fmt.Errorf("remote launcher returned no runtime session"))
		}
		if payload.Rows > 0 && payload.Cols > 0 && s.resize != nil {
			_ = s.resize(session.RuntimeID, payload.Rows, payload.Cols)
		}

		return remoteBridgeSpawnResponse(
			request.ID,
			*session,
			firstNonEmptyBridgeValue(session.MachineID, s.machineID()),
		)

	default:
		return remoteBridgeError(request.ID, "UNSUPPORTED_COMMAND", fmt.Errorf("remote bridge command %q is unsupported", request.Command))
	}
}

func (s remoteBridgeServer) openVerifiedRuntime(runtimeID string) (registry.RuntimeSession, remoteBridgeRuntimeClient, error) {
	if s.lookup == nil || s.dialRuntime == nil {
		return registry.RuntimeSession{}, nil, fmt.Errorf("remote bridge runtime access is unavailable")
	}
	session, ok := s.lookup(runtimeID)
	if !ok {
		return registry.RuntimeSession{}, nil, fmt.Errorf("runtime %q is not registered", runtimeID)
	}
	if !session.IsActive() || !session.HostLive() {
		return registry.RuntimeSession{}, nil, fmt.Errorf("runtime %q is not live", runtimeID)
	}
	client, err := s.dialRuntime(runtimeID)
	if err != nil {
		return registry.RuntimeSession{}, nil, err
	}
	status, err := client.Status()
	if err != nil {
		_ = client.Close()
		return registry.RuntimeSession{}, nil, err
	}
	if err := verifyRemoteBridgeRuntimeIdentity(session, status); err != nil {
		_ = client.Close()
		return registry.RuntimeSession{}, nil, err
	}
	return session, client, nil
}

func verifyRemoteBridgeRuntimeIdentity(session registry.RuntimeSession, status protocol.StatusData) error {
	if status.RuntimeID != session.RuntimeID {
		return fmt.Errorf("runtime identity changed")
	}
	if session.PID > 0 && status.PID != session.PID {
		return fmt.Errorf("provider PID changed for runtime %q", session.RuntimeID)
	}
	if session.ProviderGeneration > 0 && status.ProviderGeneration > 0 && status.ProviderGeneration != session.ProviderGeneration {
		return fmt.Errorf("provider generation changed for runtime %q", session.RuntimeID)
	}
	if session.HostPID > 0 && status.HostPID != session.HostPID {
		return fmt.Errorf("SessionHost PID changed for runtime %q", session.RuntimeID)
	}
	if session.HostGeneration > 0 && status.HostGeneration > 0 && status.HostGeneration != session.HostGeneration {
		return fmt.Errorf("SessionHost generation changed for runtime %q", session.RuntimeID)
	}
	return nil
}

func (s remoteBridgeServer) serveAttach(ctx context.Context, in io.Reader, out io.Writer) error {
	if !s.remoteEnabled() {
		response := remoteBridgeError("", "REMOTE_EXECUTION_DISABLED", fmt.Errorf("remote execution is disabled on this host"))
		return json.NewEncoder(out).Encode(response)
	}
	reader := bufio.NewReaderSize(in, 64*1024)
	frame, err := readRemoteBridgeFrame(reader, remoteBridgeMaxFrameBytes)
	if err != nil {
		response := remoteBridgeError("", "INVALID_REQUEST", err)
		_ = json.NewEncoder(out).Encode(response)
		return err
	}
	var request protocol.Request
	if err := decodeStrictRemoteBridgeJSON(frame, &request); err != nil {
		response := remoteBridgeError("", "INVALID_REQUEST", err)
		_ = json.NewEncoder(out).Encode(response)
		return err
	}
	if request.Version != protocol.ProtocolVersion || request.Command != protocol.CmdRemoteBridgeAttach {
		err := fmt.Errorf("remote bridge attach request is invalid")
		response := remoteBridgeError(request.ID, "INVALID_REQUEST", err)
		_ = json.NewEncoder(out).Encode(response)
		return err
	}
	var payload protocol.RemoteBridgeRuntimePayload
	if err := decodeStrictRemoteBridgeJSON(request.Payload, &payload); err != nil {
		response := remoteBridgeError(request.ID, "INVALID_REQUEST", err)
		_ = json.NewEncoder(out).Encode(response)
		return err
	}
	if err := protocol.ValidateRemoteBridgeRuntime(payload); err != nil {
		response := remoteBridgeError(request.ID, "INVALID_REQUEST", err)
		_ = json.NewEncoder(out).Encode(response)
		return err
	}

	session, client, err := s.openVerifiedRuntime(payload.RuntimeID)
	if err != nil {
		response := remoteBridgeError(request.ID, "RUNTIME_UNAVAILABLE", err)
		_ = json.NewEncoder(out).Encode(response)
		return err
	}
	defer client.Close()

	attachResponse, err := client.Send(protocol.CmdAttach, nil)
	if err != nil {
		response := remoteBridgeError(request.ID, "ATTACH_FAILED", err)
		_ = json.NewEncoder(out).Encode(response)
		return err
	}
	var history string
	if len(attachResponse.Data) > 0 {
		if err := json.Unmarshal(attachResponse.Data, &history); err != nil {
			response := remoteBridgeError(request.ID, "ATTACH_FAILED", fmt.Errorf("invalid terminal history response: %w", err))
			_ = json.NewEncoder(out).Encode(response)
			return err
		}
	}
	if err := client.ClearDeadline(); err != nil {
		response := remoteBridgeError(request.ID, "ATTACH_FAILED", err)
		_ = json.NewEncoder(out).Encode(response)
		return err
	}
	ack, err := protocol.NewResponse(protocol.RemoteBridgeAttachData{
		RuntimeID: session.RuntimeID,
		History:   history,
	})
	if err != nil {
		return err
	}
	ack.ID = request.ID
	ack.RuntimeID = session.RuntimeID
	if err := json.NewEncoder(out).Encode(ack); err != nil {
		return err
	}

	rawConn := client.RawConn()
	if rawConn == nil || client.Reader() == nil {
		return fmt.Errorf("runtime %q attach stream is unavailable", session.RuntimeID)
	}
	cancelClose := context.AfterFunc(ctx, func() { _ = rawConn.Close() })
	defer cancelClose()

	errCh := make(chan error, 2)
	go func() {
		_, copyErr := io.Copy(out, client.Reader())
		errCh <- copyErr
	}()
	go func() {
		_, copyErr := io.Copy(rawConn, reader)
		errCh <- copyErr
	}()

	firstErr := <-errCh
	_ = rawConn.Close()
	secondErr := <-errCh
	if firstErr != nil && !errors.Is(firstErr, net.ErrClosed) && !errors.Is(firstErr, io.EOF) {
		return firstErr
	}
	if secondErr != nil && !errors.Is(secondErr, net.ErrClosed) && !errors.Is(secondErr, io.EOF) {
		return secondErr
	}
	return nil
}

func readRemoteBridgeFrame(reader *bufio.Reader, limit int) ([]byte, error) {
	if reader == nil {
		return nil, fmt.Errorf("remote bridge input is required")
	}
	var frame []byte
	for {
		part, isPrefix, err := reader.ReadLine()
		frame = append(frame, part...)
		if len(frame) > limit {
			return nil, fmt.Errorf("remote bridge frame exceeds %d bytes", limit)
		}
		if err != nil {
			return nil, err
		}
		if !isPrefix {
			return frame, nil
		}
	}
}

func validateRemoteBridgeExistingRuntime(existing registry.RuntimeSession, payload protocol.RemoteBridgeSpawnPayload) error {
	checks := []struct {
		name     string
		existing string
		requested string
	}{
		{name: "agent_id", existing: existing.AgentID, requested: payload.AgentID},
		{name: "project_id", existing: existing.ProjectID, requested: payload.ProjectID},
		{name: "provider", existing: existing.ProviderID, requested: payload.Provider},
		{name: "profile", existing: existing.ProfileID, requested: payload.Profile},
		{name: "workspace", existing: existing.Workspace, requested: payload.Workspace},
		{name: "model", existing: existing.Model, requested: payload.Model},
	}
	for _, check := range checks {
		if strings.TrimSpace(check.existing) != strings.TrimSpace(check.requested) {
			return fmt.Errorf("runtime %q already exists with different %s", payload.RuntimeID, check.name)
		}
	}
	return nil
}

func remoteBridgeSpawnResponse(requestID string, session registry.RuntimeSession, machineID string) protocol.Response {
	response, err := protocol.NewResponse(protocol.RemoteBridgeSpawnData{
		RuntimeID:   session.RuntimeID,
		MachineID:   machineID,
		HostPID:     session.HostPID,
		ProviderPID: session.PID,
		Transport:   "nexus-remote-bridge",
	})
	if err != nil {
		return remoteBridgeError(requestID, "ENCODE_FAILED", err)
	}
	response.ID = requestID
	response.RuntimeID = session.RuntimeID
	return response
}

func resizeRemoteBridgeRuntime(runtimeID string, rows, cols int) error {
	client, err := protocol.NewClient(runtimeID)
	if err != nil {
		return err
	}
	defer client.Close()
	return client.Resize(rows, cols)
}

func remoteBridgeError(requestID, code string, err error) protocol.Response {
	message := code
	if err != nil {
		message = security.Redact(err.Error())
	}
	return protocol.Response{
		Version:   protocol.ProtocolVersion,
		ID:        requestID,
		OK:        false,
		Code:      code,
		Message:   message,
		Error:     message,
		Timestamp: time.Now().UTC(),
	}
}

func decodeStrictRemoteBridgeJSON(data []byte, out any) error {
	if len(bytes.TrimSpace(data)) == 0 {
		return fmt.Errorf("remote bridge payload is required")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return fmt.Errorf("remote bridge payload contains trailing JSON")
		}
		return err
	}
	return nil
}

func firstNonEmptyBridgeValue(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
