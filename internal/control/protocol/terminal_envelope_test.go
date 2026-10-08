package protocol

import "testing"

func TestWireEnvelopeDoesNotSwallowProviderJSON(t *testing.T) {
	cases := []struct {
		name, data string
		expected   bool
	}{
		{"Nexus request", `{"version":1,"id":"req-123","command":"lease_acquire","timestamp":"2026-01-01T00:00:00Z"}`, true},
		{"Nexus response", `{"version":1,"ok":true,"timestamp":"2026-01-01T00:00:00Z"}`, true},
		{"Codex command", `{"command":"npm test","ok":true}`, false},
		{"Codex status", `{"version":1,"ok":true,"data":{"status":"ready"}}`, false},
		{"Invalid request ID", `{"version":1,"id":"req-no","command":"input","timestamp":"2026-01-01T00:00:00Z"}`, false},
		{"Invalid timestamp", `{"version":1,"ok":true,"timestamp":"invalid"}`, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsWireControlFrame([]byte(tc.data)); got != tc.expected {
				t.Fatalf("control frame %q: got %t want %t", tc.data, got, tc.expected)
			}
		})
	}
}

func TestWireControlPrefixDistinguishesProviderJSON(t *testing.T) {
	for _, prefix := range []string{`{`, `{"v`, `{"version":1`, `{"version":1,"ok"`} {
		if !MayBeWireControlPrefix([]byte(prefix)) {
			t.Fatalf("did not buffer possible Nexus prefix %q", prefix)
		}
	}
	for _, text := range []string{`{"command":"npm test"}`, `{"ok":true}`, `{"version":2`} {
		if MayBeWireControlPrefix([]byte(text)) {
			t.Fatalf("misclassified provider JSON prefix %q", text)
		}
	}
}
