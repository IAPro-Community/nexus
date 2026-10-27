package agentwire

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"sync"
	"sync/atomic"
)

// ACP is a JSON-RPC 2.0 stream with bidirectional method calls.
// All outbound requests have one reader and one coordinator. No independent
// callers are permitted to read provider stdout.
type ACP struct {
	wire      *JSONLines
	sequence  atomic.Uint64
	mu        sync.Mutex
	pending   map[string]chan rpcReply
	handler   func(context.Context, string, json.RawMessage) (any, error)
	notify    func(string, json.RawMessage)
	closed    bool
	closeOnce sync.Once
	done      chan struct{}
}
type rpcReply struct {
	Result json.RawMessage `json:"result"`
	Error  *rpcError       `json:"error"`
}
type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *rpcError) Error() string { return fmt.Sprintf("ACP error %d: %s", e.Code, e.Message) }
func NewACP(input io.Reader, output io.Writer) *ACP {
	return &ACP{wire: NewJSONLines(input, output), pending: make(map[string]chan rpcReply), done: make(chan struct{})}
}

// OnRequest is deliberately optional. Without an installed handler, all
// server-initiated methods (including filesystem and permission requests) fail
// closed with -32601. Install a policy-enforcing handler before Pump.
func (a *ACP) OnRequest(f func(context.Context, string, json.RawMessage) (any, error)) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.handler = f
}
func (a *ACP) OnNotification(f func(string, json.RawMessage)) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.notify = f
}
func (a *ACP) Done() <-chan struct{} { return a.done }

func (a *ACP) Call(ctx context.Context, method string, params any, result any) error {
	if method == "" {
		return errors.New("ACP: empty method")
	}
	id := strconv.FormatUint(a.sequence.Add(1), 10)
	replyCh := make(chan rpcReply, 1)
	a.mu.Lock()
	if a.closed {
		a.mu.Unlock()
		return ErrClosed
	}
	a.pending[id] = replyCh
	a.mu.Unlock()
	defer func() { a.mu.Lock(); delete(a.pending, id); a.mu.Unlock() }()
	if err := a.wire.Write(map[string]any{"jsonrpc": "2.0", "id": id, "method": method, "params": params}); err != nil {
		return err
	}
	select {
	case reply := <-replyCh:
		if reply.Error != nil {
			return reply.Error
		}
		if result == nil || len(reply.Result) == 0 {
			return nil
		}
		return json.Unmarshal(reply.Result, result)
	case <-ctx.Done():
		return ctx.Err()
	case <-a.done:
		return ErrClosed
	}
}
func (a *ACP) Notify(method string, params any) error {
	if method == "" {
		return errors.New("ACP: empty method")
	}
	return a.wire.Write(map[string]any{"jsonrpc": "2.0", "method": method, "params": params})
}

// Pump MUST be running before Call. Concurrent Call and Notify are safe.
// Handlers execute asynchronously, allowing inbound permission requests while
// a prompt is outstanding. Callback owners must enforce auth and bound work.
func (a *ACP) Pump(ctx context.Context) error {
	defer a.Close()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		frame, err := a.wire.Read()
		if err != nil {
			return err
		}
		var envelope struct {
			Version string          `json:"jsonrpc"`
			ID      json.RawMessage `json:"id"`
			Method  string          `json:"method"`
			Params  json.RawMessage `json:"params"`
			Result  json.RawMessage `json:"result"`
			Error   *rpcError       `json:"error"`
		}
		if err := json.Unmarshal(frame, &envelope); err != nil {
			return err
		}
		if envelope.Version != "2.0" {
			return errors.New("ACP: unsupported JSON-RPC version")
		}
		if envelope.Method != "" {
			a.mu.Lock()
			handler := a.handler
			notification := a.notify
			a.mu.Unlock()
			if len(envelope.ID) > 0 {
				// json.RawMessage must be preserved unchanged: ACP uses string or number IDs.
				go func(id json.RawMessage, method string, params json.RawMessage) {
					var value any
					var handlerErr error
					if handler == nil {
						handlerErr = &rpcError{Code: -32601, Message: "method unavailable"}
					} else {
						value, handlerErr = handler(ctx, method, params)
					}
					response := map[string]any{"jsonrpc": "2.0", "id": id}
					if handlerErr != nil {
						if e, ok := handlerErr.(*rpcError); ok {
							response["error"] = e
						} else {
							response["error"] = rpcError{Code: -32000, Message: "request rejected"}
						}
					} else {
						response["result"] = value
					}
					_ = a.wire.Write(response)
				}(envelope.ID, envelope.Method, envelope.Params)
			} else if notification != nil {
				notification(envelope.Method, envelope.Params)
			}
			continue
		}
		if len(envelope.ID) == 0 {
			continue
		}
		// Correlate numeric and quoted string IDs into the same canonical key.
		key := string(envelope.ID)
		if len(key) > 0 && key[0] == '"' {
			if id, err := strconv.Unquote(key); err == nil {
				key = id
			}
		}
		a.mu.Lock()
		wait := a.pending[key]
		a.mu.Unlock()
		if wait != nil {
			select {
			case wait <- rpcReply{Result: envelope.Result, Error: envelope.Error}:
			default:
			}
		}
	}
}
func (a *ACP) Close() {
	a.closeOnce.Do(func() {
		a.mu.Lock()
		a.closed = true
		a.mu.Unlock()
		a.wire.Close()
		close(a.done)
	})
}

// InitResult deliberately retains capability JSON: feature gates must depend on
// negotiated provider results rather than version-string heuristics.
type InitResult struct {
	ProtocolVersion   int             `json:"protocolVersion"`
	AgentCapabilities json.RawMessage `json:"agentCapabilities"`
	AuthMethods       json.RawMessage `json:"authMethods"`
}

func (a *ACP) Initialize(ctx context.Context, protocolVersion int) (InitResult, error) {
	var out InitResult
	err := a.Call(ctx, "initialize", map[string]any{
		"protocolVersion":    protocolVersion,
		"clientCapabilities": map[string]any{"fs": map[string]bool{"readTextFile": false, "writeTextFile": false}, "terminal": false},
		"clientInfo":         map[string]any{"name": "nexus", "version": "1.0.0"},
	}, &out)
	return out, err
}
func (a *ACP) NewSession(ctx context.Context, cwd string) (string, error) {
	var result struct {
		SessionID string `json:"sessionId"`
	}
	if cwd == "" {
		return "", errors.New("ACP: session needs a working directory")
	}
	err := a.Call(ctx, "session/new", map[string]any{"cwd": cwd, "mcpServers": []any{}}, &result)
	if err != nil {
		return "", err
	}
	if result.SessionID == "" {
		return "", errors.New("ACP: session/new missing sessionId")
	}
	return result.SessionID, nil
}
func (a *ACP) Prompt(ctx context.Context, sessionID, text string) (json.RawMessage, error) {
	if sessionID == "" {
		return nil, errors.New("ACP: missing sessionId")
	}
	var result json.RawMessage
	err := a.Call(ctx, "session/prompt", map[string]any{"sessionId": sessionID, "prompt": []map[string]string{{"type": "text", "text": text}}}, &result)
	return result, err
}
func (a *ACP) Cancel(sessionID string) error {
	if sessionID == "" {
		return errors.New("ACP: missing sessionId")
	}
	return a.Notify("session/cancel", map[string]string{"sessionId": sessionID})
}
