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
		alive := IsProcessAliveWithGeneration(s.PID, s.HostGeneration)
		if !alive {
			// Check if endpoint is responsive
			client, err := protocol.NewClient(s.RuntimeID)
			if err == nil {
				_ = client.Close()
				continue
			}

			// Clean up stale socket file if present
			sockPath := protocol.EndpointPath(s.RuntimeID)
			_ = os.Remove(sockPath)

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
	}

	return cleaned, nil
}

// PurgeInactive permanently removes STALE, FAILED, and STOPPED sessions whose PID and socket are dead.
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
		alive := IsProcessAliveWithGeneration(s.PID, s.HostGeneration)
		if !alive {
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
