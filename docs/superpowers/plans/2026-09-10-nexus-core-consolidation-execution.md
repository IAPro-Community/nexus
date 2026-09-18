# Nexus Core Consolidation — Executable Plan

## Scope and boundaries

Preserve the existing dirty worktree, public `/api/v1` contracts, provider
aliases, quota evidence semantics, and optional Maestro boundary. Do not add a
new provider, scheduler, workflow language, UI redesign, or storage rewrite.
No automatic commit/push. Each implementation task is test-first.

## Task 1 — Characterization baseline [DONE]

- Files: `scripts/characterize-nexus.sh`, `internal/nexus/characterization_test.go`, `docs/refactoring/CHARACTERIZATION-REPORT.md`
- Interfaces consumed: existing store, intelligence compiler, CLI sources.
- Interfaces produced: executable inventory and baseline report.
- Failing test: fixture initially referenced a package-local helper and failed to compile.
- Expected RED: harness failure only.
- Implementation: local store fixture and read-only static inventory.
- Expected GREEN: focused Go tests and script exit 0.
- Regression verification: `go test ./internal/nexus ./internal/nexus/intelligence ./internal/nexus/store`; `./scripts/characterize-nexus.sh`.
- Commit: `test(go): characterize nexus core contracts` (manual, not created).

## Task 2 — Typed AgentSpec normalization

- Files: `internal/nexus/config.go`, new focused model/normalizer file, tests.
- Interfaces consumed: `store.Agent`, `store.AgentRevision`, `AgentConfig`, `WorkPackage`.
- Interfaces produced: typed `AgentSpec`, default normalization for legacy role-only agents.
- Failing test: QA and DevOps produce distinct compiled specialization; empty instructions preserve legacy behavior.
- Expected RED: `AgentSpec` and compiler request types absent.
- Implementation: additive typed fields in revision config; no prompt blob; normalization from legacy `Agent.Role`.
- Expected GREEN: semantics tests pass and old config JSON still parses.
- Regression verification: focused `go test ./internal/nexus/...` and store tests.
- Commit: `refactor(agents): introduce persistent agent specialization` (manual).

## Task 3 — Execution context compiler/provenance

- Files: `internal/nexus/intelligence/types.go`, `engine.go`, tests; reuse existing compiler.
- Interfaces consumed: existing `CompilePrompt`, `WorkPackageOutline`, Maestro status/catalog.
- Interfaces produced: typed request/result sections with source provenance and compatibility wrapper.
- Failing test: context contains AgentSpec, task role, project facts, optional Maestro guidance, and provider/runtime constraints with sources.
- Expected RED: no typed sections/provenance.
- Implementation: generalize the existing compiler; do not create a competing compiler.
- Expected GREEN: old `CompilePrompt` output remains valid; new result is inspectable without secrets.
- Regression verification: intelligence tests and mission prompt tests.
- Commit: `refactor(execution): centralize context compilation` (manual).

## Task 4 — Agent revision persistence and runtime provenance

- Files: `internal/nexus/config.go`, `internal/nexus/nexus.go`, store tests.
- Interfaces consumed: current revisions/generations and `StartAgent`.
- Interfaces produced: normalized spec persisted in the effective revision; generation remains linked to revision.
- Failing test: provider switch preserves AgentSpec; new generation points at effective revision; legacy role-only agent starts.
- Expected RED: current config has no spec and provider switch cannot assert identity semantics.
- Implementation: preserve old fields, normalize on read/write, keep `RevisionID` authoritative.
- Expected GREEN: persistence and provider-switch tests pass.
- Regression verification: `go test ./internal/nexus ./internal/nexus/store`.
- Commit: `refactor(agents): preserve specialization across generations` (manual).

## Task 5 — Unified execution request/pipeline

