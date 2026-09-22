# Integração com Maestro

O [Orquestrador Maestro](https://github.com/IAPro-Community/Orquestrador-Maestro)
é uma integração opcional de **metodologia, skills, risco e gates**.

```text
Nexus
  → Projects
  → Agents
  → WorkPlan / Flow
  → MissionRunner
  → ResourceScheduler
  → runtimes / providers / workspaces
  → evidence

Maestro
  → metodologia
  → skills
  → risco
  → gates
  → políticas de review/verificação
```

## Fronteira de ownership

Nexus continua sendo a autoridade de execução. Maestro não substitui:

- AgentMatcher;
- ResourceScheduler;
- WorkPlan;
- MissionRunner;
- memory engine;
- workspace manager;
- provider identity.

## Degradação

Sem Maestro:

- Direct funciona;
- Agents funcionam;
- Composer funciona;
- Flow funciona;
- Mission funciona.

Capacidades especificamente metodológicas devem aparecer como indisponíveis ou
`MAESTRO_DEGRADED`; o Nexus não inventa skills/gates para mascarar a ausência
da integração.

## Segurança

A integração deve trabalhar com contexto limitado e machine-readable. Não deve
receber por padrão credentials, `.env`, auth files ou transcripts privados
inteiros.

Veja [Como funciona](how-it-works.md) e
[Arquitetura](../architecture/overview.md).
