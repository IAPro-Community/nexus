package agentwire

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"sync"
	"sync/atomic"
)

// PiRPC handles Pi/Oh My Pi newline-delimited RPC records. It must never be
// treated as JSON-RPC 2.0; records use 'type' and 'command', not 'method'.
type PiRPC struct {
	wire     *JSONLines
	sequence atomic.Uint64
	mu       sync.Mutex
	pending  map[string]chan piReply
	onEvent  func(string, json.RawMessage)
	done     chan struct{}
	once     sync.Once
	closed   bool
}
type piReply struct {
	Command string          `json:"command"`
	Success bool            `json:"success"`
	Error   string          `json:"error"`
	Data    json.RawMessage `json:"data"`
}

func NewPiRPC(input io.Reader, output io.Writer) *PiRPC {
	return &PiRPC{wire: NewJSONLines(input, output), pending: make(map[string]chan piReply), done: make(chan struct{})}
}
func (p *PiRPC) OnEvent(callback func(string, json.RawMessage)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.onEvent = callback
}
func (p *PiRPC) Command(ctx context.Context, command string, fields map[string]any, reply *json.RawMessage) error {
	if command == "" {
		return errors.New("Pi RPC: empty command")
	}
	id := strconv.FormatUint(p.sequence.Add(1), 10)
	ch := make(chan piReply, 1)
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return ErrClosed
	}
	p.pending[id] = ch
	p.mu.Unlock()
	defer func() { p.mu.Lock(); delete(p.pending, id); p.mu.Unlock() }()
	msg := map[string]any{"type": command, "id": id}
	for k, v := range fields {
		if k == "id" || k == "type" {
			return errors.New("Pi RPC: reserved field")
		}
		msg[k] = v
	}
	if err := p.wire.Write(msg); err != nil {
		return err
	}
	select {
	case r := <-ch:
		if r.Command != command { return errors.New("Pi RPC: mismatched command response") }
  if !r.Success { return errors.New("Pi RPC: command rejected") }
		if reply != nil {
			*reply = r.Data
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-p.done:
		return ErrClosed
	}
}

// RespondUI replies to a provider-origin RPC extension dialog. The ID is
// correlated by the bridge; it must never be guessed from terminal narration.
func (p *PiRPC) RespondUI(requestID string, fields map[string]any)error{
 if requestID=="" {return errors.New("Pi RPC: missing UI request ID")}
 msg:=map[string]any{"type":"extension_ui_response","id":requestID}
 for key,value:=range fields{
  if key=="id"||key=="type"{return errors.New("Pi RPC: reserved response field")}
  msg[key]=value
 }
 return p.wire.Write(msg)
}

// Pump records events independently of command replies; 'agent_end' is not a
// settled session. Consumers should wait for 'agent_settled' instead.
func (p *PiRPC) Pump(ctx context.Context) error {
	defer p.Close()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		raw, err := p.wire.Read()
		if err != nil {
			return err
		}
		var env struct {
			Type    string `json:"type"`
			ID      string `json:"id"`
			Command string `json:"command"`
		}
		if err := json.Unmarshal(raw, &env); err != nil {
			return err
		}
		if env.Type == "response" {
			var reply piReply
			if err := json.Unmarshal(raw, &reply); err != nil {
				return err
			}
			p.mu.Lock()
			ch := p.pending[env.ID]
			p.mu.Unlock()
			if ch != nil {
				select {
				case ch <- reply:
				default:
				}
			}
			continue
		}
		p.mu.Lock()
		callback := p.onEvent
		p.mu.Unlock()
		if callback != nil {
			callback(env.Type, raw)
		}
	}
}
func (p *PiRPC) Close() {
	p.once.Do(func() { p.mu.Lock(); p.closed = true; p.mu.Unlock(); p.wire.Close(); close(p.done) })
}
