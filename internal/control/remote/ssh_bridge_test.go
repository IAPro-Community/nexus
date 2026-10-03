package remote

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/protocol"
)

func TestValidateSSHLocationAndFixedBridgeArgs(t *testing.T) {
	for _, location := range []string{"devbox", "user@devbox", "host.example.com", "[2001:db8::1]"} {
		if err := ValidateSSHLocation(location); err != nil {
			t.Fatalf("valid SSH location %q rejected: %v", location, err)
		}
	}
	for _, location := range []string{"", "-oProxyCommand=bad", "dev box", "host
bad", "ssh://devbox"} {
		if err := ValidateSSHLocation(location); err == nil {
			t.Fatalf("invalid SSH location %q accepted", location)
		}
	}

	control := sshBridgeArgs("devbox", false)
	attach := sshBridgeArgs("devbox", true)
	for _, args := range [][]string{control, attach} {
		wire := strings.Join(args, " ")
		if strings.Contains(wire, "sh -lc") || strings.Contains(wire, "{provider}") || strings.Contains(wire, "{args}") {
			t.Fatalf("SSH bridge args introduced shell/provider interpolation: %q", wire)
		}
		if !strings.Contains(wire, "BatchMode=yes") ||
			!strings.Contains(wire, "PermitLocalCommand=no") ||
			!strings.Contains(wire, "StrictHostKeyChecking=yes") ||
			!strings.Contains(wire, "ConnectTimeout=10") {
			t.Fatalf("SSH bridge missing fail-closed options: %q", wire)
		}
	}
	if got := control[len(control)-3:]; strings.Join(got, " ") != "nexus __remote-bridge --stdio" {
		t.Fatalf("unexpected control remote command: %#v", got)
	}
	if got := attach[len(attach)-3:]; strings.Join(got, " ") != "nexus __remote-bridge --attach-stdio" {
		t.Fatalf("unexpected attach remote command: %#v", got)
	}
}

