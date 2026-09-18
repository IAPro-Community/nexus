# Nexus Final Evolution Audit — Phase A Findings

Date: 2026-09-11
Audited branch: `feat/nexus-maximum-delivery`
Audited SHA: `bf0caad103499450564d670dfaffd302b745c147`
Remote reference: `origin/feat/nexus-maximum-delivery` at `ab056f558fb2886b21db60ca7951cfb92deb46c7`
Platform: Ubuntu 25.10 / Linux x86_64
Toolchain: Go 1.25.0, Node v22.17.0, Bun 1.4.0
Mode: Phase A; no production-code corrections made.

## Preliminary verdict

**NO-GO.** This is an initial independent audit, not a completion claim.
The local candidate is two commits ahead of the remote reference, but fresh
full-suite verification did not complete within the bounded audit windows.
Several claims in the previous reports are stronger than the evidence found
here, and required specification documents are absent.

## Candidate and delta

- Worktree was clean at audit start; current generated build artifacts are not
  tracked changes.
- Local-only commits:
  - `cc1ebf0` — `LaunchModeResolver`, TTY supervised-default routing, colon
    prefix handling, and characterization tests.
  - `bf0caad` — verification-document update.
- Relative to `origin/feat/nexus-maximum-delivery`: 12 files, 1,276 additions,
  142 deletions.
- The evolution baseline document identifies `ab056f5` as the prior candidate
  but does not declare a canonical BASE_SHA. The prior pre-fix report uses
  `f7f3a05`; this ambiguity is a documentation gap and must be resolved before
  claiming a complete regression diff.

## Requirements ledger — initial findings

| ID | Requirement | Implementation evidence | Unit | Integration | E2E | Docs | Status | Severity |
|---|---|---|---|---|---|---|---|---|
| IC-01 | TTY provider launch defaults to supervised | `internal/app/launch_mode.go` and `app.go` | present | not freshly proven | not run | claimed | IMPLEMENTED_UNVERIFIED | P1 |
| IC-02 | `--direct`, `--supervised`, `--print`, env override | resolver tests exist | present | not freshly proven | not run | partial | IMPLEMENTED_UNVERIFIED | P1 |
| IC-03 | canonical `:nexus` and aliases | colon router implementation/tests | present | not freshly proven | not run | required docs missing | IMPLEMENTED_UNVERIFIED | P1 |
| IC-04 | provider input/escape byte preservation | prefix tests exist | partial | not proven for PTY/provider | not run | partial | PARTIAL | P1 |
| IC-05 | contextual completion | no fresh proof located | unknown | no | no | partial | IMPLEMENTED_UNVERIFIED | P1 |
| IC-06 | handoff transactional safety and terminal follow | state-shape tests exist; no real state-machine E2E | partial | partial | missing | partial | PARTIAL | P1 |
| AR-01 | one canonical AgentMatcher | no `AgentMatcher` implementation found; mission has local selection logic | absent/parallel heuristics | no | no | active spec requires it | MISSING | P1 |
| AR-02 | TaskRequirements V2 complete and canonical | current `TaskRequirements` remains provider-resource oriented; persisted flow fields are strings | partial | no | no | partial | PARTIAL | P1 |
| AR-03 | AgentSpec personality/specialization reaches runtime | AgentSpec supports role/instructions/responsibilities/capabilities/constraints, but creation UI only persists role and options | unit compiler tests | no real execution proof | no | partial | PARTIAL | P1 |
| AR-04 | task classification and decomposition | archetype/decomposition code exists, full `nexus run` path not proven | partial | no | no | partial | IMPLEMENTED_UNVERIFIED | P1 |
| RC-01 | canonical ResourceScheduler and policy precedence | scheduler/recommendation code exists | present | partial | no | partial | IMPLEMENTED_UNVERIFIED | P1 |
| RC-02 | `:switch auto` uses scheduler | no fresh end-to-end proof | unknown | no | missing | partial | IMPLEMENTED_UNVERIFIED | P1 |
| RC-03 | failover/handoff continuity truthful | statuses include unverified variants, but no real provider/fake E2E evidence | partial | partial | missing | claims too strong | PARTIAL | P1 |
| REG-01 | backward compatibility | no complete baseline-to-HEAD matrix yet | partial | partial | no | partial | PARTIAL | P1 |
| GATE-01 | fresh complete Go suite | command timed out at 240s; JSON run timed out at 90s while suites were still executing | — | — | — | previous report says PASS | BLOCKED_EXTERNAL | P1 |
| GATE-02 | fresh race suite | command timed out at 300s | — | — | — | previous report says PASS | BLOCKED_EXTERNAL | P1 |
| GATE-03 | fresh Web quality/E2E gates | `npm --prefix web run quality:full` and `make web-verify` timed out at 180s | partial | no | no | previous report says PASS | BLOCKED_EXTERNAL | P1 |
| GATE-04 | native Windows/macOS | no native runners | — | — | — | correctly marked unverified in places | BLOCKED_EXTERNAL | P1 |

