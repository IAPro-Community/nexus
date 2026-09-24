# ADR — Agent Interop como projeção do modelo canônico de Agents

Status: **ACCEPTED / PARTIALLY IMPLEMENTED IN V1 BETA**  
Data: 2026-09-24  
Base original analisada: `main@79493344ac510ad81d741a867dd92bb86bf4215b`  
Release de integração: `v1.0.0-beta.2` candidate  
Spec associada: `DEV/SPECS/NEXUS_1_1_AGENT_INTEROP.md`

## Contexto

O Nexus v1 já possui um modelo de Agent suficientemente rico para representar
especialização sem acoplar identidade ao provider:

```text
Agent
├── AgentSpec
├── AgentRevision
└── RuntimeGeneration
    └── ProviderSession
```

A v1 também já contém:

- `AgentMode` canônico e tipado;
- `AgentMatcher` como autoridade de **quem** executa;
- `ResourceScheduler` como autoridade de **com qual recurso** executar;
- compilação de `AgentSpec` para contexto de execução;
- `MissionRunner` como única autoridade de execução durável;
- `AutonomyContract`, isolamento e evidência;
- Capacity/Cost Intelligence;
- Web/Desktop/CLI sobre o mesmo Core;
- Mission Teams e pipeline Planning → Implementation → Review → Verification;
- HUD como projeção read-only de estado canônico.

Os coding runtimes suportados, porém, expõem conceitos de agents de maneiras
diferentes. Alguns permitem agentes primários selecionáveis, outros apenas
subagents, alguns permitem permissões por agent e outros dependem de prompt ou
sandbox externo.

O objetivo do Agent Interop é permitir que uma especialização definida no Nexus seja
utilizada de forma coerente através dos runtimes suportados sem criar uma
segunda identidade de Agent, uma segunda engine de orquestração ou uma tabela
paralela de providers.

## Problema

Há hoje dois comportamentos diferentes que não podem ser confundidos.

### Especialização persistente

`AgentMode` é um preset que origina/configura um `AgentSpec` persistente.
Alterá-lo muda a identidade comportamental efetiva do Agent e pode exigir nova
`AgentRevision`/sessão.

### Papel operacional temporário

Durante uma sessão o usuário pode querer alternar entre comportamentos como:

```text
BUILD → INVESTIGATE → REVIEW → VERIFY
```

Essa troca não deve reescrever a identidade persistente do Agent, gerar
revisões artificiais ou destruir continuidade.

Além disso, quando o host possui agents nativos, o Nexus deve poder projetar
seus papéis para o formato do host, mantendo provenance e sem sobrescrever
configuração do usuário.

## Decisão

### 1. Não criar um novo Agent Runtime

O Nexus já é runtime/control plane. **Agent Interop** é adicionado como
capacidade do Core atual; o primeiro slice foi integrado à linha v1 beta.

Não serão criados:

- segundo scheduler;
- segundo Mission runner;
- segundo Agent store;
- segundo formato de memória;
- orquestrador peer-to-peer;
- runtime de agents independente do `RuntimeGeneration`.

### 2. Introduzir ExecutionRole

`ExecutionRole` representa o papel operacional temporário de um Agent durante
um turno ou uma RuntimeGeneration.

Exemplos iniciais:

```text
build        -> developer
investigate  -> investigator
review       -> reviewer
verify       -> qa-engineer
architect    -> architect
security     -> security-reviewer
```

O papel referencia a taxonomia existente. Ele não copia integralmente
`AgentSpec`.

O contexto efetivo passa a ser conceitualmente:

```text
Persistent AgentSpec
       +
ExecutionRole
       +
WorkPackage role/context
       +
Runtime constraints
```

Exemplo:

```text
Senior Developer acting as Reviewer for this interactive session
```

### 3. ExecutionRole não altera Agent identity

Trocar `build → review`:

- não cria novo `Agent`;
- não altera `Agent.Role`;
- não cria `AgentRevision` só pela troca;
- não altera provider por si só;
- não transforma preferência em PIN;
- não redefine WorkPlan;
- não cria Mission.

A seleção pode ter escopo `TURN` ou `RUNTIME`.

