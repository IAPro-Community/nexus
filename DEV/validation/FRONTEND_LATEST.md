# Frontend verification report

- Generated: `2026-10-25T10:41:55Z`
- Branch: `release/v1.0.0` @ `bf6adf0c`
- Verdict: **PASS** (10 pass / 0 fail)
- Dirty web/dist tree: **yes**

## Gates

| Gate | Hard | Status | Duration | Detail |
| --- | --- | --- | --- | --- |
| Prettier (`prettier --check`) | yes | PASS | 10576ms | Checking formatting...<br>All matched files use Prettier code style! |
| TypeScript (`tsc --noEmit`) | yes | PASS | 13983ms | ok |
| ESLint (`eslint src`) | yes | PASS | 9077ms | ok |
| Stylelint (`stylelint "src/**/*.css"`) | yes | PASS | 2691ms | ok |
| Null-safe API array access | yes | PASS | 165ms | sem .length/.map direto em campos nullable conhecidos |
| Vitest (`vitest run`) | yes | PASS | 14933ms | stderr | src/platform/platformBridge.test.ts<br><br>⚠️ Browser Environment Detected <br><br>Only UI previews are available in the browser. For full functionality, please run the application in desktop mode.<br>More information at: https://v3.wails.io/learn/build/#using-a-browser-for-development<br><br><br>stderr | src/features/agents/NewAgentModal.test.tsx<br><br>⚠️ Browser Environment Detected <br><br>Only UI previews are available in the browser. For full functionality, please run the applicat |
| i18n catalog parity | yes | PASS | 1663ms | RUN  v3.2.7 /projetos/nexus/web<br><br> ✓ src/i18n/i18n.test.ts (7 tests) 12ms<br><br> Test Files  1 passed (1)<br>      Tests  7 passed (7)<br>   Start at  06:42:46<br>   Duration  1.01s (transform 367ms, setup 0ms, collect 453ms, tests 12ms, environment 0ms, prepare 134ms) |
| Build + embed (`node scripts/build.mjs`) | yes | PASS | 2428ms | NEX 32-bit official SVG exported: /projetos/nexus/web/public/nex<br>Nexus web build complete: /projetos/nexus/web/dist<br>≈ tailwindcss v4.3.3<br><br>Done in 436ms<br><br>  dist/bundle.js                                   414.8kb<br>  dist/chunks/chunk-66W6FALB.js                    348.8kb<br>  dist/chunks/chunk-25456IRH.js                    280.9kb<br>  dist/chunks/chunk-CN5KJ7JK.js                    248.0kb<br>  dist/chunks/FlowRunsHistorySurface-K5BHCJWY.js   221.3kb<br>  dist/bundle.css   |
| Embed sync (web/dist ≡ internal/.../embedded) | yes | PASS | 1ms | bundles idênticos (424784 bytes) |
| Critical UI markers in bundle | yes | PASS | 2ms | marcadores críticos presentes (7) |

## Residual risks / next operator steps

- Automated gates green.
- If UI still looks broken in the browser: restart `nexus web` so the new embedded bundle is loaded (`make build`).
- Manual smoke (not automated here): open Project Overview, Radar, one Agent terminal, and a second Project focus switch.

### Dirty paths

