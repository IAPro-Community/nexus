package launcher

import (
	"fmt"
	"strings"
)

// CommandTemplateContext contains the provider-neutral values that a custom
// execution target may interpolate. Templates are still argv-only and never
// evaluated by a shell.
type CommandTemplateContext struct {
	Cwd         string
	TargetCwd   string
	Args        []string
	Provider    string
	Profile     string
	ProfileHome string
}

// ResolveCommandTemplate preserves the v1 compatibility API for existing
// {cwd}/{args} templates.
func ResolveCommandTemplate(template, cwd string, runtimeArgs []string) (string, []string, error) {
	return ResolveCommandTemplateContext(template, CommandTemplateContext{
		Cwd:  cwd,
		Args: runtimeArgs,
	})
}

// ResolveCommandTemplateContext expands a safe custom command template.
//
// Supported placeholders:
//   {cwd}          host-side Nexus project workspace
//   {target_cwd}   workspace path as seen by the custom target; falls back to {cwd}
//   {args}         runtime/provider arguments; must occupy a whole token
//   {provider}     logical Nexus provider id
//   {profile}      logical Nexus profile id
//   {profile_home} optional shared provider/profile home exposed by the target
//
// The grammar is intentionally not a shell. Pipes, redirects, command
// substitution and environment interpolation remain rejected.
func ResolveCommandTemplateContext(template string, ctx CommandTemplateContext) (string, []string, error) {
	tokens, err := splitCommandTemplate(template)
	if err != nil {
		return "", nil, err
	}
	if len(tokens) == 0 {
		return "", nil, fmt.Errorf("command template is empty")
	}

	targetCwd := strings.TrimSpace(ctx.TargetCwd)
	if targetCwd == "" {
		targetCwd = ctx.Cwd
	}
	values := map[string]string{
		"{cwd}":        ctx.Cwd,
		"{target_cwd}": targetCwd,
		"{provider}":   strings.TrimSpace(ctx.Provider),
		"{profile}":    strings.TrimSpace(ctx.Profile),
	}
	if strings.TrimSpace(ctx.ProfileHome) != "" {
		values["{profile_home}"] = ctx.ProfileHome
	}

	var out []string
	seenArgs := false
	for _, token := range tokens {
		if token == "{args}" {
			out = append(out, ctx.Args...)
			seenArgs = true
			continue
		}
		if strings.Contains(token, "{args}") {
			return "", nil, fmt.Errorf("{args} must occupy a whole command-template token")
		}

		expanded := token
		for placeholder, value := range values {
			expanded = strings.ReplaceAll(expanded, placeholder, value)
		}
		if strings.Contains(expanded, "{profile_home}") {
			return "", nil, fmt.Errorf("command template requires {profile_home}, but no shared profile home was configured")
		}
		if strings.Contains(expanded, "{provider}") {
			return "", nil, fmt.Errorf("command template requires {provider}, but no provider was configured")
		}
		if strings.Contains(expanded, "{profile}") {
			return "", nil, fmt.Errorf("command template requires {profile}, but no profile was configured")
		}
		out = append(out, expanded)
	}

	if !seenArgs {
		return "", nil, fmt.Errorf("command template must include {args} as its own argument")
	}
	if strings.TrimSpace(out[0]) == "" {
		return "", nil, fmt.Errorf("command template has an empty executable")
	}
	return out[0], out[1:], nil
}

// splitCommandTemplate is intentionally a small, non-shell grammar: spaces,
// quotes and backslash escapes are supported; shell operators are rejected.
func splitCommandTemplate(input string) ([]string, error) {
	var tokens []string
	var current strings.Builder
	var quote rune
	escaped := false
	flush := func() {
		if current.Len() > 0 {
			tokens = append(tokens, current.String())
			current.Reset()
		}
	}
	for _, r := range input {
		if escaped {
			switch r {
			case ' ', '\t', '\n', '\\', '\'', '"':
				current.WriteRune(r)
			default:
				// Preserve a Windows/path backslash when it was not escaping a
				// grammar metacharacter (for example C:\\Tools\\opencode.exe).
				current.WriteRune('\\')
				current.WriteRune(r)
			}
			escaped = false
			continue
		}
		if quote != 0 {
			if r == quote {
				quote = 0
			} else {
				// Inside quotes backslash is literal. This keeps Windows paths
				// stable; callers can use the other quote style when a literal
				// quote character is required.
				current.WriteRune(r)
			}
			continue
		}
		if r == '\\' {
			escaped = true
			continue
		}
		switch r {
		case '\'', '"':
			quote = r
		case ' ', '\t', '\n':
			flush()
		case '|', ';', '&', '<', '>', '`', '$':
			return nil, fmt.Errorf("command template cannot contain shell operators")
		default:
			current.WriteRune(r)
		}
	}
	if escaped || quote != 0 {
		return nil, fmt.Errorf("command template has an unfinished escape or quote")
	}
	flush()
	return tokens, nil
}
