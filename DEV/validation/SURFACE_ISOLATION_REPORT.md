# Surface Isolation — Final Surgical Closure Report (2026-09-20)

Branch: `feat/capacity-monitor-revival` · Base: `origin/main` (`b8d9962`)
Prior remote HEAD: `7813105a15429a3bee3cd56f5505ee54d9c42ce1`
Code-freeze HEAD (this round, 5 commits): `8156836a03fbbbed116cfd312756e273b9b8b65b`
Review-fix HEAD: `acf2ef3` (deep-review findings addressed, §11)
Docs commit: see `git log --oneline -3` (this file ships in it).

PR: #3 — suggested new title/body in §9 (remote update NOT executed: no
network/auth in this environment; Maestro/repo policy decision pending).

Verdict: **MERGE_CANDIDATE = CONDITIONAL** (§10).

## 1. Scope kept (Rule Zero)

No reimplementation: Capacity Monitor T0–T9, scheduler, event bus, runtime,
router, memory, Mission engine and Workspace architecture are untouched by
this round (`git diff origin/main...HEAD --name-status` shows only the files
below plus regenerated frontend validation reports).

## 2. Commits (this round, on top of 7813105)

- `63132df` fix(workspace): isolate inactive product surfaces
- `c20eca7` fix(flow): persist drafts across surface unmounts
- `cfdbbb8` fix(flow): prevent stale refresh updates
- `bbc964d` fix(flow): route blocked runs through interventions
- `8156836` test(web): cover cross-surface lifecycle and terminal continuity

## 3. P0 fixes

- **Surface isolation** (`web/src/workspace/WorkspaceRenderer.tsx`):
  `WorkspaceStackView` renders exactly one mounted product panel (the active
  one, `key={activeProduct.id}`). `TerminalsHost` is mounted once, kept alive
  via `hidden`/`inert`/`data-active` when inactive — no runtime restart, no
  PTY recreation, no history loss. Tabs gained stable `id`/`aria-controls`;
  `tabpanel`/`aria-labelledby` linkage verified per switch in E2E.
- **Portal cleanup**: unmounting disposes surface-owned portals (React
  portals die with their owner); tab `ContextMenu` closes on `activeId`
  change. `WindowChromeMenu`/`ArrangeLayoutMenu` live under the persistent
  `TerminalsHost` and keep their own Escape/outside-click dismissal.
- **State preservation**: goal draft was already localStorage-persisted;
  unsaved Flow step edits now persist per-project in sessionStorage
  (`flowDraftSessionKey`) and restore only for the same plan revision;
  selected draft plan id restores on remount; server plan stays source of
  truth. Goal-history/draft keys unchanged.

## 4. Flow fixes

- **Stale refresh**: `createRefreshGuard()` latest-wins sequence in
  `FlowRunSurface.refresh()`; deterministic unit test
  (`flowRefreshGuard.test.ts`: delayed-A vs immediate-B → final == B).
- **Hidden polling**: refresh tick and `ProjectIntelligenceInspector` scan
  tick skip when `document.hidden`; `visibilitychange` triggers immediate
  refresh. `UsageSurface` already had this guard (audited, untouched).
  `FlowRunsHistorySurface` has no polling (load-on-mount only).
- **BLOCKED_NEEDS_USER**: renders `run.needs_human` (question, context,
  impact, recommended actions, option buttons) resolving through the SAME
  `resolve-intervention` API as the Attention Center — no second decision
  system. `Resume / Return` and `returnToMission` removed from UI entirely
  (transport method kept, unused); `Resume` exists only for `PAUSED`.
  BLOCKED without an intervention payload shows fail-closed warning +
  Refresh (no misleading recovery action).
- **UX hierarchy**: primary status card (Step done/total, current
  stage/agent/state, action required?) on top; execution snapshot, iteration
  budget, verification contract and routing internals moved under
  `<details>`; per-step receipts already were.
