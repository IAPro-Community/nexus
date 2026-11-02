# CODEX — Execute a continuidade do NEX no Nexus (handoff versionado, 02/11/2026)

> **Ponto de entrada para Codex.** Este documento governa a execução da tarefa; não substitui `AGENTS.md`, contratos técnicos existentes, decisões atuais das issues ou a aprovação artística. Seu propósito é evitar recomeços, deriva visual, uso de pranchas falsas e PRs redundantes.

## Missão

Trabalhe como engenheiro sênior de frontend/animação 2D, QA e integração Wails. **Implemente e consolide o NEX como personagem oficial do IAPro Nexus**, usando o código e os assets versionados, incluindo Web HUD, Desktop HUD/Wails e preparação da biblioteca visual para Nexus-Site. Entregue código, testes, prévia fiel, evidências e PRs; não pare após produzir um plano. **Não faça merge, publicação ou mudança de gate de release sem aprovação.**

Este handoff foi criado na branch `docs/nex-codex-production-handoff-20261102`, partindo de `release/v1.0.0`; **as implementações do NEX ainda vivem em branches/PRs diferentes**. Compare os commits correntes antes de reconciliar.

## 0. Ler antes de alterar código

Leia `AGENTS.md`, `docs/design/nex/AGENTS.md`, `docs/engineering/WEB_ENGINEERING_STANDARDS.md`, `docs/engineering/ENGINEERING_STANDARDS.md`, `docs/design/NEX_32BIT_VISUAL_CONTRACT.md`, `docs/design/NEX_ANIMATION_CONTRACT.md`, `docs/design/NEX_VISUAL_MANIFEST.json`, `docs/design/nex/identity-lock.json`, `docs/design/nex/IDENTITY_GOVERNANCE.md`, `docs/design/nex/RETRO_GAME_IDLE_DIRECTION.md`, `docs/design/nex/COMPLETE_DELIVERY_PLAN.md`, `docs/design/nex/MOTION_ARCHITECTURE_V2.md`, `docs/design/nex/hybrid-v1/README.md` e `docs/design/nex/RELEASE_READINESS.json`. Consulte `docs/design/nex/COMPANION_2D_CLIPPY_LITE.md` na PR #172 se ainda não estiver no branch base.

