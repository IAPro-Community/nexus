package terminal

import (
	"errors"
	"fmt"
	"io"
	"os"
	"sync"

	"golang.org/x/term"
)

// KeyboardModeReset is deliberately conservative. It only disables private
// modes that provider TUIs commonly leave enabled after an interrupt; it does
// not rewrite user input and it never travels to the provider PTY.
//
// The sequence is safe to send repeatedly. That property is important because
// recovery can be requested by Ctrl+C, a broken attach stream, and a deferred
// cleanup at the same time.
const KeyboardModeReset = "\x1b[?1l\x1b[>0u\x1b[>4;0m\x1b[?1004l\x1b[?2004l"

// ResetKeyboardModes disables application cursor, Kitty/CSI-u keyboard
// enhancement, modifyOtherKeys, focus reporting, and bracketed paste on the
// terminal represented by w. The writer is normally the user's stdout, never
// the provider's PTY.
func ResetKeyboardModes(w io.Writer) error {
	if w == nil {
		return errors.New("terminal reset writer is nil")
	}
	_, err := io.WriteString(w, KeyboardModeReset)
	if err != nil {
		return fmt.Errorf("reset terminal keyboard modes: %w", err)
	}
	return nil
}

// TTYController owns the complete lifecycle of the user's local TTY. A
// provider runtime has its own PTY and is intentionally outside this state
// machine. Keeping the ownership here prevents the direct and supervised
// attach paths from growing subtly different cleanup behavior.
type TTYController struct {
	mu       sync.Mutex
	input    *os.File
	output   io.Writer
	original *term.State
	active   bool
}

func NewTTYController(input *os.File, output io.Writer) *TTYController {
	return &TTYController{input: input, output: output}
}

// Enter captures the original terminal state, switches to raw mode, and
// applies the safe keyboard reset before a runtime is started or attached.
// Non-TTY input is a valid direct/headless path and remains a no-op.
func (c *TTYController) Enter() error {
	if c == nil || c.input == nil || !term.IsTerminal(int(c.input.Fd())) {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.active {
		return c.resetLocked()
	}

	state, err := term.MakeRaw(int(c.input.Fd()))
	if err != nil {
		return fmt.Errorf("enter raw terminal mode: %w", err)
	}
	c.original = state
	c.active = true
	if err := c.resetLocked(); err != nil {
		_ = term.Restore(int(c.input.Fd()), state)
		c.original = nil
		c.active = false
		return err
	}
	return nil
}

// Reassert reapplies the safe reset after an interrupt or a broken stream.
// It is intentionally idempotent and does not alter the bytes sent to the
// provider.
func (c *TTYController) Reassert() error {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.active {
		return nil
	}
	return c.resetLocked()
}

// Restore first disables provider private modes and then restores the exact
// termios state captured on Enter. The reset-before-restore ordering ensures
// that a provider cannot leave the shell in application cursor/focus/paste
// mode even when it exits unexpectedly.
func (c *TTYController) Restore() error {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.active {
		return nil
	}

	resetErr := c.resetLocked()
	restoreErr := term.Restore(int(c.input.Fd()), c.original)
	c.original = nil
	c.active = false
	if resetErr != nil {
		return resetErr
	}
	if restoreErr != nil {
		return fmt.Errorf("restore terminal state: %w", restoreErr)
	}
	return nil
}

// Close is suitable for defer and deliberately ignores cleanup failures.
// Call Restore directly when the caller needs to report an error.
func (c *TTYController) Close() error {
	return c.Restore()
}

func (c *TTYController) resetLocked() error {
	return ResetKeyboardModes(c.output)
}
