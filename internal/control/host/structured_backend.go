package host

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"github.com/IAPro-Community/nexus/internal/control/agentwire"
	"github.com/IAPro-Community/nexus/internal/core/security"
)

// structuredBackend gives the existing SessionHost a terminal-shaped control
// surface while retaining a single owned child process on stdin/stdout pipes.
// Protocol JSON is never echoed to the terminal ring buffer.
type structuredBackend struct {
	mode          string
	cwd           string
	cmd           *exec.Cmd
	outputReader  *io.PipeReader
	outputWriter  *io.PipeWriter
	bridge        *agentwire.Bridge
	onStatus      func(agentwire.WireSignal)
	onSession     func(string)
	mu            sync.Mutex
	input         bytes.Buffer
	ctx           context.Context
	cancel        context.CancelFunc
	started       bool
	ready         chan error
	lines         chan string
	statusUpdates chan agentwire.WireSignal
	inputLines    chan string
	escapeState   uint8
	discardInput  bool
}

func newStructuredBackend(mode, cwd string, onStatus func(agentwire.WireSignal), onSession func(string)) *structuredBackend {
	r, w := io.Pipe()
	return &structuredBackend{mode: mode, cwd: cwd, outputReader: r, outputWriter: w, onStatus: onStatus, onSession: onSession, ready: make(chan error, 1), lines: make(chan string, 256), statusUpdates: make(chan agentwire.WireSignal, 512), inputLines: make(chan string, 64)}
}
func (b *structuredBackend) emit(msg string) {
	if msg == "" {
		return
	}
	msg = security.Redact(msg)
	// Agent output is untrusted text, not a VT command channel.
	msg = strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r == '\t' {
			return r
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, msg)
	chars := []rune(msg)
	if len(chars) > 4096 {
		msg = string(chars[:4096])
	}
	// A full queue must not stall ACP JSON-RPC input processing. Provider
	// dialog remains recoverable from its session; this is display-only.
	select {
	case b.lines <- msg:
	default:
	}
}
func (b *structuredBackend) Start(cmd *exec.Cmd, rows, cols int) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.started {
		return errors.New("structured backend already started")
	}
	if cmd == nil {
		return errors.New("missing provider command")
	}
	b.cmd = cmd
	in, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	bridge, err := agentwire.NewBridge(b.mode, out, in, b.cwd, func(signal agentwire.WireSignal) {
		// Preserve provider event order. Never silently drop an approval when
		// the bounded stream is saturated; terminate this unhealthy session.
		select {
		case b.statusUpdates <- signal:
		default:
			b.emit("\r\n[Nexus] Agent status overflow; stopping session.\r\n")
			// Do not touch b.cancel here: the protocol reader may run while
			// Start is still installing context/ownership under b.mu.
			if cmd.Process != nil {
				_ = cmd.Process.Kill()
			}
		}
	}, func(msg string) { b.emit(msg) })
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	b.ctx, b.cancel = context.WithCancel(context.Background())
	b.bridge = bridge
	b.started = true
	go b.dispatchInputs()
	go func() {
		for {
			select {
			case <-b.ctx.Done():
				return
			case signal := <-b.statusUpdates:
				if b.onStatus != nil {
					b.onStatus(signal)
				}
			}
		}
	}()
	go func() {
		for {
			select {
			case <-b.ctx.Done():
				return
			case line := <-b.lines:
				_, _ = io.WriteString(b.outputWriter, line)
			}
		}
	}()
	// Do not block the IPC or hold SessionHost.mu during protocol handshake.
	// A failed handshake stops the child rather than leaving a fake session.
	go func() {
		if err := bridge.Start(b.ctx); err != nil {
			b.ready <- err
			b.emit("\r\n[Nexus] Structured protocol handshake failed.\r\n")
			_ = cmd.Process.Kill()
			return
		}
		b.ready <- nil
		if b.onSession != nil && bridge.SessionID() != "" {
			b.onSession(bridge.SessionID())
		}
		b.emit("\r\n[Nexus] Connected to " + strings.ToUpper(b.mode) + " agent. Enter a prompt.\r\n")
	}()
	go func() {
		// Diagnostics may contain credentials: never show raw stderr in UI or
		// persist it in the attention registry.
		_, _ = io.Copy(io.Discard, stderr)
	}()
	// Only report runtime readiness after protocol initialization succeeds.
	if err := <-b.ready; err != nil {
		// SessionHost does not start waitProcess after a failed backend Start.
		// Reap this child here so handshake errors cannot accumulate zombies.
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		_ = cmd.Wait()
		if b.cancel != nil {
			b.cancel()
		}
		_ = b.outputWriter.Close()
		return fmt.Errorf("agent protocol unavailable: %w", err)
	}
	return nil
}
func (b *structuredBackend) Read(p []byte) (int, error) { return b.outputReader.Read(p) }

