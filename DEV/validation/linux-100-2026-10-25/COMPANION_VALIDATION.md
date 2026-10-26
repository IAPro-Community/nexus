# Decisão reafirmada — manter um mascote NEX — 2026-10-26

O escopo segue sendo um mascote NEX único, bem acabado e funcional. Companion 2D não deve ser desenvolvido nem integrado. A PR #55 está `CLOSED`, confirmado no GitHub em 26/10 ([registro](logs/current-pr55-product-scope-2026-10-26.json)). O checkout atual já não possui um segundo personagem, renderer ou modo 2D para apagar; não removi a apresentação `Mascote`, pois ela renderiza o NEX único.

Rechecagem final repetida na base `release/v1.0.0@7a04f1b4cf5950b97b4ea1e156dda31ac513823b` e overlay `0f9f1c0912788cf913aff1918111cf3e2b10521e22f70c9079edc6f042a08f28`: seis suítes/38 testes focados, arte oficial e E2E HUD passaram com `TMPDIR`/`GOCACHE` isolados. Logs: [Vitest](logs/mascot-single-nex-final-2026-10-26.txt), [arte](logs/mascot-single-nex-art-final-2026-10-26.txt), [E2E HUD](logs/mascot-single-nex-hud-e2e-final-2026-10-26.txt). O screenshot nativo Wails/Xvfb mostra apenas um NEX; conteúdo completo do popover, alpha e Wayland real continuam sem aprovação. Detalhes: [MASCOT_VALIDATION.md](MASCOT_VALIDATION.md).

# Escopo do mascote NEX único

## Rechecagem da direção e do build nativo atual — 2026-10-26