```
M web/bun.lock
 M web/package.json
 M web/scripts/build.mjs
 M web/scripts/e2e-composer-execution-spine.mjs
 M web/scripts/e2e-composer-themes.mjs
 M web/scripts/e2e-composer-vague-execution-spine.mjs
 M web/scripts/e2e-composer-workspace.mjs
 M web/scripts/e2e-fixture.mjs
 M web/scripts/e2e-hud.mjs
 M web/scripts/e2e-platform.mjs
 M web/scripts/e2e-surface-isolation.mjs
 M web/src/app/DesktopHudApp.tsx
 M web/src/app/NexusWorkspaceApp.tsx
 M web/src/app/WorkspaceSurfaceHost.tsx
 M web/src/app/commands/CommandPalette.tsx
 M web/src/app/commands/registry.test.ts
 M web/src/app/commands/registry.ts
 M web/src/app/modals/WelcomeModal.tsx
 M web/src/components/TerminalPane.module.scss
 M web/src/components/TerminalPane.tsx
 M web/src/features/hud/GuardianMascot.test.tsx
 M web/src/features/hud/GuardianMascot.tsx
 M web/src/features/hud/HudSettings.tsx
 M web/src/features/hud/NexPixelSprite.tsx
 M web/src/features/hud/NexusHud.tsx
 M web/src/features/hud/_hudGuardian.scss
 M web/src/features/hud/attentionNavigation.test.ts
 M web/src/features/hud/attentionNavigation.ts
 M web/src/features/hud/guardianModel.test.ts
 M web/src/features/hud/guardianModel.ts
 M web/src/features/hud/hudMiniModel.ts
 M web/src/features/hud/hudModel.test.ts
 M web/src/features/hud/hudModel.ts
 M web/src/features/hud/hudPresentation.test.ts
 M web/src/features/hud/hudPresentation.ts
 M web/src/features/overview/ProviderReadinessBanner.tsx
 M web/src/features/projects/BranchSwitcherModal.tsx
 M web/src/features/projects/ProjectHub.module.scss
 M web/src/features/projects/ProjectHub.tsx
 M web/src/features/projects/firstRunReadiness.test.ts
 M web/src/features/projects/firstRunReadiness.ts
 M web/src/features/usage/AccountLoginModal.tsx
 M web/src/features/usage/RegisterProviderModal.tsx
 M web/src/features/usage/registerProviderModel.test.ts
 M web/src/features/usage/registerProviderModel.ts
 M web/src/features/work/DirectSessionLauncher.tsx
 M web/src/features/work/directSessionModel.test.ts
 M web/src/features/work/directSessionModel.ts
 M web/src/i18n/resources.ts
 M web/src/nexus/api.ts
 M web/src/platform/platformBridge.test.ts
 M web/src/platform/wailsV3.ts
 M web/src/services/WorkspaceLayoutService.test.ts
 M web/src/types.ts
 M web/src/workspace/WorkspacePresentationProvider.tsx
 M web/src/workspace/WorkspaceRenderer.tsx
 M web/src/workspace/presentation.test.ts
 M web/src/workspace/presentation.ts
?? web/public/nex/
?? web/scripts/css-module-scope.mjs
?? web/scripts/css-module-scope.node-test.mjs
?? web/scripts/e2e-docs-site-search.mjs
?? web/scripts/e2e-first-run-import.mjs
?? web/scripts/e2e-platform.node-test.mjs
?? web/scripts/e2e-terminal-split-grid.mjs
?? web/src/app/commands/CommandPalette.module.scss
?? web/src/app/providerLoginCommandWiring.test.ts
?? web/src/components/terminalPaneResize.test.ts
?? web/src/components/terminalPaneResize.ts
?? web/src/features/hud/GuardianMascot.module.scss
?? web/src/features/hud/GuardianModeSelector.test.tsx
?? web/src/features/hud/GuardianModeSelector.tsx
?? web/src/features/hud/companionModel.test.ts
?? web/src/features/hud/companionModel.ts
?? web/src/features/projects/BranchSwitcherModal.module.scss
?? web/src/features/projects/gitWorktrees.test.ts
?? web/src/features/projects/gitWorktrees.ts
?? web/src/features/work/ContextualResourceSelection.module.scss
?? web/src/features/work/directSessionLifecycle.test.ts
?? web/src/features/work/directSessionLifecycle.ts
?? web/src/lib/providerLoginCommand.test.ts
?? web/src/lib/providerLoginCommand.ts
?? web/src/workspace/PtyTabs.module.scss
?? web/src/workspace/TerminalSplit.module.scss
```

## How to regenerate

```bash
make web-verify
# or
npm --prefix web run verify
```
