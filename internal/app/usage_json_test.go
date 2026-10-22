package app

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/IAPro-Community/nexus/internal/tui"
)

// An installed provider without any profile (e.g. OpenCode freshly installed)
// must stay visible in `nexus usage --json`: the Windows/Linux beta gates
// reject the run when an installed CLI is absent from the JSON surface.
func TestUnconfiguredUsageRowsKeepInstalledProviderVisible(t *testing.T) {
	rows := unconfiguredUsageRows([]tui.InstalledProviderInfo{{
		ID:        "opencode",
		Name:      "OpenCode",
		Version:   "1.2.3",
		Installed: true,
		Profiles:  0,
	}})
	if len(rows) != 1 {
		t.Fatalf("want 1 unconfigured row, got %d", len(rows))
	}

	payload, err := json.Marshal(rows)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(payload), "opencode") {
		t.Fatalf("payload hides the installed provider: %s", payload)
	}

	var decoded []map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatal(err)
	}
	row := decoded[0]
	if row["provider_id"] != "opencode" {
		t.Fatalf("provider_id=%v", row["provider_id"])
	}
	if row["unconfigured"] != true {
		t.Fatalf("unconfigured=%v", row["unconfigured"])
	}
	if row["authenticated"] != false {
		t.Fatalf("authenticated=%v", row["authenticated"])
	}
	if row["status"] != "UNKNOWN" {
		t.Fatalf("status=%v", row["status"])
	}
	if row["windows"] == nil {
		t.Fatal("windows must serialize as an array, not null")
	}
}

func TestUnconfiguredUsageRowsEmptyForNoCLIs(t *testing.T) {
	rows := unconfiguredUsageRows(nil)
	if len(rows) != 0 {
		t.Fatalf("want no rows, got %d", len(rows))
	}
	payload, err := json.Marshal(rows)
	if err != nil {
		t.Fatal(err)
	}
	if string(payload) != "[]" {
		t.Fatalf("empty rows must serialize as [], got %s", payload)
	}
}
