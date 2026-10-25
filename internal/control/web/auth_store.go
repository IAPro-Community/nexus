package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/config"
)

const (
	webAuthDirName       = "web-auth"
	sessionsFileName     = "sessions.json"
	deviceFileName       = "device.json"
	listenFileName       = "listen.json"
	listenFilePrefix     = "listen-"
	listenFileSuffix     = ".json"
	webAuthFileMode      = 0o600
	webAuthDirMode       = 0o700
	maxPersistedSessions = 8
	sessionPersistMinGap = time.Minute
)

type persistedSession struct {
	ID           string    `json:"id"`
	CSRFToken    string    `json:"csrf_token"`
	CreatedAt    time.Time `json:"created_at"`
	ExpiresAt    time.Time `json:"expires_at"`
	LastActiveAt time.Time `json:"last_active_at"`
}

type persistedAuth struct {
	Sessions []persistedSession `json:"sessions"`
}

type persistedDevice struct {
	Token string `json:"token"`
}

// ListenState is the on-disk pointer to the running Web Control Center so
// `nexus web open` can re-auth from any terminal.
type ListenState struct {
	URL            string `json:"url"`
	BootstrapURL   string `json:"bootstrap_url"`
	BootstrapToken string `json:"bootstrap_token,omitempty"`
	PID            int    `json:"pid"`
	Loopback       bool   `json:"loopback"`
}

func loopbackAuthStoreDir() string {
	dir, err := config.DataDir()
	if err != nil || dir == "" {
		return ""
	}
	return filepath.Join(dir, webAuthDirName)
}

func (a *AuthManager) persistLocked() error {
	if a.storeDir == "" || a.effectiveExposureLocked() != ExposureLoopback {
		return nil
	}
	if err := os.MkdirAll(a.storeDir, webAuthDirMode); err != nil {
		fmt.Fprintf(os.Stderr, "nexus web: failed to persist sessions: %v\n", err)
		return err
	}
	payload := persistedAuth{Sessions: make([]persistedSession, 0, len(a.sessions))}
	now := time.Now()
	for _, sess := range a.sessions {
		if sess == nil || sess.NonPersistable || now.After(sess.ExpiresAt) {
			continue
		}
		if now.Sub(sess.LastActiveAt) > sessionIdleTTL {
			continue
		}
		payload.Sessions = append(payload.Sessions, persistedSession{
			ID:           sess.ID,
			CSRFToken:    sess.CSRFToken,
			CreatedAt:    sess.CreatedAt,
			ExpiresAt:    sess.ExpiresAt,
			LastActiveAt: sess.LastActiveAt,
		})
	}
	sort.Slice(payload.Sessions, func(i, j int) bool {
		return payload.Sessions[i].CreatedAt.After(payload.Sessions[j].CreatedAt)
	})
	if len(payload.Sessions) > maxPersistedSessions {
		payload.Sessions = payload.Sessions[:maxPersistedSessions]
	}
	data, err := json.Marshal(payload)
	if err != nil {
		fmt.Fprintf(os.Stderr, "nexus web: failed to persist sessions: %v\n", err)
		return err
	}
	if err := atomicWriteFile(filepath.Join(a.storeDir, sessionsFileName), data); err != nil {
		fmt.Fprintf(os.Stderr, "nexus web: failed to persist sessions: %v\n", err)
		return err
	}
	a.lastPersist = now
	return nil
}

