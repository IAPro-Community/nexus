## Menu da bandeja no GNOME/Wayland após FIX-035 — 2026-10-26

- O Desktop e o HUD atuais foram registrados no `StatusNotifierWatcher` do GNOME; o menu do item do HUD disponibilizou `Mostrar`, `Ocultar` e as três apresentações do mesmo HUD.
- Pelo método `com.canonical.dbusmenu.Event`, acionei `Ocultar` e `Mostrar` na sessão GNOME/Wayland real. No primeiro ciclo, o SHA permaneceu `9957c441c81942aa40678b54f7be9bd7c5429798c17e810a75a4cbbcfca0dffd`. Após o estado voltar à geometria salva de 224×158, repeti o ciclo: o SHA permaneceu `e7f2b2c562720940636be309b883a362a60f506719983e938acdb0dcb69d2294` antes/durante/depois e continuou igual após 10 s. Os campos da preferência original não mudaram; HUD ativo e Core HTTP 200.
- Binário testado: `nexus-hud` SHA-256 `7667b895cc5c884ab687db1ad982d44ccb7d3e9ed162ba98a366ed54cce65350`; aplicação instalada e health registrados em [verificação do build instalado](logs/current-installed-wayland-geometry-verification-2026-10-26.txt). Log do ciclo de bandeja: [Ocultar/Mostrar](logs/native-hud-tray-hide-show-wayland-geometry-fix-2026-10-26.txt).
- Resultado **PASS** para essas duas ações de menu e preservação do estado. Evidências: [primeiro ciclo](logs/native-hud-tray-hide-show-wayland-geometry-fix-2026-10-26.txt), [reteste com espera de 10 s](logs/native-hud-tray-hide-show-wayland-geometry-fix-delayed-2026-10-26.txt) e [script reproduzível com nova execução](logs/native-hud-tray-geometry-script-2026-10-26.txt). Execute `tools/native_hud_tray_geometry_gnome.sh` a partir da sessão GNOME Wayland para repetir o caso. Foco com outras janelas e monitores adicionais continuam sem validação; NR-003 permanece **BLOCKED**.

## Persistência de geometria no build corrente — 2026-10-26

- `make build-hud-wails` produziu o HUD AMD64 atual: ELF x86-64, SHA-256 `ae95af1a437b3b1282111fae16449d4ce48ea581e82ffffaa5aaa3faa7351e0b`.
- Em Xvfb com Xfwm, HOME/config/dados/XDG isolados: a janela Wails real iniciou em `1204,734 224×158`; foco XTest na alça e ArrowLeft mudaram para `1188,734`. O arquivo `hud-window-state.json` gravou `1188,734 224×158`. Após SIGTERM/exit 0 e nova inicialização com o mesmo perfil, a geometria restaurada foi `1188,734 224×158`; o segundo processo também saiu com exit 0.
- Capturas: [antes do restart](screenshots/native-hud-persist-xvfb-2026-10-26/hud-position-before-restart.png), SHA-256 `d54aaf0448628902a31d7f807fd966f2096e8e7622bdef7b2f9044e44911510a`; [depois do restart](screenshots/native-hud-persist-xvfb-2026-10-26/hud-position-after-restart.png), SHA-256 `703ec5fea99bd2a8102300721806327c02f7da57f59833737a1e954288bd5958`.
- Reprodução: `TMPDIR=/home/desenvolvedor/.cache/nexus-p5-recheck/tmp NEXUS_HUD_BIN=$PWD/cmd/nexus-hud/build/bin/nexus-hud NEXUS_HUD_EVIDENCE_DIR=$PWD/DEV/validation/linux-100-2026-10-25/screenshots/native-hud-persist-xvfb-2026-10-26 dbus-run-session -- xvfb-run -a -s '-screen 0 1440x900x24' bash DEV/validation/linux-100-2026-10-25/tools/native_hud_persist_xvfb.sh`.
- Esta aprovação cobre o ciclo Wails/X11 isolado do NR-004. Não cobre o compositor host, Wayland, bandeja, auto-hide ou monitores adicionais. Log: [execução reproduzível](logs/native-hud-persist-xvfb-2026-10-26.txt); script: [native_hud_persist_xvfb.sh](tools/native_hud_persist_xvfb.sh).

