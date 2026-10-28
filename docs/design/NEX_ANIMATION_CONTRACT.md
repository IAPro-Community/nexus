# NEX — Contrato oficial de animação v1

**Estado:** especificação de movimento oficial para o mascote da linha `release/v1.0.0`. **Não significa certificação de binários.**

**Documento-mãe:** [NEX — contrato visual](NEX_32BIT_VISUAL_CONTRACT.md). **Arte de execução:** `web/src/features/hud/NexPixelSprite.tsx`. **Estilos:** `web/src/features/hud/_hudGuardian.scss` e `web/src/features/hud/GuardianMascot.module.scss`.

> **Evolução artística escolhida:** a implementação CSS/SVG atual continua funcional, mas deve ser substituída por **sprites HD de pixel art de jogo retrô, com 12 ações de idle**. A referência do personagem e os critérios de substituição estão em [Retro Game + Idle](nex/RETRO_GAME_IDLE_DIRECTION.md). Não marcar as 12 ações como implementadas antes de existirem frames versionados e preview do mesmo SHA.

## Intenção

NEX mantém a **vista frontal**, a mesma cabeça, visor, aletas, núcleo e **exatamente quatro** módulos flutuantes. A animação apresenta estados existentes, mas não torna NEX um agente, assistente autônomo ou sensor de atividade. Nenhum frame deve alterar a anatomia, a perspectiva frontal ou o modo do HUD.

Na v1, NEX usa **animação CSS vetorial suave** sobre o SVG de 128 × 128 unidades: **não** é um sprite sheet em pixel art quadro a quadro nem uma cópia perfeita da prancha raster conceitual. O termo “32 bits” descreve a direção estética. Não afirmar que há frames desenhados ou atlas completos quando apenas CSS/SVG estiver disponível.

## Anatomia articulada e camadas independentes

| Camada do SVG | Âncora/pivô | Movimento autorizado | Limite |
| --- | --- | --- | --- |
| Personagem frontal (`guardianSprite`) | centro / base | flutuação 0–3px; trabalho ±2° | não girar para 3/4 como pose padrão |
| Aleta esquerda (`data-nex-fin=left`) | (39,31) | balanço até −2° | preservar os dois lados |
| Aleta direita (`data-nex-fin=right`) | (89,31) | balanço espelhado até +2° | sem deslocamento da raiz |
| Olhos (`guardianEyes`) | (64,59) | piscada pontual; expressão muda conforme mood | não criar boca/nariz |
| Núcleo (`nexReactor`) | (64,99) | luminosidade/escala sutil; trabalho 1.1s | pulso não significa progresso mensurável |
| Módulos 1–4 (`data-nex-module=1..4`) | transform de âncora externo | deslocamento local vertical 1–2px em fases independentes | não animar o `translate(x,y)` externo |
| Órbita traseira (`guardianSignal`) | geometria fixa | oscilação de opacidade | não cruzar o rosto |
| Órbita frontal (`nexOrbitFront`) | geometria fixa | brilho complementar | não alterar anatomia |
| Sombra (`nexShadow`) | (64,123) | respiração sincronizada com flutuação | sem sombra quadrada da janela nativa |

Os módulos continuam nos pontos **(14,96), (114,96), (30,115), (98,115)** em seu grupo SVG externo. Somente a camada interna se movimenta; isso elimina saltos de posição e regressões ao trocar de estado.

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

Após instalar as dependências do `web/`, execute:

```bash
cd web
npm run art:nex:preview
```

O comando exporta os cinco SVGs a partir do **mesmo** `NexPixelSprite.tsx`, compila o SCSS real de animações e gera `web/dist/nex-motion-preview.html`. Abra esse arquivo no navegador, troque entre os cinco estados e teste a opção **Desativar animações**.

Essa página é apenas um auxílio à inspeção visual. Ela não representa o comportamento das janelas nativas nem homologa DPI, click-through, transparência, arraste ou bandeja.

## Critérios de homologação

1. `web/src/features/hud/GuardianMascot.test.tsx` valida duas aletas, quatro módulos e posicionamento de âncora no DOM.
2. `node scripts/verify-nex-visual.mjs` valida contrato, classes de animação, acessibilidade e a correspondência com estados canônicos.
3. `bun run --cwd web quality:full` e build web no SHA exato.
4. Capturar o NEX real em `idle`, `working`, `warning`, `critical`, `offline` e `motion=off`, no mínimo em 100%, 125% e 150% de DPI.
5. Verificar Windows, macOS, Linux X11 e Wayland quanto a transparência, popup, drag, hit region, tray, clique externo e ausência de "pixel shimmer".
6. A revisão de identidade usa `docs/design/assets/nex/` e a prancha original quando sua ingestão binária estiver concluída. Inspeção documental **não** equivale a GO nativo.

**Gate de release:** não promover binário/tags enquanto smoke gráfico, CI e gates nativos permanecerem FAIL/NOT_RUN.
