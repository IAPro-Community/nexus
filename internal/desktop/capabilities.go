package desktop

import (
	"os/exec"
	"runtime"
)

// Capabilities represents the native platform features supported by Nexus Desktop.
type Capabilities struct {
	Native             bool `json:"native"`
	FilePicker         bool `json:"filePicker"`
	FolderPicker       bool `json:"folderPicker"`
	Notifications      bool `json:"notifications"`
	Tray               bool `json:"tray"`
	NativeMenus        bool `json:"nativeMenus"`
	DeepLinks          bool `json:"deepLinks"`
	AutoStart          bool `json:"autoStart"`
	WindowManagement   bool `json:"windowManagement"`
	NativeVoiceCapture bool `json:"nativeVoiceCapture"`
}

// DefaultCapabilities is deliberately conservative outside a running Wails v3
// application. GetCapabilities promotes features only when the native app exists.
func DefaultCapabilities() Capabilities {
	notifications := false
	nativeVoiceCapture := false
	switch runtime.GOOS {
	case "windows":
		notifications = commandAvailable("powershell") || commandAvailable("pwsh")
	case "darwin":
		notifications = commandAvailable("osascript")
	case "linux":
		notifications = commandAvailable("notify-send")
		nativeVoiceCapture = commandAvailable("pw-record") || commandAvailable("arecord")
	}

	return Capabilities{
		Native:             true,
		Notifications:      notifications,
		WindowManagement:   true,
		NativeVoiceCapture: nativeVoiceCapture,
	}
}

func commandAvailable(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
