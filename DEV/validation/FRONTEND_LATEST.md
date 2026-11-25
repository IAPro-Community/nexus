# Frontend verification report

- Generated: `2026-11-25T06:18:48Z`
- Branch: `HEAD` @ `91752a84`
- Verdict: **PASS** (10 pass / 0 fail)
- Dirty web/dist tree: **yes**

## Gates

| Gate | Hard | Status | Duration | Detail |
| --- | --- | --- | --- | --- |
| Prettier (`prettier --check`) | yes | PASS | 11424ms | Checking formatting...<br>All matched files use Prettier code style! |
| TypeScript (`tsc --noEmit`) | yes | PASS | 15918ms | ok |
| ESLint (`eslint src`) | yes | PASS | 8564ms | /projetos/worktrees/nexus-iapro-sdk-local/web/src/features/work/FlowRunSurface.tsx<br>  122:6  warning  React Hook useCallback has a missing dependency: 'plan.current_revision'. Either include it or remove the dependency array  react-hooks/exhaustive-deps<br><br>/projetos/worktrees/nexus-iapro-sdk-local/web/src/workspace/WorkspaceProvider.tsx<br>  138:6  warning  React Hook useMemo has an unnecessary dependency: 'projectId'. Either exclude it or remove the dependency array  react-hooks/exhaustiv |
| Stylelint (`stylelint "src/**/*.css"`) | yes | PASS | 1785ms | ok |
| Null-safe API array access | yes | PASS | 120ms | sem .length/.map direto em campos nullable conhecidos |
| Vitest (`vitest run`) | yes | PASS | 12833ms | RUN  v4.1.12 /projetos/worktrees/nexus-iapro-sdk-local/web<br><br><br> Test Files  154 passed (154)<br>      Tests  1016 passed (1016)<br>   Start at  02:19:26<br>   Duration  12.41s (transform 11.46s, setup 0ms, import 36.21s, tests 3.75s, environment 18ms) |
| i18n catalog parity | yes | PASS | 980ms | RUN  v4.1.12 /projetos/worktrees/nexus-iapro-sdk-local/web<br><br><br> Test Files  1 passed (1)<br>      Tests  8 passed (8)<br>   Start at  02:19:39<br>   Duration  575ms (transform 303ms, setup 0ms, import 404ms, tests 27ms, environment 0ms) |
| Build + embed (`node scripts/build.mjs`) | yes | PASS | 2298ms | Nexus web build complete: /projetos/worktrees/nexus-iapro-sdk-local/web/dist<br>≈ tailwindcss v4.3.3<br><br>Done in 314ms<br><br>  dist/bundle.js                                  460.7kb<br>  dist/chunks/chunk-3P25JCEM.js                   386.6kb<br>  dist/chunks/chunk-RWZOLIXK.js                   359.2kb<br>  dist/chunks/chunk-L52VZCDJ.js                   283.5kb<br>  dist/chunks/FlowRunsHistorySurface-Y75KDXZU.js  261.0kb<br>  dist/bundle.css                                 171.0kb<br>  dis |
| Embed sync (web/dist ≡ internal/.../embedded) | yes | PASS | 1ms | bundles idênticos (471721 bytes) |
| Critical UI markers in bundle | yes | PASS | 3ms | marcadores críticos presentes (7) |

## Residual risks / next operator steps

- Automated gates green.
- If UI still looks broken in the browser: restart `nexus web` so the new embedded bundle is loaded (`make build`).
- Manual smoke (not automated here): open Project Overview, Radar, one Agent terminal, and a second Project focus switch.

### Dirty paths

```
M web/src/features/projects/ProjectRail.tsx
 M web/src/services/WorkspaceLayoutService.ts
 M web/src/workspace/WorkspacePresentationProvider.tsx
```

## How to regenerate

```bash
make web-verify
# or
npm --prefix web run verify
```
