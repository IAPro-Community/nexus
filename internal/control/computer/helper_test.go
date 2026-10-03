package computer

import (
	"context"
	"runtime"
	"testing"
)

func TestUnavailableProviderReportsNoFalseCapabilities(t *testing.T) {
	p := UnavailableProvider{Platform: runtime.GOOS, Reason: "missing"}
	capabilities, err := p.Capabilities(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if capabilities.Click || capabilities.Screenshot || capabilities.TypeText || capabilities.WindowManagement {
		t.Fatalf("unavailable provider reported capabilities: %+v", capabilities)
	}
	permissions, err := p.Permissions(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if permissions.Available {
		t.Fatal("unavailable provider must not report available")
	}
}
