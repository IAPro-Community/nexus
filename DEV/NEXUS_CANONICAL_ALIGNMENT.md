# Nexus Canonical Alignment Ledger

**Authority:** `docs/superpowers/specs/2026-09-01-nexus-canonical-product-contract.md`
**Functional implementation checkpoint:** `1e6404ac3dc487b78b1439a731a5ad47d16a1933`
**Branch:** `feat/nexus-canonical-review-rebuild`
**Updated:** 2026-09-01

| Canonical capability | Current implementation | Alignment |
|---|---|---|
| Project-first Workspace | Existing Project Rail/Workspace OS | ALIGNED |
| Stable Agent/View identity | Workspace V2 `viewId` + `logicalKey` | ALIGNED |
| Tabs ⇄ Desktop, same Views | Presentation state reuses same logical surfaces | ALIGNED |
| Persistent Agents | Existing AgentID/runtime-generation model | ALIGNED |
| Ask existing Agent | Explicit AgentID submission; Start & Ask preserves identity | ALIGNED |
| Project Shell | Supervised non-AI shell at project canonical cwd | ALIGNED |
| Direct work independent of Composer | Agent/New Session/Shell remain available regardless of readiness | ALIGNED |
| Composer replaces mode-centric UX | Primary Work surface is Composer; legacy labels remain internal compatibility only | ALIGNED |
| Context Readiness 5-state | SQLite/API/frontend/backend gate with drift detection | ALIGNED |
| Bounded/redacted Context Envelope | Intelligence CLI/API receives bounded durable project context | ALIGNED |
| Maestro independent | Existing adapter/catalog/gates; no competing memory/hydration engine | ALIGNED |
| Flow inside Composer | FlowDraftEditor hosts Flow Draft/Canvas/Inspector | ALIGNED |
| WorkPlan compatibility façade | Reversible WorkPlan ⇄ FlowDefinition mapping | ALIGNED |
| Sequence/dependency/parallel DAG | Deterministic order/waves; cycle/missing-dependency rejection | ALIGNED |
| EXISTING / CREATE / AUTO | Explicit Step assignment; existing Scheduler/manual allocation reused | ALIGNED |
| Draft side-effect-free | Browser E2E asserts no `/run` mutation before Approve & Run | ALIGNED |
| ContextCapsule before dispatch | Typed bounded capsule persisted before compile/provider execution | ALIGNED |
| WorkReceipt factual | Typed receipt from real execution/verification fields; no invented artifacts/decisions | ALIGNED |
| Dependency receipt handoff | Direct dependency receipts injected; raw output/transcript excluded | ALIGNED |
| Durable dispatch/no double dispatch | Dispatch intent persisted before provider launch; unknown outcome fails closed | ALIGNED |
| FlowRun Workspace surface | `flow-run:<runId>` façade over durable Mission Run + evidence | ALIGNED |
| Fail-closed FlowRun states | Unknown/internal state never maps to COMPLETED | ALIGNED |
| GUIDED/AUTONOMOUS policy | Flow façade keeps policy separate from provider permissions | ALIGNED |
| Verification + independent review | Existing Mission Runner verification/reviewer path retained | ALIGNED |
| Pause/Take Control/Resume/Cancel | Existing supported runner actions exposed, not simulated | ALIGNED |
| Capability preservation | Current source audit in `NEXUS_CAPABILITY_PRESERVATION.md` | ALIGNED |
| Deterministic browser E2E | Chromium bundle E2E PASS with authenticated/CSRF fixture and fake deterministic data plane | ALIGNED LOCALLY |
| Backend auth transport E2E | Go E2E source validates one-time bootstrap/cookie/CSRF/Origin/WS; local execution blocked by Go/module environment | READY FOR PLATFORM VALIDATION |
| Native Windows/macOS + same-SHA CI | Requires platform execution | EXTERNAL VALIDATION |
| Real-provider E2E | Deliberately separate from deterministic Core E2E | EXTERNAL VALIDATION |
| Release/signing | Not performed | EXTERNAL VALIDATION |

## Product-language reconciliation

The 2026-08-29 planning documents remain architecture history, but the following are no longer authoritative user-facing primary modes:

- DIRECT / ASSISTED / GUIDED / ORCHESTRATED / AUTONOMOUS as navigation modes;
- “Nexus Intelligence” as a competing standalone product brain;
- “Mission” as the primary planning/execution noun in the new UX.

Internal compatibility code may keep these terms where renaming would add migration risk. New product surfaces use **Direct Agent / Project Shell / Composer / Flow / Flow Run**.

## Alignment conclusion

The current implementation remains on the approved product trajectory. The rebuild did not turn Nexus into a generic automation canvas, a full Git client, a second semantic-memory system, or a second execution engine. Remaining work is platform/real-provider/release validation, not a product-concept rewrite.
