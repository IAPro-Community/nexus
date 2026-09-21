package nexus

import (
	"testing"
)

// Composer reuses MemoryFactsMap/ApplyMemoriesToBrief/LivingBrief: a valid
// durable preference must not be re-asked, and the current request wins a
// conflict with memory.
func TestComposerMemorySkipsCoveredQuestion(t *testing.T) {
	brief := newComposerBrief("melhorar o dashboard", "")
	ApplyMemoriesToBrief(&brief, map[string]string{
		"quality.verification": "make quality",
	})
	refreshComposerBrief(&brief)
	MarkMemoryCoveredUnknowns(&brief, map[string]string{
		"quality.verification": "make quality",
	})
	for _, u := range brief.Unknowns {
		if u.Field == "quality.verification" && u.Status == PromptUnknownOpen {
			t.Fatalf("memory-covered verification must not be re-asked: %+v", u)
		}
	}
	found := false
	for _, u := range brief.Unknowns {
		if u.Field == "quality.verification" {
			found = true
			if u.Source != "MEMORY" {
				t.Fatalf("memory-covered unknown must carry MEMORY provenance, got %+v", u)
			}
			if u.Status != PromptUnknownInferred {
				t.Fatalf("memory-covered unknown must be INFERRED, got %+v", u)
			}
		}
	}
	if !found {
		t.Fatalf("expected verification unknown to exist (as inferred), got %+v", brief.Unknowns)
	}
}

func TestComposerCurrentRequestWinsMemoryConflict(t *testing.T) {
	// Current request explicitly mentions npm test; memory says "make quality".
	// The brief must keep the current value and must not be overwritten.
	brief := newComposerBrief("corrigir bug, use npm test para verificar a correção", "")
	if len(brief.Quality.Verification) == 0 {
		t.Fatalf("expected current request to seed verification, got %+v", brief.Quality)
	}
	ApplyMemoriesToBrief(&brief, map[string]string{
		"quality.verification": "make quality",
	})
	refreshComposerBrief(&brief)
	MarkMemoryCoveredUnknowns(&brief, map[string]string{
		"quality.verification": "make quality",
	})
	for _, v := range brief.Quality.Verification {
		if v == "make quality" {
			t.Fatalf("conflicting memory must not append when current already covers the field: %+v", brief.Quality.Verification)
		}
	}
	// The unknown must reflect the USER answer, not MEMORY.
	for _, u := range brief.Unknowns {
		if u.Field == "quality.verification" && u.Source == "MEMORY" {
			t.Fatalf("conflict must not be retagged MEMORY when current wins: %+v", u)
		}
	}
}
