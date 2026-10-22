# Frontend verification report

- Generated: `2026-10-17T18:34:42Z`
- Branch: `release/v1.0.0` @ `85c5b958`
- Verdict: **PASS** (10 pass / 0 fail)
- Dirty web/dist tree: no

## Gates

| Gate | Hard | Status | Duration | Detail |
| --- | --- | --- | --- | --- |
| Prettier (`prettier --check`) | yes | PASS | 8332ms | Checking formatting...<br>All matched files use Prettier code style! |
| TypeScript (`tsc --noEmit`) | yes | PASS | 17122ms | ok |
| ESLint (`eslint src`) | yes | PASS | 20861ms | ok |
| Stylelint (`stylelint "src/**/*.css"`) | yes | PASS | 5881ms | ok |
| Null-safe API array access | yes | PASS | 112ms | sem .length/.map direto em campos nullable conhecidos |
| Vitest (`vitest run`) | yes | PASS | 15445ms | stderr | src/features/projects/projectDirectoryPicker.test.ts<br><br>⚠️ Browser Environment Detected <br><br>Only UI previews are available in the browser. For full functionality, please run the application in desktop mode.<br>More information at: https://v3.wails.io/learn/build/#using-a-browser-for-development<br><br><br>stderr | src/platform/platformBridge.test.ts<br><br>⚠️ Browser Environment Detected <br><br>Only UI previews are available in the browser. For full functionality, please run th |
| i18n catalog parity | yes | PASS | 1825ms | RUN  v3.2.7 D:/dev/nexus-remediation/nexus-homologation/web<br><br> ✓ src/i18n/i18n.test.ts (7 tests) 14ms<br><br> Test Files  1 passed (1)<br>      Tests  7 passed (7)<br>   Start at  15:35:51<br>   Duration  1.06s (transform 320ms, setup 0ms, collect 467ms, tests 14ms, environment 0ms, prepare 152ms) |
| Build + embed (`node scripts/build.mjs`) | yes | PASS | 8774ms | Nexus web build complete: D:\dev\nexus-remediation\nexus-homologation\web\dist<br>≈ tailwindcss v4.3.3<br><br>Done in 764ms<br><br>  dist\bundle.js                                  374.6kb<br>  dist\chunks\chunk-GPXL3T2K.js                   348.8kb<br>  dist\chunks\chunk-25456IRH.js                   280.9kb<br>  dist\chunks\chunk-NJXM4AT6.js                   226.6kb<br>  dist\chunks\FlowRunsHistorySurface-7IOELUHI.js  221.3kb<br>  ...and 58 more output files...<br><br>⚡ Done in 5994ms |
| Embed sync (web/dist ≡ internal/.../embedded) | yes | PASS | 2ms | bundles idênticos (383584 bytes) |
| Critical UI markers in bundle | yes | PASS | 4ms | marcadores críticos presentes (7) |

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
