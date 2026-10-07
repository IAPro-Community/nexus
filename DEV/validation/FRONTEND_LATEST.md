# Frontend verification report

- Generated: `2026-10-07T12:26:27Z`
- Branch: `release/v1.0.0` @ `2892f301`
- Verdict: **FAIL** (9 pass / 1 fail)
- Dirty web/dist tree: no

## Gates

| Gate | Hard | Status | Duration | Detail |
| --- | --- | --- | --- | --- |
| Prettier (`prettier --check`) | yes | FAIL | 5251ms | exit 1<br>[warn] src/notifications/attentionPushCopy.ts<br>[warn] src/notifications/capacityAlert.test.ts<br>[warn] src/notifications/InAppNotificationCenter.tsx<br>[warn] src/notifications/inAppNotificationModel.test.ts<br>[warn] src/notifications/inAppNotificationModel.ts<br>[warn] src/notifications/notificationModel.test.ts<br>[warn] src/notifications/notificationModel.ts<br>[warn] src/notifications/notificationPrefs.ts<br>[warn] src/notifications/PushNotificationManager.ts<br>[warn] src/plat |
| TypeScript (`tsc --noEmit`) | yes | PASS | 8790ms | ok |
| ESLint (`eslint src`) | yes | PASS | 7983ms | ok |
| Stylelint (`stylelint "src/**/*.css"`) | yes | PASS | 1569ms | ok |
| Null-safe API array access | yes | PASS | 92ms | sem .length/.map direto em campos nullable conhecidos |
| Vitest (`vitest run`) | yes | PASS | 7530ms | stderr | src/platform/platformBridge.test.ts<br><br>⚠️ Browser Environment Detected <br><br>Only UI previews are available in the browser. For full functionality, please run the application in desktop mode.<br>More information at: https://v3.wails.io/learn/build/#using-a-browser-for-development<br><br><br>stderr | src/features/projects/projectDirectoryPicker.test.ts<br><br>⚠️ Browser Environment Detected <br><br>Only UI previews are available in the browser. For full functionality, please run th |
| i18n catalog parity | yes | PASS | 1255ms | RUN  v3.2.7 C:/Users/kiver/nexus-beta7-windows-validation/web<br><br> ✓ src/i18n/i18n.test.ts (7 tests) 10ms<br><br> Test Files  1 passed (1)<br>      Tests  7 passed (7)<br>   Start at  09:26:59<br>   Duration  697ms (transform 186ms, setup 0ms, collect 272ms, tests 10ms, environment 0ms, prepare 111ms) |
| Build + embed (`node scripts/build.mjs`) | yes | PASS | 1717ms | Nexus web build complete: C:\Users\kiver\nexus-beta7-windows-validation\web\dist<br>≈ tailwindcss v4.3.3<br><br>Done in 210ms<br><br>  dist\bundle.js                                  372.5kb<br>  dist\chunks\chunk-XKZL7LFT.js                   348.8kb<br>  dist\chunks\chunk-KWCMGYPG.js                   280.3kb<br>  dist\chunks\chunk-FGAY72LC.js                   224.6kb<br>  dist\chunks\FlowRunsHistorySurface-Q3PSER6E.js  221.3kb<br>  ...and 58 more output files...<br><br>⚡ Done in 848ms |
| Embed sync (web/dist ≡ internal/.../embedded) | yes | PASS | 93ms | bundles idênticos (381488 bytes) |
| Critical UI markers in bundle | yes | PASS | 3ms | marcadores críticos presentes (7) |

## Residual risks / next operator steps

- Hard gates failed — do not claim frontend delivery until green.
  - Fix `format` then re-run `make web-verify`.

### Notes

- Binário `nexus` local não encontrado neste check — após PASS, rode `make build` para instalar.

## How to regenerate

```bash
make web-verify
# or
npm --prefix web run verify
```