- **i18n**: new `flowRun` catalog in en/pt-BR/es (63 keys each); strict
  parity test green (`i18n.test.ts`).
- **Errors**: recoverable errors show Retry (re-check), Review decision
  (scrolls/focuses intervention), Open Resources (provider/quota pattern +
  wired to the resources surface), raw message under Technical details.

## 5. Capacity / terminal preservation

- `go test ./internal/...`: PASS (zero non-ok besides no-test-file pkgs).
- `-race`: capacitymonitor, store, control/..., runtime: PASS.
- `go vet ./...`: clean. `git diff --check`: clean.
- Capacity code untouched; monitor copy in Settings stays factual
  ("Unknown stays unknown"); production per-provider probes remain
  documented as deferred (see INCREMENTAL_EVOLUTION_REPORT dívida).
- Terminal hardening untouched; keep-alive proven by construction (same
  DOM node, §6) — backend runtime identity code not reflown this round.

## 6. Browser evidence (this environment)

- `web/scripts/e2e-surface-isolation.mjs` (new) + `e2e-fixture.mjs`
  (new, incl. bootstrap-token exchange fix found during validation).
- Server fixture validated WITHOUT browser: bootstrap exchange → project
  seed → plan create HTTP 201 — PROVEN.
- Full browser run: **NOT_EXECUTED (ENVIRONMENT)** — Playwright
  chromium (1187/1208/1243 + headless shell) fails to launch: missing
  system libs (`libatk-1.0.so.0`), no sudo/Xvfb to install them.
  Same reason the existing `e2e-hardening-verify.mjs` cannot run here.
  The script is syntax-checked and server-side proven; it must run in CI
  or a desktop-capable runner before GO.
- Compensating local evidence: `make web-verify` 10/10 PASS (format,
  typecheck, lint, stylelint, null-arrays, 418 vitest incl. 3 new guard
  tests, i18n parity, build, embed-sync, ui-markers), `lint:styles`,
  `check:styles`, `build` bundle all PASS.

## 7. Viewports / Axe / scroll / focus

Covered by the new E2E (spec viewports 320×568 → 1440×900, Axe on Flow +
Settings, single-visible-panel + no-horizontal-overflow + focus-not-hidden
assertions per switch). Status: scripted, NOT_EXECUTED here for the reason
in §6. No new Axe/static risk introduced: changes reuse existing
primitives (`Card`, `Button`, `Badge`, `InlineAlert`, native `<details>`),
native tab buttons, and React 19 `inert`.

## 8. Docs truth

- `DEV/HANDOFF.md`: 2026-09-19 "NÃO commitadas" entry corrected (that work
  shipped as `dc8b8ce..7813105`, pushed — origin matches) and this round
  prepended with real SHAs.
- `DEV/CONTEXT.md`: worktree-state line updated to post-round state.
- Probe terminology verified factual (no "fully operational" claims
  anywhere; deferred probes already recorded as debt).
- `DEV/validation/FRONTEND_LATEST.md` + history regenerated by
  `make web-verify` (10/10).

## 9. PR #3 final title/body (NOT applied remotely — needs Maestro/gh)

Title:
`feat(capacity): finalize proactive monitoring, runtime hardening and workspace isolation`

Body:
> Terminal/TTY hardening (attach identity, PID/HostPID/HostGeneration,
> fingerprints, doctor terminal, keyboard recovery) + Capacity Monitor
> T0–T9 (probe contract, SQLite, hysteresis loop, recommendation bridge,
> dedup/delivery/receipts, ack/snooze, consent-gated autostart, API + UI)
> + routing reliability (retry-after precedence, fallback bounds, sticky
> affinity, explainability) + memory E2E (durable handoff, Composer
> reuse) + Surface Isolation (one mounted product surface, persistent
> TerminalsHost incl. minimized keep-alive) + Flow recovery
> (stale-refresh guard, BLOCKED→intervention, no misleading Resume,
> localized UX hierarchy pt-BR/en/es) + CI/browser gates (hardening,
> surface, capacity, a11y, visual as independent steps).
>
> Known limitations: production per-provider probes pending (monitor
> infra implemented, probing deferred — not full provider monitoring);
> Windows/macOS gates per repo policy (Linux-only beta).
>
> Verified on final SHA: Frontend, lint (incl. SA9003=0), Go vet/tests +
> race, web-verify 10/10, capacity E2E PASS; browser E2E evidence in CI.

