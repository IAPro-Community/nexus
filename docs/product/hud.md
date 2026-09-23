# Nexus HUD

> Status inicial: **experimental** na branch `feat/nexus-hud`.
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

Uma recomendação do ResourceScheduler exige requisitos. O Usage atual possui uma
recomendação genérica para a superfície de recursos. Enquanto não existir um
contexto de tarefa/Mission mais específico, o HUD reutiliza esse contrato como
**recomendação geral** e expõe a explicação do scheduler. Ele não afirma que o
resultado é o melhor recurso para toda tarefa.

Quando uma Mission/Agent fornecer requisitos canônicos, a projeção pode carregar
a recomendação contextual correspondente, sem alterar a regra de scoring.

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

No início desta feature:

| Superfície | Estado |
| --- | --- |
| Web | implementação/certificação desta branch |
| Linux Desktop HUD | alvo inicial nativo |
| Windows Desktop HUD | UNVERIFIED até certificação nativa same-SHA |
| macOS Desktop HUD | UNVERIFIED até certificação nativa same-SHA |

Cross-compilation não altera esses estados.

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

## Limitações conhecidas do M1

- Windows e macOS continuam `UNVERIFIED` até execução nativa.
- always-on-top/posicionamento absoluto no Wayland depende do compositor e deve
  ser capability-detected.
- tray/menu bar só vira claim quando houver implementação e teste nativos.
- PWA é melhoria não bloqueante até existir decisão específica baseada no Web
  atual.
