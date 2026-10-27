package nexus

import (
	"context"
	"testing"

	"github.com/IAPro-Community/nexus/internal/nexus/intelligence"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

// stubComposerIntelligence satisfies the product requirement that Composer
// mutations run with a configured provider while keeping the session workflow
// deterministic and offline.
type stubComposerIntelligence struct{}

func (stubComposerIntelligence) Name() string                   { return "stub" }
func (stubComposerIntelligence) Available(context.Context) bool { return true }
func (stubComposerIntelligence) AnalyzeIntent(context.Context, string, map[string]any) (*intelligence.IntentAnalysis, error) {
	return &intelligence.IntentAnalysis{}, nil
}
func (stubComposerIntelligence) EvaluateAmbiguities(context.Context, *intelligence.IntentAnalysis) ([]intelligence.AmbiguityItem, error) {
	return nil, nil
}
func (stubComposerIntelligence) GeneratePlanOutline(context.Context, *intelligence.IntentAnalysis, map[string]string, map[string]any) ([]intelligence.WorkPackageOutline, error) {
	return nil, nil
}
func (stubComposerIntelligence) ExtractBrief(context.Context, string, map[string]any) (*intelligence.BriefExtraction, error) {
	return &intelligence.BriefExtraction{}, nil
}

func TestComposerApplicationServiceSessionWorkflow(t *testing.T) {
	n := openTestNexus(t)
	n.intelligenceProviderForTest = stubComposerIntelligence{}
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "Composer App", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	service := NewComposerApplicationService(n)

	created, err := service.Create(context.Background(), project.ID, "Plan a feature", "Use the existing prompt", "EXISTING_PROMPT")
	if err != nil {
		t.Fatal(err)
	}
	if created.Session.ProjectID != project.ID {
		t.Fatalf("created session project = %q, want %q", created.Session.ProjectID, project.ID)
	}

	listed, err := service.List(context.Background(), project.ID)
	if err != nil || len(listed) != 1 {
		t.Fatalf("listed sessions = %#v, err = %v", listed, err)
	}
	fetched, err := service.Get(context.Background(), created.Session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if fetched.Session.ID != created.Session.ID {
		t.Fatalf("fetched session = %q, want %q", fetched.Session.ID, created.Session.ID)
	}
}
