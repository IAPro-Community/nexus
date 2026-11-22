package web

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/originpolicy"
	"github.com/gorilla/websocket"
)

const (
	sessionCookieName = "ai_control_session"
	deviceCookieName  = "ai_control_device"
	csrfHeaderName    = "X-CSRF-Token"
	sessionTTL        = 24 * time.Hour
	sessionIdleTTL    = 4 * time.Hour
	deviceCookieTTL   = 90 * 24 * time.Hour

	// Remote pairing codes are Crockford base32: 32^8 ≈ 2^40 candidates,
	// visually unambiguous (no I/L/O/U) and typo-tolerant on entry.
	accessCodeLength      = 8
	accessCodeAlphabet    = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"
	accessCodeMaxFailures = 10

	// RoleController grants full remote control (commands, terminals, writes).
	// RoleViewer is a read-only session: it may read the workspace but every
	// mutating request is refused. The owner picks the role when issuing the
	// invitation, so a device can never escalate itself.
	RoleController = "controller"
	RoleViewer     = "viewer"
)

type Session struct {
	ID           string
	CSRFToken    string
	CreatedAt    time.Time
	ExpiresAt    time.Time
	LastActiveAt time.Time
	// A session granted while remotely exposed must never become persistent
	// just because the tunnel was subsequently switched off.
	NonPersistable bool
	// Remote device metadata, filled when a tunnel bootstrap or access-code
	// pairing mints the session. DeviceID is an opaque, non-secret handle so
	// the owner can revoke one device without exposing the bearer session ID.
	Remote      bool
	DeviceID    string
	DeviceLabel string
	DeviceKind  string
	// Role is the permission level granted by the invitation that created
	// this session. Empty means the local host session, which is always a
	// full controller.
	Role string
}

// CanControl reports whether the session may perform mutating operations.
// Only sessions explicitly granted RoleController (or the local host session)
// may write; a viewer session is read-only.
func (s *Session) CanControl() bool {
	if s == nil {
		return false
	}
	if !s.Remote {
		return true
	}
	return s.Role == RoleController
}

// RemoteDevice is the owner-visible, redacted view of one paired device.
type RemoteDevice struct {
	DeviceID   string    `json:"device_id"`
	Label      string    `json:"label"`
	Kind       string    `json:"kind"`
	PairedAt   time.Time `json:"paired_at"`
	LastSeenAt time.Time `json:"last_seen_at"`
	ExpiresAt  time.Time `json:"expires_at"`
}

type remoteInvitation struct {
	Host      string
	ExpiresAt time.Time
	// AccessCode is the short human-typed alternative to the QR bootstrap
	// link. It shares the invitation's host binding, TTL and single-use
	// semantics and is never written to logs or persisted storage.
	AccessCode string
	// Role is the permission level granted to the device that redeems this
	// invitation. It is chosen by the owner when the invitation is issued
	// and can never be changed by the remote device itself.
	Role string
}

type AuthManager struct {
	mu                 sync.RWMutex
	bootstrapToken     string
	bootstrapCreatedAt time.Time
	usedBootstrap      bool
	sessions           map[string]*Session
	desktopSession     *Session
	listenHost         string
	listenPort         string
	storeDir           string
	lastPersist        time.Time
	entropy            io.Reader
	tunnelActive       bool // true when a Cloudflare Quick Tunnel exposes this server to the internet
	clock              func() time.Time
	deviceToken        string
	remoteInvitations  map[string]remoteInvitation
	// accessCodeFailures counts consecutive rejected pairing codes since the
	// last successful exchange or tunnel generation change. It is a global
	// brute-force guard: when the cap trips, every pending invitation is
	// destroyed and the owner must issue a fresh one.
	accessCodeFailures int
	sessionWatchers    map[string]map[chan struct{}]struct{}
}

func (a *AuthManager) SetDesktopSession(sess *Session) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.desktopSession != nil && (sess == nil || a.desktopSession.ID != sess.ID) {
		delete(a.sessions, a.desktopSession.ID)
		a.notifySessionRevokedLocked(a.desktopSession.ID)
	}
	a.desktopSession = sess
	if sess != nil {
		a.sessions[sess.ID] = sess
	}
	_ = a.persistLocked()
}

