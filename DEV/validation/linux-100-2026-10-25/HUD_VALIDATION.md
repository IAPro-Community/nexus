# Validação do HUD Linux

## Evidência mais recente — FIX-012

Fontes testadas: fingerprint `516b706a2c3cdca1a6d76200833a785186063350fa4d41db4494fcb67bdb7f7a` (1.404 arquivos; `logs/source-provenance-gdk-fix.txt`). O Desktop e o HUD foram recompilados para AMD64. O HUD Wails foi aberto em XWayland com `G_DEBUG=fatal-criticals`; AT-SPI abriu e fechou o popover três vezes (224×158 ↔ 334×496), sem critical GTK/GDK, e o processo encerrou normalmente. A captura do modo Mascote mostra uma única arte NEX: [screenshot](screenshots/native-hud-gdk-fix-mascot-xwayland.png), SHA-256 `98b4320b329512b349af90284f5851b9080b20be3d97fc4bccf19520ac16cd3c`.

Regressão: [teste focado](logs/go-test-hud-popup-fix-2026-10-25.log) e [race detector](logs/go-race-hud-popup-fix-2026-10-25.log). Smoke nativo: [abertura/fechamento](logs/native-hud-gdk-fix-open-close-xwayland.txt). A verificação dos cinco arquivos do mascote passou em 25/25: [log](logs/mascot-no-2d-mode-regression-2026-10-25.log).

## Execução nativa

O Desktop Wails/Core e o HUD foram iniciados no GNOME 49/Wayland por XWayland (`GDK_BACKEND=x11`), com HOME/XDG isolados. O overlay corrigido é `996e1063540370263158e70903477f4cbbc3035dbb33b93161e603bffbe3c133`. O Desktop raw e os tarballs Desktop/HUD do workflow foram executados; o HUD corrigido também foi movido por teclado e restaurou a posição após reinício. Os processos de teste encerraram com exit 0.

Após FIX-010, `make build-desktop-wails build-hud-wails` recompilou os dois binários. O HUD atual foi iniciado com `GDK_BACKEND=x11` e D-Bus da sessão, usando HOME/XDG Nexus isolados. A janela real `nexus-hud` abriu em 224×158 e a captura mostra um único NEX. Binário HUD SHA-256 `f9acd2de7827f7ce4529d99189f0fa4116e6ca1a4ab05a9c3da030a30a4c4348`; screenshot em [native-hud-single-nex-mascot-fix-xwayland.png](screenshots/native-hud-single-nex-mascot-fix-xwayland.png).

- Desktop: [1280×800](screenshots/native-nexus-desktop-final-xwayland.png).
- HUD: [224×158](screenshots/native-nexus-hud-single-mascot-xwayland.png), com um mascote NEX.
- Binários: [hashes atuais](logs/current-binary-sha256.txt), [ELF](logs/current-binary-file.txt), [ldd Desktop](logs/current-desktop-ldd.txt) e [ldd HUD](logs/current-hud-ldd.txt).
- Startup anterior: [Desktop](logs/native-desktop-startup.log) e [HUD](logs/native-hud-startup.log). Build atual: [Desktop](logs/native-desktop-xwayland-position-fix.txt), [HUD e posição](logs/native-xwayland-keyboard-position-fix.txt), [pacotes nativos](logs/linux-native-packages-xwayland-position-fix.txt), [hashes e dependências](logs/linux-xwayland-position-fix-build.txt).

## Wayland direto

O HUD independente também foi iniciado com `GDK_BACKEND=wayland` na sessão GNOME real e no D-Bus do usuário. A árvore AT-SPI expôs o controle acessível do único NEX e o seletor. O quadro inicial foi 224×158; expandir abriu 334×496; Barra reduziu para 220×60; o tray nativo restaurou Mascote. A seleção voltou após encerrar e reiniciar o HUD. Evidências: [resumo funcional](logs/wayland-hud-functional-summary.txt), [ações AT-SPI](logs/wayland-hud-atspi-validation.log), [reinício](logs/wayland-hud-restart.log) e [hash do binário](logs/wayland-hud-binary-sha256.txt).

