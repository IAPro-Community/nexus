# Frontend verification report

- Generated: `2026-09-20T12:58:08Z`
- Branch: `feat/capacity-monitor-revival` @ `8cbe8a9`
- Verdict: **PASS** (10 pass / 0 fail)
- Dirty web/dist tree: **yes**

## Gates

| Gate | Hard | Status | Duration | Detail |
| --- | --- | --- | --- | --- |
| Prettier (`prettier --check`) | yes | PASS | 7806ms | Checking formatting...<br>All matched files use Prettier code style! |
| TypeScript (`tsc --noEmit`) | yes | PASS | 10748ms | ok |
| ESLint (`eslint src`) | yes | PASS | 5975ms | ok |
| Stylelint (`stylelint "src/**/*.css"`) | yes | PASS | 1298ms | ok |
| Null-safe API array access | yes | PASS | 98ms | sem .length/.map direto em campos nullable conhecidos |
| Vitest (`vitest run`) | yes | PASS | 12940ms | ✓ src/notifications/capacityAlert.test.ts (5 tests) 6ms<br> ✓ src/app/projectSelection.test.ts (9 tests) 6ms<br> ✓ src/app/activityModel.test.ts (1 test) 4ms<br> ✓ src/features/projects/projectDirectoryPicker.test.ts (2 tests) 8ms<br> ✓ src/design-system/primitives/ContextMenu.test.ts (1 test) 6ms<br> ✓ src/features/work/flowDraftEditorModel.test.ts (5 tests) 5ms<br> ✓ src/nexus/terminalSettings.test.ts (6 tests) 10ms<br> ✓ src/app/workspaceMissionRoute.test.ts (3 tests) 3ms<br> ✓ src/lib/safeAr |
| i18n catalog parity | yes | PASS | 1109ms | RUN  v3.2.7 /projetos/tools/IAPro-Nexus-Workspace-OS-Handoff-2026-08-29/ai-manager/web<br><br> ✓ src/i18n/i18n.test.ts (7 tests) 15ms<br><br> Test Files  1 passed (1)<br>      Tests  7 passed (7)<br>   Start at  12:58:48<br>   Duration  642ms (transform 245ms, setup 0ms, collect 332ms, tests 15ms, environment 0ms, prepare 82ms) |
| Build + embed (`node scripts/build.mjs`) | yes | PASS | 1781ms | Nexus web build complete: /projetos/tools/IAPro-Nexus-Workspace-OS-Handoff-2026-08-29/ai-manager/web/dist<br>≈ tailwindcss v4.3.3<br><br>Done in 273ms<br><br>  dist/bundle.js                                   354.6kb<br>  dist/chunks/chunk-5YEWBEYR.js                    340.4kb<br>  dist/chunks/chunk-5DVGUEP5.js                    278.7kb<br>  dist/chunks/FlowRunsHistorySurface-HQLH4W5F.js   215.9kb<br>  dist/chunks/chunk-FDQS3BTZ.js                    208.8kb<br>  dist/bundle.css                |
| Embed sync (web/dist ≡ internal/.../embedded) | yes | PASS | 1ms | bundles idênticos (363159 bytes) |
| Critical UI markers in bundle | yes | PASS | 3ms | marcadores críticos presentes (7) |

## Residual risks / next operator steps

- Automated gates green.
- If UI still looks broken in the browser: restart `nexus web` so the new embedded bundle is loaded (`make build`).
- Manual smoke (not automated here): open Project Overview, Radar, one Agent terminal, and a second Project focus switch.

### Dirty paths

```
M web/package.json
 M web/scripts/e2e-hardening-verify.mjs
 M web/src/app/NexusShell.tsx
 M web/src/app/workspace-os.css
 M web/src/features/work/FlowRunsHistorySurface.tsx
 M web/src/workspace/WorkspaceRenderer.tsx
```

## How to regenerate

```bash
make web-verify
# or
npm --prefix web run verify
```
