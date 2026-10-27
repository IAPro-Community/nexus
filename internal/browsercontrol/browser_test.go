package browsercontrol

import "testing"

func TestValidURL(t *testing.T) {
	for _, raw := range []string{"http://localhost:3000", "https://example.org/path?q=1", "http://127.0.0.1:8080"} {
		if err := ValidURL(raw); err != nil {
			t.Errorf("%q should be valid: %v", raw, err)
		}
	}
	for _, raw := range []string{"", "file:///etc/passwd", "javascript:alert(1)", "http://", "https://user:pass@example.org", "about:blank", "ftp://example.org"} {
		if err := ValidURL(raw); err == nil {
			t.Errorf("%q should be rejected", raw)
		}
	}
}
func TestManagerEmptyAndClose(t *testing.T) {
	m := NewManager()
	if got := m.List(""); len(got) != 0 {
		t.Fatalf("new manager has sessions: %d", len(got))
	}
	if m.Close("not-exists") {
		t.Fatal("closing unknown session returned true")
	}
	m.Shutdown()
}
