package nexus

import (
	"context"
	"testing"
)

func TestAutomaticSkillResolutionUsesKnownCatalogOnly(t *testing.T) {
	n := openTestNexus(t)
	n.maestroStatus = func() MaestroStatus { return MaestroStatus{Available: false, Mode: MaestroOff} }

	got, err := n.ResolveAutomaticSkillIDs(context.Background(), "", "estou fazendo review e quero validar as skills")
	if err != nil {
		t.Fatalf("automatic skill resolution: %v", err)
	}
	if len(got) == 0 {
		t.Fatal("expected automatic skills for review/verification prompt")
	}
	for _, id := range got {
		if _, err := validateGenericSkillIDs(n, "", []string{id}); err != nil {
			t.Fatalf("automatic resolver selected unknown skill %q: %v", id, err)
		}
	}
	if len(got) > maxAutomaticSkills {
		t.Fatalf("automatic skill set must stay bounded, got %v", got)
	}
}
