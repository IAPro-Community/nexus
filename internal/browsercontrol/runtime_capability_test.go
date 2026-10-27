package browsercontrol

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestRuntimeCapabilityLifecycleAndScoping(t *testing.T) {
	t.Setenv("NEXUS_DATA_DIR", t.TempDir())
	token, err := IssueRuntimeCapability("runtime-test-01", "project-A")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(token, "nbr_runtime-test-01.") {
		t.Fatal("unexpected token format")
	}
	if _, ok := ResolveRuntimeCapability(token); ok {
		t.Fatal("capability must require a registered live runtime")
	}
	path, err := capFile("runtime-test-01")
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), token) {
		t.Fatal("capability token was persisted in plain text")
	}
	var cap RuntimeCapability
	if err := json.Unmarshal(b, &cap); err != nil || cap.ProjectID != "project-A" {
		t.Fatal("capability not scoped to project")
	}
	RevokeRuntimeCapability("runtime-test-01")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("revocation failed")
	}
	for _, invalid := range []string{"../other", "a/b", "", "one:two"} {
		if _, err := IssueRuntimeCapability(invalid, "project-A"); err == nil {
			t.Fatalf("unsafe runtime ID accepted: %q", invalid)
		}
	}
}
func TestBrowserEndpointIsLocalAndEphemeral(t *testing.T) {
	t.Setenv("NEXUS_DATA_DIR", t.TempDir())
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/health" {
			w.WriteHeader(http.StatusOK)
		} else {
			http.NotFound(w, r)
		}
	})}
	go func() { _ = server.Serve(listener) }()
	defer server.Close()
	if err := PublishBrowserEndpoint(listener.Addr().(*net.TCPAddr).Port, os.Getpid()); err != nil {
		t.Fatal(err)
	}
	if url := DiscoverBrowserEndpoint(); !strings.HasPrefix(url, "http://127.0.0.1:") {
		t.Fatalf("local discovery failed: %q", url)
	}
	UnpublishBrowserEndpoint(os.Getpid())
	if DiscoverBrowserEndpoint() != "" {
		t.Fatal("endpoint remained after shutdown")
	}
}
func TestBrowserRuntimeRejectsEmptyScope(t *testing.T) {
	m := NewManager()
	if _, err := m.OpenRuntimeBrowser(context.Background(), RuntimeCapability{}, ""); err == nil {
		t.Fatal("unscoped browser provisioning accepted")
	}
}
