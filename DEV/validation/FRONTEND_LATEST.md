# Frontend verification report

- Generated: `2026-10-27T18:55:23Z`
- Branch: `release/v1.0.0` @ `2207f320`
- Verdict: **PASS** (10 pass / 0 fail)
- Dirty web/dist tree: **yes**

## Gates

| Gate | Hard | Status | Duration | Detail |
| --- | --- | --- | --- | --- |
| Prettier (`prettier --check`) | yes | PASS | 7275ms | Checking formatting...<br>All matched files use Prettier code style! |
| TypeScript (`tsc --noEmit`) | yes | PASS | 10223ms | ok |
| ESLint (`eslint src`) | yes | PASS | 5527ms | /projetos/nexus/web/src/features/browser/BrowserSurface.tsx<br>  255:6  warning  React Hook useEffect has missing dependencies: 'projectId' and 'session'. Either include them or remove the dependency array  react-hooks/exhaustive-deps<br>  373:6  warning  React Hook useEffect has a missing dependency: 'session'. Either include it or remove the dependency array                    react-hooks/exhaustive-deps<br>  417:6  warning  React Hook useEffect has a missing dependency: 'session'. Either incl |
| Stylelint (`stylelint "src/**/*.css"`) | yes | PASS | 1263ms | ok |
| Null-safe API array access | yes | PASS | 78ms | sem .length/.map direto em campos nullable conhecidos |
| Vitest (`vitest run`) | yes | PASS | 6233ms | stderr | src/platform/platformBridge.test.ts<br><br>⚠️ Browser Environment Detected <br><br>Only UI previews are available in the browser. For full functionality, please run the application in desktop mode.<br>More information at: https://v3.wails.io/learn/build/#using-a-browser-for-development<br><br><br>stderr | src/features/work/clarificationModel.test.ts<br><br>⚠️ Browser Environment Detected <br><br>Only UI previews are available in the browser. For full functionality, please run the applic |
| i18n catalog parity | yes | PASS | 693ms | RUN  v4.1.12 /projetos/nexus/web<br><br> ✓ src/i18n/i18n.test.ts (7 tests) 17ms<br><br> Test Files  1 passed (1)<br>      Tests  7 passed (7)<br>   Start at  14:55:54<br>   Duration  421ms (transform 222ms, setup 0ms, import 281ms, tests 17ms, environment 0ms) |
| Build + embed (`node scripts/build.mjs`) | yes | PASS | 1714ms | NEX 32-bit official SVG exported: /projetos/nexus/web/public/nex<br>Nexus web build complete: /projetos/nexus/web/dist<br>≈ tailwindcss v4.3.3<br><br>Done in 198ms<br><br>  dist/bundle.js                                   425.1kb<br>  dist/chunks/chunk-BONOGKGG.js                    359.1kb<br>  dist/chunks/chunk-QGVOQYY7.js                    281.2kb<br>  dist/chunks/chunk-S3S2TXOM.js                    272.7kb<br>  dist/chunks/FlowRunsHistorySurface-KNDGV3CG.js   226.2kb<br>  dist/bundle.css   |
| Embed sync (web/dist ≡ internal/.../embedded) | yes | PASS | 1ms | bundles idênticos (435257 bytes) |
| Critical UI markers in bundle | yes | PASS | 4ms | marcadores críticos presentes (7) |

## Residual risks / next operator steps

- Automated gates green.
- If UI still looks broken in the browser: restart `nexus web` so the new embedded bundle is loaded (`make build`).
- Manual smoke (not automated here): open Project Overview, Radar, one Agent terminal, and a second Project focus switch.

### Dirty paths

```
M web/src/app/NexusWorkspaceApp.tsx
 M web/src/app/components/ProductUpdatesCenter.module.scss
 M web/src/app/components/ProductUpdatesCenter.tsx
 M web/src/app/tour/ProductTour.module.scss
 M web/src/app/tour/ProductTour.tsx
 M web/src/components/AttentionNotification.test.ts
 M web/src/components/AttentionNotificationManager.tsx
 M web/src/features/settings/SettingsSurface.tsx
 M web/src/features/work/ComposerReadOnlyHistory.tsx
 M web/src/features/work/ComposerStart.tsx
 M web/src/features/work/ComposerSurface.module.scss
 M web/src/features/work/FeatureTour.module.scss
 M web/src/features/work/FeatureTour.tsx
 M web/src/features/work/FlowRunsHistorySurface.module.scss
 M web/src/features/work/FlowRunsHistorySurface.tsx
 M web/src/features/work/featureAnnouncements.test.ts
 M web/src/features/work/featureAnnouncements.ts
 M web/src/features/work/featureTour.test.ts
 M web/src/features/work/featureTour.ts
 M web/src/features/work/featureTourAnchors.test.ts
 M web/src/features/work/featureTourDefinitions.ts
 M web/src/i18n/resources.ts
 M web/src/notifications/attentionDelivery.test.ts
 M web/src/notifications/attentionDelivery.ts
 M web/src/types.ts
```

## How to regenerate

```bash
make web-verify
# or
npm --prefix web run verify
```
