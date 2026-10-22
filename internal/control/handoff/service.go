package handoff

import (
	"context"

	"github.com/IAPro-Community/nexus/internal/control/driver"
	"github.com/IAPro-Community/nexus/internal/control/launcher"
	"github.com/IAPro-Community/nexus/internal/control/registry"
)

// LaunchPort is the narrow runtime boundary required by handoff. It keeps
// account/context transfer independent from the concrete launcher singleton.
type LaunchPort interface {
	Launch(context.Context, launcher.LaunchOptions) (*registry.RuntimeSession, error)
}

// Dependencies are the owned control-plane dependencies for a handoff flow.
// Callers should provide the same registry and launcher used by their Nexus
// application instance.
type Dependencies struct {
	Registry *registry.Registry
	Drivers  *driver.Registry
	Launcher LaunchPort
}

// Service coordinates account and context handoffs with explicit ownership.
type Service struct {
	registry *registry.Registry
	drivers  *driver.Registry
	launcher LaunchPort
}

func NewService(deps Dependencies) *Service {
	if deps.Registry == nil {
		deps.Registry = registry.DefaultRegistry()
	}
	if deps.Drivers == nil {
		deps.Drivers = driver.DefaultRegistry()
	}
	if deps.Launcher == nil {
		deps.Launcher = launcher.Default()
	}
	return &Service{registry: deps.Registry, drivers: deps.Drivers, launcher: deps.Launcher}
}

func defaultService() *Service {
	return NewService(Dependencies{})
}

func cloneRuntimeLabels(labels map[string]string) map[string]string {
	if len(labels) == 0 {
		return nil
	}
	out := make(map[string]string, len(labels))
	for key, value := range labels {
		out[key] = value
	}
	return out
}
