//go:build !windows

package terminal

import "os/exec"

func testEchoCommand(value string) *exec.Cmd { return exec.Command("echo", value) }

func testCatCommand() *exec.Cmd { return exec.Command("cat") }
