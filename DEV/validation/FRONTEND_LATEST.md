# Frontend verification report

- Generated: `2026-09-19T18:36:31Z`
- Branch: `feat/capacity-monitor-revival` @ `483f12e`
- Verdict: **PASS** (10 pass / 0 fail)
- Dirty web/dist tree: **yes**

## Gates

| Gate | Hard | Status | Duration | Detail |
| --- | --- | --- | --- | --- |
| Prettier (`prettier --check`) | yes | PASS | 6118ms | Checking formatting...<br>All matched files use Prettier code style! |
| TypeScript (`tsc --noEmit`) | yes | PASS | 10248ms | ok |
| ESLint (`eslint src`) | yes | PASS | 7704ms | ok |
| Stylelint (`stylelint "src/**/*.css"`) | yes | PASS | 2816ms | ok |
| Null-safe API array access | yes | PASS | 197ms | sem .length/.map direto em campos nullable conhecidos |
| Vitest (`vitest run`) | yes | PASS | 20231ms | ✓ src/app/routerIntegration.test.tsx (3 tests) 6ms<br> ✓ src/app/documentTitle.test.ts (6 tests) 5ms<br> ✓ src/nexus/terminalSettings.test.ts (6 tests) 6ms<br> ✓ src/keyboard/KeyboardShortcutRegistry.test.ts (4 tests) 7ms<br> ✓ src/app/workspaceSurfaceStyles.test.ts (2 tests) 6ms<br> ✓ src/features/work/projectIntelligencePresentation.test.ts (4 tests) 6ms<br> ✓ src/workspace/ptyLiveChrome.test.ts (4 tests) 4ms<br> ✓ src/lib/safeArray.test.ts (3 tests) 48ms<br> ✓ src/app/versionHonesty.test.ts ( |
| i18n catalog parity | yes | PASS | 1773ms | RUN  v3.2.7 /projetos/tools/IAPro-Nexus-Workspace-OS-Handoff-2026-08-29/nexus-capacity-worktree/web<br><br> ✓ src/i18n/i18n.test.ts (7 tests) 13ms<br><br> Test Files  1 passed (1)<br>      Tests  7 passed (7)<br>   Start at  18:37:18<br>   Duration  1.02s (transform 425ms, setup 0ms, collect 517ms, tests 13ms, environment 0ms, prepare 197ms) |
| Build + embed (`node scripts/build.mjs`) | yes | PASS | 2839ms | Nexus web build complete: /projetos/tools/IAPro-Nexus-Workspace-OS-Handoff-2026-08-29/nexus-capacity-worktree/web/dist<br>≈ tailwindcss v4.3.3<br><br>Done in 482ms<br><br>  dist/bundle.js                                   351.9kb<br>  dist/chunks/chunk-6XQRYVMA.js                    340.3kb<br>  dist/chunks/chunk-2NR5NDJ6.js                    278.7kb<br>  dist/chunks/FlowRunsHistorySurface-BZXCZW5D.js   214.4kb<br>  dist/chunks/chunk-XNP5XJDR.js                    202.1kb<br>  dist/bundle.css   |
| Embed sync (web/dist ≡ internal/.../embedded) | yes | PASS | 1ms | bundles idênticos (360353 bytes) |
| Critical UI markers in bundle | yes | PASS | 33ms | marcadores críticos presentes (7) |

## Residual risks / next operator steps

- Automated gates green.
- If UI still looks broken in the browser: restart `nexus web` so the new embedded bundle is loaded (`make build`).
- Manual smoke (not automated here): open Project Overview, Radar, one Agent terminal, and a second Project focus switch.

### Dirty paths

```
M web/src/features/settings/SettingsSurface.tsx
 M web/src/features/usage/UsageAccountCard.tsx
 M web/src/features/usage/UsageSurface.tsx
 M web/src/features/work/FlowRunSurface.tsx
 M web/src/i18n/resources.ts
 M web/src/nexus/api.ts
 M web/src/notifications/InAppNotificationCenter.tsx
 M web/src/notifications/inAppNotificationModel.ts
 M web/src/types.ts
?? web/src/features/settings/CapacityMonitorSettings.module.scss
?? web/src/features/settings/CapacityMonitorSettings.tsx
?? web/src/notifications/capacityAlert.test.ts
```

## How to regenerate

```bash
make web-verify
# or
npm --prefix web run verify
```
