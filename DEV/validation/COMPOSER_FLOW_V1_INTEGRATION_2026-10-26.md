# Nexus v1.0.0 — Composer + Flow integration gate (2026-10-26)

**Source branch:** `release/v1.0.0`  
**Integration state:** SOURCE_CHANGES_COMMITTED  
**Release certification:** **NO_GO — tests and platform evidence not available**  

Do not publish or call the Composer/Flow integration stable based on source
commits alone. This record is an integration inventory and executable
homologation recipe, **not** test evidence.

## Confirmed source behaviors

- Composer is the beginner-friendly AI-enabled discovery assistant. Mutations
  are guarded by Nexus Intelligence; Create checks a real provider round trip.
- When provider unavailable, saved Composer sessions and compiled artifacts
  remain readable in a dedicated read-only view.
- Composer recommends Agent vs Flow based on observed risk, dependencies,
  concurrency and durable-execution requirements. The recommendation never
  starts execution.
- `PromptArtifact` materialization freezes `composer.handoff.v1`, with
  `session_id`, `artifact_id`, version/hash, user-approved answers,
  `PROPOSED` recommendations, assumptions and unresolved `BLOCKING` choices.
- AI-INFERRED values remain labeled in executable prompts; only CONFIRMED
  facts can be promoted into first-class WorkPlan facts.
- Flow uses its existing `@xyflow/react` editor and `MissionRunner`, with
  an informational handoff panel on draft and run surfaces.
- Preview, advisory and strict preflight plus both Mission admission paths
  fail closed on unresolved handoff decisions.
- `PlanApplicationService.Update` refuses to erase/change the Composer
  handoff and immutable lineage during Flow revisions. Replanning produces a
  new Composer elaboration and fresh WorkPlan.
- The user-initiated Mission review endpoint creates a separate Composer
  session, identifies the original Mission and attempts to read its historical
  WorkPlan revision. It does not modify the active run/snapshot.
- Maestro remains optional; no new scheduler/runner was introduced.

## Required local checks

From the repository root, on the exact branch/commit under evaluation:

```bash
git status --short
git rev-parse HEAD
go test ./internal/nexus/... ./internal/control/web/...
cd web
bun install --frozen-lockfile
bun run typecheck
bun run lint
bun run test
bun run build
bun run test:e2e:composer
bun run test:e2e:composer-spine
bun run test:e2e:composer-vague-spine
```

The three browser E2E scenarios use a loopback OpenAI-compatible fixture and
an isolated Nexus configuration directory. They must not use live credentials.
Run browser tests on Linux first, and repeat critical GUI smoke checks on
Windows/macOS before declaring cross-platform support.

## Manual acceptance matrix

| Case | Required outcome |
| --- | --- |
| Intelligence OFF / invalid CLI | Composer creation blocked; no persisted orphan session |
| Intelligence unavailable after prior planning | History and immutable artifacts readable |
| Novice, vague idea | Useful consultant summary and options; no ritual checklist |
| Existing project | Only grounded project facts claimed as verified |
| Material ambiguity | User-readable justification; only truly blocking choices gate |
| Composer → Flow | Frozen artifact hash/version and decision provenance preserved |
| Flow edit | Source contract and blockers cannot be removed or rewritten |
| Preflight / GO | Outstanding BLOCKING decision prevents Mission start |
| Simple work | Agent route suggested; Flow optional |
| Complex work | Flow recommendation explained without auto-run |
| Runtime discovery | User can request new Composer elaboration; active snapshot unchanged |
| Stale plan revision | Approval token rejected by Mission admission |
| Restart, failure, resource shortage | Existing MissionRunner policies and evidence preserved |

## Known limitations / remaining sign-off

- No Go test output, TypeScript/Vitest/build output, or Playwright screenshots
  were collected by this integration pass. GitHub returned no checks/workflow
  runs for the current commits.
- The full aspirational product experience (automated multi-perspective product
  discovery, universal prototyping, benchmarked solution scoring, complete
  cross-platform GUI certification) has **not** been certified by this work.
- A live supported Nexus Intelligence provider must be tested in a real
  installation in addition to the fake provider E2E.
- Review release truth, CSS/accessibility and legacy E2E expectations before
  moving the beta release gate to PASS.

## Cleanup regression addendum — Composer + Flow deletion

The UI now exposes `Excluir elaboração` on the Composer header and saved
sessions (including Intelligence-OFF history), and `Excluir Flow` in the
Flow Studio draft gallery and the active draft editor.

The DELETE endpoints require `expected_revision` and check foreign-key
dependencies **before** deleting. They are intentionally not delete-cascade
operations for Mission history.

| Validation | Expected outcome |
| --- | --- |
| Delete an unused Composer session | Confirmation, 200, session/turn/artifact/variant records removed |
| Delete session with a linked Flow | 409; Composer and Flow remain intact |
| Delete session with recorded delivery | 409; delivery provenance remains intact |
| Delete session with stale revision | 409; no rows removed |
| Delete session with Intelligence OFF | Cleanup allowed; new AI elaboration still blocked |
| Delete unused Flow from gallery | 200; list refreshes, revision records removed |
| Delete unused Flow from editor | 200; navigate back to Flow Studio |
| Delete Flow with Mission runs/snapshots | 409; execution/evidence preserved |
| Delete Flow with schedules | 409; schedule retained |
| Delete Flow with legacy mission link | 409; legacy execution preserved |
| Delete Flow with stale revision | 409; current draft remains |
| User cancels confirmation | No DELETE request sent |
| Windows/Linux/macOS UI | Visible, keyboard-accessible controls; no accidental card-open |
| Mobile Flow list | Button remains reachable, not nested in open-card button |

Added tests:
- `internal/nexus/store/planning_deletion_test.go`
- `web/src/nexus/api.test.ts`

Required local commands:
```bash
go test ./internal/nexus/store ./internal/nexus ./internal/control/web
cd web && npm ci && npm run test -- --run src/nexus/api.test.ts
```

**Certification remains NO_GO until these commands and GUI checks pass.**