## PR #59 — regressões de restore/spawn revalidadas em Linux — 2026-10-26

- Proveniência: HEAD/remoto `7a04f1b4cf5950b97b4ea1e156dda31ac513823b` + fingerprint das fontes `0f9f1c0912788cf913aff1918111cf3e2b10521e22f70c9079edc6f042a08f28` (1.428 arquivos).
- `go test -race -count=1 ./internal/executil ./internal/hudlink ./internal/desktop`: PASS; `hudlink` e `desktop` passaram sob race. `executil` compilou sem testes Linux. `go vet` passou nos três pacotes.
- A tentativa inicial usou `/tmp` cheio e falhou ao escrever artefatos temporários; a repetição com `TMPDIR` e `GOCACHE` isolados passou. Permanecem apenas avisos do Wails/GTK por APIs X11 depreciadas.
- Esta é evidência de regressão Go Linux. O caminho de criação Windows de `CommandNoConsole` e o HUD vinculado no Windows não foram executados neste ambiente. Log: [teste, race e vet](logs/p9-recheck-hud-race-vet-2026-10-26.txt).

## Smoke Wails no overlay de fontes final — 2026-10-26

- Base/remoto 7a04f1b4cf5950b97b4ea1e156dda31ac513823b + overlay eb038f52b8555c362a9129a3680746e0769ba4cbe733450b5c7a3b34a3ecbe05; binário HUD SHA-256 12e83d1568dd79d082d2dd6640032c96e4e60ad2485d3364ff55e52cbd8a09a4, ELF x86-64/AMD64.
- D-Bus e HOME/XDG isolados, Xvfb, perfil sem credenciais. A janela Wails real apareceu em 224×158 com um único NEX; o clique abriu a janela popover independente. [Captura do NEX](screenshots/native-hud-workspace-fix-2026-10-26/hud-mascot.png), SHA-256 8e71b369295d80c62ea0a891772edc75f38c1780ce359f2dc97fabfa52674ef1; [captura após clique](screenshots/native-hud-workspace-fix-2026-10-26/hud-expanded-root.png), SHA-256 5512f71aae768bdbc9ccfe6b564481513d0256cad1ae0d4d3e9f6787c0145a4e; [log](logs/current-hud-native-workspace-fix-xvfb-2026-10-26.txt).
- O screenshot RGB do desktop após clique não certifica visualmente os controles dentro do popover. Alpha, click-through e Wayland do compositor host continuam BLOCKED. Esta execução não toca no serviço Desktop/HUD instalado.

## Reexecução nativa do HUD após os fixes — 2026-10-26

- `make build build-desktop-wails build-hud-wails`: PASS. HUD ELF AMD64 SHA-256 `76aabefd32c3eb88b5693546b3402e46f6006006250e2eff47822fa598e33e15`; overlay `0df26de4a35641d1e5d02ded7757b257fa7f4f15747220416574c083b4127208`.
- Wails iniciado em D-Bus/Xvfb isolados: janela `224×158`, uma arte NEX, clique expandiu controles. Evidências em `p5-recheck-2026-10-26T0347Z/screenshots/native-hud-isolated-bus/`; log `p5-recheck-2026-10-26T0347Z/logs/native-hud-current-build-xvfb-isolated-bus.txt`.
- Desktop Wails também passou com HOME vazio e com perfil Cursor fixture já autenticado. Estas são execuções Xvfb, não validação do compositor real. Alpha/Wayland continuam BLOCKED.

# Validação do HUD Linux

## Recheck visual nativo do overlay exato — mascote NEX único — 2026-10-25

O binário `nexus-hud` compilado do overlay `2c23ebb0c7726a200acbde9326c23ad6023d8fad0238c2adfef993677203f2fa` (base `7a04f1b4cf5950b97b4ea1e156dda31ac513823b`) foi iniciado isoladamente em XWayland. A janela Wails pertencente ao PID do processo estabilizou em 224×158. As três amostras nativas após 5, 15 e 30 segundos exibem **um único NEX**; o processo encerrou com exit 0. O SHA-256 do binário HUD foi `61c84e95e5f59d3d2f7f7a04849d3c8a2a8fa4489253983492ca7b6d59d5056b`.

