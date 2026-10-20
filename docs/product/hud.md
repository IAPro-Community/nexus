# Nexus HUD

> **Estado atual:** implementação do HUD está na linha `release/v1.0.0`. As referências a M1 e branches anteriores abaixo são histórico de design, não o estado de entrega. A certificação do HUD desktop exige execução nativa no mesmo SHA em Linux, Windows e macOS; veja [suporte de plataformas](../operations/platform-support.md).
>
> O Nexus HUD é uma superfície de apresentação. Ele não é um segundo Capacity
> Monitor, scheduler, cost engine, runtime de Agents ou MissionRunner.

## Objetivo

O HUD mantém o estado operacional do Nexus visível sem exigir que o usuário
permaneça na superfície Usage. Existem duas experiências oficiais:

- **Web HUD**: superfície persistente no Nexus Web OS, com adaptação para mobile.
- **Desktop HUD**: gadget nativo independente da janela principal, conectado ao
  mesmo Core/API.

A arquitetura é:

```text
Capacity Monitor ─────┐
ResourceScheduler ────┤
Cost Intelligence ────┤
Agents / Runtime ─────┤
MissionRunner ─────────┤
                      ↓
                 HUD Projection
                      │
              ┌───────┴───────┐
              ↓               ↓
          Desktop HUD       Web HUD
```

## Regras de honestidade

- `UNKNOWN` permanece `UNKNOWN`.
- dados stale/unavailable não são apresentados como atuais;
- custo `PROVIDER_REPORTED`, `API_EQUIVALENT_ESTIMATE`, `INFERRED` e
  `UNKNOWN` preserva a proveniência do ledger;
- o HUD não executa provider CLIs e não cria probes;
- alertas usam o estado/triagem canônicos do Capacity Monitor;
- recomendações usam o ResourceScheduler; o frontend não recalcula ranking;
- sucesso de Mission deriva do MissionRunner/evidence, nunca do texto do modelo;
- o HUD não utiliza LLM.

## M1

A primeira vertical slice deve conter:

1. projeção backend read-only e testável;
2. Web HUD persistente, com modo compacto e expansão;
3. adaptação mobile apropriada;
4. HUD Desktop Linux independente da janela principal;
5. Capacity e recomendação canônica;
6. preferências de apresentação separadas das preferências do Capacity Monitor;
7. feature flag experimental;
8. acessibilidade e testes.

Agents, Missions, Cost e alertas entram na projeção desde que possam ser
consumidos sem criar lógica paralela; UX operacional completa continua sendo
expandida nos milestones seguintes.

## Recomendação de recurso

O HUD não solicita uma recomendação nova durante refresh. Quando o Capacity
Monitor entra em `WARNING`, `CRITICAL` ou `EXHAUSTED`, ele reutiliza o
recommender canônico do ResourceScheduler uma única vez na transição e persiste
a recomendação no mesmo evento durável de Capacity.

A projeção do HUD somente apresenta essa recomendação se o `fingerprint` do
evento ainda corresponder a um estado degradado atual e não estiver
acknowledged/snoozed. Depois de recovery, a recomendação antiga desaparece.

Se não houver recomendação canônica vigente, o HUD não inventa uma alternativa e
a superfície Resource permanece ausente/UNKNOWN. Contextos futuros de
Mission/Agent podem projetar recomendações contextuais próprias desde que venham
do mesmo ResourceScheduler, sem recalcular scoring no HUD.

## Preferências

Preferências do HUD são de apresentação e não alteram
`CapacityMonitorPreferences`.

Comuns:

- enabled;
- mode: `AUTO | CAPACITY | RESOURCE | AGENT | MISSION | ALERT | MINIMAL`;
- providers visíveis;
- capacity/agents/missions/cost/recommendation;
- densidade;
- theme;
- privacy mode.

Desktop adiciona placement, display, opacity, always-on-top, snap e auto-hide.
Web adiciona dock/floating e posição dentro da viewport.

## Prioridade AUTO

A prioridade é determinística e não usa LLM:

```text
CRITICAL alert
> human intervention
> Mission blocked
> Agent running
> Resource recommendation
> Capacity summary
```

A implementação deve reutilizar enums/estados canônicos sempre que existirem.

## Desktop process model

No milestone M1 histórico, o projeto usava Wails v2.15.0, que disponibilizava uma janela por aplicação. O HUD foi isolado em um processo de apresentação independente. A release 1.0 atual já usa Wails v3, mantendo o shell nativo independente.

A estratégia do M1 é um shell Wails leve `nexus-hud`:

- contém apenas apresentação e integração nativa;
- conecta-se ao mesmo loopback Core/API quando disponível;
- não possui Capacity Monitor, scheduler, cost engine ou MissionRunner próprios;
- mantém isolamento de falha da superfície;
- usa o lease já existente do Capacity Monitor caso seja necessário iniciar um
  Core local na ausência de outro processo.

