package nexus

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

type recordingProgress struct {
	mu    sync.Mutex
	steps []string
}

func (r *recordingProgress) Step(id, status string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.steps = append(r.steps, id+":"+status)
}

func TestCreateComposerSessionEmitsProgressSteps(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "Progress", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	sink := &recordingProgress{}
	prev := composerBootstrapTimeout
	composerBootstrapTimeout = 50 * time.Millisecond
	t.Cleanup(func() { composerBootstrapTimeout = prev })

	view, err := n.CreateComposerSessionWithPromptProgress(
		context.Background(),
		project.ID,
		"Ship docs",
		"Write professional documentation",
		sink,
	)
	if err != nil {
		t.Fatal(err)
	}
	if view == nil || view.Session.ID == "" {
		t.Fatal("expected session")
	}
	joined := strings.Join(sink.steps, " ")
	for _, need := range []string{"brief:started", "brief:done", "persist:started", "persist:done"} {
		if !strings.Contains(joined, need) {
			t.Fatalf("missing step %q in %v", need, sink.steps)
		}
	}
}
