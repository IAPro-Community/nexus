# Frontend verification report

- Generated: `2026-11-16T19:00:33Z`
- Branch: `release/v1.0.0` @ `21acb3b5`
- Verdict: **PASS** (10 pass / 0 fail)
- Dirty web/dist tree: no

## Gates

| Gate | Hard | Status | Duration | Detail |
| --- | --- | --- | --- | --- |
| Prettier (`prettier --check`) | yes | PASS | 14385ms | Checking formatting...<br>All matched files use Prettier code style! |
| TypeScript (`tsc --noEmit`) | yes | PASS | 14904ms | ok |
| ESLint (`eslint src`) | yes | PASS | 8026ms | /projetos/nexus/web/src/features/work/FlowRunSurface.tsx<br>  122:6  warning  React Hook useCallback has a missing dependency: 'plan.current_revision'. Either include it or remove the dependency array  react-hooks/exhaustive-deps<br><br>✖ 1 problem (0 errors, 1 warning) |
| Stylelint (`stylelint "src/**/*.css"`) | yes | PASS | 1583ms | ok |
| Null-safe API array access | yes | PASS | 89ms | sem .length/.map direto em campos nullable conhecidos |
| Vitest (`vitest run`) | yes | PASS | 11312ms | RUN  v4.1.12 /projetos/nexus/web<br><br><br> Test Files  142 passed (142)<br>      Tests  904 passed (904)<br>   Start at  15:01:13<br>   Duration  10.91s (transform 8.91s, setup 0ms, import 24.08s, tests 3.04s, environment 30ms) |
| i18n catalog parity | yes | PASS | 1137ms | RUN  v4.1.12 /projetos/nexus/web<br><br><br> Test Files  1 passed (1)<br>      Tests  8 passed (8)<br>   Start at  15:01:24<br>   Duration  737ms (transform 444ms, setup 0ms, import 547ms, tests 31ms, environment 0ms) |
| Build + embed (`node scripts/build.mjs`) | yes | PASS | 2216ms | Nexus web build complete: /projetos/nexus/web/dist<br>≈ tailwindcss v4.3.3<br><br>Done in 284ms<br><br>  dist/bundle.js                                  435.7kb<br>  dist/chunks/chunk-YHPWPAJ3.js                   359.1kb<br>  dist/chunks/chunk-4YRQH3AP.js                   339.5kb<br>  dist/chunks/chunk-4HQ5TX2J.js                   282.7kb<br>  dist/chunks/FlowRunsHistorySurface-YQRPX7N7.js  254.0kb<br>  dist/bundle.css                                 161.8kb<br>  dist/chunks/chunk-LJIWKH22.js |
| Embed sync (web/dist ≡ internal/.../embedded) | yes | PASS | 1ms | bundles idênticos (446133 bytes) |
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