func (a *AuthManager) GetDesktopSession() *Session {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.desktopSession == nil {
		return nil
	}
	if _, exists := a.sessions[a.desktopSession.ID]; !exists {
		a.desktopSession = nil
		return nil
	}
	now := time.Now()
	if now.After(a.desktopSession.ExpiresAt) || now.Sub(a.desktopSession.LastActiveAt) > sessionIdleTTL {
		delete(a.sessions, a.desktopSession.ID)
		a.notifySessionRevokedLocked(a.desktopSession.ID)
		a.desktopSession = nil
		_ = a.persistLocked()
		return nil
	}
	return a.desktopSession
}

func NewAuthManager(listenHost, listenPort string) (*AuthManager, string, error) {
	return NewAuthManagerWithStore(listenHost, listenPort, "")
}

// NewAuthManagerWithStore restores loopback sessions from storeDir so a
// browser cookie survives `nexus web` restart on this machine.
func NewAuthManagerWithStore(listenHost, listenPort, storeDir string) (*AuthManager, string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return nil, "", err
	}
	bootstrapToken := hex.EncodeToString(bytes)

	auth := &AuthManager{
		bootstrapToken:     bootstrapToken,
		bootstrapCreatedAt: time.Now(),
		sessions:           make(map[string]*Session),
		sessionWatchers:    make(map[string]map[chan struct{}]struct{}),
		listenHost:         listenHost,
		listenPort:         listenPort,
		storeDir:           strings.TrimSpace(storeDir),
		entropy:            rand.Reader,
	}
	auth.mu.Lock()
	auth.loadPersistedLocked()
	auth.loadDeviceTokenLocked()
	auth.mu.Unlock()
	return auth, bootstrapToken, nil
}

// CheckOrigin verifies that browser-originated requests target the exact Host
// seen by the HTTP server. Direct non-browser HTTP clients may omit Origin,
// while WebSocket upgrades must always provide it.
func CheckOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		referer := r.Header.Get("Referer")
		if referer == "" {
			if websocket.IsWebSocketUpgrade(r) || strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
				return false
			}
			return true
		}
		u, err := url.Parse(referer)
		if err != nil {
			return false
		}
		origin = u.Scheme + "://" + u.Host
	}
	return originpolicy.Validate(r.Host, origin)
}

func (a *AuthManager) ValidateOrigin(r *http.Request) bool {
	return CheckOrigin(r)
}

// requireExactOrigin rejects empty Origin. Browser POSTs always send it;
// anonymous curl to loopback-reauth must not succeed by omitting the header.
func requireExactOrigin(r *http.Request) bool {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return false
	}
	return originpolicy.Validate(r.Host, origin)
}

func sameOriginFetch(r *http.Request) bool {
	site := strings.TrimSpace(r.Header.Get("Sec-Fetch-Site"))
	if site == "" {
		return true
	}
	return strings.EqualFold(site, "same-origin")
}