- Files: existing direct execution and mission executor files, new cohesive pipeline file only if no equivalent exists, tests.
- Interfaces consumed: `executeAgentPrompt`, `nexusPackageExecutor`, `MissionRunner`, provider driver registry.
- Interfaces produced: shared request/context/launch path for Direct and Mission/WorkPackage modes.
- Failing test: direct and WorkPackage modes both resolve the same AgentSpec/task-role composition; Maestro off remains valid.
- Expected RED: direct path bypasses existing WorkPackage compiler and mission path has separate composition.
- Implementation: adapt current paths to one application-level pipeline while retaining runtime/runner lifecycle.
- Expected GREEN: direct, automated, and orchestrated tests converge before provider launch.
- Regression verification: focused mission/direct tests, race test on touched packages.
- Commit: `refactor(execution): unify agent execution pipeline` (manual).

## Task 6 — Maestro integration through compiled context

- Files: Maestro boundary and compiler tests/docs only where needed.
- Interfaces consumed: `MaestroClient`, gate validation, project Maestro mode.
- Interfaces produced: optional guidance section with explicit provenance; Direct OFF does not call Maestro.
- Failing test: Maestro ON adds guidance; OFF leaves direct context unchanged; unavailable configured gates fail closed as existing contract requires.
- Expected RED: new compiler request does not model optional guidance.
- Implementation: adapt current boundary; no Maestro reimplementation.
- Expected GREEN: existing Maestro tests plus new context tests.
- Regression verification: `go test ./internal/nexus/...`.
- Commit: `refactor(maestro): route optional guidance through execution context` (manual).

## Task 7 — CLI truth and parsing

- Files: `internal/app/nexus_cli_cmds.go`, command registry/helper files only if justified, tests, CLI matrix/report.
- Interfaces consumed: existing `Run`, Core plan compiler/runner, provider registry.
- Interfaces produced: truthful help/registry and table-driven positional parsing.
- Failing test: every advertised plan subcommand has an accessible handler; `agents PROJECT` selects PROJECT; aliases remain valid.
- Expected RED: current compile/run and one-argument agents cases fail.
- Implementation: wire existing Core capability where reachable; remove only demonstrably false help; derive help from one registry if compatible.
- Expected GREEN: CLI tests cover help, aliases, missing args, unknown commands, and JSON flags.
- Regression verification: `go test ./internal/app ./internal/nexus/...`; binary smoke checks.
- Commit: `fix(terminal): make nexus command surface truthful` (manual).

## Task 8 — Vertical API extraction and contracts

- Files: `internal/control/web/*` touched by evidence, route/handler tests, API docs.
- Interfaces consumed: current handler responses, route composition already present.
- Interfaces produced: cohesive application calls without provider/prompt logic in HTTP adapters.
- Failing test: public routes preserve request/response/error contracts.
- Expected RED: only for handlers proven to duplicate business rules.
- Implementation: incremental vertical extraction; no god service.
- Expected GREEN: route contract suite, auth/CSRF and metadata tests.
- Regression verification: `go test ./internal/control/web/...` and full Go suite.
- Commit: `refactor(web): modularize nexus transport by capability` (manual).

## Task 9 — Documentation and full verification

- Files: architecture/product glossary, `DEV/WORKLOG.md`, `DEV/VERIFY.md`, `DEV/HANDOFF.md`, `DEV/CONTEXT.md`, `DEV/SPECS/ACTIVE.md`, report.
- Interfaces consumed: final implementation and test evidence.
- Interfaces produced: documented Agent/Revision/Generation/Provider/Session/Spec/WorkPackage/Mission/Maestro model.
- Failing test: documentation/report checklist identifies any unmatched claim.
- Expected RED: stale statements are found by review, not hidden.
- Implementation: update durable docs with exact evidence and residual risks.
- Expected GREEN: all applicable repository gates pass or are explicitly marked blocked/not verified.
- Regression verification: full gate listed in `DEV/VERIFY.md`, final diff/status/stat review.
- Commit: `docs(nexus): document consolidated execution model` (manual).

## Final review gate

Run a fresh code/security/architecture review over the diff, then rerun the
complete verification gate. No completion claim is valid without fresh command
output and an explicit residual-risk list.
