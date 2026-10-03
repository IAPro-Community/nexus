package remote

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/protocol"
	"github.com/IAPro-Community/nexus/internal/control/registry"
	"github.com/IAPro-Community/nexus/internal/core/security"
)

const managedProxyMaxFrameBytes = 64 * 1024

type ManagedBridgeControl interface {
	Status(context.Context, string) (protocol.RemoteBridgeRuntimeStatusData, error)
	Resize(context.Context, string, int, int) error
	Control(context.Context, string, protocol.CommandType, any) (protocol.Response, error)
	Close() error
}

type managedAttachStream struct {
	History string
	Reader  io.Reader
	Writer  io.Writer
	Close   func() error
}

type managedAttachOpener func(context.Context, string, string) (*managedAttachStream, error)

var defaultManagedAttachOpener managedAttachOpener = func(
	ctx context.Context,
	location string,
	runtimeID string,
) (*managedAttachStream, error) {
	stream, err := OpenSSHAttach(ctx, location, runtimeID)
	if err != nil {
		return nil, err
	}
	return &managedAttachStream{
		History: stream.History,
		Reader:  stream.Reader,
		Writer:  stream.Writer,
		Close:   stream.Close,
	}, nil
}

type ManagedRuntimeProxyConfig struct {
	Session      registry.RuntimeSession
	TargetID     string
	Location     string
	Bridge       ManagedBridgeControl
	Registry     *registry.Registry
	AttachOpener managedAttachOpener
}

// ManagedRuntimeProxy exposes one remote SessionHost through the ordinary local
// Nexus IPC endpoint. Existing terminal, MissionRunner and runtime callers can
// therefore remain transport-agnostic.
type ManagedRuntimeProxy struct {
	mu       sync.Mutex
	session  registry.RuntimeSession
	targetID string
	location string
	bridge   ManagedBridgeControl
	registry *registry.Registry
	attach   managedAttachOpener
	listener net.Listener
	ctx      context.Context
	cancel   context.CancelFunc
	closed   bool
}

// StartManagedRuntimeProxy binds the local runtime endpoint and registers a
// local proxy host identity. The provider PID remains remote-only metadata.
func StartManagedRuntimeProxy(
	ctx context.Context,
	cfg ManagedRuntimeProxyConfig,
) (*ManagedRuntimeProxy, registry.RuntimeSession, error) {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return nil, registry.RuntimeSession{}, err
		}
	}
	if cfg.Bridge == nil {
		return nil, registry.RuntimeSession{}, errors.New("managed runtime proxy requires a remote bridge")
	}
	if strings.TrimSpace(cfg.Session.RuntimeID) == "" {
		return nil, registry.RuntimeSession{}, errors.New("managed runtime proxy requires runtime id")
	}
	if strings.TrimSpace(cfg.Session.MachineID) == "" {
		return nil, registry.RuntimeSession{}, errors.New("managed runtime proxy requires remote machine identity")
	}
	targetID := strings.TrimSpace(cfg.TargetID)
	location := strings.TrimSpace(cfg.Location)
	if targetID == "" {
		return nil, registry.RuntimeSession{}, errors.New("managed runtime proxy requires target id")
	}
	if err := ValidateSSHLocation(location); err != nil {
		return nil, registry.RuntimeSession{}, err
	}

	reg := cfg.Registry
	if reg == nil {
		reg = registry.DefaultRegistry()
	}
	attach := cfg.AttachOpener
	if attach == nil {
		attach = defaultManagedAttachOpener
	}

	listener, err := protocol.Listen(cfg.Session.RuntimeID)
	if err != nil {
		return nil, registry.RuntimeSession{}, fmt.Errorf("bind managed runtime proxy: %w", err)
	}

	hostPath, err := os.Executable()
	if err != nil {
		_ = listener.Close()
		_ = os.Remove(protocol.EndpointPath(cfg.Session.RuntimeID))
		return nil, registry.RuntimeSession{}, fmt.Errorf("resolve Nexus proxy binary: %w", err)
	}
	hostFingerprint, err := registry.BinaryFingerprint(hostPath)
	if err != nil {
		_ = listener.Close()
		_ = os.Remove(protocol.EndpointPath(cfg.Session.RuntimeID))
		return nil, registry.RuntimeSession{}, fmt.Errorf("fingerprint Nexus proxy binary: %w", err)
	}

	session := cfg.Session
	session.Transport = registry.TransportManagedSSHProxy
	session.Location = location
	session.HostPID = os.Getpid()
	session.HostGeneration = time.Now().UnixNano()
	session.HostBinaryPath = hostPath
	session.HostBinaryFingerprint = hostFingerprint
	session.ProviderBinaryPath = ""
	session.ProviderBinaryFingerprint = ""
	session.ControlEndpoint = protocol.EndpointPath(session.RuntimeID)
	session.State = registry.StateRunning
	session.UpdatedAt = time.Now().UTC()
	if session.Labels == nil {
		session.Labels = map[string]string{}
	}
	session.Labels["nexus.remote_target_id"] = targetID
	session.Labels["nexus.remote_transport"] = "ssh"

	if err := reg.Register(session); err != nil {
		_ = listener.Close()
		_ = os.Remove(protocol.EndpointPath(session.RuntimeID))
		return nil, registry.RuntimeSession{}, fmt.Errorf("register managed runtime proxy: %w", err)
	}

	proxyCtx, cancel := context.WithCancel(context.Background())
	proxy := &ManagedRuntimeProxy{
		session:  session,
		targetID: targetID,
		location: location,
		bridge:   cfg.Bridge,
		registry: reg,
		attach:   attach,
		listener: listener,
		ctx:      proxyCtx,
		cancel:   cancel,
	}
	go proxy.serve()
	return proxy, session, nil
}

