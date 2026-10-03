package app

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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

type remoteBridgeServer struct {
	launcher      remoteBridgeLaunchPort
	machineID     func() string
	remoteEnabled func() bool
	nexusVersion  string
	osName        string
	arch          string
	resize        func(string, int, int) error
	lookup        func(string) (registry.RuntimeSession, bool)
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
	}
}

func remoteBridgeCmd(args []string) error {
	if len(args) != 1 || args[0] != "--stdio" {
		return fmt.Errorf("usage: %s __remote-bridge --stdio", progName())
	}
	server := newRemoteBridgeServer()
	return server.serve(context.Background(), os.Stdin, os.Stdout)
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
