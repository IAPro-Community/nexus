# Funcionalidades

## TL;DR

Use **Direct** para uma tarefa simples. Use Composer/Flow quando precisar de
clareza e estrutura. Use Mission quando precisar de execução durável,
dependências, múltiplos Agents, gates ou recovery.

> **Status funcional ≠ status de release.** Uma capacidade pode estar
> implementada no source candidato enquanto a release estável continua em
> certificação.

| Objetivo | Capacidade | Estado no source candidato | Onde começar |
| --- | --- | --- | --- |
| trabalhar imediatamente | Projects + Direct + terminal | Disponível | [Direct](../product/direct.md) |
| manter especialização operacional | Agents persistentes | Disponível | [Agents](../product/agents.md) |
| escolher/observar recursos | Providers, profiles, quota/capacity | Disponível; evidência varia por provider | [Providers](../product/providers-and-usage.md) |
| refinar um pedido | Composer | Disponível | [Composer](../product/composer.md) |
| revisar plano/dependências | WorkPlan + Flow | Disponível | [Flow](../product/flow.md) |
| executar com estado durável | MissionRunner | Disponível no source candidato; certificação de release em andamento | [Missions](../product/missions.md) |
| coordenar especialistas | Mission Teams / dynamic coordination | Disponível no source candidato | [Coordenação](../product/agent-coordination.md) |
| delegar objetivo composto | Conductor | Disponível conforme superfície/policy | [Conductor](../product/conductor.md) |
| aplicar método/skills externos | Maestro | Opcional | [Maestro](../product/maestro.md) |
| operar via script | CLI + JSON | Disponível | [CLI](../cli/README.md) |
| diagnosticar host | Doctor / security / status | Disponível | [Operação](../operations/README.md) |

## Profundidade recomendada

```text
tarefa simples
   ↓
 Direct
   │
   ├─ precisa clarificar? → Composer
   │                         ↓
   │                      WorkPlan / Flow
   │                         ↓
   └─ precisa execução durável/gates? → Mission
                                      ↓
                               COMPLETED_VERIFIED
```

## Recursos avançados

- [Como funciona ponta a ponta](../product/how-it-works.md)
- [Coordenação entre Agents](../product/agent-coordination.md)
- [Worktrees](../architecture/filesystem-identity.md)
- [Agent routing](../architecture/agent-routing.md)
- [Resource routing](../architecture/resource-routing.md)
- [Mission runtime](../architecture/mission-runtime.md)
- [Continuidade](../architecture/session-continuity.md)
- [Guardrails](../architecture/guardrails.md)
- [Matriz por superfície](../reference/feature-matrix.md)
