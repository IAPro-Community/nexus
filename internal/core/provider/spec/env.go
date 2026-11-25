package spec

import "github.com/IAPro-Community/nexus/internal/runtime"

func overlayEnv(base []string, profileHome string, overrides map[string]string, unset []string) []string {
	return runtime.EnvSet(base, runtime.ApplyProfileIsolationEnv(profileHome, overrides), unset...)
}
