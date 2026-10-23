# NEX — mascote oficial do Nexus no HUD nativo da release 1.0

## Implementação

O Guardian não substitui o Nexus Core nem os cálculos de Capacity,
ResourceScheduler, Agents, Missions ou Cost Intelligence. É uma apresentação
alternativa dos dados existentes, com três escolhas persistidas no backend:

- **Barra:** cápsula tradicional, com alça de arraste e tamanhos existentes.
- **Mascote:** NEX, mascote oficial frontal em pixel art vetorial 32-bit-inspired (visor expressivo, duas aletas, núcleo luminoso e módulos orbitais), dentro de
  uma superfície nativa aproximada de 108 × 112 DIPs. Clicar abre o popup de
  detalhes, enquanto uma alça separada permite arrastar a janela.
- **Bandeja:** oculta a cápsula; o processo continua ativo. O menu
  nativo do Nexus HUD permite **Mostrar**, **Ocultar** e **Sair**.

As escolhas são gravadas em `hud-window-state.json` por
`HUDController.SetHUDAppearance`; `Mode`, que controla a prioridade
das informações exibidas, permanece independente.

## Apresentação padrão, preferências e Wayland

**O Mascote é padrão na instalação nova**, sem variáveis de ambiente.
O detalhe em segunda janela também é a configuração inicial preferida em
Windows, macOS e Linux X11. A aparência (Mascote, Barra, Bandeja) pode
ser alterada no popup ou no menu nativo da bandeja e persiste no arquivo
`hud-window-state.json`. Uma escolha explícita anterior do usuário por
Barra ou Bandeja continua sendo respeitada.

**Popup independente é preferência de layout, não pré-requisito do Mascote.**
O usuário pode desativá-lo sem perder o personagem. O Linux Wayland usa
a expansão dentro da própria janela, porque o compositor pode impedir
posicionar uma segunda janela com coordenadas globais. O Mascote permanece
como apresentação padrão também no Wayland.

O menu nativo permite recuperar a janela quando está oculta na Bandeja;
a seleção de uma aparência diferente não exige reinicialização.

## Animações de caráter exclusivamente visual

O NEX utiliza o sprite canônico em `web/src/features/hud/NexPixelSprite.tsx`, desenhado em SVG com pixels geométricos e CSS Modules. O antigo robô retangular genérico foi substituído na camada visual; o componente de integração `GuardianMascot.tsx`, suas props, a apresentação BAR/MASCOT/TRAY e o Core são preservados.
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
- `web/src/features/hud/GuardianMascot.tsx`: integração do personagem e controles.
- `web/src/features/hud/NexPixelSprite.tsx`: arte canônica Compact de NEX 32-bit-inspired, renderizada internamente.
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

**Onde escolher:** execute o `nexus-hud` compilado a partir do
commit atualizado. O Mascote aparece por padrão. Clique nele para abrir
detalhes e use os controles Barra / Mascote / Bandeja. A bandeja também
oferece diretamente as três apresentações. No Wayland o painel abre na
mesma janela; o personagem não é substituído pela Barra automaticamente.

A publicação de uma nova prévia não comprova homologação gráfica ou
assinatura; use apenas pacotes do mesmo SHA e mantenha os gates Windows,
macOS e Linux antes de promover a versão distribuída.

O workflow `unsigned-preview.yml` exige opt-in explícito e roda as suítes
Go/frontend antes de publicar. Em caso de reprovação não há novos binários:
a última prévia permanece anterior ao Guardian.

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
   No macOS, o adaptador Quartz fica experimental e **desativado por
   padrão** (opt-in `NEXUS_HUD_MAC_CLICKTHROUGH=1`), porque ainda precisa
   de validação com Retina, múltiplos monitores e clique real atrás da janela.
   O Linux continua sem click-through nativo: Wails não implementa
   `SetIgnoreMouseEvents` nessa plataforma. Em X11 será necessário
   um adaptador de regiões de entrada do compositor/GTK; em Wayland, validar
   um fallback limitado sem prometer coordenadas globais.
   Qualquer bloqueio de clique no WebView continua NO_GO.
7. Executar testes Go/TS, lint, formatação, build e os verificadores nativos
   no **mesmo SHA**. Não reutilizar capturas de commits anteriores.

**Estado de certificação:** implementação de código, testes e documentação
não substituem execução gráfica real. Não há declaração de GO cross-platform
nem liberação automática do novo modo nesta etapa.

## Identidade oficial NEX 32 bits (23/10/2026)

A atualização do desenho é exclusivamente de apresentação: ela não adiciona novos moods nem modifica os estados canônicos `idle`, `working`, `warning` e `offline`. O símbolo/logo institucional do Nexus permanece inalterado. O sprite usa `viewBox=128x128` dentro da área existente de exibição (aproximadamente 92px), mantendo a janela nativa e a área clicável anteriores; atenção operacional continua indicada pelo balão e atributos acessíveis da integração.

A aprovação visual estabelece NEX frontal como mascote oficial. O logotipo institucional permanece inalterado; ver [contrato visual](../design/NEX_32BIT_VISUAL_CONTRACT.md). O asset Compact é código-fonte editável, e Standard/Hero são derivados visuais previstos, não artefatos já disponíveis.

**Gate pendente:** avaliar fidelidade visual em escala nativa, legibilidade em temas e DPI diferentes, animação com `prefers-reduced-motion`, transparência e hit-testing em Windows/macOS/Linux, mais build frontend e smoke gráfico no mesmo SHA. Uma revisão do SVG isolado não comprova certificação de release.
