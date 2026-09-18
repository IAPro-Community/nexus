//go:build !windows

package terminal

import (
	"bytes"
	"strings"
	"testing"

	"github.com/creack/pty"
)

func TestTTYControllerResetsOnEnterReassertAndRestore(t *testing.T) {
	master, slave, err := pty.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer master.Close()
	defer slave.Close()

	var output bytes.Buffer
	controller := NewTTYController(slave, &output)
	if err := controller.Enter(); err != nil {
		t.Fatal(err)
	}
	if err := controller.Reassert(); err != nil {
		t.Fatal(err)
	}
	if err := controller.Restore(); err != nil {
		t.Fatal(err)
	}

	if got := strings.Count(output.String(), KeyboardModeReset); got != 3 {
		t.Fatalf("reset count=%d, want 3; output=%q", got, output.String())
	}
}
