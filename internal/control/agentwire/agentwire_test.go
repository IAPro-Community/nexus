package agentwire

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestJSONLinesBounds(t *testing.T) {
	j := NewJSONLines(strings.NewReader("{\"x\":\"a\\u2028b\"}\n"), io.Discard)
	frame, err := j.Read()
	if err != nil || !json.Valid(frame) {
		t.Fatalf("invalid frame: %s %v", frame, err)
	}
	huge := NewJSONLines(strings.NewReader(strings.Repeat("x", MaxFrame+1)+"\n"), io.Discard)
	_, err = huge.Read()
	if !errors.Is(err, ErrFrameTooLarge) {
		t.Fatalf("expected frame bound, got %v", err)
	}
}

type synchronizedBuffer struct {
	mu   sync.Mutex
	data bytes.Buffer
}

func (s *synchronizedBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.data.Write(p)
}
func (s *synchronizedBuffer) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.data.Len()
}
func TestACPRequestCorrelation(t *testing.T) {
	in, out := io.Pipe()
	defer in.Close()
	defer out.Close()
	written := &synchronizedBuffer{}
	acp := NewACP(in, written)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = acp.Pump(ctx) }()
	result := make(chan error, 1)
	go func() {
		var response struct {
			Count int `json:"count"`
		}
		err := acp.Call(ctx, "test/ping", map[string]any{}, &response)
		if err == nil && response.Count != 5 {
			err = errors.New("wrong response")
		}
		result <- err
	}()
	deadline := time.After(time.Second)
	for written.Len() == 0 {
		select {
		case <-deadline:
			t.Fatal("ACP request not written")
		default:
			time.Sleep(time.Millisecond)
		}
	}
	_, _ = out.Write([]byte("{\"jsonrpc\":\"2.0\",\"id\":\"1\",\"result\":{\"count\":5}}\n"))
	select {
	case err := <-result:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("no correlated response")
	}
}
func TestPiRPCAcknowledgementAndSettledEvent(t *testing.T) {
	in, out := io.Pipe()
	defer in.Close()
	defer out.Close()
	written := &synchronizedBuffer{}
	pi := NewPiRPC(in, written)
	events := make(chan string, 2)
	pi.OnEvent(func(kind string, _ json.RawMessage) { events <- kind })
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = pi.Pump(ctx) }()
	complete := make(chan error, 1)
	go func() { complete <- pi.Command(ctx, "prompt", map[string]any{"message": "hello"}, nil) }()
	deadline := time.After(time.Second)
	for written.Len() == 0 {
		select {
		case <-deadline:
			t.Fatal("Pi RPC not written")
		default:
			time.Sleep(time.Millisecond)
		}
	}
	_, _ = out.Write([]byte("{\"type\":\"response\",\"id\":\"1\",\"command\":\"prompt\",\"success\":true}\n{\"type\":\"agent_end\"}\n{\"type\":\"agent_settled\"}\n"))
	select {
	case err := <-complete:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("Pi command never completed")
	}
	if <-events != "agent_end" || <-events != "agent_settled" {
		t.Fatal("lost settled event")
	}
}
func TestReducerIdentityAndEpoch(t *testing.T) {
	r := NewReducer()
	input := Observation{
		RuntimeID: "first", Epoch: 1, Sequence: 1, Turn: TurnRunning,
		Evidence: Authoritative, Request: &Intervention{ID: "question1", Kind: Question},
	}
	got, err := r.Apply(input)
	if err != nil || len(got.Attention) != 1 {
		t.Fatalf("initial state: %+v %v", got, err)
	}
	got, _ = r.Apply(input)
	if len(got.Attention) != 1 {
		t.Fatal("duplicate interaction")
	}
	second := input
	second.RuntimeID = "second"
	got, _ = r.Apply(second)
	if len(got.Attention) != 1 {
		t.Fatal("merged independent runtimes")
	}
	got, _ = r.Apply(Observation{RuntimeID: "first", Epoch: 1, Sequence: 2, Evidence: Inferred, ResolvedID: "question1"})
	if len(got.Attention) != 1 {
		t.Fatal("inferred input incorrectly resolved a pending action")
	}
	got, _ = r.Apply(Observation{RuntimeID: "first", Epoch: 1, Sequence: 3, Evidence: Authoritative, ResolvedID: "question1"})
	if len(got.Attention) != 0 {
		t.Fatal("confirmed interaction persisted")
	}
	got, _ = r.Apply(input)
	if len(got.Attention) != 0 {
		t.Fatal("stale replay")
	}
	got, _ = r.Apply(Observation{RuntimeID: "first", Epoch: 2, Sequence: 1, Turn: TurnRunning})
	if got.Epoch != 2 || got.Sequence != 1 {
		t.Fatal("epoch change not fenced")
	}
}

