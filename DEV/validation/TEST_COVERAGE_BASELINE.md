# Test coverage baseline

STATUS: CURRENT  
GIT_SHA: `3760df2` + uncommitted hardening  
GENERATED_AT: 2026-09-12  
ENVIRONMENT: linux/amd64 go1.25.0  
EVIDENCE_STREAM_ID: `DEV/validation/current/coverage-nexus.out`

Profile: `go test ./internal/nexus/... -coverprofile=DEV/validation/current/coverage-nexus.out -covermode=atomic`

Global `./internal/nexus/...` statement coverage: **59.6%**

This is the frozen baseline for this campaign. Coverage must not regress on touched critical files. Raw percentage is not the success criterion.

Frontend: Vitest has **no coverage plugin** installed. Do not add one in this campaign. Behavioral Vitest files remain the frontend gate (`make test-frontend`).

| package/module | current (func highlights) | criticality | uncovered critical behavior | target |
|----------------|---------------------------|-------------|-----------------------------|---------|
| intent_router DecideIntent | 85.7% | critical | DecideIntentForProject 70.6% HTTP | ≥85% |
| task_requirements Classify | 100% | critical | none | ≥90% |
| agent_matching MatchAgents | 84.2% | critical | live store pool | ≥85% |
| runtime_routing ResolveTaskModel | 100% | critical | containsModel 0% | ≥85% |
| configuredModelCandidates | 92.9% | critical | Allocate() I/O 0% | ≥90% new |
| skills catalog NewCatalog | 90.9% | critical | ResolveCapability 0%, SourceExternal unused | ≥85% |
| attention_policy ClassifyAttention | 76.9% | critical | some summarizeState branches | ≥85% |
| watchdog ApplyProgressWatchdog | 91.3% | critical | StallTimeout disabled branch | ≥90% new |
| flow FlowFromWorkPlan | 100% | critical | WorkPlanFromFlow 82.1% | ≥85% |
| composer session APIs | mixed 42–100% | high | AddComposerTurnExpected 42.6% | behavioral |
| runner ExecuteNextStep | 71.6% | critical | ListRuns 0% | behavioral transitions |
| validation_evidence recordMission | 72.3% | critical | RecordValidationEvidence adapter 0% | ≥85% |
| mission_executor Allocate/Execute | 0% | I/O | fake executor used in runner tests | behavioral, not % |
| web flowRunModel | behavioral tests PASS | high | no % | no new framework |

Touched in this campaign (must not regress): `intent_router.go`, `mission_executor.go` `configuredModelCandidates`, `runner/watchdog.go`, `runner/attention_policy.go`, `flow.go` via contract tests.
