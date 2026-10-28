# Frontend verification report

- Generated: `2026-10-28T11:51:35Z`
- Branch: `release/v1.0.0` @ `f8315c81`
- Verdict: **PASS** (10 pass / 0 fail)
- Dirty web/dist tree: **yes**

## Gates

| Gate | Hard | Status | Duration | Detail |
| --- | --- | --- | --- | --- |
| Prettier (`prettier --check`) | yes | PASS | 6003ms | Checking formatting...<br>All matched files use Prettier code style! |
| TypeScript (`tsc --noEmit`) | yes | PASS | 9434ms | ok |
| ESLint (`eslint src`) | yes | PASS | 5427ms | /projetos/nexus/web/src/features/browser/BrowserSurface.tsx<br>  255:6  warning  React Hook useEffect has missing dependencies: 'projectId' and 'session'. Either include them or remove the dependency array  react-hooks/exhaustive-deps<br>  373:6  warning  React Hook useEffect has a missing dependency: 'session'. Either include it or remove the dependency array                    react-hooks/exhaustive-deps<br>  417:6  warning  React Hook useEffect has a missing dependency: 'session'. Either incl |
| Stylelint (`stylelint "src/**/*.css"`) | yes | PASS | 1207ms | ok |
| Null-safe API array access | yes | PASS | 78ms | sem .length/.map direto em campos nullable conhecidos |
| Vitest (`vitest run`) | yes | PASS | 5269ms | stderr | src/platform/platformBridge.test.ts<br><br>⚠️ Browser Environment Detected <br><br>Only UI previews are available in the browser. For full functionality, please run the application in desktop mode.<br>More information at: https://v3.wails.io/learn/build/#using-a-browser-for-development<br><br><br>stderr | src/features/projects/projectDirectoryPicker.test.ts<br><br>⚠️ Browser Environment Detected <br><br>Only UI previews are available in the browser. For full functionality, please run th |
| i18n catalog parity | yes | PASS | 701ms | RUN  v4.1.12 /projetos/nexus/web<br><br> ✓ src/i18n/i18n.test.ts (7 tests) 13ms<br><br> Test Files  1 passed (1)<br>      Tests  7 passed (7)<br>   Start at  07:52:02<br>   Duration  412ms (transform 221ms, setup 0ms, import 279ms, tests 13ms, environment 0ms) |
| Build + embed (`node scripts/build.mjs`) | yes | PASS | 1527ms | NEX 32-bit official SVG exported: /projetos/nexus/web/dist/nex<br>Nexus web build complete: /projetos/nexus/web/dist<br>≈ tailwindcss v4.3.3<br><br>Done in 197ms<br><br>  dist/bundle.js                                   425.7kb<br>  dist/chunks/chunk-AYVAVEXM.js                    359.1kb<br>  dist/chunks/chunk-QGVOQYY7.js                    281.2kb<br>  dist/chunks/chunk-MWQSFZRS.js                    275.8kb<br>  dist/chunks/FlowRunsHistorySurface-5XV4ZZLV.js   226.2kb<br>  dist/bundle.css     |
| Embed sync (web/dist ≡ internal/.../embedded) | yes | PASS | 1ms | bundles idênticos (435889 bytes) |
| Critical UI markers in bundle | yes | PASS | 3ms | marcadores críticos presentes (7) |

## Residual risks / next operator steps

- Automated gates green.
- If UI still looks broken in the browser: restart `nexus web` so the new embedded bundle is loaded (`make build`).
- Manual smoke (not automated here): open Project Overview, Radar, one Agent terminal, and a second Project focus switch.

### Dirty paths

```
M web/package.json
 M web/src/app/tour/tourTargets.test.ts
 M web/src/components/attentionToastModel.test.ts
 M web/src/features/agents/NewAgentModal.tsx
 M web/src/features/work/ComposerReadOnlyHistory.tsx
 M web/src/features/work/ComposerSurface.tsx
 M web/src/features/work/FeatureTour.tsx
 M web/src/features/work/FlowRunSurface.tsx
 M web/src/features/work/featureReveal.test.ts
 M web/src/features/work/featureReveal.ts
 M web/src/features/work/featureTour.test.ts
 M web/src/i18n/resources.ts
 M web/src/types.ts
?? web/vitest.config.ts
```

## How to regenerate

```bash
make web-verify
# or
npm --prefix web run verify
```
