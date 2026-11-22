package web

import (
	"net/http"
	"strings"
)

// handleRemoteDevices lets the host owner list and revoke individual paired
// devices. Remote sessions are refused: a paired phone must never be able to
// enumerate or revoke the owner's other devices (privilege escalation).
func (s *Server) handleRemoteDevices(w http.ResponseWriter, r *http.Request) {
	sess := s.auth.AuthenticateRequest(r)
	if sess == nil || sess.Remote {
		writeError(w, http.StatusForbidden, "device management requires the host session")
		return
	}

	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Cache-Control", "no-store")
		writeJSON(w, http.StatusOK, map[string]any{
			"devices": s.auth.RemoteDevices(),
		})
	case http.MethodDelete:
		deviceID := strings.TrimSpace(r.URL.Query().Get("device_id"))
		if deviceID == "" {
			writeError(w, http.StatusBadRequest, "missing device_id")
			return
		}
		if !s.auth.RevokeDevice(deviceID) {
			writeError(w, http.StatusNotFound, "unknown device")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"revoked": deviceID})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
