package web

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/originpolicy"
	"github.com/IAPro-Community/nexus/internal/core/entitlement"
	nexuscore "github.com/IAPro-Community/nexus/internal/nexus"
)

// TunnelStatusResponse is the contract returned by tunnel status/start/stop.
type TunnelStatusResponse struct {
	Active               bool   `json:"active"`
	Starting             bool   `json:"starting"`
	URL                  string `json:"url,omitempty"`
	BootstrapURL         string `json:"bootstrap_url,omitempty"`
	CloudflaredInstalled bool   `json:"cloudflared_installed"`
	CloudflaredVersion   string `json:"cloudflared_version,omitempty"`
	Error                string `json:"error,omitempty"`
}

// tunnelManager holds the runtime state for the Cloudflare Quick Tunnel.
type tunnelManager struct {
	mu          sync.Mutex
	tunnel      *Tunnel
	cancel      context.CancelFunc
	startCancel context.CancelFunc
	starting    bool
}

// trackTunnelExit ensures that an old cloudflared process cannot invalidate
// a newer tunnel generation's authentication when it exits asynchronously.
func tunnelIsActive(tunnel *Tunnel) bool {
	if tunnel == nil {
		return false
	}
	tunnel.mu.Lock()
	defer tunnel.mu.Unlock()
	return !tunnel.stopped
}

func (s *Server) trackTunnelExit(tunnel *Tunnel) {
	onExit := func() {
		tm := s.tunnelManager()
		tm.mu.Lock()
		owned := tm.tunnel == tunnel
		if owned {
			tm.tunnel = nil
			tm.cancel = nil
			if host := extractTunnelHost(tunnel.URL); host != "" {
				originpolicy.UnregisterTunnelHost(host)
			}
			// Keep revocation within the generation lock: a stale exit
			// must never disarm auth after a replacement tunnel starts.
			s.auth.SetTunnelActive(false)
		}
		tm.mu.Unlock()
	}
	// cloudflared's waiter reads OnExit under tunnel.mu. Register under
	// the same lock and handle an exit that already happened.
	tunnel.mu.Lock()
	tunnel.OnExit = onExit
	alreadyStopped := tunnel.stopped
	tunnel.mu.Unlock()
	if alreadyStopped {
		onExit()
	}
}

// AttachExternalTunnel makes CLI-started tunnels visible in the same status,
// invite, stop, and cleanup flow as tunnels launched from Settings.
func (s *Server) AttachExternalTunnel(tunnel *Tunnel) error {
	if tunnel == nil || extractTunnelHost(tunnel.URL) == "" ||
		extractTunnelHost(tunnel.URL) != s.tunnelHost || !s.auth.IsTunnelActive() {
		return fmt.Errorf("external tunnel does not match the active server hostname")
	}
	tm := s.tunnelManager()
	tm.mu.Lock()
	if tunnelIsActive(tm.tunnel) {
		tm.mu.Unlock()
		return fmt.Errorf("another tunnel is already active")
	}
	tm.tunnel = tunnel
	tm.cancel = tunnel.cancel
	tm.mu.Unlock()
	s.trackTunnelExit(tunnel)
	return nil
}

// NewTunnelInviteURL issues a fresh host-bound authorization link after the
// public tunnel has been registered, independent of server bootstrap age.
func (s *Server) NewTunnelInviteURL() (string, string, error) {
	tm := s.tunnelManager()
	tm.mu.Lock()
	active := !tm.starting && tunnelIsActive(tm.tunnel)
	url := ""
	if active {
		url = tm.tunnel.URL
	}
	tm.mu.Unlock()
	if !active || url == "" {
		return "", "", fmt.Errorf("no active tunnel")
	}
	token, err := s.auth.IssueTunnelInvitation(extractTunnelHost(url))
	if err != nil {
		return "", "", err
	}
	return url, url + "/#nexus_bootstrap=" + token, nil
}

func (s *Server) handleTunnelStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	installed := IsCloudflaredInstalled()
	version := ""
	if installed {
		p, _ := cloudflaredPath()
		version = CloudflaredVersion(p)
	}

	tm := s.tunnelManager()
	tm.mu.Lock()
	active := !tm.starting && tunnelIsActive(tm.tunnel)
	starting := tm.starting
	url := ""
	if active {
		url = tm.tunnel.URL
	}
	tm.mu.Unlock()

	writeJSON(w, http.StatusOK, TunnelStatusResponse{
		Active:               active,
		Starting:             starting,
		URL:                  url,
		CloudflaredInstalled: installed,
		CloudflaredVersion:   version,
	})
}

