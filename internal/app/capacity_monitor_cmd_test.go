package app

import (
	"testing"
)

func TestCapacityMonitorCmd_HelpAndUnknown(t *testing.T) {
	if err := capacityMonitorCmd([]string{"help"}); err != nil {
		t.Fatalf("unexpected error for help: %v", err)
	}

	if err := capacityMonitorCmd([]string{"--help"}); err != nil {
		t.Fatalf("unexpected error for --help: %v", err)
	}

	if err := capacityMonitorCmd([]string{"unknown-subcommand"}); err == nil {
		t.Fatalf("expected error for unknown subcommand, got nil")
	}
}

func TestCapacityMonitorCmd_StatusJSON(t *testing.T) {
	if err := capacityMonitorCmd([]string{"status", "--json"}); err != nil {
		t.Fatalf("unexpected error for status --json: %v", err)
	}
}
