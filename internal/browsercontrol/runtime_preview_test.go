package browsercontrol

import (
	"os"
	"testing"

	"github.com/IAPro-Community/nexus/internal/control/registry"
)

type previewInspectorFixture struct {
	session           registry.RuntimeSession
	child             bool
	listener          bool
	processGeneration string
}

func (f *previewInspectorFixture) RuntimeSession(runtimeID string) (registry.RuntimeSession, bool) {
	return f.session, runtimeID == f.session.RuntimeID
}
func (f *previewInspectorFixture) IsRuntimeDescendant(_ registry.RuntimeSession, processID int) bool {
	return f.child && processID == 202
}
func (f *previewInspectorFixture) ProcessGeneration(processID int) (string, bool) {
	return f.processGeneration, processID == 202 && f.processGeneration != ""
}
func (f *previewInspectorFixture) OwnsListeningPort(processID, port int) bool {
	return f.listener && processID == 202 && port == 4173
}
func (f *previewInspectorFixture) FindRuntimeListener(_ registry.RuntimeSession, port int) (int, string, bool) {
	if !f.child || !f.listener || port != 4173 || f.processGeneration == "" {
		return 0, "", false
	}
	return 202, f.processGeneration, true
}

func TestRuntimePreviewRegistrationRequiresSupervisedProcessAndListener(t *testing.T) {
	inspector := &previewInspectorFixture{
		session: registry.RuntimeSession{
			RuntimeID: "runtime-preview-1", ProjectID: "project-1", State: registry.StateRunning,
			PID: 101, HostPID: os.Getpid(), Transport: "ipc",
		},
		child: true, listener: true, processGeneration: "start-1",
	}
	previews := NewRuntimePreviewRegistry(inspector)
	registration := RuntimePreviewRegistration{
		RuntimeID: "runtime-preview-1", ProjectID: "project-1",
		URL: "http://127.0.0.1:4173/app", ProcessID: 202, Port: 4173,
	}
	if err := previews.Register(registration); err != nil {
		t.Fatalf("register supervised preview: %v", err)
	}
	if _, err := previews.Resolve(registration.RuntimeID, registration.ProjectID, "http://127.0.0.1:4173/other"); err != nil {
		t.Fatalf("same supervised origin should resolve: %v", err)
	}
	for _, target := range []string{
		"https://example.org/",
		"http://127.0.0.1:9999/",
		"http://127.0.0.2:4173/",
	} {
		if _, err := previews.Resolve(registration.RuntimeID, registration.ProjectID, target); err == nil {
			t.Errorf("navigation to unregistered target %q was accepted", target)
		}
	}
	if _, err := previews.Resolve(registration.RuntimeID, "other-project", ""); err == nil {
		t.Fatal("registration crossed project scope")
	}
	inspector.child = false
	if _, err := previews.Resolve(registration.RuntimeID, registration.ProjectID, ""); err == nil {
		t.Fatal("stale process lineage remained authorized")
	}
}

func TestRuntimePreviewFirstOpenDiscoversOnlyDescendantOwnedPort(t *testing.T) {
	inspector := &previewInspectorFixture{
		session: registry.RuntimeSession{
			RuntimeID: "runtime-preview-1", ProjectID: "project-1", State: registry.StateRunning,
			PID: 101, HostPID: os.Getpid(), Transport: "ipc",
		},
		child: true, listener: true, processGeneration: "start-1",
	}
	previews := NewRuntimePreviewRegistry(inspector)
	registered, err := previews.ResolveOrRegister("runtime-preview-1", "project-1", "http://localhost:4173/")
	if err != nil {
		t.Fatalf("discover first runtime preview: %v", err)
	}
	if registered.ProcessID != 202 || registered.Port != 4173 {
		t.Fatalf("unexpected discovered owner: %+v", registered)
	}
	if _, err := previews.ResolveOrRegister("runtime-preview-1", "project-1", "http://localhost:4174/"); err == nil {
		t.Fatal("unrelated port was accepted without a descendant listener")
	}
}

func TestRuntimePreviewRegistrationFailsClosedWithoutSupervisor(t *testing.T) {
	registration := RuntimePreviewRegistration{
		RuntimeID: "runtime-preview-1", ProjectID: "project-1",
		URL: "http://localhost:4173", ProcessID: 202, Port: 4173,
	}
	if err := NewRuntimePreviewRegistry(nil).Register(registration); err == nil {
		t.Fatal("registration succeeded without OS process supervisor")
	}
	if _, err := NewRuntimePreviewRegistry(nil).Resolve(registration.RuntimeID, registration.ProjectID, ""); err == nil {
		t.Fatal("resolution succeeded without OS process supervisor")
	}
}

