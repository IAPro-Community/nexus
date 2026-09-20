# Frontend verification report

- Generated: `2026-09-20T13:15:06Z`
- Branch: `feat/capacity-monitor-revival` @ `46d5cbd`
- Verdict: **PASS** (10 pass / 0 fail)
- Dirty web/dist tree: no

## Gates

| Gate | Hard | Status | Duration | Detail |
| --- | --- | --- | --- | --- |
| Prettier (`prettier --check`) | yes | PASS | 7284ms | Checking formatting...<br>All matched files use Prettier code style! |
| TypeScript (`tsc --noEmit`) | yes | PASS | 10042ms | ok |
| ESLint (`eslint src`) | yes | PASS | 6829ms | ok |
| Stylelint (`stylelint "src/**/*.css"`) | yes | PASS | 1587ms | ok |
| Null-safe API array access | yes | PASS | 120ms | sem .length/.map direto em campos nullable conhecidos |
| Vitest (`vitest run`) | yes | PASS | 15732ms | ✓ src/notifications/capacityAlert.test.ts (5 tests) 5ms<br> ✓ src/app/tour/tour.test.ts (5 tests) 8ms<br> ✓ src/design-system/primitives/ContextMenu.test.ts (1 test) 3ms<br> ✓ src/app/projectSelection.test.ts (9 tests) 5ms<br> ✓ src/nexus/terminalUsability.test.ts (6 tests) 33ms<br> ✓ src/app/documentTitle.test.ts (6 tests) 5ms<br> ✓ src/features/work/projectIntelligencePresentation.test.ts (4 tests) 14ms<br> ✓ src/features/work/flowDraftEditorModel.test.ts (5 tests) 15ms<br> ✓ src/app/workspace |
| i18n catalog parity | yes | PASS | 1684ms | RUN  v3.2.7 /projetos/tools/IAPro-Nexus-Workspace-OS-Handoff-2026-08-29/ai-manager/web<br><br> ✓ src/i18n/i18n.test.ts (7 tests) 13ms<br><br> Test Files  1 passed (1)<br>      Tests  7 passed (7)<br>   Start at  13:15:48<br>   Duration  1.07s (transform 489ms, setup 0ms, collect 579ms, tests 13ms, environment 0ms, prepare 163ms) |
| Build + embed (`node scripts/build.mjs`) | yes | PASS | 2613ms | Nexus web build complete: /projetos/tools/IAPro-Nexus-Workspace-OS-Handoff-2026-08-29/ai-manager/web/dist<br>≈ tailwindcss v4.3.3<br><br>Done in 384ms<br><br>  dist/bundle.js                                   354.6kb<br>  dist/chunks/chunk-5YEWBEYR.js                    340.4kb<br>  dist/chunks/chunk-5DVGUEP5.js                    278.7kb<br>  dist/chunks/FlowRunsHistorySurface-HQLH4W5F.js   215.9kb<br>  dist/chunks/chunk-FDQS3BTZ.js                    208.8kb<br>  dist/bundle.css                |
| Embed sync (web/dist ≡ internal/.../embedded) | yes | PASS | 1ms | bundles idênticos (363159 bytes) |
| Critical UI markers in bundle | yes | PASS | 2ms | marcadores críticos presentes (7) |

## Residual risks / next operator steps

- Automated gates green.
- If UI still looks broken in the browser: restart `nexus web` so the new embedded bundle is loaded (`make build`).
- Manual smoke (not automated here): open Project Overview, Radar, one Agent terminal, and a second Project focus switch.

## How to regenerate

```bash
make web-verify
# or
npm --prefix web run verify
```
