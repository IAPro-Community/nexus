package desktop

import (
	"runtime"
	"testing"
)

func TestNativeVoiceCaptureCapabilityMatchesLinuxRecorderAvailability(t *testing.T) {
	got := DefaultCapabilities().NativeVoiceCapture
	if runtime.GOOS != "linux" {
		if got {
			t.Fatal("native voice capture must not be advertised outside Linux fallback")
		}
		return
	}
	want := commandAvailable("pw-record") || commandAvailable("arecord")
	if got != want {
		t.Fatalf("native voice capture capability = %v, want %v", got, want)
	}
}