func (a *AuthManager) CreateSession() (*Session, error) {
	bytes := make([]byte, 32)
	if _, err := io.ReadFull(a.entropyReader(), bytes); err != nil {
		return nil, err
	}
	sessID := hex.EncodeToString(bytes)

	csrfBytes := make([]byte, 16)
	if _, err := io.ReadFull(a.entropyReader(), csrfBytes); err != nil {
		return nil, err
	}
	csrfToken := hex.EncodeToString(csrfBytes)

	now := time.Now()
	sess := &Session{
		ID:           sessID,
		CSRFToken:    csrfToken,
		CreatedAt:    now,
		ExpiresAt:    now.Add(sessionTTL),
		LastActiveAt: now,
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	sess.NonPersistable = a.effectiveExposureLocked() != ExposureLoopback
	a.sessions[sessID] = sess
	_ = a.persistLocked()
	return sess, nil
}

func (a *AuthManager) AuthenticateRequest(r *http.Request) *Session {
	var token string
	if cookie, err := r.Cookie(sessionCookieName); err == nil && cookie.Value != "" {
		token = cookie.Value
	}
	if token == "" {
		if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
			token = strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
		}
	}
	if token == "" {
		token = strings.TrimSpace(r.Header.Get("X-Nexus-Session"))
	}
	wsUpgrade := websocket.IsWebSocketUpgrade(r) || strings.EqualFold(r.Header.Get("Upgrade"), "websocket")

	a.mu.Lock()
	defer a.mu.Unlock()

	if token == "" && wsUpgrade {
		// Browser WebSocket cannot set Authorization; query tokens remain the
		// transport on loopback only. Reject them on private --remote binds and
		// when a Quick Tunnel is active so session IDs are not forwarded through
		// LAN sniffers or public intermediaries.
		if isLoopbackHost(a.listenHost) && !a.tunnelActive {
			token = strings.TrimSpace(r.URL.Query().Get("token"))
			if token == "" {
				token = strings.TrimSpace(r.URL.Query().Get("session"))
			}
		}
	}

	// Desktop sessions are provisioned by CreateDesktopSession and handed to the
	// Wails shell via GetBootstrapInfo / /api/v1/desktop/bootstrap. Callers must
	// present that session as cookie, Bearer, or X-Nexus-Session. Origin/Referer
	// alone must never grant authentication — those headers are trivial to spoof
	// from any local process talking to loopback.
	if token == "" {
		return nil
	}

	sess, ok := a.sessions[token]
	if !ok {
		return nil
	}

	now := time.Now()
	// Check absolute expiry and idle TTL (A8 security requirement)
	if now.After(sess.ExpiresAt) || now.Sub(sess.LastActiveAt) > sessionIdleTTL {
		if a.desktopSession != nil && a.desktopSession.ID == token {
			a.desktopSession = nil
		}
		delete(a.sessions, token)
		a.notifySessionRevokedLocked(token)
		_ = a.persistLocked()
		return nil
	}

	// A viewer session is read-only. The denial is enforced by the request
	// dispatchers (authMiddleware, plan/run routes, terminal upgrade) so no
	// individual handler can forget the check.

	sess.LastActiveAt = now
	if a.storeDir != "" && now.Sub(a.lastPersist) >= sessionPersistMinGap {
		_ = a.persistLocked()
	}
	return sess
}

// WatchSession returns a channel that closes when the session is revoked or
// expires. The boolean is false when the session is already inactive.
func (a *AuthManager) WatchSession(sessionID string) (<-chan struct{}, func(), bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	sess, ok := a.sessions[sessionID]
	if !ok {
		return nil, func() {}, false
	}
	now := a.now()
	if now.After(sess.ExpiresAt) || now.Sub(sess.LastActiveAt) > sessionIdleTTL {
		if a.desktopSession != nil && a.desktopSession.ID == sessionID {
			a.desktopSession = nil
		}
		delete(a.sessions, sessionID)
		a.notifySessionRevokedLocked(sessionID)
		_ = a.persistLocked()
		return nil, func() {}, false
	}
	ch := make(chan struct{})
	if a.sessionWatchers == nil {
		a.sessionWatchers = make(map[string]map[chan struct{}]struct{})
	}
	if a.sessionWatchers[sessionID] == nil {
		a.sessionWatchers[sessionID] = make(map[chan struct{}]struct{})
	}
	a.sessionWatchers[sessionID][ch] = struct{}{}
	cleanup := func() {
		a.mu.Lock()
		defer a.mu.Unlock()
		watchers := a.sessionWatchers[sessionID]
		delete(watchers, ch)
		if len(watchers) == 0 {
			delete(a.sessionWatchers, sessionID)
		}
	}
	return ch, cleanup, true
}

// SessionActive reports whether a session still satisfies its absolute and
// idle expiry limits. It does not refresh LastActiveAt.
func (a *AuthManager) SessionActive(sessionID string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	sess, ok := a.sessions[sessionID]
	if !ok {
		return false
	}
	now := a.now()
	if now.After(sess.ExpiresAt) || now.Sub(sess.LastActiveAt) > sessionIdleTTL {
		if a.desktopSession != nil && a.desktopSession.ID == sessionID {
			a.desktopSession = nil
		}
		delete(a.sessions, sessionID)
		a.notifySessionRevokedLocked(sessionID)
		_ = a.persistLocked()
		return false
	}
	return true
}

