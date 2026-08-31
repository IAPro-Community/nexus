# Nexus V0 Final Audit

## Source

Baseline: `c96885218b3cff319f481aed4423dc5fd7b52a30`  
Final SHA: `4dedddf43febf44c29307ebbe2ac61c11e158778`  
Branch: `feat/nexus-v0-implementation`

## Capability Preservation

The baseline capabilities were audited against source, existing tests and current diffs. No capabilities were removed. One real regression was found in `TerminalPane`, where the consumer used the old protocol discriminator after the parser migration; it was repaired by switching control handling to `frame.type === 'control'` and `frame.event`.

## New V0 Capabilities

| Capability | Status | Evidence |
|---|---|---|
| Terminal protocol isolation | PASS | protocol tests and full frontend suite |
| Workspace presentation/Desktop | PARTIAL | model/provider/renderer implementation and tests |
| Ask existing Agent | PARTIAL | API/UI implementation and validation test |
| Attention stale-runtime suppression | PARTIAL | visibility regression and durable dismissal |
| Project Shell | PARTIAL | `ShellDriver` + `ProjectShellSurface`; focused Go tests and frontend build pass; native/browser lifecycle unverified |
| Context Readiness | PARTIAL | five-state fingerprint contract and drift tests; hydration persistence/gate not wired |
| Composer/Flow V0 | PARTIAL | reversible façade, deterministic DAG and bounded capsule/receipt contracts added; dedicated UI/persistence/browser journey incomplete |

## Tests

| Suite | Platform | Result |
|---|---|---|
| Go test/vet/build/race | Linux | PASS |
| Frontend typecheck/lint/tests/build | Linux | PASS — 30 files / 105 tests in canonical worktree |
| Embedded bundle sync/diff check | Linux | PASS |
| Browser local harness | Chromium/Linux | SMOKE PASS; authenticated bootstrap failed, therefore no full user journey claim |
| Native Windows runtime | Windows | UNVERIFIED_ON_WINDOWS |
| Native macOS runtime | macOS | UNVERIFIED_ON_MACOS |
| Same-SHA CI/release | External | UNVERIFIED |

## Security

Local CSRF, origin, session, launch environment, config and race coverage passed. `govulncheck`, `gitleaks` and `actionlint` were unavailable. No new critical finding was observed.

## Remaining Issues

- Project Shell and dedicated Composer/Context Readiness/Flow V0 surfaces remain incomplete.
- Native Windows/macOS, real provider, full authenticated browser, same-SHA CI and release evidence remain unavailable.
- No push, merge or release was performed.

## Verdict

`PARTIAL` — the canonical recovery is consolidated and locally validated, but it is not a production GO.
