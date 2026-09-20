# Frontend verification report

- Generated: `2026-09-20T02:41:21Z`
- Branch: `feat/capacity-monitor-revival` @ `7813105`
- Verdict: **PASS** (10 pass / 0 fail)
- Dirty web/dist tree: **yes**

## Gates

| Gate | Hard | Status | Duration | Detail |
| --- | --- | --- | --- | --- |
| Prettier (`prettier --check`) | yes | PASS | 5319ms | Checking formatting...<br>All matched files use Prettier code style! |
| TypeScript (`tsc --noEmit`) | yes | PASS | 8958ms | ok |
| ESLint (`eslint src`) | yes | PASS | 5015ms | ok |
| Stylelint (`stylelint "src/**/*.css"`) | yes | PASS | 1063ms | ok |
| Null-safe API array access | yes | PASS | 69ms | sem .length/.map direto em campos nullable conhecidos |
| Vitest (`vitest run`) | yes | PASS | 10417ms | ✓ src/workspace/ptyLiveChrome.test.ts (4 tests) 5ms<br> ✓ src/app/maestroHonesty.test.ts (2 tests) 4ms<br> ✓ src/features/agents/askAgentModel.test.ts (2 tests) 5ms<br> ✓ src/nexus/terminalSettings.test.ts (6 tests) 5ms<br> ✓ src/features/overview/overviewResumeModel.test.ts (2 tests) 5ms<br> ✓ src/workspace/surfaceAttention.test.ts (6 tests) 5ms<br> ✓ src/app/routerIntegration.test.tsx (3 tests) 4ms<br> ✓ src/features/agents/NewAgentModal.test.tsx (1 test) 5ms<br> ✓ src/features/work/flowRunFil |
| i18n catalog parity | yes | PASS | 1047ms | RUN  v3.2.7 /projetos/tools/IAPro-Nexus-Workspace-OS-Handoff-2026-08-29/ai-manager/web<br><br> ✓ src/i18n/i18n.test.ts (7 tests) 18ms<br><br> Test Files  1 passed (1)<br>      Tests  7 passed (7)<br>   Start at  02:41:52<br>   Duration  589ms (transform 239ms, setup 0ms, collect 315ms, tests 18ms, environment 0ms, prepare 71ms) |
| Build + embed (`node scripts/build.mjs`) | yes | PASS | 2040ms | Nexus web build complete: /projetos/tools/IAPro-Nexus-Workspace-OS-Handoff-2026-08-29/ai-manager/web/dist<br>≈ tailwindcss v4.3.3<br><br>Done in 283ms<br><br>  dist/bundle.js                                   352.3kb<br>  dist/chunks/chunk-TYNMXDXP.js                    340.3kb<br>  dist/chunks/chunk-5DVGUEP5.js                    278.7kb<br>  dist/chunks/FlowRunsHistorySurface-MLW4VFGX.js   215.4kb<br>  dist/chunks/chunk-G2KBK62E.js                    208.7kb<br>  dist/bundle.css                |
| Embed sync (web/dist ≡ internal/.../embedded) | yes | PASS | 3ms | bundles idênticos (360765 bytes) |
| Critical UI markers in bundle | yes | PASS | 3ms | marcadores críticos presentes (7) |

## Residual risks / next operator steps

- Automated gates green.
- If UI still looks broken in the browser: restart `nexus web` so the new embedded bundle is loaded (`make build`).
- Manual smoke (not automated here): open Project Overview, Radar, one Agent terminal, and a second Project focus switch.

### Dirty paths

```
M web/src/app/WorkspaceSurfaceHost.tsx
 M web/src/features/work/FlowRunSurface.tsx
 M web/src/features/work/FlowRunsHistorySurface.tsx
 M web/src/features/work/ProjectIntelligenceInspector.tsx
 M web/src/features/work/flowDraftEditorModel.ts
 M web/src/features/work/useFlowDraftEditor.ts
 M web/src/i18n/resources.ts
 M web/src/workspace/WorkspaceRenderer.tsx
?? web/scripts/e2e-fixture.mjs
?? web/scripts/e2e-surface-isolation.mjs
?? web/src/features/work/flowRefreshGuard.test.ts
?? web/src/features/work/flowRefreshGuard.ts
```

## How to regenerate

```bash
make web-verify
# or
npm --prefix web run verify
```
