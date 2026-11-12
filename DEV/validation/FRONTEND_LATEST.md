# Frontend verification report

- Generated: `2026-11-09T19:10:01Z`
- Branch: `release/v1.0.0` @ `e7268be6`
- Verdict: **PASS** (10 pass / 0 fail)
- Dirty web/dist tree: **yes**

## Gates

| Gate | Hard | Status | Duration | Detail |
| --- | --- | --- | --- | --- |
| Prettier (`prettier --check`) | yes | PASS | 6858ms | Checking formatting...<br>All matched files use Prettier code style! |
| TypeScript (`tsc --noEmit`) | yes | PASS | 10586ms | ok |
| ESLint (`eslint src`) | yes | PASS | 5585ms | /projetos/nexus/web/src/features/work/ComparisonLabPanel.tsx<br>  10:3  warning  'ComparisonRunEvidence' is defined but never used. Allowed unused vars must match /^_/u  @typescript-eslint/no-unused-vars<br><br>/projetos/nexus/web/src/features/work/FlowRunSurface.tsx<br>  122:6  warning  React Hook useCallback has a missing dependency: 'plan.current_revision'. Either include it or remove the dependency array  react-hooks/exhaustive-deps<br><br>✖ 2 problems (0 errors, 2 warnings) |
| Stylelint (`stylelint "src/**/*.css"`) | yes | PASS | 1194ms | ok |
| Null-safe API array access | yes | PASS | 92ms | sem .length/.map direto em campos nullable conhecidos |
| Vitest (`vitest run`) | yes | PASS | 5916ms | RUN  v4.1.12 /projetos/nexus/web<br><br><br> Test Files  135 passed (135)<br>      Tests  850 passed (850)<br>   Start at  15:10:25<br>   Duration  5.58s (transform 4.77s, setup 0ms, import 12.84s, tests 1.40s, environment 18ms) |
| i18n catalog parity | yes | PASS | 693ms | RUN  v4.1.12 /projetos/nexus/web<br><br><br> Test Files  1 passed (1)<br>      Tests  8 passed (8)<br>   Start at  15:10:31<br>   Duration  411ms (transform 224ms, setup 0ms, import 282ms, tests 15ms, environment 0ms) |
| Build + embed (`node scripts/build.mjs`) | yes | PASS | 1611ms | NEX 32-bit official SVG exported: /projetos/nexus/web/dist/nex<br>Nexus web build complete: /projetos/nexus/web/dist<br>≈ tailwindcss v4.3.3<br><br>Done in 214ms<br><br>  dist/bundle.js                                  437.9kb<br>  dist/chunks/chunk-44MWUHYB.js                   359.1kb<br>  dist/chunks/chunk-4QLW7V67.js                   320.8kb<br>  dist/chunks/chunk-4HQ5TX2J.js                   282.7kb<br>  dist/chunks/FlowRunsHistorySurface-PDEKAM5D.js  254.0kb<br>  dist/bundle.css          |
| Embed sync (web/dist ≡ internal/.../embedded) | yes | PASS | 0ms | bundles idênticos (448420 bytes) |
| Critical UI markers in bundle | yes | PASS | 3ms | marcadores críticos presentes (7) |

## Residual risks / next operator steps

- Automated gates green.
- If UI still looks broken in the browser: restart `nexus web` so the new embedded bundle is loaded (`make build`).
- Manual smoke (not automated here): open Project Overview, Radar, one Agent terminal, and a second Project focus switch.

### Dirty paths

```
M web/package.json
 M web/src/app/tour/tourTargets.test.ts
 M web/src/features/agents/AgentsSurface.module.scss
 M web/src/features/agents/AgentsSurface.tsx
 M web/src/features/browser/BrowserSurface.module.scss
 M web/src/features/browser/BrowserSurface.tsx
 M web/src/features/settings/intelligenceProfiles.test.ts
 M web/src/features/work/ComparisonLabPanel.tsx
 M web/src/features/work/featureAnnouncements.test.ts
 M web/src/features/work/featureAnnouncements.ts
 M web/src/features/work/featureReveal.test.ts
 M web/src/features/work/featureTour.test.ts
 M web/src/features/work/featureTourAnchors.test.ts
 M web/src/features/work/featureTourDefinitions.ts
 M web/src/i18n/resources.ts
 M web/src/nexus/api.ts
 M web/src/notifications/InAppNotificationCenter.tsx
 M web/src/types.ts
?? web/scripts/e2e-agent-groups-live-session.mjs
?? web/scripts/e2e-browser-design-inspect.mjs
?? web/src/app/focusQueueModel.test.ts
?? web/src/app/focusQueueModel.ts
?? web/src/features/agents/AgentsSurface.test.ts
?? web/src/features/agents/agentGroupsModel.test.ts
?? web/src/features/agents/agentGroupsModel.ts
?? web/src/features/browser/designInspectHandoff.test.ts
?? web/src/features/browser/designInspectHandoff.ts
?? web/src/notifications/FocusQueueSection.module.scss
?? web/src/notifications/FocusQueueSection.tsx
```

## How to regenerate

```bash
make web-verify
# or
npm --prefix web run verify
```
