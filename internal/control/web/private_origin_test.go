package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/IAPro-Community/nexus/internal/control/originpolicy"
)

// In the private mode (Tailscale or a private DNS name served by a trusted
// proxy) there is no Cloudflare tunnel, but the operator still has to be able
// to pair a device. The invitation binds to the published private origin.
func TestPrivateRemoteOriginIssuesPairableInvitation(t *testing.T) {
	const origin = "https://nexus.tail1234.ts.net"
	t.Setenv("NEXUS_REMOTE_ORIGIN", origin)

	auth, _, err := NewAuthManager("127.0.0.1", "13000")
	if err != nil {
		t.Fatal(err)
	}
	srv := &Server{auth: auth, tunnel: &tunnelManager{}}

	url, bootstrapURL, code, err := srv.NewTunnelInviteURLWithRole(RoleController)
	if err != nil {
		t.Fatal(err)
	}
	if url != origin {
		t.Fatalf("invitation url = %q, want %q", url, origin)
	}
	if !strings.HasPrefix(bootstrapURL, origin+"/#nexus_bootstrap=") {
		t.Fatalf("bootstrap url = %q", bootstrapURL)
	}
	if len(code) != accessCodeLength {
		t.Fatalf("access code = %q", code)
	}

	// The paired device redeems the code against the published origin.
	sess, ok := auth.ExchangeAccessCode(code, "nexus.tail1234.ts.net")
	if !ok || sess == nil {
		t.Fatal("the pairing code must be redeemable on the published private origin")
	}
	if sess.Role != RoleController {
		t.Fatalf("role = %q", sess.Role)
	}
	if !sess.Remote {
		t.Fatal("a paired device must be marked as remote")
	}
}

// A viewer invitation keeps its read-only role in the private mode.
func TestPrivateRemoteOriginKeepsTheViewerRole(t *testing.T) {
	const origin = "https://nexus.tail1234.ts.net"
	t.Setenv("NEXUS_REMOTE_ORIGIN", origin)

	auth, _, err := NewAuthManager("127.0.0.1", "13000")
	if err != nil {
		t.Fatal(err)
	}
	srv := &Server{auth: auth, tunnel: &tunnelManager{}}

	_, _, code, err := srv.NewTunnelInviteURLWithRole(RoleViewer)
	if err != nil {
		t.Fatal(err)
	}
	sess, ok := auth.ExchangeAccessCode(code, "nexus.tail1234.ts.net")
	if !ok || sess == nil {
		t.Fatal("the pairing code must be redeemable on the published private origin")
	}
	if sess.Role != RoleViewer {
		t.Fatalf("role = %q, want %q", sess.Role, RoleViewer)
	}
	if sess.CanControl() {
		t.Fatal("a viewer session must not control the workspace")
	}
}

// Without a published private origin and without a tunnel there is no
// address to pair against, so no invitation may be issued.
func TestNoOriginMeansNoInvitation(t *testing.T) {
	t.Setenv("NEXUS_REMOTE_ORIGIN", "")

	auth, _, err := NewAuthManager("127.0.0.1", "13000")
	if err != nil {
		t.Fatal(err)
	}
	srv := &Server{auth: auth, tunnel: &tunnelManager{}}
	if _, _, _, err := srv.NewTunnelInviteURLWithRole(RoleController); err == nil {
		t.Fatal("no invitation may be issued without a reachable origin")
	}
}

// The invitation is bound to the published host: a device reaching the Nexus
// under another host must not redeem it.
func TestPrivateRemoteOriginRejectsAnotherHost(t *testing.T) {
	const origin = "https://nexus.tail1234.ts.net"
	t.Setenv("NEXUS_REMOTE_ORIGIN", origin)

	auth, _, err := NewAuthManager("127.0.0.1", "13000")
	if err != nil {
		t.Fatal(err)
	}
	srv := &Server{auth: auth, tunnel: &tunnelManager{}}
	_, _, code, err := srv.NewTunnelInviteURLWithRole(RoleController)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := auth.ExchangeAccessCode(code, "other.tail1234.ts.net"); ok {
		t.Fatal("an invitation must not be redeemable from another host")
	}
}

// A published private origin may carry a port. The browser sends that port in
// the Host header, so the invitation must still be redeemable: the binding is
// about the hostname the operator published, not about the port.
func TestPrivateRemoteOriginWithPortStaysPairable(t *testing.T) {
	const origin = "https://nexus.tail1234.ts.net:8443"
	t.Setenv("NEXUS_REMOTE_ORIGIN", origin)

	auth, _, err := NewAuthManager("127.0.0.1", "13000")
	if err != nil {
		t.Fatal(err)
	}
	srv := &Server{auth: auth, tunnel: &tunnelManager{}}
	_, _, code, err := srv.NewTunnelInviteURLWithRole(RoleController)
	if err != nil {
		t.Fatal(err)
	}
	// The browser reaches the origin with the port in the Host header.
	if _, ok := auth.ExchangeAccessCode(code, "nexus.tail1234.ts.net:8443"); !ok {
		t.Fatal("the pairing code must be redeemable on the published origin with its port")
	}
}