func TestRuntimePreviewRegistrationRevokesReusedProcessID(t *testing.T) {
	inspector := &previewInspectorFixture{
		session: registry.RuntimeSession{
			RuntimeID: "runtime-preview-1", ProjectID: "project-1", State: registry.StateRunning,
			PID: 101, HostPID: os.Getpid(), Transport: "ipc",
		},
		child: true, listener: true, processGeneration: "start-1",
	}
	previews := NewRuntimePreviewRegistry(inspector)
	registration := RuntimePreviewRegistration{
		RuntimeID: "runtime-preview-1", ProjectID: "project-1", URL: "http://localhost:4173/",
		ProcessID: 202, Port: 4173,
	}
	if err := previews.Register(registration); err != nil {
		t.Fatal(err)
	}
	inspector.processGeneration = "start-2"
	if _, err := previews.Resolve(registration.RuntimeID, registration.ProjectID, ""); err == nil {
		t.Fatal("reused process ID retained the previous registration")
	}
}

func TestRuntimePreviewRegistrationRejectsUnboundHostOrPort(t *testing.T) {
	inspector := &previewInspectorFixture{
		session: registry.RuntimeSession{
			RuntimeID: "runtime-preview-1", ProjectID: "project-1", State: registry.StateRunning,
			PID: 101, HostPID: os.Getpid(), Transport: "ipc",
		},
		child: true, listener: true, processGeneration: "start-1",
	}
	previews := NewRuntimePreviewRegistry(inspector)
	base := RuntimePreviewRegistration{RuntimeID: "runtime-preview-1", ProjectID: "project-1", ProcessID: 202, Port: 4173}
	for _, raw := range []string{
		"https://example.org/app",
		"http://localhost/app",
		"http://localhost.evil.test:4173/",
		"http://user:secret@localhost:4173/",
	} {
		candidate := base
		candidate.URL = raw
		if err := previews.Register(candidate); err == nil {
			t.Errorf("unsafe/unbound preview URL accepted: %q", raw)
		}
	}
	wrongPort := base
	wrongPort.URL = "http://localhost:4173/"
	wrongPort.Port = 4174
	if err := previews.Register(wrongPort); err == nil {
		t.Fatal("URL/registered listener port mismatch accepted")
	}
	inspector.child = false
	if err := previews.Register(baseWithURL(base, "http://localhost:4173/")); err == nil {
		t.Fatal("process outside runtime lineage accepted")
	}
	inspector.child = true
	inspector.listener = false
	if err := previews.Register(baseWithURL(base, "http://localhost:4173/")); err == nil {
		t.Fatal("port not owned by supervised process accepted")
	}
}

func TestRuntimeBrowserOpenFailsClosedUntilSupervisedPreviewRegistered(t *testing.T) {
	manager := NewManager()
	if _, err := manager.OpenRuntimeBrowser(t.Context(), RuntimeCapability{RuntimeID: "runtime-preview-1", ProjectID: "project-1"}, "http://localhost:4173/"); err == nil {
		t.Fatal("runtime browser opened without a supervised preview registration")
	}
}

func TestNavigationPolicyScopesRuntimePreviewButPreservesGrantedBrowserAutomation(t *testing.T) {
	inspector := &previewInspectorFixture{
		session: registry.RuntimeSession{
			RuntimeID: "runtime-preview-1", ProjectID: "project-1", State: registry.StateRunning,
			PID: 101, HostPID: os.Getpid(), Transport: "ipc",
		},
		child: true, listener: true, processGeneration: "start-1",
	}
	previews := NewRuntimePreviewRegistry(inspector)
	if err := previews.Register(RuntimePreviewRegistration{
		RuntimeID: "runtime-preview-1", ProjectID: "project-1", URL: "http://localhost:4173/",
		ProcessID: 202, Port: 4173,
	}); err != nil {
		t.Fatal(err)
	}
	runtimeSession := &Session{
		info:          Info{OwnerRuntimeID: "runtime-preview-1", ProjectID: "project-1"},
		previewOrigin: "http://localhost:4173", previewRegistry: previews,
	}
	if err := runtimeSession.validateNavigation("http://localhost:4173/settings"); err != nil {
		t.Fatalf("same supervised origin rejected: %v", err)
	}
	for _, url := range []string{
		"https://example.org/",
		"http://localhost:9999/",
		"file:///etc/passwd",
	} {
		if err := runtimeSession.validateNavigation(url); err == nil {
			t.Errorf("runtime preview navigated outside its supervised origin: %q", url)
		}
	}

	// Human-granted browser automation sessions have no runtime preview binding;
	// their existing external-site workflow remains available.
	grantedSession := &Session{}
	if err := grantedSession.validateNavigation("https://example.org/path"); err != nil {
		t.Fatalf("external browser automation navigation was unexpectedly blocked: %v", err)
	}
}

func baseWithURL(base RuntimePreviewRegistration, raw string) RuntimePreviewRegistration {
	base.URL = raw
	return base
}
