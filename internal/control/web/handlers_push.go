package web

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

// pushService returns the per-server Web Push service, creating it on first
// use. Key generation and persistence are cheap and happen once.
func (s *Server) pushService() (*pushService, error) {
	if s.push != nil {
		return s.push, nil
	}
	s.pushInit.Do(func() {
		s.push, s.pushErr = newPushService(loopbackAuthStoreDir())
	})
	return s.push, s.pushErr
}

// handlePushState reports whether a phone can subscribe and the VAPID public
// key the browser needs. The key is public by definition; no secret leaves.
func (s *Server) handlePushState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	svc, err := s.pushService()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "push service unavailable")
		return
	}
	sess := s.auth.AuthenticateRequest(r)
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, map[string]any{
		"enabled":       true,
		"public_key":    svc.PublicKey(),
		"subscriptions": svc.Count(),
		"role":          sessionRole(sess),
	})
}

type pushSubscriptionPayload struct {
	Endpoint string `json:"endpoint"`
	Keys     struct {
		P256DH string `json:"p256dh"`
		Auth   string `json:"auth"`
	} `json:"keys"`
}

// handlePushSubscribe stores one browser subscription. Both the host and a
// paired device may subscribe: the phone is usually the device that needs the
// notification when its tab is closed.
func (s *Server) handlePushSubscribe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	svc, err := s.pushService()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "push service unavailable")
		return
	}
	var payload pushSubscriptionPayload
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid subscription payload")
		return
	}
	if err := svc.Subscribe(payload.Endpoint, payload.Keys.P256DH, payload.Keys.Auth); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"subscribed":    true,
		"subscriptions": svc.Count(),
	})
}

// handlePushUnsubscribe removes the subscription matching the endpoint so a
// user can opt out from the same device that registered it.
func (s *Server) handlePushUnsubscribe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	svc, err := s.pushService()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "push service unavailable")
		return
	}
	var payload struct {
		Endpoint string `json:"endpoint"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	if strings.TrimSpace(payload.Endpoint) == "" {
		writeError(w, http.StatusBadRequest, "endpoint is required")
		return
	}
	if err := svc.Unsubscribe(payload.Endpoint); err != nil {
		writeError(w, http.StatusInternalServerError, "unable to persist subscription state")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"subscribed":    false,
		"subscriptions": svc.Count(),
	})
}

// handlePushTest delivers one real notification to every subscription. It is
// limited to the host session so a paired device cannot use it to spam others.
func (s *Server) handlePushTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	sess := s.auth.AuthenticateRequest(r)
	if sess == nil || sess.Remote {
		writeError(w, http.StatusForbidden, "push test requires the host session")
		return
	}
	svc, err := s.pushService()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "push service unavailable")
		return
	}
	payload, err := json.Marshal(map[string]any{
		"title": "Nexus",
		"body":  "Notificações remotas ativas neste dispositivo.",
		"url":   "/",
		"tag":   "nexus-push-test",
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "unable to build payload")
		return
	}
	writeJSON(w, http.StatusOK, svc.Notify(payload))
}
