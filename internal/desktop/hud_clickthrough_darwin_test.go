//go:build darwin && cgo

package desktop

import "testing"

func TestHUDMacClickThroughIsOptIn(t *testing.T) {
	t.Setenv("NEXUS_HUD_MAC_CLICKTHROUGH", "")
	if hudHitTestingSupported() {
		t.Fatal("unverified Quartz click-through must be disabled by default")
	}
	t.Setenv("NEXUS_HUD_MAC_CLICKTHROUGH", "1")
	if !hudHitTestingSupported() {
		t.Fatal("explicit macOS opt-in must enable the native cursor watcher")
	}
}
