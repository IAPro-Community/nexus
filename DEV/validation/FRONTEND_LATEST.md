# Frontend verification report

- Generated: `2026-10-09T03:52:52Z`
- Branch: `fix/windows-beta10-homologation-20261008` @ `0782f876`
- Verdict: **PASS** (10 pass / 0 fail)
- Dirty web/dist tree: no

## Gates

| Gate | Hard | Status | Duration | Detail |
| --- | --- | --- | --- | --- |
| Prettier (`prettier --check`) | yes | PASS | 4854ms | Checking formatting...<br>All matched files use Prettier code style! |
| TypeScript (`tsc --noEmit`) | yes | PASS | 7356ms | ok |
| ESLint (`eslint src`) | yes | PASS | 4678ms | ok |
| Stylelint (`stylelint "src/**/*.css"`) | yes | PASS | 1287ms | ok |
| Null-safe API array access | yes | PASS | 75ms | sem .length/.map direto em campos nullable conhecidos |
| Vitest (`vitest run`) | yes | PASS | 6773ms | stderr | src/platform/platformBridge.test.ts<br><br>⚠️ Browser Environment Detected <br><br>Only UI previews are available in the browser. For full functionality, please run the application in desktop mode.<br>More information at: https://v3.wails.io/learn/build/#using-a-browser-for-development<br><br><br>stderr | src/features/projects/projectDirectoryPicker.test.ts<br><br>⚠️ Browser Environment Detected <br><br>Only UI previews are available in the browser. For full functionality, please run th |
| i18n catalog parity | yes | PASS | 1179ms | RUN  v3.2.7 C:/Users/kiver/AppData/Local/Temp/nexus-windows-release-95c7121d/web<br><br> ✓ src/i18n/i18n.test.ts (7 tests) 9ms<br><br> Test Files  1 passed (1)<br>      Tests  7 passed (7)<br>   Start at  00:53:17<br>   Duration  609ms (transform 179ms, setup 0ms, collect 256ms, tests 9ms, environment 0ms, prepare 131ms) |
| Build + embed (`node scripts/build.mjs`) | yes | PASS | 1512ms | Nexus web build complete: C:\Users\kiver\AppData\Local\Temp\nexus-windows-release-95c7121d\web\dist<br>≈ tailwindcss v4.3.3<br><br>Done in 217ms<br><br>  dist\bundle.js                                  375.9kb<br>  dist\chunks\chunk-OTEGMETV.js                   348.8kb<br>  dist\chunks\chunk-ZWPOKFBC.js                   280.8kb<br>  dist\chunks\chunk-NJXM4AT6.js                   226.6kb<br>  dist\chunks\FlowRunsHistorySurface-X7KPHEGA.js  221.3kb<br>  ...and 58 more output files...<br><br>Don |
| Embed sync (web/dist ≡ internal/.../embedded) | yes | PASS | 86ms | bundles idênticos (384958 bytes) |
| Critical UI markers in bundle | yes | PASS | 2ms | marcadores críticos presentes (7) |

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
