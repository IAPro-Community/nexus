# HUD nativo: janela de popup independente (experimento)

**Estado:** integração na `release/v1.0.0` via PR #56.
A nova instalação inicia em **Mascote**, sem variável de ambiente.
A segunda janela é padrão em sessões Windows, macOS e Linux X11,
mas permanece configurável pelo usuário no próprio HUD. Em Wayland,
o Mascote abre o painel na mesma janela, sem tentar posicionamento
global de um popup separado.

## Objetivo

O Nexus Guardian mantém um Mascote 2D de 108 x 112 DIPs por padrão.
O usuário pode escolher Barra (220 x 60 DIPs no preset Compacto)
ou Bandeja. Quando permitido pelo sistema, os detalhes são exibidos
em janela Wails separada, com WebView/React próprios e Core compartilhado.
O popup começa oculto e surge apenas mediante interação.

### Configuração reversível e persistida

Use **Barra / Mascote / Bandeja** no painel ou no menu da bandeja.
A opção **Popup independente** controla apenas se os detalhes aparecem
numa segunda janela; ela não ativa nem desativa o Mascote.
As preferências sobrevivem à reinicialização. Não é necessário configurar
`NEXUS_HUD_DETACHED_POPOVER`.

## Componentes

- `cmd/nexus-hud/main.go` cria a apresentação escolhida e, por padrão
  em sistemas compatíveis, uma segunda janela `Nexus HUD Popup`, com rota
  `/?nativeHud=1&hudView=popover` e `Hidden=true`.
- `cmd/nexus-hud/window_options.go` compartilha o contrato de
  transparência, moldura, sombras e escala das opções Wails.
- `internal/desktop/hud_detached.go` ancora o popup acima da
  cápsula, ou abaixo se não houver espaço, e limita seu tamanho à
  work area. A cápsula nunca recebe as dimensões do painel.
- `HUDController.SetHUDExpanded` continua o único contrato
  de abertura/fechamento. Ele redimensiona apenas no modo legado;
  no modo separado, mostra/oculta a segunda janela.
- `NexusHud.tsx` renderiza apenas a cápsula na primeira WebView e
  apenas o painel na segunda. `GetHUDExpanded` sincroniza fechamento
  por Escape, perda de foco e botões, evitando estados presos.
- O tray continua associado à cápsula; a janela popup não substitui
  o processo ou os dados existentes.

## Gates obrigatórios

1. Build e testes unitários Go: `./internal/desktop/...` e
   `./cmd/nexus-hud`; lint/typecheck/format e testes Web.
2. Windows/macOS/Linux X11: clicar 20 vezes para abrir/fechar;
   medir que **a cápsula permanece compacta** e o popup é outra
   janela visível, com título/handle próprios.
3. Arrastar a cápsula, usar presets Compacto/Confortável/Grande,
   e abrir novamente; popup deve surgir perto da cápsula.
4. Verificar o comportamento da perda de foco, Escape, tray,
   Core offline, kill switch, DPI/monitores e reinicialização.
5. Registrar screenshots reais das duas janelas, sombreamento
   e bloqueio de cliques nas regiões transparentes. Dois
   retângulos transparentes não significam click-through perfeito.
6. Em Linux Wayland, confirmar que o Mascote continua disponível e
   seu painel expande na mesma janela, sem posicionamento global.

No Windows, o script nativo já possui um gate interativo específico:

```powershell
$sha = (git rev-parse HEAD).Trim()
.\scripts\homologate-hud-native-windows.ps1 `
  -HudExe (Resolve-Path '.\cmd\nexus-hud\build\bin\nexus-hud.exe').Path `
  -EvidenceDirectory (Join-Path $PWD 'hud-evidence/windows-detached') `
  -ExpectedSha $sha -Cycles 20 -Detached
```

Esse gate exige que a cápsula permaneça compacta, localiza a segunda
janela pelo seu título nativo, abre/fecha 20 vezes e captura a janela real.
Ainda requer sessão desktop e revisão humana da transparência.

Os scripts Linux/macOS `homologate-hud-native-*.sh` continuam cobrindo
a geometria **legada**, não certificam automaticamente o popup separado;
adaptações equivalentes e execução nativa permanecem obrigatórias.

## Auditoria da apresentação e animações

- A janela de detalhes pré-renderiza o conteúdo em sua WebView oculta, sem
  esperar o primeiro intervalo de sincronização, evitando uma tela em branco
  quando `Show()` abre o popup.
- A entrada do painel usa somente `opacity` e `transform` por 150 ms;
  a janela nativa não é redimensionada para simular uma animação.
- `prefers-reduced-motion: reduce` desativa a animação; o HUD web/legado
  mantém as transições existentes.
- Os estados de rendering são cobertos por `hudPresentation.test.ts`.
- O Mascote 2D possui estados ocioso, trabalhando, atenção e offline,
  com `prefers-reduced-motion` para respeitar acessibilidade.
- Ainda são obrigatórias capturas sequenciais e testes de foco/clique em
  sessões gráficas reais. O CI não comprova que a animação roda sem flicker.

O popup separado reduz a área clicável invisível da cápsula, mas não
implementa automaticamente click-through por pixel no WebView2 ou GTK/Wayland.
Não declarar GO nativo sem homologação do mesmo SHA.
