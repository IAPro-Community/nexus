# Homologação gráfica nativa do Nexus HUD

Este procedimento comprova clique, popup, geometria, sombra, transparência e
arraste nos executáveis reais de Windows, Linux X11 e macOS. Não confundir com
Playwright/Web E2E ou screenshot sintético.

## Preparação do candidato

Use um clone limpo da branch release/v1.0.0, com um único SHA completo de
40 caracteres. Compile Core, frontend e Nexus HUD a partir dessa mesma árvore,
com as versões de Go e Bun declaradas em docs/operations/platform-support.md.
Não reutilize executáveis antigos. Faça os testes em uma sessão gráfica
desbloqueada, sem outra instância Nexus HUD; os scripts evitam interferir
com instalações existentes e criam diretórios de dados isolados.

Os testes de mouse são executados localmente. Não enviam capturas ou contas
para servidores externos. Inspecione logs privados antes de compartilhá-los.

## Windows

Após executar o build nativo da release, no PowerShell, a partir da raiz:

    $sha = (git rev-parse HEAD).Trim()
    $hud = (Resolve-Path 'cmd/nexus-hud/build/bin/nexus-hud.exe').Path
    .\scripts\homologate-hud-native-windows.ps1 -HudExe $hud -EvidenceDirectory (Join-Path $PWD 'hud-evidence/windows') -ExpectedSha $sha -Cycles 20

O script usa Win32 para clique físico e medição de dimensões. As screenshots
incluem margem externa, permitindo observar a sombra que denuncia o retângulo.

## Linux X11

Após compilar com make build-hud-wails, numa sessão X11 **real**:

    bash scripts/homologate-hud-native-linux.sh "$PWD/cmd/nexus-hud/build/bin/nexus-hud" "$PWD/hud-evidence/linux-x11" 20

Exige xdotool, ImageMagick (import), python3 e sha256sum.
Uma sessão Wayland retorna BLOCKED, não PASS; deve ter revisão separada,
adequada ao compositor. Xvfb não substitui evidência de transparência no
desktop de produção.

## macOS

Após compilar o .app via scripts/build-wails-v3-macos-app.sh (conforme a
certificação de release), identifique o binário real em Contents/MacOS.
Execute:

    bash scripts/homologate-hud-native-macos.sh "/caminho/IAPro Nexus HUD.app/Contents/MacOS/nexus-hud" "$PWD/hud-evidence/darwin" 20

Pode ser necessário conceder permissões de Acessibilidade e Gravação da Tela
ao terminal que está executando o script; sem elas o teste permanece BLOCKED.
Exige swiftc e screencapture, já presentes num ambiente de desenvolvimento
macOS configurado.

## Resultados esperados

Cada script cria HUD_NATIVE_RESULTS.json, HUD_BEFORE.png, HUD_EXPANDED.png e
logs privados. O máximo automático é PASS_FUNCTIONAL_PENDING_VISUAL, porque
capturar uma imagem não comprova ausência de sombra nem click-through.

A revisão visual real deve verificar: sombra retangular, fundo opaco,
clipping, desenho arredondado, posição junto à taskbar/dock, DPI,
reinicialização, três tamanhos, arraste e abertura/fechamento do popup.
Para click-through, colocar um aplicativo de teste atrás do HUD e clicar
na parte **transparente** da superfície; deve receber o clique, não o HUD.
Qualquer falha é NO_GO. Isso é distinto dos 20 ciclos de abrir/fechar.

Após revisão, preencher no JSON de cada plataforma:
- visual_shadow_review = PASS somente com evidência visual revisada.
- transparent_click_through = PASS somente após teste real desse comportamento.
- review.reviewer e review.reviewed_at_utc com responsável e timestamp UTC.
- build_attestation.source_sha e build_attestation.binary_sha256 obtidos
  ao confrontar o SHA do checkout com o hash do executável gerado por ele.
O binary_sha256 já é produzido automaticamente pelos scripts.
Não inventar ou preencher PASS para verificações não realizadas.

## Agregação fail-closed no mesmo SHA

    python3 scripts/verify-hud-native-evidence.py --sha SHA_COMPLETO --windows hud-evidence/windows --linux-x11 hud-evidence/linux-x11 --darwin hud-evidence/darwin --min-cycles 20 --output hud-evidence/HUD_CROSS_PLATFORM_SUMMARY.json

O agregador aceita GO somente para três relatórios do mesmo SHA, vinte
ciclos completos por plataforma, screenshots, revisão visual e de interação
explicitamente aprovadas, e atestado binário compatível. Falha funcional
confirmada retorna NO_GO. Qualquer evidência ausente retorna INCONCLUSIVE.

O gate acima não certifica Wayland, assinatura/notarização, instalador,
todas as combinações de monitores ou permissões corporativas. Estes são
gates complementares. A matriz HUD continua NO_GO enquanto faltarem
evidências dos três sistemas para o mesmo commit.

## CI e limites

Os workflows agora verificam sintaxe das automações nativas, o contrato
de janela Wails e testes do agregador. Runner headless ou execução de
aplicativo por oito segundos não equivalem a homologação visual.
Não publicar tag ou afirmar paridade visual apenas pelo CI.
