package profile

import (
	coreprovider "github.com/IAPro-Community/nexus/internal/core/provider"
	"github.com/IAPro-Community/nexus/internal/core/provider/adapters/agy"
	"github.com/IAPro-Community/nexus/internal/core/provider/adapters/claude"
	"github.com/IAPro-Community/nexus/internal/core/provider/adapters/codex"
	"github.com/IAPro-Community/nexus/internal/core/provider/adapters/cursor"
	"github.com/IAPro-Community/nexus/internal/core/provider/adapters/opencode"
	"github.com/IAPro-Community/nexus/internal/core/provider/adapters/zcode"
)

func init() {
	coreprovider.RegisterBuiltin(func() coreprovider.Provider { return codex.New() })
	coreprovider.RegisterBuiltin(func() coreprovider.Provider { return agy.New() })
	coreprovider.RegisterBuiltin(func() coreprovider.Provider { return claude.New() })
	coreprovider.RegisterBuiltin(func() coreprovider.Provider { return opencode.New() })
	coreprovider.RegisterBuiltin(func() coreprovider.Provider { return cursor.New() })
	coreprovider.RegisterBuiltin(func() coreprovider.Provider { return zcode.New() })
}

// Registry returns the shared extensible provider adapter registry.
func Registry() *coreprovider.Registry {
	return coreprovider.DefaultRegistry()
}
