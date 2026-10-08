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
func (s *Server) trackTunnelExit(tunnel *Tunnel) {
	tunnel.OnExit = func() {
		tm := s.tunnelManager()
		tm.mu.Lock()
		owned := tm.tunnel == tunnel
		if owned {
			tm.tunnel = nil
			tm.cancel = nil
		}
		tm.mu.Unlock()
		if owned {
			if host := extractTunnelHost(tunnel.URL); host != "" {
				originpolicy.UnregisterTunnelHost(host)
			}
			s.auth.SetTunnelActive(false)
		}
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
	defer tm.mu.Unlock()
	if tm.tunnel != nil && !tm.tunnel.stopped {
		return fmt.Errorf("another tunnel is already active")
	}
	tm.tunnel = tunnel
	tm.cancel = tunnel.cancel
	s.trackTunnelExit(tunnel)
	return nil
}

// NewTunnelInviteURL issues a fresh host-bound authorization link after the
// public tunnel has been registered, independent of server bootstrap age.
func (s *Server) NewTunnelInviteURL() (string, string, error) {
	tm := s.tunnelManager()
	tm.mu.Lock()
	active := tm.tunnel != nil && !tm.tunnel.stopped
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
	active := tm.tunnel != nil && !tm.tunnel.stopped
	url := ""
	if active {
		url = tm.tunnel.URL
	}
	tm.mu.Unlock()

	writeJSON(w, http.StatusOK, TunnelStatusResponse{
		Active:               active,
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

	tm := s.tunnelManager()
	tm.mu.Lock()
	if tm.tunnel != nil && !tm.tunnel.stopped || tm.starting {
		tm.mu.Unlock()
		writeError(w, http.StatusConflict, "tunnel already active")
		return
	}
	tm.starting = true
	tm.mu.Unlock()

	installed := IsCloudflaredInstalled()
	version := ""
	if installed {
		p, _ := cloudflaredPath()
		version = CloudflaredVersion(p)
	}

	ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancel()
	tm.mu.Lock()
	tm.startCancel = cancel
	tm.mu.Unlock()
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

	// Arm tunnel auth fail-closed as soon as the public URL exists — before
	// WaitForTunnel — so bootstrap/query-token cannot stay reusable while the
	// hostname is already world-reachable.
	if tunnelHost := extractTunnelHost(tunnel.URL); tunnelHost != "" {
		originpolicy.RegisterTunnelHost(tunnelHost)
		s.tunnelHost = tunnelHost
	}
	s.auth.SetTunnelActive(true)

	s.trackTunnelExit(tunnel)

	readyCtx, readyCancel := context.WithTimeout(ctx, 60*time.Second)
	defer readyCancel()
	if err := tunnel.WaitForTunnel(readyCtx); err != nil {
		if tunnelHost := extractTunnelHost(tunnel.URL); tunnelHost != "" {
			originpolicy.UnregisterTunnelHost(tunnelHost)
		}
		s.auth.SetTunnelActive(false)
		_ = tunnel.Stop()
		tm.mu.Lock()
		tm.starting = false
		tm.mu.Unlock()
		writeJSON(w, http.StatusBadGateway, TunnelStatusResponse{
			CloudflaredInstalled: installed,
			CloudflaredVersion:   version,
			Error:                "tunnel readiness failed: " + err.Error(),
		})
		return
	}

	tm.mu.Lock()
	tm.tunnel = tunnel
	tm.cancel = tunnel.cancel
	tm.starting = false
	tm.mu.Unlock()

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

	if tm.tunnel == nil || tm.tunnel.stopped {
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
