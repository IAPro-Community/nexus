package host

import (
	"context"

	"github.com/IAPro-Community/nexus/internal/control/registry"
)

var (
	PerformAccountHandoff func(ctx context.Context, sourceRuntimeID, targetSpec string) (*registry.RuntimeSession, error)
	PerformContextHandoff func(ctx context.Context, sourceRuntimeID, targetProvider, targetProfile string) (*registry.RuntimeSession, error)
)
