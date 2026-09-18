# Nexus Core Consolidation — Characterization Report

Date: 2026-09-10
Branch: `feat/nexus-maximum-delivery`
HEAD analyzed: `bfc90fc98acd700c165c8e7df26ac3d3670dce54`
Worktree: dirty before this campaign; existing changes were preserved.

## Evidence commands

- `git branch --show-current` → `feat/nexus-maximum-delivery`
- `git rev-parse HEAD` → `bfc90fc98acd700c165c8e7df26ac3d3670dce54`
- `go test ./internal/nexus ./internal/nexus/intelligence ./internal/nexus/store` → PASS
- `./scripts/characterize-nexus.sh` → PASS; read-only inventory
- Static evidence was gathered with `rg` over `internal/`, `cmd/`, `web/`, `docs/`, and `DEV/`.

## Capability matrix

| Capability | CLI | API | Core | Web/Desktop | Real status | Evidence |
|---|---|---|---|---|---|---|
| project list/create | wired | wired | implemented | wired | IMPLEMENTED | `internal/app/nexus_cli_cmds.go`, project handlers/store |
| agent list/create/start/ask | wired | wired | implemented | wired | IMPLEMENTED | agent handlers, `Nexus.StartAgent`, store |
| agent revision/generation | indirect | detail response | implemented | surfaced | IMPLEMENTED | `store/agents.go`, migration `0001_init.sql` |
| persistent specialization | role only | role only | IMPLEMENTED | surfaced via revisions | FIXED IN CAMPAIGN | typed `intelligence.AgentSpec` in revision config; legacy normalization |
| plan list/create/show | wired | wired | implemented | wired | IMPLEMENTED | `nexus_cli_cmds.go`, `plan.go`, plan handlers |
| plan compile | wired | core compiler | implemented | partial | FIXED IN CAMPAIGN | CLI now calls `Nexus.CompilePackagePrompt` |
| plan run | wired | runner path | implemented through mission runner | partial | FIXED IN CAMPAIGN | CLI now calls `Nexus.StartMissionRun` |
| mission execution | not direct | wired | implemented | wired | IMPLEMENTED | `runner`, mission handlers |
| provider registry | direct aliases | metadata/resources | implemented | wired | IMPLEMENTED | `internal/core/provider`, driver registry |
| quota/evidence | wired | wired | implemented | wired | IMPLEMENTED | `internal/core/quota`; UNKNOWN is distinct |
| Maestro guidance | optional command/API | wired | optional boundary | surfaced | IMPLEMENTED/OPTIONAL | `MaestroClient`, gates, ADR |

Classification is based on reachable symbols and tests, not help text alone.

## CLI truth findings

1. `planCmd` help announces `compile` and `run`, but its switch currently only
   implements `list/ls`, `create`, and `show`; these two surfaces return an
   unknown-subcommand error.
2. `plan compile` has a Core equivalent: `NexusEngine.CompilePrompt` and
   `compileTargetPackagePrompt` in `internal/nexus/plan.go`.
3. `plan run` has a Core equivalent through `MissionRunner.ExecuteNextStep`
   and `Nexus.Runner()`, but no CLI adapter was found in the dispatcher.
4. `agentsCmd` receives `args[1:]` from `Run`; its project selection checks
   `len(args) > 1` and reads `args[1]`, so `nexus agents PROJECT` does not use
   the positional project argument. This is a confirmed parsing defect.
5. Provider aliases are dispatched through the existing provider registry and
   must remain compatible.

## Agent semantics before consolidation

- `store.Agent` is persistent and provider-independent at the identity level.
- `AgentRevision.Config` is an opaque JSON object parsed as `nexus.AgentConfig`.
- `AgentConfig` contains provider/profile/model/workspace/isolation/options and
  Maestro mode, but no typed specialization contract.
- `RuntimeGeneration.RevisionID` already records the effective revision, and
  `ProviderSession` is separate from Agent identity.
- WorkPackage role was passed to the existing intelligence compiler as one
  string and was not explicitly composed with persistent Agent role.

## Agent semantics after the first implementation slice

- `intelligence.AgentSpec` is typed and persisted inside `AgentConfig`, which
  is the JSON payload of `AgentRevision`.
- `NormalizeAgentSpec` maps legacy `store.Agent.Role` to `AgentSpec.Role`
  without inventing instructions or capabilities.
- `CompileExecutionContext` is the single generalized compiler. It emits
  inspectable sections with sources and composes persistent Agent role with
  WorkPackage role.
- Direct `AskAgent` compiles custom AgentSpec behavior before submission;
  role-only legacy agents keep their prior raw-prompt behavior.
- Mission/WorkPackage compilation loads the assigned Agent revision and uses
  that same compiler. Provider switching does not change AgentSpec.

## Risks and debt discovered

- Adding specialization by replacing `Role` would break legacy agents and
  existing matching logic; normalization must be additive and revision-aware.
- Adding a second prompt compiler would duplicate `NexusEngine.CompilePrompt`;
  the existing intelligence compiler is the consolidation point.
- HTTP handlers still contain transport plus some application orchestration;
  extraction must be vertical and contract-tested, not cosmetic.
- Native Windows/macOS behavior is not verifiable on this Linux host.

## Characterization tests added

- Legacy role survives persistence and a generation keeps its revision link.
- Existing WorkPackage role reaches the current compiler output.

These tests passed before any production-code semantic change.
