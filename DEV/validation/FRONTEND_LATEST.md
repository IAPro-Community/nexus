# Frontend verification report

- Generated: `2026-10-16T02:59:41Z`
- Branch: `fix/release-beta10-windows-closure-20261014` @ `36445f10`
- Verdict: **PASS** (10 pass / 0 fail)
- Dirty web/dist tree: no

## Gates

| Gate | Hard | Status | Duration | Detail |
| --- | --- | --- | --- | --- |
| Prettier (`prettier --check`) | yes | PASS | 8616ms | Checking formatting...<br>All matched files use Prettier code style! |
| TypeScript (`tsc --noEmit`) | yes | PASS | 12790ms | ok |
| ESLint (`eslint src`) | yes | PASS | 7251ms | ok |
| Stylelint (`stylelint "src/**/*.css"`) | yes | PASS | 1927ms | ok |
| Null-safe API array access | yes | PASS | 157ms | sem .length/.map direto em campos nullable conhecidos |
| Vitest (`vitest run`) | yes | PASS | 12553ms | stderr | src/platform/platformBridge.test.ts<br><br>⚠️ Browser Environment Detected <br><br>Only UI previews are available in the browser. For full functionality, please run the application in desktop mode.<br>More information at: https://v3.wails.io/learn/build/#using-a-browser-for-development<br><br><br>stderr | src/features/projects/projectDirectoryPicker.test.ts<br><br>⚠️ Browser Environment Detected <br><br>Only UI previews are available in the browser. For full functionality, please run th |
| i18n catalog parity | yes | PASS | 2176ms | RUN  v3.2.7 C:/Users/kiver/nexus-fix-b10/web<br><br> ✓ src/i18n/i18n.test.ts (7 tests) 17ms<br><br> Test Files  1 passed (1)<br>      Tests  7 passed (7)<br>   Start at  00:00:25<br>   Duration  1.23s (transform 284ms, setup 0ms, collect 416ms, tests 17ms, environment 0ms, prepare 231ms) |
| Build + embed (`node scripts/build.mjs`) | yes | PASS | 3123ms | Nexus web build complete: C:\Users\kiver\nexus-fix-b10\web\dist<br>≈ tailwindcss v4.3.3<br><br>Done in 414ms<br><br>  dist\bundle.js                                  376.1kb<br>  dist\chunks\chunk-7MBDQLDS.js                   348.8kb<br>  dist\chunks\chunk-25456IRH.js                   280.9kb<br>  dist\chunks\chunk-NJXM4AT6.js                   226.6kb<br>  dist\chunks\FlowRunsHistorySurface-EVYUTDXR.js  221.3kb<br>  ...and 58 more output files...<br><br>⚡ Done in 1550ms |
| Embed sync (web/dist ≡ internal/.../embedded) | yes | PASS | 189ms | bundles idênticos (385156 bytes) |
| Critical UI markers in bundle | yes | PASS | 3ms | marcadores críticos presentes (7) |

## Residual risks / next operator steps

- Automated gates green.
- If UI still looks broken in the browser: restart `nexus web` so the new embedded bundle is loaded (`make build`).
- Manual smoke (not automated here): open Project Overview, Radar, one Agent terminal, and a second Project focus switch.

### Notes

- Binário `nexus` local não encontrado neste check — após PASS, rode `make build` para instalar.

## How to regenerate

```bash
make web-verify
# or
npm --prefix web run verify
```
