package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestViewerInvitationGrantsReadOnlySession(t *testing.T) {
	auth, _, err := NewAuthManager("127.0.0.1", "0")
	if err != nil {
		t.Fatalf("NewAuthManager: %v", err)
	}
	auth.SetTunnelActive(true)

	_, code, err := auth.IssueTunnelInvitationWithRole("example.trycloudflare.com", RoleViewer)
	if err != nil {
		t.Fatalf("IssueTunnelInvitationWithRole: %v", err)
	}

	sess, ok := auth.ExchangeAccessCode(code, "example.trycloudflare.com")
	if !ok || sess == nil {
		t.Fatal("viewer invitation must redeem successfully")
	}
	if sess.Role != RoleViewer {
		t.Fatalf("session role = %q, want %q", sess.Role, RoleViewer)
	}
	if !sess.Remote {
		t.Fatal("paired session must be marked remote")
	}
	if sess.CanControl() {
		t.Fatal("a viewer session must never be able to control the workspace")
	}
}

func TestControllerInvitationKeepsFullControl(t *testing.T) {
	auth, _, err := NewAuthManager("127.0.0.1", "0")
	if err != nil {
		t.Fatalf("NewAuthManager: %v", err)
	}
	auth.SetTunnelActive(true)

	_, code, err := auth.IssueTunnelInvitation("example.trycloudflare.com")
	if err != nil {
		t.Fatalf("IssueTunnelInvitation: %v", err)
	}
	sess, ok := auth.ExchangeAccessCode(code, "example.trycloudflare.com")
	if !ok || sess == nil {
		t.Fatal("controller invitation must redeem successfully")
	}
	if !sess.CanControl() {
		t.Fatal("a controller invitation must keep full control")
	}
}

func TestIssueTunnelInvitationRejectsUnknownRole(t *testing.T) {
	auth, _, err := NewAuthManager("127.0.0.1", "0")
	if err != nil {
		t.Fatalf("NewAuthManager: %v", err)
	}
	auth.SetTunnelActive(true)
	if _, _, err := auth.IssueTunnelInvitationWithRole("example.trycloudflare.com", "admin"); err == nil {
		t.Fatal("an unsupported role must be refused")
	}
}

func TestEnforceControlBlocksViewerMutation(t *testing.T) {
	viewer := &Session{Remote: true, Role: RoleViewer}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects", strings.NewReader("{}"))
	rec := httptest.NewRecorder()
	if enforceControl(viewer, rec, req) {
		t.Fatal("a viewer session must be refused on mutating requests")
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}

	readReq := httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
	if !enforceControl(viewer, httptest.NewRecorder(), readReq) {
		t.Fatal("a viewer session must be allowed to read")
	}

	controller := &Session{Remote: true, Role: RoleController}
	if !enforceControl(controller, httptest.NewRecorder(), req) {
		t.Fatal("a controller session must be allowed to mutate")
	}

	local := &Session{}
	if !enforceControl(local, httptest.NewRecorder(), req) {
		t.Fatal("the local host session must always keep control")
	}
}

// TestViewerInvitationIsRefusedOnMutation walks the real HTTP path: an owner
// issues a viewer invitation, the paired device redeems the code, and its
// mutating request is refused while reads keep working.
func TestViewerInvitationIsRefusedOnMutation(t *testing.T) {
	auth, _, err := NewAuthManager("127.0.0.1", "13000")
	if err != nil {
		t.Fatalf("NewAuthManager: %v", err)
	}
	auth.SetTunnelActive(true)
	srv := &Server{
		auth:   auth,
		tunnel: &tunnelManager{tunnel: &Tunnel{URL: "https://example.trycloudflare.com"}},
	}
	handler := srv.authMiddleware(srv.handleTunnelQR)

	owner, err := auth.CreateSession()
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	issue := httptest.NewRequest(http.MethodPost, "http://127.0.0.1/api/v1/tunnel/invite",
		strings.NewReader(`{"role":"viewer"}`))
	issue.Header.Set("Origin", "http://127.0.0.1")
	issue.AddCookie(&http.Cookie{Name: sessionCookieName, Value: owner.ID})
	issue.Header.Set(csrfHeaderName, owner.CSRFToken)
	rec := httptest.NewRecorder()
	handler(rec, issue)
	if rec.Code != http.StatusOK {
		t.Fatalf("viewer invite issuance got %d: %s", rec.Code, rec.Body.String())
	}
	var invitation struct {
		AccessCode string `json:"access_code"`
		Role       string `json:"role"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &invitation); err != nil {
		t.Fatalf("invitation payload: %v", err)
	}
	if invitation.Role != RoleViewer {
		t.Fatalf("invitation role = %q, want %q", invitation.Role, RoleViewer)
	}

	viewerSess, ok := auth.ExchangeAccessCode(invitation.AccessCode, "example.trycloudflare.com")
	if !ok || viewerSess == nil {
		t.Fatal("viewer invitation must redeem")
	}

	mutation := httptest.NewRequest(http.MethodPost, "http://127.0.0.1/api/v1/projects", strings.NewReader("{}"))
	mutation.Header.Set("Origin", "http://127.0.0.1")
	mutation.AddCookie(&http.Cookie{Name: sessionCookieName, Value: viewerSess.ID})
	mutation.Header.Set(csrfHeaderName, viewerSess.CSRFToken)
	mutationRec := httptest.NewRecorder()
	handler(mutationRec, mutation)
	if mutationRec.Code != http.StatusForbidden {
		t.Fatalf("viewer mutation got %d, want %d", mutationRec.Code, http.StatusForbidden)
	}

	read := httptest.NewRequest(http.MethodGet, "http://127.0.0.1/api/v1/projects", nil)
	read.Header.Set("Origin", "http://127.0.0.1")
	read.AddCookie(&http.Cookie{Name: sessionCookieName, Value: viewerSess.ID})
	readRec := httptest.NewRecorder()
	handler(readRec, read)
	if readRec.Code == http.StatusForbidden {
		t.Fatal("a viewer session must keep read access")
	}
}

// TestTunnelInviteRejectsUnknownRole keeps the invitation contract strict.
func TestTunnelInviteRejectsUnknownRole(t *testing.T) {
	auth, _, err := NewAuthManager("127.0.0.1", "13000")
	if err != nil {
		t.Fatalf("NewAuthManager: %v", err)
	}
	auth.SetTunnelActive(true)
	srv := &Server{
		auth:   auth,
		tunnel: &tunnelManager{tunnel: &Tunnel{URL: "https://example.trycloudflare.com"}},
	}
	owner, err := auth.CreateSession()
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1/api/v1/tunnel/invite",
		strings.NewReader(`{"role":"root"}`))
	req.Header.Set("Origin", "http://127.0.0.1")
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: owner.ID})
	req.Header.Set(csrfHeaderName, owner.CSRFToken)
	rec := httptest.NewRecorder()
	srv.authMiddleware(srv.handleTunnelQR)(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unknown role got %d, want %d", rec.Code, http.StatusBadRequest)
	}
}
