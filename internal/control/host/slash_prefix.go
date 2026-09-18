package host

import (
	"bytes"
	"strings"
)

type PrefixState int

const (
	StateIdle PrefixState = iota
	StateBuffering
	StateControlCommand
	StatePassthrough
)

type RouterActionType int

const (
	ActionNone RouterActionType = iota
	ActionForwardBytes
	ActionControlCommand
	ActionSuggestions
)

type RouterOutput struct {
	Action       RouterActionType
	ForwardBytes []byte
	ControlCmd   string
	Suggestions  string
}

// SlashPrefixRouter handles input bytes character by character, buffering the
// canonical Nexus control prefix (/nexus, //nexus) and instantly forwarding
// non-matching bytes to the child process.
type SlashPrefixRouter struct {
	state      PrefixState
	prefixBuf  bytes.Buffer
	commandBuf bytes.Buffer
}

// NewSlashPrefixRouter creates an initialized SlashPrefixRouter.
func NewSlashPrefixRouter() *SlashPrefixRouter {
	return &SlashPrefixRouter{state: StateIdle}
}

// Reset resets the internal state machine back to StateIdle.
func (r *SlashPrefixRouter) Reset() {
	r.state = StateIdle
	r.prefixBuf.Reset()
	r.commandBuf.Reset()
}

func isPrefixOfKnown(s string) bool {
	targets := []string{"/nexus", "//nexus", ":nexus", "::nexus"}
	for _, t := range targets {
		if strings.HasPrefix(t, s) {
			return true
		}
	}
	return false
}

// isControlPrefix reports whether the trimmed, lowercased prefix should be
// intercepted as a Nexus control command (not forwarded to the child).
func isControlPrefix(s string) bool {
	switch s {
	case "/nexus", ":nexus":
		return true
	}
	return false
}

// isEscapePrefix reports whether the prefix is a double-delimiter escape
// (e.g. "//nexus", "::ai") that should forward the single-delimiter form
// to the child process.
func isEscapePrefix(s string) bool {
	switch s {
	case "//nexus", "::nexus":
		return true
	}
	return false
}

// strippedEscape removes one leading delimiter from an escape prefix.
// "//nexus" → "/nexus", "::ai" → ":ai"
func strippedEscape(s string) string {
	if len(s) >= 2 && s[0] == s[1] && (s[0] == '/' || s[0] == ':') {
		return s[1:]
	}
	return s
}

