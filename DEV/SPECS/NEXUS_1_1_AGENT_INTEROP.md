# Nexus Agent Interop — spec histórica “1.1”

Status: **PARTIALLY IMPLEMENTED / INTEGRATED INTO V1 BETA**  
Release target atual: **v1.0.0-beta.2**  
Base original de planejamento: `main@79493344ac510ad81d741a867dd92bb86bf4215b`  
ADR: `docs/architecture/ADR-agent-interop-projection.md`

> O nome do arquivo é preservado para rastreabilidade. A linha 1.1 deixou de ser
> uma linha de release separada em 24/09/2026; o escopo implementado foi
> incorporado à estabilização da v1 enquanto ela ainda está em beta. Itens não
> implementados continuam sujeitos à própria capability/evidence matrix.

## 1. Objetivo

Permitir que os papéis/especializações já definidos no Nexus sejam utilizados
de forma consistente através dos coding runtimes suportados, aproveitando
agents/subagents nativos quando disponíveis e degradando honestamente para o
context compiler existente quando não estiverem disponíveis.

A experiência alvo é:

```text
Persistent Agent
      +
Execution Role
      +
Provider capability
      ↓
best supported activation
      ↓
Codex / Claude / Cursor / Gemini / OpenCode / AGY
```

O usuário escolhe `BUILD`, `INVESTIGATE`, `REVIEW`, `VERIFY`,
`ARCHITECT` ou `SECURITY` no Nexus. O Nexus mantém identidade, cost,
capacity, lineage e evidence independentemente do provider.

## 2. Não objetivos

Esta campanha não deve:

- criar novo Mission engine;
- criar novo ResourceScheduler;
- criar novo AgentMatcher;
- substituir `AgentSpec`;
- substituir Skills v1;
- criar marketplace público de agents;
- permitir peer-to-peer não rastreado;
- usar host-native subagents como state machine de Mission;
- sobrescrever arquivos do usuário sem ownership comprovado;
- declarar hot-switch/read-only/permissions suportados sem evidência;
- tornar AgentMode dependente de um provider fixo;
- tornar HUD autoridade de execução.

## 3. Invariantes herdados da V1

```text
AgentMatcher      = QUEM
ResourceScheduler = COM QUAL RECURSO
MissionRunner     = ESTADO DA EXECUÇÃO
WorkPlan          = O QUE FOI AUTORIZADO
AgentSpec         = ESPECIALIZAÇÃO PERSISTENTE
ExecutionRole     = PAPEL OPERACIONAL TEMPORÁRIO
```

Também permanecem obrigatórios:

- `UNKNOWN` não vira sucesso;
- side effects sem prova falham fechado;
- reviewer/investigator read-only só são `ENFORCED` com prova;
- provider/model/profile não são identidade do Agent;
- Web/Desktop/CLI usam o mesmo Core;
- WorkReceipt/Evidence continuam sendo a prova de conclusão;
- native host orchestration não substitui MissionRunner.

## 4. Estado existente a reutilizar

A implementação deve reutilizar diretamente:

- `internal/nexus/agent_modes.go`;
- `internal/nexus/intelligence.AgentSpec`;
- `internal/nexus/agent_matching.go`;
- `internal/nexus/config.go`;
- `internal/core/provider/spec`;
- `internal/control/driver`;
- `internal/control/registry`;
- `RuntimeGeneration`;
- `ResourceScheduler`;
- `MissionRunner`;
- `AutonomyContract`;
- `ContextCapsule`;
- `WorkReceipt`;
- Capacity Monitor;
- Cost Intelligence;
- HUD projection;
- Workspace OS Command Palette;
- profile homes isolados do Nexus.

É proibido criar uma cópia paralela dessas autoridades.

## 5. Modelo funcional

### 5.1 AgentMode

Continua sendo o catálogo persistente de especializações.

Exemplos existentes:

```text
developer
senior-developer
reviewer
investigator
architect
security-reviewer
qa-engineer
performance-investigator
documentation-specialist
generalist
backend-engineer
frontend-engineer
devops-release
data-analyst
```

### 5.2 ExecutionRole

Novo conceito orientado à sessão/turno.

Contrato inicial:

