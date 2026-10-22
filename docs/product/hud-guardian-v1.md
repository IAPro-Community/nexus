# Nexus Guardian — apresentação nativa do HUD na release 1.0

## Implementação

O Guardian não substitui o Nexus Core nem os cálculos de Capacity,
ResourceScheduler, Agents, Missions ou Cost Intelligence. É uma apresentação
alternativa dos dados existentes, com três escolhas persistidas no backend:

- **Barra:** cápsula tradicional, com alça de arraste e tamanhos existentes.
- **Mascote:** pequeno personagem robótico 2D em pixel art vetorial, dentro de
  uma superfície nativa aproximada de 108 × 112 DIPs. Clicar abre o popup de
  detalhes, enquanto uma alça separada permite arrastar a janela.
- **Bandeja:** oculta a cápsula; o processo continua ativo. O menu
  nativo do Nexus HUD permite **Mostrar**, **Ocultar** e **Sair**.

As escolhas são gravadas em `hud-window-state.json` por
`HUDController.SetHUDAppearance`; `Mode`, que controla a prioridade
das informações exibidas, permanece independente.

## Opt-in e fallback

O popup independente permanece **opt-in** enquanto não houver
homologação nativa completa. É possível ativá-lo diretamente no painel
expandido, pela opção **Popup independente: Sim**. Essa preferência persiste
sem precisar reiniciar. Após a ativação, o seletor **Barra / Mascote / Bandeja**
fica disponível.

O menu da bandeja também permite escolher **Guardian: Mascote** em Windows,
macOS e Linux X11, ativando o popup independente sob demanda. Wayland mantém
a Barra e desabilita a opção de popup independente.

Para testes e automação ainda é possível usar a variável de ambiente:

```bash
NEXUS_HUD_DETACHED_POPOVER=1 ./cmd/nexus-hud/build/bin/nexus-hud
```

Em PowerShell:

```powershell
$env:NEXUS_HUD_DETACHED_POPOVER = '1'
.\cmd\nexus-hud\build\bin\nexus-hud.exe
```

No painel expandido, os controles **Barra / Mascote / Bandeja**
alteram a escolha imediatamente e persistem após reinício. A bandeja
nativa disponibiliza um caminho de recuperação quando a interface está
oculta, inclusive após escolher **Somente bandeja**.

O modo Mascote **não pode ser selecionado** sem o popup independente.
Quando esta opção não estiver habilitada (ou quando a sessão for
Linux Wayland), uma aparência de Mascote salva anteriormente é
convertida para Barra para não abrir uma janela larga cortada.

A Bandeja pode ser selecionada mesmo no HUD clássico. Após ocultar,
é possível recuperar o HUD clicando no ícone da bandeja ou escolhendo
**Mostrar**. O programa continua em modo Bandeja no próximo boot
até que outra aparência seja escolhida.

## Animações de caráter exclusivamente visual

O personagem é desenhado em SVG com pixels geométricos e CSS Modules.
Não depende de vídeos, assets externos, WebGL ou URLs remotas.

| Estado | Fonte | Comportamento |
| --- | --- | --- |
| Ocioso | Sem agentes trabalhando, sem intervenção | Flutuação lenta e piscada |
| Trabalhando | Agentes em status WORKING | Movimento curto de trabalho e sinal acelerado |
| Atenção | Alertas/mission que exigem atenção | Balanço breve e balão `Atenção!` |
| Offline | Projeção sem disponibilidade confirmada | Visual dessaturado, sem simular atividade |

Em `prefers-reduced-motion: reduce`, todos os movimentos do personagem,
alerta e painel ficam desativados. Os estados continuam identificáveis pela
cor/forma e pelo texto acessível. Não anunciar alerta a partir de quota
desconhecida e não consultar os provedores para gerar movimentos.

## Arquitetura

- `internal/desktop/hud_appearance.go`: API nativa de apresentação.
- `internal/desktop/hud_window.go`: restauração, geometria e persistência.
- `cmd/nexus-hud/window_options.go`: janela compacta, transparência e início
  oculto quando configurado para Bandeja.
- `web/src/features/hud/GuardianMascot.tsx`: personagem e controles.
- `web/src/features/hud/_hudGuardian.scss`: movimentos e acessibilidade.
- `web/src/features/hud/guardianModel.ts`: seleção do estado a partir de
  telemetria já existente.
- `web/src/features/hud/NexusHud.tsx`: troca de apresentação e popup.
- `web/src/platform/wailsV3.ts`: ponte Wails.

## Por que o mascote pode não aparecer em uma versão instalada

A branch `release/v1.0.0` **não é** o mesmo objeto que os executáveis
já publicados. Cada `preview-<sha>` é uma cópia imutável do commit informado
em sua página; a atualização da branch não recompila automaticamente os
ZIPs de previews anteriores nem atualiza o canal de release assinada.

Para identificar um pacote, confira `SOURCE_SHA.txt` no ZIP Windows ou
TAR Linux. No macOS, compare a página da prévia e o SHA de origem de seus
artefatos; `nexus-hud` é um aplicativo separado de `nexus-desktop`.
A prévia precisa incluir o commit que acrescentou
`GuardianMascot.tsx` e `internal/desktop/hud_appearance.go`.

**Onde escolher:** execute `nexus-hud` da compilação atual e abra a
cápsula. Em Windows/macOS/Linux X11, ative
**Popup independente** no painel expandido e selecione **Mascote**.
Também é possível escolher **Guardian: Mascote** no menu de bandeja.
Wayland mantém o fallback Barra. O Mascote não aparece automaticamente no
`nexus-desktop` nem na página Web.

A publicação de uma nova prévia não comprova homologação gráfica ou
assinatura; use apenas pacotes do mesmo SHA e mantenha os gates Windows,
macOS e Linux antes de promover a versão distribuída.

## Homologação obrigatória antes de liberar por padrão

1. Windows e macOS: alternar 20 vezes Barra → Mascote → Barra; verificar
   que a janela usa dimensões compactas, não há sombra quadrada e os cliques
   continuam funcionando.
2. Linux X11 e, separadamente, Wayland: verificar transparência real e
   fallback; uma sessão X11 não certifica Wayland.
3. Bandeja: selecionar, reiniciar, recuperar pelo ícone, expandir o painel,
   escolher Barra/Mascote, fechar por Escape e por clique externo.
4. Arraste: usar alça do Mascote e da Barra, teclas de direção,
   várias escalas DPI, monitores múltiplos e desconexão de monitor.
5. Animar: capturar vídeo real das sequências ocioso, trabalhando, atenção,
   offline e `prefers-reduced-motion`; não aceitar flicker, popup em branco
   ou disparos infinitos de avisos.
6. Click-through: clicar em pontos vazios do retângulo nativo com outro
   aplicativo atrás. O Windows possui um watcher experimental de regiões
   interativas na cápsula, inclusive no modo popup independente, mas sua
   eficácia ainda exige vídeo/capturas reais em monitor/DPI diferentes.
   macOS e Linux ainda carecem de um adaptador de hit-test equivalente.
   Se o WebView/Wails bloquear o clique, registrar NO_GO.
7. Executar testes Go/TS, lint, formatação, build e os verificadores nativos
   no **mesmo SHA**. Não reutilizar capturas de commits anteriores.

**Estado de certificação:** implementação de código, testes e documentação
não substituem execução gráfica real. Não há declaração de GO cross-platform
nem liberação automática do novo modo nesta etapa.
