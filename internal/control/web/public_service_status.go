package web

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sync"
	"time"
)

// Public service incidents are not capacity, quota, authentication or router
// eligibility evidence. This module never sees provider credentials.
type PublicServiceStatus struct {
	Provider  string    `json:"provider"`
	Indicator string    `json:"indicator"`
	Detail    string    `json:"detail,omitempty"`
	CheckedAt *time.Time `json:"checked_at,omitempty"`
	Stale     bool      `json:"stale"`
}

type publicStatusSource struct {
	provider string
	url      string
}

type publicStatusCache struct {
	mu      sync.Mutex
	client  *http.Client
	clock   func() time.Time
	sources []publicStatusSource
	next    time.Time
	last    []PublicServiceStatus
}

var defaultPublicStatusCache = &publicStatusCache{
	client: &http.Client{
		Timeout: 3 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	},
	clock: time.Now,
	sources: []publicStatusSource{
		{"openai", "https://status.openai.com/api/v2/status.json"},
		{"claude", "https://status.claude.com/api/v2/status.json"},
		{"github", "https://www.githubstatus.com/api/v2/status.json"},
	},
}

// snapshot serializes misses and caches both successful and failed checks.
// The explicit allowlist prevents arbitrary URL fetches/SSRF.
func (c *publicStatusCache) snapshot(ctx context.Context) []PublicServiceStatus {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := c.clock().UTC()
	if now.Before(c.next) && len(c.last) > 0 {
		return append([]PublicServiceStatus(nil), c.last...)
	}
	output := make([]PublicServiceStatus, len(c.sources))
	var wg sync.WaitGroup
	for i, source := range c.sources {
		wg.Add(1)
		go func(i int, source publicStatusSource) {
			defer wg.Done()
			result := PublicServiceStatus{Provider: source.provider, Indicator: "UNKNOWN", Stale: true}
			if i < len(c.last) && c.last[i].Provider == source.provider {
				result = c.last[i]
				result.Stale = true
			}
			if next, err := c.probe(ctx, source); err == nil {
				result = next
				result.CheckedAt = &now
				result.Stale = false
			}
			output[i] = result
		}(i, source)
	}
	wg.Wait()
	c.last = output
	c.next = now.Add(5 * time.Minute)
	return append([]PublicServiceStatus(nil), output...)
}

func (c *publicStatusCache) probe(ctx context.Context, source publicStatusSource) (PublicServiceStatus, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, source.url, nil)
	if err != nil {
		return PublicServiceStatus{}, err
	}
	req.Header.Set("Accept", "application/json")
	res, err := c.client.Do(req)
	if err != nil {
		return PublicServiceStatus{}, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return PublicServiceStatus{}, errors.New("public status unavailable")
	}
	var payload struct {
		Status struct {
			Indicator string `json:"indicator"`
			Description string `json:"description"`
		} `json:"status"`
	}
	if err := json.NewDecoder(io.LimitReader(res.Body, 64<<10)).Decode(&payload); err != nil {
		return PublicServiceStatus{}, err
	}
	switch payload.Status.Indicator {
	case "none", "minor", "major", "critical":
	default:
		return PublicServiceStatus{}, errors.New("invalid public status indicator")
	}
	detail := []rune(payload.Status.Description)
	if len(detail) > 120 {
		detail = detail[:120]
	}
	return PublicServiceStatus{
		Provider: source.provider, Indicator: payload.Status.Indicator, Detail: string(detail),
	}, nil
}

func (h *NexusHandler) handlePublicServiceStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	writeJSON(w, http.StatusOK, defaultPublicStatusCache.snapshot(r.Context()))
}
