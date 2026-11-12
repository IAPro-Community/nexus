package browsercontrol

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/IAPro-Community/nexus/internal/control/registry"
)

// RuntimePreviewInspector is implemented by the platform process supervisor.
// Every method must use process-generation-aware identities: PIDs can be reused.
// A nil inspector intentionally disables runtime-owned live previews.
type RuntimePreviewInspector interface {
	RuntimeSession(runtimeID string) (registry.RuntimeSession, bool)
	ProcessGeneration(processID int) (string, bool)
	IsRuntimeDescendant(session registry.RuntimeSession, processID int) bool
	OwnsListeningPort(processID, port int) bool
	FindRuntimeListener(session registry.RuntimeSession, port int) (processID int, generation string, ok bool)
}

// RuntimePreviewRegistration binds a local app endpoint to a process supervised
// under one live Nexus runtime. The MCP caller cannot create registrations.
type RuntimePreviewRegistration struct {
	RuntimeID string
	ProjectID string
	URL       string
	ProcessID int
	Port      int
}

type runtimePreviewEntry struct {
	registration RuntimePreviewRegistration
	hostPID      int
	hostGen      int64
	providerPID  int
	providerGen  int64
	processGen   string
}

// RuntimePreviewRegistry admits only explicit registrations validated against
// a live runtime and an OS supervisor. Without an inspector it fails closed.
type RuntimePreviewRegistry struct {
	mu        sync.RWMutex
	inspector RuntimePreviewInspector
	entries   map[string]runtimePreviewEntry
}

func NewRuntimePreviewRegistry(inspector RuntimePreviewInspector) *RuntimePreviewRegistry {
	return &RuntimePreviewRegistry{inspector: inspector, entries: make(map[string]runtimePreviewEntry)}
}

