package spec

import "github.com/IAPro-Community/nexus/internal/runtime"

func overlayEnv(base []string, overrides map[string]string, unset []string) []string {
	return runtime.EnvSet(base, overrides, unset...)
}
