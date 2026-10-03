package runtime

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/config"
)

const terminalTraceEnv = "NEXUS_TERMINAL_TRACE"

var terminalTraceMu sync.Mutex

type terminalTraceRecord struct {
	At        string `json:"at"`
	RuntimeID string `json:"runtime_id"`
	Stage     string `json:"stage"`
	Size      int    `json:"size"`
	Shape     string `json:"shape"`
}

// TerminalTraceEnabled reports whether byte-path diagnostics are explicitly
// enabled. Trace is off by default and never records raw prompt text.
func TerminalTraceEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(terminalTraceEnv))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// TerminalTracePath returns the per-runtime JSONL trace path.
func TerminalTracePath(runtimeID string) string {
	dataDir, err := config.DataDir()
	if err != nil || strings.TrimSpace(dataDir) == "" {
		dataDir = filepath.Join(os.TempDir(), "nexus")
	}
	return filepath.Join(dataDir, "terminal-trace", sanitizeTraceToken(runtimeID)+".jsonl")
}

// TraceTerminalBytes records only a masked byte signature. Printable prompt
// content is replaced with dots; terminal syntax/control bytes remain visible.
func TraceTerminalBytes(runtimeID, stage string, data []byte) {
	if !TerminalTraceEnabled() || len(data) == 0 {
		return
	}
	traceTerminalRecord(runtimeID, stage, len(data), TerminalTraceShape(data))
}

// TraceTerminalShape records a shape already masked by a trusted frontend.
// It exists solely to correlate xterm/onData with server-side stages.
func TraceTerminalShape(runtimeID, stage, shape string, size int) {
	if !TerminalTraceEnabled() || strings.TrimSpace(shape) == "" {
		return
	}
	if size < 0 {
		size = 0
	}
	if len(shape) > 2048 {
		shape = shape[:2048] + "…"
	}
	traceTerminalRecord(runtimeID, stage, size, shape)
}

// TerminalTraceShape exposes terminal control structure without retaining
// ordinary user/provider text. Digits are masked and non-terminal printable
// bytes become dots.
func TerminalTraceShape(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	const maxBytes = 256
	limit := len(data)
	truncated := false
	if limit > maxBytes {
		limit = maxBytes
		truncated = true
	}

	const (
		traceNormal = iota
		traceAfterESC
		traceCSI
		traceSS3
		traceOSC
		traceOSCEscape
	)
	state := traceNormal
	var b strings.Builder
	for index := 0; index < limit; index++ {
		value := data[index]

		switch state {
		case traceAfterESC:
			switch value {
			case '[':
				b.WriteByte('[')
				state = traceCSI
			case 'O':
				b.WriteByte('O')
				state = traceSS3
			case ']':
				b.WriteByte(']')
				state = traceOSC
			default:
				appendMaskedTraceByte(&b, value)
				state = traceNormal
			}
			continue
		case traceCSI:
			if value >= '0' && value <= '9' {
				b.WriteByte('#')
				continue
			}
			if value >= 0x40 && value <= 0x7e {
				b.WriteByte(value)
				state = traceNormal
				continue
			}
			if value >= 0x20 && value <= 0x3f {
				b.WriteByte(value)
				continue
			}
			appendMaskedTraceByte(&b, value)
			continue
		case traceSS3:
			if value >= 0x20 && value <= 0x7e {
				b.WriteByte(value)
			} else {
				appendMaskedTraceByte(&b, value)
			}
			state = traceNormal
			continue
		case traceOSC:
			switch value {
			case 0x07:
				b.WriteString("<07>")
				state = traceNormal
			case 0x1b:
				b.WriteString("<1b>")
				state = traceOSCEscape
			default:
				appendMaskedTraceByte(&b, value)
			}
			continue
		case traceOSCEscape:
			if value == '\\' {
				b.WriteByte('\\')
				state = traceNormal
			} else {
				appendMaskedTraceByte(&b, value)
				state = traceOSC
			}
			continue
		}

		if value == 0x1b {
			b.WriteString("<1b>")
			state = traceAfterESC
			continue
		}
		if value == '[' && looksLikeBareTerminalSequence(data[index:limit]) {
			b.WriteByte('[')
			state = traceCSI
			continue
		}
		appendMaskedTraceByte(&b, value)
	}
	if truncated {
		b.WriteString("…")
	}
	return b.String()
}

func appendMaskedTraceByte(b *strings.Builder, value byte) {
	switch {
	case value < 0x20 || value == 0x7f:
		fmt.Fprintf(b, "<%02x>", value)
	case value >= 0x20 && value <= 0x7e:
		b.WriteByte('.')
	default:
		b.WriteByte('*')
	}
}

func looksLikeBareTerminalSequence(data []byte) bool {
	if len(data) < 2 || data[0] != '[' {
		return false
	}
	next := data[1]
	if (next >= '0' && next <= '9') || next == '?' || next == '<' || next == '[' {
		return true
	}
	switch next {
	case 'A', 'B', 'C', 'D', 'H', 'F', 'I', 'O', 'R', 'M', 'm':
		return true
	default:
		return false
	}
}

type terminalTracingWriter struct {
	out       io.Writer
	runtimeID string
	stage     string
}

func (w terminalTracingWriter) Write(data []byte) (int, error) {
	TraceTerminalBytes(w.runtimeID, w.stage, data)
	if w.out == nil {
		return len(data), nil
	}
	return w.out.Write(data)
}

// NewTerminalTracingWriter instruments a byte boundary without changing bytes.
func NewTerminalTracingWriter(out io.Writer, runtimeID, stage string) io.Writer {
	return terminalTracingWriter{out: out, runtimeID: runtimeID, stage: stage}
}

func traceTerminalRecord(runtimeID, stage string, size int, shape string) {
	path := TerminalTracePath(runtimeID)
	terminalTraceMu.Lock()
	defer terminalTraceMu.Unlock()

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer f.Close()

	record := terminalTraceRecord{
		At:        time.Now().UTC().Format(time.RFC3339Nano),
		RuntimeID: runtimeID,
		Stage:     sanitizeTraceToken(stage),
		Size:      size,
		Shape:     shape,
	}
	encoded, err := json.Marshal(record)
	if err != nil {
		return
	}
	_, _ = f.Write(append(encoded, '\n'))
}

func sanitizeTraceToken(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown"
	}
	var b strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9',
			r == '-', r == '_', r == '.':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	if b.Len() == 0 {
		return "unknown"
	}
	return b.String()
}
