# Arquitetura

A arquitetura canônica da v1 está resumida em
[`ARCHITECTURE.md`](../../ARCHITECTURE.md). Esta página é o índice técnico
para aprofundamento.

![Componentes do Nexus v1](../assets/diagrams/nexus-v1-components.svg)

## Visão de responsabilidade

```text
Web / Desktop / CLI
        ↓
     Nexus Core
        ↓
Projects · Agents · Providers · Runtime
        ↓
Composer → WorkPlan → Flow → GO
        ↓
   MissionRunner
        ↓
Scheduler · Worktrees · Agents
        ↓
ContextCapsule · WorkReceipt · Evidence
        ↓
Global DoD → COMPLETED_VERIFIED
```

O Maestro é integração opcional de método, skills, risco e gates. Execução,
estado, scheduling e evidence permanecem no Nexus.

## Leia por assunto

| Assunto | Documento |
| --- | --- |
| arquitetura completa | [ARCHITECTURE.md](../../ARCHITECTURE.md) |
| Core / ownership | [core.md](core.md) |
| lifecycle da Mission | [mission-runtime.md](mission-runtime.md) |
| guardrails / fail closed | [guardrails.md](guardrails.md) |
| Agents e routing | [agent-routing.md](agent-routing.md) |
| resource scheduler | [resource-routing.md](resource-routing.md) |
| Mission Teams | [mission-teams.md](mission-teams.md) |
| runtime supervisionado | [runtime-lifecycle.md](runtime-lifecycle.md) |
| provider contract | [provider-contract.md](provider-contract.md) |
| persistência | [persistence.md](persistence.md) |
| eventos | [events.md](events.md) |
| filesystem/worktrees | [filesystem-identity.md](filesystem-identity.md) |
| continuidade | [session-continuity.md](session-continuity.md) |
| atualização | [updates.md](updates.md) |

## Separação importante

### Planejamento

`PromptArtifact → WorkPlan → Flow`

### Execução

`MissionRun → PackageRun → WorkReceipt`

Flow não é uma segunda máquina de execução. MissionRunner não é um segundo
planner.

## Plataforma

Um diagrama de arquitetura não certifica um sistema operacional. Consulte
[suporte de plataformas](../operations/platform-support.md) e o
[status de release](../release-status.md).