// TouchSession refreshes idle activity after an authenticated WebSocket frame.
func (a *AuthManager) TouchSession(sessionID string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	sess, ok := a.sessions[sessionID]
	if !ok {
		return false
	}
	now := a.now()
	if now.After(sess.ExpiresAt) || now.Sub(sess.LastActiveAt) > sessionIdleTTL {
		if a.desktopSession != nil && a.desktopSession.ID == sessionID {
			a.desktopSession = nil
		}
		delete(a.sessions, sessionID)
		a.notifySessionRevokedLocked(sessionID)
		_ = a.persistLocked()
		return false
	}
	sess.LastActiveAt = now
	if a.storeDir != "" && now.Sub(a.lastPersist) >= sessionPersistMinGap {
		_ = a.persistLocked()
	}
	return true
}

func (a *AuthManager) notifySessionRevokedLocked(sessionID string) {
	for ch := range a.sessionWatchers[sessionID] {
		close(ch)
	}
	delete(a.sessionWatchers, sessionID)
}

// RevokeSession destroys an existing authenticated session (logout).
func (a *AuthManager) RevokeSession(sessionID string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.desktopSession != nil && a.desktopSession.ID == sessionID {
		a.desktopSession = nil
	}
	delete(a.sessions, sessionID)
	a.notifySessionRevokedLocked(sessionID)
	_ = a.persistLocked()
}

// AnnotateDevice records which device owns a freshly paired remote session and
// returns its opaque device handle. Only a derived label and a coarse kind are
// stored: the raw User-Agent and the client IP are never persisted. Loopback
// sessions are not remote devices and are ignored.
func (a *AuthManager) AnnotateDevice(sessionID, label, kind string) (string, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.effectiveExposureLocked() == ExposureLoopback {
		return "", false
	}
	sess, ok := a.sessions[sessionID]
	if !ok || sess == nil {
		return "", false
	}
	raw := make([]byte, 8)
	if _, err := io.ReadFull(a.entropyReader(), raw); err != nil {
		return "", false
	}
	if len(label) > 64 {
		label = label[:64]
	}
	sess.Remote = true
	sess.DeviceID = hex.EncodeToString(raw)
	sess.DeviceLabel = label
	sess.DeviceKind = kind
	if sess.Role == "" {
		sess.Role = RoleController
	}
	return sess.DeviceID, true
}

// RemoteDevices lists paired devices for the Settings panel, pruning sessions
// that already expired so the list never advertises a dead device.
func (a *AuthManager) RemoteDevices() []RemoteDevice {
	a.mu.Lock()
	defer a.mu.Unlock()
	now := a.now()
	devices := make([]RemoteDevice, 0, len(a.sessions))
	for id, sess := range a.sessions {
		if sess == nil || !sess.Remote {
			continue
		}
		if now.After(sess.ExpiresAt) || now.Sub(sess.LastActiveAt) > sessionIdleTTL {
			delete(a.sessions, id)
			a.notifySessionRevokedLocked(id)
			continue
		}
		devices = append(devices, RemoteDevice{
			DeviceID:   sess.DeviceID,
			Label:      sess.DeviceLabel,
			Kind:       sess.DeviceKind,
			PairedAt:   sess.CreatedAt,
			LastSeenAt: sess.LastActiveAt,
			ExpiresAt:  sess.ExpiresAt,
		})
	}
	sort.Slice(devices, func(i, j int) bool {
		return devices[i].LastSeenAt.After(devices[j].LastSeenAt)
	})
	return devices
}

// RevokeDevice revokes exactly one paired device by its opaque handle, leaving
// every other session untouched. Unknown handles and local sessions are
// refused so this can never be used to log out the host.
func (a *AuthManager) RevokeDevice(deviceID string) bool {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	for id, sess := range a.sessions {
		if sess == nil || !sess.Remote || sess.DeviceID != deviceID {
			continue
		}
		delete(a.sessions, id)
		a.notifySessionRevokedLocked(id)
		_ = a.persistLocked()
		return true
	}
	return false
}