A instrução vigente é um único mascote NEX bem acabado e funcional, sem modo 2D separado. Inspecionei o código e executei o binário Wails atual: `NexusHud → GuardianMascot → NexPixelSprite` apresenta somente um NEX; Barra/Mascote/Bandeja são modos de apresentação do mesmo HUD. Não há um segundo personagem para remover nem Companion 2D ativo para aprovar. Captura nativa atual e limites de alpha estão em [MASCOT_VALIDATION.md](MASCOT_VALIDATION.md#reteste-do-build-wails-corrente--2026-10-26).

> **Escopo vigente:** o produto tem um NEX único. A PR #55 está `CLOSED`; sua personagem e modo Companion 2D estão fora do escopo e não são critério de aprovação. Os resultados correntes estão em [MASCOT_VALIDATION.md](MASCOT_VALIDATION.md).

## Reconfirmação — 2026-10-25

O usuário reafirmou que o produto deve ter um só mascote muito bem acabado e funcional, sem modo 2D separado. Confirmei no código atual que a PR #55 não está integrada e não há segundo personagem para remover. O botão “Mascote” exibe o mesmo NEX de `NexusHud → GuardianMascot → NexPixelSprite`; as opções Barra e Bandeja também são apresentações do mesmo HUD.

- **PASS:** 29 testes focados, check da arte oficial e E2E do HUD Chromium; logs em `logs/mascot-user-direction-recheck-2026-10-25.log`, `logs/mascot-art-check-recheck-2026-10-25.log` e `logs/mascot-hud-e2e-user-direction-2026-10-25.log`.
- **PASS visual nativo no overlay exato:** o HUD Wails `nexus-hud` do overlay `2c23ebb0…` abriu em XWayland, geometria estável 224×158, e três amostras em 5/15/30 segundos mostram um único NEX. Binário SHA-256 `61c84e95e5f59d3d2f7f7a04849d3c8a2a8fa4489253983492ca7b6d59d5056b`; procedimento, screenshots e hashes em [MASCOT_VALIDATION.md](MASCOT_VALIDATION.md#recheck-exato-do-overlay-corrente--um-único-nex--2026-10-25) e [HUD_VALIDATION.md](HUD_VALIDATION.md#recheck-visual-nativo-do-overlay-exato--mascote-nex-único--2026-10-25). O E2E Chromium segue como evidência complementar.
- O código-base tinha um modelo de estado/atenção chamado `companionModel`, usado pelo mesmo renderer do NEX; não havia um segundo personagem desenhado na janela. O overlay atual removeu esse modelo paralelo e mantém o comportamento no `mascotModel`. Não existe modo visual Companion 2D a integrar nem a usar como critério de aprovação. `companionMotion` é lido para migrar a preferência para `mascotMotion`; a chave antiga de adiamento continua como armazenamento v1 do mesmo NEX para preservar estado salvo. Alpha e atenção ligada a execução real seguem pendentes de homologação.

Data: 2026-10-25

Branch: `release/v1.0.0`

HEAD local/remoto: `7a04f1b4cf5950b97b4ea1e156dda31ac513823b`
Fingerprint atual das fontes: `d3caa597ae3bc061ad0048ad4d390b1b9bb0011cc496e4d213d0ec6119400796` (1.422 arquivos; manifesto `logs/source-provenance-p5-plan-revision-cas-2026-10-25.txt`). A árvore tem alterações locais identificadas por fingerprint, sem commit imutável.

## Decisão de produto

A direção confirmada pelo usuário é ter **um único mascote NEX**, muito bem acabado e funcional. Não desenvolver nem integrar uma personagem separada ou um modo Companion 2D.

## Inspeção do código atual

- O caminho visual ativo é `NexusHud → GuardianMascot → NexPixelSprite`.
- `mascotModel` transforma estado e atenção para o mesmo NEX; não existe uma entidade ou renderer de mascote separado.
- O seletor `Barra / Mascote / Bandeja` controla apresentações do HUD. `Mascote` exibe o único NEX.
- `NexusGuardian.tsx`, `companionModel.ts`, atlas separado e renderer/modo Companion não existem no código ativo. A PR #55 não integra esta release.
- `companionMotion` permanece como entrada de migração para `mascotMotion`; `iapro:nexus:companion:snooze:v1` continua lendo e gravando adiamentos do mesmo NEX para preservar estado. Nenhuma chave ativa outro personagem ou modo.

**Resultado:** Companion 2D separado = `NOT_PRESENT` e fora do escopo. Não há segundo mascote nem modo 2D no checkout atual. A apresentação `Mascote` foi mantida porque exibe o NEX único.

## Evidência corrente Linux — overlay P5

Cinco suítes do overlay atual passaram em 29/29 testes; `bun run art:nex:check` e a verificação estrutural do SVG oficial passaram. O renderer mantém exatamente um `data-nex-art`, sem o marcador antigo de Companion. Evidências: [testes focados](logs/p5-overlay-single-mascot-regressions-2026-10-25.txt), [arte oficial](logs/p5-overlay-nex-art-check-2026-10-25.txt) e [estrutura visual](logs/p5-overlay-nex-visual-structure-2026-10-25.txt).

Revisei visualmente as capturas Wails registradas: o modo Mascote mostra um NEX; o balão de atenção aparece junto desse mesmo NEX; e o painel de preferências seleciona Mascote entre Barra/Mascote/Bandeja. Não há segundo personagem nessas telas.

- [Modo Mascote — um NEX](screenshots/native-hud-current-overlay-xwayland-visible.png), SHA-256 `b86f040a0fce2ef2837bc27a3b81a84898d25d555a8d14bf24cff51a6d4a5ad7`.
- [Balão de atenção com fixture](screenshots/native-hud-attention-fixture-xwayland.png), SHA-256 `b2bff6d18ab8214d682430c3f097e93bf1e338a1924af7ccb7b25f661c7ba08a`.
- [Preferências do HUD](screenshots/native-hud-expanded-single-nex-auth-fix-xwayland.png), SHA-256 `b344a1010ea87942d1010dafe837f9f00d84eb659fb37c75aa4f649b87628270`.

No reteste nativo do overlay exato, as capturas estáveis em [5 s](screenshots/native-hud-first-paint-current-2c23ebb0/hud-first-paint-5s.png), [15 s](screenshots/native-hud-first-paint-current-2c23ebb0/hud-first-paint-15s.png) e [30 s](screenshots/native-hud-first-paint-current-2c23ebb0/hud-first-paint-30s.png) mostram o mesmo NEX; os checksums e o PID estão em `logs/p8-current-overlay-native-hud-first-paint-2026-10-25.txt`. Essa evidência atualiza a janela de primeiro desenho, mas não certifica alpha, entrada do compositor, foco ou atenção originada por agente real.

O status se refere à ausência de um segundo mascote e à renderização única. Atenção ligada a evento real de agente, transparência alpha no compositor Wayland e persistência integral dos estados seguem pendentes; veja [MASCOT_VALIDATION.md](MASCOT_VALIDATION.md), [TEST_MATRIX.md](TEST_MATRIX.md) e [auditoria do renderer único](logs/mascot-single-renderer-current-audit.txt).

As lacunas de atenção real no HUD e transparência alpha dizem respeito à homologação funcional do único mascote; elas não indicam ausência de um Companion 2D.

## HUD nativo: balão e adiamento com fixture

No overlay de fontes `4e7afe7bc0e99c8d11e34dd3db9295829ab41081173bc9a7cc9b262f3c92111b`, o HUD Wails real foi executado em XWayland com perfil Nexus temporário. Um registro controlado `REQUIRE_USER` apareceu no balão do único NEX. XTest clicou em “Adiar”; a janela voltou ao tamanho compacto e a API canônica continuou retornando a tarefa (`HTTP 200`, `needs_you_count=1`). A evidência visual e o clique passaram em NR-012/MASCOT-006. Isso valida fixture e interação nativa; não prova ainda a integração de um evento real de agente, que segue BLOCKED em BLK-009.

- [Captura nativa do balão](screenshots/native-hud-attention-fixture-xwayland.png), SHA-256 `b2bff6d18ab8214d682430c3f097e93bf1e338a1924af7ccb7b25f661c7ba08a`.
- [Captura após “Adiar”](screenshots/native-hud-attention-fixture-snoozed-xwayland.png), SHA-256 `cbad75a5d16d15e17d92b5d99447f33955281a9f35b3d36f4682dc822c159d5e`.
- [Preferências do HUD](screenshots/native-hud-expanded-single-nex-auth-fix-xwayland.png).
- [Setup reproduzível da fixture](logs/native-attention-fixture-setup.txt); [log do clique e consulta canônica](logs/native-attention-fixture-click-snooze.txt); [build do HUD](logs/native-attention-current-hud-build.log) e [identidade do binário](logs/native-attention-current-hud-binary.txt), SHA-256 `d16edb3ad61dea3e3cceb4124c6524145abe04f7afa019c6ff1f0e8cdbbdde2d`.
