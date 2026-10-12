# Frontend verification report

- Generated: `2026-10-12T22:00:18Z`
- Branch: `fix/windows-beta10-homologation-20261008` @ `2c8e9ef4`
- Verdict: **PASS** (10 pass / 0 fail)
- Dirty web/dist tree: no

## Gates

| Gate | Hard | Status | Duration | Detail |
| --- | --- | --- | --- | --- |
| Prettier (`prettier --check`) | yes | PASS | 7241ms | Checking formatting...<br>All matched files use Prettier code style! |
| TypeScript (`tsc --noEmit`) | yes | PASS | 12352ms | ok |
| ESLint (`eslint src`) | yes | PASS | 6000ms | ok |
| Stylelint (`stylelint "src/**/*.css"`) | yes | PASS | 1617ms | ok |
| Null-safe API array access | yes | PASS | 93ms | sem .length/.map direto em campos nullable conhecidos |
| Vitest (`vitest run`) | yes | PASS | 9709ms | stderr | src/platform/platformBridge.test.ts<br><br>⚠️ Browser Environment Detected <br><br>Only UI previews are available in the browser. For full functionality, please run the application in desktop mode.<br>More information at: https://v3.wails.io/learn/build/#using-a-browser-for-development<br><br><br>stderr | src/features/projects/projectDirectoryPicker.test.ts<br><br>⚠️ Browser Environment Detected <br><br>Only UI previews are available in the browser. For full functionality, please run th |
| i18n catalog parity | yes | PASS | 1740ms | RUN  v3.2.7 C:/Users/kiver/Documents/Codex/2026-10-07/prompt-fechamento-oficial-tagging-e-publica/work/nexus-windows-hud-evidence-elevated/web<br><br> ✓ src/i18n/i18n.test.ts (7 tests) 13ms<br><br> Test Files  1 passed (1)<br>      Tests  7 passed (7)<br>   Start at  19:00:55<br>   Duration  926ms (transform 251ms, setup 0ms, collect 359ms, tests 13ms, environment 0ms, prepare 238ms) |
| Build + embed (`node scripts/build.mjs`) | yes | PASS | 2321ms | Nexus web build complete: C:\Users\kiver\Documents\Codex\2026-10-07\prompt-fechamento-oficial-tagging-e-publica\work\nexus-windows-hud-evidence-elevated\web\dist<br>≈ tailwindcss v4.3.3<br><br>Done in 325ms<br><br>  dist\bundle.js                                  375.9kb<br>  dist\chunks\chunk-OTEGMETV.js                   348.8kb<br>  dist\chunks\chunk-ZWPOKFBC.js                   280.8kb<br>  dist\chunks\chunk-NJXM4AT6.js                   226.6kb<br>  dist\chunks\FlowRunsHistorySurface-X7KPH |
| Embed sync (web/dist ≡ internal/.../embedded) | yes | PASS | 110ms | bundles idênticos (384958 bytes) |
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