func (a *AuthManager) clearPersistedSessionsLocked() error {
	if a.storeDir == "" {
		return nil
	}
	path := filepath.Join(a.storeDir, sessionsFileName)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (a *AuthManager) loadPersistedLocked() {
	if a.storeDir == "" || !a.isLoopbackListen() {
		return
	}
	data, err := os.ReadFile(filepath.Join(a.storeDir, sessionsFileName))
	if err != nil || len(data) == 0 {
		return
	}
	var payload persistedAuth
	if json.Unmarshal(data, &payload) != nil {
		return
	}
	now := time.Now()
	for _, item := range payload.Sessions {
		if item.ID == "" || item.CSRFToken == "" || now.After(item.ExpiresAt) {
			continue
		}
		lastActive := item.LastActiveAt
		if lastActive.IsZero() {
			lastActive = item.CreatedAt
		}
		if now.Sub(lastActive) > sessionIdleTTL {
			continue
		}
		a.sessions[item.ID] = &Session{
			ID:           item.ID,
			CSRFToken:    item.CSRFToken,
			CreatedAt:    item.CreatedAt,
			ExpiresAt:    item.ExpiresAt,
			LastActiveAt: lastActive,
		}
	}
}

func (a *AuthManager) persistDeviceTokenLocked() error {
	if a.storeDir == "" || a.deviceToken == "" || a.effectiveExposureLocked() != ExposureLoopback {
		return nil
	}
	if err := os.MkdirAll(a.storeDir, webAuthDirMode); err != nil {
		fmt.Fprintf(os.Stderr, "nexus web: failed to persist device token: %v\n", err)
		return err
	}
	data, err := json.Marshal(persistedDevice{Token: a.deviceToken})
	if err != nil {
		fmt.Fprintf(os.Stderr, "nexus web: failed to persist device token: %v\n", err)
		return err
	}
	if err := atomicWriteFile(filepath.Join(a.storeDir, deviceFileName), data); err != nil {
		fmt.Fprintf(os.Stderr, "nexus web: failed to persist device token: %v\n", err)
		return err
	}
	return nil
}

func (a *AuthManager) loadDeviceTokenLocked() {
	if a.storeDir == "" || !a.isLoopbackListen() || a.deviceToken != "" {
		return
	}
	data, err := os.ReadFile(filepath.Join(a.storeDir, deviceFileName))
	if err != nil || len(data) == 0 {
		return
	}
	var payload persistedDevice
	if json.Unmarshal(data, &payload) != nil {
		return
	}
	token := strings.TrimSpace(payload.Token)
	if token == "" {
		return
	}
	a.deviceToken = token
}

func writeListenState(state ListenState) error {
	if !state.Loopback {
		state.BootstrapURL = ""
		state.BootstrapToken = ""
	}
	dir := loopbackAuthStoreDir()
	if dir == "" {
		return os.ErrInvalid
	}
	if err := os.MkdirAll(dir, webAuthDirMode); err != nil {
		return err
	}
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return atomicWriteFile(listenStatePath(dir, state.PID), data)
}

func listenStatePath(dir string, pid int) string {
	if pid <= 0 {
		return filepath.Join(dir, listenFileName)
	}
	return filepath.Join(dir, fmt.Sprintf("%s%d%s", listenFilePrefix, pid, listenFileSuffix))
}

func removeListenState(pid int) {
	dir := loopbackAuthStoreDir()
	if dir == "" {
		return
	}
	if pid > 0 {
		_ = os.Remove(listenStatePath(dir, pid))
	}
	// Remove the single-file format written by earlier Nexus versions when it
	// belongs to this server. Other live servers keep their own state files.
	path := filepath.Join(dir, listenFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var current ListenState
	if json.Unmarshal(data, &current) != nil {
		_ = os.Remove(path)
		return
	}
	if current.PID != 0 && current.PID != pid {
		return
	}
	_ = os.Remove(path)
}

// ReadListenState returns the listen file if a live nexus web process owns it.
func ReadListenState() (ListenState, error) {
	dir := loopbackAuthStoreDir()
	if dir == "" {
		return ListenState{}, os.ErrNotExist
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ListenState{}, err
	}
	type candidate struct {
		path    string
		modTime time.Time
		legacy  bool
	}
	candidates := make([]candidate, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		legacy := name == listenFileName
		if !legacy && (!strings.HasPrefix(name, listenFilePrefix) || !strings.HasSuffix(name, listenFileSuffix)) {
			continue
		}
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		candidates = append(candidates, candidate{
			path:    filepath.Join(dir, name),
			modTime: info.ModTime(),
			legacy:  legacy,
		})
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].modTime.Equal(candidates[j].modTime) && candidates[i].legacy != candidates[j].legacy {
			return !candidates[i].legacy
		}
		return candidates[i].modTime.After(candidates[j].modTime)
	})
	for _, candidate := range candidates {
		data, err := os.ReadFile(candidate.path)
		if err != nil {
			continue
		}
		var state ListenState
		if json.Unmarshal(data, &state) != nil || state.URL == "" || !processAlive(state.PID) {
			continue
		}
		if listenHealthOK(state.URL) {
			return state, nil
		}
	}
	return ListenState{}, os.ErrNotExist
}

func listenHealthOK(baseURL string) bool {
	client := &http.Client{Timeout: 400 * time.Millisecond}
	resp, err := client.Get(baseURL + "/api/v1/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func atomicWriteFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	ok := false
	defer func() {
		if !ok {
			_ = os.Remove(tmpName)
		}
	}()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(webAuthFileMode); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return err
	}
	ok = true
	return nil
}
