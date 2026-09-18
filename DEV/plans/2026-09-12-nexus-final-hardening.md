# Nexus Final Hardening Implementation Plan

> **For agentic workers:** execute task-by-task with TDD. Do not reopen architecture.

**Goal:** Prove and harden the existing Nexus destination architecture so ordinary, unattended, restart, and CI usage cannot silently regress ownership, Attention, Composer/Flow optionality, evidence, or overnight bounds.

**Architecture:** Keep Nexus as operational intelligence and Maestro as optional methodology. WorkPlan remains execution truth. Attention and notifications remain projections. Harden PARTIAL findings; do not redesign PASS systems.

**Tech Stack:** Go 1.25, Vitest, Makefile, SQLite store, in-memory runner tests.

## Global Constraints

- No new architecture generation.
- No new JS coverage framework (`vitest` has no coverage plugin installed; Go coverage only).
- Do not fake provider/platform/8h evidence as PASS.
- Conventional commits only if the operator requests a commit.
- TDD: RED → correct failure → GREEN → refactor.

### Task 1: WorkUnit model pool (no leftover AgentConfig.Model)

**Files:** `internal/nexus/runtime_routing_test.go`, `internal/nexus/mission_executor.go`

- [ ] Failing test: `configuredModelCandidates` with only `AgentConfig.Model` returns empty AUTO pool
- [ ] Remove singleton mint from `cfg.Model`
- [ ] AUTO with empty inventory must not retain leftover `current.Model`

### Task 2: Progress watchdog

**Files:** `internal/nexus/runner/watchdog.go`, `watchdog_test.go`, `types.go`, `contract.go`, `runner.go`

- [ ] Stall in EXECUTING without LastProgressAt advance → FAILED_NO_PROGRESS, not NEEDS_YOU
- [ ] Default StallTimeoutSeconds = 900

### Task 3: Composer/Flow anti-regression + C1–C4

**Files:** `internal/nexus/intent_router.go`, `intent_router_test.go`, `composer_flow_contract_test.go`

- [ ] C1 DIRECT no Composer required
- [ ] C2 Project Intelligence covers test framework
- [ ] C3 complete prompt → PLAN, 0 questions
- [ ] C4 conflicting must/must-not → CLARIFY one question
- [ ] Flow round-trip remains semantic identity

### Task 4: Attention restart + Make attention-e2e

**Files:** `internal/nexus/runner/attention_e2e_test.go`, `Makefile`

### Task 5: Fault injection + overnight harness

**Files:** `internal/nexus/runner/fault_injection_test.go`, `overnight_soak_test.go`, `Makefile`

### Task 6: Coverage + canonical Make surface

**Files:** `Makefile`, `DEV/validation/TEST_COVERAGE_BASELINE.md`

### Task 7: Documentation current vs historical

**Files:** `DEV/validation/current/RELEASE_STATUS.md`, `DEV/DOCUMENTATION_INVENTORY.md`, `DEV/INDEX.md`, banners on stale FINAL_* reports

### Task 8: Frontend FAILED_NO_PROGRESS mapping + contract JSON

**Files:** `web/src/features/work/flowRunModel.test.ts`

### Task 9: Adversarial review + gates + FINAL_HARDENING_REPORT
