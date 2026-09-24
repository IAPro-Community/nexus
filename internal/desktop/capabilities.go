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

// DefaultCapabilities returns the standard native desktop capabilities.
func DefaultCapabilities() Capabilities {
	filePicker := false
	folderPicker := false
	notifications := false
	autoStart := false
	nativeVoiceCapture := false
	switch runtime.GOOS {
	case "windows":
		notifications = commandAvailable("powershell") || commandAvailable("pwsh")
		autoStart = commandAvailable("reg.exe")
	case "darwin":
		filePicker = commandAvailable("osascript")
		folderPicker = filePicker
		notifications = filePicker
		autoStart = true
	case "linux":
		filePicker = commandAvailable("zenity")
		folderPicker = filePicker
		notifications = commandAvailable("notify-send")
		autoStart = true
		nativeVoiceCapture = commandAvailable("pw-record") || commandAvailable("arecord")
	}

	return Capabilities{
		Native:             true,
		FilePicker:         filePicker,
		FolderPicker:       folderPicker,
		Notifications:      notifications,
		Tray:               false,
		NativeMenus:        false,
		DeepLinks:          false,
		AutoStart:          autoStart,
		WindowManagement:   true,
		NativeVoiceCapture: nativeVoiceCapture,
	}
}

func commandAvailable(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
