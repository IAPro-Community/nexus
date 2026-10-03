package browser

import (
	"strings"
	"testing"
)

func TestValidateDesignURL(t *testing.T) {
	for _, value := range []string{"http://127.0.0.1:3000", "https://example.com/path"} {
		if err := validateDesignURL(value); err != nil {
			t.Fatalf("valid URL %q rejected: %v", value, err)
		}
	}
	for _, value := range []string{"file:///tmp/index.html", "javascript:alert(1)", "/relative", "https://user:secret@example.com"} {
		if err := validateDesignURL(value); err == nil {
			t.Fatalf("unsafe/non-network URL %q accepted", value)
		}
	}
}

func TestScanDevToolsEndpoint(t *testing.T) {
	out := make(chan string, 1)
	scanDevToolsEndpoint(strings.NewReader("noise\nDevTools listening on ws://127.0.0.1:43210/devtools/browser/abc\n"), out)
	if got := <-out; got != "ws://127.0.0.1:43210/devtools/browser/abc" {
		t.Fatalf("endpoint=%q", got)
	}
}
