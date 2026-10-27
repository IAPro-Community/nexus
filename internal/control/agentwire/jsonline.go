// Package agentwire implements version-neutral framed transports used by Nexus
// harness adapters. It intentionally knows neither models nor agent planning.
package agentwire

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
)

const MaxFrame = 4 << 20 // Bound untrusted subprocess output to 4 MiB/frame.

var ErrFrameTooLarge = errors.New("agentwire: frame exceeds maximum size")
var ErrClosed = errors.New("agentwire: connection closed")

// JSONLines implements the byte-framing common to ACP JSON-RPC and Pi RPC.
// This is LF-only: U+2028 and U+2029 inside a JSON string are not separators.
type JSONLines struct {
	reader *bufio.Reader
	writer io.Writer
	mu     sync.Mutex
	closed bool
}

func NewJSONLines(input io.Reader, output io.Writer) *JSONLines {
	return &JSONLines{reader: bufio.NewReader(input), writer: output}
}

func (j *JSONLines) Read() (json.RawMessage, error) {
	// ReadSlice with bounded buffering: a malicious process must not allocate
	// arbitrarily large lines, even before a newline arrives.
	var out []byte
	for {
		frag, err := j.reader.ReadSlice('\n')
		if len(out)+len(frag) > MaxFrame {
			return nil, ErrFrameTooLarge
		}
		out = append(out, frag...)
		if err == bufio.ErrBufferFull {
			continue
		}
		if err != nil {
			if err == io.EOF && (len(out) == 0 || len(bytes.TrimSpace(out)) == 0) {
				return nil, io.EOF
			}
			return nil, fmt.Errorf("agentwire: incomplete record: %w", err)
		}
		trimmed := bytes.TrimSpace(out)
		if len(trimmed) == 0 {
			out = out[:0]
			continue
		}
		if !json.Valid(trimmed) {
			return nil, errors.New("agentwire: invalid JSON line")
		}
		return json.RawMessage(trimmed), nil
	}
}

func (j *JSONLines) Write(value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if len(data) > MaxFrame {
		return ErrFrameTooLarge
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.closed {
		return ErrClosed
	}
	// Lock protects records from being interleaved by simultaneous calls.
	buf := make([]byte, 0, len(data)+1)
	buf = append(buf, data...)
	buf = append(buf, '\n')
	n, err := j.writer.Write(buf)
	if err != nil {
		return err
	}
	if n != len(buf) {
		return io.ErrShortWrite
	}
	return nil
}

// Close marks output closed but does not close stdin/stdout owned by the host.
func (j *JSONLines) Close() {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.closed = true
}
