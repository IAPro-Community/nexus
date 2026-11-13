# NEX — Contrato oficial de animação v1

**Estado:** especificação de movimento oficial para o mascote da linha `release/v1.0.0`. **Não significa certificação de binários.**

**Documento-mãe:** [NEX — contrato visual](NEX_32BIT_VISUAL_CONTRACT.md). **Arte de execução:** o atlas locked `web/public/nex/hud-32bit/nex-idle-atlas-128.png` renderizado por `web/src/features/hud/NexPixelSprite.tsx`. **Estilos:** `web/src/features/hud/NexPixelSprite.module.scss` e `web/src/features/hud/GuardianMascot.module.scss`.

> **Evolução artística escolhida:** a implementação CSS/SVG anterior foi
> substituída pelo **atlas raster locked NEX-ID-v1** (`hud_32bit_idle_atlas`,
> 12 ações × 8 frames em 1024×1536), renderizado por `NexPixelSprite.tsx` com
> `steps()` CSS por linha do atlas. A referência do personagem e os critérios
> de substituição estão em [Retro Game + Idle](nex/RETRO_GAME_IDLE_DIRECTION.md).
> Os 12 ids de ação já têm frames versionados no atlas locked; o aceite visual
> do render no HUD segue pelo gate `owner_visual_approval`.

## Intenção

NEX mantém a **vista frontal**, a mesma cabeça, visor, aletas, núcleo e **exatamente quatro** módulos flutuantes. A animação apresenta estados existentes, mas não torna NEX um agente, assistente autônomo ou sensor de atividade. Nenhum frame deve alterar a anatomia, a perspectiva frontal ou o modo do HUD.

Na v1, NEX usa **playback de frames do atlas raster locked** (12 ações de idle derivadas do PNG aprovado): **não** é um sprite sheet desenhado à mão em pixel art nem uma cópia perfeita da prancha conceitual. O termo “32 bits” descreve a direção estética. Não afirmar que há frames desenhados manualmente ou giro 360° quando apenas os shifts de camada do raster estiverem disponíveis.

## Anatomia articulada e camadas independentes

Na execução atual (atlas raster locked NEX-ID-v1), o movimento vem dos
**frames do próprio raster aprovado** (shifts de camadas do PNG original,
não desenho quadro a quadro): cada ação de idle é uma linha de 8 frames
128×128 tocada em 8 fps via `steps()`. O wrapper do sprite é âncora e não
recebe transform de animação; as regras de estado abaixo continuam válidas
como contrato semântico.

| Camada/estado | Âncora/pivô | Movimento autorizado | Limite |
| --- | --- | --- | --- |
| Sprite do atlas (`data-nex-art="hud-32bit-atlas"`) | centro / base | sem transform de âncora | nenhum deslocamento do wrapper |
| Linha de ação (idle_hover … idle_wake) | frame interno | playback de 8 frames, 125 ms cada | loop somente no hover (frame 8 = frame 1) |
| Drowse/wake | frame interno | segurar o último frame até o director trocar | não sinalizar sono real do sistema |
| Mood (`data-nex-mood`) | overlay de tratamento | véu colorido soft-light; offline dessaturado e imóvel | sem frames dedicados de mood ainda; nunca inferir sucesso |
| Working | frame interno | respiração vertical 1.3s de 2px | não indica progresso mensurável |

Os módulos, aletas, olhos e núcleo continuam presentes **dentro de cada
frame do raster aprovado** — a anatomia é garantida pelo SHA do atlas no
`identity-lock.json`, não por seletores de camada SVG.

## Contrato por estado: apenas evidência real

| Estado renderizado | Proveniência | Animação | Cuidados |
| --- | --- | --- | --- |
| `idle` | projeção disponível, nenhuma operação nem atenção | flutuação 3.6s; piscada 6.5s; aletas 4.8s; módulos alternados 3.4s | baixa distração |
| `working` | execução de agente ou missão em andamento | oscilação do corpo 1.3s, núcleo 1.1s; módulos em fases | não indica progresso percentual |
| `warning` | `CompanionMood.attention` (severidade real) | olhos laranja, sinal 0.9s e breve gesto lateral | não emitir alerta a partir de quota desconhecida |
| `critical` | intervenção crítica projetada | olhos vermelho-rosa, pulso moderado 1.1s; aletas estabilizadas | sem flashes agressivos |
| `offline` | fonte sem disponibilidade confirmada | imóvel, olhos desativados/dessaturados | nenhum sinal falso de trabalho |

