package desktop

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func TestNexV2Controller100CycleStabilityAndLegacyIsolation(t *testing.T) {
	dir := t.TempDir()
	legacyPath := filepath.Join(dir, "hud-window-state.json")
	legacySentinel := []byte(`{"appearance":"MASCOT","width":224}`)
	if err := os.WriteFile(legacyPath, legacySentinel, 0o600); err != nil {
		t.Fatal(err)
	}

	displays := []DisplayDescriptor{
		{
			ID:          "eDP-1",
			WorkArea:    application.Rect{X: 0, Y: 0, Width: 1920, Height: 1032},
			ScaleFactor: 1.25,
			Primary:     true,
		},
	}

	var mu sync.Mutex
	lastRect := application.Rect{X: 1744, Y: 856, Width: 160, Height: 160}
	ctrl := NewNexV2Controller(
		NewNexV2StateManager(filepath.Join(dir, "nex-v2-state.json")),
		NexV2PlatformCapabilities{
			GlobalPositioning: true,
			SecondaryWindow:   true,
			Tray:              true,
			Platform:          "linux",
			Session:           "x11",
		},
		NexV2WindowHooks{
			ApplyRect: func(r application.Rect) {
				mu.Lock()
				lastRect = r
				mu.Unlock()
			},
			GetDisplays: func() []DisplayDescriptor { return displays },
			GetRect: func() application.Rect {
				mu.Lock()
				defer mu.Unlock()
				return lastRect
			},
		},
	)

	presets := []string{"compact", "standard", "large"}
	for i := 0; i < 100; i++ {
		if open, err := ctrl.SetPanelOpen(true); err != nil || !open {
			t.Fatalf("cycle %d open failed: open=%v err=%v", i, open, err)
		}
		ctrl.HandleFocusChange(true)
		ctrl.HandleFocusChange(false)
		if open, err := ctrl.SetPanelOpen(false); err != nil || open {
			t.Fatalf("cycle %d close failed: open=%v err=%v", i, open, err)
		}
		if _, err := ctrl.SetShowAvatar(i%2 == 0); err != nil {
			t.Fatalf("cycle %d showAvatar failed: %v", i, err)
		}
		if _, err := ctrl.SetCompactBar(i%3 == 0); err != nil {
			t.Fatalf("cycle %d compactBar failed: %v", i, err)
		}
		if _, err := ctrl.SetPreset(presets[i%len(presets)]); err != nil {
			t.Fatalf("cycle %d setPreset failed: %v", i, err)
		}
		ctrl.HandleDisplayChange(displays)
	}

	// Verify final geometry stayed cleanly inside the 1920x1032 work area.
	mu.Lock()
	finalRect := lastRect
	mu.Unlock()
	if finalRect.X < 0 || finalRect.Y < 0 || finalRect.X+finalRect.Width > 1920 || finalRect.Y+finalRect.Height > 1032 {
		t.Fatalf("geometry escaped work area after 100 cycles: %+v", finalRect)
	}

	// Verify legacy HUD state file was never modified.
	gotLegacy, err := os.ReadFile(legacyPath)
	if err != nil || string(gotLegacy) != string(legacySentinel) {
		t.Fatalf("legacy hud-window-state.json mutated: %q err=%v", string(gotLegacy), err)
	}
}

func TestNexV2ControllerConcurrentTogglesAndWaylandFallback(t *testing.T) {
	ctrl := NewNexV2Controller(
		NewNexV2StateManager(filepath.Join(t.TempDir(), "nex-v2-state.json")),
		NexV2PlatformCapabilities{
			GlobalPositioning:  false,
			ClickThrough:       false,
			GlobalHotkey:       false,
			GlobalHotkeyReason: "Restricted on Wayland",
			SecondaryWindow:    false,
			Tray:               true,
			Platform:           "linux",
			Session:            "wayland",
		},
		NexV2WindowHooks{},
	)

	if _, err := ctrl.MoveByDelta(16, 0); err == nil {
		t.Fatal("Wayland controller must reject global MoveByDelta cleanly")
	}

	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, _ = ctrl.SetPanelOpen(idx%2 == 0)
			_, _ = ctrl.SetDoNotDisturb(idx%3 == 0)
			_, _ = ctrl.SetPaused(idx%4 == 0)
			_, _ = ctrl.SetReducedMotion(idx%5 == 0)
		}(i)
	}
	wg.Wait()
	_ = ctrl.GetPresenceState()
}

func TestNexV2ControllerSurfaceRectTransitions(t *testing.T) {
	displays := []DisplayDescriptor{
		{
			ID:          "DP-1",
			WorkArea:    application.Rect{X: 0, Y: 0, Width: 1920, Height: 1080},
			ScaleFactor: 1.0,
			Primary:     true,
		},
	}
	var lastRect application.Rect
	ctrl := NewNexV2Controller(
		NewNexV2StateManager(filepath.Join(t.TempDir(), "nex-v2-state.json")),
		NexV2PlatformCapabilities{GlobalPositioning: true, Platform: "linux", Session: "x11"},
		NexV2WindowHooks{
			ApplyRect:   func(r application.Rect) { lastRect = r },
			GetDisplays: func() []DisplayDescriptor { return displays },
		},
	)

	if open, err := ctrl.SetPanelOpen(true); err != nil || !open {
		t.Fatalf("SetPanelOpen(true) failed: open=%v err=%v", open, err)
	}
	if lastRect.Width != 440 || lastRect.Height != 540 {
		t.Fatalf("expected 440x540 when panel open, got %+v", lastRect)
	}

	if open, err := ctrl.SetPanelOpen(false); err != nil || open {
		t.Fatalf("SetPanelOpen(false) failed: open=%v err=%v", open, err)
	}
	if lastRect.Width != 160 || lastRect.Height != 160 {
		t.Fatalf("expected 160x160 square preset when panel closed, got %+v", lastRect)
	}

	if _, err := ctrl.SetCompactBar(true); err != nil {
		t.Fatalf("SetCompactBar(true) failed: %v", err)
	}
	if lastRect.Width != 360 || lastRect.Height != 56 {
		t.Fatalf("expected 360x56 compact bar rect, got %+v", lastRect)
	}
}
