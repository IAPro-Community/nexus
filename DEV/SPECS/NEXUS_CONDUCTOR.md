# Nexus Conductor — Spec Técnica

> Status: implementado (Fases 1–7)
> Plano original: `.omx/plans/2026-09-13-nexus-conductor.md`

## 1. Visão Geral

O **Nexus Conductor** é a camada de orquestração nativa que conecta Goals (objetivos do usuário) a Missions (execução autônoma). Ele unifica os caminhos de promoção CLI e Web num único `Dispatch()` Core, persiste uma política global, instrumenta cada execução para observabilidade, e separa worker efêmero de agente persistente.

```text
Usuário: "Implementar autenticação JWT"
         ↓
    Conductor.Dispatch()
         ↓
    DecomposePrompt → FlowProposal
         ↓
    CreateWorkPlan → WorkPlan
         ↓
    StartMissionRun → MissionRun
         ↓
    Agent executa (headless, read-only review)
         ↓
    GoalAssessment (PASS/FAIL/REDO)
         ↓
    Resultado sintetizado ao Lead
```

## 2. Invariantes

1. Não criar segundo WorkPlan, Flow engine, MissionRunner, Agent model, ResourceScheduler ou evidence ledger.
2. Preservar Agent persistente como identidade principal.
3. `nexus agy -p` e `nexus run <provider>` continuam passthrough direto.
4. Conductor usa o pipeline existente Intent → Delegation → WorkPlan → MissionRunner → ValidationEvidence.
5. Integração Git é globalmente `OFF` por padrão.
6. Provider nunca recebe permissão implícita para commit, push, deploy ou Git destrutivo.
7. Nenhuma credencial, transcript bruto ou prompt sensível entra em API, evento ou log.
8. Worker efêmero usa Agent descartável filtrado e reconciliado.

## 3. Schema — Migração 0019

```sql
-- nexus_settings: singleton global para política do Conductor
CREATE TABLE IF NOT EXISTS nexus_settings (
    key        TEXT PRIMARY KEY,
    value      TEXT NOT NULL DEFAULT '{}',
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_nexus_settings_updated ON nexus_settings(updated_at);

-- agents: coluna identity_kind para distinguir PERSISTENT de EPHEMERAL_WORKER
ALTER TABLE agents ADD COLUMN identity_kind TEXT NOT NULL DEFAULT 'PERSISTENT';
```

### Store methods

| Método | Arquivo | Descrição |
|---|---|---|
| `GetSetting(key)` | `store/settings.go` | Lê uma chave do nexus_settings |
| `PutSetting(key, value)` | `store/settings.go` | Upsert idempotente |
| `ListAgentsFiltered(projectID, identityKind)` | `store/agents.go` | Lista agentes com filtro por identity_kind |
| `ReconcileEphemeralWorkers()` | `store/agents.go` | Remove workers STOPPED/FAILED |
| `PurgeExpiredEventsSince(now, retentionDays)` | `store/events_purge.go` | Remove eventos expirados |

## 4. Tipos Core

### 4.1 ConductorPolicy

Persistido em `nexus_settings` sob key `"conductor:policy"`.

```go
type ConductorPolicy struct {
    Version          int    `json:"version"`
    ReviewMode       *string `json:"review_mode,omitempty"`       // nil = defaults
    RetentionDays    *int    `json:"retention_days,omitempty"`    // default 90
    IntegrationOptIn *bool   `json:"integration_opt_in,omitempty"` // default false
}
```

- `GetConductorPolicy(st)` — lê; se ausente ou inválido, retorna default.
- `PutConductorPolicy(st, p)` — persiste; idempotente.

### 4.2 GoalContract

Snapshot imutável do objetivo no momento da promoção. Congelado no `ExecutionSnapshot`.

```go
type GoalContract struct {
    GoalDescription      string    `json:"goal_description"`
    AcceptedAssumptions  []string  `json:"accepted_assumptions,omitempty"`
    AutoPromoteToMission bool      `json:"auto_promote_to_mission"`
    LeadID               string    `json:"lead_id"`
    ProjectID            string    `json:"project_id"`
    GoalID               string    `json:"goal_id"`
    CreatedAt            time.Time `json:"created_at"`
}
```

### 4.3 GoalAssessment

Veredicto estruturado produzido pelo read_only_review.

```go
type GoalAssessmentResult string
const (
    GoalAssessmentPass GoalAssessmentResult = "PASS"
    GoalAssessmentFail GoalAssessmentResult = "FAIL"
    GoalAssessmentRedo GoalAssessmentResult = "REDO"
)

type GoalAssessment struct {
    Result      GoalAssessmentResult `json:"result"`
    Summary     string               `json:"summary"`
    Violations  []string             `json:"violations,omitempty"`
    Assumptions []string             `json:"assumptions,omitempty"`
    AssessedAt  time.Time            `json:"assessed_at"`
}
```

### 4.4 AgentInvocation

