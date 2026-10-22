# Verificação no perfil real do usuário — 2026-10-22

Complementa `HOMOLOGATION.md` (estado isolado) e `DEPLOYMENT.md` (instalação). Aqui o alvo é o **ambiente real**: o binário instalado, o perfil `%LOCALAPPDATA%\nexus` do usuário e o host em uso.

## Ambiente

- Instalação: `%LOCALAPPDATA%\Programs\IAPro Nexus`, commit `955b197f`, `nexus-hud.exe` SHA-256 `8364a94222fb1235228ea9ba4b4308ae1aeb895aa31970f45becd08091eb4065`.
- Servidor: PID `24184`, `http://127.0.0.1:13000/api/v1/health` → **200** `{"status":"ok","version":"1.0.0-beta.10"}`; rotas protegidas sem sessão → **401** `AUTH_REQUIRED` (comportamento esperado).
- Estado inicial do usuário: `appearance: TRAY` (HUD recolhido na bandeja), sem `detachedPopover`.
- Escala 100%, janela compacta `220x60`.

## Fluxo executado (UI Automation, sem mocks)

1. Exibir a janela da bandeja (`nexus-hud.exe` reabre a instância única e mostra o HUD).
2. Abrir o painel e rolar até o rodapé: **`Popup independente: Não`** encontrado e visível.
3. Clicar no toggle → `detachedPopover: true` persistido; popup criado **oculto**; cápsula permanece `220x60`.
4. Reabrir: popup visível em `334x496`, cápsula ainda `220x60` (sem janela transparente sobrando).
5. **`Mascote`** habilitado no popup → `Appearance = MASCOT`; cápsula `136x112` renderizando o mascote (limite `SM_CXMINTRACK = 136`; altura exata).
6. Reabrir pelo mascote e escolher **`Bandeja`** → `Appearance = TRAY`, HUD e popup ocultos.

Resultado: `RESULT.json` → `verdict: PASS`, `failed: []`. Estado final: `appearance: TRAY` + `detachedPopover: true` (sem mudança visual para o usuário, com o Mascote já liberado).

Capturas: `PANEL_WITH_TOGGLE.png` (rodapé do painel com o toggle) e `REAL_MASCOT.png` (cápsula em modo mascote no host).

## Achado de usabilidade

O toggle fica no **rodapé** do painel. Em perfis com vários provedores (Agy, codex, gemini, opencode) o conteúdo passa da altura de `496` DIP e é preciso rolar até o fim para alcançá-lo. O seletor de modo no topo já explica o gate no tooltip ("Ative \"Popup independente\" nas configurações do HUD para usar o mascote"). Funciona, mas a descoberta depende da rolagem.

## App desktop

`nexus-desktop.exe` (instalado) iniciado no host: anexou ao Core já em execução (`attached to existing Web Core at http://127.0.0.1:13000`), abriu a janela `IAPro Nexus` `1280x800` e renderizou o workspace real `games-tvbox-secondplay-qa` (127 elementos na árvore UIA, sem erro no stderr além do log informativo do WebView2). Encerrado em seguida; servidor e HUD permaneceram saudáveis.

Captura: `DESKTOP_APP.png`.

## Limites

- Sem confirmação visual humana das capturas e sem matriz Linux/macOS.
- O clique/toque foi dirigido por UI Automation; o arraste do mascote e o click-through da segunda janela continuam `NOT_TESTED`.
- Release segue **NO_GO_WINDOWS** (assinatura/instalador e smokes interativos pendentes).
