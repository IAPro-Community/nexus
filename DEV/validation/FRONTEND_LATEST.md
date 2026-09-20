# Frontend verification report

- Generated: `2026-09-20T12:32:50Z`
- Branch: `feat/capacity-monitor-revival` @ `cf5e2c7`
- Verdict: **PASS** (10 pass / 0 fail)
- Dirty web/dist tree: no

## Gates

| Gate | Hard | Status | Duration | Detail |
| --- | --- | --- | --- | --- |
| Prettier (`prettier --check`) | yes | PASS | 7982ms | Checking formatting...<br>All matched files use Prettier code style! |
| TypeScript (`tsc --noEmit`) | yes | PASS | 12246ms | ok |
| ESLint (`eslint src`) | yes | PASS | 6748ms | ok |
| Stylelint (`stylelint "src/**/*.css"`) | yes | PASS | 1372ms | ok |
| Null-safe API array access | yes | PASS | 88ms | sem .length/.map direto em campos nullable conhecidos |
| Vitest (`vitest run`) | yes | PASS | 14322ms | ✓ src/features/projects/projectRail.test.ts (4 tests) 6ms<br> ✓ src/nexus/terminalSettings.test.ts (6 tests) 6ms<br> ✓ src/app/tour/tour.test.ts (5 tests) 5ms<br> ✓ src/features/work/flowCanvasInput.test.ts (4 tests) 6ms<br> ✓ src/app/workspaceSurfaceStyles.test.ts (2 tests) 4ms<br> ✓ src/workspace/surfaceAttention.test.ts (6 tests) 4ms<br> ✓ src/app/routerIntegration.test.tsx (3 tests) 8ms<br> ✓ src/keyboard/KeyboardShortcutRegistry.test.ts (4 tests) 4ms<br> ✓ src/workspace/ptyLiveChrome.test.t |
| i18n catalog parity | yes | PASS | 1429ms | RUN  v3.2.7 /projetos/tools/IAPro-Nexus-Workspace-OS-Handoff-2026-08-29/ai-manager/web<br><br> ✓ src/i18n/i18n.test.ts (7 tests) 21ms<br><br> Test Files  1 passed (1)<br>      Tests  7 passed (7)<br>   Start at  12:33:33<br>   Duration  788ms (transform 290ms, setup 0ms, collect 393ms, tests 21ms, environment 0ms, prepare 87ms) |
| Build + embed (`node scripts/build.mjs`) | yes | PASS | 2196ms | Nexus web build complete: /projetos/tools/IAPro-Nexus-Workspace-OS-Handoff-2026-08-29/ai-manager/web/dist<br>≈ tailwindcss v4.3.3<br><br>Done in 330ms<br><br>  dist/bundle.js                                   353.7kb<br>  dist/chunks/chunk-5YEWBEYR.js                    340.4kb<br>  dist/chunks/chunk-5DVGUEP5.js                    278.7kb<br>  dist/chunks/FlowRunsHistorySurface-ETTUGYKX.js   215.8kb<br>  dist/chunks/chunk-FDQS3BTZ.js                    208.8kb<br>  dist/bundle.css                |
| Embed sync (web/dist ≡ internal/.../embedded) | yes | PASS | 0ms | bundles idênticos (362219 bytes) |
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