func TestSSHBridgeRoundTripPreservesStructuredPayload(t *testing.T) {
	requestReader, requestWriter := io.Pipe()
	responseReader, responseWriter := io.Pipe()
	defer requestReader.Close()
	defer responseWriter.Close()

	client := &SSHBridgeClient{
		targetID: "devbox",
		location: "devbox",
		stdin:    requestWriter,
		stdout:   bufio.NewReader(responseReader),
		stderr:   newTailBuffer(remoteBridgeStderrLimit),
	}

	serverDone := make(chan error, 1)
	go func() {
		reader := bufio.NewReader(requestReader)
		line, err := readBoundedLine(reader, remoteBridgeMaxResponse)
		if err != nil {
			serverDone <- err
			return
		}
		var request protocol.Request
		if err := json.Unmarshal(line, &request); err != nil {
			serverDone <- err
			return
		}
		if request.Command != protocol.CmdRemoteBridgeSpawn {
			serverDone <- errors.New("unexpected bridge command")
			return
		}
		var payload protocol.RemoteBridgeSpawnPayload
		if err := json.Unmarshal(request.Payload, &payload); err != nil {
			serverDone <- err
			return
		}
		if len(payload.Args) != 2 || payload.Args[1] != "literal ; $(echo data) && no-shell" {
			serverDone <- errors.New("argv was not preserved as structured data")
			return
		}
		response, err := protocol.NewResponse(protocol.RemoteBridgeSpawnData{
			RuntimeID: payload.RuntimeID,
			MachineID: "machine-remote",
			Transport: "nexus-remote-bridge",
		})
		if err != nil {
			serverDone <- err
			return
		}
		response.ID = request.ID
		response.RuntimeID = payload.RuntimeID
		raw, err := json.Marshal(response)
		if err != nil {
			serverDone <- err
			return
		}
		_, err = responseWriter.Write(append(raw, '
'))
		serverDone <- err
	}()

	data, err := client.Spawn(context.Background(), protocol.RemoteBridgeSpawnPayload{
		RuntimeID: "runtime-1",
		AgentID:   "agent-1",
		ProjectID: "project-1",
		Provider:  "codex",
		Profile:   "work",
		Model:     "gpt-5.6",
		Workspace: "/srv/project with spaces",
		Args:      []string{"--prompt", "literal ; $(echo data) && no-shell"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if data.RuntimeID != "runtime-1" || data.MachineID != "machine-remote" {
		t.Fatalf("unexpected spawn data: %+v", data)
	}
	if err := <-serverDone; err != nil {
		t.Fatal(err)
	}
}

func TestSSHBridgeRoundTripRejectsCorrelationMismatch(t *testing.T) {
	requestReader, requestWriter := io.Pipe()
	responseReader, responseWriter := io.Pipe()
	client := &SSHBridgeClient{
		targetID: "devbox",
		location: "devbox",
		stdin:    requestWriter,
		stdout:   bufio.NewReader(responseReader),
		stderr:   newTailBuffer(remoteBridgeStderrLimit),
	}
	defer requestReader.Close()
	defer responseWriter.Close()

	go func() {
		reader := bufio.NewReader(requestReader)
		line, _ := readBoundedLine(reader, remoteBridgeMaxResponse)
		var request protocol.Request
		_ = json.Unmarshal(line, &request)
		response, _ := protocol.NewResponse(protocol.RemoteBridgeRuntimeStatusData{RuntimeID: "runtime-1"})
		response.ID = "wrong-id"
		raw, _ := json.Marshal(response)
		_, _ = responseWriter.Write(append(raw, '
'))
	}()

	_, err := client.Status(context.Background(), "runtime-1")
	if err == nil || !strings.Contains(err.Error(), "correlation mismatch") {
		t.Fatalf("correlation mismatch must fail closed, got %v", err)
	}
}

func TestSSHBridgeRoundTripSurfacesStructuredBridgeError(t *testing.T) {
	requestReader, requestWriter := io.Pipe()
	responseReader, responseWriter := io.Pipe()
	client := &SSHBridgeClient{
		targetID: "devbox",
		location: "devbox",
		stdin:    requestWriter,
		stdout:   bufio.NewReader(responseReader),
		stderr:   newTailBuffer(remoteBridgeStderrLimit),
	}
	defer requestReader.Close()
	defer responseWriter.Close()

	go func() {
		reader := bufio.NewReader(requestReader)
		line, _ := readBoundedLine(reader, remoteBridgeMaxResponse)
		var request protocol.Request
		_ = json.Unmarshal(line, &request)
		response := protocol.Response{
			Version: protocol.ProtocolVersion,
			ID:      request.ID,
			OK:      false,
			Code:    "REMOTE_EXECUTION_DISABLED",
			Message: "remote execution is disabled on this host",
		}
		raw, _ := json.Marshal(response)
		_, _ = responseWriter.Write(append(raw, '
'))
	}()

	_, err := client.Status(context.Background(), "runtime-1")
	var bridgeErr *BridgeError
	if !errors.As(err, &bridgeErr) || bridgeErr.Code != "REMOTE_EXECUTION_DISABLED" {
		t.Fatalf("expected structured BridgeError, got %T %v", err, err)
	}
}

func TestSSHBridgeCancellationConsumesTransport(t *testing.T) {
	requestReader, requestWriter := io.Pipe()
	responseReader, responseWriter := io.Pipe()
	client := &SSHBridgeClient{
		targetID: "devbox",
		location: "devbox",
		stdin:    requestWriter,
		stdout:   bufio.NewReader(responseReader),
		stderr:   newTailBuffer(remoteBridgeStderrLimit),
	}
	defer requestReader.Close()
	defer responseWriter.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, err := client.Status(ctx, "runtime-1")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("cancelled bridge request error=%v", err)
	}
	if !client.closed {
		t.Fatal("cancelled bridge request must consume transport")
	}
}

func TestReadBoundedLineAndTailBufferStayBounded(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("ok
"))
	line, err := readBoundedLine(reader, 8)
	if err != nil || string(line) != "ok" {
		t.Fatalf("bounded line=%q err=%v", line, err)
	}

	tooLarge := bufio.NewReader(strings.NewReader(strings.Repeat("x", 9) + "
"))
	if _, err := readBoundedLine(tooLarge, 8); err == nil {
		t.Fatal("oversized line must fail")
	}

	buffer := newTailBuffer(8)
	_, _ = buffer.Write([]byte("1234"))
	_, _ = buffer.Write([]byte("567890"))
	if got := buffer.String(); got != "34567890" {
		t.Fatalf("tail buffer=%q", got)
	}
}