// Write consumes the text-only command surface. A raw PTY keystroke is not
// itself an operator approval; requests must carry their native interaction ID.
// Input is never mirrored back to the terminal (it may contain secrets).
func (b *structuredBackend) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.started || b.ctx == nil {
		return 0, errors.New("structured backend not started")
	}
	for _, c := range p {
		// Drop xterm cursor/control sequences rather than forwarding "[A"
		// as the literal start of a model prompt.
		if b.escapeState != 0 {
			switch b.escapeState {
			case 1:
				if c == '[' || c == 'O' {
					b.escapeState = 2
				} else {
					b.escapeState = 0
				}
			case 2:
				if c >= 0x40 && c <= 0x7e {
					b.escapeState = 0
				}
			}
			continue
		}
		if c == 0x1b {
			b.escapeState = 1
			continue
		}
		if c == 0x7f || c == '\b' {
			raw := b.input.Bytes()
			if len(raw) != 0 {
				_, length := utf8.DecodeLastRune(raw)
				b.input.Truncate(len(raw) - length)
			}
			continue
		}
		if c == '\r' || c == '\n' {
			line := strings.TrimSpace(b.input.String())
			tooLong := b.discardInput
			b.input.Reset()
			b.discardInput = false
			if tooLong {
				b.emit("\r\n[Nexus] Input too long; discarded.\r\n")
				continue
			}
			if line == "" {
				continue
			}
			select {
			case b.inputLines <- line:
				b.emit("\r\n[Nexus] Input submitted.\r\n")
			default:
				return 0, errors.New("structured input queue full; retry after previous commands complete")
			}
			continue
		}
		if c < 0x20 {
			continue
		}
		if b.discardInput {
			continue
		}
		if b.input.Len() >= 16384 {
			b.input.Reset()
			b.discardInput = true
			continue
		}
		_ = b.input.WriteByte(c)
	}
	return len(p), nil
}