## Concrete findings

### P1 — Previous verification claims are not reproducible in this audit

The fresh commands `go test -count=1 ./...`, `go test -race -count=1 ./...`,
`npm --prefix web run quality:full`, and `make web-verify` exceeded their
bounded timeouts. `go vet ./...`, `make security`, and `make build` completed.
The JSON Go run showed long-running host/web/profile tests and sudo attempts to
edit `/etc/hosts`; it did not reach a final result before timeout.

### P1 — Required architectural evidence is missing

The requested files `docs/refactoring/FUNCTIONAL_COVERAGE_MATRIX.md`,
`docs/architecture/agent-routing.md`, `docs/architecture/resource-routing.md`,
`docs/architecture/session-continuity.md`,
`docs/product/interactive-control.md`, and
`docs/product/account-switching.md` do not exist. This blocks documentation
consistency and the requested requirements ledger traceability.

### P1 — Agent routing is not proven as one canonical matcher

Global search found resource recommendation plus mission-specific selection
(`selectReusableFlowAgent`, `missionTaskRequirements`, and direct assignment
branches), but no canonical `AgentMatcher` contract returning eligibility,
score breakdown, confidence, and rejection reasons. This fails the explicit
single-source requirement until proven or implemented.

### P1 — Persistent-agent “personality” is only partially implemented

`intelligence.AgentSpec` can persist behavioral fields and the compiler renders
them. However, `NewAgentModal` creates an agent with a role and stores provider
options; it does not populate `AgentSpec.Instructions`, responsibilities,
strengths, domains, tags, or verification policy. Therefore specialty presets
are currently role/config presets, not complete persistent personalities.

### P1 — 409 execution boundary needs classification

`handleAgentStart` maps every `ResolveStartParams` error to HTTP 409, and
`handleAgentAsk` maps every `AskAgent` error to 409. The response body must be
captured for the reported “new agent” failure; without that exact body, the
failure cannot be safely classified as stale prepared context, missing
resource selection, already-live runtime, or another conflict.

### P2 — Baseline identity is inconsistent

`EVOLUTION_BASELINE.md` records `ab056f5` as the candidate while the prior
pre-fix report uses `f7f3a05` as baseline and labels local changes uncommitted.
The current SHA is `bf0caad`. The final report must preserve this history and
explicitly choose the technically justified comparison base.

## Security and platform observations

- `make security` completed with “No vulnerabilities found”. This is not a
  substitute for flow-level review of new routing/handoff paths.
- Native Windows and macOS execution is unavailable; cross-compilation is not
  native verification.
- Test output attempted privileged `/etc/hosts` changes and failed sudo
  authentication while tests still reported pass. This is environmental noise
  and a test-isolation concern, not evidence of successful host mutation.

## Phase A disposition

No production code was changed during this audit phase. Corrective closure is
not yet complete. The next phase must first capture the exact 409 response,
resolve the canonical baseline, then address P1 gaps with focused regression
tests before rerunning the full gates.
