package flags

import "github.com/IAPro-Community/nexus/internal/core/provider/spec"

type CanonicalAlias = spec.CanonicalAlias

var BuiltinAliases = spec.BuiltinAliases

func Normalize(provider string, args []string, userAliases map[string]map[string][]string) []string {
	return spec.Normalize(provider, args, userAliases)
}
