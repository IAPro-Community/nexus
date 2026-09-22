package nexus

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func TestComposerSkillCatalogReturnsSafeMetadataOnly(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "Skill Catalog", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}

	items, err := n.ListComposerSkillCatalog(context.Background(), project.ID, "security")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) == 0 {
		t.Fatal("expected a security skill match")
	}
	found := false
	for _, item := range items {
		if item.ID == "security" {
			found = true
		}
		raw, err := json.Marshal(item)
		if err != nil {
			t.Fatal(err)
		}
		payload := strings.ToLower(string(raw))
		for _, forbidden := range []string{"instructions", "permissions", "provenance", "path"} {
			if strings.Contains(payload, forbidden) {
				t.Fatalf("browser-safe skill metadata leaked %q: %s", forbidden, raw)
			}
		}
	}
	if !found {
		t.Fatalf("security query did not return canonical builtin: %+v", items)
	}
}

func TestSelectComposerSkillUsesExistingProposalStateMachine(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "Skill Select", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	brief := newComposerBrief("add regression tests", "")
	session, err := st.CreateComposerSession(store.ComposerSession{
		ProjectID: project.ID,
		Title:     "Regression tests",
		State:     store.ComposerExploring,
		BriefJSON: mustJSON(brief),
	})
	if err != nil {
		t.Fatal(err)
	}

	view, err := n.SelectComposerSkill(context.Background(), session.ID, "testing")
	if err != nil {
		t.Fatal(err)
	}
	var selected *store.ComposerSkillProposal
	for i := range view.Skills {
		if view.Skills[i].SkillID == "testing" {
			selected = &view.Skills[i]
			break
		}
	}
	if selected == nil {
		t.Fatalf("selected skill not present in Composer view: %+v", view.Skills)
	}
	if selected.State != store.ComposerSkillAccepted || selected.Source != "builtin" || !selected.Available {
		t.Fatalf("manual selection did not use the existing proposal state machine: %+v", selected)
	}
}
