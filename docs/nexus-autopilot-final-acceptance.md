# Nexus Autopilot — Final Acceptance Matrix

Audit date: 2026-09-07. This is an adversarial acceptance record. `PASS` means
the behavior was exercised or covered by a deterministic regression test in this
checkout. Documentation or a type/schema alone is not evidence.

| Requirement | Status | Evidence | Test | Notes |
| --- | --- | --- | --- | --- |
| Composer produces structured executable plan | PASS | Composer/plan compiler and work-plan model | Go Nexus suite | Plan is canonical in Nexus/Maestro contract |
| Plan readiness validation | PASS | Deterministic DAG/readiness validator | `plan_readiness_test.go` | Rejects missing DoD/verification and cycles |
| Plan canonical ownership | PASS | Maestro/Nexus ownership matrix | Gap audit | Flow remains a projection/adapter |
| Plan ↔ Flow round-trip | PASS | `flowModel.ts` conversion and existing tests | Frontend Vitest + Go plan tests | Identity/dependencies survive conversion |
| Flow uses mature graph library | PASS | `@xyflow/react` 12.11.6 in `FlowCanvas.tsx` | Typecheck/build | Replaced manual SVG/pointer graph |
| Dependency editing | PASS | React Flow connection callback to canonical mutation | Flow integration tests | Cycle checks remain in mutation boundary |
| Layout versus execution dependency | PASS | Node positions are local visual state; edges are canonical | Flow model tests | Dragging does not mutate dependencies |
| Parallel execution | PASS | Existing MissionRunner scheduler and package dispatch | Go runner tests | Bounded concurrency remains enforced |
| Agent assignment and override | PASS | Existing Nexus assignment strategy and Flow editor | Nexus tests | Real provider execution remains environment-dependent |
| Persistent GO/run creation | PASS | Mission snapshot, durable repository and leases | Runner durable tests | GO is not a one-shot UI action |
| Task convergence | PASS | Verification failure remediation path | Runner verification tests | Agent text is not accepted as proof |
| Global convergence | PASS | `VERIFYING`, global evidence, reopen/remediate/pass | `global_verification_test.go` | Global failure keeps run non-terminal |
| Evidence-based verification | PASS | Command exit codes and persisted verification results | Runner tests | Textual agent claim is insufficient |
| Definition of Done controls DONE | PASS | Completion path requires package/global verification | Go runner suite | Legacy direct runner without global commands is compatibility mode |
| Stagnation strategy change | FAIL | Threshold/configuration exists | No complete behavioral third-cycle proof found | Must prove strategy/agent/replan change in runtime |
| Communication router | FAIL | Routing-related contracts exist | No complete live coordination run found | Memory/artifact-first behavior is not fully demonstrated |
| Shared memory/artifact producer-consumer | FAIL | Stores and receipts exist | No end-to-end producer/consumer acceptance run | Need artifact absence and consumption proof |
| Workspace/worktree isolation under parallel writes | FAIL | Worktree implementation exists | No adversarial two-writer live run | Conflict/merge behavior not proven end to end |
| Restart recovery during active provider run | BLOCKED_EXTERNAL | Durable state tests only | Native/provider process not available in audit host | Requires kill/restart with authenticated runtime |
| Agent process crash handling | BLOCKED_EXTERNAL | Recovery code and unit tests | No real provider process kill | Requires live agent process |
| Provider failure/fallback | BLOCKED_EXTERNAL | Provider/fallback code exists | No authenticated outage/quota injection run | Requires provider credentials or test provider |
| Pause/resume/cancel | PASS | Runner state transitions and cancellation tests | Go runner suite | Live PTY cancellation still needs native E2E |
| Observability/timeline | PASS | Persisted events/run state and Flow run surfaces | Existing Nexus tests + build | Full live overnight timeline not captured |
| Flow execution projection | PASS | Flow receives run/task state models | Typecheck/build and Flow surfaces | Live provider display not proven |
| Keyboard/accessibility regression | PASS | Semantic controls and frontend quality gates | Frontend verify | Full browser keyboard pass remains absent |
| Frontend quality | PASS | Build/typecheck/lint/style/null-array/i18n gates | `bun run verify` | Existing 36 lint warnings remain non-fatal |
| Backend quality | PASS | Tests, race, vet, golangci-lint v2 | `make quality` and Go commands | No lint issues |
| Live browser E2E | FAIL | Existing script is hardening smoke test | Provider-backed Flow E2E not executed | Acceptance scenario is missing |
| Overnight autonomous sandbox | FAIL | Failure-injection unit test only | No unattended full sandbox run | Critical release blocker |
| Real Nexus dogfooding feature | FAIL | No recorded autonomous self-run | Not executed | Critical release blocker |
| Security/policy preservation | PASS | Existing command/workspace policy paths preserved | Go tests/vet/build | No new unrestricted policy was added |

## Verdict

`FINAL_ACCEPTANCE = FAIL`

The control-plane changes are locally buildable and several false-success paths
are covered. The product promise is not yet proven because the critical live
acceptance scenarios — overnight sandbox, provider-backed recovery/crash,
communication/artifact coordination, and dogfooding — were not executed. The
correct recommendation is `NOT_READY` until those tests pass or a real external
blocker is recorded with reproducible evidence.