func TestJSONLinesWhitespaceAndEmptyLines(t *testing.T) {
	input := "\n   \n\r\n{\"id\":1}\r\n\n  \n{\"id\":2}\n   \n"
	j := NewJSONLines(strings.NewReader(input), io.Discard)

	f1, err := j.Read()
	if err != nil {
		t.Fatalf("read frame 1 failed: %v", err)
	}
	if string(f1) != "{\"id\":1}" {
		t.Fatalf("unexpected frame 1: %s", string(f1))
	}

	f2, err := j.Read()
	if err != nil {
		t.Fatalf("read frame 2 failed: %v", err)
	}
	if string(f2) != "{\"id\":2}" {
		t.Fatalf("unexpected frame 2: %s", string(f2))
	}

	_, err = j.Read()
	if err != io.EOF {
		t.Fatalf("expected io.EOF after trailing whitespace, got %v", err)
	}
}

func TestACPConcurrentOutOfOrderRequests(t *testing.T) {
	in, out := io.Pipe()
	clientIn, clientOut := io.Pipe()
	defer in.Close()
	defer out.Close()
	defer clientIn.Close()
	defer clientOut.Close()

	acp := NewACP(in, clientOut)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() { _ = acp.Pump(ctx) }()

	type testResp struct {
		Name string `json:"name"`
	}

	resCh1 := make(chan testResp, 1)
	errCh1 := make(chan error, 1)
	resCh2 := make(chan testResp, 1)
	errCh2 := make(chan error, 1)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		var r testResp
		err := acp.Call(ctx, "ping1", map[string]any{}, &r)
		errCh1 <- err
		resCh1 <- r
	}()

	go func() {
		defer wg.Done()
		var r testResp
		err := acp.Call(ctx, "ping2", map[string]any{}, &r)
		errCh2 <- err
		resCh2 <- r
	}()

	// Read both requests from clientIn to learn their dynamic IDs
	wireReader := NewJSONLines(clientIn, io.Discard)
	req1Raw, err := wireReader.Read()
	if err != nil {
		t.Fatalf("failed reading req 1: %v", err)
	}
	req2Raw, err := wireReader.Read()
	if err != nil {
		t.Fatalf("failed reading req 2: %v", err)
	}

	var req1, req2 struct {
		ID     string `json:"id"`
		Method string `json:"method"`
	}
	_ = json.Unmarshal(req1Raw, &req1)
	_ = json.Unmarshal(req2Raw, &req2)

	// Send responses in reverse order of reception
	resp2 := fmt.Sprintf("{\"jsonrpc\":\"2.0\",\"id\":%q,\"result\":{\"name\":%q}}\n", req2.ID, req2.Method)
	resp1 := fmt.Sprintf("{\"jsonrpc\":\"2.0\",\"id\":%q,\"result\":{\"name\":%q}}\n", req1.ID, req1.Method)

	_, _ = out.Write([]byte(resp2))
	_, _ = out.Write([]byte(resp1))

	wg.Wait()

	if err := <-errCh1; err != nil {
		t.Fatalf("call 1 failed: %v", err)
	}
	if r1 := <-resCh1; r1.Name != "ping1" {
		t.Fatalf("call 1 expected 'ping1', got '%s'", r1.Name)
	}

	if err := <-errCh2; err != nil {
		t.Fatalf("call 2 failed: %v", err)
	}
	if r2 := <-resCh2; r2.Name != "ping2" {
		t.Fatalf("call 2 expected 'ping2', got '%s'", r2.Name)
	}
}