```go
type ExecutionRoleScope string

const (
    ExecutionRoleTurn    ExecutionRoleScope = "TURN"
    ExecutionRoleRuntime ExecutionRoleScope = "RUNTIME"
)

type AccessIntent string

const (
    AccessWrite       AccessIntent = "WRITE"
    AccessReadOnly    AccessIntent = "READ_ONLY"
    AccessVerifyOnly  AccessIntent = "VERIFY_ONLY"
)

type ExecutionRole struct {
    ID                     string                 `json:"id"`
    Label                  string                 `json:"label"`
    BaseAgentModeID        string                 `json:"base_agent_mode_id"`
    AccessIntent           AccessIntent           `json:"access_intent"`
    RequiredCapabilities   []string               `json:"required_capabilities,omitempty"`
    RecommendedSkillIDs    []string               `json:"recommended_skill_ids,omitempty"`
    DefaultScope           ExecutionRoleScope     `json:"default_scope"`
    VerificationPolicy     intelligence.VerificationPolicy
}
```

Catálogo inicial:

| ExecutionRole | Base AgentMode | Access intent |
| --- | --- | --- |
| `build` | `developer` | WRITE |
| `investigate` | `investigator` | READ_ONLY |
| `review` | `reviewer` | READ_ONLY |
| `verify` | `qa-engineer` | VERIFY_ONLY |
| `architect` | `architect` | READ_ONLY |
| `security` | `security-reviewer` | READ_ONLY |

A role `build` não substitui o AgentSpec persistente. Ela apenas define a
intenção operacional daquele contexto.

### 5.3 Composição do contexto

O compiler deve receber o papel explicitamente:

```go
type ExecutionContextRequest struct {
    Agent         AgentSpec
    ExecutionRole *ExecutionRoleContext
    Project       ProjectContext
    Task          WorkPackageContext
    Skills        []string
    Guidance      ExecutionGuidance
    Runtime       RuntimeConstraints
}
```

Precedência semântica:

```text
hard security/runtime constraints
    >
explicit WorkPackage authorization
    >
ExecutionRole
    >
persistent AgentSpec preferences
    >
provider defaults
```

ExecutionRole não pode ampliar autorização concedida pelo
`AutonomyContract`.

## 6. Binding de role

### 6.1 Runtime binding

Adicionar estado durável separado da revisão do Agent:

```go
type RuntimeExecutionRoleBinding struct {
    ID                  string
    RuntimeGenerationID string
    AgentID             string
    RoleID              string
    Scope               ExecutionRoleScope
    ActivationStrategy  ActivationStrategy
    Enforcement         EnforcementLevel
    ProjectionReceiptID string
    ActivatedBy         string
    ActivatedAt         time.Time
}
```

Requisitos:

- um binding ativo por RuntimeGeneration;
- troca gera evento durável;
- não altera `AgentRevision`;
- reconnect/refresh recupera o binding;
- nova RuntimeGeneration pode iniciar com role default;
- TURN override não substitui binding RUNTIME.

### 6.2 Turn override

`AskAgent`/API poderá receber:

```json
{
  "prompt": "...",
  "execution_role": "review",
  "execution_role_scope": "TURN"
}
```

Esse override:

- entra no prompt receipt;
- entra na atribuição de cost/telemetry;
- não modifica configuração persistente;
- expira ao término do turno.

## 7. Agent projection capabilities

Adicionar ao capability model atual:

```go
type AgentProfileCapabilities struct {
    Definitions      driver.CapabilityEvidence `json:"definitions"`
    PrimaryAgent     driver.CapabilityEvidence `json:"primary_agent"`
    Subagents        driver.CapabilityEvidence `json:"subagents"`
    InSessionSwitch  driver.CapabilityEvidence `json:"in_session_switch"`
    DynamicReload    driver.CapabilityEvidence `json:"dynamic_reload"`
    ToolPermissions  driver.CapabilityEvidence `json:"tool_permissions"`
    ModelOverride    driver.CapabilityEvidence `json:"model_override"`
    ProjectScope     driver.CapabilityEvidence `json:"project_scope"`
    UserScope        driver.CapabilityEvidence `json:"user_scope"`
    RemoteProtocol   driver.CapabilityEvidence `json:"remote_protocol"`
}
```

Adicionar campo em `EffectiveCapabilities`:

```go
AgentProfiles AgentProfileCapabilities `json:"agent_profiles"`
```

### 7.1 Regra de truth

`SUPPORTED` exige teste/probe aceito.

Se a documentação diz que existe, mas Nexus ainda não validou no provider
detectado:

```text
NOT_TESTED
```

Se parte do contrato foi validada:

```text
PARTIAL
```

Se a versão detectada não é reconhecida:

```text
UNKNOWN
```

Nunca retornar `SUPPORTED, Tested:false` para uma capability usada como
boundary de segurança.

## 8. Enforcement