func (p *ManagedRuntimeProxy) serve() {
	p.mu.Lock()
	listener := p.listener
	p.mu.Unlock()
	if listener == nil {
		return
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			p.mu.Lock()
			closed := p.closed
			p.mu.Unlock()
			if closed {
				return
			}
			_ = p.registry.UpdateState(p.session.RuntimeID, registry.StateFailed)
			return
		}
		go p.handleConnection(conn)
	}
}

func (p *ManagedRuntimeProxy) handleConnection(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReaderSize(conn, managedProxyMaxFrameBytes)
	for {
		frame, err := readBoundedLine(reader, managedProxyMaxFrameBytes)
		if err != nil {
			return
		}
		var request protocol.Request
		if err := json.Unmarshal(frame, &request); err != nil {
			_ = writeProxyResponse(conn, proxyErrorResponse("", "INVALID_REQUEST", err))
			return
		}
		if request.Version != 0 && request.Version != protocol.ProtocolVersion {
			_ = writeProxyResponse(conn, proxyErrorResponse(request.ID, "ERROR_PROTOCOL_VERSION", errors.New("unsupported protocol version")))
			return
		}
		if request.Command == protocol.CmdAttach {
			p.handleAttach(conn, reader, request)
			return
		}

		response, closeAfter := p.handleRPC(request)
		if err := writeProxyResponse(conn, response); err != nil {
			return
		}
		if closeAfter {
			go p.Close()
			return
		}
	}
}

