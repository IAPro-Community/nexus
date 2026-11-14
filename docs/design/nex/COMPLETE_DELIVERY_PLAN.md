# NEX 1.0 — Entrega conjunta HUD + Site (Companion 2D)

**Direção atualizada em 2026-11-01:** o objetivo passa a ser um **assistente 2D contextual e expressivo, inspirado no comportamento do Office Assistant/Clippy**, preservando o robô NEX original. O giro 360° e um runtime Three.js **saem do escopo obrigatório** do MVP. Esta decisão substitui somente a obrigatoriedade anterior de 16 vistas, mantendo a necessidade de uma experiência coerente em HUD e Nexus-Site, fidelidade visual e homologação com aprovação humana.

**Contrato vigente:** [COMPANION_2D_CLIPPY_LITE.md](COMPANION_2D_CLIPPY_LITE.md). **Catálogo histórico de 12 idles:** [RETRO_GAME_IDLE_DIRECTION.md](RETRO_GAME_IDLE_DIRECTION.md) — pode permanecer implementado e evoluir, sem exigir 12 sprite sheets exclusivos. **Giro futuro:** [MOTION_ARCHITECTURE_V2.md](MOTION_ARCHITECTURE_V2.md) / issue #129.

## Por que reduzir

O repositório já contém a execução do atlas raster locked (`NexIdleDirector`, `NexPixelSprite`, `GuardianMascot`), referências raster verificáveis, um laboratório de Hero e um catálogo de gestos. Produzir um modelo 3D, 16 novas vistas ou outro motor antes de provar a usabilidade gastaria recursos sem resolver o problema principal: reagir com clareza aos estados reais do produto.

## Fonte de identidade e de estados

- Personagem escolhido: [identity-lock.json](identity-lock.json), [referência visual](hybrid-v1/README.md), sem copiar o Clippy nem reinventar a anatomia do Nex.
- Fonte operacional: `mascotModel.ts` + Centro de Atenção, sem novas consultas ao provider e sem estados fictícios.
- Núcleo visual: `NexIdleDirector`, `NexPixelSprite`, `GuardianMascot`; animação 2D por playback do atlas raster locked + CSS/WAAPI. Sem dependências de render pesadas como requisito.
- Oito comportamentos UX: `ambient`, `greet`, `look`, `thinking`, `attention`, `error`, `resolved`, `rest`. A reação `resolved` depende estritamente de um evento de conclusão real e deduplicável; na sua ausência, não tocar a animação em produção.
- Cinco moods operacionais: `idle`, `working`, `attention/warning`, `critical`, `offline`, com preempção e movimento desativado no offline.

## Etapas implementáveis

### A — Reuso e baseline
Conferir PR #171 e branches recentes antes de tocar código; preservar assets do PR #147 e CSS/frames existentes. Proibir motor 3D novo, segunda máquina de estado e sprites artificiais de "360°". Tirar snapshots do HUD existente para comparação.

### B — Oito comportamentos composicionais
Mapear eventos reais + microgestos em uma função pura. Não gerar oito conjuntos de arte. Implementar idle suave, olhar/cumprimentar, trabalho, atenção, erro, conclusão verificável e repouso/despertar, com reset e cancelamento correto. Manter bolhas, snooze, atalhos, drag e hit regions do HUD intactos.

### C — Integração 2D HUD + Site
No HUD, garantir leitura nos tamanhos Compact/Standard usando fallback atual ou camadas fidelizadas. No site, reutilizar o laboratório Hero como demo com oito estados e labels de simulação. Sem colocar o Nex na homepage antes de passar os gates conjuntos.

### D — QA e aceite
Verificar testes do scheduler/cleanup, cinco moods, oito cenários, teclado, touch, `prefers-reduced-motion`, visibilidade/offscreen, real browser 390px, ergonomia do HUD nativo, uso de CPU e fidelidade em 64/92/128px. Registrar o que **não foi executado**, e obter aprovação visual humana no SHA final.

## Gate de publicação

Continuam onze gates em [RELEASE_READINESS.json](RELEASE_READINESS.json), todos inicialmente `BLOCKED`: master, Hero, Compact/Standard, oito comportamentos, cinco moods, segurança de interação, HUD, Site, QA nativa, QA web e aceite humano. `--require-ready` deve **falhar** enquanto qualquer gate estiver bloqueado. Os artefatos não são publicados como completos sem aprovação explícita.

A exigência antiga `turnaround_360` e o gate de "12 sprite sheets" **foram substituídos no MVP**, não marcados `PASSED`. O catálogo e o histórico permanecem nos documentos anteriores.

```bash
node scripts/verify-nex-release-readiness.mjs
node scripts/verify-nex-release-readiness.mjs --require-ready  # falha até GO
```

## Rastreabilidade

- #141 — umbrella, atualizar requisitos para Companion 2D.
- #119 — identidade, source e Compact/Standard continuam.
- #120 — catálogo e scheduler de 12 idles existentes; apenas 8 comportamentos UX no MVP.
- #129 — **futuro**, não bloqueia Companion 2D.
- PR #171 — base prioritária de integração já em desenvolvimento; reconciliar antes de desenvolver outro motor.
- Nexus-Site — laboratório existente permanece privado/noindex até revisão e homologação.

**Regra:** não afirmar "completo", "publicado" ou "homologado" com base em documentação ou teste isolado. Não alterar o estado canônico do Core para viabilizar animação.
