package web

import (
	"context"
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/IAPro-Community/nexus/internal/nexus/runner"
)

func attentionItem(missionID string, level runner.AttentionLevel) *runner.AttentionItem {
	return &runner.AttentionItem{
		MissionID:  missionID,
		State:      runner.StateBlockedNeedsUser,
		Level:      level,
		ReasonCode: "needs_input",
		Summary:    "Resposta necessária",
		Question:   "Qual versão devo publicar?",
	}
}

func TestAttentionTrackerNotifiesOnlyNewEpisodes(t *testing.T) {
	tracker := newAttentionTracker()
	first := []attentionSnapshot{
		{missionID: "run-a", state: "WAITING_FOR_USER", reasonCode: "needs_input"},
		{missionID: "run-b", state: "WAITING_FOR_USER", reasonCode: "needs_input"},
	}
	if got := tracker.Diff(first); len(got) != 2 {
		t.Fatalf("first poll fresh = %d, want 2", len(got))
	}
	// Polling the same pending episodes again must stay silent.
	if got := tracker.Diff(first); len(got) != 0 {
		t.Fatalf("second poll fresh = %d, want 0", len(got))
	}
	// A new episode notifies while the known ones stay quiet.
	third := append(append([]attentionSnapshot{}, first...),
		attentionSnapshot{missionID: "run-c", state: "WAITING_FOR_USER", reasonCode: "needs_input"})
	fresh := tracker.Diff(third)
	if len(fresh) != 1 || fresh[0].missionID != "run-c" {
		t.Fatalf("third poll fresh = %+v, want only run-c", fresh)
	}
	// A resolved episode that returns later is announced again.
	if got := tracker.Diff(first[:1]); len(got) != 0 {
		t.Fatalf("prune poll fresh = %d, want 0", len(got))
	}
	if got := tracker.Diff(third); len(got) != 2 {
		t.Fatalf("re-raised episodes fresh = %d, want 2", len(got))
	}
}

func TestAttentionSnapshotPrefersQuestion(t *testing.T) {
	item := attentionItem("run-a", runner.AttentionRequireUser)
	snapshot := snapshotFromAttention(item)
	if snapshot.summary != "Qual versão devo publicar?" {
		t.Fatalf("summary = %q", snapshot.summary)
	}
	if snapshot.state != string(runner.StateBlockedNeedsUser) {
		t.Fatalf("state = %q", snapshot.state)
	}
	if got := snapshotFromAttention(nil); got.missionID != "" {
		t.Fatal("nil item must not panic or invent an id")
	}
}

func TestAttentionPushPayloadCarriesSummary(t *testing.T) {
	payload := attentionPushPayload([]attentionSnapshot{{summary: "Qual versão devo publicar?"}})
	if len(payload) == 0 {
		t.Fatal("payload must not be empty")
	}
	text := string(payload)
	for _, want := range []string{"nexus-attention", "Qual versão devo publicar?"} {
		if !strings.Contains(text, want) {
			t.Fatalf("payload missing %q: %s", want, text)
		}
	}
}

// TestPollAttentionDeliversOncePerEpisode wires the real push service to a
// fake endpoint and proves a repeated poll of the same episode does not wake
// the phone twice.
func TestPollAttentionDeliversOncePerEpisode(t *testing.T) {
	var delivered int32
	endpoint := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&delivered, 1)
		if got := r.Header.Get("Content-Encoding"); got != "aes128gcm" {
			t.Errorf("Content-Encoding = %q", got)
		}
		if got := r.Header.Get("Authorization"); len(got) < 12 || got[:6] != "vapid " {
			t.Errorf("missing VAPID authorization: %q", got)
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer endpoint.Close()

	clientKey, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	authSecret := make([]byte, 16)
	if _, err := rand.Read(authSecret); err != nil {
		t.Fatal(err)
	}

	svc, err := newPushService(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Subscribe(
		endpoint.URL+"/push/1",
		base64.RawURLEncoding.EncodeToString(clientKey.PublicKey().Bytes()),
		base64.RawURLEncoding.EncodeToString(authSecret),
	); err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	pending := []*runner.AttentionItem{attentionItem("run-a", runner.AttentionRequireUser)}
	srv := &Server{
		push: svc,
		attentionSource: func(context.Context) (*runner.AttentionGroup, error) {
			return &runner.AttentionGroup{NeedsYou: pending}, nil
		},
	}

	tracker := newAttentionTracker()
	if got := srv.pollAttention(context.Background(), tracker); got != 1 {
		t.Fatalf("first poll delivered = %d, want 1", got)
	}
	if got := srv.pollAttention(context.Background(), tracker); got != 0 {
		t.Fatalf("second poll delivered = %d, want 0", got)
	}
	if total := atomic.LoadInt32(&delivered); total != 1 {
		t.Fatalf("endpoint hits = %d, want 1", total)
	}

	// A new episode notifies again.
	pending = append(pending, attentionItem("run-b", runner.AttentionRequireUser))
	if got := srv.pollAttention(context.Background(), tracker); got != 1 {
		t.Fatalf("new episode delivered = %d, want 1", got)
	}
	if total := atomic.LoadInt32(&delivered); total != 2 {
		t.Fatalf("endpoint hits = %d, want 2", total)
	}
}

// A subscription that the browser dropped (410) must be pruned instead of
// retried forever.
func TestPollAttentionPrunesGoneSubscription(t *testing.T) {
	endpoint := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusGone)
	}))
	defer endpoint.Close()

	clientKey, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	authSecret := make([]byte, 16)
	if _, err := rand.Read(authSecret); err != nil {
		t.Fatal(err)
	}
	svc, err := newPushService(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Subscribe(
		endpoint.URL+"/push/gone",
		base64.RawURLEncoding.EncodeToString(clientKey.PublicKey().Bytes()),
		base64.RawURLEncoding.EncodeToString(authSecret),
	); err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	srv := &Server{
		push: svc,
		attentionSource: func(context.Context) (*runner.AttentionGroup, error) {
			return &runner.AttentionGroup{
				NeedsYou: []*runner.AttentionItem{attentionItem("run-a", runner.AttentionRequireUser)},
			}, nil
		},
	}
	tracker := newAttentionTracker()
	if got := srv.pollAttention(context.Background(), tracker); got != 0 {
		t.Fatalf("delivered = %d, want 0", got)
	}
	if svc.Count() != 0 {
		t.Fatalf("subscriptions after 410 = %d, want 0", svc.Count())
	}
}
