# CLI command matrix

Source of truth for the top-level command surface is `internal/app/app.go`
(`Run` and `usage()`). The matrix below is intentionally compact; provider
subcommand options remain provider-native and are not duplicated here.

| Command family | Dispatch | Help documented | Alias/notes | Core path |
| --- | --- | --- | --- | --- |
| `help` | `usage()` | yes | `-h`, `--help` | render only |
| `version` | `versionCmd` | yes | `-v`, `--version`, `--json` | buildinfo |
| `web` | `controlWebCmd` | yes | `open` | Web/Core server |
| `projects` | `projectsCmd` | yes | `project` | Nexus store |
| `agents` | `agentsCmd` | yes | `agent` | Nexus store |
| `plan` | `planCmd` | yes | `plans` | Nexus WorkPlan |
| `providers` | `providersCmd` | yes | `--json` | provider registry |
| `profiles` | `profilesCmd` | yes | `list`, `ls` | profile store |
| `paths` | `paths` | yes | — | config/data paths |
| `start`/`stop`/`ps`/`attach` | control commands | yes | `running`, `ui`, `control` | runtime registry/driver |
| `handoff`/`continue` | control/resume dispatch | yes | `continue` has context-sensitive routing | session/runtime |
| `usage` | `usageCmd` | yes | `quota` | quota engine |
| `doctor` | `doctorCmd` | yes | `--json`, `--bundle` | diagnostics |
| `security` | `securityCmd` | yes | `--json` | security audit |
| `login`/`logout`/`add`/`remove`/`use` | profile commands | yes | `rm`, `switch`, `swap` | profile/provider core |
| `rename` | `renameCmd` | yes | — | profile store |
| `sessions`/`workspaces`/`bind`/`unbind`/`bindings` | workspace/session commands | yes | JSON variants | workspace/session core |
| `current` | `currentCmd` | yes | — | provider/profile state |
| `run`/`resume` | execution commands | yes | provider-native resume retained | provider/runtime core |
| `update` | `updateCmd` | yes | `upgrade` | Nexus update service |
| `export`/`issue-report` | export/report commands | yes | — | local diagnostics |
| `maestro` | `maestroCmd` | yes | optional integration only | Maestro boundary |
| `completion` | `completionCmd` | yes | bash/zsh/fish/powershell | render only |
| provider direct commands | `executeProviderWithSmartSelection` | yes | `codex`, `agy`, `claude`, `opencode`, `gemini`, `cursor`; `provider:profile` | provider registry |

## Verified plan subcommands

`plan compile <plan-id> <package-id> [phase-id]` is wired to
`Nexus.CompilePackagePrompt` and returns the existing compilation result.
`plan run <plan-id> [agent-id]` is wired to `Nexus.StartMissionRun` and the
existing `runner.DefaultAutonomyContract`; it does not create a second runner.
Both cases have missing-argument tests in `internal/app/app_test.go`.

The `agents` dispatcher receives arguments after the command token, so a
project selector is read from `args[0]`. This is covered by a two-project
selection test. A full generated registry remains deferred because provider
direct commands intentionally retain their provider-native parser and aliases.

The next CLI work should add table-driven tests for unknown commands, missing
arguments, help, aliases, and exit codes. This consolidation does not replace
the existing manual parser because direct provider compatibility is a public
contract.