O Desktop também iniciou diretamente em Wayland e expôs a página principal e os controles de entrada; veja [resumo](logs/wayland-desktop-functional-summary.txt) e [log de startup](logs/wayland-desktop-startup.log). Captura direta do compositor e verificação do canal alpha continuam bloqueadas.

## Cobertura

| Fluxo | Status | Evidência / limite |
| --- | --- | --- |
| Inicialização do Desktop e HUD | PASS | Capturas e logs do build atual acima. |
| Um mascote NEX no modo Mascote | PASS | Captura do HUD Wails pós-FIX-012 e 25/25 testes focados; sem segundo renderer ou modo Companion 2D. |
| Barra e controles do HUD | PASS | Capturas XWayland do overlay frontend anterior, idêntico no código web ao atual. |
| Clique na área vazia e região interativa | PASS | XTest anterior; ver [observações](../linux-100-2026-10-24/logs/native-clickthrough-observations.md). |
| Bandeja, auto-hide, foco e vários monitores | NOT_RUN | Não exercitado no host atual. |
| Posição e geometria após reinício completo em todas as apresentações | NOT_RUN | O caso de movimento por teclado em XWayland e persistência da posição foi aprovado em NATIVE-006; a matriz de geometrias/modos completa continua pendente. |
| Balão nativo alimentado por evento real | BLOCKED | Evento real do runtime ainda não foi ligado ao HUD. |
| Balão nativo e ação “Adiar” em fixture controlada | PASS | O HUD Wails real exibiu estado `REQUIRE_USER` ao lado de um único NEX. Clique XTest em “Adiar” recolheu o balão, e o endpoint `/api/v1/attention` continuou com a fixture `needs_you`. [Captura](screenshots/native-hud-attention-fixture-xwayland.png), [após ação](screenshots/native-hud-attention-fixture-snoozed-xwayland.png), [log](logs/native-attention-fixture-click-snooze.txt). |
| Transparência alpha | BLOCKED | Captura disponível não preserva alpha. |
| HUD funcional em Wayland direto | PASS | Expansão/recolhimento, seletor, tray e persistência da apresentação foram observados. |
| Teclado e foco em Wayland direto | BLOCKED | A tentativa de tecla sintética por AT-SPI não alterou o HUD; não há caminho físico controlado nesta VM. O caminho XWayland passou separadamente em NATIVE-006. |
| Movimento por teclado e restauração de posição em XWayland | PASS | ArrowRight moveu X=50→66→82; o processo reiniciado restaurou x=82. Screenshots e hashes em [evidência de posição](logs/native-xwayland-keyboard-position-fix.txt). |
| Soak curto e crash/recovery do HUD empacotado | PASS | 60 s, RSS 172840–173884 KB, 3 filhos; após SIGKILL não houve órfãos e a posição x=82 foi restaurada. O soak prolongado continua pendente em NR-007. |
| Captura de pixels/alpha em Wayland | BLOCKED | GNOME negou captura D-Bus; `grim` não é suportado pelo compositor e o portal exigiu seleção interativa. |
| Movimento e GPU sob hardware acelerado | NOT_RUN | A VM não oferece aceleração 3D utilizável. |

As capturas atuais são RGB. Fundo preto na captura não confirma nem reprova transparência. O ambiente sem GPU também limita a conclusão sobre custo de animações.



## Tar HUD do overlay atual

O tar `nexus-hud_Linux_x86_64.tar.gz` do overlay `dac0d00b8410f70ed02c8b4c324780349655fed8f7e580d1944da1ec3eaecefa` foi extraído e executado em perfil isolado. Depois da restauração geométrica, a janela X11 `nexus-hud` abriu em 224×158, mostrou um único NEX e encerrou com exit 0. O primeiro detector buscava o título errado (`Nexus HUD`) e capturava a janela antes de sua geometria final; a validação foi repetida por PID/título e dimensões. Log: [smoke do pacote atual](logs/native-packaged-hud-current-overlay.txt); [captura](screenshots/native-hud-packaged-current-overlay-xwayland.png).