// dispatchInputs keeps command ordering deterministic while allowing the
// ongoing provider prompt to await a later /permit, /answer, or /cancel.
func (b *structuredBackend) dispatchInputs() {
	promptActive := false
	completed := make(chan struct{}, 1)
	for {
		select {
		case <-b.ctx.Done():
			return
		case <-completed:
			promptActive = false
		case line := <-b.inputLines:
			if strings.HasPrefix(line, "/") {
				b.execute(b.ctx, b.bridge, line)
				continue
			}
			if promptActive {
				b.emit("\r\n[Nexus] Agent is already processing a prompt.\r\n")
				continue
			}
			promptActive = true
			go func(prompt string) {
				b.execute(b.ctx, b.bridge, prompt)
				select {
				case completed <- struct{}{}:
				case <-b.ctx.Done():
				}
			}(line)
		}
	}
}
func (b *structuredBackend) execute(ctx context.Context, bridge *agentwire.Bridge, line string) {
	if line == "/cancel" {
		requestCtx, cancel := context.WithTimeout(ctx, 15*1000000000)
		defer cancel()
		if err := bridge.CancelTurn(requestCtx); err != nil {
			b.emit("\r\n[Nexus] Cancel request failed.\r\n")
		} else {
			b.emit("\r\n[Nexus] Cancel requested.\r\n")
		}
		return
	}
	if strings.HasPrefix(line, "/permit ") {
		parts := strings.SplitN(line, " ", 3)
		if len(parts) != 3 {
			b.emit("\r\n[Nexus] Usage: /permit <request-id> <option-id>\r\n")
			return
		}
		if err := bridge.Choose(parts[1], parts[2]); err != nil {
			b.emit("\r\n[Nexus] Invalid or expired permission selection.\r\n")
		} else {
			b.emit("\r\n[Nexus] Permission response submitted; awaiting agent status.\r\n")
		}
		return
	}
	if strings.HasPrefix(line, "/answer ") {
		parts := strings.SplitN(line, " ", 3)
		if len(parts) != 3 {
			b.emit("\r\n[Nexus] Usage: /answer <request-id> <text>\r\n")
			return
		}
		if err := bridge.ReplyText(parts[1], parts[2]); err != nil {
			b.emit("\r\n[Nexus] Invalid or expired text request.\r\n")
		} else {
			b.emit("\r\n[Nexus] Response submitted.\r\n")
		}
		return
	}
	if strings.HasPrefix(line, "/") {
		b.emit("\r\n[Nexus] Unknown structured command.\r\n")
		return
	}
	if err := bridge.Prompt(ctx, line); err != nil {
		b.emit("\r\n[Nexus] Prompt could not be delivered: " + security.Redact(err.Error()) + "\r\n")
	}
}
func (b *structuredBackend) Close() error {
	b.mu.Lock()
	cancel := b.cancel
	bridge := b.bridge
	b.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if bridge != nil {
		bridge.Stop()
	}
	_ = b.outputWriter.Close()
	return b.outputReader.Close()
}
func (b *structuredBackend) Wait() error {
	b.mu.Lock()
	cmd := b.cmd
	b.mu.Unlock()
	if cmd == nil {
		return errors.New("provider not started")
	}
	err := cmd.Wait()
	_ = b.outputWriter.Close()
	return err
}
func (b *structuredBackend) WaitContext(ctx context.Context) error {
	done := make(chan error, 1)
	go func() { done <- b.Wait() }()
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		_ = b.Kill()
		<-done
		return ctx.Err()
	}
}
func (b *structuredBackend) Signal(sig os.Signal) error {
	b.mu.Lock()
	cmd := b.cmd
	b.mu.Unlock()
	if cmd == nil || cmd.Process == nil {
		return errors.New("provider not running")
	}
	return cmd.Process.Signal(sig)
}
func (b *structuredBackend) Kill() error {
	b.mu.Lock()
	cmd := b.cmd
	cancel := b.cancel
	b.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if cmd == nil || cmd.Process == nil {
		return errors.New("provider not running")
	}
	return cmd.Process.Kill()
}
func (b *structuredBackend) PID() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.cmd == nil || b.cmd.Process == nil {
		return 0
	}
	return b.cmd.Process.Pid
}
func (*structuredBackend) Resize(rows, cols int) error { return nil }
func (*structuredBackend) SupportsResize() bool        { return false }
func (*structuredBackend) SupportsRawMode() bool       { return false }
func (b *structuredBackend) Mechanism() string {
	return fmt.Sprintf("Nexus supervised %s stdin/stdout JSONL", b.mode)
}
func (*structuredBackend) Supervise() error { return nil }

// The current terminal abstraction has no native keep-alive callback. A
// protocol bridge is always tied to the SessionHost process and provider PID.
var _ interface {
	io.Reader
	io.Writer
	io.Closer
	Start(*exec.Cmd, int, int) error
	Resize(int, int) error
	PID() int
	Wait() error
	WaitContext(context.Context) error
	Signal(os.Signal) error
	Kill() error
	SupportsResize() bool
	SupportsRawMode() bool
	Mechanism() string
	Supervise() error
} = (*structuredBackend)(nil)