```go
type EnforcementLevel string

const (
    EnforcementEnforced    EnforcementLevel = "ENFORCED"
    EnforcementBestEffort  EnforcementLevel = "BEST_EFFORT"
    EnforcementUnverified  EnforcementLevel = "UNVERIFIED"
    EnforcementUnsupported EnforcementLevel = "UNSUPPORTED"
)
```

Resolver deve considerar:

- `ExecutionRole.AccessIntent`;
- provider `ReviewIsolation`;
- sandbox;
- tool permissions nativas;
- project/worktree isolation;
- `AutonomyContract`;
- capability evidence.

Para `READ_ONLY`, se nenhuma camada técnica comprovar bloqueio de escrita,
Nexus pode executar via overlay apenas quando a policy permitir, mas deve
mostrar `BEST_EFFORT`/alerta e não `ENFORCED`.

## 9. Projection adapter

Não adicionar métodos de filesystem/configuração diretamente em
`ControlDriver`.

Novo pacote sugerido:

```text
internal/core/provider/agentprojection/
├── contract.go
├── registry.go
├── ownership.go
├── opencode/
├── claude/
├── cursor/
├── gemini/
├── codex/
└── agy/
```

Contrato:

```go
type ProjectionScope string

const (
    ProjectionNexusProfile ProjectionScope = "NEXUS_PROFILE"
    ProjectionProject      ProjectionScope = "PROJECT"
    ProjectionUser         ProjectionScope = "USER"
)

type ActivationStrategy string

const (
    ActivationNativeHotSwitch ActivationStrategy = "NATIVE_HOT_SWITCH"
    ActivationNativeOnLaunch  ActivationStrategy = "NATIVE_ON_LAUNCH"
    ActivationNativeSubagent  ActivationStrategy = "NATIVE_SUBAGENT"
    ActivationNexusOverlay    ActivationStrategy = "NEXUS_OVERLAY"
    ActivationUnsupported     ActivationStrategy = "UNSUPPORTED"
)

type ProjectionRequest struct {
    Provider        string
    Profile         string
    Workspace       string
    Scope           ProjectionScope
    AgentMode       nexus.AgentMode
    ExecutionRole   nexus.ExecutionRole
    EffectiveSpec   intelligence.AgentSpec
}

type GeneratedArtifact struct {
    Path        string
    Content     []byte
    Mode        fs.FileMode
    ContentHash string
}

type ProjectionPlan struct {
    Provider   string
    Scope      ProjectionScope
    Strategy   ActivationStrategy
    Artifacts  []GeneratedArtifact
    Conflicts  []ProjectionConflict
    Warnings   []string
}

type ProjectionReceipt struct {
    ID               string
    Provider         string
    Profile          string
    RoleID           string
    Scope            ProjectionScope
    Strategy         ActivationStrategy
    RendererVersion  string
    SourceHash       string
    Artifacts        []ProjectionArtifactReceipt
    AppliedAt        time.Time
}

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

## 10. Ownership/reconciliation

### 10.1 Default

`NEXUS_PROFILE` é o default.

O Nexus deve aproveitar o profile home já isolado por provider.

### 10.2 Project scope

Somente por ação explícita:

```bash
nexus agents export --provider cursor --scope project
```

Antes de escrever:

- calcular plan;
- detectar conflicts;
- mostrar paths;
- exigir confirmação em modo interativo;
- aceitar `--yes` apenas em automação explícita.

### 10.3 User scope

Nunca default.

Deve exigir:

```bash
--scope user
```

e preview de paths.

### 10.4 Regras de overwrite

```text
MISSING
  -> CREATE

MANAGED_UNMODIFIED
  -> UPDATE

MANAGED_MODIFIED
  -> CONFLICT

UNMANAGED
  -> CONFLICT / NEVER OVERWRITE
