//go:build !windows

package host

import "os"

var testTerminalBinary = "cat"

func testTerminalArgs() []string { return nil }

var interactiveTestTerminalBinary string

func init() {
	if exe, err := os.Executable(); err == nil {
		interactiveTestTerminalBinary = exe
	} else {
		interactiveTestTerminalBinary = "sh"
	}
}

func interactiveTestTerminalArgs() []string {
	if interactiveTestTerminalBinary != "sh" {
		return []string{"-nexus-echo-helper"}
	}
	return []string{"-c", "printf 'NEXUS_TEST_READY\\n'; exec cat"}
}