Registro de cada execução de agente. Armazenado como JSON em event metadata (`kind="agent_invocation"`).

```go
type AgentInvocationStatus string
const (
    AgentInvocationRunning   AgentInvocationStatus = "RUNNING"
    AgentInvocationCompleted AgentInvocationStatus = "COMPLETED"
    AgentInvocationFailed    AgentInvocationStatus = "FAILED"
    AgentInvocationCancelled AgentInvocationStatus = "CANCELED"
)

type AgentInvocation struct {
    GoalID     string                `json:"goal_id"`
    AgentID    string                `json:"agent_id"`
    ProjectID  string                `json:"project_id"`
    Status     AgentInvocationStatus `json:"status"`
    StartedAt  time.Time             `json:"started_at"`
    FinishedAt *time.Time            `json:"finished_at,omitempty"`
    Assessment *GoalAssessment       `json:"assessment,omitempty"`
    Errors     []string              `json:"errors,omitempty"`
}
```

### 4.5 IntegrationReceipt

Registro de uma tentativa de integração (fast-forward ff-only).

```go
type IntegrationReceipt struct {
    BaseSHA  string    `json:"base_sha"`
    HeadSHA  string    `json:"head_sha"`
    Status   string    `json:"status"` // "SUCCESS" | "CONFLICT"
    Error    string    `json:"error,omitempty"`
    PulledAt time.Time `json:"pulled_at"`
}
```

### 4.6 ExecutionIdentity / AllocationResult

```go
type ExecutionIdentityKind string
const (
    ExecutionIdentityPersistent      ExecutionIdentityKind = "PERSISTENT"
    ExecutionIdentityEphemeralWorker ExecutionIdentityKind = "EPHEMERAL_WORKER"
)

type ExecutionIdentity struct {
    Kind ExecutionIdentityKind `json:"kind"`
}

type AllocationResult struct {
    AgentID   string `json:"agent_id"`
    Workspace string `json:"workspace"`
    Kind      string `json:"kind,omitempty"`
}
```

## 5. Dispatch — Entrypoint Único

`Conductor.Dispatch()` é o único entrypoint para roteamento de goals. CLI e Web delegam a ele.

```go
type DispatchRequest struct {
    ProjectID            string         `json:"project_id"`
    Goal                 string         `json:"goal"`
    SourcePrompt         string         `json:"source_prompt"`
    LeadAgentID          string         `json:"lead_agent_id"`
    DelegationMode       DelegationMode `json:"delegation_mode"`
    GoalID               string         `json:"goal_id,omitempty"`
    AutoPromoteToMission bool           `json:"auto_promote_to_mission"`
}

type DispatchResult struct {
    GoalID           string             `json:"goal_id"`
    PlanID           string             `json:"plan_id,omitempty"`
    RunID            string             `json:"run_id"`
    Status           DispatchStatus     `json:"status"`
    Run              *runner.MissionRun `json:"run,omitempty"`
    AdmissionBlocked bool               `json:"admission_blocked,omitempty"`
    AdmissionStage   string             `json:"admission_stage,omitempty"`
    FallbackSafe     bool               `json:"fallback_safe,omitempty"`
    Error            string             `json:"error,omitempty"`
}

type DispatchStatus string
const (
    DispatchAccepted  DispatchStatus = "ACCEPTED"
    DispatchRejected  DispatchStatus = "REJECTED"
    DispatchFailed    DispatchStatus = "FAILED"
    DispatchCompleted DispatchStatus = "COMPLETED"
)
```

### Pipeline

1. `DecomposePromptIntoFlowProposal` → FlowProposal
2. `WorkPlanFromFlow` → WorkPlan
3. `ReadDelegationDecisionFacts` → valida delegação
4. `WithLeadAgent` → vincula Lead
5. `CreateWorkPlan` → persiste WorkPlan
6. `StartMissionRun` → executa admission e, somente se aceita, cria `MissionRun` + worker
7. `WaitDispatch` / `waitPromotedMission` → aguarda estado estável

Para terminais interativos, `StartDispatch` é executado enquanto o Enter ainda
está retido e a linha original permanece no provider. Uma promoção heurística
que falha com `FallbackSafe=true` pode voltar ao Lead com segurança porque
nenhum `MissionRun` foi iniciado. Bloqueios conhecidos de admission também
carregam `AdmissionBlocked=true` e `AdmissionStage` para UX acionável. Pedido explícito/confirmado de Mission não
degrada silenciosamente para Direct. Falha não tipada também não faz fallback
automático, pois o boundary de side effects não foi provado.

## 6. CLI — `nexus conductor`

```text
nexus conductor policy get           # mostra política atual
nexus conductor policy set '<json>'  # atualiza política
nexus conductor dispatch <goal>      # despacha objetivo
nexus conductor dispatch --json <goal>  # saída JSON
```

## 7. API HTTP

