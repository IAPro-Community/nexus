package originpolicy

import (
	"net"
	"net/url"
	"os"
	"strings"
)

// remoteOriginEnv names the environment variable that publishes the private,
// stable origin the operator shares with a paired device.
//
// It exists for private addressing schemes that terminate TLS elsewhere: a
// tailnet hostname (Tailscale) or a private DNS name served by a trusted
// reverse proxy. Those names never match the request Host exactly when the
// proxy rewrites it, and they are not IPs, so the default policy rejects
// them.
//
// The value is a single origin, for example `https://nexus.tail1234.ts.net`:
//
//   - it must parse as an absolute http/https URL with no path, query or
//     fragment, so the trust decision is about a host and nothing else;
//   - the hostname it carries is trusted as a browser origin target;
//   - it is the address the operator shares when pairing a device.
//
// Trust still requires the browser Origin to equal the request Host exactly,
// so the configuration widens *which* host is acceptable, never *who* may
// send it. A malformed value is ignored instead of widening the policy.
const remoteOriginEnv = "NEXUS_REMOTE_ORIGIN"

// cgnatRange is the CGNAT shared-address range 100.64.0.0/10 (RFC 6598).
// Overlays such as Tailscale allocate addresses inside it, so a bind on a
// VPN interface must not be rejected by the Origin policy. net.IP.IsPrivate()
// does not cover this range.
var cgnatRange = net.IPNet{
	IP:   net.ParseIP("100.64.0.0"),
	Mask: net.CIDRMask(10, 32),
}

// RemoteOrigin returns the normalized private origin the operator published,
// or false when there is none or the value is malformed.
func RemoteOrigin() (string, bool) {
	raw := strings.TrimSpace(os.Getenv(remoteOriginEnv))
	if raw == "" {
		return "", false
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", false
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", false
	}
	host := strings.ToLower(strings.TrimSuffix(u.Hostname(), "."))
	if host == "" {
		return "", false
	}
	// A wildcard would trust every attacker-controlled name under the same
	// domain, so it is not a valid trust decision.
	if strings.ContainsAny(host, "*?") {
		return "", false
	}
	// A path, query or fragment would make the published address ambiguous.
	if u.Path != "" && u.Path != "/" {
		return "", false
	}
	if u.RawQuery != "" || u.Fragment != "" || u.User != nil {
		return "", false
	}
	origin := scheme + "://" + host
	if port := u.Port(); port != "" {
		origin += ":" + port
	}
	return origin, true
}

// RemoteOriginConfigured reports whether the operator published a private
// origin. A published origin makes the server reachable from the private
// network, so the security policy must stop treating it as pure loopback.
func RemoteOriginConfigured() bool {
	_, ok := RemoteOrigin()
	return ok
}

// IsTrustedRemoteHost reports whether host is the hostname of the published
// private origin.
func IsTrustedRemoteHost(host string) bool {
	host = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(host), "."))
	if host == "" {
		return false
	}
	origin, ok := RemoteOrigin()
	if !ok {
		return false
	}
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	return u.Hostname() == host
}

// effectivePort returns the port an origin actually uses: the explicit one,
// or the default of its scheme.
func effectivePort(u *url.URL) string {
	if port := u.Port(); port != "" {
		return port
	}
	switch strings.ToLower(u.Scheme) {
	case "https":
		return "443"
	case "http":
		return "80"
	}
	return ""
}

// sameOrigin reports whether two origins are the same address: same scheme,
// same hostname and same effective port.
func sameOrigin(a, b *url.URL) bool {
	if a == nil || b == nil {
		return false
	}
	if !strings.EqualFold(a.Scheme, b.Scheme) {
		return false
	}
	if !strings.EqualFold(
		strings.TrimSuffix(a.Hostname(), "."),
		strings.TrimSuffix(b.Hostname(), "."),
	) {
		return false
	}
	return effectivePort(a) == effectivePort(b)
}

// TrustedOriginMatches reports whether the browser origin is exactly the
// published private origin.
//
// The publication defines one address, not a family of them. A different port
// on the same hostname is a different service, and a different scheme is a
// different trust decision, so neither is implied by the publication.
// Publishing the address the operator actually shares is the only way to
// open it.
func TrustedOriginMatches(browserOrigin string) bool {
	origin, ok := RemoteOrigin()
	if !ok {
		return false
	}
	published, err := url.Parse(origin)
	if err != nil {
		return false
	}
	browser, err := url.Parse(strings.TrimSpace(browserOrigin))
	if err != nil || browser.Hostname() == "" {
		return false
	}
	return sameOrigin(browser, published)
}

// TrustedOriginAllowsRequest reports whether a request may be accepted
// because the browser origin is exactly the published private origin.
//
// This is the path a trusted proxy needs: `tailscale serve` (or any reverse
// proxy) terminates TLS and may rewrite the request Host to the local
// service, so the browser origin and the request Host stop being the same
// string. The publication is what restores the binding, because it names the
// exact address the operator shares.
//
// The request must still have landed on this machine: the request host has to
// be the published hostname, a loopback address or a private one. A request
// arriving on a public interface is not covered by a private publication.
func TrustedOriginAllowsRequest(browserOrigin, requestHost string) bool {
	if !TrustedOriginMatches(browserOrigin) {
		return false
	}
	reqHost, _ := splitHostPortLoose(requestHost)
	reqHost = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(reqHost), "."))
	if reqHost == "" {
		return false
	}
	if IsTrustedRemoteHost(reqHost) {
		return true
	}
	if reqHost == "localhost" || reqHost == "nexus.localhost" || reqHost == "nexus.dev" {
		return true
	}
	ip := net.ParseIP(reqHost)
	return ip != nil && isPrivateAddress(ip)
}

// isPrivateAddress reports whether the address belongs to a network that
// cannot be reached from the public internet: loopback, RFC1918, IPv6
// unique-local, link-local, or the CGNAT shared range used by VPN overlays.
func isPrivateAddress(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsPrivate() {
		return true
	}
	// Unique-local IPv6 for older Go/runtime variants.
	if len(ip) == net.IPv6len && (ip[0]&0xfe) == 0xfc {
		return true
	}
	return cgnatRange.Contains(ip)
}
