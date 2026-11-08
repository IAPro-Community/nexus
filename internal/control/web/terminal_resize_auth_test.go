package web

import "testing"

func TestValidTerminalResizeRequest(t *testing.T) {
	for _, tc := range []struct {
		name       string
		isWriter   bool
		rows, cols int
		want       bool
	}{
		{"controller valid", true, 52, 312, true},
		{"viewer cannot resize shared PTY", false, 24, 80, false},
		{"invalid rows", true, 0, 80, false},
		{"invalid cols", true, 24, -1, false},
		{"overflow rows", true, 65536, 80, false},
		{"overflow cols", true, 24, 65536, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := validTerminalResizeRequest(tc.isWriter, tc.rows, tc.cols); got != tc.want {
				t.Fatalf("validTerminalResizeRequest(%t, %d, %d)=%t, want %t", tc.isWriter, tc.rows, tc.cols, got, tc.want)
			}
		})
	}
}