| Método | Rota | Descrição |
|---|---|---|
| `GET` | `/api/v1/conductor/policy` | Retorna política atual |
| `PUT` | `/api/v1/conductor/policy` | Atualiza política |
| `POST` | `/api/v1/conductor/dispatch` | Despacha objetivo |

## 8. Worker Efêmero

- `IdentityKind = "EPHEMERAL_WORKER"` no `agents.identity_kind`.
- `ListAgents()` exclui ephemeral workers por padrão.
- `ListAgentsFiltered(projectID, "EPHEMERAL_WORKER")` inclui explicitamente.
- `ReconcileEphemeralWorkers()` remove workers STOPPED/FAILED no startup.
- Criado via `CreateAgent` com `IdentityKind: "EPHEMERAL_WORKER"`.
- Não aparece em `ListAgents` nem na UI.

## 9. read_only_review

- Capability `ReadOnlyReview` adicionada a `EffectiveCapabilities` e `ControlCapabilities`.
- derivada de: `len(spec.ReviewMode) > 0 && headless == SUPPORTED && submit_prompt == SUPPORTED`.
- OpenCode: `ReadOnlyReview=UNSUPPORTED` até existir Agent Projection/permission enforcement comprovado; não usa `--permission-mode plan`.
- Codex: `ReviewMode: []string{"--sandbox", "read-only", "--ask-for-approval", "never"}`.
- Claude: `ReviewMode: []string{"--permission-mode", "plan", "--output-format", "text"}`.

## 10. Integração (Opt-in)

- `IntegrationService.Integrate(ctx, req)` — fast-forward ff-only.
- Habilitado via `ConductorPolicy.IntegrationOptIn = true`.
- Desabilitado por padrão; retorna erro se chamado sem opt-in.
- Grava `IntegrationReceipt` em event metadata.
- Nunca faz push, deploy, tag ou checkout automático.

## 11. Observabilidade

### Instrumentação

- `executeAgentPrompt` grava `AgentInvocation` (RUNNING antes, COMPLETED/FAILED depois).
- Startup executa `PurgeExpiredEvents` com retenção da política (default 90d).

### Event Metadata

| Kind | Conteúdo |
|---|---|
| `agent_invocation` | JSON de `AgentInvocation` |
| `integration_receipt` | JSON de `IntegrationReceipt` |

## 12. Arquivos

### Criados

| Arquivo | Descrição |
|---|---|
| `internal/nexus/store/migrations/0019_conductor_policy.sql` | Migração |
| `internal/nexus/store/settings.go` | GetSetting/PutSetting |
| `internal/nexus/store/events_purge.go` | Purge de eventos |
| `internal/nexus/store/conductor_store_test.go` | Testes do store |
| `internal/nexus/conductor.go` | Tipos core |
| `internal/nexus/conductor_dispatch.go` | Dispatch entrypoint |
| `internal/nexus/conductor_store.go` | RecordAgentInvocation, PurgeExpiredEvents |
| `internal/nexus/conductor_test.go` | Testes do conductor |
| `internal/nexus/integration.go` | IntegrationService |
| `internal/nexus/integration_test.go` | Testes de integração |
| `internal/control/web/handlers_conductor.go` | Handlers HTTP |

### Modificados

| Arquivo | Mudança |
|---|---|
| `internal/nexus/store/models.go` | `IdentityKind` no `Agent` |
| `internal/nexus/store/agents.go` | `agentColumns`, `scanAgentRows`, `ListAgentsFiltered`, `ReconcileEphemeralWorkers` |
| `internal/nexus/mission_service.go` | `Goal *GoalContract` no `missionExecutionSnapshot` |
| `internal/nexus/mission_execution.go` | Instrumentação `AgentInvocation` |
| `internal/nexus/nexus.go` | Startup purge, `purgeExpiredEvents` |
| `internal/nexus/runner/types.go` | `ExecutionIdentity`, `AllocationResult.Kind` |
| `internal/control/driver/driver.go` | `ReadOnlyReview` capability |
| `internal/control/driver/opencode_driver.go` | Popula `ReadOnlyReview` |
| `internal/core/provider/spec/opencode.go` | `ReviewMode` setado |
| `internal/control/web/routes.go` | Rotas conductor |
| `internal/app/interactive_delegation.go` | Usa `StartDispatch` antes de consumir o prompt e `WaitDispatch` após admissão |
| `internal/control/web/promotion.go` | Mesma fronteira de admission/fallback no terminal Web |
| `internal/app/nexus_cli_cmds.go` | `conductorCmd` |
| `internal/app/app.go` | Registrado `conductor` |

## 13. Verificação

```bash
go build ./...                              # OK
go vet ./...                                # OK
go test -race ./internal/nexus/...          # 502 passed
make lint-go test-go                        # OK
make security                               # No vulnerabilities
```

Erros frontend preexistentes (`PlanBuilderSurface.tsx`, `FlowPipeline.tsx`) não são desta feature.
