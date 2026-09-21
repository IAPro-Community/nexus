package store

import (
	"database/sql"
	"errors"
	"testing"
	"time"
)

func seedEvent(t *testing.T, s *Store, id, fp string, at time.Time) {
	t.Helper()
	if err := s.PutCapacityEvent(CapacityEventRecord{
		ID: id, Fingerprint: fp, EventType: "QUOTA_LOW",
		Provider: "codex", Profile: "kiver", QuotaGroup: "g", WindowKind: "5h",
		Severity: "WARNING", PayloadJSON: `{"state":"WARNING"}`,
		EmittedAt: at,
	}); err != nil {
		t.Fatalf("PutCapacityEvent(%s) = %v", id, err)
	}
}

func TestPutFindCapacityEvent(t *testing.T) {
	s := openTestStore(t)
	if _, err := s.FindCapacityEvent("nope"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("FindCapacityEvent missing = %v, want sql.ErrNoRows", err)
	}
	now := time.Now().UTC().Truncate(time.Millisecond)
	seedEvent(t, s, "evt-1", "fp-1", now)
	got, err := s.FindCapacityEvent("fp-1")
	if err != nil {
		t.Fatalf("FindCapacityEvent() = %v", err)
	}
	if got.ID != "evt-1" || got.Severity != "WARNING" || got.PayloadJSON == "" {
		t.Fatalf("event mismatch: %+v", got)
	}
	if !got.EmittedAt.Equal(now) {
		t.Fatalf("emitted_at = %v, want %v", got.EmittedAt, now)
	}
	// Re-put same ID atualiza (idempotente), não duplica.
	seedEvent(t, s, "evt-1", "fp-1", now.Add(time.Minute))
	got, err = s.FindCapacityEvent("fp-1")
	if err != nil || !got.EmittedAt.Equal(now.Add(time.Minute)) {
		t.Fatalf("re-put = %+v,%v, want updated timestamp", got, err)
	}
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM capacity_events`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("rows = %d,%v, want 1 (upsert, não insert)", n, err)
	}
}

func TestFindCapacityEventReturnsLatest(t *testing.T) {
	s := openTestStore(t)
	base := time.Now().UTC().Truncate(time.Millisecond)
	seedEvent(t, s, "evt-old", "fp-x", base)
	seedEvent(t, s, "evt-new", "fp-x", base.Add(time.Hour))
	got, err := s.FindCapacityEvent("fp-x")
	if err != nil || got.ID != "evt-new" {
		t.Fatalf("latest = %+v,%v, want evt-new", got, err)
	}
}

func TestPutDeliveryReceiptPersists(t *testing.T) {
	s := openTestStore(t)
	now := time.Now().UTC().Truncate(time.Millisecond)
	if err := s.PutDeliveryReceipt(DeliveryReceipt{
		AlertID: "evt-1", Channel: "bus", Fingerprint: "fp-1",
		Status: "delivered", DeliveredAt: now,
	}); err != nil {
		t.Fatalf("PutDeliveryReceipt() = %v", err)
	}
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM capacity_delivery_receipts WHERE alert_id='evt-1'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("receipts for evt-1 = %d,%v, want 1", n, err)
	}
	// Receipt de ack do usuário usa o mesmo append-only (canal "user").
	if err := s.PutDeliveryReceipt(DeliveryReceipt{
		AlertID: "evt-1", Channel: "user", Fingerprint: "fp-1",
		Status: "acked", DeliveredAt: now,
	}); err != nil {
		t.Fatalf("ack receipt = %v", err)
	}
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM capacity_delivery_receipts WHERE alert_id='evt-1'`).Scan(&n); err != nil || n != 2 {
		t.Fatalf("receipts after ack = %d,%v, want 2", n, err)
	}
}

func TestListCapacityEventsHistory(t *testing.T) {
	s := openTestStore(t)
	base := time.Now().UTC().Truncate(time.Millisecond)
	seedEvent(t, s, "evt-1", "fp-1", base)
	seedEvent(t, s, "evt-2", "fp-2", base.Add(time.Minute))
	all, err := s.ListCapacityEvents(50)
	if err != nil || len(all) != 2 {
		t.Fatalf("history = %d,%v, want 2", len(all), err)
	}
	if all[0].ID != "evt-2" {
		t.Fatalf("history order = %q first, want evt-2 (mais recente)", all[0].ID)
	}
}