// DeviceLabelFromUserAgent derives a short human label such as "Chrome ·
// Android" plus a coarse kind, without keeping the raw User-Agent string.
func DeviceLabelFromUserAgent(userAgent string) (label, kind string) {
	ua := strings.ToLower(userAgent)
	browser := "Navegador"
	switch {
	case strings.Contains(ua, "edg/"):
		browser = "Edge"
	case strings.Contains(ua, "opr/") || strings.Contains(ua, "opera"):
		browser = "Opera"
	case strings.Contains(ua, "firefox") || strings.Contains(ua, "fxios"):
		browser = "Firefox"
	case strings.Contains(ua, "crios") || strings.Contains(ua, "chrome"):
		browser = "Chrome"
	case strings.Contains(ua, "safari"):
		browser = "Safari"
	}
	system := "Desconhecido"
	kind = "desktop"
	switch {
	case strings.Contains(ua, "android"):
		system, kind = "Android", "mobile"
	case strings.Contains(ua, "iphone") || strings.Contains(ua, "ipod"):
		system, kind = "iOS", "mobile"
	case strings.Contains(ua, "ipad"):
		system, kind = "iPadOS", "tablet"
	case strings.Contains(ua, "windows"):
		system = "Windows"
	case strings.Contains(ua, "mac os") || strings.Contains(ua, "macintosh"):
		system = "macOS"
	case strings.Contains(ua, "linux"):
		system = "Linux"
	}
	return browser + " · " + system, kind
}

// sessionRole exposes the effective permission level of a session to the
// client. The local host session is always a controller.
func sessionRole(sess *Session) string {
	if sess == nil {
		return ""
	}
	if !sess.Remote {
		return RoleController
	}
	if sess.Role == RoleViewer {
		return RoleViewer
	}
	return RoleController
}

// RotateSession atomically creates a new session ID/CSRF token and deletes the old one.
func (a *AuthManager) RotateSession(oldSessionID string) (*Session, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	bytes := make([]byte, 32)
	if _, err := io.ReadFull(a.entropyReader(), bytes); err != nil {
		return nil, err
	}
	sessID := hex.EncodeToString(bytes)

	csrfBytes := make([]byte, 16)
	if _, err := io.ReadFull(a.entropyReader(), csrfBytes); err != nil {
		return nil, err
	}
	csrfToken := hex.EncodeToString(csrfBytes)

	now := time.Now()
	previous := a.sessions[oldSessionID]
	nonPersistable := a.effectiveExposureLocked() != ExposureLoopback
	if previous != nil && previous.NonPersistable {
		nonPersistable = true
	}
	sess := &Session{
		ID:             sessID,
		CSRFToken:      csrfToken,
		CreatedAt:      now,
		ExpiresAt:      now.Add(sessionTTL),
		LastActiveAt:   now,
		NonPersistable: nonPersistable,
	}
	delete(a.sessions, oldSessionID)
	a.notifySessionRevokedLocked(oldSessionID)
	a.sessions[sessID] = sess
	if a.desktopSession != nil && a.desktopSession.ID == oldSessionID {
		a.desktopSession = sess
	}
	_ = a.persistLocked()
	return sess, nil
}

// SetTunnelActive informs the auth manager that a Cloudflare Quick Tunnel
// is exposing this server to the internet. Effective exposure becomes
// PUBLIC_REMOTE: bootstrap is one-time, cookies must be Secure, and sessions
// must not be persisted for later local restore.
func (a *AuthManager) SetTunnelActive(active bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.tunnelActive != active {
		a.remoteInvitations = nil // revoke links when tunnel stops or changes generation
		a.accessCodeFailures = 0
	}
	a.tunnelActive = active
	if active {
		// Drop any on-disk session cache so a restart cannot revive remote
		// privileges minted while the tunnel was open.
		_ = a.clearPersistedSessionsLocked()
	}
}

// IsTunnelActive reports whether a Cloudflare Quick Tunnel is currently
// exposing this server to the internet.
func (a *AuthManager) IsTunnelActive() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.tunnelActive
}

