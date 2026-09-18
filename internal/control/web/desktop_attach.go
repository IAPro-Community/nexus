package web

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

// AttachLoopbackSession reuses the authenticated loopback Web Control Center
// from a native Desktop shell. This keeps Web and Desktop on the same Core,
// runtime registry, broker and terminal hosts.
func AttachLoopbackSession(state ListenState) (*Session, error) {
	baseURL, err := validatedLoopbackURL(state.URL)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(state.BootstrapToken) == "" {
		return nil, fmt.Errorf("loopback web state has no bootstrap token")
	}
	client := &http.Client{
		Timeout: 3 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	bootstrapReq, err := http.NewRequest(http.MethodPost, baseURL+"/api/v1/auth/bootstrap",
		strings.NewReader(fmt.Sprintf(`{"token":%q}`, state.BootstrapToken)))
	if err != nil {
		return nil, fmt.Errorf("create loopback bootstrap request: %w", err)
	}
	bootstrapReq.Header.Set("Content-Type", "application/json")
	bootstrapResp, err := client.Do(bootstrapReq)
	if err != nil {
		return nil, fmt.Errorf("exchange loopback bootstrap token: %w", err)
	}
	defer bootstrapResp.Body.Close()
	if bootstrapResp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, bootstrapResp.Body)
		return nil, fmt.Errorf("loopback bootstrap exchange returned HTTP %d", bootstrapResp.StatusCode)
	}

	var sessionToken string
	for _, cookie := range bootstrapResp.Cookies() {
		if cookie.Name == sessionCookieName {
			sessionToken = strings.TrimSpace(cookie.Value)
			break
		}
	}
	if sessionToken == "" {
		_, _ = io.Copy(io.Discard, bootstrapResp.Body)
		return nil, fmt.Errorf("loopback bootstrap exchange returned no session cookie")
	}
	_, _ = io.Copy(io.Discard, bootstrapResp.Body)

	sessionReq, err := http.NewRequest(http.MethodGet, baseURL+"/api/v1/session", nil)
	if err != nil {
		return nil, fmt.Errorf("create loopback session request: %w", err)
	}
	sessionReq.Header.Set("Authorization", "Bearer "+sessionToken)
	sessionReq.Header.Set("Origin", baseURL)
	sessionResp, err := client.Do(sessionReq)
	if err != nil {
		return nil, fmt.Errorf("read loopback session: %w", err)
	}
	defer sessionResp.Body.Close()
	if sessionResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("loopback session returned HTTP %d", sessionResp.StatusCode)
	}
	var payload struct {
		Authenticated bool   `json:"authenticated"`
		CSRFToken     string `json:"csrf_token"`
	}
	if err := json.NewDecoder(sessionResp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode loopback session: %w", err)
	}
	if !payload.Authenticated || payload.CSRFToken == "" {
		return nil, fmt.Errorf("loopback session is not authenticated")
	}

	return &Session{ID: sessionToken, CSRFToken: payload.CSRFToken}, nil
}

// NewLoopbackBackendProxy forwards API and WebSocket requests from the Wails
// asset server to the already-running Web Control Center.
func NewLoopbackBackendProxy(baseURL string) (http.Handler, error) {
	return NewLoopbackBackendHandler(baseURL, nil)
}

// NewLoopbackBackendHandler keeps SPA navigation inside the native asset
// server while forwarding only Core API/WebSocket requests to the Web process.
// This prevents a Wails route transition from loading a second remote document
// and losing the Desktop WebView state.
func NewLoopbackBackendHandler(baseURL string, assets fs.FS) (http.Handler, error) {
	target, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse loopback backend URL: %w", err)
	}
	if _, err := validatedLoopbackURL(baseURL); err != nil {
		return nil, err
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	director := proxy.Director
	proxy.Director = func(req *http.Request) {
		director(req)
		req.Host = target.Host
		// Wails sends its own origin (wails://wails). The backend is loopback
		// and must validate the proxied request against its actual host.
		req.Header.Set("Origin", target.Scheme+"://"+target.Host)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if !strings.HasPrefix(req.URL.Path, "/api/") && req.Method == http.MethodGet && assets != nil {
			index, readErr := fs.ReadFile(assets, "index.html")
			if readErr == nil {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(index)
				return
			}
		}
		proxy.ServeHTTP(w, req)
	}), nil
}

func validatedLoopbackURL(raw string) (string, error) {
	target, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || target.Scheme != "http" || target.Hostname() == "" {
		return "", fmt.Errorf("invalid loopback backend URL")
	}
	if !isLoopbackHost(target.Hostname()) {
		return "", fmt.Errorf("backend URL is not loopback")
	}
	return strings.TrimRight(target.String(), "/"), nil
}
