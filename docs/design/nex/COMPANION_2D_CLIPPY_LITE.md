# NEX Companion 2D — direção Clippy-like (MVP, 2026-11-01)

> **Decisão de produto vigente:** substituir o requisito de giro 360°/motor 3D e de doze artes de animação exclusivas por um **assistente contextual 2D expressivo, discreto e leve**. A implementação pode conservar o catálogo técnico de 12 `idle_*` existente, mas a entrega **não depende de desenhar 12 spritesheets independentes**. O objetivo é a sensação de personagem vivo do antigo Office Assistant (Clippy), **sem copiar os assets, voz, personalidade ou marca da Microsoft**.

## 1. Objetivo e não objetivos

- Reutilizar **o NEX escolhido**, não criar novo mascote: capacete branco arredondado, visor navy com olhos, duas aletas, dois acoplamentos, torso compacto, reator central, quatro módulos orbitais e anel inferior.
- Uma animação deve comunicar **atenção ou personalidade**, nunca fingir que o Core executou trabalho, completou tarefa, detectou falha ou recebeu uma mensagem.
- Entregar **oito comportamentos visuais componíveis** no HUD, com apresentação correspondente no site e uma mesma linguagem visual. A experiência deve ser agradável após muitas horas.
- **Não usar Three.js/WebGL como requisito**, nem gerar 16 vistas 360° nesta entrega. Não introduzir Rive/PixiJS por padrão. CSS Modules + SVG/raster em camadas + WAAPI quando necessário são suficientes enquanto os perfis não demonstram problema. Canvas 2D é alternativa somente com medições. Rive é alternativa de autoria, não uma dependência automática.
- Não reescrever `NexIdleDirector`, `NexPixelSprite`, `GuardianMascot` ou o modelo de atenção sem uma justificativa com teste de regressão. **PR #171 é o trabalho prioritário a reaproveitar**, reconciliar seus commits antes de qualquer substituição.
- O 360° da issue #129 é **PARKED/FUTURE**, fora do gate do MVP. O player futuro pode continuar no laboratório, sem homepage pública e sem falsa afirmação de implementação.

## 2. Oito comportamentos obrigatórios

Os oito IDs abaixo são **comportamentos UX**, compostos de microgestos já existentes. Não significam oito arquivos pesados ou oito máquinas de estados.

| ID | Gatilho autoritativo | Composição visual | Política |
|---|---|---|---|
| `ambient` | Mood `idle`, visível, motion on | `idle_hover`, piscar discreto, respiração dos quatro módulos | Sem mudança de semântica |
| `greet` | Entrada, hover/focus ou clique **real** do usuário | Inclinação breve e piscada; voltar ao repouso | Cooldown por sessão; clique mantém ação atual do HUD |
| `look` | Cursor local dentro da hitbox existente / foco de teclado | Olhar ± pequeno deslocamento no visor; reset ao sair | Não instalar listener global de cursor |
| `thinking` | Mood `working` oriundo do `mascotMood` atual | Olhos concentrados, pulso leve do núcleo, órbita rítmica | Sem porcentagem ou barra de progresso inventada |
| `attention` | Item existente em `projectMascotAttention` | Módulo/aceno breve na direção da bolha, alerta canônico | Bolha/actions atuais; NÃO abrir/popular conteúdo automaticamente |
| `error` | Evidência real `critical`/failure; não apenas latência | Expressão preocupada, pulso sutil; sem flashing contínuo | Preempta idles; manter acesso à causa |
| `resolved` | Evento de conclusão positiva explícita e com ID estável **se disponível** | Microcomemoração única, não repetitiva | Se não houver evento real/ID, não animar; nunca deduzir por redução de contagem |
| `rest` | Idle prolongado, seguido de interação visível | Sonolência `idle_drowse` + `idle_wake` | Diferente de offline; despertar só com interação |

**Preservar os cinco moods operacionais canônicos**: `idle`, `working`, `attention/warning`, `critical`, `offline`. Comportamentos de UX são **projeções visuais locais**; não criar mais um Core state machine ou provider polling.

**Observação importante:** `resolved` exige um evento comprovado; sua ausência deve ser reportada como **não conectado**, nunca encenada usando dados fake na produção. Um laboratório pode ter controles explícitos de simulação, isolados do runtime.

## 3. Fontes de verdade e precedência

1. `mascotModel.ts` / `projectMascotAttention` / `mascotMood` e APIs existentes determinam estado operacional. Não inventar disponibilidade ou resultado.
2. `NexIdleDirector` já controla microações em idle, incluindo visibilidade, cooldown, seed reprodutível e cancelamento. Reusar; evitar segundo relógio e RAF permanente.
3. Comportamentos interativos são transitórios: `offline > critical > attention > working > interações do usuário > idle`. Em `offline` não há movimento. Erro/atenção nunca são mascarados por `greet` ou `resolved`.
4. Dedupe por ID de evento/correlação *quando houver*, não somente por timestamps de render. Respeitar `snooze`, dismissal, foco, contexto de projeto e visibilidade existente.
5. Nenhum clique no mascote aprova execução, cria missão, dispara agentes ou confirma gasto sem a ação explícita da UI existente.

## 4. Arquitetura e integração mínima

**HUD (React/Wails)**

