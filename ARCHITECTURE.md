# Arquitetura do IAPro Nexus v1

Esta página descreve a arquitetura canônica atual. Documentos históricos em
`DEV/` preservam decisões e campanhas antigas, mas não substituem este mapa.

![Arquitetura lógica do Nexus v1](docs/assets/diagrams/nexus-v1-components.svg)

## 1. Princípios

1. **Nexus é o runtime e control plane.**
2. **Agent é identidade persistente; provider é recurso de execução.**
3. **WorkPlan é a fonte de verdade do plano; Flow é sua projeção visual.**
4. **MissionRunner é a autoridade de execução durável.**
5. **ResourceScheduler escolhe com qual recurso executar; não escolhe a identidade do Agent.**
6. **Evidência, não texto do modelo, determina sucesso.**
7. **Maestro é integração metodológica opcional, não um segundo orquestrador.**
8. **UNKNOWN nunca é promovido implicitamente a sucesso.**
9. **Side effects sem prova de segurança falham fechado.**
10. **CLI, Web e Desktop consomem o mesmo Core.**

## 2. Visão de alto nível

```text
                         ┌───────────────┐
                         │   Usuário     │
                         └──────┬────────┘
                                │
                 ┌──────────────┼──────────────┐
                 │              │              │
               CLI             Web          Desktop
                 └──────────────┼──────────────┘
                                ↓
                         ┌───────────────┐
                         │  Nexus Core   │
                         └──────┬────────┘
                                │
      ┌─────────────────────────┼─────────────────────────┐
      │                         │                         │
      ↓                         ↓                         ↓
 Projects / Agents       Composer / WorkPlan       Runtime / Providers
      │                         │                         │
      │                         ↓                         │
      │                       Flow                        │
      │                         │                         │
      └─────────────────────────┼─────────────────────────┘
                                ↓
                               GO
                                ↓
                         ┌───────────────┐
                         │ MissionRunner │
                         └──────┬────────┘
                                │
              ┌─────────────────┼──────────────────┐
              ↓                 ↓                  ↓
       ResourceScheduler    Worktrees          Gates/Review
              │                 │                  │
              └──────────────┬──┴──────────────────┘
                             ↓
                           Agents
                             ↓
                 WorkReceipt / Artifacts
                             ↓
                    Global Definition of Done
                             ↓
                    COMPLETED_VERIFIED
```

## 3. Fronteiras de responsabilidade

### Nexus Core

O Core mantém:

- projects e workspaces;
- Agents e revisões;
- runtime generations;
- provider registry/drivers;
- sessions e terminal transport;
- plans e revisions;
- Composer;
- Flow;
- Missions e runs;
- resource scheduling;
- intelligence/memory;
- events;
- evidence;
- recovery;
- security boundaries.

As superfícies não devem duplicar essas regras.

### Web / Desktop / CLI

São adaptadores e experiências de operação.

- **CLI**: scripting, diagnóstico, provider passthrough, execução headless.
- **Web**: operação visual, Projects, Agents, Composer, Flow e Mission.
- **Desktop**: shell nativo da mesma aplicação quando a plataforma está certificada.

### Maestro

Maestro fornece, quando habilitado:

- metodologia;
- skills;
- risco;
- gates;
- padrões de review/verificação.

Nexus continua responsável por execução, scheduler, Agents, workspaces e
persistência.

## 4. Modelo de Agent

```text
Agent (identidade persistente)
├── AgentSpec
├── AgentRevision (snapshot imutável)
└── RuntimeGeneration
    └── ProviderSession
```

Provider/model/profile podem mudar sem mudar a identidade do Agent.

A seleção é deliberadamente separada:

```text
TaskRequirements
      ↓
 AgentMatcher       → QUEM
      ↓
 ResourceScheduler  → COM QUAL RECURSO
```

Essa separação evita que preferência de provider se torne identidade ou
autoridade de execução.

## 5. Planejamento

![Fluxo ponta a ponta](docs/assets/diagrams/nexus-v1-flow.svg)

### Composer

Converte intenção/contexto em `PromptArtifact` estruturado, com critérios,
restrições, riscos e gaps explícitos.

### WorkPlan

É o contrato semântico/versionado da execução. Contém phases/packages,
dependências, requisitos e critérios.

### Flow

É a projeção visual do WorkPlan. Mover/reorganizar trabalho deve alterar o
modelo canônico, não uma cópia paralela.

### GO

Congela e autoriza uma revisão específica para execução.

## 6. MissionRunner

Leia [Runtime da Mission](docs/architecture/mission-runtime.md).

`MissionRunner` é uma máquina de estados persistente:

```text
READY → ALLOCATE → COMPILE → EXECUTE → TESTING → REVIEW → VERIFIED
```

A execução possui:

- lease/fencing;
- dispatch identity;
- retry/remediation;
- no-progress detection;
- dependency graph;
- parallel groups;
- recovery;
- human intervention;
- global verification.

