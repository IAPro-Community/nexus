# Frontend verification report

- Generated: `2026-10-09T00:09:05Z`
- Branch: `fix/windows-beta10-homologation-20261008` @ `8f07ff9f`
- Verdict: **PASS** (10 pass / 0 fail)
- Dirty web/dist tree: no

## Gates

| Gate | Hard | Status | Duration | Detail |
| --- | --- | --- | --- | --- |
| Prettier (`prettier --check`) | yes | PASS | 4736ms | Checking formatting...<br>All matched files use Prettier code style! |
| TypeScript (`tsc --noEmit`) | yes | PASS | 7774ms | ok |
| ESLint (`eslint src`) | yes | PASS | 4691ms | ok |
| Stylelint (`stylelint "src/**/*.css"`) | yes | PASS | 1286ms | ok |
| Null-safe API array access | yes | PASS | 78ms | sem .length/.map direto em campos nullable conhecidos |
| Vitest (`vitest run`) | yes | PASS | 7212ms | stderr | src/platform/platformBridge.test.ts<br><br>⚠️ Browser Environment Detected <br><br>Only UI previews are available in the browser. For full functionality, please run the application in desktop mode.<br>More information at: https://v3.wails.io/learn/build/#using-a-browser-for-development<br><br><br>stderr | src/features/projects/projectDirectoryPicker.test.ts<br><br>⚠️ Browser Environment Detected <br><br>Only UI previews are available in the browser. For full functionality, please run th |
| i18n catalog parity | yes | PASS | 1188ms | RUN  v3.2.7 C:/Users/kiver/AppData/Local/Temp/nexus-windows-release-95c7121d/web<br><br> ✓ src/i18n/i18n.test.ts (7 tests) 17ms<br><br> Test Files  1 passed (1)<br>      Tests  7 passed (7)<br>   Start at  21:09:32<br>   Duration  624ms (transform 181ms, setup 0ms, collect 255ms, tests 17ms, environment 0ms, prepare 132ms) |
| Build + embed (`node scripts/build.mjs`) | yes | PASS | 1679ms | Nexus web build complete: C:\Users\kiver\AppData\Local\Temp\nexus-windows-release-95c7121d\web\dist<br>≈ tailwindcss v4.3.3<br><br>Done in 204ms<br><br>  dist\bundle.js                                  375.0kb<br>  dist\chunks\chunk-T4ZRJ2ZU.js                   348.8kb<br>  dist\chunks\chunk-J3B5NHJN.js                   280.3kb<br>  dist\chunks\chunk-GQJ46AMX.js                   225.5kb<br>  dist\chunks\FlowRunsHistorySurface-6LARTTT4.js  221.3kb<br>  ...and 58 more output files...<br><br>Don |
| Embed sync (web/dist ≡ internal/.../embedded) | yes | PASS | 283ms | bundles idênticos (384038 bytes) |
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