// normalizePairingHost canonicalizes a request Host for comparison with the
// host an invitation is bound to: lowercase, no trailing dot and no port.
//
// The port is dropped on purpose. A published private origin may carry one
// (https://nexus.tail1234.ts.net:8443), and the browser sends the port in the
// Host header while the invitation is bound to the hostname. The port is not
// a security boundary here: the Origin policy still requires the browser
// Origin to match the request Host exactly, so the binding stays about the
// hostname the operator published.
func normalizePairingHost(value string) string {
	host := strings.ToLower(strings.TrimSpace(value))
	if host == "" {
		return ""
	}
	if parsed, _, err := net.SplitHostPort(host); err == nil {
		host = parsed
	}
	host = strings.TrimSuffix(host, ".")
	return strings.ToLower(strings.TrimSuffix(strings.Trim(host, "[]"), "."))
}

// remotePairingHostLocked resolves the host an invitation binds to.
//
// While a Cloudflare tunnel is running, that is the live tunnel hostname.
// Otherwise it is the published private origin (a tailnet hostname or a
// private DNS name), which is what makes the private-overlay mode pairable
// without a public tunnel.
//
// The caller must hold a.mu.
func (a *AuthManager) remotePairingHostLocked(host string) (string, bool) {
	host = normalizePairingHost(host)
	if a.tunnelActive {
		if host == "" {
			return "", false
		}
		return host, true
	}
	if origin, ok := originpolicy.RemoteOrigin(); ok {
		u, err := url.Parse(origin)
		if err != nil {
			return "", false
		}
		return normalizePairingHost(u.Hostname()), true
	}
	return "", false
}

// IssueTunnelInvitation creates a fresh one-time link for the active public
// tunnel, with TTL starting at invitation creation instead of server startup.
// It also mints the short pairing code shown in Settings as a QR alternative.
// The invitation grants full control; use IssueTunnelInvitationWithRole to
// issue a read-only (viewer) invitation.
func (a *AuthManager) IssueTunnelInvitation(host string) (string, string, error) {
	return a.IssueTunnelInvitationWithRole(host, RoleController)
}

