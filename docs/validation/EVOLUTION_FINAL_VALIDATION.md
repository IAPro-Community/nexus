# Nexus Final Evolution Validation

## Candidate

- Branch: `feat/nexus-maximum-delivery`
- Base candidate reference: `ab056f558fb2886b21db60ca7951cfb92deb46c7`
- Initial audited SHA: `bf0caad103499450564d670dfaffd302b745c147`
- Final corrected SHA: `bf0caad103499450564d670dfaffd302b745c147` + uncommitted working-tree corrections
- Platform: Ubuntu 25.10, Linux x86_64

## Executive Verdict

**NO-GO**. The local code compiles and the ordinary Go/Web gates are green
when run in isolation, but the three evolution milestones are not all proven
through the required real execution paths. Native Windows/macOS and live
provider continuity remain unverified. The full Go race gate exceeded 300s in
this environment.

## Initial State Found

The independent Phase A report is [`EVOLUTION_FINAL_AUDIT.md`](EVOLUTION_FINAL_AUDIT.md).
It found overstated historical claims, six missing requested specification
documents, no canonical `AgentMatcher` contract, incomplete E2E evidence, and
an operational risk that the running Web process used `/home/desenvolvedor/.local/bin/nexus`
started before the newly built repository binary.

Initial severity count: P0 **0**, P1 **18**, P2 **1** (baseline identity
ambiguity), taken from the frozen Phase A findings before correction.

## Corrections Performed

- Added durable specialty behavior to all Web agent presets: instructions,
  responsibilities, capabilities, domains, strengths, tags, and verification
  policy are now sent as `agent_spec` in the revision config.
- Extended the Go/Web `AgentSpec` contract with domains, strengths, and tags;
  the execution-context formatter includes them.
- Added regression coverage proving every canonical preset has behavioral
  specialization and that compiled context includes the new metadata.
- Added `MatchAgents` as the canonical persistent-Agent matcher with taxonomy
  normalization, hard lifecycle/capability gates, deterministic ranking and
  explainable rejection output; Flow `AUTO` now consumes it.
- Added stable `REQUIRED_RESOURCE_SELECTION` classification to Agent start/ask
  409 responses.
- Separated provider resource gates from Agent specialization matching, so
  persistent Agents remain eligible based on their behavioral AgentSpec.
- Preserved classified roles when a legacy package carries generic
  `implementer`; persisted TaskRequirements now include complexity,
  decomposition, confidence and source metadata.
- Added natural-language `nexus run "<goal>"` orchestration while preserving
  provider-native `nexus run <provider>` syntax.
- Same-provider handoff now preserves Agent/Project identity and reports
  `NATIVE_RESUME_UNVERIFIED` unless provider-level confirmation exists;
  cross-provider handoff is labeled `CONTEXT_HANDOFF`.
- Added deterministic profile-name tie-breaking to ResourceScheduler.
- Added the Phase A audit and regression-diff documents without rewriting the
  historical pre-fix report.

Post-correction evidence count: P0 **0**, remaining P1 **7** (required
integration/native/live evidence), P2 **1** (visual manifest certification
requires a commit SHA). No known production P0 remains.

The reported 409 was a valid resource-selection response when the new Agent's
default provider/profile was unavailable or not authenticated. The UI now
keeps the persistent Agent and routes the user to resource selection instead
of treating that expected conflict as a failed creation. Restart the running
Nexus process from the rebuilt binary before retesting; no process was stopped
automatically.

## Interactive Continuity

**PARTIAL / UNVERIFIED**. Resolver and byte-router unit tests exist and the
implementation supports TTY supervised defaults, explicit direct/supervised,
headless preservation, colon prefixes, and escapes. Required PTY/provider
E2E, contextual completion, and terminal-follow-handoff evidence are absent.

## Agent Routing Truth

**PARTIAL**. Persistent AgentSpec behavior reaches direct prompt compilation,
the canonical matcher covers Flow/Mission AUTO with focused tests, and
`nexus run "<goal>"` is wired through the same domain pipeline. Authenticated
runtime execution and terminal-level provenance remain unproven.

## Autonomous Resource Continuity

**PARTIAL**. ResourceScheduler, quota-aware recommendation, and honest
continuity statuses exist, but automatic failover and same/cross-provider
continuity were not proven with the required runtime/fake-provider E2E flows.

## Test Gates

Fresh execution ledger (2026-09-11, Linux x86_64, audited HEAD
`bf0caad103499450564d670dfaffd302b745c147` plus uncommitted corrections):

| Command | Result |
|---|---|
| `go test -count=1 ./...` | exit 0, PASS, ~65s, all packages |
| `go test -race -count=1 ./...` | exit 124 after 600s, UNVERIFIED; focused critical packages exit 0 |
| `go vet ./...` | exit 0, PASS, ~1.4s |
| `make lint-go` | exit 0, PASS, ~3.6s |
| `make security` | exit 0, PASS, ~9.2s — no vulnerabilities found |
| `make quality` | exit 0, PASS, ~82s |
| `make test-e2e` | exit 0, PASS; terminal/protocol/host/web race scenarios, ~163s |
| `make web-verify` | exit 0, PASS, 10/10, ~23.5s; 65 files / 330 tests |
| `make docs-verify` | exit 2, UNVERIFIED — stale visual manifest cannot certify uncommitted visual files |
| `make build` | exit 0, PASS, ~1.7s |
| focused AgentSpec/matcher/handoff tests | exit 0, PASS |

## Native Platform Matrix

| Linux | Windows | macOS |
|---|---|---|
| Build and ordinary tests PASS | UNVERIFIED — no native runner | UNVERIFIED — no native runner |

## Remaining UNVERIFIED / blockers

- Full race suite completion.
- Real PTY terminal-follow-handoff and writer lease behavior.
- Authenticated Direct/Mission/`nexus run` AgentMatcher-to-provider E2E routing.
- Resource failover safe-point E2E, including cross-provider context handoff.
- Completion matrix and live provider continuity.
- Documentation visual-capture certification for the uncommitted Web change;
  the content/documentation checks themselves remain available, but the
  manifest requires a committed source SHA.
- Native Windows/macOS execution and same-SHA CI.
- Exact user-facing 409 reproduction after restarting the rebuilt binary.

## Final Definition of Done

Interactive Continuity: **PARTIAL**

Agent Routing Truth: **PARTIAL**

Autonomous Resource Continuity: **PARTIAL**

Zero regression: **UNVERIFIED**

Documentation synchronized: **PARTIAL**

Security: **PASS for local gate**

Build: **PASS**

## Final Verdict

**NO-GO**. The implementation is materially improved, but the evidence does
not support promoting the requested completion as fully verified.
