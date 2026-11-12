//go:build linux

package browsercontrol

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/registry"
)

func TestPreviewListenerHelper(t *testing.T) {
	if os.Getenv("NEXUS_PREVIEW_LISTENER_HELPER") != "1" {
		return
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Fprintln(os.Stdout, "ERROR", err)
		os.Exit(2)
	}
	defer listener.Close()
	_, port, _ := net.SplitHostPort(listener.Addr().String())
	fmt.Fprintln(os.Stdout, port)
	for {
		time.Sleep(time.Hour)
	}
}

func TestLinuxRuntimePreviewInspectorBindsDescendantListenerAndRejectsStaleIdentity(t *testing.T) {
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(binary, "-test.run=^TestPreviewListenerHelper$")
	cmd.Env = append(os.Environ(), "NEXUS_PREVIEW_LISTENER_HELPER=1")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})
	portResult := make(chan string, 1)
	go func() {
		scanner := bufio.NewScanner(stdout)
		if scanner.Scan() {
			portResult <- scanner.Text()
			return
		}
		portResult <- ""
	}()
	var portText string
	select {
	case portText = <-portResult:
	case <-time.After(5 * time.Second):
		t.Fatal("preview listener helper did not publish its port")
	}
	port, err := strconv.Atoi(strings.TrimSpace(portText))
	if err != nil {
		t.Fatalf("listener helper returned invalid port %q", portText)
	}
	if _, ok := (linuxRuntimePreviewInspector{}).ProcessGeneration(cmd.Process.Pid); !ok {
		t.Fatal("could not capture supervised process generation from /proc")
	}

	now := time.Now().UnixNano()
	session := registry.RuntimeSession{
		RuntimeID: "linux-preview-runtime", ProjectID: "linux-preview-project",
		PID: os.Getpid(), ProviderGeneration: now, HostPID: os.Getpid(), HostGeneration: now,
		State: registry.StateRunning, Transport: "ipc",
	}
	inspector := newLinuxRuntimePreviewInspector(func(id string) (registry.RuntimeSession, bool) {
		return session, id == session.RuntimeID
	})
	previews := NewRuntimePreviewRegistry(inspector)
	url := "http://127.0.0.1:" + strconv.Itoa(port) + "/"
	registration, err := previews.ResolveOrRegister(session.RuntimeID, session.ProjectID, url)
	if err != nil {
		t.Fatalf("descendant-owned listener should register: %v", err)
	}
	if registration.ProcessID != cmd.Process.Pid || registration.Port != port {
		t.Fatalf("listener registration did not bind child PID/port: %+v", registration)
	}

	if err := previews.Register(RuntimePreviewRegistration{
		RuntimeID: session.RuntimeID, ProjectID: session.ProjectID, URL: url,
		ProcessID: 1, Port: port,
	}); err == nil {
		t.Fatal("unrelated process PID was allowed to claim the preview listener")
	}
	unused, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	_, unusedPortText, _ := net.SplitHostPort(unused.Addr().String())
	_ = unused.Close()
	unusedPort, _ := strconv.Atoi(unusedPortText)
	if _, err := previews.ResolveOrRegister(session.RuntimeID, session.ProjectID, "http://127.0.0.1:"+strconv.Itoa(unusedPort)+"/"); err == nil {
		t.Fatal("unrelated port without a descendant listener was accepted")
	}

	// A persisted registration from a previous incarnation of this PID must not
	// survive when /proc reports a different generation for the same number.
	previews.mu.Lock()
	entry := previews.entries[session.RuntimeID]
	entry.processGen += ":stale"
	previews.entries[session.RuntimeID] = entry
	previews.mu.Unlock()
	if _, err := previews.Resolve(session.RuntimeID, session.ProjectID, url); err == nil {
		t.Fatal("stale process generation retained authority over the listener")
	}
}

func TestLinuxRuntimePreviewBlocksCrossOriginDocumentRedirect(t *testing.T) {
	if os.Getenv("NEXUS_BROWSER_INTEGRATION") != "1" {
		t.Skip("set NEXUS_BROWSER_INTEGRATION=1 for real Chromium runtime-preview redirect test")
	}
	if _, err := chromeExecutable(); err != nil {
		t.Skipf("no Chromium available: %v", err)
	}
	var externalHits atomic.Int32
	external := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		externalHits.Add(1)
		_, _ = w.Write([]byte("external document must not load"))
	}))
	defer external.Close()
	local := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/redirect" {
			http.Redirect(w, r, external.URL+"/landing", http.StatusFound)
			return
		}
		_, _ = w.Write([]byte(`<html><body><h1>Local supervised preview</h1><script>fetch("` + external.URL + `/api").catch(() => {});</script></body></html>`))
	}))
	defer local.Close()
	now := time.Now().UnixNano()
	session := registry.RuntimeSession{
		RuntimeID: "runtime-preview-redirect", ProjectID: "project-preview-redirect",
		PID: os.Getpid(), ProviderGeneration: now, HostPID: os.Getpid(), HostGeneration: now,
		State: registry.StateRunning, Transport: "ipc",
	}
	inspector := newLinuxRuntimePreviewInspector(func(id string) (registry.RuntimeSession, bool) {
		return session, id == session.RuntimeID
	})
	manager := NewManagerWithRuntimePreviewInspector(inspector)
	defer manager.Shutdown()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	info, err := manager.OpenRuntimeBrowser(ctx, RuntimeCapability{RuntimeID: session.RuntimeID, ProjectID: session.ProjectID}, local.URL)
	if err != nil {
		t.Fatalf("open supervised local preview: %v", err)
	}
	browser, ok := manager.Get(info.ID)
	if !ok {
		t.Fatal("runtime browser was not retained")
	}
	if _, err := browser.Snapshot(ctx); err != nil {
		t.Fatalf("snapshot local preview: %v", err)
	}
	if hits := externalHits.Load(); hits != 0 {
		t.Fatalf("local preview made an external request %d times", hits)
	}
	_, err = browser.Do(ctx, Action{Type: "navigate", URL: local.URL + "/redirect"})
	if err == nil || !strings.Contains(err.Error(), "cross-origin document navigation") {
		t.Fatalf("cross-origin redirect was not stopped: %v", err)
	}
	time.Sleep(250 * time.Millisecond)
	if hits := externalHits.Load(); hits != 0 {
		t.Fatalf("cross-origin redirect reached destination %d times", hits)
	}
}
