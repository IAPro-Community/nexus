package web

import (
	"strings"
	"testing"
)

func TestValidateReviewPathRejectsTraversalAndAbsolutePaths(t *testing.T) {
	for _, path := range []string{"", "../secret", "../../etc/passwd", "/etc/passwd"} {
		if _, err := validateReviewPath(path); err == nil {
			t.Fatalf("path %q must be rejected", path)
		}
	}
}

func TestValidateReviewPathNormalizesSafeRelativePath(t *testing.T) {
	got, err := validateReviewPath("web/src/../src/App.tsx")
	if err != nil {
		t.Fatal(err)
	}
	if got != "web/src/App.tsx" {
		t.Fatalf("got %q", got)
	}
}

func TestParsePorcelainChangesPreservesIndexWorktreeAndRename(t *testing.T) {
	raw := " M web/src/App.tsx\x00A  internal/new.go\x00?? notes.txt\x00R  new-name.go\x00old-name.go\x00"
	changes := parsePorcelainChanges(raw)
	if len(changes) != 4 {
		t.Fatalf("changes=%d want=4: %+v", len(changes), changes)
	}
	if changes[0].Index != " " || changes[0].Worktree != "M" {
		t.Fatalf("unexpected first status: %+v", changes[0])
	}
	if changes[1].Index != "A" || changes[1].Worktree != " " {
		t.Fatalf("unexpected second status: %+v", changes[1])
	}
	if changes[2].Index != "?" || changes[2].Worktree != "?" {
		t.Fatalf("unexpected untracked status: %+v", changes[2])
	}
	if changes[3].Path != "new-name.go" || changes[3].OldPath != "old-name.go" {
		t.Fatalf("rename was not decoded from porcelain -z: %+v", changes[3])
	}
}

func TestBoundedWriterReportsTruncationWithoutShortWrite(t *testing.T) {
	writer := &boundedWriter{remaining: 4}
	input := []byte("abcdefgh")
	n, err := writer.Write(input)
	if err != nil || n != len(input) {
		t.Fatalf("write n=%d err=%v", n, err)
	}
	if writer.buf.String() != "abcd" || !writer.truncated {
		t.Fatalf("unexpected bounded output=%q truncated=%t", writer.buf.String(), writer.truncated)
	}
	if strings.Contains(writer.buf.String(), "efgh") {
		t.Fatal("bounded writer retained data beyond cap")
	}
}
