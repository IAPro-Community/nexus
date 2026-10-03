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

func TestParsePorcelainChangesPreservesIndexAndWorktreeStatus(t *testing.T) {
	changes := parsePorcelainChanges(" M web/src/App.tsx\nA  internal/new.go\n?? notes.txt\n")
	if len(changes) != 3 {
		t.Fatalf("changes=%d want=3: %+v", len(changes), changes)
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
