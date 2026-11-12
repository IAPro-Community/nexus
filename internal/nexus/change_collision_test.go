package nexus

import (
	"testing"
)

func TestParseDiffHunksExtractsNewSideRanges(t *testing.T) {
	diff := []byte(`diff --git a/main.go b/main.go
index 1111111..2222222 100644
--- a/main.go
+++ b/main.go
@@ -10,3 +12,5 @@ func main() {
 context
+added
+added2
@@ -40 +44,2 @@ func other() {
-old
+new
diff --git a/deleted.txt b/deleted.txt
deleted file mode 100644
index 3333333..0000000
--- a/deleted.txt
+++ /dev/null
@@ -1 +0,0 @@
-goodbye
`)
	hunks := ParseDiffHunks(diff)
	file, ok := hunks["main.go"]
	if !ok {
		t.Fatalf("main.go missing from parsed hunks: %v", hunks)
	}
	if len(file) != 2 {
		t.Fatalf("expected 2 hunks for main.go, got %d", len(file))
	}
	if file[0].Start != 12 || file[0].End != 16 {
		t.Fatalf("hunk 0 = %v, want start 12 end 16", file[0])
	}
	if file[1].Start != 44 || file[1].End != 45 {
		t.Fatalf("hunk 1 = %v, want start 44 end 45", file[1])
	}
	if _, ok := hunks["deleted.txt"]; ok {
		t.Fatal("deleted files must not contribute hunks")
	}
}

func TestParseDiffHunksHandlesInsertionPointAndSpaces(t *testing.T) {
	diff := []byte(`diff --git a/web/src/app with space.tsx b/web/src/app with space.tsx
--- a/web/src/app with space.tsx
+++ b/web/src/app with space.tsx
@@ -0,0 +1,3 @@
+new file
`)
	hunks := ParseDiffHunks(diff)
	file, ok := hunks["web/src/app with space.tsx"]
	if !ok {
		t.Fatalf("spaced path missing: %v", hunks)
	}
	if file[0].Start != 1 || file[0].End != 3 {
		t.Fatalf("hunk = %v, want 1..3", file[0])
	}
}

func TestDetectChangeCollisionsHunkOverlap(t *testing.T) {
	collisions := DetectChangeCollisions([]WorktreeChanges{
		{Path: "/wt/a", Hunks: map[string][]Hunk{"shared.go": {{Start: 10, End: 20}}}},
		{Path: "/wt/b", Hunks: map[string][]Hunk{"shared.go": {{Start: 18, End: 30}}}},
	})
	if len(collisions) != 1 {
		t.Fatalf("expected 1 collision, got %d", len(collisions))
	}
	c := collisions[0]
	if c.Confidence != CollisionHunkOverlap || c.Uncertain {
		t.Fatalf("overlapping hunks must be high confidence: %+v", c)
	}
	if len(c.Worktrees) != 2 || c.Worktrees[0] != "/wt/a" || c.Worktrees[1] != "/wt/b" {
		t.Fatalf("worktrees = %v, want sorted pair", c.Worktrees)
	}
}

func TestDetectChangeCollisionsSameFileDisjointIsUncertain(t *testing.T) {
	collisions := DetectChangeCollisions([]WorktreeChanges{
		{Path: "/wt/a", Hunks: map[string][]Hunk{"shared.go": {{Start: 1, End: 5}}}},
		{Path: "/wt/b", Hunks: map[string][]Hunk{"shared.go": {{Start: 100, End: 120}}}},
	})
	if len(collisions) != 1 {
		t.Fatalf("expected 1 collision, got %d", len(collisions))
	}
	c := collisions[0]
	if c.Confidence != CollisionSameFile || !c.Uncertain {
		t.Fatalf("disjoint hunks must be uncertain: %+v", c)
	}
	if c.Reason == "" {
		t.Fatal("uncertain collisions must explain themselves")
	}
}

func TestDetectChangeCollisionsDeletionRangeStillConflicts(t *testing.T) {
	// A pure insertion anchored at line 8 (zero-length new range) collides
	// with an edit touching line 8 in another worktree: the tool must
	// over-report for review rather than silently miss it.
	collisions := DetectChangeCollisions([]WorktreeChanges{
		{Path: "/wt/a", Hunks: map[string][]Hunk{"f.go": {{Start: 8, End: 8}}}},
		{Path: "/wt/b", Hunks: map[string][]Hunk{"f.go": {{Start: 8, End: 9}}}},
	})
	if len(collisions) != 1 || collisions[0].Confidence != CollisionHunkOverlap {
		t.Fatalf("adjacent insertion must overlap: %+v", collisions)
	}
}

func TestDetectChangeCollisionsIgnoresSingleWorktreeFiles(t *testing.T) {
	collisions := DetectChangeCollisions([]WorktreeChanges{
		{Path: "/wt/a", Hunks: map[string][]Hunk{"only-a.go": {{Start: 1, End: 2}}}},
		{Path: "/wt/b", Hunks: map[string][]Hunk{"only-b.go": {{Start: 1, End: 2}}}},
	})
	if len(collisions) != 0 {
		t.Fatalf("files changed by one worktree are not collisions: %+v", collisions)
	}
}

func TestDetectChangeCollisionsDeterministicOutput(t *testing.T) {
	build := func() []ChangeCollision {
		return DetectChangeCollisions([]WorktreeChanges{
			{Path: "/wt/b", Hunks: map[string][]Hunk{"z.go": {{1, 2}}, "a.go": {{1, 2}}}},
			{Path: "/wt/a", Hunks: map[string][]Hunk{"z.go": {{1, 2}}, "a.go": {{1, 2}}}},
		})
	}
	first, second := build(), build()
	if len(first) != 2 || first[0].File != "a.go" || first[1].File != "z.go" {
		t.Fatalf("collisions must sort by file: %+v", first)
	}
	for i := range first {
		if first[i].File != second[i].File || first[i].Confidence != second[i].Confidence {
			t.Fatalf("detection must be deterministic: %+v vs %+v", first[i], second[i])
		}
	}
}
