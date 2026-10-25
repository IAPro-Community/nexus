# HUD nativo Linux — validação independente

## PASS verificado

- Desktop e HUD foram compilados separadamente para Linux AMD64 com Wails v3.
- Layout nativo previsto foi montado com `nexus-desktop` e `nexus-hud` como irmãos executáveis.
- Desktop real iniciou o processo `nexus-hud --linked` no GNOME 49/Wayland. O build corrigido repetiu o spawn.
- Core embutido do build corrigido respondeu HTTP 200.
- O teste de restauração de geometria foi corrigido para selecionar explicitamente a capacidade que o teste cobre; o comportamento não simula posicionamento global em Wayland.
- A suíte Go completa, race detector, vet, HUD link unit tests e regressão focal de geometria passaram.

## BLOCKED / NOT_RUN

- Barra, Mascote, Tray, popover, transparência, click-through, auto-hide, show/hide, minimização, foco, expansão, posição, geometria e persistência não puderam ser operados e capturados na janela real. `grim` não tem protocolo screencopy disponível, GNOME Shell negou screenshot via D-Bus, Flameshot expirou e a ação AT-SPI não retornou.
- Não validamos interação de Capsule/attention bubble. O mascote da branch é apenas o Guardian antigo; a ausência do Companion 2D é documentada em `COMPANION_VALIDATION.md`.
- A sessão Wayland deste host desabilita posicionamento global arbitrário. Posição entre monitores e persistência visual são BLOCKED.
- Não havia uma sessão X11 independente. XWayland não é tratado como validação X11.
- GPU do VMware informa `No 3D enabled`; animação/consumo de GPU não homologados.

## Correção de instalação identificada

O instalador Linux distribuía `nexus-desktop`, mas não o executável `nexus-hud` que o Desktop procura no mesmo diretório. Foi corrigido localmente para validar e instalar o archive HUD pelo manifesto assinado. A verificação de release real permanece bloqueada porque o manifesto estável não existe.

## PR #59

A PR #59 segue aberta e não mergeada. O SHA f62 contém o caminho restore-on-show descrito no commit corrente; testes unitários passaram após corrigir o fixture de teste sensível ao Wayland. Não afirmamos que todas as correções da PR foram integradas ou que a interação visual esteja homologada.
