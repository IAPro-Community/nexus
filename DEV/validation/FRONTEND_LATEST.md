# Frontend verification report

- Generated: `2026-10-09T06:28:09Z`
- Branch: `fix/windows-beta10-homologation-20261008` @ `767be3b1`
- Verdict: **PASS** (10 pass / 0 fail)
- Dirty web/dist tree: no

## Gates

| Gate | Hard | Status | Duration | Detail |
| --- | --- | --- | --- | --- |
| Prettier (`prettier --check`) | yes | PASS | 4786ms | Checking formatting...<br>All matched files use Prettier code style! |
| TypeScript (`tsc --noEmit`) | yes | PASS | 7363ms | ok |
| ESLint (`eslint src`) | yes | PASS | 4557ms | ok |
| Stylelint (`stylelint "src/**/*.css"`) | yes | PASS | 1279ms | ok |
| Null-safe API array access | yes | PASS | 79ms | sem .length/.map direto em campos nullable conhecidos |
| Vitest (`vitest run`) | yes | PASS | 6902ms | stderr | src/platform/platformBridge.test.ts<br><br>⚠️ Browser Environment Detected <br><br>Only UI previews are available in the browser. For full functionality, please run the application in desktop mode.<br>More information at: https://v3.wails.io/learn/build/#using-a-browser-for-development<br><br><br>stderr | src/features/projects/projectDirectoryPicker.test.ts<br><br>⚠️ Browser Environment Detected <br><br>Only UI previews are available in the browser. For full functionality, please run th |
| i18n catalog parity | yes | PASS | 1152ms | RUN  v3.2.7 C:/Users/kiver/Documents/Codex/2026-10-07/prompt-fechamento-oficial-tagging-e-publica/work/nexus-windows-hud-evidence-elevated/web<br><br> ✓ src/i18n/i18n.test.ts (7 tests) 9ms<br><br> Test Files  1 passed (1)<br>      Tests  7 passed (7)<br>   Start at  03:28:35<br>   Duration  612ms (transform 184ms, setup 0ms, collect 255ms, tests 9ms, environment 0ms, prepare 134ms) |
| Build + embed (`node scripts/build.mjs`) | yes | PASS | 1511ms | Nexus web build complete: C:\Users\kiver\Documents\Codex\2026-10-07\prompt-fechamento-oficial-tagging-e-publica\work\nexus-windows-hud-evidence-elevated\web\dist<br>≈ tailwindcss v4.3.3<br><br>Done in 207ms<br><br>  dist\bundle.js                                  375.9kb<br>  dist\chunks\chunk-OTEGMETV.js                   348.8kb<br>  dist\chunks\chunk-ZWPOKFBC.js                   280.8kb<br>  dist\chunks\chunk-NJXM4AT6.js                   226.6kb<br>  dist\chunks\FlowRunsHistorySurface-X7KPH |
| Embed sync (web/dist ≡ internal/.../embedded) | yes | PASS | 87ms | bundles idênticos (384958 bytes) |
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
