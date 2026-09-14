package nexus

import (
	"context"
	"fmt"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/driver"
	"github.com/IAPro-Community/nexus/internal/control/handoff"
	"github.com/IAPro-Community/nexus/internal/control/launcher"
	"github.com/IAPro-Community/nexus/internal/control/protocol"
	"github.com/IAPro-Community/nexus/internal/control/registry"
	"github.com/IAPro-Community/nexus/internal/core/model"
)

// RuntimeLaunchPort is the infrastructure boundary used by the runtime
// application service. Keeping it narrower than launcher.Launcher allows the
// application contract to be tested without spawning a provider process.
type RuntimeLaunchPort interface {
	Launch(context.Context, launcher.LaunchOptions) (*registry.RuntimeSession, error)
}

// RuntimeApplicationService owns transport-facing runtime record operations.
// Provider process details remain in launcher/registry; this service composes
// them for CLI, Web and Desktop consumers without duplicating lifecycle rules.
type RuntimeApplicationService struct {
	registry *registry.Registry
	launcher RuntimeLaunchPort
	drivers  *driver.Registry
}

func NewRuntimeApplicationService(reg *registry.Registry, launch RuntimeLaunchPort, drivers *driver.Registry) *RuntimeApplicationService {
	if reg == nil {
		reg = registry.DefaultRegistry()
	}
	return &RuntimeApplicationService{registry: reg, launcher: launch, drivers: drivers}
}

func (s *RuntimeApplicationService) ready(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.registry == nil {
		return fmt.Errorf("runtime service is unavailable")
	}
	return nil
}

// List returns the current runtime records after removing stale entries.
func (s *RuntimeApplicationService) List(ctx context.Context) ([]registry.RuntimeSession, error) {
	if err := s.ready(ctx); err != nil {
		return nil, err
	}
	_, _ = s.registry.CleanupStale()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.registry.List(), nil
}

// Cleanup removes stale and inactive runtime records without exposing
// registry maintenance details to a transport adapter.
func (s *RuntimeApplicationService) Cleanup(ctx context.Context) (cleaned, purged int, err error) {
	if err := s.ready(ctx); err != nil {
		return 0, 0, err
	}
	cleaned, _ = s.registry.CleanupStale()
	if err := ctx.Err(); err != nil {
		return cleaned, 0, err
	}
	cleaned += launcher.ReapOrphanHosts(s.registry.List())
	if err := ctx.Err(); err != nil {
		return cleaned, 0, err
	}
	purged, err = s.registry.PurgeInactive()
	return cleaned, purged, err
}

// WaitForExit observes process liveness after a control-plane stop request.
// The bounded wait belongs to the runtime application contract so HTTP and
// future CLI/Desktop consumers share cancellation behavior.
func (s *RuntimeApplicationService) WaitForExit(ctx context.Context, runtimeID string, timeout time.Duration) error {
	if err := s.ready(ctx); err != nil {
		return err
	}
	if timeout <= 0 {
		return nil
	}
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	tick := time.NewTicker(25 * time.Millisecond)
	defer tick.Stop()
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		sess, ok := s.registry.Get(runtimeID)
		if !ok || sess.PID <= 0 || !registry.IsProcessAlive(sess.PID) {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			return nil
		case <-tick.C:
		}
	}
}

// Start delegates process creation to the supervised launcher.
func (s *RuntimeApplicationService) Start(ctx context.Context, opts launcher.LaunchOptions) (*registry.RuntimeSession, error) {
	if err := s.ready(ctx); err != nil {
		return nil, err
	}
	if s.launcher == nil {
		return nil, fmt.Errorf("runtime launcher is unavailable")
	}
	return s.launcher.Launch(ctx, opts)
}