- [Captura após 5 s](screenshots/native-hud-first-paint-current-2c23ebb0/hud-first-paint-5s.png), SHA-256 `7006231bc46d60eb22a3474156b3a11a4f1c71079c4d6722a431535509b8762c`.
- [Captura após 15 s](screenshots/native-hud-first-paint-current-2c23ebb0/hud-first-paint-15s.png), SHA-256 `195d5a7687854d055af25e0949474d64acd1de1f8fa23cca7015a427fae75721`.
- [Captura após 30 s](screenshots/native-hud-first-paint-current-2c23ebb0/hud-first-paint-30s.png), SHA-256 `c004a21fc218f052a59f7d06d68a4f6ae96b4569011ed0f20671c700fda4a736`.
- Procedimento e logs do processo: [amostragem Wails](logs/p8-current-overlay-native-hud-first-paint-2026-10-25.txt) e [geometria XWayland estável](logs/p8-current-overlay-native-hud-xwayland-stable-2026-10-25.txt).

Esta prova confirma apenas o primeiro desenho do NEX nesse HUD Linux/XWayland. As imagens RGB não validam alpha no compositor Wayland; evento proveniente de agente real, interação/foco desta sessão e matriz completa de temas permanecem pendentes. O quadro transitório de 80×48 está identificado como carregamento e não conta como prova visual.

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

### Reteste da sessão disponibilizada pelo ambiente de homologação — 2026-10-26

- O shell identifica GNOME/Wayland (`WAYLAND_DISPLAY=wayland-0`), mas uma captura inofensiva de 1×1 via `grim` terminou com `no wl_output`. Não foi criado screenshot.
- Uma cópia temporária do binário atual, isolada por HOME e identidade Wails, encerrou com exit 0 antes de uma janela ser observada; sem captura nem log que permitam atribuir a causa. Essa mesma cópia abriu em Xvfb/X11 (224×158, um NEX). O teste não substitui a evidência Wayland anterior: esta sessão não permitiu revalidá-la sobre o overlay atual.
- Resultado desta tentativa: **BLOCKED**, não `FAIL`. Evidência: [sonda Wayland atual](logs/wayland-direct-probe-current-2026-10-26.txt). BLK-010/011 seguem sem sign-off de compositor, entrada física ou alpha.

O HUD independente também foi iniciado com `GDK_BACKEND=wayland` na sessão GNOME real e no D-Bus do usuário. A árvore AT-SPI expôs o controle acessível do único NEX e o seletor. O quadro inicial foi 224×158; expandir abriu 334×496; Barra reduziu para 220×60; o tray nativo restaurou Mascote. A seleção voltou após encerrar e reiniciar o HUD. Evidências: [resumo funcional](logs/wayland-hud-functional-summary.txt), [ações AT-SPI](logs/wayland-hud-atspi-validation.log), [reinício](logs/wayland-hud-restart.log) e [hash do binário](logs/wayland-hud-binary-sha256.txt).

O Desktop também iniciou diretamente em Wayland e expôs a página principal e os controles de entrada; veja [resumo](logs/wayland-desktop-functional-summary.txt) e [log de startup](logs/wayland-desktop-startup.log). Captura direta do compositor e verificação do canal alpha continuam bloqueadas.

## Cobertura

