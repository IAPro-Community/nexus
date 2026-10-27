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
	order    []string
	contexts []context.Context
	err      error
}

func (s *stubSelector) SelectBestProfile(ctx context.Context, _ string, _ string, _ []model.Profile, _ map[string]model.AccountInfo, excluded []string) (*scheduler.SelectionResult, error) {
	s.contexts = append(s.contexts, ctx)
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
	if s.err != nil {
		return &scheduler.SelectionResult{}, s.err
	}
	return &scheduler.SelectionResult{}, errors.New("no usable codex profiles available")
}

func TestResolveLaunchProfilePropagatesContextToSelector(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	selector := &stubSelector{order: []string{"free"}}

	selected, err := resolveLaunchProfile(
		ctx,
		"codex",
		"",
		"/tmp",
		"",
		[]model.Profile{{Name: "free", Provider: "codex"}},
		map[string]model.AccountInfo{},
		selector,
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("resolve profile: %v", err)
	}
	if selected != "free" {
		t.Fatalf("selected=%q", selected)
	}
	if len(selector.contexts) != 1 || selector.contexts[0] != ctx {
		t.Fatalf("selector contexts=%v want canceled caller context", selector.contexts)
	}
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

func TestLaunchWithBusyFailoverDoesNotUseDefaultWhenAutoSelectionFails(t *testing.T) {
	candidates := []model.Profile{
		{Name: "exhausted-default", Provider: "agy"},
		{Name: "also-unavailable", Provider: "agy"},
	}
	l := &stubLauncher{}

	session, selected, err := launchWithBusyFailover(
		context.Background(),
		l,
		&stubSelector{err: errors.New("no usable agy profiles available")},
		"agy",
		"",
		"/tmp",
		"exhausted-default",
		candidates,
		map[string]model.AccountInfo{},
		launcher.LaunchOptions{},
		&bytes.Buffer{},
	)
	if err == nil {
		t.Fatal("expected automatic selection error")
	}
	if session != nil || selected != "" {
		t.Fatalf("session=%+v selected=%q; failed auto-selection must not choose a default", session, selected)
	}
	if len(l.launched) != 0 {
		t.Fatalf("launcher called for %v; exhausted default must not launch", l.launched)
	}
	if !bytes.Contains([]byte(err.Error()), []byte("no usable agy profiles available")) {
		t.Fatalf("selection error not propagated: %v", err)
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

type retrySelector struct {
	calls int
	err   error
	order []string
}

func (s *retrySelector) SelectBestProfile(_ context.Context, _ string, _ string, _ []model.Profile, accounts map[string]model.AccountInfo, excluded []string) (*scheduler.SelectionResult, error) {
	s.calls++
	if s.calls == 1 {
		return &scheduler.SelectionResult{}, s.err
	}
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

func TestLaunchWithBusyFailoverRescuesStaleQuotaWithRefreshRetry(t *testing.T) {
	candidates := []model.Profile{{Name: "kiver", Provider: "codex"}}
	accounts := map[string]model.AccountInfo{
		"kiver": {
			Authenticated: true,
			Health:        model.HealthHealthy,
			Email:         "kiver@test.com",
		},
	}
	l := &stubLauncher{}
	var stderr bytes.Buffer
	sel := &retrySelector{
		err:   errors.New("no usable codex profiles available: kiver (Quota desatualizada; atualize antes de selecionar automaticamente)"),
		order: []string{"kiver"},
	}

	session, selected, err := launchWithBusyFailover(
		context.Background(),
		l,
		sel,
		"codex",
		"",
		"/tmp",
		"",
		candidates,
		accounts,
		launcher.LaunchOptions{},
		&stderr,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if selected != "kiver" || session == nil {
		t.Fatalf("expected kiver to be selected after refresh retry; got selected=%s sess=%+v", selected, session)
	}
	if sel.calls < 2 {
		t.Fatalf("expected selector to be called again after refresh, calls=%d", sel.calls)
	}
}

func TestLaunchWithBusyFailoverFallsBackToHealthyProfileWhenStaleAndCannotRefresh(t *testing.T) {
	candidates := []model.Profile{{Name: "kiver", Provider: "codex"}}
	accounts := map[string]model.AccountInfo{
		"kiver": {
			Authenticated: true,
			Health:        model.HealthHealthy,
			Email:         "kiver@test.com",
		},
	}
	l := &stubLauncher{}
	var stderr bytes.Buffer
	// Selector always fails with stale quota message
	sel := &stubSelector{
		err: errors.New("no usable codex profiles available: kiver (Quota desatualizada; atualize antes de selecionar automaticamente)"),
	}

	session, selected, err := launchWithBusyFailover(
		context.Background(),
		l,
		sel,
		"codex",
		"",
		"/tmp",
		"kiver",
		candidates,
		accounts,
		launcher.LaunchOptions{},
		&stderr,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if selected != "kiver" || session == nil {
		t.Fatalf("expected fallback to kiver, got selected=%s", selected)
	}
	if !bytes.Contains(stderr.Bytes(), []byte("fallback resiliente")) {
		t.Fatalf("expected resilient fallback warning, got %q", stderr.String())
	}
}

func TestLaunchWithBusyFailoverDoesNotFallbackWhenAccountIsRateLimited(t *testing.T) {
	candidates := []model.Profile{{Name: "limited", Provider: "codex"}}
	accounts := map[string]model.AccountInfo{
		"limited": {
			Authenticated: true,
			Health:        model.HealthHealthy,
			Usage: model.UsageSnapshot{
				Status: model.UsageRateLimited,
			},
		},
	}
	l := &stubLauncher{}
	var stderr bytes.Buffer
	sel := &stubSelector{
		err: errors.New("no usable codex profiles available: limited (Conta rate-limited)"),
	}

	session, selected, err := launchWithBusyFailover(
		context.Background(),
		l,
		sel,
		"codex",
		"",
		"/tmp",
		"limited",
		candidates,
		accounts,
		launcher.LaunchOptions{},
		&stderr,
	)
	if err == nil {
		t.Fatal("expected error for rate limited profile")
	}
	if session != nil || selected != "" {
		t.Fatalf("rate limited account must not be launched: session=%+v, selected=%s", session, selected)
	}
}
