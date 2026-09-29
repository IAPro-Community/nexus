//go:build windows

package launcher

import "testing"

func TestDetachedHostCreationFlagsKeepProcessGroupWithoutConsole(t *testing.T) {
	const (
		createNewProcessGroup = 0x00000200
		createNoWindow        = 0x08000000
		detachedProcess       = 0x00000008
	)
	if detachedHostCreationFlags&createNewProcessGroup == 0 {
		t.Fatal("detached host must keep a separate process group")
	}
	if detachedHostCreationFlags&createNoWindow == 0 {
		t.Fatal("detached host must suppress a visible console window")
	}
	if detachedHostCreationFlags&detachedProcess != 0 {
		t.Fatal("DETACHED_PROCESS must not replace CREATE_NO_WINDOW for the supervised host")
	}
}
