package nexus

import (
	"context"
	"errors"
	"testing"

	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func TestComposerMaterializationCancellationDoesNotCreateFallbackFlow(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{
		Name: "Composer failure isolation", CanonicalPath: t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	view, err := n.CreateComposerSessionWithPrompt(context.Background(), project.ID,
		"Implement a real feature", "Implement backend API, frontend and end-to-end tests")
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := n.FinalizeComposerSession(context.Background(), view.Session.ID, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	plan, err := n.MaterializePromptArtifactAsFlow(ctx, artifact.ID)
	if !errors.Is(err, context.Canceled) || plan != nil {
		t.Fatalf("canceled composition materialized a Flow: plan=%v err=%v", plan, err)
	}
	plans, err := st.ListWorkPlans(project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(plans) != 0 {
		t.Fatalf("canceled decomposition persisted %d false fallback plans", len(plans))
	}
	preserved, err := st.GetPromptArtifact(artifact.ID)
	if err != nil || preserved.Hash != artifact.Hash || preserved.Version != artifact.Version {
		t.Fatalf("failed decomposition changed immutable artifact: got=%+v err=%v", preserved, err)
	}
}
