# Critical Behavior Matrix

STATUS: CURRENT  
GIT_SHA: `3760df2d5b3c6656c9c32d9cfc5d922d352c2644+hardening`  
GENERATED_AT: 2026-09-12  
ENVIRONMENT: linux/amd64  
EVIDENCE_STREAM_ID: local-hardening-behavior-matrix

| Capability | Scenario | Unit | Integration | E2E | Fault injection | Evidence | Status |
|------------|----------|------|-------------|-----|-----------------|----------|--------|
| Skills | builtin/project/user/Maestro optional | skills/catalog_test.go | skill_catalog_test.go | UNVERIFIED HTTP | source failure in catalog_test | local | PARTIAL |
| Intent | DIRECT C1 | intent_router_test.go | composer_flow_contract_test.go | UNVERIFIED browser | n/a | local | PASS |
| Intent | C2 PI facts | intent_router_test.go | composer_flow_contract_test.go | UNVERIFIED | n/a | local | PASS |
| Intent | C3 complete spec | intent_router_test.go | n/a | UNVERIFIED | n/a | local | PASS |
| Intent | C4 conflict | intent_router_test.go | n/a | UNVERIFIED | n/a | local | PASS |
| Routing | AUTO cheap vs leftover model | runtime_routing_test.go | mission_executor.go | UNVERIFIED live | n/a | local | PASS |
| Routing | PREFER/PIN | runtime_routing_test.go | runner_durable_test.go | UNVERIFIED live | n/a | local | PASS |
| Mission | success/retry/remediation | runner_test.go | runner_durable_test.go | overnight sandbox | overnightAcceptanceExecutor | local | PASS |
| Mission | stall watchdog | watchdog_test.go | ExecuteNextStep | UNVERIFIED 8h | stall clock | local | PASS |
| Handoff | account/context | control/handoff | UNVERIFIED mission | UNVERIFIED live | n/a | local | PARTIAL |
| Attention | A needs user, B running, C quota silent, D completed, E FAILED_NO_PROGRESS | attention_e2e_test.go | make attention-e2e | UNVERIFIED browser | store reopen | local | PASS |
| Attention | restart dedup | fault_injection_test.go | make attention-e2e | UNVERIFIED | JSON reopen | local | PASS |
| Composer/Flow | optional pipeline | composer_flow_contract_test.go | flow_test.go | UNVERIFIED browser | n/a | local | PASS |
| Overnight | smoke sandbox | runner_durable_test.go | make overnight-smoke | soak opt-in | injected provider fail | local | PASS |
| Overnight | 8h soak | harness exists | make overnight-soak | UNVERIFIED this host | n/a | missing | UNVERIFIED |
| Evidence | hash chain | validation_evidence_test.go | autopilot_contract_test.go | UNVERIFIED auth mission | tamper tests | local | PASS |
| Providers | Codex/OpenCode/AGY live | adapters | UNVERIFIED | UNVERIFIED | n/a | missing | UNVERIFIED |
| Platforms | Windows/macOS native | CI jobs | UNVERIFIED this SHA | UNVERIFIED | n/a | missing | UNVERIFIED |
