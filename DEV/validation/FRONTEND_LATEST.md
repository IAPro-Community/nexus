# Frontend verification report

- Generated: `2026-10-08T23:40:38Z`
- Branch: `fix/windows-beta10-homologation-20261008` @ `cd5302f9`
- Verdict: **PASS** (10 pass / 0 fail)
- Dirty web/dist tree: no

## Gates

| Gate | Hard | Status | Duration | Detail |
| --- | --- | --- | --- | --- |
| Prettier (`prettier --check`) | yes | PASS | 5131ms | Checking formatting...<br>All matched files use Prettier code style! |
| TypeScript (`tsc --noEmit`) | yes | PASS | 7890ms | ok |
| ESLint (`eslint src`) | yes | PASS | 4768ms | ok |
| Stylelint (`stylelint "src/**/*.css"`) | yes | PASS | 1348ms | ok |
| Null-safe API array access | yes | PASS | 82ms | sem .length/.map direto em campos nullable conhecidos |
| Vitest (`vitest run`) | yes | PASS | 7840ms | stderr | src/platform/platformBridge.test.ts<br><br>⚠️ Browser Environment Detected <br><br>Only UI previews are available in the browser. For full functionality, please run the application in desktop mode.<br>More information at: https://v3.wails.io/learn/build/#using-a-browser-for-development<br><br><br>stderr | src/features/projects/projectDirectoryPicker.test.ts<br><br>⚠️ Browser Environment Detected <br><br>Only UI previews are available in the browser. For full functionality, please run th |
| i18n catalog parity | yes | PASS | 1239ms | RUN  v3.2.7 C:/Users/kiver/AppData/Local/Temp/nexus-windows-release-95c7121d/web<br><br> ✓ src/i18n/i18n.test.ts (7 tests) 10ms<br><br> Test Files  1 passed (1)<br>      Tests  7 passed (7)<br>   Start at  20:41:06<br>   Duration  683ms (transform 179ms, setup 0ms, collect 258ms, tests 10ms, environment 0ms, prepare 124ms) |
| Build + embed (`node scripts/build.mjs`) | yes | PASS | 1700ms | Nexus web build complete: C:\Users\kiver\AppData\Local\Temp\nexus-windows-release-95c7121d\web\dist<br>≈ tailwindcss v4.3.3<br><br>Done in 214ms<br><br>  dist\bundle.js                                  374.4kb<br>  dist\chunks\chunk-T4ZRJ2ZU.js                   348.8kb<br>  dist\chunks\chunk-J3B5NHJN.js                   280.3kb<br>  dist\chunks\chunk-7XSBLDJ4.js                   224.9kb<br>  dist\chunks\FlowRunsHistorySurface-6LARTTT4.js  221.3kb<br>  ...and 58 more output files...<br><br>Don |
| Embed sync (web/dist ≡ internal/.../embedded) | yes | PASS | 91ms | bundles idênticos (383345 bytes) |
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