Uma seleção `RUNTIME` é persistida como binding operacional da
`RuntimeGeneration` para sobreviver a refresh/reconnect. Uma seleção `TURN`
é registrada no receipt/evento do turno, mas não substitui o binding da
RuntimeGeneration.

### 4. Introduzir Agent Projection

Quando um provider/host possui um formato nativo de agent, o Nexus poderá
materializar um `ExecutionRole`/`AgentMode` nesse formato.

Fluxo:

```text
AgentMode / AgentSpec
        |
        +-------------------+
        |                   |
Prompt compilation     Agent projection
        |                   |
        |             host-native artifacts
        |                   |
        +---------+---------+
                  |
               Provider
```

Projection é um adapter. Não se torna autoridade de identidade, scheduling ou
Mission.

### 5. Projection deve ser capability-driven

O Nexus não assumirá comportamento por nome do provider.

Será adicionado um contrato de capabilities específico:

```go
type AgentProfileCapabilities struct {
    Definitions      CapabilityEvidence
    PrimaryAgent     CapabilityEvidence
    Subagents        CapabilityEvidence
    InSessionSwitch  CapabilityEvidence
    DynamicReload    CapabilityEvidence
    ToolPermissions  CapabilityEvidence
    ModelOverride    CapabilityEvidence
    ProjectScope     CapabilityEvidence
    UserScope        CapabilityEvidence
    RemoteProtocol   CapabilityEvidence
}
```

Estados continuam evidence-bound:

```text
SUPPORTED
PARTIAL
UNSUPPORTED
UNKNOWN
NOT_TESTED
```

`SUPPORTED` exige evidência compatível com a política da capability.
Configuração documentada mas não exercitada em teste deve permanecer
`NOT_TESTED` ou `PARTIAL`, conforme o caso.

### 6. Separar discovery, materialization e activation

Um host possuir arquivo de agent não prova que o Nexus consegue ativá-lo.

A V1 trata separadamente:

```text
DISCOVERY       host encontra a definição
MATERIALIZATION Nexus consegue gerar/aplicar a definição
ACTIVATION      Nexus consegue tornar o papel ativo no momento necessário
```

A estratégia de ativação é explícita:

```go
type ActivationStrategy string

const (
    ActivationNativeHotSwitch ActivationStrategy = "NATIVE_HOT_SWITCH"
    ActivationNativeOnLaunch  ActivationStrategy = "NATIVE_ON_LAUNCH"
    ActivationNativeSubagent  ActivationStrategy = "NATIVE_SUBAGENT"
    ActivationNexusOverlay    ActivationStrategy = "NEXUS_OVERLAY"
    ActivationUnsupported     ActivationStrategy = "UNSUPPORTED"
)
```

Quando o host não fornece seleção nativa verificável, o Nexus mantém a
experiência de papel através do compilador de contexto existente
(`NEXUS_OVERLAY`).

### Estado implementado no candidato beta.2

Para OpenCode, a implementação atual separa explicitamente quatro níveis:

```text
materialization   -> arquivo/manifest gerenciado pelo Nexus
filesystem verify -> hash + ownership
discovery         -> OpenCode enumera nexus-<role> sem LLM
activation        -> opt-in externo executa opencode run --agent nexus-<role>
```

O activation probe pode consumir quota/custo e por isso exige confirmação
explícita; nunca é disparado por Plan/Apply/Verify. Quando o marker de ativação
é observado, o binding registra `NATIVE_ON_LAUNCH` vinculado ao receipt
corrente. Isso **não** altera `Enforcement`: READ_ONLY/VERIFY_ONLY continuam
`BEST_EFFORT`/`UNVERIFIED` até existir prova E2E separada de bloqueio de
escrita. Receipts antigos não podem remover nem promover uma projeção mais nova.

### 7. Enforcement é distinto de instrução

Um prompt com “não altere arquivos” não é sandbox.

Toda projeção de role deve expor a qualidade da restrição:

```text
ENFORCED
BEST_EFFORT
UNVERIFIED
UNSUPPORTED
```

Para `reviewer`, `investigator` e outros papéis read-only:

