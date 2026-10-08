# Frontend verification report

- Generated: `2026-10-08T18:13:14Z`
- Branch: `fix/windows-beta8-homologation-20261008` @ `7d2ef8aa`
- Verdict: **PASS** (10 pass / 0 fail)
- Dirty web/dist tree: no

## Gates

| Gate | Hard | Status | Duration | Detail |
| --- | --- | --- | --- | --- |
| Prettier (`prettier --check`) | yes | PASS | 5000ms | Checking formatting...<br>All matched files use Prettier code style! |
| TypeScript (`tsc --noEmit`) | yes | PASS | 7748ms | ok |
| ESLint (`eslint src`) | yes | PASS | 4868ms | ok |
| Stylelint (`stylelint "src/**/*.css"`) | yes | PASS | 1398ms | ok |
| Null-safe API array access | yes | PASS | 84ms | sem .length/.map direto em campos nullable conhecidos |
| Vitest (`vitest run`) | yes | PASS | 7693ms | stderr | src/platform/platformBridge.test.ts<br><br>⚠️ Browser Environment Detected <br><br>Only UI previews are available in the browser. For full functionality, please run the application in desktop mode.<br>More information at: https://v3.wails.io/learn/build/#using-a-browser-for-development<br><br><br>stderr | src/features/projects/projectDirectoryPicker.test.ts<br><br>⚠️ Browser Environment Detected <br><br>Only UI previews are available in the browser. For full functionality, please run th |
| i18n catalog parity | yes | PASS | 1263ms | RUN  v3.2.7 {worktree}/web<br><br> ✓ src/i18n/i18n.test.ts (7 tests) 10ms<br><br> Test Files  1 passed (1)<br>      Tests  7 passed (7)<br>   Start at  15:13:41<br>   Duration  661ms (transform 183ms, setup 0ms, collect 259ms, tests 10ms, environment 0ms, prepare 146ms) |
| Build + embed (`node scripts/build.mjs`) | yes | PASS | 1625ms | Nexus web build complete: web/dist<br>≈ tailwindcss v4.3.3<br><br>Done in 218ms<br><br>  dist\bundle.js                                  374.4kb<br>  dist\chunks\chunk-OTEGMETV.js                   348.8kb<br>  dist\chunks\chunk-J3B5NHJN.js                   280.3kb<br>  dist\chunks\chunk-BOCAUZ5C.js                   225.4kb<br>  dist\chunks\FlowRunsHistorySurface-X7KPHEGA.js  221.3kb<br>  ...and 58 more output files...<br><br>Done |
| Embed sync (web/dist ≡ internal/.../embedded) | yes | PASS | 91ms | bundles idênticos (383381 bytes) |
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
