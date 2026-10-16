# Nexus HUD

> M1 Web/Linux foi integrado à `main`. A evolução nativa continua na branch
> `feat/hud-cross-platform-completion` com um gate explícito: nenhuma nova
> feature funcional do HUD é considerada DONE antes de atingir paridade
> certificada entre Linux, Windows e macOS.
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

O projeto usa Wails v2.15.0. Wails v2 oferece uma janela por aplicação; portanto
um HUD realmente independente não deve ser simulado dentro da janela principal.

A estratégia do M1 é um shell Wails leve `nexus-hud`:

- contém apenas apresentação e integração nativa;
- conecta-se ao mesmo loopback Core/API quando disponível;
- não possui Capacity Monitor, scheduler, cost engine ou MissionRunner próprios;
- mantém isolamento de falha da superfície;
- usa o lease já existente do Capacity Monitor caso seja necessário iniciar um
  Core local na ausência de outro processo.

Uma migração futura para Wails v3 pode consolidar janelas no mesmo processo sem
alterar o contrato `HUD Projection`.

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

A branch de cross-platform roda Windows e macOS no mesmo ciclo da implementação.
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
- Wails v2 informa dimensões/tela atual, mas não ID estável, work area ou scale
  factor; o M1 faz clamp seguro para recuperar janelas off-screen, sem promover
  isso a suporte completo de multi-monitor/DPI.
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
