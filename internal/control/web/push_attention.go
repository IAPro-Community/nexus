package web

import (
	"context"
	"encoding/json"
	"sort"
	"sync"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/notify"
	"github.com/IAPro-Community/nexus/internal/nexus/runner"
)

const (
	pushAttentionInterval = 30 * time.Second
	pushAttentionMaxSeen  = 256
)

// attentionSnapshot is the minimal projection the watcher needs. Keeping the
// dedupe key separate from the runner types makes the diff logic testable
// without a repository.
type attentionSnapshot struct {
	missionID  string
	state      string
	reasonCode string
	summary    string
}

func attentionKey(item attentionSnapshot) string {
	return item.missionID + "|" + item.state + "|" + item.reasonCode
}

// attentionTracker remembers which pending items were already announced. A
// brand-new episode notifies; a re-poll of the same episode stays silent, so a
// phone is never woken every 30 seconds for one pending question.
type attentionTracker struct {
	mu   sync.Mutex
	seen map[string]struct{}
}

func newAttentionTracker() *attentionTracker {
	return &attentionTracker{seen: map[string]struct{}{}}
}

// Diff records the current pending set and returns the items that were not
// announced before. Items that left the pending set are forgotten, so the same
// question asked again later notifies again.
func (t *attentionTracker) Diff(items []attentionSnapshot) []attentionSnapshot {
	t.mu.Lock()
	defer t.mu.Unlock()
	current := make(map[string]struct{}, len(items))
	fresh := make([]attentionSnapshot, 0, 1)
	for _, item := range items {
		key := attentionKey(item)
		current[key] = struct{}{}
		if _, announced := t.seen[key]; !announced {
			fresh = append(fresh, item)
		}
	}
	if len(current) > pushAttentionMaxSeen {
		current = map[string]struct{}{}
		for _, item := range items[:pushAttentionMaxSeen] {
			current[attentionKey(item)] = struct{}{}
		}
	}
	t.seen = current
	sort.Slice(fresh, func(i, j int) bool { return fresh[i].missionID < fresh[j].missionID })
	return fresh
}

func snapshotFromAttention(item *runner.AttentionItem) attentionSnapshot {
	if item == nil {
		return attentionSnapshot{}
	}
	return attentionSnapshot{
		missionID:  item.MissionID,
		state:      string(item.State),
		reasonCode: item.ReasonCode,
		summary:    firstNonEmpty(item.Question, item.Summary, item.Impact),
	}
}

// attentionPushPayload builds the notification body for the freshly pending
// items. It stays short: a lock screen truncates aggressively.
func attentionPushPayload(fresh []attentionSnapshot) []byte {
	body := "Uma missão precisa de você."
	if len(fresh) > 0 && fresh[0].summary != "" {
		body = fresh[0].summary
	}
	payload, err := json.Marshal(map[string]any{
		"title": "Nexus — ação necessária",
		"body":  body,
		"url":   "/",
		"tag":   "nexus-attention",
	})
	if err != nil {
		return nil
	}
	return payload
}

// runAttentionPush polls the attention projection and pushes only new pending
// items. It exits when ctx is canceled (server shutdown).
func (s *Server) runAttentionPush(ctx context.Context) {
	if s.attentionSource == nil {
		return
	}
	if _, err := s.pushService(); err != nil {
		return
	}
	tracker := newAttentionTracker()
	ticker := time.NewTicker(pushAttentionInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		s.pollAttention(ctx, tracker)
	}
}

// pollAttention performs one observation cycle and returns how many
// notifications were delivered. It is separate from the loop so the diff and
// delivery can be tested without waiting for a tick.
func (s *Server) pollAttention(ctx context.Context, tracker *attentionTracker) int {
	svc, err := s.pushService()
	if err != nil {
		return 0
	}
	// Nothing to notify: skip the projection query entirely.
	if svc.Count() == 0 {
		return 0
	}
	source := s.attentionSource
	if source == nil {
		return 0
	}
	group, err := source(ctx)
	if err != nil || group == nil {
		return 0
	}
	pending := make([]attentionSnapshot, 0, len(group.NeedsYou))
	for _, item := range group.NeedsYou {
		pending = append(pending, snapshotFromAttention(item))
	}
	fresh := tracker.Diff(pending)
	if len(fresh) == 0 {
		return 0
	}
	if notify.ShouldSuppressFallback(notify.CategoryActionRequired) {
		return 0
	}
	payload := attentionPushPayload(fresh)
	if payload == nil {
		return 0
	}
	return svc.Notify(payload).Delivered
}

func (s *Server) startAttentionPush() {
	if s.attentionSource == nil {
		return
	}
	s.pushWatchOnce.Do(func() {
		ctx, cancel := context.WithCancel(context.Background())
		s.pushWatchMu.Lock()
		s.pushWatchCancel = cancel
		s.pushWatchMu.Unlock()
		go s.runAttentionPush(ctx)
	})
}
