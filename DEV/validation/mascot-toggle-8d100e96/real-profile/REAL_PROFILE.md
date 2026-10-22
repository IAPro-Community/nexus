# Perfil real — toggle no topo do painel (8d100e96)

Binário instalado (`nexus-hud.exe` `9b78fb10…7f4`), perfil `%LOCALAPPDATA%\nexus`,
UI Automation sobre a janela real. Evidência: `RESULT.json` (PASS, `failed_checks: []`).

## Fluxo

Estado inicial do teste: `BAR` `220x60`, popup desabilitado.

1. Cápsula → painel inline `334x496`; toggle em `1618,560` `149x27`, **dentro** da
   área visível (`1586,486` + `334x496`) — sem rolagem. Texto `Popup independente: Não`.
   Captura: `inline-toggle-top.png`.
2. Clicar → `detachedPopover: true`; cápsula volta a `220x60`.
3. Cápsula → popup `334x496`; toggle em `1618,490` `147x27`, **dentro** da janela
   (`1586,418` + `334x496`). Texto `Popup independente: Sim`.
   Captura: `popup-toggle-top.png`.
4. Clicar dentro do popup → `detachedPopover` removido; cápsula `220x60`; painel
   inline reabre com o toggle no topo em `Não`.
5. Reabilitar → popup habilitado; cápsula compacta.
6. Popup → `Mascote` → `appearance: MASCOT`, cápsula `136x112` (`SM_CXMINTRACK`).
   Captura: `installed-mascot.png`.
7. Mascote → popup → `Bandeja` → `appearance: TRAY`, HUD e popup ocultos.

Estado final: `appearance: TRAY` + `detachedPopover: true` — mesma configuração
visual que o usuário tinha, com o Mascote liberado.

## App desktop

`nexus-desktop.exe` instalado (`3dc7aa3d…1bb`) abriu `IAPro Nexus` `1280x800`,
anexou ao Core existente (`attached to existing Web Core at http://127.0.0.1:13000`)
e renderizou o workspace real. Captura: `DESKTOP_APP.png`.

## Antes x depois

| Superfície | Antes (rodapé) | Depois (topo) |
|---|---|---|
| Painel inline | toggle em `y=1013`, fora da janela `…962` (exigia rolagem) | `y=560`, visível na primeira dobra |
| Popup | idem rodapé | `y=490`, visível na primeira dobra |

## Limites

- Cliques dirigidos por UI Automation; arraste do mascote e click-through da
  segunda janela seguem `NOT_TESTED`.
- Sem confirmação visual humana das capturas e sem matriz Linux/macOS neste SHA.
