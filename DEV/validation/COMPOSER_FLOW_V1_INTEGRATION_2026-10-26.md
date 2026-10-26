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
