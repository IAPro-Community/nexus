# Frontend verification report

- Generated: `2026-09-20T12:36:23Z`
- Branch: `feat/capacity-monitor-revival` @ `68f7aff`
- Verdict: **PASS** (10 pass / 0 fail)
- Dirty web/dist tree: **yes**

## Gates

| Gate | Hard | Status | Duration | Detail |
| --- | --- | --- | --- | --- |
| Prettier (`prettier --check`) | yes | PASS | 5714ms | Checking formatting...<br>All matched files use Prettier code style! |
| TypeScript (`tsc --noEmit`) | yes | PASS | 9216ms | ok |
| ESLint (`eslint src`) | yes | PASS | 6158ms | ok |
| Stylelint (`stylelint "src/**/*.css"`) | yes | PASS | 1157ms | ok |
| Null-safe API array access | yes | PASS | 90ms | sem .length/.map direto em campos nullable conhecidos |
| Vitest (`vitest run`) | yes | PASS | 10221ms | ✓ src/workspace/ptyLiveChrome.test.ts (4 tests) 6ms<br> ✓ src/features/work/flowCanvasInput.test.ts (4 tests) 3ms<br> ✓ src/features/overview/overviewRecover.test.ts (5 tests) 4ms<br> ✓ src/features/agents/askAgentModel.test.ts (2 tests) 6ms<br> ✓ src/app/surfaces.test.ts (7 tests) 7ms<br> ✓ src/features/work/flowRunModel.test.ts (3 tests) 7ms<br> ✓ src/features/work/flowDraftEditorModel.test.ts (5 tests) 5ms<br> ✓ src/app/workspaceSurfaceStyles.test.ts (2 tests) 6ms<br> ✓ src/app/commands/regis |
| i18n catalog parity | yes | PASS | 970ms | RUN  v3.2.7 /projetos/tools/IAPro-Nexus-Workspace-OS-Handoff-2026-08-29/ai-manager/web<br><br> ✓ src/i18n/i18n.test.ts (7 tests) 13ms<br><br> Test Files  1 passed (1)<br>      Tests  7 passed (7)<br>   Start at  12:36:56<br>   Duration  539ms (transform 208ms, setup 0ms, collect 267ms, tests 13ms, environment 0ms, prepare 78ms) |
| Build + embed (`node scripts/build.mjs`) | yes | PASS | 1720ms | Nexus web build complete: /projetos/tools/IAPro-Nexus-Workspace-OS-Handoff-2026-08-29/ai-manager/web/dist<br>≈ tailwindcss v4.3.3<br><br>Done in 235ms<br><br>  dist/bundle.js                                   354.5kb<br>  dist/chunks/chunk-5YEWBEYR.js                    340.4kb<br>  dist/chunks/chunk-5DVGUEP5.js                    278.7kb<br>  dist/chunks/FlowRunsHistorySurface-ETTUGYKX.js   215.8kb<br>  dist/chunks/chunk-FDQS3BTZ.js                    208.8kb<br>  dist/bundle.css                |
| Embed sync (web/dist ≡ internal/.../embedded) | yes | PASS | 0ms | bundles idênticos (363034 bytes) |
| Critical UI markers in bundle | yes | PASS | 3ms | marcadores críticos presentes (7) |

## Residual risks / next operator steps

- Automated gates green.
- If UI still looks broken in the browser: restart `nexus web` so the new embedded bundle is loaded (`make build`).
- Manual smoke (not automated here): open Project Overview, Radar, one Agent terminal, and a second Project focus switch.

### Dirty paths

```
M web/scripts/e2e-capacity.mjs
 M web/src/workspace/WorkspaceRenderer.tsx
 M web/src/workspace/presentation.test.ts
 M web/src/workspace/presentation.ts
```

## How to regenerate

```bash
make web-verify
# or
npm --prefix web run verify
```