- `ENFORCED`: provider/sandbox comprovadamente bloqueia escrita;
- `BEST_EFFORT`: restrição existe apenas no contexto/instrução;
- `UNVERIFIED`: mecanismo existe, mas não há evidência suficiente;
- `UNSUPPORTED`: não existe mecanismo conhecido.

O Nexus não promoverá `BEST_EFFORT` ou `UNVERIFIED` a `ENFORCED`.

### 8. Arquivos gerados são gerenciados com ownership explícito

O default é materializar dentro do profile home isolado do Nexus, não no HOME
real do usuário nem no repositório.

Escopos:

```text
NEXUS_PROFILE  default
PROJECT        opt-in explícito
USER           opt-in explícito
```

Cada artifact terá receipt com:

- provider;
- role/mode;
- renderer version;
- target path;
- source hash;
- content hash;
- scope;
- timestamp;
- ownership = nexus.

Regras:

```text
arquivo ausente                       -> CREATE
arquivo Nexus com hash esperado       -> UPDATE
arquivo Nexus alterado externamente   -> CONFLICT
arquivo não gerenciado pelo Nexus     -> NEVER OVERWRITE
```

### 9. MissionRunner continua soberano

Native subagents de Cursor/OpenCode/Claude/Gemini não podem substituir a
orquestração do Nexus.

Para Missions:

```text
WorkPlan
  -> MissionRunner
  -> AgentMatcher
  -> ResourceScheduler
  -> Agent/RuntimeGeneration
  -> WorkReceipt/Evidence
```

continua sendo o caminho canônico.

Host-native subagents podem ser usados como capacidade auxiliar somente quando
o contrato de execução preservar lineage, limits e evidence. Na primeira
entrega da V1, native subagents são prioritariamente recurso de
Direct/Interactive e não uma nova state machine de Mission.

### 10. UX pertence ao Nexus

O Nexus oferecerá seletor próprio de role no Web/Desktop/terminal surface e na
Command Palette.

Não será reservado `Shift+Tab` globalmente por padrão porque a tecla pode
pertencer ao TUI hospedado (OpenCode, terminal, editor etc.).

UX inicial:

```text
Agent: Senior Developer
Role: REVIEW
Provider: Codex
Read-only: ENFORCED
```

Ações:

- seletor no header da sessão;
- `Ctrl/Cmd+K → Switch execution role`;
- shortcut dedicado configurável;
- eventual forwarding para seleção nativa quando a capability
  `InSessionSwitch` estiver comprovada.

## Contrato de projection

A projeção será implementada por interface opcional separada de
`ControlDriver`:

```go
type AgentProjectionAdapter interface {
    ProviderID() string
    Capabilities(context.Context, model.Profile) AgentProfileCapabilities
    Plan(context.Context, ProjectionRequest) (*ProjectionPlan, error)
    Render(context.Context, ProjectionRequest) ([]GeneratedArtifact, error)
    Apply(context.Context, ProjectionPlan) (*ProjectionReceipt, error)
    Verify(context.Context, ProjectionReceipt) (*ProjectionEvidence, error)
    Remove(context.Context, ProjectionReceipt) error
}
```

Separar a interface evita inflar `ControlDriver` com responsabilidades de
filesystem/configuração que não são parte do processo supervisionado.

## Matriz inicial de estratégia

Esta tabela é hipótese de implementação, não certificação. A V1 só promove
uma célula após probe/version/evidência no ambiente suportado.

| Provider | Definições nativas | Primary | Subagent | Estratégia inicial |
| --- | --- | --- | --- | --- |
| OpenCode | documentado | documentado | documentado | native + overlay |
| Claude Code | documentado para subagents | a validar por versão | documentado | native subagent + overlay |
| Cursor | documentado | não assumido | documentado | native subagent + overlay |
| Gemini CLI | documentado | não assumido | documentado | native subagent + overlay |
| Codex | não será presumido por diretório | não assumido | não assumido | overlay; evoluir via app-server |
| AGY | contrato próprio | não assumido | não assumido | overlay inicialmente |

Referências externas consultadas para desenho, sem substituir runtime probes:

