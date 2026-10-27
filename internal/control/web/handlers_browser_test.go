package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/IAPro-Community/nexus/internal/browsercontrol"
)

func TestBrowserRoutesUseSessionAndCSRF(t *testing.T) {
	auth, _, err := NewAuthManager("127.0.0.1", "13000")
	if err != nil {
		t.Fatal(err)
	}
	srv := &Server{auth: auth, browsers: browsercontrol.NewManager()}
	sess, err := auth.CreateSession()
	if err != nil {
		t.Fatal(err)
	}
	handler := srv.authMiddleware(srv.handleBrowserSessions)

	newRequest := func(method string, body string, withAuth bool, csrf bool) *http.Request {
		req := httptest.NewRequest(method, "http://127.0.0.1/api/v1/browser/sessions", strings.NewReader(body))
		req.Header.Set("Origin", "http://127.0.0.1")
		if withAuth {
			req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sess.ID})
		}
		if csrf {
			req.Header.Set(csrfHeaderName, sess.CSRFToken)
		}
		return req
	}

	unauthorized := httptest.NewRecorder()
	handler(unauthorized, newRequest(http.MethodGet, "", false, false))
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous browser access accepted: %d", unauthorized.Code)
	}

	noCSRF := httptest.NewRecorder()
	handler(noCSRF, newRequest(http.MethodPost, `{"project_id":"p1"}`, true, false))
	if noCSRF.Code != http.StatusForbidden {
		t.Fatalf("browser creation without CSRF accepted: %d", noCSRF.Code)
	}

	invalid := httptest.NewRecorder()
	handler(invalid, newRequest(http.MethodPost, `{"project_id":"p1","url":"file:///etc/passwd"}`, true, true))
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("unsafe URL accepted: %d (%s)", invalid.Code, invalid.Body.String())
	}

	list := httptest.NewRecorder()
	handler(list, newRequest(http.MethodGet, "", true, false))
	if list.Code != http.StatusOK || !strings.Contains(list.Body.String(), "[]") {
		t.Fatalf("authenticated list failed: %d (%s)", list.Code, list.Body.String())
	}
}