// IssueTunnelInvitationWithRole creates a one-time invitation bound to a
// permission role. RoleViewer produces a read-only session: the paired device
// can observe the workspace but cannot run commands or mutate anything.
// The role is fixed at issuance time and travels with the invitation, so a
// remote device cannot escalate to controller by re-pairing.
//
// The invitation is bound to the host the paired device will use. That host is
// the live tunnel hostname while a Cloudflare tunnel is running, and the
// published private origin otherwise (a tailnet hostname or a private DNS
// name). Without either, no invitation exists: the pair would have no
// reachable address to bind to.
func (a *AuthManager) IssueTunnelInvitationWithRole(host, role string) (string, string, error) {
	role = strings.TrimSpace(strings.ToLower(role))
	if role == "" {
		role = RoleController
	}
	if role != RoleController && role != RoleViewer {
		return "", "", fmt.Errorf("unsupported remote role")
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	host, ok := a.remotePairingHostLocked(host)
	if !ok {
		return "", "", fmt.Errorf("tunnel invitation unavailable")
	}
	now := a.now()
	for token, grant := range a.remoteInvitations {
		if !now.Before(grant.ExpiresAt) {
			delete(a.remoteInvitations, token)
		}
	}
	if len(a.remoteInvitations) >= 12 {
		return "", "", fmt.Errorf("too many pending tunnel invitations")
	}
	secret := make([]byte, 32)
	if _, err := io.ReadFull(a.entropyReader(), secret); err != nil {
		return "", "", err
	}
	code, err := generateAccessCode(a.entropyReader())
	if err != nil {
		return "", "", err
	}
	if a.remoteInvitations == nil {
		a.remoteInvitations = make(map[string]remoteInvitation)
	}
	token := hex.EncodeToString(secret)
	a.remoteInvitations[token] = remoteInvitation{
		Host:       host,
		ExpiresAt:  now.Add(remoteBootstrapTTL),
		AccessCode: code,
		Role:       role,
	}
	return token, code, nil
}

// generateAccessCode draws accessCodeLength unbiased characters from the
// Crockford alphabet. The alphabet size (32) divides 256 evenly, so modulo
// mapping introduces no distribution bias.
func generateAccessCode(reader io.Reader) (string, error) {
	buf := make([]byte, accessCodeLength)
	if _, err := io.ReadFull(reader, buf); err != nil {
		return "", err
	}
	out := make([]byte, accessCodeLength)
	for i, b := range buf {
		out[i] = accessCodeAlphabet[int(b)%len(accessCodeAlphabet)]
	}
	return string(out), nil
}

// normalizeAccessCode canonicalizes user-typed pairing input: uppercase,
// punctuation/whitespace removed and Crockford ambiguous letters remapped
// (I/L→1, O→0) so printed and typed codes always match.
func normalizeAccessCode(code string) string {
	code = strings.ToUpper(code)
	var b strings.Builder
	b.Grow(len(code))
	for _, r := range code {
		switch {
		case r == 'I' || r == 'L':
			b.WriteByte('1')
		case r == 'O':
			b.WriteByte('0')
		case (r >= '0' && r <= '9') || (r >= 'A' && r <= 'Z'):
			b.WriteRune(r)
		}
	}
	return b.String()
}

func (a *AuthManager) ExchangeBootstrapToken(token string) (*Session, bool) {
	return a.ExchangeBootstrapTokenForHost(token, "")
}

// ExchangeBootstrapTokenForHost also exchanges host-bound one-time invitations,
// while preserving CLI process-bootstrap and loopback behavior.
func (a *AuthManager) ExchangeBootstrapTokenForHost(token, host string) (*Session, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if token == "" {
		return nil, false
	}
	exposure := a.effectiveExposureLocked()
	reusable := exposure == ExposureLoopback
	consumeInvitation := false
	grantRole := RoleController
	if grant, exists := a.remoteInvitations[token]; exists {
		if !a.now().Before(grant.ExpiresAt) {
			return nil, false
		}
		// The invitation is bound to the host the paired device uses. The
		// live tunnel hostname while a tunnel runs, the published private
		// origin otherwise.
		if normalizePairingHost(host) != grant.Host {
			return nil, false
		}
		consumeInvitation = true
		reusable = false
		if grant.Role == RoleViewer {
			grantRole = RoleViewer
		}
	} else {
		if token != a.bootstrapToken {
			return nil, false
		}
		if !a.bootstrapCreatedAt.IsZero() {
			ttl := remoteBootstrapTTL
			if reusable {
				ttl = loopbackBootstrapTTL
			}
			if a.now().Sub(a.bootstrapCreatedAt) > ttl {
				return nil, false
			}
		}
		if a.usedBootstrap && !reusable {
			return nil, false
		}
	}

	sess, ok := a.newSessionLocked(exposure)
	if !ok {
		return nil, false
	}
	if consumeInvitation {
		// The invitation fixes the permission level: a viewer device stays a
		// viewer for the whole session, regardless of what it asks for later.
		sess.Role = grantRole
		sess.Remote = true
		delete(a.remoteInvitations, token)
	} else if !reusable {
		a.usedBootstrap = true
	}
	return sess, true
}

// newSessionLocked mints and registers a fresh session for a bootstrap or
// pairing exchange. Sessions minted under PUBLIC_REMOTE / PRIVATE_NETWORK are
// never persisted.
func (a *AuthManager) newSessionLocked(exposure ExposureMode) (*Session, bool) {
	bytes := make([]byte, 32)
	if _, err := io.ReadFull(a.entropyReader(), bytes); err != nil {
		return nil, false
	}
	sessID := hex.EncodeToString(bytes)

	csrfBytes := make([]byte, 16)
	if _, err := io.ReadFull(a.entropyReader(), csrfBytes); err != nil {
		return nil, false
	}
	csrfToken := hex.EncodeToString(csrfBytes)

	now := a.now()
	sess := &Session{
		ID:             sessID,
		CSRFToken:      csrfToken,
		CreatedAt:      now,
		ExpiresAt:      now.Add(sessionTTL),
		LastActiveAt:   now,
		NonPersistable: exposure != ExposureLoopback,
	}
	a.sessions[sessID] = sess
	if exposure == ExposureLoopback {
		_ = a.persistLocked()
	}
	return sess, true
}

// ExchangeAccessCode authenticates a remote browser that typed the short
// pairing code displayed next to the tunnel QR link. The code inherits the
// invitation's host binding, TTL and single-use semantics. Repeated failures
// trip a global brute-force guard that revokes every pending invitation.
func (a *AuthManager) ExchangeAccessCode(code, host string) (*Session, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	normalized := normalizeAccessCode(code)
	if len(normalized) != accessCodeLength {
		return nil, false
	}
	now := a.now()
	matchedToken := ""
	var matched remoteInvitation
	for token, grant := range a.remoteInvitations {
		if !now.Before(grant.ExpiresAt) {
			delete(a.remoteInvitations, token)
			continue
		}
		if subtle.ConstantTimeCompare([]byte(normalized), []byte(grant.AccessCode)) == 1 {
			matchedToken = token
			matched = grant
			break
		}
	}
	if matchedToken == "" {
		a.accessCodeFailures++
		if a.accessCodeFailures >= accessCodeMaxFailures {
			// Destroy every pending invitation so guessing cannot continue;
			// the owner issues a fresh invite from Settings.
			a.remoteInvitations = nil
		}
		return nil, false
	}
	a.accessCodeFailures = 0
	if normalizePairingHost(host) != matched.Host {
		// Mirror the QR token contract: reject without consuming so the
		// legitimate host can still pair with the same invitation.
		return nil, false
	}
	delete(a.remoteInvitations, matchedToken)
	sess, ok := a.newSessionLocked(a.effectiveExposureLocked())
	if !ok {
		return nil, false
	}
	// The code inherits the invitation's role: a pairing code minted for a
	// viewer invitation can never unlock controller rights.
	sess.Remote = true
	if matched.Role == RoleViewer {
		sess.Role = RoleViewer
	} else {
		sess.Role = RoleController
	}
	return sess, true
}

func isLoopbackHost(host string) bool {
	host = strings.TrimSpace(host)
	if host == "" || strings.EqualFold(host, "localhost") || host == nexusHostname {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func (a *AuthManager) isLoopbackListen() bool {
	return isLoopbackHost(a.listenHost)
}

func (a *AuthManager) entropyReader() io.Reader {
	if a.entropy != nil {
		return a.entropy
	}
	return rand.Reader
}

// IssueLoopbackDeviceToken returns the durable pairing token for this machine
// so a later same-origin loopback-reauth can mint a session without using the
// process bootstrap secret as an anonymous capability.
func (a *AuthManager) IssueLoopbackDeviceToken() (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.isLoopbackListen() || a.tunnelActive {
		return "", fmt.Errorf("loopback device token unavailable")
	}
	return a.ensureDeviceTokenLocked()
}

// MatchLoopbackDeviceToken reports whether the HttpOnly device cookie matches
// the pairing token persisted for this loopback install.
func (a *AuthManager) MatchLoopbackDeviceToken(value string) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return deviceTokenMatches(a.deviceToken, strings.TrimSpace(value))
}

func (a *AuthManager) ensureDeviceTokenLocked() (string, error) {
	if a.deviceToken != "" {
		return a.deviceToken, nil
	}
	a.loadDeviceTokenLocked()
	if a.deviceToken != "" {
		return a.deviceToken, nil
	}
	bytes := make([]byte, 32)
	if _, err := io.ReadFull(a.entropyReader(), bytes); err != nil {
		return "", err
	}
	a.deviceToken = hex.EncodeToString(bytes)
	if err := a.persistDeviceTokenLocked(); err != nil {
		a.deviceToken = ""
		return "", err
	}
	return a.deviceToken, nil
}

func deviceTokenMatches(stored, provided string) bool {
	if stored == "" || provided == "" {
		return false
	}
	if len(stored) != len(provided) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(stored), []byte(provided)) == 1
}
