# NEX 1.0 — Entrega completa e simultânea (HUD + Site)

**Decisão do proprietário em 28/10/2026:** NEX **não será apresentado no site público** até que ambas as versões — personagem Hero do site e pixel art Compact/Standard do HUD — estejam **completas, visualmente coerentes, animadas e homologadas**. O giro 360° faz parte do escopo obrigatório desta entrega, não é um acessório opcional. Não propor publicação de Hero parcial, teaser ou mudança do HUD antes disso.

## Fonte única da identidade
Manter o robô da referência aprovada: capacete branco arredondado, visor navy expressivo, duas aletas, dois acoplamentos laterais, núcleo ciano no peito, quatro módulos flutuantes e órbita inferior. Preservar anatomia ao trocar de pose, escala ou estado. O PNG editorial é **referência**, não um sprite atlas nem substituto do master editável.

- Decisão do personagem: [identity-lock](identity-lock.json), [direção pixel art](RETRO_GAME_IDLE_DIRECTION.md).
- Hero híbrido 2D: [pacote de trabalho](hybrid-v1/README.md). O SVG frontal anterior **não tem paridade suficiente** para substituir o PNG.
- Roteiro de movimento: [motion architecture](MOTION_ARCHITECTURE_V2.md) e [animações reais](../NEX_ANIMATION_CONTRACT.md).
- Estado de entrega único em formato de máquina: [RELEASE_READINESS.json](RELEASE_READINESS.json). Nenhuma conclusão baseada apenas em issue fechada ou texto em PR.

## Sequência de produção obrigatória

**Fase A — Master e assets de origem.** Incorporar imagens, PNG aprovado e fonte editável (OpenRaster ou Aseprite) por hash ao GitHub. Criar master frontal de alta resolução em camadas com pormenores de capacete, visor, olhos, aletas, módulos, anel, núcleo e preenchimento das áreas ocultas. Validar piscada e todas as bordas em movimento. Entregar um inventário único com SHA, licença, pivôs, origem e comparação contra referência.

**Fase B — Rig e 12 idles.** Implementar 12 ações do catálogo oficial com **frames/curvas aprovados**, incluindo hover, blink, glance, orbit breathe, head tilt, fin twitch, module inspect, orbit shuffle, curiosity, stretch, drowse e wake. Programar scheduler determinístico em testes, microações não repetitivas e preempção imediata nos cinco estados reais. `prefers-reduced-motion`/off, janela oculta e offline congelam tudo. Não simular sucesso, trabalho ou progresso.

**Fase C — Versão HUD Retro Game.** Derivar **Compact 64×64** e **Standard 128×128** com clusters de pixel art desenhados à mão, não reduzir Hero e aplicar filtro. Atlas leve e anti-jitter, testes em 48/64/92px, barra/bandeja/mascote, drag/click-through, foco e transparência. A versão nativa deve manter o Core e os sinais reais intactos.

**Fase D — Hero e 360° real do site.** Versão Hero fiel ao mesmo master, com movimentos expressivos em camadas; produzir 16 vistas genuínas (0° a 337,5°), 22,5° por passo, com costas e perfis reais, sem espelhamento da frontal. Atlas transparente ou WebP lossless, lazy-load e controlador com teclado/arraste, pausa fora da tela e reduced-motion. Site PT e EN; layout responsivo. Não trocar o Hero público antecipadamente.

**Fase E — Homologação e release única.** Validar source SHA, asset hashes, 12 idles, 16 vistas, as cinco reações reais, navegadores desktop/mobile, acessibilidade, consumo CPU/RAM/LCP, e smoke Windows/macOS/Linux X11/Wayland. Comparar as duas versões lado a lado. Aprovação explícita do owner do master final e do release commit antes de ativar Nexus-Site e substituir o HUD. `--require-ready` deve passar.

## Critérios irredutíveis para GO

São 11 gates em `RELEASE_READINESS.json`: master, Hero, pixel art Compact/Standard, 12 idles, cinco estados, atlas 360, integração HUD, integração Site, QA nativa, QA web, aprovação humana. Todos devem conter **evidências concretas e rastreáveis por SHA**. O gate `READY` exige aprovação no commit de publicação exato. Um teste de protótipo não equivale a um teste do instalador.

```bash
node scripts/verify-nex-release-readiness.mjs
# auditoria comum: passa enquanto manifesta BLOCKED de forma honesta
node scripts/verify-nex-release-readiness.mjs --require-ready
# exigência de publicação: deve falhar até todos os gates estarem PASSED
```

## Bloqueios atuais (não confundir com implementação concluída)

- **Resolvido em 29/10:** os PNGs originais, camadas, HTML, gerador e atlas candidato de 96 quadros constam na `release/v1.0.0` via [PR #147](https://github.com/IAPro-Community/nexus/pull/147). O OpenRaster é reconstruível a partir de seis fragmentos Git e validado por SHA-256; o desenho permanece candidato até aprovação visual. Não repetir importação nem depender de ZIPs da conversa.
- Master pixel art final 64/128px, 12 idles **em produção no HUD**, e 16 vistas 360° aprovadas **não estão entregues**.
- O laboratório do Nexus-Site é apenas QA isolado; a homepage não exibe o NEX.
- Não há aprovação visual definitiva ou certificação nativa abrangente. A release permanece **BLOCKED**.

## Relação com as issues

- [#119 — Master em pixel art](https://github.com/IAPro-Community/nexus/issues/119).
- [#120 — Biblioteca idle](https://github.com/IAPro-Community/nexus/issues/120).
- [#129 — Atlas 360°](https://github.com/IAPro-Community/nexus/issues/129).

**Regra para agentes:** não alegar “NEX pronto”, “no site” ou “HUD atualizado” sem gate `READY`, arte aprovada e SHA das duas integrações. Nenhuma versão parcial será liberada por conveniência.