| Fluxo | Status | Evidência / limite |
| --- | --- | --- |
| Inicialização do Desktop e HUD | PASS | Capturas e logs do build atual acima. |
| Um mascote NEX no modo Mascote | PASS | Captura do HUD Wails pós-FIX-012 e 25/25 testes focados; sem segundo renderer ou modo Companion 2D. |
| Barra e controles do HUD | PASS | Capturas XWayland do overlay frontend anterior, idêntico no código web ao atual. |
| Clique na área vazia e região interativa | PASS | XTest anterior; ver [observações](../linux-100-2026-10-24/logs/native-clickthrough-observations.md). |
| Apresentação Tray oculta no startup e restauração pela segunda instância | PASS | Wails/Xvfb: não havia janela visível no startup; a segunda instância exibiu o HUD. Ver teste NR-003 abaixo. |
| Auto-hide via Escape e restauração pela segunda instância | PASS | Wails/Xvfb: Escape ocultou o HUD e uma segunda instância voltou a exibi-lo. Ver teste NR-003 abaixo. |
| Menu da bandeja real, auto-hide por perda de foco e monitores adicionais | BLOCKED | Xvfb não oferece um host de bandeja nem monitores físicos; esta interação permanece sem validação no GNOME host. |
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

## Visibilidade Tray e auto-hide nativos em Xvfb — 2026-10-26

- Execução do binário Wails HUD AMD64 real, com HOME/XDG/D-Bus isolados, Xvfb e Xfwm: `dbus-run-session -- xvfb-run -a bash tools/native_hud_visibility_xvfb.sh`.
- **PASS — modo Tray:** o perfil isolado iniciou com a apresentação `TRAY`; o processo permaneceu vivo sem criar uma janela X11 visível. Uma segunda instância foi encaminhada ao processo já ativo e restaurou a janela visível. O mesmo XID foi mantido.
- **PASS — auto-hide:** com `autoHide=true`, Escape ocultou a janela Mascote (`IsUnMapped`). A segunda instância a restaurou (`IsViewable`). Os dois ciclos do processo principal terminaram com exit 0.
- Binário testado: `cmd/nexus-hud/build/bin/nexus-hud`; SHA-256 `fbe093cffa25e53a118bb5880a0087645d70ed73776678d55c588f549095c7b3`. Código: `release/v1.0.0@7a04f1b4cf5950b97b4ea1e156dda31ac513823b` + fingerprint de fontes `b73161ffd4afe7d566bd00cfdc6d6e090e95b0c99e13dbd0653aadd40fdf6eac` (árvore de trabalho modificada; fingerprint não é SHA Git).
- Evidência reproduzível: [log](logs/native-hud-visibility-xvfb-2026-10-26.txt) e [script](tools/native_hud_visibility_xvfb.sh). Tentativas anteriores de ajuste do detector estão preservadas em `logs/native-hud-visibility-xvfb-*-attempt-2026-10-26.txt`; elas não são resultados do produto e não foram contadas como PASS.
- Limites mantidos em **BLOCKED**: perda de foco real, foco de retorno pelo sistema, transparência/click-through no compositor e monitores adicionais. As ações GNOME `Ocultar`/`Mostrar` passaram na seção [menu da bandeja no GNOME/Wayland](logs/native-hud-tray-geometry-script-2026-10-26.txt); a fixture externa de foco não produziu uma janela estável no Xvfb e não foi contabilizada como aprovação.



## Tar HUD do overlay atual

O tar `nexus-hud_Linux_x86_64.tar.gz` do overlay `dac0d00b8410f70ed02c8b4c324780349655fed8f7e580d1944da1ec3eaecefa` foi extraído e executado em perfil isolado. Depois da restauração geométrica, a janela X11 `nexus-hud` abriu em 224×158, mostrou um único NEX e encerrou com exit 0. O primeiro detector buscava o título errado (`Nexus HUD`) e capturava a janela antes de sua geometria final; a validação foi repetida por PID/título e dimensões. Log: [smoke do pacote atual](logs/native-packaged-hud-current-overlay.txt); [captura](screenshots/native-hud-packaged-current-overlay-xwayland.png).
## Smoke Wails do HUD compilado — 2026-10-26

