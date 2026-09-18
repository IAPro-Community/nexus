package app

import (
	"strings"
	"testing"
)

func TestTerminalDoctorRejectsUnknownFlags(t *testing.T) {
	err := terminalDoctorCmd([]string{"--bad"})
	if err == nil || !strings.Contains(err.Error(), "unknown terminal doctor flag") {
		t.Fatalf("expected terminal doctor flag error, got %v", err)
	}
}
