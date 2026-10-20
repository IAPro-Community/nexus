package main

import (
	"testing"
)

func TestHUDWindowsOptionsDisableNativeFramelessDecorations(t *testing.T) {
	options := hudWindowsOptions()
	if !options.DisableFramelessWindowDecorations {
		t.Fatal("HUD must not draw native frame decorations around its rounded capsule")
	}
}
