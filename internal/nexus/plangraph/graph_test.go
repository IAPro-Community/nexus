package plangraph

import (
	"strings"
	"testing"
)

func TestNormalizeDependenciesResolvesUniqueTitlesToIDs(t *testing.T) {
	nodes := []Node{
		{ID: "pkg-a", Title: "Foundation"},
		{ID: "pkg-b", Title: "API", Dependencies: []string{"Foundation"}},
	}

	got, err := NormalizeAndValidate(nodes)
	if err != nil {
		t.Fatal(err)
	}
	if len(got[1].Dependencies) != 1 || got[1].Dependencies[0] != "pkg-a" {
		t.Fatalf("expected title dependency to normalize to pkg-a, got %#v", got[1].Dependencies)
	}
}

func TestNormalizeDependenciesRejectsUnknownDependency(t *testing.T) {
	_, err := NormalizeAndValidate([]Node{{ID: "pkg-a", Title: "A", Dependencies: []string{"missing"}}})
	if err == nil || !strings.Contains(err.Error(), "unknown dependency") {
		t.Fatalf("expected unknown dependency error, got %v", err)
	}
}

func TestNormalizeDependenciesRejectsAmbiguousTitle(t *testing.T) {
	_, err := NormalizeAndValidate([]Node{
		{ID: "pkg-a", Title: "Build"},
		{ID: "pkg-b", Title: "Build"},
		{ID: "pkg-c", Title: "Review", Dependencies: []string{"Build"}},
	})
	if err == nil || !strings.Contains(err.Error(), "ambiguous dependency title") {
		t.Fatalf("expected ambiguous title error, got %v", err)
	}
}

func TestNormalizeDependenciesRejectsCycles(t *testing.T) {
	_, err := NormalizeAndValidate([]Node{
		{ID: "pkg-a", Title: "A", Dependencies: []string{"pkg-b"}},
		{ID: "pkg-b", Title: "B", Dependencies: []string{"pkg-a"}},
	})
	if err == nil || !strings.Contains(err.Error(), "dependency cycle") {
		t.Fatalf("expected dependency cycle error, got %v", err)
	}
}

func TestNormalizeDependenciesRejectsDuplicateIDs(t *testing.T) {
	_, err := NormalizeAndValidate([]Node{{ID: "pkg-a", Title: "A"}, {ID: "pkg-a", Title: "B"}})
	if err == nil || !strings.Contains(err.Error(), "duplicate package id") {
		t.Fatalf("expected duplicate id error, got %v", err)
	}
}

func TestTopologicalOrderAndExecutionWavesAreDeterministic(t *testing.T) {
	nodes := []Node{
		{ID: "d", Dependencies: []string{"b", "c"}},
		{ID: "c", Dependencies: []string{"a"}},
		{ID: "b", Dependencies: []string{"a"}},
		{ID: "a"},
	}
	order, err := TopologicalOrder(nodes)
	if err != nil {
		t.Fatal(err)
	}
	wantOrder := []string{"a", "b", "c", "d"}
	for i, want := range wantOrder {
		if order[i] != want {
			t.Fatalf("order=%v, want %v", order, wantOrder)
		}
	}
	waves, err := ExecutionWaves(nodes)
	if err != nil {
		t.Fatal(err)
	}
	wantWaves := [][]string{{"a"}, {"b", "c"}, {"d"}}
	if len(waves) != len(wantWaves) {
		t.Fatalf("waves=%v, want %v", waves, wantWaves)
	}
	for i := range wantWaves {
		if strings.Join(waves[i], ",") != strings.Join(wantWaves[i], ",") {
			t.Fatalf("waves=%v, want %v", waves, wantWaves)
		}
	}
}

func TestDependentsOfAndGraphValidation(t *testing.T) {
	nodes := []Node{{ID: "a"}, {ID: "b", Dependencies: []string{"a"}}, {ID: "c", Dependencies: []string{"a"}}}
	got, err := DependentsOf(nodes, "a")
	if err != nil || strings.Join(got, ",") != "b,c" {
		t.Fatalf("dependents=%v err=%v", got, err)
	}
	if _, err := ExecutionWaves([]Node{{ID: "a", Dependencies: []string{"missing"}}}); err == nil {
		t.Fatal("expected missing dependency rejection")
	}
}
