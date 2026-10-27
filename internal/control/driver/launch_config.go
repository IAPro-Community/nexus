package driver

import (
	"fmt"
	"sort"
	"strings"

	"github.com/IAPro-Community/nexus/internal/control/flags"
	"github.com/IAPro-Community/nexus/internal/core/config"
)

// ApplyLaunchConfiguration converts the subset of Agent configuration that is
// truthfully supported by provider CLIs into command arguments. Unsupported
// values return an error instead of being silently ignored.
func ApplyLaunchConfiguration(provider, model string, options map[string]any, baseArgs []string) ([]string, error) {
	provider = strings.ToLower(strings.TrimSpace(provider))
	cfg, _ := config.LoadConfig()
	args := flags.Normalize(provider, baseArgs, cfg.FlagAliases)
	model = strings.TrimSpace(model)
	if model != "" {
		switch provider {
		case "codex", "claude", "gemini", "opencode", "agy", "pi", "omp", "mimo":
			args = append(args, "--model", model)
		case "fake":
			args = append(args, "--model", model)
		default:
			return nil, fmt.Errorf("provider %q does not declare model override support", provider)
		}
	}
	if len(options) == 0 {
		return args, nil
	}

	keys := make([]string, 0, len(options))
	for key := range options {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		value := options[key]
		switch key {
		case "extra_args":
			extra, err := stringSliceOption(value)
			if err != nil {
				return nil, fmt.Errorf("provider option extra_args: %w", err)
			}
			args = append(args, flags.NormalizePreservingRepeatedArgs(provider, extra, cfg.FlagAliases)...)
		case "mode":
			// UI-only Safe/Plan/YOLO marker. Flags live in extra_args.
			continue
		case "command_template":
			if _, ok := value.(string); !ok {
				return nil, fmt.Errorf("provider option command_template must be a string")
			}
			// Resolved by launcher after provider arguments are normalized.
			continue
		case "target_kind":
			kind, ok := value.(string)
			if !ok {
				return nil, fmt.Errorf("provider option target_kind must be a string")
			}
			switch strings.ToLower(strings.TrimSpace(kind)) {
			case "", "local", "docker", "docker-compose", "wsl", "ssh", "custom":
			default:
				return nil, fmt.Errorf("provider option target_kind %q is not supported", kind)
			}
			continue
		case "target_workspace":
			if _, ok := value.(string); !ok {
				return nil, fmt.Errorf("provider option target_workspace must be a string")
			}
			continue
		case "target_profile_home":
			if _, ok := value.(string); !ok {
				return nil, fmt.Errorf("provider option target_profile_home must be a string")
			}
			continue
		case "target_probe_template":
			if _, ok := value.(string); !ok {
				return nil, fmt.Errorf("provider option target_probe_template must be a string")
			}
			continue
		case "target_provider_version":
			if _, ok := value.(string); !ok {
				return nil, fmt.Errorf("provider option target_provider_version must be a string")
			}
			continue
		case "agent_mode_id":
			if _, ok := value.(string); !ok {
				return nil, fmt.Errorf("provider option agent_mode_id must be a string")
			}
			// Agent mode identifies persisted Nexus metadata, not a provider CLI flag.
			continue
		case "read_only_by_default":
			if _, ok := value.(bool); !ok {
				return nil, fmt.Errorf("provider option read_only_by_default must be a boolean")
			}
			// This is an Agent policy signal retained in its immutable config revision.
			continue
		case "nexus_browser_enabled":
			if _, ok := value.(bool); !ok {
				return nil, fmt.Errorf("provider option nexus_browser_enabled must be a boolean")
			}
			// Consumed by the launcher's browser MCP provisioning; never a CLI flag.
			continue
		case "execution_adapter", "origin", "role_type":
			// Agent metadata retained in its immutable config revision.
			continue
		default:
			return nil, fmt.Errorf("provider option %q is not supported by the supervised launcher", key)
		}
	}
	return args, nil
}

func stringSliceOption(value any) ([]string, error) {
	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...), nil
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			text, ok := item.(string)
			if !ok || strings.TrimSpace(text) == "" {
				return nil, fmt.Errorf("must contain non-empty strings")
			}
			out = append(out, text)
		}
		return out, nil
	default:
		return nil, fmt.Errorf("must be an array of strings")
	}
}
