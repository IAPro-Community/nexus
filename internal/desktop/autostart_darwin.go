//go:build darwin

package desktop

import (
	"bytes"
	"encoding/xml"
	"os"
	"path/filepath"
)

func darwinAutoStartPath(m *AutoStartManager) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "LaunchAgents", m.AppID+".plist"), nil
}

func xmlEscape(value string) string {
	var b bytes.Buffer
	_ = xml.EscapeText(&b, []byte(value))
	return b.String()
}

func autostartIsEnabled(m *AutoStartManager) (bool, error) {
	path, err := darwinAutoStartPath(m)
	if err != nil {
		return false, err
	}
	_, err = os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func autostartSetEnabled(m *AutoStartManager, enabled bool) error {
	path, err := darwinAutoStartPath(m)
	if err != nil {
		return err
	}
	if !enabled {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	programArguments := "<string>" + xmlEscape(m.ExecPath) + "</string>"
	for _, arg := range m.Args {
		programArguments += "\n      <string>" + xmlEscape(arg) + "</string>"
	}
	content := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>` + xmlEscape(m.AppID) + `</string>
  <key>ProgramArguments</key>
  <array>
      ` + programArguments + `
  </array>
  <key>RunAtLoad</key>
  <true/>
  <key>KeepAlive</key>
  <false/>
</dict>
</plist>
`
	return os.WriteFile(path, []byte(content), 0o644)
}
