# PR #58 — contextual workspace verification (2026-10-22)

## Scope and evidence separation

Branch: `feat/v1-contextual-workspace-experience`. PR: https://github.com/IAPro-Community/nexus/pull/58.
Feature areas: persistent terminal Split, contextual command search, Direct Session account selection and ResourceScheduler integration.

**Current verdict: NOT CERTIFIED / DO NOT MERGE.** The evidence below is limited to isolated reproductions. It does not certify the full current Git SHA, the production web bundle, PTY state transitions or native desktop compatibility.

### Isolated checks performed

- **PASS (isolated):** Node 22 + TypeScript `tsc --strict` against a stand-alone copy of command registry logic; 10 search scenarios covering scoped commands, fuzzy spelling, missing results and exact-result priority.
- **PASS (isolated):** Node 22 + TypeScript `tsc --strict` against a stand-alone copy of Direct Session lifecycle logic; 6 cases covering successful creation, create failure, allocation failure, configuration failure, failed rollback and uncertain start.
- **PASS (isolated):** Chromium 144 real layout engine with relevant declarations from `03-workspace-layout.css` and `TerminalSplit.module.scss`; 7 geometry combinations: side-by-side 20/50/80%, stacked 20/50/80%, and stacked compact viewport (360 × 640). Two live pane rectangles remain non-overlapping, a hidden third pane does not occupy grid space, and legacy `position:absolute` is overridden.
- These copies were reproduced in an offline container (GitHub clone blocked by DNS); **do not mislabel them as full-repository test executions**.

### CI integration

The existing `browser` CI job now executes `bun run test:e2e:terminal-split` after the surface-isolation suite and checks the step outcome in its final aggregate gate. The repository script now covers 7 ratios/orientations. A pending self-hosted runner job is *not* a PASS.

### Still required on the exact candidate SHA

1. `cd web && bun install --frozen-lockfile && bun run quality:full`
2. `cd web && bun run test:e2e:terminal-split && bun run test:e2e:surface && bun run test:e2e`
3. `go vet ./... && go test -count=1 ./...` using the exact Go toolchain defined by this release; include races where supported
4. Real terminal interaction: open two PTYs, split/stack, resize (verify cols/rows upstream), change focus, close one PTY, reconnect one PTY, switch back to Tabs/Desktop/Mosaic and restore persisted project layout
5. Native Windows ConPTY, macOS PTY and Linux PTY/HUD matrix, installed desktop not only browser; preserve native signing and updater gates
6. Inspect CI failures and real browser screenshots; only mark READY after all required gates pass at the **final SHA**

### Scope not silently claimed complete

The Direct Session model override is manual. A trusted per-provider/per-profile model catalog and model specialty/risk classification are not yet connected to the Direct launcher. Do not invent models or claim model auto-routing is certified. Composer and Flow feature status is unchanged.

The GitHub release branch remains protected from this PR: no merge, tag, installer or update manifest was published by these checks.
