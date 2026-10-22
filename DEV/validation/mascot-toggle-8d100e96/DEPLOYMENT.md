# Implantação local — 8d100e96

Alvo: `%LOCALAPPDATA%\Programs\IAPro Nexus` (Windows 11, usuário `kiver`).

## Artefatos

Build `production`, Go `1.26.10`, `-trimpath -buildvcs=false`, versão `1.0.0-beta.10`.

| Arquivo | SHA-256 |
|---|---|
| `nexus.exe` | `326a95a8bbb18904be13a86667cbf1f81f2e2ae9c4a89e48d16a29682126dfd9` |
| `ai.exe` (cópia) | `326a95a8bbb18904be13a86667cbf1f81f2e2ae9c4a89e48d16a29682126dfd9` |
| `nexus-desktop.exe` | `3dc7aa3d543f598205c42ce2740dad59e58696a993471258d2e51288772441bb` |
| `nexus-hud.exe` | `9b78fb107248e09102aed76baf238342fc2990f11ae99ae1648b27f1d60f17f4` |

Hashes do staging conferidos antes da cópia e dos arquivos instalados depois da
cópia (todos `match=True`).

## Runtime

- Servidor: PID `15516`, `http://127.0.0.1:13000/api/v1/health` → **200**.
- `nexus.exe version --json` → `commit: 8d100e96`, `go: go1.26.10`, `version: 1.0.0-beta.10`,
  `build_date: 2026-10-22T04:04:23Z`.
- HUD: PID `8304`, `--linked`.
- Estado do usuário preservado: `appearance: TRAY`, `detachedPopover: true`
  (HUD oculto na bandeja, Mascote liberado).

## Backup e rollback

- Backup: `%TEMP%\nexus-backup-20261022-010547` (binários anteriores, `0883e2d8`-era
  instalada a partir de `955b197f`).
- Rollback: parar `nexus.exe`/`nexus-hud.exe`, copiar os `.exe` do backup de volta
  para a pasta de instalação e reiniciar `nexus.exe` (servidor) e
  `nexus-hud.exe --linked`.