- `GuardianMascot` permanece dono do componente e da ação de abrir/recolher. Incrementar os sinais de `data-` e props existentes, sem listeners globais desnecessários, novas janelas, hitboxes ampliadas ou alterações no drag nativo.
- A execução do `NexPixelSprite` é o **atlas raster locked NEX-ID-v1** (`hud_32bit_idle_atlas`) — não é fallback. As camadas raster de `web/public/nex/hybrid-v1/` pertencem ao hero do Nexus-Site e só entram em outra superfície após comparação visual (fonte, SHA, recortes e transparência). Um PNG editorial não é automaticamente um sprite pixel-art.
- Oito comportamentos são mapeados com função pura `state+event -> behavior` e animações locais; restringir a no máximo um gesto especial por vez. Se o PR #171 já oferece transições equivalentes, usar seu contrato sem criar duplicatas.
- Preservar BAR/MASCOT/TRAY, atalhos de teclado, foco, `aria-label`, popover, `data-hud-hit-area` e preferências de motion.

**Site (Nexus-Site)**

- Reutilizar o Hero/QA já existentes; lazy-load de camadas quando visível. Reações no site são **demonstrações claramente rotuladas**, nunca telemetria ao vivo se não houver fonte autorizada.
- O laboratório pode simular os oito comportamentos. Homepage e publicação ficam condicionadas ao gate conjunto e ao aceite visual; não ativar o Nex no público apenas por um PR de documentação.

**UI/UX de assistente Clippy-like**

- O mascote fica **quieto por padrão**: não segue o mouse pelo desktop, não intercepta cliques, não invade código e não fala sozinho.
- Balões contextuais só para pendência real ou solicitação explícita, com ações abrir/adiar/fechar. Evitar avisos repetitivos, animações compulsivas e notificações sem solução.
- Sem som por padrão. Interação por mouse, touch e teclado. `prefers-reduced-motion` e opção manual congelam ações; sem dependência em WebGL.

## 5. Economia de execução / política de assets

1. **Não gerar imagens por prompt a cada estado**, não instalar Three.js, não encomendar avatar 3D, não iniciar trabalho de 16 vistas.
2. Inventariar e medir o que existe no PR #171, nos assets da PR #147, no PR de 16 frames frontais #160 e no laboratório do site. Reaproveitar somente o que preservar anatomia e proveniência.
3. Entregar primeiro as combinações de gestos mapeadas para as ações do atlas locked existente. Ativos adicionais precisam justificar melhora em comparação a uma captura em tamanho real 64/92/128px e custo/complexidade mensuráveis.
4. Rive só deve ser adotado após um POC com a arte real, licença/deployment/documentação definidos e comparação concreta de bundle, renderização e manutenção. Não duplicar arte entre Rive e código.

## 6. Critérios de aceite (evidência por SHA)

- [ ] Oito comportamentos testáveis em laboratório (incluindo `resolved` com mock explicitamente marcado), sem transformar mock em estado real.
- [ ] Cinco moods operacionais reais com preempção correta e `offline` estático.
- [ ] `GuardianMascot` e `NexIdleDirector` atuais reutilizados; sem segundo motor, polling ou dependência nova sem justificativa.
- [ ] Foco, clique, toque, snooze, drag, click-through, popover e preferências de motion preservados.
- [ ] Timer/RAF/observer cleanup no unmount, tab oculta e offscreen; sem reprodução de eventos antigos após retomada.
- [ ] Comparação visual 64/92/128px e Hero responsivo, fidelidade ao NEX aprovado e imagens fonte com hashes.
- [ ] Testes unitários/componente, Playwright Chrome/mobile e QA nativa em plataformas realmente executadas (registrar NOT_RUN nas demais).
- [ ] Renderização reduzida/parada por `prefers-reduced-motion`; WCAG 2.2 AA.
- [ ] Gate conjunto `RELEASE_READINESS.json` mantém `BLOCKED` até HUD **e** site homologados e aceite humano no SHA exato.
- [ ] A revisão de escopo **não** converte o 360° em falso `PASSED`; ele sai do gate obrigatório e permanece rastreável como trabalho futuro.

## 7. Sequência de execução (para Codex)

1. **STOP de expansão de escopo**: não gerar 3D/360 e não fazer novos motores; inspecionar PR #171, branch de release e os assets já versionados.
2. Reusar o scheduler/visual atual e implementar a integração incremental dos oito comportamentos; um PR pequeno para interação e testes.
3. Montar laboratório acessível de oito estados no HUD; revisar arte e animação nas dimensões reais, com screenshots comparativas.
4. Reusar o mesmo contrato no laboratório do Nexus-Site; implementar lazy-load e controles demonstração/testes sem expor estado fictício em produção.
5. Atualizar a matriz de QA e evidências dos onze gates; solicitar aceite visual humano apenas quando existirem capturas reais por commit.
6. **Não fazer merge automático** de PRs experimentais e não tocar em branches que o Codex esteja modificando concorrencialmente.

## 8. Decisões de escopo e compatibilidade

- A decisão de **2026-11-01** substitui, para este MVP, a exigência anterior de 28/10 de "16 vistas/360 obrigatório". A exigência de uma identidade única e homologação conjunta de HUD + site **permanece**.
- A issue #120 e o catálogo de 12 microações não são descartados: o código já escrito pode ser aproveitado, mas a arte final individual de 12 movimentos não bloqueia o MVP.
- A issue #129 não deve ser encerrada como entregue; é backlog futuro, sem efeito no manifesto da release simplificada.
- A issue #141 continua umbrella para a entrega nova, com este contrato anexado e rastreabilidade das mudanças.
