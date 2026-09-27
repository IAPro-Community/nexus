//go:build windows

package desktop

import (
	"errors"
	"os/exec"
	"strings"

	"github.com/IAPro-Community/nexus/internal/executil"
)

const windowsRunKey = `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`

func autostartIsEnabled(m *AutoStartManager) (bool, error) {
	err := executil.Command("reg", "query", windowsRunKey, "/v", m.AppID).Run()
	if err == nil {
		return true, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return false, nil
	}
	return false, err
}

func autostartSetEnabled(m *AutoStartManager, enabled bool) error {
	if !enabled {
		err := executil.Command("reg", "delete", windowsRunKey, "/v", m.AppID, "/f").Run()
		if err == nil {
			return nil
		}
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil
		}
		return err
	}
	value := windowsAutostartCommand(m.ExecPath, m.Args)
	return executil.Command("reg", "add", windowsRunKey, "/v", m.AppID, "/t", "REG_SZ", "/d", value, "/f").Run()
}

func windowsAutostartCommand(execPath string, args []string) string {
	parts := []string{windowsQuoteArg(execPath)}
	for _, arg := range args {
		parts = append(parts, windowsQuoteArg(arg))
	}
	return strings.Join(parts, " ")
}

func windowsQuoteArg(value string) string {
	if value == "" {
		return `""`
	}
	if !strings.ContainsAny(value, " \t\"") {
		return value
	}
	var b strings.Builder
	b.WriteByte('"')
	backslashes := 0
	for _, r := range value {
		if r == '\\' {
			backslashes++
			continue
		}
		if r == '"' {
			b.WriteString(strings.Repeat("\\", backslashes*2+1))
			b.WriteRune(r)
			backslashes = 0
			continue
		}
		b.WriteString(strings.Repeat("\\", backslashes))
		backslashes = 0
		b.WriteRune(r)
	}
	b.WriteString(strings.Repeat("\\", backslashes*2))
	b.WriteByte('"')
	return b.String()
}
