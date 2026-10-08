package web

import "testing"

func TestIsProtocolControlFrame(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want bool
	}{
		{name: "lease request", in: `{"version":1,"id":"req-1","command":"lease_acquire","timestamp":"2026-01-01T00:00:00Z"}`, want: true},
		{name: "ok response", in: `{"version":1,"ok":true,"data":"\"lease_acquired\"","timestamp":"2026-01-01T00:00:00Z"}`, want: true},
		{name: "error response", in: `{"version":1,"ok":false,"error":"boom","timestamp":"2026-01-01T00:00:00Z"}`, want: true},
		{name: "pty text", in: "hello from codex\n", want: false},
		{name: "json-looking prompt", in: `{"note":"user typed this"}`, want: false},
		{name: "Codex JSON command", in: `{"command":"npm test","ok":true}`, want: false},
		{name: "Codex JSON status", in: `{"version":1,"ok":true,"data":{"count":1}}`, want: false},
		{name: "empty", in: "   ", want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isProtocolControlFrame([]byte(tc.in)); got != tc.want {
				t.Fatalf("isProtocolControlFrame(%q)=%v want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestScrubProtocolFramesFromOutput(t *testing.T) {
	in := "welcome\n" +
		`{"version":1,"id":"req-1","command":"lease_acquire","timestamp":"2026-01-01T00:00:00Z"}` + "\n" +
		"real pty line\n" +
		`{"version":1,"ok":true,"data":"\"lease_acquired\"","timestamp":"2026-01-01T00:00:00Z"}` + "\n" +
		`{"note":"user typed this"}` + "\n" +
		`{"command":"npm test","ok":true}` + "\n"
	got := scrubProtocolFramesFromOutput(in)
	want := "welcome\nreal pty line\n{\"note\":\"user typed this\"}\n{\"command\":\"npm test\",\"ok\":true}\n"
	if got != want {
		t.Fatalf("scrubProtocolFramesFromOutput()\n got %q\nwant %q", got, want)
	}
	if scrubProtocolFramesFromOutput(`{"version":1,"id":"req-2","command":"lease_acquire","timestamp":"2026-01-01T00:00:00Z"}`) != "" {
		t.Fatal("expected empty when history is only protocol frames")
	}
}