// Get reads one runtime record from the live registry.
func (s *RuntimeApplicationService) Get(ctx context.Context, runtimeID string) (registry.RuntimeSession, error) {
	if err := s.ready(ctx); err != nil {
		return registry.RuntimeSession{}, err
	}
	sess, ok := s.registry.Get(runtimeID)
	if !ok {
		return registry.RuntimeSession{}, fmt.Errorf("runtime %q not found", runtimeID)
	}
	return sess, nil
}

// Detail combines a runtime record with truthful provider capabilities when
// the driver registry is available. A missing driver is not an application
// error because legacy runtimes may outlive an installed provider.
func (s *RuntimeApplicationService) Detail(ctx context.Context, runtimeID string) (registry.RuntimeSession, *driver.EffectiveCapabilities, error) {
	sess, err := s.Get(ctx, runtimeID)
	if err != nil {
		return registry.RuntimeSession{}, nil, err
	}
	if s.drivers == nil {
		return sess, nil, nil
	}
	d, err := s.drivers.Get(sess.ProviderID)
	if err != nil || d == nil {
		return sess, nil, nil
	}
	caps := d.EffectiveCaps(ctx, modelProfile(sess))
	return sess, &caps, nil
}

func modelProfile(sess registry.RuntimeSession) model.Profile {
	return model.Profile{Name: sess.ProfileID, Provider: sess.ProviderID}
}

func (s *RuntimeApplicationService) UpdateTitle(ctx context.Context, runtimeID, title string) error {
	if err := s.ready(ctx); err != nil {
		return err
	}
	return s.registry.UpdateTitle(runtimeID, title)
}

// Stop requests a graceful stop over the existing runtime protocol. Callers
// may deliberately treat a missing IPC endpoint as best-effort cleanup, as the
// legacy HTTP contract did.
func (s *RuntimeApplicationService) Stop(ctx context.Context, runtimeID string) error {
	if err := s.ready(ctx); err != nil {
		return err
	}
	client, err := protocol.NewClient(runtimeID)
	if err != nil {
		return err
	}
	stopErr := client.StopContext(ctx)
	closeErr := client.Close()
	if stopErr != nil {
		return stopErr
	}
	return closeErr
}

func (s *RuntimeApplicationService) MarkStopped(ctx context.Context, runtimeID string) error {
	if err := s.ready(ctx); err != nil {
		return err
	}
	return s.registry.TransitionState(runtimeID, registry.StateStopped)
}

// Respond sends interactive input through the existing runtime protocol.
func (s *RuntimeApplicationService) Respond(ctx context.Context, runtimeID, input string) error {
	if err := s.ready(ctx); err != nil {
		return err
	}
	client, err := protocol.NewClient(runtimeID)
	if err != nil {
		return err
	}
	defer client.Close()
	if input == "" || input[len(input)-1] != '\n' {
		input += "\n"
	}
	_, err = client.SendContext(ctx, protocol.CmdInput, protocol.InputPayload{Data: input})
	return err
}

func (s *RuntimeApplicationService) AccountHandoff(ctx context.Context, runtimeID, target string) (*registry.RuntimeSession, error) {
	if err := s.ready(ctx); err != nil {
		return nil, err
	}
	return handoff.NewService(handoff.Dependencies{
		Registry: s.registry,
		Drivers:  s.drivers,
		Launcher: s.launcher,
	}).PerformAccountHandoff(ctx, runtimeID, target)
}

func (s *RuntimeApplicationService) ContextHandoff(ctx context.Context, runtimeID, provider, profile string) (*registry.RuntimeSession, error) {
	if err := s.ready(ctx); err != nil {
		return nil, err
	}
	return handoff.NewService(handoff.Dependencies{
		Registry: s.registry,
		Drivers:  s.drivers,
		Launcher: s.launcher,
	}).PerformContextHandoff(ctx, runtimeID, provider, profile)
}

func (s *RuntimeApplicationService) Delete(ctx context.Context, runtimeID string) error {
	if err := s.ready(ctx); err != nil {
		return err
	}
	return s.registry.Delete(runtimeID)
}
