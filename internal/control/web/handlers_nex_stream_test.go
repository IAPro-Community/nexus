package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/nexus"
)

func TestNexEventsAndModelWatchEndpoints(t *testing.T) {
	t.Setenv("NEXUS_DISABLE_HUD", "1")

	watch := nexus.NewModelCatalogWatchService(filepath.Join(t.TempDir(), "watch.json"))
	now := time.Now().UTC()
	watch.ObserveEntries([]nexus.ModelWatchEntry{
		{Provider: "claude", Profile: "work", ModelID: "claude-sonnet-4", IsDefault: true, Source: "provider_snapshot"},
	}, now)

	stream := nexus.NewNexEventStream(32)
	stream.Record(nexus.NexEventEnvelope{
		Fingerprint: "ev-initial-1",
		Kind:        "MODEL_CATALOG_ADDED",
		Severity:    "INFO",
		Source:      "model_catalog_watch",
		Title:       "claude · claude-sonnet-4",
		Provider:    "claude",
		CreatedAt:   now,
	})

	auth, _, err := NewAuthManager("127.0.0.1", "0")
	if err != nil {
		t.Fatal(err)
	}
	sess, err := auth.CreateSession()
	if err != nil {
		t.Fatal(err)
	}

	handler := &NexusHandler{
		auth:       auth,
		modelWatch: watch,
		nexEvents:  stream,
	}
	srv := &Server{auth: auth}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/nex/events", srv.authMiddleware(handler.handleNexEvents))
	mux.HandleFunc("/api/v1/nex/models/watch", srv.authMiddleware(handler.handleNexModelWatch))
	ts := httptest.NewServer(mux)
	defer ts.Close()

	// Unauthenticated requests must be rejected with 401.
	resUnauth, err := http.Get(ts.URL + "/api/v1/nex/events")
	if err != nil {
		t.Fatal(err)
	}
	_ = resUnauth.Body.Close()
	if resUnauth.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 unauthenticated events, got %d", resUnauth.StatusCode)
	}

	// Authenticated POST must be rejected with 405.
	postReq, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/nex/events", nil)
	postReq.Header.Set("Authorization", "Bearer "+sess.ID)
	postReq.Header.Set("X-CSRF-Token", sess.CSRFToken)
	postRes, err := ts.Client().Do(postReq)
	if err != nil {
		t.Fatal(err)
	}
	_ = postRes.Body.Close()
	if postRes.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 on POST, got %d", postRes.StatusCode)
	}

	// Authenticated GET /api/v1/nex/events returns initial event.
	getReq, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/v1/nex/events?since=0", nil)
	getReq.Header.Set("Authorization", "Bearer "+sess.ID)
	getRes, err := ts.Client().Do(getReq)
	if err != nil {
		t.Fatal(err)
	}
	var evResp nexEventsResponse
	if err := json.NewDecoder(getRes.Body).Decode(&evResp); err != nil {
		_ = getRes.Body.Close()
		t.Fatal(err)
	}
	_ = getRes.Body.Close()
	if getRes.StatusCode != http.StatusOK || evResp.Revision != 1 || len(evResp.Events) != 1 {
		t.Fatalf("unexpected events response status=%d body=%+v", getRes.StatusCode, evResp)
	}

	// Cursor progression with since=1 returns 0 duplicate events.
	req2, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/v1/nex/events?since=1", nil)
	req2.Header.Set("Authorization", "Bearer "+sess.ID)
	res2, err := ts.Client().Do(req2)
	if err != nil {
		t.Fatal(err)
	}
	var evResp2 nexEventsResponse
	if err := json.NewDecoder(res2.Body).Decode(&evResp2); err != nil {
		_ = res2.Body.Close()
		t.Fatal(err)
	}
	_ = res2.Body.Close()
	if len(evResp2.Events) != 0 || evResp2.Revision != 1 {
		t.Fatalf("expected 0 events after cursor=1, got %+v", evResp2)
	}

	// GET /api/v1/nex/models/watch returns FRESH catalog snapshot.
	mwReq, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/v1/nex/models/watch", nil)
	mwReq.Header.Set("Authorization", "Bearer "+sess.ID)
	mwRes, err := ts.Client().Do(mwReq)
	if err != nil {
		t.Fatal(err)
	}
	var mwSnap nexus.ModelCatalogWatchSnapshot
	if err := json.NewDecoder(mwRes.Body).Decode(&mwSnap); err != nil {
		_ = mwRes.Body.Close()
		t.Fatal(err)
	}
	_ = mwRes.Body.Close()
	if mwRes.StatusCode != http.StatusOK || mwSnap.Freshness != nexus.ModelWatchFreshnessFresh || len(mwSnap.Entries) != 1 {
		t.Fatalf("unexpected model watch response status=%d snap=%+v", mwRes.StatusCode, mwSnap)
	}
}
