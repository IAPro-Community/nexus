# Implantação local — `release/v1.0.0` @ `4659ced7` — 2026-10-21

## Origem

- Branch: `release/v1.0.0`.
- Commit implantado: `4659ced7706a755003b61466c51a45e3a39df315` (código `047f4348` + commit de docs/evidência).
- Toolchain: Go `1.26.10` (`GOTOOLCHAIN`), `-trimpath -buildvcs=false`.
- `nexus-desktop.exe` e `nexus-hud.exe` com tag `production`; os três binários com ldflags de versão/commit/data.

## Artefatos instalados

Diretório: `%LOCALAPPDATA%\Programs\IAPro Nexus`.
Backup dos binários anteriores: `%TEMP%\nexus-backup-20261021-232756`.

| Arquivo | SHA-256 instalado | Confere com o build |
|---|---|---|
| `nexus.exe` | `bcc070caf732e66469283c67b39d144466644a8e3949903b4c2c8cdcd9c38a7c` | sim |
| `ai.exe` | `bcc070caf732e66469283c67b39d144466644a8e3949903b4c2c8cdcd9c38a7c` | sim (cópia de `nexus.exe`) |
| `nexus-desktop.exe` | `afea475567f31d2e81222385d2a1e4c92db66a559d3646a3bd9482cbfa345119` | sim |
| `nexus-hud.exe` | `e9425d5656ca70df11d7a08e1ad42da82e1f0531964bfadf4d6384d2b84fe736` | sim |

Processos substituídos: `nexus.exe` PID 18236 (servidor na 13000) e PID 21264 (`__control-host`). O arquivo `.nexus-managed` não foi alterado.

## Verificação pós-implantação

- Servidor: PID `9244`, `http://127.0.0.1:13000/` responde **HTTP 200**.
- `nexus.exe version --json`:

```json
{
  "arch": "amd64",
  "build_date": "2026-10-22T02:27:37Z",
  "commit": "4659ced",
  "go": "go1.26.10",
  "os": "windows",
  "version": "1.0.0-beta.10"
}
```

- HUD: PID `26548`, `nexus-hud.exe --linked`, SHA-256 `e9425d56...fe736`.
- Click-through no **binário instalado**: `deployed-click-through/RESULT.json` — `verdict: PASS`, `transparent_corner_click_through: true`, `capsule_click_through: false`, `click_through_restored: true`, `foreground_after: Nexus ClickTarget`.

## Observações

- O HUD em modo `--linked` inicia com a janela oculta e é exibido pela aplicação/ícone de bandeja. O binário anterior (backup, 2026-10-18) apresenta o mesmo comportamento neste host — não é regressão do build novo.
- O HUD de teste do harness usa estado isolado; os testes não escreveram no estado do usuário.
- `docs-verify` continua PASS (192 Markdown / 33 assets) e o manifesto visual aponta para `047f4348`.

## Rollback

1. Encerrar `nexus.exe` e `nexus-hud.exe`.
2. Copiar de volta `nexus.exe`, `ai.exe`, `nexus-desktop.exe` e `nexus-hud.exe` de `%TEMP%\nexus-backup-20261021-232756`.
3. Iniciar `nexus.exe` no diretório de instalação e confirmar HTTP 200 em `127.0.0.1:13000`.