- OpenCode agents: https://opencode.ai/v2/docs/agents
- Cursor subagents: https://prod.cursor.com/docs/subagents
- Gemini CLI subagents: https://geminicli.com/docs/core/subagents/
- Claude Code `.claude`/subagents: https://code.claude.com/docs/
- Codex como plataforma/app-server:
  https://developers.openai.com/pt-BR/blog/codex-as-a-platform

## Relação com HUD

HUD permanece projection-only.

Após Agent Interop, poderá projetar:

- execution role;
- provider/profile/model;
- activation strategy;
- enforcement quality;
- projection health;
- cost/capacity existentes.

HUD não renderiza agent files, não escolhe provider, não executa LLM e não
mantém outra cópia da role.

## Relação com ResourceScheduler

`AgentMode.DefaultProvider` não deve se tornar autoridade.

Na V1 será deprecado semanticamente em favor de preferência/requisitos:

```text
ExecutionRole
  -> capability requirements
  -> ResourceScheduler
  -> provider/profile/model
```

Uma preferência pode ser `PREFER`; somente policy explícita do usuário pode
produzir `PIN`.

### Custom execution targets

Agent Projection é independente de como o provider é iniciado. O provider
continua sendo `opencode`, `codex`, `claude` etc.; Docker/VPN/WSL/SSH são
targets de execução, não novos providers.

O launcher mantém `command_template` como contrato argv-only, com placeholders
tipados e sem shell. `cwd` (host) e `target_cwd` (target) são distintos para
suportar mounts com caminhos diferentes, especialmente Windows → Docker. Para projeção nativa em target customizado, uma
`target_profile_home` compartilhada precisa ser explicitamente configurada e
validada. A versão do provider deve ser obtida no target por probe bounded quando
possível.

Ausência de filesystem compartilhado não impede a execução customizada: ela
degrada apenas a Agent Projection nativa, preservando Nexus overlay e reportando
a qualidade real do enforcement.

## Consequências

### Positivas

- mesma intenção de Agent em múltiplos coding runtimes;
- UX consistente mesmo quando o host não possui selector nativo;
- aproveita subagents/primary agents onde existem;
- preserva Agent identity, attribution, cost e continuity;
- reduz lock-in de um único CLI;
- abre caminho para exportação controlada e ecossistema de profiles.

### Custos

- adapters precisam ser version-aware;
- formatos externos podem mudar;
- testes E2E por provider tornam-se obrigatórios;
- hot-switch nativo não será igualmente disponível em todos os hosts;
- generated-file ownership precisa de reconciliação e rollback.

## Alternativas rejeitadas

### Copiar Cavecrew diretamente

Rejeitado porque a V1 já possui taxonomia e `AgentSpec` mais ricos e porque
manter arquivos independentes por provider criaria drift.

### Transformar todo AgentMode em primary agent

Rejeitado porque nem todo host possui primary custom agent e porque dezenas de
modos tornariam o selector ruidoso.

### Trocar AgentSpec a cada mudança de papel

Rejeitado porque polui `AgentRevision`, lineage e continuity.

### Delegar Mission para o orquestrador interno do host

Rejeitado porque cria duas autoridades de execução e perde evidence/fencing.

### Escrever diretamente no HOME real do usuário

Rejeitado como default por risco de sobrescrita, segurança e falta de rollback.

## Rollout

1. capability truth e probes;
2. `ExecutionRole` + bindings;
3. projection core e ownership receipts;
4. OpenCode;
5. Claude Code;
6. Cursor;
7. Gemini CLI;
8. Codex overlay + investigação app-server;
9. AGY;
10. HUD/UX polish;
11. cross-platform certification.

## Critério de conclusão para a V1 estável

O Agent Interop só pode ser considerado concluído para `v1.0.0` quando:

- o baseline beta candidato estiver preservado por SHA/tag;
- os contratos Go e migration tiverem testes;
- pelo menos dois providers tiverem projection E2E real;
- read-only não for anunciado como enforced sem prova;
- role switch não alterar Agent identity;
- MissionRunner continuar autoridade única;
- artifacts do usuário não forem sobrescritos;
- Web/Desktop/CLI consumirem o mesmo Core.