func (p *ManagedRuntimeProxy) handleRPC(request protocol.Request) (protocol.Response, bool) {
	switch request.Command {
	case protocol.CmdPing:
		response, err := p.bridge.Control(p.ctx, p.session.RuntimeID, protocol.CmdPing, nil)
		return p.localizeControlResponse(request.ID, response, err, "PING_FAILED"), false

	case protocol.CmdStatus:
		status, err := p.bridge.Status(p.ctx, p.session.RuntimeID)
		if err != nil {
			return proxyErrorResponse(request.ID, "STATUS_FAILED", err), false
		}
		response, err := protocol.NewResponse(protocol.StatusData{
			RuntimeID:             p.session.RuntimeID,
			ProviderID:            p.session.ProviderID,
			ProfileID:             p.session.ProfileID,
			ProviderSessionID:     p.session.ProviderSessionID,
			Workspace:             p.session.Workspace,
			PID:                   p.session.PID,
			ProviderGeneration:    p.session.ProviderGeneration,
			HostPID:               p.session.HostPID,
			HostGeneration:        p.session.HostGeneration,
			HostBinaryPath:        p.session.HostBinaryPath,
			HostBinaryFingerprint: p.session.HostBinaryFingerprint,
			State:                 status.State,
			ControlLevel:          status.ControlLevel,
			StartedAt:             p.session.StartedAt,
			Continuity:            p.session.Continuity,
		})
		if err != nil {
			return proxyErrorResponse(request.ID, "STATUS_ENCODE_FAILED", err), false
		}
		response.ID = request.ID
		response.RuntimeID = p.session.RuntimeID
		response.State = status.State
		response.Code = "STATUS_OK"
		response.Action = string(protocol.CmdStatus)
		response.Message = "remote runtime status"
		return response, false

	case protocol.CmdResize:
		var payload protocol.ResizePayload
		if err := json.Unmarshal(request.Payload, &payload); err != nil || payload.Rows <= 0 || payload.Cols <= 0 {
			return proxyErrorResponse(request.ID, "INVALID_RESIZE", errors.New("valid rows and cols are required")), false
		}
		if err := p.bridge.Resize(p.ctx, p.session.RuntimeID, payload.Rows, payload.Cols); err != nil {
			return proxyErrorResponse(request.ID, "RESIZE_FAILED", err), false
		}
		response, _ := protocol.NewResponse("resized")
		response.ID = request.ID
		response.RuntimeID = p.session.RuntimeID
		return response, false

	case protocol.CmdInput,
		protocol.CmdSubmitPrompt,
		protocol.CmdEvents,
		protocol.CmdUsage:
		response, err := p.bridge.Control(
			p.ctx,
			p.session.RuntimeID,
			request.Command,
			json.RawMessage(request.Payload),
		)
		return p.localizeControlResponse(request.ID, response, err, "CONTROL_FAILED"), false

	case protocol.CmdStop, protocol.CmdTerminate:
		response, err := p.bridge.Control(
			p.ctx,
			p.session.RuntimeID,
			request.Command,
			json.RawMessage(request.Payload),
		)
		if err != nil {
			return proxyErrorResponse(request.ID, "CONTROL_FAILED", err), false
		}
		return p.localizeControlResponse(request.ID, response, nil, "CONTROL_FAILED"), true

	case protocol.CmdHandoff, protocol.CmdContinue, protocol.CmdSlash:
		return proxyErrorResponse(
			request.ID,
			"REMOTE_CONTINUITY_UNAVAILABLE",
			errors.New("cross-machine handoff/continue/slash control is not enabled for managed SSH runtimes"),
		), false

	case protocol.CmdDetach, protocol.CmdLeaseAcquire, protocol.CmdLeaseRelease:
		return proxyErrorResponse(
			request.ID,
			"ATTACH_CHANNEL_REQUIRED",
			errors.New("terminal lease and detach commands must use the attached stream"),
		), false

	default:
		return proxyErrorResponse(
			request.ID,
			"UNSUPPORTED_COMMAND",
			fmt.Errorf("command %q is not supported by the managed runtime proxy", request.Command),
		), false
	}
}

func (p *ManagedRuntimeProxy) handleAttach(
	conn net.Conn,
	reader *bufio.Reader,
	request protocol.Request,
) {
	stream, err := p.attach(p.ctx, p.location, p.session.RuntimeID)
	if err != nil {
		_ = writeProxyResponse(conn, proxyErrorResponse(request.ID, "ATTACH_FAILED", err))
		return
	}
	defer stream.Close()

	response, err := protocol.NewResponse(stream.History)
	if err != nil {
		_ = writeProxyResponse(conn, proxyErrorResponse(request.ID, "ATTACH_FAILED", err))
		return
	}
	response.ID = request.ID
	response.RuntimeID = p.session.RuntimeID
	if err := writeProxyResponse(conn, response); err != nil {
		return
	}

	errCh := make(chan error, 2)
	go func() {
		_, copyErr := io.Copy(stream.Writer, reader)
		errCh <- copyErr
	}()
	go func() {
		_, copyErr := io.Copy(conn, stream.Reader)
		errCh <- copyErr
	}()

	<-errCh
	_ = stream.Close()
	_ = conn.Close()
	<-errCh
}

func (p *ManagedRuntimeProxy) localizeControlResponse(
	requestID string,
	response protocol.Response,
	err error,
	code string,
) protocol.Response {
	if err != nil {
		return proxyErrorResponse(requestID, code, err)
	}
	response.Version = protocol.ProtocolVersion
	response.ID = requestID
	if response.RuntimeID == "" {
		response.RuntimeID = p.session.RuntimeID
	}
	return response
}

func (p *ManagedRuntimeProxy) Close() error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	p.closed = true
	listener := p.listener
	p.listener = nil
	bridge := p.bridge
	cancel := p.cancel
	p.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if listener != nil {
		_ = listener.Close()
	}
	var bridgeErr error
	if bridge != nil {
		bridgeErr = bridge.Close()
	}
	_ = os.Remove(protocol.EndpointPath(p.session.RuntimeID))
	_ = p.registry.TransitionState(p.session.RuntimeID, registry.StateStopped)
	return bridgeErr
}

func proxyErrorResponse(requestID, code string, err error) protocol.Response {
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

func writeProxyResponse(writer io.Writer, response protocol.Response) error {
	if writer == nil {
		return errors.New("proxy response writer is unavailable")
	}
	raw, err := json.Marshal(response)
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	_, err = writer.Write(raw)
	return err
}