**Precedência de escopo:** issue [#141](https://github.com/IAPro-Community/nexus/issues/141), revisada em 01/11, exige **8 comportamentos contextuais 2D componíveis**, mantém a biblioteca de **12 IDs idle**, e **adia o 360° real** da issue [#129](https://github.com/IAPro-Community/nexus/issues/129). Não ressuscite as 16 vistas e 12 sprite sheets independentes como gates obrigatórios da primeira entrega por ler documentação antiga. Entrega conjunta HUD + Nexus-Site e aprovação da identidade continuam obrigatórias.

## 1. Reconciliação — NÃO reimplementar o que existe

Audite, compare e organize as PRs antes de escrever substitutos:

- [#171](https://github.com/IAPro-Community/nexus/pull/171): `NexIdleDirector` e testes dos 12 IDs; motion e laboratório.
- [#172](https://github.com/IAPro-Community/nexus/pull/172): contrato atual Companion 2D.
- [#173](https://github.com/IAPro-Community/nexus/pull/173): `nexCompanionBehavior`, oito behaviors e dedupe de conclusão.
- [#175](https://github.com/IAPro-Community/nexus/pull/175): rig Canvas2D com pixel raster candidato e quatro órbitas.
- [#178](https://github.com/IAPro-Community/nexus/pull/178): melhorias de órbita, pivôs e transições.
- [#183](https://github.com/IAPro-Community/nexus/pull/183): correção de `COMPLETED_VERIFIED`; **stacked** sobre `feat/nex-rig-idle-orbit-polish-20261101`, não sobre a release.
- [#174](https://github.com/IAPro-Community/nexus/issues/174): divergência entre artwork selecionado e assets executáveis; **bloqueio real** da aprovação artística.
- [#119](https://github.com/IAPro-Community/nexus/issues/119) e [#120](https://github.com/IAPro-Community/nexus/issues/120): master pixel-art e catálogo de idle.

Brancha principal para conferir a implementação acumulada: `feat/nex-rig-idle-orbit-polish-20261101`. Ela pode estar divergente da `release/v1.0.0`; compare por SHA, detecte conflitos e integre seletivamente em nova branch técnica. **Não efetue merge cego, não descarte alterações concorrentes e não feche PRs com critérios pendentes.**

Mapeie os proprietários atuais:
- `web/src/features/hud/mascotModel.ts`: única projeção dos sinais operacionais reais.
- `web/src/features/hud/GuardianMascot.tsx` / `NexusHud.tsx`: integração e acessibilidade do HUD.
- `web/src/features/hud/NexPixelSprite.tsx`: fallback Compact SVG, **não** master Hero.
- `web/src/features/hud/nexIdleDirector.ts` / `nexCompanionBehavior.ts`: animação de apresentação nas branches de trabalho.
- `web/public/nex/hybrid-v1/` e `web/public/nex/rig-v1/`: PNGs/camadas reais **candidatos** e laboratório de animação.
- `docs/design/nex/hybrid-v1/ora-parts/`: fonte OpenRaster reconstruível.

## 2. Identidade visual e produção de assets: evitar falso master

O usuário selecionou o personagem da prancha editorial original (1536×1024, SHA-256 `08dcb5446a1f12a34c913104e2f9991be8b94595e1e6ccb76d1e5e2c0b351b66`), **não versionada em Git no inventário realizado**. O raster de `web/public/nex/hybrid-v1/nex_source_original.png` (1254×1254, SHA-256 `ef7a4aeb7b24d2ab5f199f913d351414b8fafb0b6208b3ab2b665013adcb9118`) está versionado, porém permanece **CANDIDATE / UNAPPROVED**, e não comprova paridade com a referência selecionada. Não invente que são a mesma imagem.

Anatomia obrigatória: capacete metálico branco arredondado, visor navy com dois olhos ciano, **duas aletas técnicas verticais e dois acoplamentos circulares**, torso compacto flutuante, **um núcleo central** e **exatamente quatro módulos magnéticos independentes**, com anel energético inferior. Não adicionar boca, braços/pernas humanos, garra, orelhas felinas, quinto módulo, cores verdes ou logotipo alterado.

A pasta de estudos gerados por IA em 02/11 foi **rejeitada/não verificada**: contém elementos felinos, membros adicionais, alterações de paleta e vistas que não são um 360° válido. **Nenhum desses PNGs foi incorporado à branch e não devem ser usados como production sprites.** Sua proveniência consta em `CODEX_ASSET_PROVENANCE_20261102.json`. Não recorte colagens conceituais e não use `rotateY` como falso turno completo.

Execute a cadeia segura:
1. `node scripts/verify-nex-hybrid-assets.mjs --require-binaries` e `node scripts/restore-nex-openraster.mjs`; confirme hashes do PNG e OpenRaster e leia `web/public/nex/hybrid-v1/manifest.json`.
2. Inspecione visualmente a imagem raster candidata, cada PNG individual, pivôs, máscaras e underpainting; verifique a real separação de cabeça/capacete, visor, aletas, torso, núcleo, anel e módulos.
3. Implemente **somente movimentos suportados pelas camadas reais**, preservando pixels e preenchendo áreas anteriormente ocluídas quando a geometria permitir. Para movimentos impossíveis sem novos desenhos, produza uma especificação de arte, previews de comparação e status `BLOCKED_ART`, sem inventar fidelidade.
4. Construa um master editável versionado (ORA ou Aseprite, atlas/JSON de pivôs) e variantes ópticas 64/92/128px somente quando os desenhos estiverem coerentes. Toda nova arte deve ter PNG RGBA, metadados, fonte, hash SHA-256 e comparação lado a lado com a referência disponível.
5. Para a prancha editorial selecionada ausente do Git, procure apenas em fontes já autorizadas do projeto (branches, anexos/artefatos de PRs e repositórios). Não paralise a engenharia mecânica por essa ausência; **não substitua nem peça uploads ao usuário como solução automática**. Registre que a validação final de paridade requer recuperar esses bytes ou obter uma decisão formal de identidade.

## 3. Estados operacionais, comportamentos e idle — três níveis diferentes

**Moods canônicos do sistema (não duplicar na engine):** `idle`, `working`, `attention`, `critical`, `offline`. `attention` é apresentado no sprite como `warning`. Somente o Attention Center/Core autoriza alertas e conclusão; os estados **não são botões artificiais de operação**.

**Oito comportamentos visuais de Companion:** `ambient`, `greet`, `look`, `thinking`, `attention`, `error`, `resolved`, `rest`. `still` é a apresentação neutra de motion-off/offline. O comportamento `resolved` só pode ser acionado por evento real e novo do Attention Center, com `COMPLETED_VERIFIED` reconhecido (#183); não festejar histórico, falha ou sucesso não verificado.

**Catálogo obrigatório de doze gestos idle:** `idle_hover`, `idle_blink`, `idle_glance`, `idle_orbit_breathe`, `idle_head_tilt`, `idle_fin_twitch`, `idle_module_inspect`, `idle_orbit_shuffle`, `idle_curiosity`, `idle_micro_stretch`, `idle_drowse`, `idle_wake`. Oito behaviors e doze IDs não exigem 20 artes separadas: componha com uma única hierarquia articulada e estados de expressão.

O `IdleDirector` controla **apenas gestos de apresentação**: seed injetável, cooldown, preempção imediata por mudanças para `working/attention/critical/offline`, uma microação especial por vez, no máximo ~2 especiais/min, silêncio e baixa distração enquanto o usuário lê código. `drowse` só após inatividade real e `wake` por interação.

**Prioridade:** motion-off / offline / invisível > critical > attention > working > evento real resolvido > interação explícita > idle automático. Pausar efeitos ao ocultar WebView, minimizar janela, usar tray, aba oculta, `prefers-reduced-motion` ou opção `motion=off`. Limpar timers e RAF ao desmontar. Em motion-off manter conteúdo textual acessível.

## 4. Runtime, UX e superfícies

- Preservar BAR / MASCOT / TRAY, alça nativa de arraste e hit-regions do Wails, popup, foco, snooze e click-through. **Não usar drag do Framer Motion na janela nativa**.
- Render 2D leve: camadas PNG com CSS/WAAPI ou Canvas2D existentes; sem Three.js, engine 3D, novo polling, canal IPC ou LLM. Não adicionar Framer Motion como dependência sem justificar a necessidade diante da engine existente.
- Web: interações por hover, foco, teclado, ponteiro e clique usando controles semânticos; parallax discreto, quatro esferas com pivôs estáveis, frente/trás, sem semitransparência incorreta durante oclusão.
- Desktop: animação interna independente da geometria da janela; redução de movimentos e visibilidade conforme Wails; compatibilidade com Windows/macOS/Linux X11/Wayland deve ser **testada realmente** e não presumida.
- Site: laboratório privado/noindex antes de homepage. Integração HUD + Site permanece uma única decisão de publicação, mas os protótipos podem ser revisados sem publicação; não alterar `RELEASE_READINESS.json` para READY sem aceite humano no SHA exato.
- Mantenha código TypeScript strict, 1 componente conceitual por arquivo, SCSS Modules, tokens semânticos `--nx-*`, i18n PT-BR/EN/ES, WCAG 2.2 AA e o design system existente, como exige `AGENTS.md`.

## 5. Entrega incrementada e testes

Primeiro entregue o **inventário real**: base/head/SHA/PRs e quais componentes/movimentos funcionam de fato, distinguindo unit tests de render real. Em seguida implemente incrementos que remediem falhas demonstráveis, documentando dependências e evitando trabalho redundante.

Execute conforme disponível (sem simular resultados):
```bash
node scripts/verify-nex-visual.mjs
node scripts/verify-nex-identity-lock.mjs
node scripts/verify-nex-hybrid-assets.mjs --require-binaries
node scripts/verify-nex-release-readiness.mjs
bun install --cwd web --frozen-lockfile
bun run --cwd web typecheck
bun run --cwd web lint
bun run --cwd web test
bun run --cwd web build
bun run --cwd web test:e2e:hud
# Descubra scripts extras reais do package.json nas branches de animação:
# test:nex:rig, art:nex:hybrid:qa, art:nex:motion:qa (se existirem)
```

Atente aos comandos efetivamente definidos no checkout: `quality`/`quality:full` e `check:styles` também são obrigatórios pelo contrato. Verifique animações com captura real em 390/768/1280px, e representação 48/64/92/128px. Teste cinco moods, oito behaviors e 12 IDs, mudança de estados no meio do gesto, conclusão `COMPLETED_VERIFIED` sem replay, teclado, `prefers-reduced-motion`, zero vazamentos de timers, visibilidade, oclusão dos quatro módulos, contrastes e estabilidade de layout. Em native, comprove DPI, transparência, click-through, drag e tray apenas nas plataformas executadas.

## 6. Aceite e relatório obrigatório

A entrega técnica precisa conter: caminho de todas as alterações, commits, PRs base/head, status e screenshots/GIFs reais de cada ação, matriz **5 moods × 8 behaviors × 12 IDs** discriminando `IMPLEMENTED_ART`, `CSS_PROTOTYPE`, `TEST_ONLY` ou `BLOCKED_ART`; SHA-256 e procedência dos PNGs; testes executados e logs; limitações de performance; riscos de integração Nexus-Site/Wails; evidências e o estado honestamente `BLOCKED` até homologação.

**Não marque a release como pronta, não desative checks, não publique assets rejeitados, não crie um novo mascote e não espere instruções adicionais para tarefas executáveis.** Abra PRs draft incrementais e encerre cada ciclo com progresso real comprovado. Use o orçamento de tokens com foco em código correto, reuso e QA, não em gerar intermináveis variações de prompts.
