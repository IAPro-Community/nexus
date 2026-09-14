package nexus

import (
	"strings"
	"testing"

	"github.com/IAPro-Community/nexus/internal/nexus/intelligence"
)

func TestPackagesFromOutlinesDropsSchemaPlaceholders(t *testing.T) {
	pkgs := packagesFromOutlines("quero saber como vou testar", []intelligence.WorkPackageOutline{{
		Title: "...", Goal: "...", Priority: "HIGH",
		Dependencies: []string{"package title"}, Role: "implementer",
		Acceptance: []string{"measurable criterion"},
	}})
	if len(pkgs) != 1 {
		t.Fatalf("got %d packages", len(pkgs))
	}
	if pkgs[0].Title == "..." || strings.TrimSpace(pkgs[0].Title) == "" {
		t.Fatalf("placeholder title leaked: %q", pkgs[0].Title)
	}
	if pkgs[0].Goal == "..." || strings.TrimSpace(pkgs[0].Goal) == "" {
		t.Fatalf("placeholder goal leaked: %q", pkgs[0].Goal)
	}
	if len(pkgs[0].Dependencies) != 0 {
		t.Fatalf("placeholder dependency leaked: %#v", pkgs[0].Dependencies)
	}
}

func TestPackagesFromOutlinesResolvesTitleDependenciesToIDs(t *testing.T) {
	pkgs := packagesFromOutlines("ship auth", []intelligence.WorkPackageOutline{
		{Title: "Foundation", Goal: "schema", Priority: "HIGH", Role: "architect"},
		{Title: "API", Goal: "http", Priority: "HIGH", Dependencies: []string{"Foundation", "package title"}, Role: "implementer"},
	})
	if len(pkgs) != 2 {
		t.Fatalf("got %d packages", len(pkgs))
	}
	if len(pkgs[1].Dependencies) != 1 || pkgs[1].Dependencies[0] != pkgs[0].ID {
		t.Fatalf("expected API to depend on Foundation id %q, got %#v", pkgs[0].ID, pkgs[1].Dependencies)
	}
}

func TestPackagesFromOutlinesSynthesizesWhenProviderCopiesSchema(t *testing.T) {
	pkgs := packagesFromOutlines("documentar testes do proxy", nil)
	if len(pkgs) != 1 {
		t.Fatalf("got %d packages", len(pkgs))
	}
	if !strings.Contains(pkgs[0].Goal, "documentar testes do proxy") {
		t.Fatalf("expected goal from user request, got %q", pkgs[0].Goal)
	}
}
