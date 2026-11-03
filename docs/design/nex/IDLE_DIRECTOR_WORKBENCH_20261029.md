# NEX — 12 ações idle: motor reutilizável e laboratório visual

**Escopo deste incremento:** código de coreografia independente em `web/src/features/hud/nexIdleDirector.ts`, testes determinísticos e um laboratório HTML/PNG produzido na conversa, **não instalado no HUD nem no site público**.

## O que é reproduzível

- `NEX_IDLE_ACTIONS` tem 12 IDs canônicos definidos em `RETRO_GAME_IDLE_DIRECTION.md`.
- `NexIdleDirector` permite ambiente `idle/working/warning/critical/offline`, sinal de visibilidade, preferência explícita e `reducedMotion`.
- Motor local por `setTimeout` (uma cadeia de timers, sem polling de Core ou Maestro), PRNG com semente estável, não repete a ação imediatamente.
- Microação especial somente quando o humor projetado é `idle`; eventos reais de trabalho/atenção/offline preemptam, limpam timers e zeram a pose.
- Longa inatividade (>3 minutos) pode disparar `idle_drowse`; `idle_wake` depende de interação efetiva.
- 24–36 s entre ações especiais, evitando repetição excessiva; modos com movimento reduzido/oculto param.
- Testes unitários do scheduler em `nexIdleDirector.test.ts`.

## Laboratório com os 12 gestos

Um ZIP `NEX_12_Idles_Workbench.zip` foi criado nesta conversa (SHA-256 `9021960dc1416c665d9f1834afd11e42f6479b489ad88769b9e15efa01b7b20c`) com **PNG original byte-identical**, olhos, aletas e quatro módulos extraídos, HTML com 12 controles e catálogo JSON.

O Chromium real carregou **10 camadas de imagem (zero quebradas)**, expôs 12 botões e 4 módulos, acionou manualmente os seis gestos novos `module_inspect`, `orbit_shuffle`, `curiosity`, `micro_stretch`, `drowse`, `wake`, e não produziu erro JavaScript. `motion=off` resultou em `animation-name=none`; `critical` interrompeu a ação ativa.

**ATENÇÃO:** ações do HTML são transformações CSS sobre pixels originais; isso **não** equivale a sprite sheet desenhada quadro a quadro, nem a animação de cabeça separada com underpainting. A fonte final editável deve ser validada no master.

## Comparação de escalas (candidatos, não masters)

Foi gerado `NEX_Pixel_Scale_Candidates.zip` (SHA-256 `9eeeb6dfa2e2cb425d39969802bf13bfc025a99df0d37f230ee2e44862404785`) com três amostras de escala 128×128, 64×64 e 32×32. A leitura da silhueta foi preservada, porém existem perdas de material, separação de aletas e pixel clusters intencionais. **Não** usar esses arquivos reduzidos como master final do HUD; isso exige redesenho óptico em camadas.

## Estado honesto dos gates

- `docs/design/nex/RELEASE_READINESS.json` continua `BLOCKED`; a entrega não foi promovida a `PASSED`.
- **Fonte artística e atlas 96 quadros agora versionados:** PNG original, camadas reais e atlas experimental estão na `release/v1.0.0` após PR #147. O HTML específico do workbench de 12 botões e os PNGs de escala reduzida continuam como demonstradores externos, não como assets de produção. Os 12 quadros finais hand-drawn e sua integração ao HUD permanecem pendentes.
- O director encontra-se versionado como código testável, mas **ainda não conectado ao GuardianMascot** — aguarda atlas pixel art aprovado para evitar exposição de animação provisória.
- `360°` segue condicionado às 16 vistas coerentes, inclusive costas e oclusão, não à distorção do frontal.
- A publicação do Nexus-Site permanece bloqueada conforme a PR #8.

## Próximos gates

Incorporar os arquivos reais via GitHub com hash, revisar master editável e manter grid/proporções entre 64/128/Hero, criar 16 views 360°, atlas + JSON, conectar renderer do Site e HUD ao mesmo catálogo, testar redução de movimento e nativo, obter aprovação explícita do owner para o SHA da release.
