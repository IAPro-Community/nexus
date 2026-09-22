# Nexus Core

O Nexus Core é a autoridade operacional compartilhada por CLI, Web e Desktop.
As superfícies renderizam e transportam ações; estado durável, seleção de
recursos, runtime, projetos, workspaces, eventos e evidência pertencem ao Core.

Veja [consumer-matrix.md](consumer-matrix.md) para a matriz de ownership por
superfície.

## Ownership atual

| Domínio | Autoridade | Persistência / side effects |
| --- | --- | --- |
| Projects, Agents, plans e Missions | `internal/nexus` + `internal/nexus/store` | estado local/SQLite |
| Runtime/process sessions | `internal/control/registry`, drivers e runtime | processos do SO + registry |
| PTY/terminal transport | `internal/control/terminal`, Web transport | PTY/ConPTY + WebSocket |
| Providers | provider registry + control drivers | config/perfis locais do provider |
| Usage/quota | quota engine + adapters | cache com qualidade de evidência |
| Events | event bus/store | timeline estruturada |
| Auth Web | `internal/control/web` | sessão local + CSRF |
| Maestro | boundary opcional | catálogo/status externo quando habilitado |

## Direção de dependência

```text
CLI / Web / Desktop
        ↓
transport / application boundaries
        ↓
     Nexus Core
        ↓
interfaces de domínio
        ↑
infraestrutura: OS, provider CLI, filesystem, network
```

Transport handlers não devem virar uma segunda camada de regra de produto.

## Fronteiras de aplicação

O Core possui boundaries explícitos para áreas como:

- Project;
- Agent;
- Resource;
- Plan;
- Composer;
- Mission planning;
- MissionRun;
- Runtime;
- Intelligence.

Esses serviços são finos: delegam regras de lifecycle às autoridades já
existentes em vez de duplicar máquinas de estado.

## Planejamento vs execução

Há duas famílias distintas:

### Planning records

- WorkPlan;
- Mission definition;
- tasks/assignments;
- Flow projection.

### Execution records

- MissionRun;
- PackageRun;
- dispatch;
- WorkReceipt;
- ContextCapsule;
- evidence.

Alterar um registro de planejamento não equivale a mutar silenciosamente uma
Mission já em execução.

## Semântica de Agent

Agent é identidade persistente. Provider é adapter/recurso.

`AgentSpec` guarda especialização independente de provider.
`AgentRevision` congela configuração efetiva.
`RuntimeGeneration` registra a encarnação concreta que executou.

`WorkPackage.role` é contexto temporário da tarefa e compõe com
`AgentSpec.role`; não substitui a identidade persistente.

## Execução compartilhada

Direct, Assisted, Orchestrated e Mission convergem em componentes canônicos:

- compilação de contexto;
- Agent identity;
- provider adapter;
- runtime supervisionado;
- observabilidade;
- security boundaries.

Maestro adiciona metodologia/gates quando habilitado; não substitui esses
componentes.

## Transport HTTP

`internal/control/web` é adapter. Endpoints são agrupados por domínio e devem
chamar boundaries do Core.

A API é interna ao produto; estabilidade pública/SDK não é prometida.

## Referências

- [Arquitetura raiz](../../ARCHITECTURE.md)
- [Runtime da Mission](mission-runtime.md)
- [Guardrails](guardrails.md)
- [Events](events.md)
- [Persistence](persistence.md)
