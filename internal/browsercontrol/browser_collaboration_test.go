package browsercontrol

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestBrowserHumanInterventionWorkflow(t *testing.T) {
	s := &Session{info: Info{ID: "browser-1", ProjectID: "project-1"}}
	agent := "agent:executor-1"
	human := "human:desktop-1"
	other := "agent:executor-2"

	ok, reason := s.AcquireAgentControl(agent)
	if !ok {
		t.Fatalf("agent cannot begin: %s", reason)
	}
	if ok, _ := s.AcquireAgentControl(other); ok {
		t.Fatal("another agent stole a live lease")
	}
	state, err := s.RequestHuman(agent, "Login and select the customer account")
	if err != nil || state.Status != "requested" {
		t.Fatalf("handoff request: %+v %v", state, err)
	}
	if ok, _ := s.AcquireAgentControl(agent); ok {
		t.Fatal("agent resumed before human")
	}
	if _, err := s.SnapshotAs(context.Background(), agent); err == nil {
		t.Fatal("agent read the page during private login")
	}
	if _, err := s.ScreenshotAs(context.Background(), agent); err == nil {
		t.Fatal("agent screenshot during private login")
	}
	if _, err := s.DoAs(context.Background(), agent, Action{Type: "key", Key: "Enter"}); err == nil {
		t.Fatal("agent sent input during handoff")
	}
	if ok, _ := s.AcquireControl(human); ok {
		t.Fatal("ordinary acquire bypassed private takeover")
	}
	if _, err := s.BeginHuman(human); err != nil {
		t.Fatal(err)
	}
	if !s.CanControl(human) {
		t.Fatal("human has no control")
	}
	if _, err := s.BeginHuman("human:someone-else"); err == nil {
		t.Fatal("other human stole control")
	}
	s.mu.Lock()
	s.controllerUntil = time.Now().Add(-time.Second)
	s.mu.Unlock()
	if ok, _ := s.AcquireAgentControl(agent); ok {
		t.Fatal("expired human lease resumed agent without confirmation")
	}
	// Human must renew the takeover before finishing.
	if _, err := s.BeginHuman(human); err != nil {
		t.Fatalf("human cannot renew: %v", err)
	}
	if _, err := s.FinishHuman(human); err != nil {
		t.Fatalf("human cannot finish: %v", err)
	}
	if ok, msg := s.AcquireAgentControl(agent); !ok {
		t.Fatalf("agent cannot resume: %s", msg)
	}
	if ok, _ := s.AcquireAgentControl(other); ok {
		t.Fatal("other agent stole resumed controller")
	}
}

func TestBrowserCapabilityScopeAndRevocation(t *testing.T) {
	m := NewManager()
	first := &Session{info: Info{ID: "one", ProjectID: "p1"}}
	second := &Session{info: Info{ID: "two", ProjectID: "p2"}}
	m.sessions["one"] = first
	m.sessions["two"] = second
	grant, token, err := m.IssueAgentGrant("one", "Codex")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(token, "nb_") || len(token) != 67 {
		t.Fatal("invalid agent capability")
	}
	if grant.SessionID != "one" || !strings.HasPrefix(grant.Actor, "agent:") {
		t.Fatalf("unscoped grant: %+v", grant)
	}
	got, session, ok := m.ResolveAgentGrant(token)
	if !ok || got.ID != grant.ID || session != first {
		t.Fatal("grant resolved wrong browser")
	}
	if _, _, ok := m.ResolveAgentGrant(token + "x"); ok {
		t.Fatal("wrong token accepted")
	}
	if _, _, ok := m.ResolveAgentGrant(""); ok {
		t.Fatal("empty token accepted")
	}
	if len(m.ListAgentGrants("two")) != 0 || len(m.ListAgentGrants("one")) != 1 {
		t.Fatal("grant leaked across projects")
	}
	if m.RevokeAgentGrant("two", grant.ID) {
		t.Fatal("cross-project revoke accepted")
	}
	if !m.RevokeAgentGrant("one", grant.ID) {
		t.Fatal("revoke failed")
	}
	if _, _, ok := m.ResolveAgentGrant(token); ok {
		t.Fatal("revoked token still accepted")
	}
}

func TestBrowserHandoffReasonValidation(t *testing.T) {
	s := &Session{}
	if ok, _ := s.AcquireAgentControl("agent:a"); !ok {
		t.Fatal("control unavailable")
	}
	if _, err := s.RequestHuman("human:h", "login"); err == nil {
		t.Fatal("human can request agent handoff")
	}
	if _, err := s.RequestHuman("agent:a", ""); err == nil {
		t.Fatal("empty reason accepted")
	}
	if _, err := s.RequestHuman("agent:a", strings.Repeat("x", 501)); err == nil {
		t.Fatal("oversized reason accepted")
	}
	if _, err := s.RequestHuman("agent:a", "MFA required"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CancelHuman("human:owner"); err != nil {
		t.Fatalf("human cannot reject: %v", err)
	}
	if s.HandoffState().Status != "cancelled" {
		t.Fatal("cancel not recorded")
	}
}

func TestBrowserProactiveHumanTakeover(t *testing.T) {
	s := &Session{}
	agent := "agent:coder"
	human := "human:phone"
	if ok, _ := s.AcquireAgentControl(agent); !ok {
		t.Fatal("agent did not acquire browser")
	}
	state, err := s.BeginHuman(human)
	if err != nil || state.Status != "human_active" {
		t.Fatalf("user could not interrupt agent: %+v %v", state, err)
	}
	if !s.CanControl(human) {
		t.Fatal("human was not given control")
	}
	if _, err := s.DoAs(context.Background(), agent, Action{Type: "key", Key: "Enter"}); err == nil {
		t.Fatal("stale agent command executed after takeover")
	}
	info := s.ControlInfo()
	if info["controller"] != "human" {
		t.Fatalf("human session ID exposed in control status: %+v", info)
	}
	if ok, _ := s.AcquireAgentControl(agent); ok {
		t.Fatal("agent resumed before completion")
	}
	if _, err := s.FinishHuman(human); err != nil {
		t.Fatal(err)
	}
	if ok, reason := s.AcquireAgentControl(agent); !ok {
		t.Fatalf("agent could not resume: %s", reason)
	}
}

func TestAgentCannotObserveAnOrdinaryHumanSession(t *testing.T) {
	s := &Session{}
	if ok, _ := s.AcquireControl("human:phone"); !ok {
		t.Fatal("human could not acquire browser")
	}
	if _, err := s.SnapshotAs(context.Background(), "agent:codex"); err == nil {
		t.Fatal("agent read page while another human was in control")
	}
	if _, err := s.ScreenshotAs(context.Background(), "agent:codex"); err == nil {
		t.Fatal("agent captured human-controlled page")
	}
}