Em `GuardianMascot.tsx`, `CompanionMood.attention` é apresentado como `GuardianMood.warning`. O estado `critical` só ocorre com severidade crítica na projeção do Companion. Estados `thinking`, `routing` e `success` são conceitos para versões futuras, não contratos do runtime 1.0.

## Política de acessibilidade

- `data-motion=off` desativa **toda** animação CSS dentro do sprite, incluindo as animações mais específicas de `working` e `critical`, o hover no botão e a animação de entrada da camada do Guardian.
- `prefers-reduced-motion: reduce` oferece o mesmo congelamento mesmo quando `data-motion=on`.
- O modo offline suspende também os movimentos de corpo, aletas, olhos, módulos, órbitas, reator e sombra.
- Informações de atenção continuam em texto acessível, botões e indicador; não depender de cor/piscada. A cor não é prova de conclusão.
- A animação é puramente visual: sem `requestAnimationFrame`, `setInterval`, novo canal IPC, chamadas de provider ou animação da janela nativa.
- Eventos de foco, hover e popup não devem afetar o Core nem a área nativa de hit-testing; a alça continua separada.

## Usos nas diferentes superfícies

**Compact:** HUD em 92×92 CSS px no tamanho padrão; tamanho confortável/grande usa o mesmo master SVG, mas exige inspeção óptica. A animação é mínima e opcional.

**Standard:** usar a identidade frontal em onboarding/docs com movimentos mais explicativos, mantendo os mesmos pivôs e cinco estados reais. Ainda não existe master artístico Standard certificado.

**Hero:** movimentos de apresentação do site podem usar parallax/partículas e transições, mas **não** inventar gestos humanos nem modificar os cinco estados da v1. Não replicar uma imagem da prancha como frames diferentes sem mesmo master em camadas.

## Prévia local sem executar o HUD nativo

O sprite do HUD renderiza o **atlas raster locked NEX-ID-v1**
(`web/public/nex/hud-32bit/nex-idle-atlas-128.png`, SHA verificado contra
`docs/design/nex/identity-lock.json`) por meio de `NexPixelSprite.tsx` +
`NexPixelSprite.module.scss`. A prévia local é o lab E2E do Companion, que
exercita o HUD real e captura os oito comportamentos:

```bash
cd web
npm run test:e2e:companion-lab
```

As capturas ficam em `DEV/validation/current/companion-lab/captures/`. O
export estático de SVGs da direção vetorial anterior foi aposentado junto com
o desenho rejeitado; o raster locked é a arte canônica.

Essa página é apenas um auxílio à inspeção visual. Ela não representa o comportamento das janelas nativas nem homologa DPI, click-through, transparência, arraste ou bandeja.

## Critérios de homologação

1. `web/src/features/hud/GuardianMascot.test.tsx` valida duas aletas, quatro módulos e posicionamento de âncora no DOM.
2. `node scripts/verify-nex-visual.mjs` valida contrato, classes de animação, acessibilidade e a correspondência com estados canônicos.
3. `bun run --cwd web quality:full` e build web no SHA exato.
4. Capturar o NEX real em `idle`, `working`, `warning`, `critical`, `offline` e `motion=off`, no mínimo em 100%, 125% e 150% de DPI.
5. Verificar Windows, macOS, Linux X11 e Wayland quanto a transparência, popup, drag, hit region, tray, clique externo e ausência de "pixel shimmer".
6. A revisão de identidade usa `docs/design/assets/nex/` e a prancha original quando sua ingestão binária estiver concluída. Inspeção documental **não** equivale a GO nativo.

**Gate de release:** não promover binário/tags enquanto smoke gráfico, CI e gates nativos permanecerem FAIL/NOT_RUN.
