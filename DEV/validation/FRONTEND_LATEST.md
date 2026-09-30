# Frontend verification report

- Generated: `2026-09-30T20:42:58Z`
- Branch: `release/v1.0.0` @ `2da6892`
- Verdict: **PASS** (10 pass / 0 fail)
- Dirty web/dist tree: no

## Gates

| Gate | Hard | Status | Duration | Detail |
| --- | --- | --- | --- | --- |
| Prettier (`prettier --check`) | yes | PASS | 7940ms | Checking formatting...<br>All matched files use Prettier code style! |
| TypeScript (`tsc --noEmit`) | yes | PASS | 11328ms | ok |
| ESLint (`eslint src`) | yes | PASS | 6662ms | ok |
| Stylelint (`stylelint "src/**/*.css"`) | yes | PASS | 2162ms | ok |
| Null-safe API array access | yes | PASS | 78ms | sem .length/.map direto em campos nullable conhecidos |
| Vitest (`vitest run`) | yes | PASS | 9312ms | stderr | src/platform/platformBridge.test.ts<br><br>⚠️ Browser Environment Detected <br><br>Only UI previews are available in the browser. For full functionality, please run the application in desktop mode.<br>More information at: https://v3.wails.io/learn/build/#using-a-browser-for-development<br><br><br>stderr | src/features/agents/NewAgentModal.test.tsx<br><br>⚠️ Browser Environment Detected <br><br>Only UI previews are available in the browser. For full functionality, please run the applicat |
| i18n catalog parity | yes | PASS | 1326ms | RUN  v3.2.7 /projetos/nexus/web<br><br> ✓ src/i18n/i18n.test.ts (7 tests) 14ms<br><br> Test Files  1 passed (1)<br>      Tests  7 passed (7)<br>   Start at  16:43:36<br>   Duration  855ms (transform 363ms, setup 0ms, collect 397ms, tests 14ms, environment 0ms, prepare 156ms) |
| Build + embed (`node scripts/build.mjs`) | yes | PASS | 1711ms | Nexus web build complete: /projetos/nexus/web/dist<br>≈ tailwindcss v4.3.3<br><br>Done in 253ms<br><br>  dist/bundle.js                                   374.2kb<br>  dist/chunks/chunk-3EAVDZAB.js                    348.8kb<br>  dist/chunks/chunk-3FH2DOZO.js                    278.9kb<br>  dist/chunks/chunk-FGAY72LC.js                    224.6kb<br>  dist/chunks/FlowRunsHistorySurface-STLUAENE.js   221.3kb<br>  dist/bundle.css                                  158.2kb<br>  dist/chunks/chunk-Y2AW7 |
| Embed sync (web/dist ≡ internal/.../embedded) | yes | PASS | 0ms | bundles idênticos (383175 bytes) |
| Critical UI markers in bundle | yes | PASS | 3ms | marcadores críticos presentes (7) |

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