// ProcessByte processes a single byte through the prefix state machine.
func (r *SlashPrefixRouter) ProcessByte(b byte) RouterOutput {
	// Deprecated compatibility parser for explicit CmdSlash callers only.
	// SessionHost CmdInput never invokes this function; terminal bytes are
	// forwarded byte-for-byte to the provider PTY.
	if b == 0x03 || b == 0x15 { // Ctrl+C or Ctrl+U
		r.Reset()
		return RouterOutput{Action: ActionForwardBytes, ForwardBytes: []byte{b}}
	}

	switch r.state {
	case StateIdle:
		if b == '/' || b == ':' {
			r.state = StateBuffering
			r.prefixBuf.WriteByte(b)
			return RouterOutput{Action: ActionNone}
		}
		if b == '\r' || b == '\n' {
			return RouterOutput{Action: ActionForwardBytes, ForwardBytes: []byte{b}}
		}
		r.state = StatePassthrough
		return RouterOutput{Action: ActionForwardBytes, ForwardBytes: []byte{b}}

	case StateBuffering:
		if b == '\t' {
			trimmedPrefix := strings.ToLower(strings.TrimSpace(r.prefixBuf.String()))
			if isControlPrefix(trimmedPrefix) {
				r.state = StateControlCommand
				r.commandBuf.Reset()
				r.commandBuf.WriteString(r.prefixBuf.String())
				r.commandBuf.WriteByte(' ')
				r.prefixBuf.Reset()
				return RouterOutput{Action: ActionSuggestions, Suggestions: slashSuggestions(" ")}
			}
		}
		// Check if delimiter encountered
		if b == ' ' || b == '\t' || b == '\r' || b == '\n' {
			trimmedPrefix := strings.ToLower(strings.TrimSpace(r.prefixBuf.String()))
			if isControlPrefix(trimmedPrefix) {
				// Confirmed command prefix
				r.state = StateControlCommand
				r.commandBuf.Reset()
				r.commandBuf.WriteString(r.prefixBuf.String())
				r.commandBuf.WriteByte(b)
				r.prefixBuf.Reset()

				if b == '\r' || b == '\n' {
					cmd := strings.TrimSpace(r.commandBuf.String())
					r.Reset()
					return RouterOutput{Action: ActionControlCommand, ControlCmd: cmd}
				}
				return RouterOutput{Action: ActionNone}
			}

			if isEscapePrefix(trimmedPrefix) {
				r.state = StatePassthrough
				// Strip one leading delimiter: "//" → "/", "::" → ":"
				stripped := strippedEscape(trimmedPrefix)
				out := append([]byte(stripped), b)
				r.prefixBuf.Reset()
				if b == '\r' || b == '\n' {
					r.state = StateIdle
				}
				return RouterOutput{Action: ActionForwardBytes, ForwardBytes: out}
			}

			// Diverged delimiter
			r.state = StatePassthrough
			out := append(r.prefixBuf.Bytes(), b)
			r.prefixBuf.Reset()
			if b == '\r' || b == '\n' {
				r.state = StateIdle
			}
			return RouterOutput{Action: ActionForwardBytes, ForwardBytes: out}
		}

		r.prefixBuf.WriteByte(b)
		candidate := strings.ToLower(r.prefixBuf.String())
		if isPrefixOfKnown(candidate) {
			return RouterOutput{Action: ActionNone}
		}

		// Diverged from all known slash prefixes
		r.state = StatePassthrough
		out := make([]byte, r.prefixBuf.Len())
		copy(out, r.prefixBuf.Bytes())
		r.prefixBuf.Reset()
		return RouterOutput{Action: ActionForwardBytes, ForwardBytes: out}

	case StateControlCommand:
		if b == '\t' {
			return RouterOutput{Action: ActionSuggestions, Suggestions: slashSuggestions(r.commandBuf.String())}
		}
		if b == '\r' || b == '\n' {
			cmd := strings.TrimSpace(r.commandBuf.String())
			r.Reset()
			return RouterOutput{Action: ActionControlCommand, ControlCmd: cmd}
		}
		if b == 0x7f || b == 0x08 { // Backspace
			if r.commandBuf.Len() > 0 {
				buf := r.commandBuf.Bytes()
				r.commandBuf.Reset()
				r.commandBuf.Write(buf[:len(buf)-1])
			}
			return RouterOutput{Action: ActionNone}
		}
		r.commandBuf.WriteByte(b)
		return RouterOutput{Action: ActionNone}

	case StatePassthrough:
		if b == '\r' || b == '\n' {
			r.state = StateIdle
		}
		return RouterOutput{Action: ActionForwardBytes, ForwardBytes: []byte{b}}
	}

	return RouterOutput{Action: ActionForwardBytes, ForwardBytes: []byte{b}}
}

func slashSuggestions(command string) string {
	parts := strings.Fields(strings.ToLower(command))
	if len(parts) <= 1 {
		return "status  usage  accounts  handoff  continue  detach  stop  help"
	}
	options := []string{"status", "usage", "accounts", "handoff", "continue", "detach", "stop", "help"}
	needle := parts[len(parts)-1]
	filtered := make([]string, 0, len(options))
	for _, option := range options {
		if strings.HasPrefix(option, needle) {
			filtered = append(filtered, option)
		}
	}
	return strings.Join(filtered, "  ")
}
