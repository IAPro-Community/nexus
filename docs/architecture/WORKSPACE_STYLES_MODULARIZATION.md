# Nexus 1.0 — Workspace CSS modularization and maintainability audit

Date: 2026-10-09. Original release base: `e0bb8ee9b6d55d5a70af36a0ee5298c7ad862ab2`.

## Findings (measured from the release repository tree)

- 61 frontend CSS/SCSS files, ~336 KB total.
- `web/src/app/workspace-os.css`: ~179 KB, ~8,250 lines, ~53% of stylesheet bytes. Global rules, tokens, desktop chrome, terminal chrome, adaptive layouts, notifications, modals and composer shared one cascade.
- Second-largest stylesheet: `features/work/ComposerSurface.module.scss` ~12.8 KB. Third: `features/maestro/MaestroSurface.module.scss` ~12.5 KB. Most feature styles are already SCSS Modules and **do not justify a repository-wide CSS rewrite**.
- Larger **frontend implementation units** are `workspace/WorkspaceRenderer.tsx` (~64 KB), `nexus/AgentTerminal.tsx` (~54 KB), `app/NexusWorkspaceApp.tsx` (~52 KB), `features/settings/SettingsSurface.tsx` (~35 KB), and `nexus/api.ts` (~34 KB). `i18n/resources.ts` (~257 KB) is a localization aggregate and needs its own ownership/build analysis before splitting.
- Larger **Go production units** include `internal/tui/usage_table.go` (~66 KB), `internal/nexus/mission_executor.go` (~58 KB), `internal/app/app.go` (~56 KB), and `internal/nexus/runner/runner.go` (~54 KB). These are analysis targets, **not modified** in this refactor.

File size is a maintainability signal, **not evidence of broken behavior**. Prioritize responsibilities, coupling, testability and risk before moving code.

## What changed in this PR — mechanical, no semantic CSS changes

The complete legacy `workspace-os.css` content was separated **byte-for-byte** into 16 original ordered files under `web/src/app/workspace-styles/`:

1. `01-foundation.css` — global tokens and scheme defaults;
2. `02-shell-and-rail.css` — application shell, topbar and project rail;
3. `03-workspace-layout.css` — workspace layout and stacks;
4. `04-shared-controls.css` — reusable global control rules;
5. `05-terminal-and-panels.css` — terminal and surface chrome;
6. `06-responsive-core.css` — foundational media queries;
7. `07-topbar-and-dialogs.css` — topbar controls and dialogs;
8. `08-welcome-and-modals.css` — welcome and application modal styles;
9. `09-project-management.css` — project manager, desktop lists and browser dialogs;
10. `10-capacity-and-attention.css` — account capacity and attention;
11. `11-notifications-and-composer.css` — attention drawers, notifications and composer;
12. `12-zen-and-flow.css` — Zen/focus controls and flow editor;
13. `13-communication-and-task-ui.css` — notification components and task/status UI;
14. `14-desktop-windows.css` — window chrome, customization and handles;
15. `15-context-and-responsive.css` — container queries, context drawers and adaptive chrome;
16. `16-splash-and-settings.css` — splash and settings chrome.

The labels describe the **dominant ownership** of an ordered legacy slice; some historic selectors remain interleaved. Do not interpret these as CSS Modules or definitive feature ownership. Do not reorder the sections. Their combined contents equal the **original Git blob** of `workspace-os.css`, SHA-1 `877d00b0ec666c4db498682effe4235e298d3166`.

The `workspace-os.css` entrypoint now owns only ordered native CSS imports. Both build entries (`web/src/index.css` and `web/src/index.tsx`) retain their original references to it.

### Regression safety

- Existing source-inspection unit tests for workspace z-index/responsive rules and Flow Canvas scrolling now load the **composed stylesheet**, not just the import facade.
- `npm --prefix web run check:styles` now checks the allowlist **and** the 16-section import order/exact filenames/size ceiling. No new global CSS path is implicitly permitted.
- `cd web && node scripts/check-workspace-styles.mjs --baseline` provides an optional exact original-blob proof of the mechanical extraction.
- `npm --prefix web run lint:styles`, `format:check`, `typecheck`, `test`, `build:css`, `build` and browser E2E must still be executed on this commit. The mechanical parity does **not** prove that the bundler resolves CSS imports identically.

## Recommended follow-up architecture

### P0 — Certify the exact CSS refactor SHA before merge

- Run `npm --prefix web run quality:full`, `npm --prefix web run build:css`, and `npm --prefix web run test:e2e` with an available browser/runner.
- Compare browser rendering for dark, light, high contrast and compact/comfortable on desktop/mobile; include scroll, focus, dialogs, terminal and multistack windows.
- Keep `release/v1.0.0` untouched by this branch until Windows native homologation concludes; certification on older commits does not validate a new release HEAD.

