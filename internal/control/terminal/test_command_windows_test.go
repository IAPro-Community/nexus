//go:build windows

package terminal

import "os/exec"

func testEchoCommand(value string) *exec.Cmd {
	return exec.Command("cmd.exe", "/D", "/Q", "/C", "echo", value)
}

func testCatCommand() *exec.Cmd {
	return exec.Command("cmd.exe", "/D", "/Q", "/C", "more")
}
