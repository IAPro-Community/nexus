package spec

import (
	"strings"
)

// CanonicalAlias defines the flag mapping per provider.
type CanonicalAlias struct {
	Description   string
	TakesValue    bool
	ProviderFlags map[string][]string
}

// BuiltinAliases defines universal aliases shared across AI CLI tools.
var BuiltinAliases = map[string]CanonicalAlias{
	"--yolo": {
		Description: "Bypass permission prompts and approval checks",
		ProviderFlags: map[string][]string{
			"agy":    {"--dangerously-skip-permissions"},
			"claude": {"--dangerously-skip-permissions"},
			"codex":  {"--dangerously-bypass-approvals-and-sandbox"},
		},
	},
	"-y": {
		Description: "Short alias for --yolo",
		ProviderFlags: map[string][]string{
			"agy":    {"--dangerously-skip-permissions"},
			"claude": {"--dangerously-skip-permissions"},
			"codex":  {"--dangerously-bypass-approvals-and-sandbox"},
		},
	},
	"--continue": {
		Description: "Continue the most recent conversation session",
		ProviderFlags: map[string][]string{
			"agy":    {"--continue"},
			"claude": {"--continue"},
			"codex":  {"resume", "--last"},
		},
	},
	"-c": {
		Description: "Short alias for --continue (AGY/Claude only; Codex uses native -c/--config)",
		ProviderFlags: map[string][]string{
			"agy":    {"--continue"},
			"claude": {"--continue"},
		},
	},
	"--resume": {
		Description: "Resume a specific session by ID",
		TakesValue:  true,
		ProviderFlags: map[string][]string{
			"agy":    {"--conversation={value}"},
			"claude": {"--resume", "{value}"},
			"codex":  {"resume", "{value}"},
		},
	},
	"-r": {
		Description: "Short alias for --resume",
		TakesValue:  true,
		ProviderFlags: map[string][]string{
			"agy":    {"--conversation={value}"},
			"claude": {"--resume", "{value}"},
			"codex":  {"resume", "{value}"},
		},
	},
	"--print": {
		Description: "Run prompt non-interactively and print response",
		ProviderFlags: map[string][]string{
			"agy":    {"--print"},
			"claude": {"--print"},
			"codex":  {"exec"},
		},
	},
	"-p": {
		Description: "Short alias for --print (AGY/Claude only; Codex uses native -p/--profile)",
		ProviderFlags: map[string][]string{
			"agy":    {"--print"},
			"claude": {"--print"},
		},
	},
	"--effort": {
		Description: "Model reasoning effort (low, medium, high)",
		TakesValue:  true,
		ProviderFlags: map[string][]string{
			"agy":   {"--effort", "{value}"},
			"codex": {"-c", `model_reasoning_effort="{value}"`},
		},
	},
	"--plan": {
		Description: "Start agent in planning mode",
		ProviderFlags: map[string][]string{
			"agy": {"--mode", "plan"},
		},
	},
	"--accept-edits": {
		Description: "Start agent in accept edits mode",
		ProviderFlags: map[string][]string{
			"agy": {"--mode", "accept-edits"},
		},
	},
}

// Normalize maps recognized aliases to provider-native flags.
func Normalize(provider string, args []string, userAliases map[string]map[string][]string) []string {
	return normalize(provider, args, userAliases, false)
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
