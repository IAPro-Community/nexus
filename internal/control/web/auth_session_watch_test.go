package web

import (
	"testing"
	"time"
)

func TestAuthManagerSessionWatcher(t *testing.T) {
	auth, _, err := NewAuthManager("127.0.0.1", "13000")
	if err != nil {
		t.Fatal(err)
	}
	session, err := auth.CreateSession()
	if err != nil {
		t.Fatal(err)
	}
	watch, cleanup, active := auth.WatchSession(session.ID)
	if !active {
		t.Fatal("fresh session was not watchable")
	}
	if !auth.TouchSession(session.ID) {
		t.Fatal("fresh session could not be touched")
	}
	auth.RevokeSession(session.ID)
	select {
	case <-watch:
	case <-time.After(time.Second):
		t.Fatal("revoking session did not notify active watchers")
	}
	cleanup()
	if auth.SessionActive(session.ID) {
		t.Fatal("revoked session remained active")
	}
}

func TestAuthManagerSessionWatcherNotifiedOnExpiry(t *testing.T) {
	auth, _, err := NewAuthManager("127.0.0.1", "13000")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	auth.clock = func() time.Time { return now }
	session, err := auth.CreateSession()
	if err != nil {
		t.Fatal(err)
	}
	watch, cleanup, active := auth.WatchSession(session.ID)
	if !active {
		t.Fatal("fresh session was not watchable")
	}
	now = session.ExpiresAt.Add(time.Second)
	if auth.SessionActive(session.ID) {
		t.Fatal("expired session remained active")
	}
	select {
	case <-watch:
	case <-time.After(time.Second):
		t.Fatal("expiring session did not notify active watchers")
	}
	cleanup()
}