// The status must publish the private origin so the operator sees the stable
// address, and the pairing button becomes available without a tunnel.
func TestTunnelStatusPublishesThePrivateOrigin(t *testing.T) {
	const origin = "https://nexus.tail1234.ts.net"
	t.Setenv("NEXUS_REMOTE_ORIGIN", origin)

	auth, _, err := NewAuthManager("127.0.0.1", "13000")
	if err != nil {
		t.Fatal(err)
	}
	srv := &Server{auth: auth, tunnel: &tunnelManager{}}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1/api/v1/tunnel/status", nil)
	srv.handleTunnelStatus(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var status TunnelStatusResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &status); err != nil {
		t.Fatal(err)
	}
	if status.RemoteOrigin != origin {
		t.Fatalf("remote origin = %q, want %q", status.RemoteOrigin, origin)
	}
	if status.Active {
		t.Fatal("no tunnel is running in the private mode")
	}
	// A published private origin is stable by construction: it is a name the
	// operator controls, not a hostname the transport mints at every start.
	if !status.StableURL {
		t.Fatal("the published private origin must be reported as stable")
	}
}

// The private origin must reach the Origin policy: a browser on the tailnet
// hostname is an accepted origin, while a different hostname is not.
func TestPrivateRemoteOriginIsAnAcceptedBrowserOrigin(t *testing.T) {
	const origin = "https://nexus.tail1234.ts.net"
	t.Setenv("NEXUS_REMOTE_ORIGIN", origin)

	if !originpolicy.Validate("nexus.tail1234.ts.net", origin) {
		t.Fatal("the published private origin must be accepted as a browser origin")
	}
	if originpolicy.Validate("other.tail1234.ts.net", "https://other.tail1234.ts.net") {
		t.Fatal("an unpublished hostname must stay rejected")
	}
}

// The private mode must work through the real HTTP path the phone uses:
// the host issues the invitation, the device redeems the pairing code, and
// the resulting session is registered as a paired device that the host can
// revoke. This is the path the unit-level tests above do not cover.
func TestPrivateRemoteOriginPairsThroughTheHTTPPath(t *testing.T) {
	const origin = "https://nexus.tail1234.ts.net"
	t.Setenv("NEXUS_REMOTE_ORIGIN", origin)

	auth, _, err := NewAuthManager("127.0.0.1", "13000")
	if err != nil {
		t.Fatal(err)
	}
	srv := &Server{auth: auth, tunnel: &tunnelManager{}}

	// The host issues the invitation from its own session.
	hostSession, err := auth.CreateSession()
	if err != nil {
		t.Fatal(err)
	}
	invite := httptest.NewRequest(http.MethodPost, "https://"+origin+"/api/v1/tunnel/invite", nil)
	invite.Host = "nexus.tail1234.ts.net"
	invite.Header.Set("Origin", origin)
	invite.Header.Set(csrfHeaderName, hostSession.CSRFToken)
	invite.AddCookie(&http.Cookie{Name: sessionCookieName, Value: hostSession.ID})
	inviteRec := httptest.NewRecorder()
	srv.authMiddleware(srv.handleTunnelQR)(inviteRec, invite)
	if inviteRec.Code != http.StatusOK {
		t.Fatalf("invitation over HTTP = %d: %s", inviteRec.Code, inviteRec.Body.String())
	}
	var issued struct {
		URL        string `json:"url"`
		AccessCode string `json:"access_code"`
	}
	if err := json.Unmarshal(inviteRec.Body.Bytes(), &issued); err != nil {
		t.Fatal(err)
	}
	if issued.URL != origin {
		t.Fatalf("invitation url = %q, want %q", issued.URL, origin)
	}

	// The phone redeems the pairing code against the published origin.
	pair := httptest.NewRequest(http.MethodPost, "https://"+origin+"/api/v1/auth/pair",
		strings.NewReader(`{"code":"`+issued.AccessCode+`"}`))
	pair.Host = "nexus.tail1234.ts.net"
	pair.Header.Set("Origin", origin)
	pairRec := httptest.NewRecorder()
	srv.handleAuthPair(pairRec, pair)
	if pairRec.Code != http.StatusOK {
		t.Fatalf("pairing over HTTP = %d: %s", pairRec.Code, pairRec.Body.String())
	}
	cookies := pairRec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("pairing must set a session cookie")
	}
	var sessionCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == sessionCookieName {
			sessionCookie = cookie
		}
	}
	if sessionCookie == nil {
		t.Fatal("pairing must set the session cookie")
	}
	if !sessionCookie.Secure || !sessionCookie.HttpOnly {
		t.Fatalf("the private mode must issue a Secure, HttpOnly cookie: %+v", sessionCookie)
	}

	// The paired device is registered and the host can revoke it.
	devices := auth.RemoteDevices()
	if len(devices) != 1 {
		t.Fatalf("paired devices = %d, want 1", len(devices))
	}
	if devices[0].DeviceID == "" {
		t.Fatal("the paired device must carry an opaque device id")
	}
	if !auth.RevokeDevice(devices[0].DeviceID) {
		t.Fatal("the host must be able to revoke the paired device")
	}
	if len(auth.RemoteDevices()) != 0 {
		t.Fatal("the revoked device must leave the registry")
	}
}