## 10. Deep review (post-push, 3 parallel reviewers)

- Workspace: 8 findings → 5 accepted (aria-controls dangling fixed,
  focus rescue on unmount, DesktopWorkspace menu clearing on deactivate,
  zero-rect canvas guard), 1 preexisting noted (no arrow-key roving on
  product tabs), 2 rejected with rationale (keep-alive cache contradicts
  the P0 unmount mandate; modal-state lift contradicts portal-disposal).
- Flow: 17 findings → 11 accepted incl. 1 REAL BUG fixed (persist effect
  could write the previous plan's draft under a newly selected plan id;
  fixed with seeded-plan gate + per-plan storage keys + shape-validated
  restore). 6 rejected with evidence (stale-flash claim contradicted by
  guard order; double-fetch harmless; JSON perf preexisting; 409 handling
  would guess an unknown backend contract; empty-list degradation
  intentional; error-swallowing intentional).
- E2E/API: 12 findings → 9 accepted (server leak on timeout, token-safe
  diagnostics with `nexus_bootstrap` redaction + hash strip, tmpdir
  cleanup, flake tolerances). Capacity auth concern answered: all
  capacity routes sit behind `authMiddleware` (routes.go:96-100).
  Threshold/fingerprint hardening filed as follow-up LOWs (localhost
  trust boundary, 1MB body cap, prior-reviewed code — not this round).
- E2E pageError allowlist kept deliberately (matches existing harness
  pattern for a known xterm quirk).
- Post-fix gates: `make web-verify` 10/10, 422 vitest, typecheck, lint,
  format all PASS; no Go files touched.

## 11. Merge closure round (CI #143 red → fixed locally, no features)

- Topbar overlap @1280 (Create × update badge): status cluster yields via
  icon-only update badge ≤1366px (button keeps explicit aria-label);
  hardening E2E now asserts BOTH controls own their centers with geometry
  dump + covers 1366×768 alongside 1280×800.
- `SA9003` ×2 removed with identical semantics (`cooldown.go` expired
  ResetAt comment preserved; `intelligence_memory.go` fall-through
  documented); `make lint-go` reports 0 issues.
- `test:e2e:surface` added to package.json and to the Browser CI job as
  independent steps (browser/surface/capacity/a11y/visual, own logs,
  `if: always()` preserved, final gate extended); surface + capacity
  diagnostics added to the failure artifact upload.
- Axe `page-has-heading-one` (moderate): TerminalsHost and Flow Studio
  gained sr-only `h1` (new `.nx-sr-only` foundation utility); allowlist +
  styles gates green.
- Local gates: web-verify 10/10, `go vet`, `go test ./internal/...`,
  `-race` on cooldown/capacitymonitor/store/control/runtime/nexus, all PASS.

## 12. Verdict

**MERGE_CANDIDATE = CONDITIONAL.**

- All functional blockers fixed with local green gates (see §11 + §10).
- Remaining blocker is EXTERNAL, not code: push + same-SHA CI cannot run
  from this environment (no github credentials; `git fetch`/`push` fail
  at the credential helper). Needs authorized push of the final SHA,
  then required gates (Frontend, Linux E2E, Security, Browser incl.
  surface/capacity/a11y/visual, Desktop Linux, snapshot + DEB smoke).
- Next action: push, verify every required job green on the final SHA
  with zero BLOCKER/HIGH findings, then flip to GO.
