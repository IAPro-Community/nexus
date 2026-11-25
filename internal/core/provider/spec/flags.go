package spec

import (
	"strings"

	sdkprovider "github.com/IAPro-Community/IAPro-SDK/providers/spec"
)

type CanonicalAlias = sdkprovider.CanonicalAlias

var BuiltinAliases = sdkprovider.BuiltinAliases

// Normalize maps recognized aliases to provider-native flags.
func Normalize(provider string, args []string, userAliases map[string]map[string][]string) []string {
	return sdkprovider.Normalize(provider, args, userAliases)
}

// NormalizePreservingRepeatedArgs maps aliases while retaining repeated
// provider-native arguments such as --add-dir. Alias expansions remain
// deduplicated against each other and already-emitted native equivalents.
func NormalizePreservingRepeatedArgs(provider string, args []string, userAliases map[string]map[string][]string) []string {
	return normalize(provider, args, userAliases, true)
}

func normalize(provider string, args []string, userAliases map[string]map[string][]string, preserveRepeatedArgs bool) []string {
	provider = strings.ToLower(strings.TrimSpace(provider))
	if len(args) == 0 {
		return args
	}

	var result []string
	seenEmitted := make(map[string]bool)
	aliasEmitted := make(map[string]bool)

	for i := 0; i < len(args); i++ {
		arg := args[i]
		flagName := arg
		flagVal := ""
		hasInlineVal := false

		if strings.HasPrefix(arg, "--") && strings.Contains(arg, "=") {
			parts := strings.SplitN(arg, "=", 2)
			flagName = parts[0]
			flagVal = parts[1]
			hasInlineVal = true
		}

		if userAliases != nil {
			if provMap, ok := userAliases[flagName]; ok {
				if flags, hasProv := provMap[provider]; hasProv && len(flags) > 0 {
					for _, f := range flags {
						result = append(result, f)
						seenEmitted[f] = true
						aliasEmitted[f] = true
					}
					continue
				}
			}
		}

		if alias, ok := BuiltinAliases[flagName]; ok {
			if flags, hasProv := alias.ProviderFlags[provider]; hasProv && len(flags) > 0 {
				val := flagVal
				if alias.TakesValue {
					if !hasInlineVal && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
						i++
						val = args[i]
					}
				}

				expanded := make([]string, 0, len(flags))
				for _, f := range flags {
					resolved := f
					if alias.TakesValue && strings.Contains(resolved, "{value}") {
						resolved = strings.ReplaceAll(resolved, "{value}", val)
					} else if alias.TakesValue && val != "" && !strings.Contains(f, "{value}") && f == flags[len(flags)-1] && f != "-c" {
						resolved = val
					}
					expanded = append(expanded, resolved)
				}

				for _, exp := range expanded {
					if seenEmitted[exp] {
						continue
					}
					result = append(result, exp)
					if exp == "-c" || exp == "-p" {
						continue
					}
					seenEmitted[exp] = true
					aliasEmitted[exp] = true
				}
				continue
			}
		}

		if !seenEmitted[arg] || (preserveRepeatedArgs && !aliasEmitted[arg]) {
			result = append(result, arg)
			seenEmitted[arg] = true
		}
	}

	return result
}
