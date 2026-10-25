# Escopo do mascote NEX único

Data: 2026-10-25

Branch: `release/v1.0.0`

Commit Git base: `bf6adf0c80ac3531d7291844512a14a017ab0002`
Fingerprint das fontes validadas nesta rodada: `516b706a2c3cdca1a6d76200833a785186063350fa4d41db4494fcb67bdb7f7a` (1.404 arquivos; manifesto `logs/source-provenance-gdk-fix.txt`).

## Decisão de produto

A direção confirmada pelo usuário é ter **um único mascote NEX**, muito bem acabado e funcional. Não desenvolver nem integrar uma personagem separada ou um modo Companion 2D.

## Inspeção do código atual

- O caminho visual ativo é `NexusHud → GuardianMascot → NexPixelSprite`.
- `companionModel` transforma dados de estado e atenção do HUD para o mesmo NEX; não renderiza personagem própria.
- O seletor `Barra / Mascote / Bandeja` controla apresentações do HUD. `Mascote` exibe o único NEX.
- O checkout não contém `NexusGuardian.tsx`, atlas separado ou renderer Companion. A PR #55 foi fechada e não integra esta release.

**Resultado:** Companion 2D separado = `NOT_PRESENT` e fora do escopo. Não havia modo 2D nem segundo mascote integrado para remover. A apresentação `Mascote` foi mantida porque é a forma de exibir o NEX único.

## Evidência Linux

O teste focalizado do mascote foi repetido após FIX-012 e passou em 25/25 casos; veja [log da revalidação](logs/mascot-no-2d-mode-regression-2026-10-25.log). A captura nativa mais recente do HUD Wails mostra um NEX; SHA-256 `98b4320b329512b349af90284f5851b9080b20be3d97fc4bccf19520ac16cd3c`. O registro detalhado está em [MASCOT_VALIDATION.md](MASCOT_VALIDATION.md), [TEST_MATRIX.md](TEST_MATRIX.md) e [auditoria do renderer único](logs/mascot-single-renderer-current-audit.txt).

As lacunas de atenção real no HUD e transparência alpha dizem respeito à homologação funcional do único mascote; elas não indicam ausência de um Companion 2D.

## HUD nativo: balão e adiamento com fixture

No overlay de fontes `4e7afe7bc0e99c8d11e34dd3db9295829ab41081173bc9a7cc9b262f3c92111b`, o HUD Wails real foi executado em XWayland com perfil Nexus temporário. Um registro controlado `REQUIRE_USER` apareceu no balão do único NEX. XTest clicou em “Adiar”; a janela voltou ao tamanho compacto e a API canônica continuou retornando a tarefa (`HTTP 200`, `needs_you_count=1`). A evidência visual e o clique passaram em NR-012/MASCOT-006. Isso valida fixture e interação nativa; não prova ainda a integração de um evento real de agente, que segue BLOCKED em BLK-009.

- [Captura nativa do balão](screenshots/native-hud-attention-fixture-xwayland.png), SHA-256 `b2bff6d18ab8214d682430c3f097e93bf1e338a1924af7ccb7b25f661c7ba08a`.
- [Captura após “Adiar”](screenshots/native-hud-attention-fixture-snoozed-xwayland.png), SHA-256 `cbad75a5d16d15e17d92b5d99447f33955281a9f35b3d36f4682dc822c159d5e`.
- [Preferências do HUD](screenshots/native-hud-expanded-single-nex-auth-fix-xwayland.png).
- [Setup reproduzível da fixture](logs/native-attention-fixture-setup.txt); [log do clique e consulta canônica](logs/native-attention-fixture-click-snooze.txt); [build do HUD](logs/native-attention-current-hud-build.log) e [identidade do binário](logs/native-attention-current-hud-binary.txt), SHA-256 `d16edb3ad61dea3e3cceb4124c6524145abe04f7afa019c6ff1f0e8cdbbdde2d`.