A migração para Wails v3 foi incorporada sem duplicar o contrato `HUD Projection`. Uma eventual consolidação de janelas é uma decisão de produto posterior.

## Plataformas

A matriz oficial permanece em
[`docs/operations/platform-support.md`](../operations/platform-support.md).

Estado após o merge do M1:

| Superfície | Estado |
| --- | --- |
| Web | **PASS** no M1 |
| Linux Desktop HUD | **PASS** no M1 nativo/X11 |
| Windows Desktop HUD | gate nativo obrigatório na frente cross-platform |
| macOS Desktop HUD | gate nativo obrigatório na frente cross-platform |

A implementação cross-platform integra o source candidato; execução certificada depende dos gates por plataforma e SHA.
Cross-compilation não altera esses estados e nenhum recurso novo de produto deve
passar à frente da paridade nativa.

## Privacidade

Privacy Mode deve ocultar identificadores sensíveis de apresentação (conta,
workspace/projeto, session identifiers e custo quando configurado) sem alterar
o estado operacional básico de provider/capacidade.

Segredos, tokens OAuth/API e conteúdo privado de conversas nunca pertencem à
projeção.

## Acessibilidade

O HUD deve suportar teclado, foco visível, nomes acessíveis completos,
percentuais textuais, reduced motion, high contrast e zoom/scaling. Informação
não pode depender apenas de cor ou ícone.

## Performance

A superfície não cria polling de provider. Atualizações são derivadas do Core,
eventos existentes e revalidação centralizada/bounded. O objetivo do Desktop HUD
em idle é CPU praticamente zero; métricas devem ser registradas no relatório de
entrega antes de qualquer claim estável.

## PWA

**Decisão M1: DEFERRED.**

O Web HUD não depende de PWA e já é uma superfície oficial do Nexus Web OS. O
frontend atual não possui um service worker/manifest canônico que possa ser
estendido sem introduzir uma nova política de cache, atualização e recuperação.

Adicionar PWA no primeiro milestone aumentaria o blast radius do fechamento sem
resolver uma dependência funcional do HUD. A evolução para installable web app,
standalone mode, browser notifications e badge continua prevista para o
milestone de UX avançada, desde que:

- reutilize o pipeline de notifications existente;
- não crie cache paralelo de estado operacional;
- invalide dados de Capacity/Agents/Missions corretamente;
- não apresente estado stale como atual;
- possua estratégia explícita de update/rollback do service worker.

## Limitações conhecidas do M1

- Windows e macOS continuam `UNVERIFIED` até execução nativa same-SHA.
- No M1 histórico (Wails v2), work area, ID estável de display e scale factor eram limitados. A release atual em Wails v3 expõe esses dados; suporte completo de multi-monitor/DPI ainda requer smoke nativo no SHA vigente.
- always-on-top/posicionamento absoluto no Wayland depende do compositor e deve
  ser capability-detected.
- tray/menu bar só vira claim quando houver implementação e teste nativos.
- PWA é melhoria não bloqueante até existir decisão específica baseada no Web
  atual.


## Minimal taskbar capsule (October 2026)

The HUD's compact surface is an always-on-top **capsule above the system taskbar**, not an unsupported injection into Windows Taskbar. Its default size is 220 × 60 DIPs on Desktop; the interactive capsule itself is roughly 190 × 43 pixels. It uses the official Nexus icon, one highlighted provider and its limiting-window remaining percentage.

The capsule is intentionally a **projection**, not a control plane:
- When a running Agent or Mission identifies a provider/profile, this is the account in execution. Otherwise the highlighted measurable account is labeled *Recurso em destaque*, **never** *Conta ativa*.
- Unknown or offline quota uses an em dash, a neutral ring and an explanation; neither means 0% or unlimited.
- Displayed warning severity originates in the Capacity Monitor. Provider alternatives are informational and sorted for readability, **not** treated as the ResourceScheduler's recommendation.
- If there is an active canonical ResourceScheduler recommendation it may be shown with a link to the existing Usage surface. The HUD never performs a silent account switch.
- Subscription capacity, API costs and per-request tokens are different resource channels; the capsule does not derive one from another.

Behaviors:
1. **Idle**: branded capsule, one highlighted provider and remaining capacity.
2. **Hover (Web)**: short non-interactive tooltip with window and renewal evidence.
3. **Click**: popover with highlighted resource, up to three alternatives, interventions and optional canonical recommendation.
4. **Warning/critical**: colored ring, capacity events and acknowledge/snooze via existing endpoints.
5. **Unknown/offline**: neutral ring, unknown remaining quota and last observed connection when available.

Desktop Wails v3 has a transparent independent native window and native tray recovery. The window starts anchored above the taskbar, and resizing to 334 × 496 DIPs keeps its bottom-right edge anchored. **Rich hover content on Native is not yet feature-equivalent with Web**: native currently exposes a standard tooltip, since a separate temporary hover window needs capability-specific implementation and testing. This limitation must not be described as full parity.

