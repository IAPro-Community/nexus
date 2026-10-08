# Frontend verification report

- Generated: `2026-10-08T19:17:11Z`
- Branch: `fix/windows-beta8-homologation-20261008` @ `ab397c54`
- Verdict: **PASS** (10 pass / 0 fail)
- Dirty web/dist tree: no

## Gates

| Gate | Hard | Status | Duration | Detail |
| --- | --- | --- | --- | --- |
| Prettier (`prettier --check`) | yes | PASS | 5028ms | Checking formatting...<br>All matched files use Prettier code style! |
| TypeScript (`tsc --noEmit`) | yes | PASS | 7583ms | ok |
| ESLint (`eslint src`) | yes | PASS | 4813ms | ok |
| Stylelint (`stylelint "src/**/*.css"`) | yes | PASS | 1293ms | ok |
| Null-safe API array access | yes | PASS | 108ms | sem .length/.map direto em campos nullable conhecidos |
| Vitest (`vitest run`) | yes | PASS | 8480ms | stderr | src/platform/platformBridge.test.ts<br><br>⚠️ Browser Environment Detected <br><br>Only UI previews are available in the browser. For full functionality, please run the application in desktop mode.<br>More information at: https://v3.wails.io/learn/build/#using-a-browser-for-development<br><br><br>stderr | src/features/projects/projectDirectoryPicker.test.ts<br><br>⚠️ Browser Environment Detected <br><br>Only UI previews are available in the browser. For full functionality, please run th |
| i18n catalog parity | yes | PASS | 1390ms | RUN  v3.2.7 web<br><br> ✓ src/i18n/i18n.test.ts (7 tests) 10ms<br><br> Test Files  1 passed (1)<br>      Tests  7 passed (7)<br>   Start at  16:17:39<br>   Duration  777ms (transform 181ms, setup 0ms, collect 273ms, tests 10ms, environment 0ms, prepare 182ms) |
| Build + embed (`node scripts/build.mjs`) | yes | PASS | 1749ms | Nexus web build complete: web/dist<br>≈ tailwindcss v4.3.3<br><br>Done in 221ms<br><br>  dist\bundle.js                                  374.4kb<br>  dist\chunks\chunk-OTEGMETV.js                   348.8kb<br>  dist\chunks\chunk-J3B5NHJN.js                   280.3kb<br>  dist\chunks\chunk-BOCAUZ5C.js                   225.4kb<br>  dist\chunks\FlowRunsHistorySurface-X7KPHEGA.js  221.3kb<br>  ...and 58 more output files...<br><br>Done |
| Embed sync (web/dist ≡ internal/.../embedded) | yes | PASS | 95ms | bundles idênticos (383381 bytes) |
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
