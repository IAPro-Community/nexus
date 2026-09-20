# Frontend verification report

- Generated: `2026-09-20T12:12:43Z`
- Branch: `feat/capacity-monitor-revival` @ `4cb4e26`
- Verdict: **PASS** (10 pass / 0 fail)
- Dirty web/dist tree: **yes**

## Gates

| Gate | Hard | Status | Duration | Detail |
| --- | --- | --- | --- | --- |
| Prettier (`prettier --check`) | yes | PASS | 8102ms | Checking formatting...<br>All matched files use Prettier code style! |
| TypeScript (`tsc --noEmit`) | yes | PASS | 12026ms | ok |
| ESLint (`eslint src`) | yes | PASS | 7212ms | ok |
| Stylelint (`stylelint "src/**/*.css"`) | yes | PASS | 1337ms | ok |
| Null-safe API array access | yes | PASS | 69ms | sem .length/.map direto em campos nullable conhecidos |
| Vitest (`vitest run`) | yes | PASS | 14787ms | ✓ src/features/projects/projectDirectoryPicker.test.ts (2 tests) 10ms<br> ✓ src/notifications/capacityAlert.test.ts (5 tests) 30ms<br> ✓ src/features/work/flowRefreshGuard.test.ts (3 tests) 12ms<br> ✓ src/features/work/missionAutonomyModel.test.ts (1 test) 7ms<br> ✓ src/features/conversation/nexusConversationMapper.test.ts (6 tests) 10ms<br> ✓ src/features/projects/projectRail.test.ts (4 tests) 6ms<br> ✓ src/app/tour/tour.test.ts (5 tests) 5ms<br> ✓ src/features/work/clarificationModel.test.ts ( |
| i18n catalog parity | yes | PASS | 1245ms | RUN  v3.2.7 /projetos/tools/IAPro-Nexus-Workspace-OS-Handoff-2026-08-29/ai-manager/web<br><br> ✓ src/i18n/i18n.test.ts (7 tests) 15ms<br><br> Test Files  1 passed (1)<br>      Tests  7 passed (7)<br>   Start at  12:13:27<br>   Duration  701ms (transform 305ms, setup 0ms, collect 396ms, tests 15ms, environment 0ms, prepare 86ms) |
| Build + embed (`node scripts/build.mjs`) | yes | PASS | 2222ms | Nexus web build complete: /projetos/tools/IAPro-Nexus-Workspace-OS-Handoff-2026-08-29/ai-manager/web/dist<br>≈ tailwindcss v4.3.3<br><br>Done in 335ms<br><br>  dist/bundle.js                                   353.4kb<br>  dist/chunks/chunk-TYNMXDXP.js                    340.3kb<br>  dist/chunks/chunk-5DVGUEP5.js                    278.7kb<br>  dist/chunks/FlowRunsHistorySurface-MLW4VFGX.js   215.4kb<br>  dist/chunks/chunk-FDQS3BTZ.js                    208.8kb<br>  dist/bundle.css                |
| Embed sync (web/dist ≡ internal/.../embedded) | yes | PASS | 1ms | bundles idênticos (361871 bytes) |
| Critical UI markers in bundle | yes | PASS | 2ms | marcadores críticos presentes (7) |

## Residual risks / next operator steps

- Automated gates green.
- If UI still looks broken in the browser: restart `nexus web` so the new embedded bundle is loaded (`make build`).
- Manual smoke (not automated here): open Project Overview, Radar, one Agent terminal, and a second Project focus switch.

### Dirty paths

```
M web/package.json
 M web/src/app/workspace-os.css
 M web/src/i18n/resources.ts
 M web/src/workspace/WorkspacePresentationProvider.tsx
 M web/src/workspace/WorkspaceRenderer.tsx
 M web/src/workspace/presentation.test.ts
 M web/src/workspace/presentation.ts
?? web/scripts/e2e-capacity.mjs
```

## How to regenerate

```bash
make web-verify
# or
npm --prefix web run verify
```
