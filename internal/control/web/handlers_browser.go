package web

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/browsercontrol"
	"github.com/gorilla/websocket"
)

// Browser routes reuse the Nexus authentication, CSRF and tunnel security boundary.
// DevTools sockets and Chrome debugging credentials are never exposed to clients.
func (s *Server) handleBrowserSessions(w http.ResponseWriter, r *http.Request) {
	if s.browsers == nil {
		writeError(w, http.StatusServiceUnavailable, "browser service unavailable")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, s.browsers.List(r.URL.Query().Get("project_id")))
	case http.MethodPost:
		var req struct {
			ProjectID string `json:"project_id"`
			URL       string `json:"url"`
		}
		r.Body = http.MaxBytesReader(w, r.Body, 8192)
		d := json.NewDecoder(r.Body)
		d.DisallowUnknownFields()
		if err := d.Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid browser session request")
			return
		}
		if strings.TrimSpace(req.ProjectID) == "" {
			writeError(w, http.StatusBadRequest, "project_id required")
			return
		}
		if req.URL != "" {
			if err := browsercontrol.ValidURL(req.URL); err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
		}
		ctx, cancel := contextWithTimeout(r, 20*time.Second)
		defer cancel()
		info, err := s.browsers.Create(ctx, req.ProjectID, req.URL)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, info)
	default:
		w.Header().Set("Allow", "GET, POST")
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// contextWithTimeout provides a bounded lifecycle for Chromium calls.
func contextWithTimeout(r *http.Request, d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(r.Context(), d)
}

func (s *Server) handleBrowserSession(w http.ResponseWriter, r *http.Request) {
	if s.browsers == nil {
		writeError(w, http.StatusServiceUnavailable, "browser service unavailable")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/browser/sessions/"), "/")
	parts := strings.Split(path, "/")
	if len(parts) < 1 || len(parts) > 2 || len(parts[0]) != 32 {
		writeError(w, http.StatusNotFound, "browser session not found")
		return
	}
	session, ok := s.browsers.Get(parts[0])
	if !ok {
		writeError(w, http.StatusNotFound, "browser session not found")
		return
	}
	action := ""
	if len(parts) == 2 {
		action = parts[1]
	}
	ctx, cancel := contextWithTimeout(r, 12*time.Second)
	defer cancel()
	switch {
	case action == "stream" && r.Method == http.MethodGet:
		s.handleBrowserStream(w, r, session)
	case action == "control" && r.Method == http.MethodPost:
		actor := s.auth.AuthenticateRequest(r)
		if actor == nil { writeError(w, http.StatusUnauthorized, "authentication required"); return }
		var input struct { Mode string `json:"mode"` }
		r.Body = http.MaxBytesReader(w, r.Body, 1024)
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(w, http.StatusBadRequest, "invalid control request"); return
		}
		switch input.Mode {
		case "acquire", "renew":
			if ok, reason := session.AcquireControl(actor.ID); !ok {
				writeError(w, http.StatusConflict, reason); return
			}
			writeJSON(w, http.StatusOK, map[string]any{"controller": true, "lease_seconds": 25})
		case "release":
			session.ReleaseControl(actor.ID)
			writeJSON(w, http.StatusOK, map[string]any{"controller": false})
		default:
			writeError(w, http.StatusBadRequest, "invalid control mode")
		}
	case action == "" && r.Method == http.MethodGet:
		writeJSON(w, http.StatusOK, session.Info())
	case action == "" && r.Method == http.MethodDelete:
		s.browsers.Close(parts[0])
		writeJSON(w, http.StatusOK, map[string]string{"status": "closed"})
	case action == "frame" && r.Method == http.MethodGet:
		img, err := session.Screenshot(ctx)
		if err != nil {
			writeError(w, http.StatusBadGateway, "browser screenshot unavailable")
			return
		}
		w.Header().Set("Content-Type", "image/jpeg")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		_, _ = w.Write(img)
	case action == "snapshot" && r.Method == http.MethodGet:
		snapshot, err := session.Snapshot(ctx)
		if err != nil {
			writeError(w, http.StatusBadGateway, "browser snapshot unavailable")
			return
		}
		writeJSON(w, http.StatusOK, snapshot)
	case action == "action" && r.Method == http.MethodPost:
		r.Body = http.MaxBytesReader(w, r.Body, 16384)
		d := json.NewDecoder(r.Body)
		d.DisallowUnknownFields()
		var input browsercontrol.Action
		if err := d.Decode(&input); err != nil {
			writeError(w, http.StatusBadRequest, "invalid browser action")
			return
		}
		actor := s.auth.AuthenticateRequest(r)
		if actor == nil { writeError(w, http.StatusUnauthorized, "authentication required"); return }
		if !session.CanControl(actor.ID) {
			writeError(w, http.StatusConflict, "browser input requires an active control lease")
			return
		}
		info, err := session.Do(ctx, input)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, info)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleBrowserStream broadcasts CDP screencast frames to authenticated Nexus
// clients. Chromium's DevTools URL is never sent to the frontend.
func (s *Server) handleBrowserStream(w http.ResponseWriter, r *http.Request, session *browsercontrol.Session) {
	upgrader := websocket.Upgrader{
		ReadBufferSize: 1024,
		WriteBufferSize: 32 * 1024,
		CheckOrigin: func(req *http.Request) bool { return s.auth.ValidateOrigin(req) },
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil { return }
	defer conn.Close()
	frames, unsubscribe, err := session.SubscribeFrames()
	if err != nil {
		_ = conn.WriteJSON(map[string]any{"type": "error", "message": "browser session closed"})
		return
	}
	defer unsubscribe()
	disconnected := make(chan struct{})
	go func() {
		defer close(disconnected)
		conn.SetReadLimit(1024)
		for {
			if _, _, err := conn.ReadMessage(); err != nil { return }
		}
	}()
	// Immediate image fallback while the screencast handshake completes.
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	img, shotErr := session.Screenshot(ctx)
	cancel()
	if shotErr == nil {
		_ = conn.SetWriteDeadline(time.Now().Add(8*time.Second))
		if err := conn.WriteJSON(map[string]any{"type": "frame", "data": base64.StdEncoding.EncodeToString(img)}); err != nil { return }
	}
	keepalive := time.NewTicker(20 * time.Second)
	defer keepalive.Stop()
	for {
		select {
		case <-r.Context().Done(): return
		case <-disconnected: return
		case data, ok := <-frames:
			if !ok { return }
			_ = conn.SetWriteDeadline(time.Now().Add(8*time.Second))
			if err := conn.WriteJSON(map[string]any{"type": "frame", "data": string(data)}); err != nil { return }
		case <-keepalive.C:
			_ = conn.SetWriteDeadline(time.Now().Add(8*time.Second))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil { return }
		}
	}
}
