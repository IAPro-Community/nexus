# Missions e Autopilot

Mission é a camada avançada para executar trabalho aprovado em etapas coordenadas:

```text
READY → ALLOCATE → COMPILE_PROMPT → EXECUTE → TEST → REVIEW → VERIFY
```

O runtime deve concluir por evidência, não porque um Agent disse “done”. Use Mission quando o trabalho justificar duração, dependências, múltiplos workers, gates ou recuperação. Não é requisito para uma AI Session direta.

Mission/WorkPackage execution shares the same Agent, context compiler,
provider adapter, runtime lifecycle, and observability path as Direct
execution. A WorkPackage role is temporary task context; it does not mutate
the persistent AgentSpec role. Maestro guidance is an optional input to the
shared compiler, not a replacement for Nexus execution.

Mission planning records (mission, task and assignment) are managed through
the Core application boundary. A `MissionRun` is a separate execution
instance owned by the durable runner; changing planning data must not be
confused with mutating a live run.
