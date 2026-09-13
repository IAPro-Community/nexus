# Inventário funcional

> Fonte de verdade da documentação pública. Inventário produzido em 2026-09-13
> a partir de implementação, rotas, help da CLI, testes e documentação
> canônica. “Experimental” significa que o código/contrato existe, mas a
> evidência operacional completa não deve ser inferida.

| Funcionalidade | Problema resolvido | Tipo / status | Implementação e acesso | Dependências / limites | Testes e evidência | Documento |
| --- | --- | --- | --- | --- | --- | --- |
| Projetos e workspaces | mantém contexto de repositórios locais | Core, Web, CLI — Disponível | `internal/nexus/project_application.go`; `/api/v1/projects`; `nexus projects` | filesystem local acessível | `project_application_test.go`; rotas Web | [Projects](../product/overview.md) |
| Direct / AI Session | abre uma conversa sem plano obrigatório | Web, CLI — Disponível | `web/src/features/work/directSessionModel.ts`; `nexus <provider>` | provider instalado e perfil autenticado | testes de modelo e help CLI | [Direct](../product/direct.md) |
| Agents persistentes | separa identidade operacional de runtime | Core, Web, CLI — Disponível | `internal/nexus/agent_application.go`; `/api/v1/agents`; `nexus agents` | projeto registrado | `agent_application_test.go`, `agent_spec_test.go` | [Agentes](../product/agents.md) |
| Runtimes supervisionados | observa, conecta e recupera sessões | Core, Web, CLI — Disponível | `internal/control/launcher`; `/api/v1/runtimes`; `start/stop/ps/attach` | suporte de shell/PTY da plataforma | `launcher_test.go`, rotas | [Terminais](../product/terminals.md) |
| Providers e perfis | escolhe e isola CLIs oficiais | Core, Web, CLI — Disponível com evidência por provider | `internal/core/provider`, `internal/profile`; `/providers`, `/profiles` | instalação e login externos | adapters e `providers status --json` | [Providers e quota](../product/providers-and-usage.md) |
| Uso / quota | torna capacidade observável sem inventar dados | Core, Web, CLI — Disponível com estados de evidência | `internal/nexus/quota_monitor.go`; `nexus usage --json` | provider pode retornar `UNKNOWN`/`CACHED` | `quota_monitor_test.go`, adapter usage tests | [Providers e quota](../product/providers-and-usage.md) |
| Composer | transforma intenção em brief/prompt | Core, Web — Disponível | `internal/nexus/composer*.go`; `/composer-sessions` | inteligência configurada pode ser necessária | `composer*_test.go` | [Composer](../product/composer.md) |
| WorkPlan e Flow | modela etapas, dependências e revisão | Core, Web, CLI — Disponível | `internal/nexus/plan*.go`, `flow*.go`; `/plans`; `nexus plan` | plano válido | `flow_test.go`, `plan_application_test.go` | [Flow](../product/flow.md) |
| Mission Runner | executa plano com gates e evidência | Core, Web, CLI — Experimental | `internal/nexus/runner`; `/runs`, `/missions`; `nexus run` | contratos/autonomia/recursos; não é certificação de provider | runner durable/evidence/global verification tests | [Missions](../product/missions.md) |
| Delegação automática | usa especialistas apenas quando o benefício supera overhead | Core, sessão supervisionada — Experimental | `internal/nexus/delegation.go`, `interactive_lead.go`, `internal/app/interactive_delegation.go` | depth limitado, ownership e matching; provider real depende do ambiente | `delegation_test.go`, `interactive_lead_test.go` | [Funcionalidades](../features/index.md) |
| Maestro | adiciona método, skills e gates ao Nexus | Core, Web, CLI — Opcional / Experimental | `internal/nexus/maestro.go`; `/maestro`; `nexus maestro` | instalação externa pode não existir | `maestro_test.go`, `maestrogates` tests | [Maestro](../product/maestro.md) |
| API local | serve Web/Desktop de forma autenticada | API — Interna | `internal/control/web/routes.go` | não é contrato público; loopback/sessão/CSRF | `server_routes_test.go`, handler tests | [API local](../api/README.md) |
| Atualização assinada | verifica update e integridade de artefato | Core, CLI, Web — Disponível com requisitos de release | `internal/update`; `nexus update` | depende de manifest/assinatura/release | `manifest_test.go`, negative tests | [Atualizações](../architecture/updates.md) |
| Tunnel | oferece acesso remoto controlado | Core, Web, CLI — Disponível; sensível à segurança | `/tunnel/*`; `nexus tunnel` | revisão de rede e URL compartilhada | handlers e controles de origem | [Operação](../operations/README.md) |
| Desktop Wails | disponibiliza shell nativa | Desktop — Beta / evidência por plataforma | `cmd/nexus-desktop`, `internal/desktop` | build e dependências variam por OS | `desktop_test.go`; matriz de plataforma | [Desktop](../desktop/overview.md) |

## Não classificados como funcionalidade pública

- `DEV/`, `docs/superpowers/`, `TODO/` e relatórios de validação são memória,
  planejamento ou evidência de engenharia; não são promessas de produto.
- A API local é documentada como **Interna** para evitar que endpoints de UI
  sejam confundidos com SDK público.
- Recursos descritos em `docs/future/` permanecem **Planejados** apenas quando
  a própria documentação registra essa intenção explicitamente.