func TestACPProcessPipeClosureDuringCall(t *testing.T) {
	in, out := io.Pipe()
	written := &synchronizedBuffer{}
	acp := NewACP(in, written)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() { _ = acp.Pump(ctx) }()

	errCh := make(chan error, 1)
	go func() {
		var resp map[string]any
		errCh <- acp.Call(ctx, "test/hang", map[string]any{}, &resp)
	}()

	// Wait until request is on wire
	deadline := time.After(2 * time.Second)
	for {
		if written.Len() > 0 {
			break
		}
		select {
		case <-deadline:
			t.Fatal("request was not written")
		default:
			time.Sleep(5 * time.Millisecond)
		}
	}

	// Close pipe to simulate subprocess death
	_ = in.Close()
	_ = out.Close()

	select {
	case err := <-errCh:
		if err == nil {
			t.Fatal("expected error on pipe closure, got nil")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("call remained blocked after pipe closure")
	}
}

func TestPiRPCConcurrentOutOfOrderCommands(t *testing.T) {
	in, out := io.Pipe()
	clientIn, clientOut := io.Pipe()
	defer in.Close()
	defer out.Close()
	defer clientIn.Close()
	defer clientOut.Close()

	pi := NewPiRPC(in, clientOut)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() { _ = pi.Pump(ctx) }()

	errCh1 := make(chan error, 1)
	errCh2 := make(chan error, 1)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		errCh1 <- pi.Command(ctx, "cmd_one", map[string]any{"arg": 1}, nil)
	}()

	go func() {
		defer wg.Done()
		errCh2 <- pi.Command(ctx, "cmd_two", map[string]any{"arg": 2}, nil)
	}()

	wireReader := NewJSONLines(clientIn, io.Discard)
	raw1, err := wireReader.Read()
	if err != nil {
		t.Fatalf("failed reading pi cmd 1: %v", err)
	}
	raw2, err := wireReader.Read()
	if err != nil {
		t.Fatalf("failed reading pi cmd 2: %v", err)
	}

	var cmd1, cmd2 struct {
		ID   string `json:"id"`
		Type string `json:"type"`
	}
	_ = json.Unmarshal(raw1, &cmd1)
	_ = json.Unmarshal(raw2, &cmd2)

	// Reply out of order: cmd2 first, then cmd1
	resp2 := fmt.Sprintf("{\"type\":\"response\",\"id\":%q,\"command\":%q,\"success\":true}\n", cmd2.ID, cmd2.Type)
	resp1 := fmt.Sprintf("{\"type\":\"response\",\"id\":%q,\"command\":%q,\"success\":true}\n", cmd1.ID, cmd1.Type)

	_, _ = out.Write([]byte(resp2))
	_, _ = out.Write([]byte(resp1))

	wg.Wait()

	if err := <-errCh1; err != nil {
		t.Fatalf("cmd_one failed: %v", err)
	}
	if err := <-errCh2; err != nil {
		t.Fatalf("cmd_two failed: %v", err)
	}
}

func TestPiRPCPipeClosureDuringCommand(t *testing.T) {
	in, out := io.Pipe()
	written := &synchronizedBuffer{}
	pi := NewPiRPC(in, written)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() { _ = pi.Pump(ctx) }()

	errCh := make(chan error, 1)
	go func() {
		errCh <- pi.Command(ctx, "hang_cmd", map[string]any{}, nil)
	}()

	deadline := time.After(2 * time.Second)
	for {
		if written.Len() > 0 {
			break
		}
		select {
		case <-deadline:
			t.Fatal("command not written")
		default:
			time.Sleep(5 * time.Millisecond)
		}
	}

	_ = in.Close()
	_ = out.Close()

	select {
	case err := <-errCh:
		if err == nil {
			t.Fatal("expected error on pipe closure, got nil")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("command remained blocked after pipe closure")
	}
}