```

`--force` não deve sobrescrever arquivo unmanaged silenciosamente.
Se existir futuramente, deverá exigir backup + confirmação explícita e registrar
receipt.

## 11. Provider adapters — baseline alvo

### 11.1 OpenCode

Alvo:

- gerar agents nativos no profile isolado;
- distinguir primary/subagent/all;
- mapear read-only para permissions quando suportado;
- validar discovery;
- validar seleção por TUI/command quando automatizável;
- não assumir hot-switch em sessão existente sem probe.

Referência externa atual:
https://opencode.ai/v2/docs/agents

Primeiro provider recomendado para E2E de primary Agent devido ao contrato
explícito de primary/subagent.

### 11.2 Claude Code

Alvo:

- gerar `.claude/agents/*.md` no profile do Nexus;
- mapear tools/skills quando compatível;
- validar discovery via `/agents` ou superfície equivalente;
- usar subagent nativo quando comprovado;
- main-role continua overlay enquanto primary custom não estiver comprovado
  para a versão suportada.

Referência:
https://code.claude.com/docs/

### 11.3 Cursor

Alvo:

- gerar `.cursor/agents/*.md`;
- expor reviewer/investigator/verifier como subagents;
- preservar Nexus como seletor principal de ExecutionRole;
- não tentar transformar Cursor Agent em primary custom sem capability.

Referência:
https://prod.cursor.com/docs/subagents

### 11.4 Gemini CLI

Alvo:

- gerar `.gemini/agents/*.md`;
- usar local custom subagents quando comprovado;
- mapear tools/model apenas quando a versão suportar;
- A2A remote agents fica fora do primeiro milestone.

Referência:
https://geminicli.com/docs/core/subagents/

### 11.5 Codex

Primeira entrega:

- `NEXUS_OVERLAY`;
- manter `codex exec`/terminal atual;
- não depender de diretório de custom agent como contrato estável.

Segunda entrega:

- investigar/migrar parte do control plane para `codex app-server`;
- streaming estruturado;
- approvals;
- cancel turn;
- lifecycle de conversation;
- eventual integração mais profunda de role.

Referência:
https://developers.openai.com/pt-BR/blog/codex-as-a-platform

### 11.6 AGY

- overlay primeiro;
- capability probe própria;
- não compartilhar adapter com OpenCode apenas por semelhança histórica;
- manter provider ID e contrato independentes.

## 11.7 Custom execution targets

A V1 deve suportar provider lógico separado do mecanismo usado para iniciar o
processo. Um Agent OpenCode pode, por exemplo, executar pelo binário local, por
Docker/VPN Workspace, Docker Compose, WSL, SSH ou outro wrapper supervisionado.

O contrato de compatibilidade é `command_template`, argv-only e sem shell:

```text
{cwd}
{target_cwd}
{args}
{provider}
{profile}
{profile_home}
```

`{args}` deve ocupar um token inteiro. Operadores de shell, redirecionamento e
command substitution continuam proibidos.

Metadados tipados da V1:

```text
execution_adapter=command_template
target_kind=local|docker|docker-compose|wsl|ssh|custom
target_workspace=<path visible inside target>       # optional, defaults to {cwd}
target_profile_home=<host-visible shared home>   # opcional
target_probe_template=<safe argv template>       # recomendado
target_provider_version=<declared fallback>      # não preferido
```

`{cwd}` é o workspace no host Nexus. `{target_cwd}` é o caminho visto pelo
target e cai para `{cwd}` quando não configurado. Isso cobre, por exemplo,
Windows `C:\\...` montado como `/workspace` no container.

Para targets como `vpn-dev-workspace`, execução customizada não depende de
Agent Projection. Se `target_profile_home` não for compartilhada/visível ao
Nexus, o runtime continua funcionando por command template + Nexus overlay, mas
a projeção nativa permanece indisponível/UNVERIFIED.

Quando uma home compartilhada existe, o Nexus pode materializar os agents
nativos no mesmo filesystem visto pelo OpenCode no target. A versão do provider
deve preferencialmente ser detectada pelo `target_probe_template` no próprio
target, e não inferida a partir do host.

### VPN Dev Workspace compatibility

A V1 considera os dois layouts reais do `vpn-dev-workspace`:

- Linux/WSL Linux filesystem: home e workspace são bind-mounted no mesmo caminho.
  `target_workspace` pode permanecer vazio e `target_profile_home` pode apontar
  para a home compartilhada do host.
- Windows + Docker Desktop: o projeto Windows é montado em `/workspace` e a
  home Linux padrão vive no volume Docker `vpn_windows_home`. Configure
  `target_workspace=/workspace`. Não inventar uma `target_profile_home`
  host-visible para esse volume: custom execution continua suportada e a role
  degrada para Nexus overlay/BEST_EFFORT até existir um transporte de
  materialização verificável para o volume.

Exemplo Windows:

```text
provider=opencode
target_kind=docker
target_workspace=/workspace
command_template=docker exec -it -w "{target_cwd}" vpn-dev-workspace-terminal-1 {provider} {args}
target_probe_template=docker exec vpn-dev-workspace-terminal-1 {provider} {args}
```

Read-only em custom target nunca é automaticamente ENFORCED. Até existir
discovery + activation + permission-enforcement evidence do target, a UI e o
Core devem permanecer em BEST_EFFORT/UNVERIFIED.

## 12. Correções prévias obrigatórias

### P0 — capability truth

Remover qualquer uso relevante de:

```text
SUPPORTED + Tested:false
```

quando a capability for usada para sandbox/read-only/activation.

### P0 — AGY != OpenCode

Web e configuração devem tratar `agy` e `opencode` como providers separados.

O Core já os registra separadamente.

### P0 — provider preference

`AgentMode.DefaultProvider` deve deixar de funcionar como associação de
identidade.

Opções:

1. renomear para `PreferredProvider`; ou
2. remover do catálogo e derivar requisitos para Scheduler.

Preferência é `PREFER`, nunca `PIN`.

### P0 — AgentMode != ExecutionRole

Não reutilizar `SafeApply` para role switch.

Role switch não pode produzir nova `AgentRevision`.

## 13. API local

Endpoints propostos:

```text
GET  /api/v1/agent-roles
GET  /api/v1/providers/{id}/agent-capabilities

GET  /api/v1/runtimes/{id}/execution-role
POST /api/v1/runtimes/{id}/execution-role

POST /api/v1/agent-projections/plan
POST /api/v1/agent-projections/apply
GET  /api/v1/agent-projections/{id}
POST /api/v1/agent-projections/{id}/verify
DELETE /api/v1/agent-projections/{id}
```

Exemplo de troca:

```json
POST /api/v1/runtimes/rt_x/execution-role

{
  "role_id": "review",
  "scope": "RUNTIME"
}
```

Resposta mínima:

```json
{
  "runtime_id": "rt_x",
  "agent_id": "agt_x",
  "role_id": "review",
  "activation_strategy": "NEXUS_OVERLAY",
  "enforcement": "ENFORCED",
  "projection_health": "VERIFIED"
}
```

API continua interna. Automação externa deve preferir CLI `--json`.

## 14. CLI

Comandos propostos:

```bash
nexus agents modes
nexus agents roles

nexus agent role <agent-id> review
nexus agent role <agent-id> build
nexus agent role <agent-id> --show

nexus agents projection plan --provider opencode
nexus agents projection apply --provider opencode
nexus agents projection verify --provider opencode

nexus agents export --provider cursor --scope project
nexus agents export --provider gemini --scope user

nexus doctor agents
```

Todos devem possuir `--json` onde fizer sentido.

`nexus doctor agents` deve apresentar matriz evidence-bound.

## 15. UX Web/Desktop

### 15.1 Header de terminal

Adicionar no contexto da AI Session/Agent terminal:

```text
Agent: Senior Developer
Role: REVIEW ▾
Provider: Codex
Read-only: ENFORCED
```

Não mostrar detalhes avançados por default.

Tooltip/details:

- activation strategy;
- projection health;
- capability evidence;
- provider version;
- last projection verification.

### 15.2 Command Palette

Adicionar grupo `Agent Role`:

```text
Switch role: Build
Switch role: Investigate
Switch role: Review
Switch role: Verify
Switch role: Architect
Switch role: Security
```

### 15.3 Shortcut

Não usar `Shift+Tab` como default Nexus.

Adicionar shortcut configurável; candidato inicial:

```text
Ctrl/Cmd+Shift+A
```

O popup deve aceitar teclado completo e respeitar focus trap/accessibility.

### 15.4 Native host selector

Quando o host possuir selector próprio:

- não sequestrar keybind;
- não tentar pintar estado como sincronizado sem confirmação;
- mostrar `Native agent available` quando apenas materializado;
- mostrar `Native agent active` somente após activation evidence.

## 16. HUD

HUD recebe apenas projeção do Core.

Campos aditivos:

```go
ExecutionRole       string
ActivationStrategy  string
Enforcement         string
ProjectionHealth    string
```

Modo AGENT pode exibir:

```text
REVIEW · Claude
read-only ENFORCED
projection VERIFIED
```

HUD não executa projection nem troca provider diretamente.

## 17. Cost/Capacity

Toda observação deve continuar atribuída ao mesmo Agent/RuntimeGeneration.

Adicionar dimensions quando disponíveis:

```text
execution_role
activation_strategy
native_subagent_name
```

Trocar role não reinicia contador nem cria outro Agent.

ResourceScheduler pode considerar `RequiredCapabilities` da role.

Exemplo:

```text
review
  requires: read_only_review
```

Se provider atual não satisfaz:

- em Direct/Interactive, solicitar escolha/fallback conforme policy;
- em Mission, scheduler segue contratos existentes;
- `PREFER` pode cair para alternativa;
- `PIN` falha fechado.

## 18. Mission integration

### Milestone inicial

Mission usa `ExecutionRole` como informação de package/contexto, mas continua
executando pelo caminho canônico.

Não delegar package inteira ao orquestrador interno do provider.

### Futuro

Uma native subagent execution só poderá participar de Mission se retornar:

- identidade correlacionável;
- bounded input;
- bounded output;
- exit/result;
- changed files;
- evidence;
- cancel semantics;
- no-progress control;
- ownership de side effects.

Até lá, native subagents ficam restritos a Direct/Interactive/auxiliary work.

## 19. Persistência/migration

Criar migration SQLite aditiva.

Tabela sugerida:

```sql
runtime_execution_roles (
  id TEXT PRIMARY KEY,
  runtime_generation_id TEXT NOT NULL,
  agent_id TEXT NOT NULL,
  role_id TEXT NOT NULL,
  scope TEXT NOT NULL,
  activation_strategy TEXT NOT NULL,
  enforcement TEXT NOT NULL,
  projection_receipt_id TEXT,
  activated_by TEXT,
  activated_at TEXT NOT NULL
)
```

Tabela de receipts:

```sql
agent_projection_receipts (
  id TEXT PRIMARY KEY,
  provider TEXT NOT NULL,
  profile TEXT NOT NULL,
  role_id TEXT NOT NULL,
  scope TEXT NOT NULL,
  strategy TEXT NOT NULL,
  renderer_version TEXT NOT NULL,
  source_hash TEXT NOT NULL,
  payload_json TEXT NOT NULL,
  applied_at TEXT NOT NULL,
  verified_at TEXT
)
```

Não colocar secrets, tokens ou conteúdo sensível no receipt.

## 20. Eventos

Adicionar eventos duráveis:

```text
AGENT_ROLE_CHANGED
AGENT_PROJECTION_PLANNED
AGENT_PROJECTION_APPLIED
AGENT_PROJECTION_VERIFIED
AGENT_PROJECTION_CONFLICT
AGENT_PROJECTION_REMOVED
```

Payload deve evitar prompt/transcript bruto.

## 21. Telemetria

Somente metadata allowlisted:

- provider;
- role id;
- strategy;
- enforcement;
- result;
- duration bucket;
- conflict reason class.

Sem:

- system prompt;
- paths completos quando identificáveis;
- conteúdo do artifact;
- user prompt;
- secrets.

## 22. Security

### 22.1 Path safety

Projection adapters devem:

- normalizar paths;
- rejeitar traversal;
- rejeitar symlink escape;
- usar atomic write;
- preservar mode;
- usar backup quando aplicável;
- validar ownership antes de update/remove.

### 22.2 Prompt injection

Native agent file é configuração executável do host.

Conteúdo proveniente de projeto/arquivo externo nunca deve ser interpolado em
frontmatter/config sem escaping/validation.

### 22.3 Project export

Antes de exportar para repositório:

- verificar worktree;
- mostrar diff;
- nunca stage/commit automaticamente;
- nunca push;
- respeitar AutonomyContract.

### 22.4 Removal

Remove somente artifacts cujo receipt/hash/ownership sejam comprovados.

## 23. Version awareness

Cada provider adapter deve registrar:

```text
provider version
renderer version
probe version
last verified capability set
```

Mudança de provider version deve poder rebaixar capabilities a `UNKNOWN` até
nova verificação se o contrato mudou fora da faixa reconhecida.

Não manter `KnownVersions` como única prova; probes devem ser preferidos.

## 24. Probes

Cada adapter precisa de probes pequenos e determinísticos.

Exemplos:

### OpenCode

- materializar fixture agent;
- iniciar em profile isolado;
- listar/descobrir agents;
- validar primary/subagent visibility;
- validar permission deny quando aplicável.

### Cursor

- materializar `.cursor/agents`;
- iniciar CLI no profile isolado;
- comprovar discovery;
- invocar explicitamente subagent em fixture quando suportado.

### Gemini

- materializar `.gemini/agents`;
- executar `/agents`/mecanismo testável;
- validar discovery;
- validar tool restrictions.

### Claude

- materializar `.claude/agents`;
- validar discovery por comando suportado;
- executar subagent fixture sem side effect.

### Codex

Primeiro milestone comprova apenas overlay.

App-server terá campanha separada.

## 25. Test strategy

### Unit

- role catalog;
- composition precedence;
- enforcement resolver;
- projection path resolver;
- ownership state machine;
- conflict detection;
- renderer goldens;
- version capability downgrade;
- scope policy.

### Integration

- SQLite migration;
- binding survives reload;
- TURN override does not replace RUNTIME role;
- projection plan/apply/verify/remove;
- unmanaged file is never overwritten;
- modified managed file conflicts;
- AgentRevision count unchanged on role switch;
- cost attribution remains same Agent;
- ResourceScheduler requirements incorporate role capabilities.

### E2E provider

Fixture repository mínimo.

Para cada provider promovido:

```text
detect
→ isolated profile
→ materialize
→ discover
→ activate/invoke
→ enforce expected permissions
→ verify
→ cleanup
```

### Web E2E

- keyboard role selector;
- Command Palette;
- accessibility;
- mobile layout;
- stale response latest-wins;
- terminal remains alive across role switch;
- role chip updates without recreating PTY quando overlay permitir.

### Desktop

Mesmo Core.

Sem lógica de role no shell nativo.

Validar Linux primeiro; Windows/macOS entram conforme plataforma for certificada.

## 26. Feature flags

Adicionar:

```text
agent_interop
agent_native_projection
```

Sugestão:

```text
agent_interop:
  experimental -> beta -> default on

agent_native_projection:
  experimental
  provider-gated
```

Overlay pode amadurecer antes da projeção nativa.

## 27. Entitlements

A arquitetura não deve exigir entitlement para a lógica básica de role.

Caso o produto comercialize features no futuro, o boundary deve ser capability
server-side já existente, por exemplo:

```text
advanced_agent_profiles
team_agent_catalog
managed_agent_distribution
```

Não bloquear segurança básica/read-only atrás de plano comercial.

## 28. Compatibilidade

- Agents v1 continuam válidos;
- ausência de role binding = comportamento atual;
- `AgentSpec` JSON atual continua legível;
- provider sem adapter continua via overlay;
- project sem arquivos de agents continua funcionando;
- migration é aditiva;
- rollback da feature não perde Agent identity;
- Mission sem ExecutionRole continua comportamento v1.

## 29. Sequência de implementação

### M0 — Freeze e truth

- preservar/taggear v1.0.0 antes de promoção funcional;
- capability audit;
- corrigir AGY/OpenCode UI;
- corrigir `SUPPORTED/Tested:false` onde relevante;
- formalizar preferência de provider.

### M1 — ExecutionRole Core

- catálogo;
- resolver;
- compiler;
- store/migration;
- events;
- API/CLI;
- unit/integration tests.

Gate:

```text
role switch does not create AgentRevision
```

### M2 — Nexus UX

- header selector;
- Command Palette;
- keyboard;
- API state;
- accessibility;
- HUD projection.

Gate:

```text
same UX works with a provider that has no native agent support
```

### M3 — Projection Core

- contract;
- ownership;
- plan/apply/verify/remove;
- receipts;
- doctor output;
- goldens.

Gate:

```text
unmanaged user file cannot be overwritten
```

### M4 — OpenCode adapter

- primary/subagent renderers;
- permissions;
- real E2E;
- activation evidence.

Gate:

```text
native discovery + invocation + read-only evidence
```

### M5 — Claude adapter

- subagent renderer;
- discovery;
- E2E;
- overlay/native strategy resolution.

### M6 — Cursor adapter

- subagent renderer;
- discovery/invocation;
- E2E;
- no false primary-agent claim.

### M7 — Gemini adapter

- local subagent renderer;
- discovery/invocation;
- E2E;
- A2A explicitly deferred.

### M8 — Codex control evolution

- evaluate app-server adapter;
- structured events;
- approvals;
- cancel;
- conversation lifecycle;
- preserve terminal fallback.

Não bloquear M1–M7 em M8.

### M9 — AGY adapter

- provider-specific capabilities;
- projection se houver contrato estável;
- otherwise overlay remains supported.

### M10 — Release certification

- provider/version matrix;
- Linux;
- Windows/macOS where native platform certification exists;
- install/upgrade regression;
- HUD regression;
- Mission regression;
- cost/capacity regression.

## 30. Branch strategy

A especificação deixou de ser planning-only: o escopo foi absorvido pela V1 beta. A implementação deve permanecer em branches funcionais da linha V1:

```text
feat/v1-agent-interop
```

Implementação funcional deve iniciar de uma `main` que contenha o baseline
v1.0.0 preservado e os merges pós-v1 intencionais.

Branches funcionais devem explicitar o escopo V1, por exemplo:

```text
feat/v1-opencode-agent-projection
```

Evitar desenvolver Agent Interop dentro de:

- `feat/native-platform-support`;
- branch histórica de closure;
- branch de Capacity;
- branch de HUD já mergeada.

## 31. Definition of Done

A feature só pode ser considerada concluída quando:

1. `ExecutionRole` não muda Agent identity.
2. Role switch não gera `AgentRevision` artificial.
3. Role TURN e RUNTIME possuem semântica testada.
4. Provider selection continua pertencendo ao ResourceScheduler.
5. `DefaultProvider` não funciona como PIN oculto.
6. Capability matrix é evidence-bound.
7. Read-only exibe enforcement real.
8. OpenCode projection possui E2E real.
9. Pelo menos um segundo provider possui projection E2E real.
10. Cursor/Gemini/Claude sem proof de primary não exibem primary como supported.
11. Codex funciona por overlay mesmo sem app-server.
12. Unmanaged file nunca é sobrescrito.
13. Modified managed file gera conflict.
14. Remove só remove artifact comprovadamente gerenciado.
15. Profile scope é default.
16. Project/User scope são opt-in.
17. Command Palette troca role.
18. Shortcut não quebra input do TUI hospedado.
19. PTY não é recriado desnecessariamente em overlay role switch.
20. HUD apenas projeta o estado.
21. Cost permanece atribuível ao mesmo Agent/runtime.
22. Capacity/scheduler recebem requirements da role.
23. MissionRunner continua autoridade única.
24. Mission regression suite permanece verde.
25. Web accessibility permanece verde.
26. Linux CI permanece verde.
27. Windows/macOS não recebem claim sem native evidence.
28. Docs distinguem native/overlay/best-effort/enforced.
29. Provider version desconhecida rebaixa evidence quando necessário.
30. Rollback/desabilitar feature preserva Agents v1.

## 32. Riscos

### Drift de formatos externos

Mitigação:

- adapter versionado;
- goldens;
- runtime probes;
- `UNKNOWN` em versão desconhecida.

### UX inconsistente entre hosts

Mitigação:

- Nexus selector é canônico;
- native selector é otimização, não requisito.

### Dupla orquestração

Mitigação:

- native agents não assumem Mission state;
- MissionRunner permanece único owner.

### Falsa segurança de read-only

Mitigação:

- EnforcementLevel explícito;
- capability evidence;
- workspace fingerprint onde aplicável.

### Poluição de configuração do usuário

Mitigação:

- profile scope default;
- ownership receipt;
- conflict fail-closed.

### Explosão de agents no selector

Mitigação:

- poucos ExecutionRoles de alto nível;
- especializações adicionais continuam no catálogo de Agents/subagents;
- roles e persistent AgentModes são conceitos distintos.

## 33. Métricas de produto

Sem conteúdo de prompt:

- role switches por sessão;
- roles utilizadas;
- native vs overlay;
- projection conflicts;
- projection verification success;
- provider fallback por capability;
- tempo de activation;
- taxa de downgrade para BEST_EFFORT/UNKNOWN;
- custo por execution role;
- subagent invocation count quando observável.

## 34. Critérios para V1.1 RC

Para declarar `NEXUS_V1_AGENT_INTEROP = GO`:

- P0 truth fixes concluídos;
- M1–M3 completos;
- OpenCode E2E PASS;
- pelo menos um entre Claude/Cursor/Gemini E2E PASS;
- nenhum BLOCKER/HIGH aberto no contrato;
- provider matrix gerada do mesmo SHA;
- Linux full CI + browser + desktop regressions PASS;
- Mission regression PASS;
- HUD regression PASS;
- migration forward/backward compatibility documentada;
- docs de usuário publicadas;
- claims de native support limitadas ao que possui evidência.

## 35. Referências internas

- `ARCHITECTURE.md`
- `internal/nexus/agent_modes.go`
- `internal/nexus/intelligence/types.go`
- `internal/nexus/agent_matching.go`
- `internal/nexus/config.go`
- `internal/core/provider/spec/`
- `internal/control/driver/`
- `docs/product/agents.md`
- `docs/architecture/agent-routing.md`
- `docs/architecture/mission-teams.md`
- `docs/architecture/guardrails.md`
- `docs/product/hud.md`
- `docs/architecture/ADR-agent-interop-projection.md`

## 36. Primeira implementação recomendada

Quando a branch funcional for criada, a primeira PR de código deve ser
deliberadamente pequena:

```text
ExecutionRole catalog
+ runtime binding store
+ compiler support
+ API read/write
+ tests proving no AgentRevision change
```

Não incluir projection adapters nessa primeira PR.

Isso cria o conceito corretamente no Core antes de acoplar formatos externos e
reduz o risco de desenhar o modelo em função do OpenCode.
