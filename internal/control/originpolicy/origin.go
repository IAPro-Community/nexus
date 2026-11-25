package originpolicy

import (
	"net"
	"net/url"
	"strings"
	"sync"
)

// allowedTunnelHosts is a set of hostnames that are allowed as tunnel endpoints.
// These are dynamically registered when a Cloudflare Quick Tunnel is started.
var (
	allowedTunnelHosts = make(map[string]struct{})
	allowedTunnelMu    sync.RWMutex
)

// RegisterTunnelHost adds a hostname to the allowed tunnel endpoints.
// Call when starting a Cloudflare Quick Tunnel.
func RegisterTunnelHost(hostname string) {
	allowedTunnelMu.Lock()
	defer allowedTunnelMu.Unlock()
	allowedTunnelHosts[strings.ToLower(strings.TrimSuffix(hostname, "."))] = struct{}{}
}

// UnregisterTunnelHost removes a hostname from the allowed tunnel endpoints.
// Call when stopping a Cloudflare Quick Tunnel.
func UnregisterTunnelHost(hostname string) {
	allowedTunnelMu.Lock()
	defer allowedTunnelMu.Unlock()
	delete(allowedTunnelHosts, strings.ToLower(strings.TrimSuffix(hostname, ".")))
}

func isAllowedTunnelHost(host string) bool {
	allowedTunnelMu.RLock()
	defer allowedTunnelMu.RUnlock()
	_, ok := allowedTunnelHosts[strings.ToLower(strings.TrimSuffix(host, "."))]
	return ok
}

// IsDesktopOrigin returns true if the origin is generated strictly by the native desktop shell (Wails).
func IsDesktopOrigin(u *url.URL) bool {
	if u == nil {
		return false
	}
	scheme := strings.ToLower(u.Scheme)
	hostname := strings.ToLower(strings.TrimSuffix(u.Hostname(), "."))

	switch scheme {
	case "wails":
		// Wails v3 serves the native WebView from wails://localhost, while
		// older shells use wails://wails. The wails scheme cannot be forged
		// by a regular browser, so localhost is safe only for this scheme.
		return hostname == "wails" || hostname == "wails.localhost" || hostname == "localhost"
	case "http", "https":
		return hostname == "wails" || hostname == "wails.localhost"
	default:
		return false
	}
}

func isDesktopOrigin(u *url.URL) bool {
	return IsDesktopOrigin(u)
}

// IsTrustedDesktopRequest verifies if the request comes strictly from the Wails native desktop shell
// and targets loopback or the internal wails host.
func IsTrustedDesktopRequest(requestHost, originHeader, refererHeader string) bool {
	reqHost, _ := splitHostPortLoose(requestHost)
	reqHost = strings.ToLower(strings.TrimSuffix(reqHost, "."))

	isLoopback := reqHost == "wails" || reqHost == "wails.localhost" || reqHost == "localhost" || reqHost == "nexus.localhost" || reqHost == "nexus.dev"
	if !isLoopback {
		ip := net.ParseIP(reqHost)
		isLoopback = ip != nil && ip.IsLoopback()
	}
	if !isLoopback {
		return false
	}

	originHeader = strings.TrimSpace(originHeader)
	refererHeader = strings.TrimSpace(refererHeader)

	// If Origin is provided, it is authoritative and MUST be valid.
	if originHeader != "" {
		u, err := url.Parse(originHeader)
		return err == nil && IsDesktopOrigin(u)
	}

	// If Origin is absent, fallback to Referer which MUST be valid.
	if refererHeader != "" {
		u, err := url.Parse(refererHeader)
		return err == nil && IsDesktopOrigin(u)
	}

	return false
}

// Validate accepts browser origins only when they target the exact HTTP Host
// of the request and that target is local/private. This supports SSH tunnels
// because the browser Host/Origin pair reflects the tunnel endpoint, while
// rejecting unrelated private-network origins.
func Validate(requestHost, origin string) bool {
	u, err := url.Parse(strings.TrimSpace(origin))
	if err != nil {
		return false
	}
	reqHost, reqPort := splitHostPortLoose(requestHost)
	reqHost = strings.ToLower(strings.TrimSuffix(reqHost, "."))

	// Allow native desktop WebView origins (Wails) targeting internal loopback or wails hosts.
	if isDesktopOrigin(u) {
		if reqHost == "wails" || reqHost == "wails.localhost" || reqHost == "localhost" || reqHost == "nexus.localhost" || reqHost == "nexus.dev" {
			return true
		}
		ip := net.ParseIP(reqHost)
		return ip != nil && ip.IsLoopback()
	}

	if (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" {
		return false
	}
	originHost := strings.ToLower(strings.TrimSuffix(u.Hostname(), "."))

	// A private, stable hostname (a tailnet name or a private DNS entry) must
	// be published explicitly by the operator. The publication defines one
	// exact address: same scheme, same hostname, same effective port.
	//
	// This check comes before the exact-host rule because a trusted proxy that
	// terminates TLS (tailscale serve and friends) rewrites the request Host
	// to the local service, so the browser origin and the request Host stop
	// being the same string. The publication is what restores the binding.
	if TrustedOriginAllowsRequest(strings.TrimSpace(origin), requestHost) {
		return true
	}

	if originHost != reqHost {
		return false
	}
	originPort := u.Port()
	if originPort == "" {
		if u.Scheme == "https" {
			originPort = "443"
		} else {
			originPort = "80"
		}
	}
	if reqPort != "" && reqPort != originPort {
		return false
	}
	if reqHost == "localhost" || reqHost == "nexus.localhost" || reqHost == "nexus.dev" {
		return true
	}
	if isAllowedTunnelHost(reqHost) {
		return true
	}
	// A private overlay address (RFC1918, ULA, link-local, or the CGNAT range
	// used by Tailscale and similar overlays) is reachable only from the
	// private network, so the exact-host rule above is sufficient.
	ip := net.ParseIP(reqHost)
	if ip != nil && isPrivateAddress(ip) {
		return true
	}
	return false
}

func splitHostPortLoose(value string) (string, string) {
	value = strings.TrimSpace(value)
	if host, port, err := net.SplitHostPort(value); err == nil {
		return strings.Trim(host, "[]"), port
	}
	return strings.Trim(value, "[]"), ""
}
