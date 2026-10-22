# Implantação local — `release/v1.0.0` @ `955b197f` — 2026-10-22

## Origem

- Branch: `release/v1.0.0`, commit implantado `955b197fd4960a6713dc224640044f35ab740461` (código `f789db63` + commit de docs/evidência).
- Toolchain: Go `1.26.10` (`GOTOOLCHAIN`), `-trimpath -buildvcs=false`, ldflags de versão/commit/data.
- `nexus-desktop.exe` e `nexus-hud.exe` com tag `production`; `nexus.exe`/`ai.exe` sem tag, como nas implantações anteriores.

## Artefatos instalados

Diretório: `%LOCALAPPDATA%\Programs\IAPro Nexus`.
Backup dos binários anteriores: `%TEMP%\nexus-backup-20261022-003138`.

| Arquivo | SHA-256 instalado | Confere com o build |
|---|---|---|
| `nexus.exe` | `f8cd64d5d52155e714346394581ec4fa65cfe8652176aa6a4b17649ca13cffde` | sim |
| `ai.exe` | `f8cd64d5d52155e714346394581ec4fa65cfe8652176aa6a4b17649ca13cffde` | sim (cópia de `nexus.exe`) |
| `nexus-desktop.exe` | `72b3cc50bd45e655f5c328ecb4b3263dbc71284bcccc51a8caf14c26cefc310e` | sim |
| `nexus-hud.exe` | `8364a94222fb1235228ea9ba4b4308ae1aeb895aa31970f45becd08091eb4065` | sim |

Processo anterior substituído: `nexus.exe` PID 9244 (servidor na 13000). O arquivo `.nexus-managed` não foi alterado.

## Verificação pós-implantação

- Servidor: PID `24184`, `http://127.0.0.1:13000/` responde **HTTP 200**.
- `nexus.exe version --json`:

```json
{
  "arch": "amd64",
  "build_date": "2026-10-22T03:30:33Z",
  "commit": "955b197f",
  "go": "go1.26.10",
  "os": "windows",
  "version": "1.0.0-beta.10"
}
```

- HUD: PID `21168`, `nexus-hud.exe --linked`, SHA-256 `8364a942...4065` (idêntico ao artefato homologado).
- Estado real do usuário em `%LOCALAPPDATA%\nexus\hud-window-state.json`: `appearance: TRAY` (HUD recolhido na bandeja) e `detachedPopover` ausente ⇒ popup desligado. É exatamente o cenário que o novo toggle resolve: ao abrir o painel, o botão mostra `Popup independente: Não` e o Mascote aparece desabilitado até a ativação.

## Evidência no artefato exato

O harness `scripts/homologate-hud-mascot-toggle-windows.ps1` rodou **antes da cópia** contra o binário `nexus-hud.exe` SHA-256 `8364a94222fb1235228ea9ba4b4308ae1aeb895aa31970f45becd08091eb4065`, com `ExpectedSha` e commit `955b197f`:

- `deployed-artifact/RESULT.json` → `verdict: PASS`, `failed_checks: []`.
- Cobre: toggle no painel, criação do popup, restauração do tamanho compacto, Mascote habilitado e aplicado (`136x112`), reinício com preferência persistida, desativação dentro do popup, volta para `BAR` e reabertura do painel inline.

O instalado é byte-a-byte o mesmo arquivo (SHA-256 confere), portanto a evidência se aplica ao artefato implantado. A execução não foi repetida contra o processo instalado para não encerrar a sessão `--linked` do usuário; o HUD do harness usa estado isolado e não tocou no estado real.

## Rollback

1. Encerrar `nexus.exe` e `nexus-hud.exe`.
2. Copiar de volta `nexus.exe`, `ai.exe`, `nexus-desktop.exe` e `nexus-hud.exe` de `%TEMP%\nexus-backup-20261022-003138`.
3. Iniciar `nexus.exe` no diretório de instalação e confirmar HTTP 200 em `127.0.0.1:13000`.
4. Se necessário, remover `detachedPopover` de `%LOCALAPPDATA%\nexus\hud-window-state.json` com os processos encerrados.
