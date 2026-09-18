package nexus

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/launcher"
	"github.com/IAPro-Community/nexus/internal/control/registry"
)

type runtimeApplicationLauncher struct {
	called bool
	opts   launcher.LaunchOptions
	sess   *registry.RuntimeSession
}

func (l *runtimeApplicationLauncher) Launch(_ context.Context, opts launcher.LaunchOptions) (*registry.RuntimeSession, error) {
	l.called = true
	l.opts = opts
	return l.sess, nil
}

func TestRuntimeApplicationServiceStartDelegatesLaunch(t *testing.T) {
	launcherStub := &runtimeApplicationLauncher{sess: &registry.RuntimeSession{RuntimeID: "runtime-1", State: registry.StateRunning}}
	service := NewRuntimeApplicationService(registry.NewRegistry(""), launcherStub, nil)

	sess, err := service.Start(context.Background(), launcher.LaunchOptions{ProviderID: "codex", ProfileID: "work"})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if !launcherStub.called || launcherStub.opts.ProviderID != "codex" || sess.RuntimeID != "runtime-1" {
		t.Fatalf("launch contract not preserved: called=%v opts=%+v session=%+v", launcherStub.called, launcherStub.opts, sess)
	}
}

func TestRuntimeApplicationServiceRejectsCanceledContextBeforeLaunch(t *testing.T) {
	launcherStub := &runtimeApplicationLauncher{}
	service := NewRuntimeApplicationService(registry.NewRegistry(""), launcherStub, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := service.Start(ctx, launcher.LaunchOptions{ProviderID: "codex"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context canceled", err)
	}
	if launcherStub.called {
		t.Fatal("canceled start must not call the launcher")
	}
}

func TestRuntimeApplicationServiceReadsAndMutatesRegistry(t *testing.T) {
	reg := registry.NewRegistry("")
	if err := reg.Register(registry.RuntimeSession{RuntimeID: "runtime-1", Title: "old", State: registry.StateRunning}); err != nil {
		t.Fatal(err)
	}
	service := NewRuntimeApplicationService(reg, &runtimeApplicationLauncher{}, nil)

	sess, err := service.Get(context.Background(), "runtime-1")
	if err != nil || sess.Title != "old" {
		t.Fatalf("get = %+v, err=%v", sess, err)
	}
	if err := service.UpdateTitle(context.Background(), "runtime-1", "new"); err != nil {
		t.Fatalf("update title: %v", err)
	}
	if err := service.Delete(context.Background(), "runtime-1"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := service.Get(context.Background(), "runtime-1"); err == nil {
		t.Fatal("deleted runtime should not be readable")
	}
}

func TestRuntimeApplicationServiceWaitForExitHonorsCanceledContext(t *testing.T) {
	reg := registry.NewRegistry("")
	if err := reg.Register(registry.RuntimeSession{RuntimeID: "runtime-cancel-wait", PID: 1, State: registry.StateRunning}); err != nil {
		t.Fatal(err)
	}
	service := NewRuntimeApplicationService(reg, &runtimeApplicationLauncher{}, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	started := time.Now()
	err := service.WaitForExit(ctx, "runtime-cancel-wait", time.Second)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context canceled", err)
	}
	if elapsed := time.Since(started); elapsed > 100*time.Millisecond {
		t.Fatalf("canceled wait took %s", elapsed)
	}
}

func TestRuntimeApplicationServiceControlRejectsCanceledContext(t *testing.T) {
	service := NewRuntimeApplicationService(registry.NewRegistry(""), &runtimeApplicationLauncher{}, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	checks := []struct {
		name string
		call func() error
	}{
		{name: "stop", call: func() error { return service.Stop(ctx, "runtime-1") }},
		{name: "mark stopped", call: func() error { return service.MarkStopped(ctx, "runtime-1") }},
		{name: "respond", call: func() error { return service.Respond(ctx, "runtime-1", "input") }},
	}
	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			if err := check.call(); !errors.Is(err, context.Canceled) {
				t.Fatalf("error = %v, want context canceled", err)
			}
		})
	}
}
