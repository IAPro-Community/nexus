package nexus

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func openTestNexusStore(t *testing.T) *store.Store {
	t.Helper()
	dir := t.TempDir()
	st, err := store.Open(filepath.Join(dir, "nexus.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func TestConductorPolicyRoundTrip(t *testing.T) {
	st := openTestNexusStore(t)
	project, _ := st.CreateProject(store.Project{Name: "C", CanonicalPath: t.TempDir()})
	_ = project

	p, err := GetConductorPolicy(st)
	if err != nil {
		t.Fatal(err)
	}
	if p.Version != 1 {
		t.Fatalf("expected version 1, got %d", p.Version)
	}
	if p.RetentionDays == nil || *p.RetentionDays != 90 {
		t.Fatalf("expected retention 90, got %v", p.RetentionDays)
	}

	p2 := *p
	d := 30
	p2.RetentionDays = &d
	if err := PutConductorPolicy(st, &p2); err != nil {
		t.Fatal(err)
	}
	got, err := GetConductorPolicy(st)
	if err != nil {
		t.Fatal(err)
	}
	if *got.RetentionDays != 30 {
		t.Fatalf("expected retention 30, got %d", *got.RetentionDays)
	}
}

func TestConductorPolicyInvalidJSON(t *testing.T) {
	st := openTestNexusStore(t)
	if err := st.PutSetting("conductor:policy", "NOT-JSON"); err != nil {
		t.Fatal(err)
	}
	p, err := GetConductorPolicy(st)
	if err != nil {
		t.Fatal(err)
	}
	if p.Version != 1 {
		t.Fatalf("expected fallback version 1, got %d", p.Version)
	}
}

func TestGoalAssessmentString(t *testing.T) {
	a := GoalAssessment{Result: GoalAssessmentPass, Summary: "ok"}
	if a.String() != "PASS" {
		t.Fatalf("expected PASS, got %q", a.String())
	}
	a.Result = GoalAssessmentFail
	if a.String() != "FAIL" {
		t.Fatalf("expected FAIL, got %q", a.String())
	}
	a.Result = GoalAssessmentRedo
	if a.String() != "REDO" {
		t.Fatalf("expected REDO, got %q", a.String())
	}
}

func TestGoalContractDefaults(t *testing.T) {
	gc := GoalContract{
		GoalDescription: "test",
		LeadID:          "lead-1",
		ProjectID:       "proj-1",
		GoalID:          "goal-1",
		CreatedAt:       time.Now().UTC(),
	}
	if gc.GoalDescription != "test" {
		t.Fatal("goal description mismatch")
	}
	if gc.LeadID != "lead-1" {
		t.Fatal("lead id mismatch")
	}
}

func TestPurgeExpiredEvents(t *testing.T) {
	st := openTestNexusStore(t)
	n, err := PurgeExpiredEvents(st, 90)
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("expected 0 purged, got %d", n)
	}
}
