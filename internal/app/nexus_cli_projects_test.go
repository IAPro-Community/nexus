package app

import "testing"

func TestParseProjectAddArgsPreservesLegacyPositionalName(t *testing.T) {
	got, err := parseProjectAddArgs([]string{"/tmp/project", "Legacy Name"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Path != "/tmp/project" || got.Name != "Legacy Name" || got.Isolation != "" {
		t.Fatalf("unexpected parsed options: %+v", got)
	}
}

func TestParseProjectAddArgsSupportsExplicitWorktreeIsolation(t *testing.T) {
	got, err := parseProjectAddArgs([]string{
		"/tmp/project",
		"--name", "Certification",
		"--isolation", "worktree",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Path != "/tmp/project" || got.Name != "Certification" || got.Isolation != "worktree" {
		t.Fatalf("unexpected parsed options: %+v", got)
	}
}

func TestParseProjectAddArgsRejectsInvalidOrAmbiguousOptions(t *testing.T) {
	for _, args := range [][]string{
		{},
		{"/tmp/project", "--isolation"},
		{"/tmp/project", "--isolation", "container"},
		{"/tmp/project", "--unknown"},
		{"/tmp/project", "one", "two"},
		{"/tmp/project", "legacy", "--name", "duplicate"},
	} {
		if _, err := parseProjectAddArgs(args); err == nil {
			t.Fatalf("expected parse error for %#v", args)
		}
	}
}