- Binário `cmd/nexus-hud/build/bin/nexus-hud`, ELF AMD64, SHA-256 `bf8be17a3ca080c109087fa29b73e3cb1402f3486a010ceeb1b80b799b2eaafb`; `ldd` não apontou dependências ausentes.
- **PASS parcial:** a janela Wails apareceu em Xvfb em 224×158, mostrou um NEX e um clique abriu os controles. Capturas RGB: [modo Mascote](screenshots/native-hud-current-build-xvfb-2026-10-26/hud-mascot.png) e [após clique](screenshots/native-hud-current-build-xvfb-2026-10-26/hud-expanded-root.png); hashes em `screenshots/native-hud-current-build-xvfb-2026-10-26/SHA256SUMS`.
- Execução isolada por `tools/native_hud_current_build_xvfb.sh`; resultado em `logs/native-hud-current-build-xvfb-2026-10-26-retest2.log`. O harness foi ajustado após selecionar inicialmente a janela auxiliar InputOnly 10×10.
- O código ativo mantém somente `NexusHud → GuardianMascot → NexPixelSprite`; não há modo/renderer Companion 2D separado. Xvfb não valida canal alpha, click-through, foco ou comportamento no compositor Wayland real; esses gates permanecem BLOCKED.
## Reteste mais recente — custo idle do NEX — 2026-10-26

- Build: `release/v1.0.0@7a04f1b4cf5950b97b4ea1e156dda31ac513823b` + fingerprint fonte `b73161ffd4afe7d566bd00cfdc6d6e090e95b0c99e13dbd0653aadd40fdf6eac`; ELF HUD AMD64 SHA-256 `fbe093cffa25e53a118bb5880a0087645d70ed73776678d55c588f549095c7b3`.
- O Wails abriu em Xvfb com janela 224×158 e mostrou um NEX. Clique abriu os controles do HUD. [Captura do mascote](screenshots/native-hud-idle-motion-fix-xvfb-2026-10-26/hud-mascot.png) e [captura expandida](screenshots/native-hud-idle-motion-fix-xvfb-2026-10-26/hud-expanded-root.png).
- Idle nativo: 1,5% e 0,4% de CPU em 20 s; o script aplica limite de regressão de 15% por processo WebKit. O app instalado mediu 0,5% e 0,0%. O NEX permanece estático em idle com uma piscada de 120 ms a cada 6,4 s; animações de estados ativos permanecem.
- Evidências: `logs/native-hud-idle-cpu-after-fix-xvfb-2026-10-26.txt`, `logs/native-installed-hud-desktop-idle-cpu-retest-2026-10-26.txt`, `logs/native-hud-visual-after-idle-motion-fix-xvfb-2026-10-26.txt` e `tools/native_hud_idle_cpu_xvfb.sh`.
- Limites: Xvfb usa software rendering; sem confirmação de GPU, Wayland alpha, compositor, multimonitor ou carga prolongada. O comportamento em Wayland nativo continua BLOCKED e não foi convertido em PASS.

## Amostra de repouso do build instalado após FIX-035 — 2026-10-26

- Serviço instalado em Ubuntu GNOME/Wayland, Desktop e HUD ativos. Duas janelas de 60 s mediram deltas de CPU por processo via `/proc`: Desktop 0,90%/0,87%, HUD 0,43%/0,45%, WebKit até 0,47% e depois 0,28%, Network até 0,02%. CPU agregado do cgroup foi 7,07% na amostra independente.
- Um processo WebKit cresceu 13,53 MiB na primeira janela e 0,85 MiB na repetição; a memória do cgroup caiu 0,72 MiB em uma janela separada. Não foi observado crescimento contínuo nessas amostras, mas isso não substitui soak de memória. Core manteve HTTP 200.
- Evidências: [primeira amostra por processo](logs/current-installed-desktop-hud-idle-resource-check-proc-60s-2026-10-26.txt), [repetição por processo](logs/current-installed-desktop-hud-idle-resource-check-proc-repeat-60s-2026-10-26.txt), [cgroup por 60 s](logs/current-installed-desktop-hud-idle-resource-check-60s-2026-10-26.txt) e [amostra corrigida de 20 s](logs/current-installed-desktop-hud-idle-resource-check-corrected-2026-10-26.txt). A tentativa de instrumentação anterior com ordem de propriedade incorreta não conta como métrica; [o log inicial](logs/current-installed-desktop-hud-idle-resource-check-2026-10-26.txt) fica preservado como histórico.
- Escopo curto: não comprova memória sob soak longo, rendering acelerado por GPU ou compositor Wayland completo. NR-010 continua BLOCKED para sign-off desses cenários.
