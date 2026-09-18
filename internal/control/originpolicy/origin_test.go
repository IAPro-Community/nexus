package originpolicy

import "testing"

func TestValidateRequiresExactRequestTarget(t *testing.T) {
	cases := []struct {
		host, origin string
		ok           bool
	}{
		{"127.0.0.1:9090", "http://127.0.0.1:9090", true},
		{"localhost:9090", "http://localhost:9090", true},
		{"192.168.1.10:9090", "https://192.168.1.10:9090", true},
		{"192.168.1.10:9090", "https://192.168.1.11:9090", false},
		{"127.0.0.1:9090", "http://127.0.0.1:9091", false},
		{"127.0.0.1:9090", "http://localhost:9090", false},
		{"localhost.evil.com:9090", "http://localhost.evil.com:9090", false},
		{"8.8.8.8:9090", "http://8.8.8.8:9090", false},
		{"127.0.0.1:9090", "wails://wails", true},
		{"localhost:9090", "wails://wails", true},
		{"wails", "wails://wails", true},
		{"127.0.0.1:9090", "http://wails.localhost", true},
		{"127.0.0.1:9090", "http://wails.localhost.evil.com", false},
		{"127.0.0.1:9090", "https://wails.localhost.attacker", false},
		{"127.0.0.1:9090", "http://notwails.localhost", false},
		{"127.0.0.1:9090", "wails://evil.com", false},
		{"127.0.0.1:9090", "wails://wails.localhost.evil.com", false},
		{"127.0.0.1:9090", "wails://attacker", false},
		{"192.168.1.10:9090", "wails://wails", false},
		{"8.8.8.8:9090", "wails://wails", false},
	}
	for _, tc := range cases {
		if got := Validate(tc.host, tc.origin); got != tc.ok {
			t.Fatalf("host=%q origin=%q got=%v want=%v", tc.host, tc.origin, got, tc.ok)
		}
	}
}

func TestIsTrustedDesktopRequest(t *testing.T) {
	cases := []struct {
		host, origin, referer string
		want                  bool
	}{
		{"127.0.0.1:13000", "wails://wails", "", true},
		{"localhost:13000", "http://wails.localhost", "", true},
		{"127.0.0.1:13000", "", "wails://wails/index.html", true},
		{"127.0.0.1:13000", "", "http://wails.localhost/app", true},
		{"wails", "wails://wails", "", true},
		{"wails", "", "", false}, // Host: wails alone without desktop origin/referer must be rejected
		// Conflicting headers: invalid Origin must NOT fall back to valid Referer
		{"127.0.0.1:13000", "wails://evil.com", "wails://wails", false},
		{"127.0.0.1:13000", "http://evil.com", "http://wails.localhost", false},
		// Negative tests: bypass attempts with malicious domains
		{"127.0.0.1:13000", "wails://evil.com", "", false},
		{"127.0.0.1:13000", "wails://wails.localhost.evil.com", "", false},
		{"127.0.0.1:13000", "wails://attacker", "", false},
		{"127.0.0.1:13000", "http://wails.localhost.evil.com", "", false},
		{"127.0.0.1:13000", "", "https://wails.localhost.attacker/steal", false},
		{"127.0.0.1:13000", "", "wails://evil.com/app", false},
		{"127.0.0.1:13000", "http://evil.com", "", false},
		{"127.0.0.1:13000", "", "", false},
		// Non-loopback host must be rejected even with valid desktop origin header
		{"192.168.1.10:13000", "wails://wails", "", false},
		{"8.8.8.8:13000", "wails://wails", "", false},
	}

	for _, tc := range cases {
		if got := IsTrustedDesktopRequest(tc.host, tc.origin, tc.referer); got != tc.want {
			t.Fatalf("host=%q origin=%q referer=%q got=%v want=%v", tc.host, tc.origin, tc.referer, got, tc.want)
		}
	}
}