func parseRuntimePreviewURL(raw string) (*url.URL, int, error) {
	raw = NormalizeURL(raw)
	if err := ValidURL(raw); err != nil {
		return nil, 0, err
	}
	u, err := url.Parse(raw)
	if err != nil || u == nil {
		return nil, 0, errors.New("invalid runtime preview URL")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, 0, errors.New("runtime preview must use HTTP(S)")
	}
	host := strings.TrimSuffix(strings.ToLower(u.Hostname()), ".")
	if host != "localhost" {
		ip := net.ParseIP(host)
		if ip == nil || !ip.IsLoopback() {
			return nil, 0, errors.New("runtime preview host must be loopback")
		}
	}
	portText := u.Port()
	if portText == "" {
		return nil, 0, errors.New("runtime preview must include its registered port")
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port < 1 || port > 65535 {
		return nil, 0, errors.New("runtime preview port is invalid")
	}
	return u, port, nil
}

func runtimePreviewOrigin(u *url.URL) string {
	return strings.ToLower(u.Scheme) + "://" + strings.ToLower(u.Host)
}

func (r *RuntimePreviewRegistry) validate(reg RuntimePreviewRegistration) (runtimePreviewEntry, error) {
	if r == nil || r.inspector == nil {
		return runtimePreviewEntry{}, errors.New("runtime preview supervisor unavailable; registration rejected")
	}
	if strings.TrimSpace(reg.RuntimeID) == "" || strings.TrimSpace(reg.ProjectID) == "" || reg.ProcessID <= 0 {
		return runtimePreviewEntry{}, errors.New("runtime preview registration requires runtime, project and supervised process")
	}
	u, port, err := parseRuntimePreviewURL(reg.URL)
	if err != nil {
		return runtimePreviewEntry{}, err
	}
	if reg.Port != 0 && reg.Port != port {
		return runtimePreviewEntry{}, errors.New("runtime preview URL port does not match supervised listener")
	}
	session, ok := r.inspector.RuntimeSession(reg.RuntimeID)
	if !ok || session.RuntimeID != reg.RuntimeID || session.ProjectID != reg.ProjectID || !session.IsActive() || !session.HostLive() {
		return runtimePreviewEntry{}, errors.New("runtime preview owner is not a live runtime in the requested project")
	}
	if session.PID <= 0 || !r.inspector.IsRuntimeDescendant(session, reg.ProcessID) {
		return runtimePreviewEntry{}, errors.New("runtime preview process is not supervised by the runtime")
	}
	processGen, ok := r.inspector.ProcessGeneration(reg.ProcessID)
	if !ok || strings.TrimSpace(processGen) == "" {
		return runtimePreviewEntry{}, errors.New("runtime preview process identity is unavailable")
	}
	if !r.inspector.OwnsListeningPort(reg.ProcessID, port) {
		return runtimePreviewEntry{}, errors.New("runtime preview process does not own the requested listening port")
	}
	reg.URL = u.String()
	reg.Port = port
	return runtimePreviewEntry{registration: reg, hostPID: session.HostPID, hostGen: session.HostGeneration, providerPID: session.PID, providerGen: session.ProviderGeneration, processGen: processGen}, nil
}

// Register accepts only a live Nexus-supervised process that owns the exact
// loopback listener in the URL. Registration is not exposed over browser MCP.
func (r *RuntimePreviewRegistry) Register(reg RuntimePreviewRegistration) error {
	entry, err := r.validate(reg)
	if err != nil {
		return err
	}
	r.mu.Lock()
	r.entries[reg.RuntimeID] = entry
	r.mu.Unlock()
	return nil
}

// ResolveOrRegister performs runtime-bound port discovery on the first open.
// The caller supplies a URL, but the inspector must find a listener on that
// exact port owned by a descendant of the runtime before it is admitted.
func (r *RuntimePreviewRegistry) ResolveOrRegister(runtimeID, projectID, requestedURL string) (RuntimePreviewRegistration, error) {
	if r == nil || r.inspector == nil {
		return RuntimePreviewRegistration{}, errors.New("runtime preview supervisor unavailable")
	}
	u, port, err := parseRuntimePreviewURL(requestedURL)
	if err != nil {
		return RuntimePreviewRegistration{}, err
	}
	session, ok := r.inspector.RuntimeSession(runtimeID)
	if !ok || session.ProjectID != projectID {
		return RuntimePreviewRegistration{}, errors.New("runtime preview owner is not a live runtime in the requested project")
	}
	r.mu.RLock()
	_, registered := r.entries[runtimeID]
	r.mu.RUnlock()
	if registered {
		resolved, resolveErr := r.Resolve(runtimeID, projectID, u.String())
		if resolveErr == nil && resolved.Port == port {
			return resolved, nil
		}
	}
	processID, generation, ok := r.inspector.FindRuntimeListener(session, port)
	if !ok || processID <= 0 || strings.TrimSpace(generation) == "" {
		return RuntimePreviewRegistration{}, errors.New("no runtime-supervised process owns the requested preview port")
	}
	reg := RuntimePreviewRegistration{RuntimeID: runtimeID, ProjectID: projectID, URL: u.String(), ProcessID: processID, Port: port}
	if err := r.Register(reg); err != nil {
		return RuntimePreviewRegistration{}, err
	}
	created, err := r.Resolve(runtimeID, projectID, u.String())
	if err != nil {
		return RuntimePreviewRegistration{}, err
	}
	r.mu.RLock()
	createdEntry, ok := r.entries[runtimeID]
	r.mu.RUnlock()
	if !ok || createdEntry.processGen != generation {
		r.Revoke(runtimeID)
		return RuntimePreviewRegistration{}, errors.New("runtime preview process identity changed during registration")
	}
	return created, nil
}

// Resolve revalidates process lineage and listener ownership on every access.
// A stale registration is deleted and can never authorize a different port.
func (r *RuntimePreviewRegistry) Resolve(runtimeID, projectID, requestedURL string) (RuntimePreviewRegistration, error) {
	if r == nil || r.inspector == nil {
		return RuntimePreviewRegistration{}, errors.New("runtime preview supervisor unavailable")
	}
	r.mu.RLock()
	entry, ok := r.entries[runtimeID]
	r.mu.RUnlock()
	if !ok || entry.registration.ProjectID != projectID {
		return RuntimePreviewRegistration{}, errors.New("no supervised preview is registered for this runtime and project")
	}
	current, err := r.validate(entry.registration)
	if err != nil || current.hostPID != entry.hostPID || current.hostGen != entry.hostGen || current.providerPID != entry.providerPID || current.providerGen != entry.providerGen || current.processGen != entry.processGen {
		r.Revoke(runtimeID)
		if err != nil {
			return RuntimePreviewRegistration{}, fmt.Errorf("registered runtime preview is no longer supervised: %w", err)
		}
		return RuntimePreviewRegistration{}, errors.New("runtime preview owner changed; registration revoked")
	}
	if strings.TrimSpace(requestedURL) != "" {
		wanted, _, parseErr := parseRuntimePreviewURL(requestedURL)
		registered, _, registeredErr := parseRuntimePreviewURL(entry.registration.URL)
		if parseErr != nil || registeredErr != nil || runtimePreviewOrigin(wanted) != runtimePreviewOrigin(registered) {
			return RuntimePreviewRegistration{}, errors.New("runtime preview navigation must stay on the supervised origin")
		}
	}
	return entry.registration, nil
}

func (r *RuntimePreviewRegistry) Revoke(runtimeID string) {
	if r == nil {
		return
	}
	r.mu.Lock()
	delete(r.entries, runtimeID)
	r.mu.Unlock()
}
