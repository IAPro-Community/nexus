package skills

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCatalogResolvesBuiltinSkillsDeterministically(t *testing.T) {
	catalog, err := NewCatalog(NewBuiltinSource())
	if err != nil {
		t.Fatal(err)
	}

	resolved, ok := catalog.Resolve("coding")
	if !ok {
		t.Fatal("expected builtin coding skill")
	}
	if resolved.Source != SourceBuiltin || resolved.Version == "" || resolved.Hash == "" {
		t.Fatalf("builtin skill lacks generic provenance: %+v", resolved)
	}
	if resolved.Instructions == "" {
		t.Fatal("builtin skill must carry bounded instructions")
	}
}

func TestCatalogPreservesSourcePrecedenceBeforeVersion(t *testing.T) {
	project := staticSource{id: "project", skills: []Skill{{ID: "shared", Name: "project", Version: "1.0.0", Source: SourceProject}}}
	user := staticSource{id: "user", skills: []Skill{{ID: "shared", Name: "user-newer", Version: "9.0.0", Source: SourceUser}}}
	catalog, err := NewCatalog(user, project)
	if err != nil {
		t.Fatal(err)
	}
	resolved, ok := catalog.Resolve("shared")
	if !ok || resolved.Version != "1.0.0" || resolved.Source != SourceProject {
		t.Fatalf("project ownership must outrank newer user copies, got %+v", resolved)
	}
	if resolved.Copies != 2 {
		t.Fatalf("expected two discovered copies, got %d", resolved.Copies)
	}
}

func TestCatalogUsesHighestVersionWithinSameSourceTier(t *testing.T) {
	older := staticSource{id: "project-a", skills: []Skill{{ID: "shared", Name: "old", Version: "1.0.0", Source: SourceProject}}}
	newer := staticSource{id: "project-b", skills: []Skill{{ID: "shared", Name: "new", Version: "2.0.0", Source: SourceProject}}}
	catalog, err := NewCatalog(older, newer)
	if err != nil {
		t.Fatal(err)
	}
	resolved, ok := catalog.Resolve("shared")
	if !ok || resolved.Version != "2.0.0" || resolved.Source != SourceProject {
		t.Fatalf("highest version should win inside one source tier, got %+v", resolved)
	}
}

func TestDirectorySourceReadsBoundedSkillMarkdown(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, "demo")
	if err := os.MkdirAll(skillDir, 0o700); err != nil {
		t.Fatal(err)
	}
	content := "---\nid: demo\nname: Demo Skill\nversion: 1.2.0\ncapabilities: [testing, review]\ntriggers: [test]\nrisk: low\n---\nUse the existing test harness.\n"
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	source := NewDirectorySource("project", SourceProject, []string{root})
	items, err := source.Discover(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ID != "demo" || items[0].Source != SourceProject {
		t.Fatalf("unexpected directory skills: %+v", items)
	}
	if items[0].Hash == "" || items[0].Instructions == "" || len(items[0].Capabilities) != 2 {
		t.Fatalf("skill metadata/provenance not parsed: %+v", items[0])
	}
}

func TestDirectorySourceNeverExecutesSkillScriptsDuringDiscovery(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, "safe-skill")
	scriptsDir := filepath.Join(skillDir, "scripts")
	if err := os.MkdirAll(scriptsDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nid: safe-skill\n---\nInspect only.\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(root, "executed")
	script := "#!/bin/sh\nprintf executed > " + marker + "\n"
	if err := os.WriteFile(filepath.Join(scriptsDir, "danger.sh"), []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}

	source := NewDirectorySource("project", SourceProject, []string{root})
	items, err := source.Discover(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ID != "safe-skill" {
		t.Fatalf("unexpected discovery result: %+v", items)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("skill discovery executed a colocated script: %v", err)
	}
}

func TestCatalogSameVersionPrefersProjectOverMaestro(t *testing.T) {
	project := staticSource{id: "project", skills: []Skill{{ID: "shared", Name: "project", Version: "1.0.0", Source: SourceProject}}}
	maestro := staticSource{id: "maestro", skills: []Skill{{ID: "shared", Name: "maestro", Version: "1.0.0", Source: SourceMaestro}}}
	catalog, err := NewCatalog(maestro, project)
	if err != nil {
		t.Fatal(err)
	}
	resolved, ok := catalog.Resolve("shared")
	if !ok || resolved.Source != SourceProject {
		t.Fatalf("project skill must outrank maestro at equal version, got %+v", resolved)
	}
}

func TestNormalizeSkillHashIsStableAcrossObservationTime(t *testing.T) {
	first, err := normalizeSkill(Skill{ID: "demo", Instructions: "do the thing", Source: SourceBuiltin}, string(SourceBuiltin))
	if err != nil {
		t.Fatal(err)
	}
	second, err := normalizeSkill(Skill{ID: "demo", Instructions: "do the thing", Source: SourceBuiltin, Provenance: Provenance{ObservedAt: first.Provenance.ObservedAt.Add(time.Hour)}}, string(SourceBuiltin))
	if err != nil {
		t.Fatal(err)
	}
	if first.Hash == "" || first.Hash != second.Hash {
		t.Fatalf("skill hash must ignore ObservedAt: %q vs %q", first.Hash, second.Hash)
	}
}

func TestCatalogWorksWhenOptionalSourcesAreUnavailable(t *testing.T) {
	source := NewExternalSource("external", func(context.Context) ([]Skill, error) {
		return nil, ErrSourceUnavailable
	})
	catalog, err := NewCatalog(NewBuiltinSource(), source)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := catalog.Resolve("testing"); !ok {
		t.Fatal("builtin catalog must remain usable when optional source is unavailable")
	}
}

func TestDirectorySourceSkipsCyclicOptionalRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "skills-loop")
	if err := os.Symlink(root, root); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	source := NewDirectorySource("user", SourceUser, []string{root})
	items, err := source.Discover(context.Background())
	if err != nil {
		t.Fatalf("cyclic optional source must degrade without failing catalog: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("cyclic root should not yield skills: %+v", items)
	}
}

type staticSource struct {
	id     string
	skills []Skill
}

func (s staticSource) ID() string { return s.id }

func (s staticSource) Available(context.Context) bool { return true }

func (s staticSource) Discover(context.Context) ([]Skill, error) { return s.skills, nil }

func (s staticSource) Resolve(_ context.Context, id string) (Skill, bool, error) {
	for _, item := range s.skills {
		if item.ID == id {
			return item, true, nil
		}
	}
	return Skill{}, false, nil
}
