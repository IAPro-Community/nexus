package protocol

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"net"
	"testing"
	"time"
)

func TestProtocolSerialization(t *testing.T) {
	req, err := NewRequest(CmdResize, ResizePayload{Rows: 24, Cols: 80})
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	if req.Command != CmdResize {
		t.Errorf("expected CmdResize, got %s", req.Command)
	}
	if req.ID == "" {
		t.Errorf("expected request ID to be populated")
	}

	resp, err := NewResponse(StatusData{
		RuntimeID: "rt-123",
		State:     "RUNNING",
		PID:       42,
	})
	if err != nil {
		t.Fatalf("failed to create response: %v", err)
	}
	if !resp.OK {
		t.Errorf("expected OK true")
	}

	errResp := NewErrorResponse("something went wrong")
	if errResp.OK || errResp.Error != "something went wrong" {
		t.Errorf("unexpected error response: %+v", errResp)
	}
}

func TestControlResponseHasStableEnvelope(t *testing.T) {
	resp := NewControlResponse(ControlResult{
		OK: true, Code: "STOP_REQUESTED", RuntimeID: "rt-1", Action: string(CmdStop),
		State: "STOPPING", Message: "stop requested", CorrelationID: "lineage-1",
	})
	if !resp.OK || resp.Code != "STOP_REQUESTED" || resp.RuntimeID != "rt-1" || resp.Action != string(CmdStop) || resp.State != "STOPPING" || resp.CorrelationID != "lineage-1" {
		t.Fatalf("unexpected control response: %+v", resp)
	}
	var payload ControlResult
	if err := json.Unmarshal(resp.Data, &payload); err != nil {
		t.Fatal(err)
	}
	if payload != (ControlResult{OK: true, Code: "STOP_REQUESTED", RuntimeID: "rt-1", Action: string(CmdStop), State: "STOPPING", Message: "stop requested", CorrelationID: "lineage-1"}) {
		t.Fatalf("unexpected response data: %+v", payload)
	}
}

func TestSendContextInterruptsBlockedConnection(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	client := &Client{runtimeID: "context-test", conn: clientConn, reader: bufio.NewReader(clientConn)}
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	started := time.Now()
	_, err := client.SendContext(ctx, CmdPing, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("SendContext error = %v, want context.Canceled", err)
	}
	if elapsed := time.Since(started); elapsed >= time.Second {
		t.Fatalf("cancellation took %s", elapsed)
	}
	if client.conn != nil {
		t.Fatal("canceled transport remained reusable after an interrupted RPC")
	}
}

func TestClientServerCommunication(t *testing.T) {
	runtimeID := "test-rt-001"
	l, err := Listen(runtimeID)
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer l.Close()

	// Mock Server
	go func() {
		conn, err := l.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		reader := bufio.NewReader(conn)
		for {
			line, err := reader.ReadBytes('\n')
			if err != nil {
				return
			}
			var req Request
			if err := json.Unmarshal(line, &req); err != nil {
				return
			}

			var resp Response
			switch req.Command {
			case CmdPing:
				resp, _ = NewResponse("pong")
			case CmdStatus:
				resp, _ = NewResponse(StatusData{
					RuntimeID: runtimeID,
					State:     "RUNNING",
					PID:       1001,
				})
			default:
				resp = NewErrorResponse("unknown command")
			}
			data, _ := json.Marshal(resp)
			_, _ = conn.Write(append(data, '\n'))
		}
	}()

	client, err := NewClient(runtimeID)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	if err := client.Ping(); err != nil {
		t.Errorf("ping failed: %v", err)
	}

	st, err := client.Status()
	if err != nil {
		t.Fatalf("status failed: %v", err)
	}
	if st.RuntimeID != runtimeID || st.State != "RUNNING" || st.PID != 1001 {
		t.Errorf("unexpected status data: %+v", st)
	}
}

func TestSubmitPromptRequestUsesDedicatedCommandAndPayload(t *testing.T) {
	req, err := NewRequest(CmdSubmitPrompt, SubmitPromptPayload{Prompt: "/ai literal prompt"})
	if err != nil {
		t.Fatal(err)
	}
	if req.Command != CmdSubmitPrompt {
		t.Fatalf("command=%s", req.Command)
	}
	var payload SubmitPromptPayload
	if err := json.Unmarshal(req.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Prompt != "/ai literal prompt" {
		t.Fatalf("prompt=%q", payload.Prompt)
	}
}
