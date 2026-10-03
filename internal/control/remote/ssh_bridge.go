package remote

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strings"
	"sync"

	"github.com/IAPro-Community/nexus/internal/control/protocol"
	"github.com/IAPro-Community/nexus/internal/core/security"
)

const (
	remoteBridgeBinary       = "nexus"
	remoteBridgeMaxResponse  = 1024 * 1024
	remoteBridgeStderrLimit  = 32 * 1024
)

var sshLocationPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:@%+\[\]-]{0,239}$`)

type BridgeError struct {
	Code    string
	Message string
}

func (e *BridgeError) Error() string {
	if strings.TrimSpace(e.Message) == "" {
		return e.Code
	}
	return e.Code + ": " + e.Message
}

type SSHBridgeClient struct {
	mu       sync.Mutex
	targetID string
	location string
	cmd      *exec.Cmd
	stdin    io.WriteCloser
	stdout   *bufio.Reader
	stderr   *tailBuffer
	closed   bool
}

type SSHAttachStream struct {
	History string
	Reader  io.Reader
	Writer  io.Writer

	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stderr *tailBuffer
	once   sync.Once
}

func OpenSSHBridge(ctx context.Context, targetID, location string) (*SSHBridgeClient, protocol.RemoteBridgeHelloData, error) {
	targetID = strings.TrimSpace(targetID)
	location = strings.TrimSpace(location)
	if err := protocol.ValidateRemoteBridgeHello(protocol.RemoteBridgeHelloPayload{
		ProtocolVersion: protocol.ProtocolVersion,
		TargetID:        targetID,
	}); err != nil {
		return nil, protocol.RemoteBridgeHelloData{}, err
	}
	if err := ValidateSSHLocation(location); err != nil {
		return nil, protocol.RemoteBridgeHelloData{}, err
	}

	cmd := exec.CommandContext(ctx, "ssh", sshBridgeArgs(location, false)...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, protocol.RemoteBridgeHelloData{}, err
	}
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return nil, protocol.RemoteBridgeHelloData{}, err
	}
	stderr := newTailBuffer(remoteBridgeStderrLimit)
	cmd.Stderr = stderr
	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		return nil, protocol.RemoteBridgeHelloData{}, fmt.Errorf("start ssh bridge: %w", err)
	}

	client := &SSHBridgeClient{
		targetID: targetID,
		location: location,
		cmd:      cmd,
		stdin:    stdin,
		stdout:   bufio.NewReaderSize(stdoutPipe, 64*1024),
		stderr:   stderr,
	}
	hello, err := client.Hello(ctx)
	if err != nil {
		_ = client.Close()
		return nil, protocol.RemoteBridgeHelloData{}, err
	}
	return client, hello, nil
}

func ValidateSSHLocation(location string) error {
	location = strings.TrimSpace(location)
	if !sshLocationPattern.MatchString(location) || strings.HasPrefix(location, "-") {
		return fmt.Errorf("invalid SSH location %q", location)
	}
	return nil
}

func sshBridgeArgs(location string, attach bool) []string {
	mode := "--stdio"
	if attach {
		mode = "--attach-stdio"
	}
	return []string{
		"-T",
		"-o", "BatchMode=yes",
		"-o", "PermitLocalCommand=no",
		"-o", "StrictHostKeyChecking=yes",
		"-o", "ConnectTimeout=10",
		"-o", "ConnectionAttempts=1",
		"-o", "LogLevel=ERROR",
		location,
		remoteBridgeBinary,
		"__remote-bridge",
		mode,
	}
}

func (c *SSHBridgeClient) Hello(ctx context.Context) (protocol.RemoteBridgeHelloData, error) {
	var data protocol.RemoteBridgeHelloData
	err := c.roundTrip(ctx, protocol.CmdRemoteBridgeHello, protocol.RemoteBridgeHelloPayload{
		ProtocolVersion: protocol.ProtocolVersion,
		TargetID:        c.targetID,
	}, &data)
	if err != nil {
		return protocol.RemoteBridgeHelloData{}, err
	}
	if err := protocol.ValidateRemoteBridgeHelloData(data); err != nil {
		return protocol.RemoteBridgeHelloData{}, err
	}
	return data, nil
}

func (c *SSHBridgeClient) Spawn(ctx context.Context, payload protocol.RemoteBridgeSpawnPayload) (protocol.RemoteBridgeSpawnData, error) {
	if err := protocol.ValidateRemoteBridgeSpawn(payload); err != nil {
		return protocol.RemoteBridgeSpawnData{}, err
	}
	var data protocol.RemoteBridgeSpawnData
	if err := c.roundTrip(ctx, protocol.CmdRemoteBridgeSpawn, payload, &data); err != nil {
		return protocol.RemoteBridgeSpawnData{}, err
	}
	return data, nil
}

func (c *SSHBridgeClient) Status(ctx context.Context, runtimeID string) (protocol.RemoteBridgeRuntimeStatusData, error) {
	payload := protocol.RemoteBridgeRuntimePayload{RuntimeID: runtimeID}
	if err := protocol.ValidateRemoteBridgeRuntime(payload); err != nil {
		return protocol.RemoteBridgeRuntimeStatusData{}, err
	}
	var data protocol.RemoteBridgeRuntimeStatusData
	if err := c.roundTrip(ctx, protocol.CmdRemoteBridgeStatus, payload, &data); err != nil {
		return protocol.RemoteBridgeRuntimeStatusData{}, err
	}
	return data, nil
}

func (c *SSHBridgeClient) Resize(ctx context.Context, runtimeID string, rows, cols int) error {
	payload := protocol.RemoteBridgeResizePayload{RuntimeID: runtimeID, Rows: rows, Cols: cols}
	if err := protocol.ValidateRemoteBridgeResize(payload); err != nil {
		return err
	}
	return c.roundTrip(ctx, protocol.CmdRemoteBridgeResize, payload, nil)
}

func (c *SSHBridgeClient) roundTrip(ctx context.Context, command protocol.CommandType, payload any, out any) error {
	if ctx == nil {
		ctx = context.Background()
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed || c.stdin == nil || c.stdout == nil {
		return errors.New("SSH bridge is closed")
	}

	request, err := protocol.NewRequest(command, payload)
	if err != nil {
		return err
	}
	wire, err := json.Marshal(request)
	if err != nil {
		return err
	}
	wire = append(wire, '\n')

	type result struct {
		response protocol.Response
		err      error
	}
	done := make(chan result, 1)
	go func() {
		if _, err := c.stdin.Write(wire); err != nil {
			done <- result{err: err}
			return
		}
		line, err := readBoundedLine(c.stdout, remoteBridgeMaxResponse)
		if err != nil {
			done <- result{err: err}
			return
		}
		var response protocol.Response
		if err := json.Unmarshal(line, &response); err != nil {
			done <- result{err: err}
			return
		}
		done <- result{response: response}
	}()

	select {
	case <-ctx.Done():
		c.abortLocked()
		return ctx.Err()
	case got := <-done:
		if got.err != nil {
			return c.transportError(got.err)
		}
		response := got.response
		if response.Version != protocol.ProtocolVersion {
			return fmt.Errorf("remote bridge protocol version %d is unsupported", response.Version)
		}
		if response.ID != request.ID {
			return fmt.Errorf("remote bridge response correlation mismatch")
		}
		if !response.OK {
			return &BridgeError{Code: response.Code, Message: security.Redact(firstNonEmpty(response.Message, response.Error))}
		}
		if out != nil && len(response.Data) > 0 {
			if err := json.Unmarshal(response.Data, out); err != nil {
				return fmt.Errorf("decode remote bridge response: %w", err)
			}
		}
		return nil
	}
}

func (c *SSHBridgeClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil
	}
	c.closed = true
	return c.terminateLocked()
}

func (c *SSHBridgeClient) abortLocked() {
	if c.closed {
		return
	}
	c.closed = true
	_ = c.terminateLocked()
}

func (c *SSHBridgeClient) terminateLocked() error {
	if c.stdin != nil {
		_ = c.stdin.Close()
		c.stdin = nil
	}
	if c.cmd == nil || c.cmd.Process == nil {
		return nil
	}
	_ = c.cmd.Process.Kill()
	err := c.cmd.Wait()
	c.cmd = nil
	if err != nil && !isExpectedProcessExit(err) {
		return c.transportError(err)
	}
	return nil
}

func (c *SSHBridgeClient) transportError(err error) error {
	if err == nil {
		return nil
	}
	detail := ""
	if c.stderr != nil {
		detail = strings.TrimSpace(c.stderr.String())
	}
	if detail == "" {
		return fmt.Errorf("SSH bridge transport failed: %w", err)
	}
	return fmt.Errorf("SSH bridge transport failed: %w: %s", err, security.Redact(detail))
}

func OpenSSHAttach(ctx context.Context, location, runtimeID string) (*SSHAttachStream, error) {
	location = strings.TrimSpace(location)
	if err := ValidateSSHLocation(location); err != nil {
		return nil, err
	}
	payload := protocol.RemoteBridgeRuntimePayload{RuntimeID: strings.TrimSpace(runtimeID)}
	if err := protocol.ValidateRemoteBridgeRuntime(payload); err != nil {
		return nil, err
	}

	cmd := exec.CommandContext(ctx, "ssh", sshBridgeArgs(location, true)...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return nil, err
	}
	stderr := newTailBuffer(remoteBridgeStderrLimit)
	cmd.Stderr = stderr
	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		return nil, fmt.Errorf("start ssh attach bridge: %w", err)
	}
	stdout := bufio.NewReaderSize(stdoutPipe, 64*1024)

	request, err := protocol.NewRequest(protocol.CmdRemoteBridgeAttach, payload)
	if err != nil {
		_ = stdin.Close()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return nil, err
	}
	wire, err := json.Marshal(request)
	if err != nil {
		_ = stdin.Close()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return nil, err
	}
	if _, err := stdin.Write(append(wire, '\n')); err != nil {
		_ = stdin.Close()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return nil, fmt.Errorf("write ssh attach handshake: %w", err)
	}

	line, err := readBoundedLine(stdout, remoteBridgeMaxResponse)
	if err != nil {
		_ = stdin.Close()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return nil, remoteProcessError("read ssh attach handshake", err, stderr)
	}
	var response protocol.Response
	if err := json.Unmarshal(line, &response); err != nil {
		_ = stdin.Close()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return nil, fmt.Errorf("decode ssh attach handshake: %w", err)
	}
	if response.Version != protocol.ProtocolVersion || response.ID != request.ID {
		_ = stdin.Close()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return nil, fmt.Errorf("ssh attach handshake correlation/version mismatch")
	}
	if !response.OK {
		_ = stdin.Close()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return nil, &BridgeError{Code: response.Code, Message: security.Redact(firstNonEmpty(response.Message, response.Error))}
	}
	var data protocol.RemoteBridgeAttachData
	if err := json.Unmarshal(response.Data, &data); err != nil {
		_ = stdin.Close()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return nil, fmt.Errorf("decode ssh attach acknowledgement: %w", err)
	}
	if data.RuntimeID != payload.RuntimeID {
		_ = stdin.Close()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return nil, fmt.Errorf("ssh attach runtime mismatch")
	}

	return &SSHAttachStream{
		History: data.History,
		Reader:  stdout,
		Writer:  stdin,
		cmd:     cmd,
		stdin:   stdin,
		stderr:  stderr,
	}, nil
}

func (s *SSHAttachStream) Close() error {
	var result error
	s.once.Do(func() {
		if s.stdin != nil {
			_ = s.stdin.Close()
		}
		if s.cmd != nil && s.cmd.Process != nil {
			_ = s.cmd.Process.Kill()
			err := s.cmd.Wait()
			if err != nil && !isExpectedProcessExit(err) {
				result = remoteProcessError("close ssh attach", err, s.stderr)
			}
		}
	})
	return result
}

func readBoundedLine(reader *bufio.Reader, limit int) ([]byte, error) {
	if reader == nil {
		return nil, errors.New("bridge response reader is unavailable")
	}
	var line []byte
	for {
		part, isPrefix, err := reader.ReadLine()
		line = append(line, part...)
		if len(line) > limit {
			return nil, fmt.Errorf("bridge response exceeded %d bytes", limit)
		}
		if err != nil {
			return nil, err
		}
		if !isPrefix {
			return line, nil
		}
	}
}

func remoteProcessError(prefix string, err error, stderr *tailBuffer) error {
	detail := ""
	if stderr != nil {
		detail = strings.TrimSpace(stderr.String())
	}
	if detail == "" {
		return fmt.Errorf("%s: %w", prefix, err)
	}
	return fmt.Errorf("%s: %w: %s", prefix, err, security.Redact(detail))
}

func isExpectedProcessExit(err error) bool {
	var exitErr *exec.ExitError
	return errors.As(err, &exitErr)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

type tailBuffer struct {
	mu    sync.Mutex
	limit int
	data  []byte
}

func newTailBuffer(limit int) *tailBuffer {
	return &tailBuffer{limit: limit}
}

func (b *tailBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.limit <= 0 {
		return len(p), nil
	}
	if len(p) >= b.limit {
		b.data = append(b.data[:0], p[len(p)-b.limit:]...)
		return len(p), nil
	}
	if overflow := len(b.data) + len(p) - b.limit; overflow > 0 {
		b.data = append([]byte(nil), b.data[overflow:]...)
	}
	b.data = append(b.data, p...)
	return len(p), nil
}

func (b *tailBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return string(append([]byte(nil), b.data...))
}