func (s *Server) handleTunnelStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := nexuscore.Default().RequireCapability(r.Context(), entitlement.CapRemoteAccess); err != nil {
		writeError(w, http.StatusForbidden, "remote access entitlement denied: "+err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancel()
	tm := s.tunnelManager()
	tm.mu.Lock()
	if tunnelIsActive(tm.tunnel) || tm.starting {
		tm.mu.Unlock()
		writeError(w, http.StatusConflict, "tunnel already active")
		return
	}
	// Install cancellation atomically with the pending-start flag, so a
	// concurrent Stop can always cancel DNS/download/process startup.
	tm.starting = true
	tm.startCancel = cancel
	tm.mu.Unlock()

	installed := IsCloudflaredInstalled()
	version := ""
	if installed {
		p, _ := cloudflaredPath()
		version = CloudflaredVersion(p)
	}
	defer func() {
		tm.mu.Lock()
		tm.startCancel = nil
		tm.mu.Unlock()
	}()

	tunnel, err := StartTunnel(ctx, s.port())
	if err != nil {
		tm.mu.Lock()
		tm.starting = false
		tm.mu.Unlock()
		writeJSON(w, http.StatusBadGateway, TunnelStatusResponse{
			CloudflaredInstalled: installed,
			CloudflaredVersion:   version,
			Error:                err.Error(),
		})
		return
	}

	// Stop may have cancelled a startup while StartTunnel was fetching or
	// launching cloudflared. Never publish a process after that cancellation.
	tm.mu.Lock()
	if err := ctx.Err(); err != nil {
		tm.starting = false
		tm.mu.Unlock()
		_ = tunnel.Stop()
		writeError(w, http.StatusConflict, "tunnel startup cancelled")
		return
	}
	// Arm auth fail-closed before readiness. Register the tunnel under the
	// same manager lock so concurrent Stop sees and terminates it.
	if tunnelHost := extractTunnelHost(tunnel.URL); tunnelHost != "" {
		originpolicy.RegisterTunnelHost(tunnelHost)
		s.tunnelHost = tunnelHost
	}
	s.auth.SetTunnelActive(true)
	tm.tunnel = tunnel
	tm.cancel = tunnel.cancel
	tm.mu.Unlock()
	s.trackTunnelExit(tunnel)

	readyCtx, readyCancel := context.WithTimeout(ctx, 60*time.Second)
	defer readyCancel()
	if err := tunnel.WaitForTunnel(readyCtx); err != nil {
		tm.mu.Lock()
		if tm.tunnel == tunnel {
			tm.tunnel = nil
			tm.cancel = nil
			if tunnelHost := extractTunnelHost(tunnel.URL); tunnelHost != "" {
				originpolicy.UnregisterTunnelHost(tunnelHost)
			}
			s.auth.SetTunnelActive(false)
		}
		tm.starting = false
		tm.mu.Unlock()
		_ = tunnel.Stop()
		writeJSON(w, http.StatusBadGateway, TunnelStatusResponse{
			CloudflaredInstalled: installed,
			CloudflaredVersion:   version,
			Error:                "tunnel readiness failed: " + err.Error(),
		})
		return
	}

	tm.mu.Lock()
	running := tm.tunnel == tunnel && tunnelIsActive(tunnel) && ctx.Err() == nil
	tm.starting = false
	if !running && tm.tunnel == tunnel {
		tm.tunnel = nil
		tm.cancel = nil
		if tunnelHost := extractTunnelHost(tunnel.URL); tunnelHost != "" {
			originpolicy.UnregisterTunnelHost(tunnelHost)
		}
		s.auth.SetTunnelActive(false)
	}
	tm.mu.Unlock()
	if !running {
		_ = tunnel.Stop()
		writeError(w, http.StatusBadGateway, "tunnel closed during startup")
		return
	}

	writeJSON(w, http.StatusOK, TunnelStatusResponse{
		Active:               true,
		URL:                  tunnel.URL,
		CloudflaredInstalled: installed,
		CloudflaredVersion:   version,
	})
}

func (s *Server) handleTunnelStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	tm := s.tunnelManager()
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Also cancel the pre-publication phase, when tm.tunnel is still nil.
	// Keep starting=true until the canceled Start handler has unwound.
	if tm.starting && tm.startCancel != nil {
		tm.startCancel()
	}
	if !tunnelIsActive(tm.tunnel) {
		writeJSON(w, http.StatusOK, TunnelStatusResponse{Active: false})
		return
	}

	// Unregister tunnel host
	if tm.tunnel.URL != "" {
		if tunnelHost := extractTunnelHost(tm.tunnel.URL); tunnelHost != "" {
			originpolicy.UnregisterTunnelHost(tunnelHost)
		}
	}

	_ = tm.tunnel.Stop()
	tm.tunnel = nil
	if tm.cancel != nil {
		tm.cancel()
		tm.cancel = nil
	}

	// Inform auth manager that tunnel is no longer active.
	s.auth.SetTunnelActive(false)

	writeJSON(w, http.StatusOK, TunnelStatusResponse{Active: false})
}

// extractTunnelHost extracts the hostname from a tunnel URL.
func extractTunnelHost(tunnelURL string) string {
	if strings.HasPrefix(tunnelURL, "https://") {
		url := strings.TrimPrefix(tunnelURL, "https://")
		if idx := strings.Index(url, "/"); idx >= 0 {
			return url[:idx]
		}
		return url
	}
	return ""
}

func (s *Server) port() int {
	// http.Server.Addr is empty when Serve is called with our pre-bound
	// listener. Cloudflared must use the actual TCP port, not DefaultPort.
	if s.listener != nil {
		if addr, ok := s.listener.Addr().(*net.TCPAddr); ok && addr.Port > 0 {
			return addr.Port
		}
	}
	return DefaultPort
}

// handleTunnelQR mints a fresh invitation on an authenticated POST.
// GET requests must never create or reveal credentials.
func (s *Server) handleTunnelQR(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	tunnelURL, bootstrapURL, err := s.NewTunnelInviteURL()
	if err != nil {
		writeError(w, http.StatusConflict, "unable to generate invitation: "+err.Error())
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, map[string]any{
		"url": tunnelURL,
		"bootstrap_url": bootstrapURL,
		"expires_in_seconds": int(remoteBootstrapTTL.Seconds()),
	})
}
