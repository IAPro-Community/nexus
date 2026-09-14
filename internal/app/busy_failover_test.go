package app

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/IAPro-Community/nexus/internal/control/launcher"
	"github.com/IAPro-Community/nexus/internal/control/registry"
	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/core/provider/adapters/codex"
	"github.com/IAPro-Community/nexus/internal/core/scheduler"
)

type stubSelector struct {
	order []string
}

func (s *stubSelector) SelectBestProfile(_ context.Context, _ string, _ string, _ []model.Profile, _ map[string]model.AccountInfo, excluded []string) (*scheduler.SelectionResult, error) {
	skip := map[string]bool{}
	for _, name := range excluded {
		skip[name] = true
	}
	for _, name := range s.order {
		if skip[name] {
			continue
		}
		return &scheduler.SelectionResult{SelectedProfile: &model.Profile{Name: name, Provider: "codex"}}, nil
	}
	return &scheduler.SelectionResult{}, errors.New("no usable codex profiles available")
}

type stubLauncher struct {
	busy     map[string]bool
	launched []string
}

func (s *stubLauncher) Launch(_ context.Context, opts launcher.LaunchOptions) (*registry.RuntimeSession, error) {
	s.launched = append(s.launched, opts.ProfileID)
	if s.busy[opts.ProfileID] {
		return nil, codex.ErrTUIBusy
	}
	return &registry.RuntimeSession{RuntimeID: "rt-" + opts.ProfileID, ProfileID: opts.ProfileID, State: registry.StateRunning}, nil
}

func TestLaunchWithBusyFailoverFallsBackWhenAutoSelecting(t *testing.T) {
	candidates := []model.Profile{{Name: "busy", Provider: "codex"}, {Name: "free", Provider: "codex"}}
	accounts := map[string]model.AccountInfo{"busy": {}, "free": {}}
	l := &stubLauncher{busy: map[string]bool{"busy": true}}
	var stderr bytes.Buffer

	sess, selected, err := launchWithBusyFailover(
		context.Background(),
		l,
		&stubSelector{order: []string{"busy", "free"}},
		"codex",
		"",
		"/tmp",
		"",
		candidates,
		accounts,
		launcher.LaunchOptions{Timeout: 1},
		&stderr,
	)
	if err != nil {
		t.Fatalf("launch: %v", err)
	}
	if selected != "free" || sess == nil || sess.ProfileID != "free" {
		t.Fatalf("selected=%s session=%+v", selected, sess)
	}
	if len(l.launched) != 2 || l.launched[0] != "busy" || l.launched[1] != "free" {
		t.Fatalf("launched=%v", l.launched)
	}
	if !bytes.Contains(stderr.Bytes(), []byte("busy")) {
		t.Fatalf("expected fallback warning, got %q", stderr.String())
	}
}

func TestLaunchWithBusyFailoverExplicitProfileDoesNotFallback(t *testing.T) {
	candidates := []model.Profile{{Name: "busy", Provider: "codex"}, {Name: "free", Provider: "codex"}}
	l := &stubLauncher{busy: map[string]bool{"busy": true}}

	_, selected, err := launchWithBusyFailover(
		context.Background(),
		l,
		&stubSelector{order: []string{"busy", "free"}},
		"codex",
		"busy",
		"/tmp",
		"",
		candidates,
		map[string]model.AccountInfo{},
		launcher.LaunchOptions{},
		&bytes.Buffer{},
	)
	if !errors.Is(err, codex.ErrTUIBusy) {
		t.Fatalf("err=%v want ErrTUIBusy", err)
	}
	if selected != "busy" {
		t.Fatalf("selected=%s", selected)
	}
	if len(l.launched) != 1 {
		t.Fatalf("launched=%v", l.launched)
	}
}

func TestLaunchWithBusyFailoverExhaustsCandidates(t *testing.T) {
	candidates := []model.Profile{{Name: "a", Provider: "codex"}, {Name: "b", Provider: "codex"}}
	l := &stubLauncher{busy: map[string]bool{"a": true, "b": true}}

	_, selected, err := launchWithBusyFailover(
		context.Background(),
		l,
		&stubSelector{order: []string{"a", "b"}},
		"codex",
		"",
		"/tmp",
		"",
		candidates,
		map[string]model.AccountInfo{},
		launcher.LaunchOptions{},
		&bytes.Buffer{},
	)
	if err == nil {
		t.Fatal("expected exhausted profiles error")
	}
	if selected != "b" && selected != "a" {
		t.Fatalf("selected=%s", selected)
	}
	if !bytes.Contains([]byte(err.Error()), []byte("no fallback profiles are available")) {
		t.Fatalf("err=%v", err)
	}
}
