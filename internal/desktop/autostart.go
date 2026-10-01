package desktop

import (
	"os"
	"strings"
)

// AutoStartOptions defines a login-startup entry without coupling the caller to
// the operating system's persistence mechanism.
type AutoStartOptions struct {
	AppName  string
	AppID    string
	ExecPath string
	Args     []string
}

// AutoStartManager configures desktop application startup at user login.
type AutoStartManager struct {
	AppName  string
	AppID    string
	ExecPath string
	Args     []string
}

// NewAutoStartManager preserves the legacy Nexus Desktop behavior.
func NewAutoStartManager(appName, execPath string) *AutoStartManager {
	return NewAutoStartManagerWithOptions(AutoStartOptions{
		AppName:  appName,
		AppID:    "iapro-nexus",
		ExecPath: execPath,
		Args:     []string{"--minimized"},
	})
}

// NewAutoStartManagerWithOptions creates a platform-neutral autostart manager.
func NewAutoStartManagerWithOptions(options AutoStartOptions) *AutoStartManager {
	if strings.TrimSpace(options.AppName) == "" {
		options.AppName = "IAPro Nexus"
	}
	options.AppID = sanitizeAutoStartID(options.AppID)
	if strings.TrimSpace(options.ExecPath) == "" {
		options.ExecPath, _ = os.Executable()
	}
	return &AutoStartManager{
		AppName:  options.AppName,
		AppID:    options.AppID,
		ExecPath: options.ExecPath,
		Args:     append([]string(nil), options.Args...),
	}
}

// IsEnabled checks whether login startup is configured for this application.
func (m *AutoStartManager) IsEnabled() bool {
	enabled, err := autostartIsEnabled(m)
	return err == nil && enabled
}

// SetEnabled enables or disables login startup.
func (m *AutoStartManager) SetEnabled(enabled bool) error {
	return autostartSetEnabled(m, enabled)
}

func sanitizeAutoStartID(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "iapro-nexus"
	}
	var b strings.Builder
	b.Grow(len(value))
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9',
			r == '.', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	return strings.Trim(b.String(), "-.")
}
