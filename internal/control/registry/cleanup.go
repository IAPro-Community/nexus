package registry

import (
	"os"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/protocol"
)

// CleanupStale scans registered active sessions, checks if their process/endpoint is dead,
// marks dead sessions as STALE, and removes orphaned socket files.
func (r *Registry) CleanupStale() (int, error) {
	r.mu.Lock()
	var candidates []RuntimeSession
	for _, s := range r.sessions {
		if s.IsActive() {
			candidates = append(candidates, s)
		}
	}
	r.mu.Unlock()

	cleaned := 0
	for _, s := range candidates {
		// A responsive IPC endpoint is the strongest liveness proof. On Windows
		// a provider child can survive a dead SessionHost, so child PID alone must
		// never keep a stale runtime generation alive.
		client, err := protocol.NewClient(s.RuntimeID)
		if err == nil {
			_ = client.Close()
			continue
		}

		if s.HostLive() {
			continue
		}

		_ = os.Remove(protocol.EndpointPath(s.RuntimeID))

		r.mu.Lock()
		current, ok := r.sessions[s.RuntimeID]
		if ok && current.IsActive() {
			cleaned++
		}
		_ = r.saveLocked(func(fresh map[string]RuntimeSession) {
			if cur, ok := fresh[s.RuntimeID]; ok && cur.IsActive() {
				cur.State = StateStale
				cur.UpdatedAt = time.Now()
				fresh[s.RuntimeID] = cur
			}
		})
		r.mu.Unlock()
	}

	return cleaned, nil
}

// PurgeInactive permanently removes STALE, FAILED, and STOPPED sessions once the SessionHost identity is dead. HostGeneration belongs to HostPID, never to the provider child PID.
func (r *Registry) PurgeInactive() (int, error) {
	r.mu.Lock()
	var candidates []RuntimeSession
	for _, s := range r.sessions {
		if !s.IsActive() {
			candidates = append(candidates, s)
		}
	}
	r.mu.Unlock()

	purged := 0
	for _, s := range candidates {
		if !s.HostLive() {
			// Clean up socket file
			sockPath := protocol.EndpointPath(s.RuntimeID)
			_ = os.Remove(sockPath)

			r.mu.Lock()
			_ = r.saveLocked(func(fresh map[string]RuntimeSession) {
				if _, ok := fresh[s.RuntimeID]; ok {
					delete(fresh, s.RuntimeID)
					purged++
				}
			})
			r.mu.Unlock()
		}
	}

	return purged, nil
}