Native validation requires exact-SHA Windows/Linux/macOS smoke, taskbar positioning, DPI/multi-monitor, window restoration, tray recovery, and click-through tests for transparent empty regions. Browser E2E should capture compact, hover and expanded states and run accessibility checks. No production release claim follows from merely committing the UI.

## Correções de estabilidade do HUD compacto (18/10/2026)

- Restauração força a janela nativa de volta a 220×60 DIPs, preservando a âncora inferior direita do monitor. Isso evita a reabertura invisível de uma janela expandida após reiniciar.
- Expansão/recolhimento consulta a dimensão **real** da janela nativa, não apenas a última dimensão persistida.
- O provider só pode ser denominado *Conta em execução* quando provider **e** profile correspondem à identidade de Agent em execução; nomes de provider isolados não identificam contas múltiplas.
- Missions não terminais voltam a aparecer na visão expandida. Intervenções humanas têm prioridade e histórico terminado não substitui execução corrente.
- Percentuais `ESTIMATED` são marcados com `~` e indicador tracejado; `CACHED` é identificado como cache; origem não confiável/sem dados usa `—`.
- Testes unitários cobrem identidade incompleta, evidência de quota, Missions ativas/terminadas e restauração de geometria antiga/expandida.

A validação Windows anteriormente executada permanece como evidência histórica para os SHAs testados. **Estes novos commits não herdam homologação Windows exata**: os gates, o smoke real do HUD, a restauração após reinicialização e as evidências visuais precisam ser executados novamente no HEAD final. Não publicar tag ou release com base apenas no registro anterior.


> **Cost evidence:** `PROVIDER_REPORTED` means a value present in the
> provider's usage telemetry, **not** an invoice or proof of a charge.
> The HUD labels this evidence `reported`. `API_EQUIVALENT_ESTIMATE`
> and `INFERRED` remain estimates; no subscription token valuation is
> merged into confirmed billing. An offline native HUD is never marked
> ready solely because a previous snapshot remains in memory.

## Incidente: cápsula nativa sem popup (2026-10-20)

**Causa reproduzível por inspeção do contrato:** o Core respondia `enabled:false`
porque `flags.HUD` era desligado por padrão, embora `nexus-hud` fosse instalado
sem `NEXUS_ENABLE_HUD=1`. A apresentação nativa interpretava `disabled/offline`
como `waiting` e o clique somente executava refresh; `SetHUDExpanded` nunca
era chamado. O E2E Web ativava a flag artificialmente, mascarando o problema.

**Política corrigida:**
- A projeção de leitura (autenticada, sem probes de provider) agora está habilitada
  por padrão para permitir o HUD nativo pronto para uso.
- `NEXUS_DISABLE_HUD=1` mantém o kill switch de backend prioritário, sem contorno
  pela UI; no estado desativado, a cápsula nativa deve abrir um diagnóstico.
- O widget Web continua **opt-in** por preferência local; API ativa não significa
  widget Web obrigatório.
- O clique chama `SetHUDExpanded` e só considera a expansão concluída depois do
  sucesso da chamada Wails. Falhas registram erro e não fingem popup aberto.
- Se a projeção estiver desativada/offline, a janela pode ser expandida e mostrar
  detalhes de indisponibilidade, atualizar e abrir o Nexus, sem falsificar quota.

**Gates obrigatórios antes do GO:** executar o **mesmo SHA** do Core e HUD nativo
atualizados; a mera atualização do frontend enquanto um Core antigo estiver rodando
não corrige o retorno `enabled:false`. No Windows e Linux, verificar em sessão
gráfica interativa, com `NEXUS_ENABLE_HUD` e `NEXUS_DISABLE_HUD` não definidos:

1. A cápsula abre ao clicar, com cerca de **334 × 496 DIPs** para o popover.
2. Segundo clique ou botão de fechar recolhe para **220 × 60 DIPs**.
3. Com duas contas no mesmo provider, a conta em execução só é exibida se
   `provider+profile` estiverem verificados.
4. Com perda de conexão ao Core, o clique abre um diagnóstico (não ignora o evento).
5. Com `NEXUS_DISABLE_HUD=1` no Core, o diagnóstico informa a indisponibilidade;
   o kill switch não é ignorado.
6. Validar posição junto à barra de tarefas, DPI, dois monitores e reinício.
7. Arquivar screenshot da cápsula e popup, gravação do clique e evidência de
   `aria-expanded=true`, API `enabled` e geometria nativa, sempre no mesmo SHA.

**Não há paridade pixel-perfect certificada com o mockup.** O hover no HUD nativo
continua simples, e o E2E Web não substitui teste do clique real Wails/GTK/WebView2.
A correção de código não equivale a homologação nativa até os gates acima passarem.