### P1 — Incrementally migrate global selectors to local `*.module.scss`

- First migrate components with clear boundaries and tests: project management, context drawer, splash and settings. Keep tokens, density variables, root theme attributes and true global overlays in foundation.
- After each extraction, delete the old global selectors, assert no duplicate specificity/cascade overrides remain, and run visual/Axe/browser regression.
- Preserve a documented compatibility layer for third-party portals, xterm chrome, window z-index and dynamically applied per-window accents.
- Keep CSS sections under 1,000 lines as an enforceable budget, but **do not** treat the initial 16 sections as the final architecture.

### P1/P2 — Separate follow-up refactors for large React/Go files

- `WorkspaceRenderer.tsx`: separate window/stack layout orchestration from UI rendering, retain drag/focus/surface routing tests.
- `AgentTerminal.tsx`: isolate keyboard/mouse/escape handling, PTY lifecycle and React presentation; **do not rewrite it while Codex/Windows scroll regression is being certified**.
- `NexusWorkspaceApp.tsx`: extract navigation/notification/window management hooks only with behavior tests.
- `SettingsSurface.tsx`: move appearance/settings category components to colocated modules after PR #49 UI settles.
- Go/mission/runtime files: review dependency cycles, exported contracts and branch-specific native regression gates before extracting state machines.

## Release integration and subsequent responsibility extractions

- The modularization from PR #51 was integrated directly into `release/v1.0.0` with Git fast-forward.
- The styling and tokens of superseded PR #49 were reconciled: `17-appearance-presets.css` is the final stylesheet layer, and theme preferences/registry/settings/tests are integrated. The draft PR #49 is **closed without merge** to prevent accidental resurrection of the old monolith.
- `SettingsSurface.tsx`: selection of appearance presets extracted to `ThemeAccordionSelector.tsx` with unchanged WAI-ARIA choices and theme context.
- `WorkspaceRenderer.tsx`: window customization and arrangement menu UI moved to `WorkspaceChromeMenus.tsx`. Custom user titles, AI title permission, accent and icon controls retain the original event handlers.
- `FlowRunSurface.tsx`: pure execution state/status/routing display helpers moved to `flowRunPresentation.ts` and regression cases added.
- `internal/tui/usage_table.go`: account/session filtering, stable availability sorting, quota parsing, provider label and relative time helpers moved unmodified to `usage_table_rows.go`. The 513-line View, table and modal rendering block was moved unmodified to `usage_table_view.go` (rendering only). Terminal input, transport, state transition and quota semantics are untouched.
- Larger stateful and risk-heavy files (`AgentTerminal.tsx`, `mission_executor.go`, `NexusWorkspaceApp.tsx`) remain **unrefactored** until per-boundary tests and native certification exist. File size alone is not authorization to rewrite their execution paths.

### Required certification on the final release SHA

Run `cd web && npm run quality:full && npm run test:e2e:composer-themes`; run `go test ./internal/tui/...` (with the Go version required by go.mod), then focused Windows WebView2/ConPTY and terminal wheel/arrow/ESC tests. Static byte parity and mechanical extraction are not substitutes for these runtime gates.

## Status

- Complete: mechanical separation into 16 original sections plus final appearance section, guarded import manifest, adapted tests, theme integration and targeted responsibilities extracted.
- Static gate: exact legacy CSS source parity proven during extraction; CI/build/runtime validation still pending.
- Not yet complete: full Bun/Playwright/WebView2 build and visual certification.
- Unchanged execution paths: Windows Go runtime, Codex PTY/ConPTY behavior, routing, tunnel, signing, mobile and main; the release branch was updated directly.

## Adendo: rolagem Web após modularização — 22/10/2026

Ao contrário da extração mecânica documentada acima, a alteração posterior
em `web/src/app/workspace-styles/03-workspace-layout.css` **é uma mudança
semântica deliberada**: o browser permite rolagem dentro do workspace e do
painel ativo quando a altura do conteúdo excede o espaço disponível.

`web/src/index.tsx` registra a plataforma em
`document.documentElement.dataset.nexusPlatform`; a regra CSS é restrita
a `data-nexus-platform="web"` para preservar a composição do Desktop.
O teste `web/src/app/workspaceSurfaceStyles.test.ts` verifica a presença das
regras e `web/scripts/e2e-surface-isolation.mjs` contém uma prova de
scroll na viewport mobile/laptop. **Esses testes não foram certificados no
SHA da intervenção**; recaptura com zoom, navegação e conteúdo real ainda é
gate de liberação. Histórico visual do manifesto anterior é referencial.
