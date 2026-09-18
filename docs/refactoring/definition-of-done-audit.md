# Definition of Done audit

Updated: 2026-09-11 (independent evolution corrective closure)

This is an evidence audit of the consolidation campaign, not a release claim.
Green local gates prove the touched tree on Linux; they do not prove native
platform behavior or that every architectural target has been extracted.

| Area | Status | Evidence / remaining work |
| --- | --- | --- |
| Core ownership map | PARTIAL | `core.md`, lifecycle and consumer matrix exist; runtime/process ownership is now registry-injected from application/launcher into SessionHost, and handoff uses an explicit dependency service, while deeper process supervision still spans control and runner. |
| API routes/handlers | PASS for touched domains | Route groups and domain handler files exist; remaining cross-domain handlers are intentionally on the Nexus aggregate. |
| API contracts/errors | PARTIAL | `/api/v1/system/info`, stable `APIError`, `SystemInfoResponse` and `RuntimeDetailResponse` contracts are named and tested without changing wire fields; runtime, event, Mission, resource scheduler, clarification and WorkPlan DTOs are now typed in the Web client, while legacy error inference and a small number of transport responses remain incremental. |
| Web/Desktop client | PASS for current Web transport | Shared `api.ts` transport and typed Nexus facade are covered; CLI/TUI remain direct-Core by design, with TUI registry ownership injectable. |
| CLI compatibility | PASS for audited surface | Help/dispatcher tests and smoke pass; provider-native parsing and full generated command registry remain deferred. |
| Provider registry | PASS for current registry | Segregated interfaces, capabilities and normalized IDs are tested; Nexus application instances and the system doctor can inject/use an owned driver registry across resource discovery, continuity, autonomous execution and intelligence, while provider-specific launch policy still has deliberate adapter switches. |
| Usage/quota evidence | PASS for model contract | Status/source/time/reset plus optional absolute values and confidence are representable; no new scraping was added. |
| Runtime lifecycle | PASS for current application scope | `RunApplicationService` owns MissionRun start/list/control calls, `RuntimeApplicationService` owns raw runtime operations and injects handoff dependencies, launcher-owned registries are injected into SessionHost/attention updates, resource discovery and handoff process waits preserve request cancellation, and the registry exposes tested explicit state transitions. Deeper process supervision remains in control infrastructure. |
| Events/observability | PASS for current runtime scope | Event bus and durable activity persist optional correlation IDs; SessionHost now emits explicit process facts plus `RUNTIME_STARTED`, `RUNTIME_STOPPED` or `RUNTIME_FAILED`, with ownership metadata when supplied. Provider-wide taxonomy remains incremental. |
| Security | PASS for touched scope | Auth/CSRF, filesystem symlink traversal, redaction and security gate have evidence; native platform security remains unverified. |
| Tests/gates | PARTIAL on Linux | Fresh isolated Go tests, vet, lint, security, Web quality (330 tests), `make web-verify` 10/10 and build pass. The full uncached Go race command is long-running in this environment; focused race packages pass. |
| Native matrix | PARTIAL | Local cross-builds verified CLI binaries for Windows amd64 and macOS amd64/arm64, plus `go test -c` for `internal/control/registry` on those targets. Native test/smoke execution and same-SHA CI evidence remain unavailable. |

## Remaining debt

- P0: none identified by the current local consolidation scope.
- P1: complete runtime/process ownership isolation; extend event producers to
  provider/quota transitions where concrete evidence exists; obtain same-SHA
  native CI evidence; complete the evolution E2E scenarios for matcher,
  terminal follow-handoff and failover.
- P2: complete generated CLI registry/contract matrix, typed DTO coverage for
  remaining endpoints, and optional absolute quota producers where providers
  expose them.

## Deliberate correlation boundary

Process-start/exit events do not receive a new MissionRun correlation field in
this slice. Autonomous MissionRun execution currently launches through the
existing agent prompt contract without carrying a run identifier into the
launcher. Adding one would change launcher/registry/host contracts without a
verified caller need. MissionRun lifecycle and handoff events already carry
durable correlation IDs; process correlation remains P1 until the launch
ownership contract is explicitly extended and tested end to end.

## Deliberately deferred

Major visual redesign, new provider integrations, quota scraping, orchestration
engine changes, Maestro redesign, automatic commits/pushes, and native platform
claims remain outside the verified local scope.
