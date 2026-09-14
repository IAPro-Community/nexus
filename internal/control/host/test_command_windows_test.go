//go:build windows

package host

import "os"

var testTerminalBinary = "cmd.exe"

func testTerminalArgs() []string {
	// Keep the child alive for lifecycle/IPC tests. `more` exits immediately
	// when its stdin is not a console, making the named pipe disappear before
	// the test client can connect.
	return []string{"/D", "/Q", "/C", "ping -t 127.0.0.1 >nul"}
}

var interactiveTestTerminalBinary string

func init() {
	if exe, err := os.Executable(); err == nil {
		interactiveTestTerminalBinary = exe
	} else {
		interactiveTestTerminalBinary = "powershell.exe"
	}
}

func interactiveTestTerminalArgs() []string {
	if interactiveTestTerminalBinary != "powershell.exe" {
		return []string{"-nexus-echo-helper"}
	}
	return []string{
		"-NoLogo",
		"-NoProfile",
		"-NonInteractive",
		"-ExecutionPolicy",
		"Bypass",
		"-Command",
		"$in=[Console]::OpenStandardInput();$out=[Console]::OpenStandardOutput();$ready=[Text.Encoding]::UTF8.GetBytes(\"NEXUS_TEST_READY`n\");$out.Write($ready,0,$ready.Length);$out.Flush();while(($b=$in.ReadByte()) -ge 0){$out.WriteByte([byte]$b);$out.Flush()}",
	}
}