## 7. ResourceScheduler

O scheduler seleciona provider/profile/model com base em:

- capability requirements;
- policy;
- preferência ou pin explícito;
- disponibilidade;
- quota/capacity;
- isolamento suportado;
- saúde/evidência.

`PREFER` é fallible. `PIN` é fail-closed.

O scheduler não substitui AgentMatcher e não cria um Agent novo por decisão de
provider.

## 8. Workspaces e isolamento

- Direct pode operar no workspace do projeto.
- Escrita autônoma prefere worktree dedicado.
- Escrita paralela exige isolamento.
- Review/Investigation é read-only quando o provider comprova esse modo.
- Nenhuma Mission deve sobrescrever silenciosamente o trabalho de outra.

## 9. Contexto e evidência

O Nexus evita usar transcript bruto como memória canônica.

### ContextCapsule

Handoff limitado com:

- objetivo;
- dependências;
- receipts;
- relevant paths;
- critérios;
- constraints;
- handoff metadata.

### WorkReceipt

Registra resultado de uma package:

- status;
- summary;
- changed files;
- commands;
- testes;
- artifacts;
- verification;
- Agent;
- revisões Git quando disponíveis.

### Evidence Bundle

Agrupa evidência necessária para auditoria/release de uma Mission.

## 10. Coordenação entre Agents

![Coordenação entre Agents](docs/assets/diagrams/nexus-v1-agent-coordination.svg)

O protocolo v1 é fechado:

- `NEED_INFO`;
- `ARTIFACT_REQUEST`;
- `REVIEW_REQUEST`;
- `DELEGATE`.

Toda solicitação passa pelo MissionRunner. Não existe peer-to-peer livre como
fonte de estado.

Detalhes: [Coordenação entre Agents](docs/product/agent-coordination.md).

## 11. Segurança e side effects

![Guardrails](docs/assets/diagrams/nexus-v1-guardrails.svg)

A autoridade principal é o `AutonomyContract`.

Exemplos de enforcement:

- `AllowedFilePatterns`;
- `DisallowDestructiveGit`;
- `AllowGitPush`;
- `AllowDeploy`;
- `AllowExternalNetwork`;
- `AllowSecretAccess`;
- `AllowPaidServices`;
- retries/iterations/no-progress;
- human operations permitidas.

Leia [Guardrails](docs/architecture/guardrails.md).

## 12. Provider layer

Providers são adaptadores para CLIs/runtimes oficiais. O Nexus:

- detecta capabilities;
- prepara ambiente isolado;
- gerencia profile/config paths;
- executa PTY/ConPTY;
- registra runtime/session;
- observa quota quando existe evidência;
- classifica falhas;
- permite fallback conforme policy.

Autenticação permanece pertencendo ao provider oficial. O Nexus não é um proxy
OAuth central.

## 13. Runtime supervisionado

Componentes relevantes:

- `internal/control/host`: processo supervisionado;
- `internal/control/driver`: adapters/capabilities;
- `internal/control/registry`: sessions/runtimes;
- `internal/control/terminal`: PTY/ConPTY;
- `internal/control/events`: eventos estruturados;
- `internal/nexus/runner`: Mission state machine;
- `internal/nexus/store`: persistência de produto.

## 14. Persistência

Estado durável usa stores locais/SQLite conforme o domínio. A persistência deve
suportar:

- restart;
- idempotência;
- lineage;
- recovery;
- evidence;
- leitura pelas superfícies sem reconstruir estado a partir de transcripts.

## 15. API local

A API `/api/v1` é interna às superfícies do produto. Ela usa bootstrap/session
local, CSRF e loopback por padrão. Não é um SDK público estável.

Scripts devem preferir CLI/`--json` quando possível.

## 16. Status de plataforma

Build e execução nativa são gates diferentes. Cross-compilation não prova
runtime real.

Consulte [Suporte de plataformas](docs/operations/platform-support.md) e o
[status de release](DEV/validation/current/RELEASE_STATUS.md).

## 17. Documentos técnicos relacionados

- [Arquitetura detalhada](docs/architecture/overview.md)
- [Nexus Core](docs/architecture/core.md)
- [Mission runtime](docs/architecture/mission-runtime.md)
- [Guardrails](docs/architecture/guardrails.md)
- [Provider contract](docs/architecture/provider-contract.md)
- [Runtime lifecycle](docs/architecture/runtime-lifecycle.md)
- [Mission Teams](docs/architecture/mission-teams.md)
- [Resource routing](docs/architecture/resource-routing.md)
- [Events](docs/architecture/events.md)

## 18. O que esta arquitetura não faz

A v1 não cria:

- proxy universal de LLM;
- segundo scheduler;
- segundo memory engine;
- segundo formato de plano;
- peer-to-peer mesh de Agents;
- permissão implícita de push/deploy;
- falsa certificação de plataforma sem evidência.
