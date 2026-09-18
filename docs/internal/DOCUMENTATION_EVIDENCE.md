# Evidence ledger da documentação

Esta tabela liga as afirmações relevantes a fontes que podem ser revisadas. Ela
é um mecanismo contra documentação inventada: testes demonstram contratos de
código, não sucesso de uma conta/provider em todos os ambientes.

| Afirmação | Evidência primária | Evidência de teste / operação | Página pública |
| --- | --- | --- | --- |
| Nexus possui Web, Desktop e CLI sobre um Core | `cmd/nexus/main.go`, `cmd/nexus-desktop/main.go`, `internal/nexus/` | `internal/desktop/desktop_test.go`, `web/src/app/routes.test.ts` | [visão geral](../product/overview.md) |
| Direct não exige Composer ou Flow | `web/src/features/work/directSessionModel.ts`, `docs/product/direct.md` | `directSessionModel.test.ts` | [Quick Start](../quick-start.md) |
| Agent e runtime são conceitos separados | `internal/nexus/agent_application.go`, `internal/nexus/runtime_application.go` | `agent_application_test.go`, `runtime_application_test.go` | [Agentes](../product/agents.md) |
| Provider/perfil/quota têm estados explícitos | `internal/profile/usage.go`, `internal/nexus/quota_monitor.go` | `usage_test.go`, `quota_monitor_test.go` | [providers e quota](../product/providers-and-usage.md) |
| Composer, WorkPlan e Flow existem no Core | `internal/nexus/composer_application.go`, `plan_application.go`, `flow.go` | `composer_application_test.go`, `plan_application_test.go`, `flow_test.go` | [Composer](../product/composer.md), [Flow](../product/flow.md) |
| Mission mantém evidência e verificação global | `internal/nexus/runner/`, `internal/nexus/validation_evidence.go` | `global_verification_test.go`, `evidence_test.go` | [Missions](../product/missions.md) |
| Delegação do Lead é política separada de matching e routing | `internal/nexus/delegation.go`, `interactive_lead.go`, `agent_matching.go`, `runtime_routing.go` | `delegation_test.go`, `interactive_lead_test.go`, `agent_matching_test.go` | [funcionalidades](../features/index.md) |
| A API local usa sessão, bootstrap e middleware | `internal/control/web/routes.go`, `server.go` | `server_routes_test.go`, handlers | [API local](../api/README.md) |
| A CLI expõe comandos e aliases canônicos | `internal/app/`, `cmd/nexus/main.go` | `go run ./cmd/nexus --help` em 2026-09-13 | [CLI](../cli/README.md) |
| Capturas são sintéticas e automatizadas | `web/scripts/docs-capture.mjs`, `docs/assets/screenshots/manifest.json` | `scripts/docs-verify.mjs` | [tour visual](../product/visual-tour.md) |
| Atualizações verificam manifest/assinatura | `internal/update/`, `install.sh`, `install.ps1` | `internal/update/negative_test.go`, `internal/release/installer_test.go` | [atualizações](../architecture/updates.md) |

## Convenções

- **Disponível**: implementação e contrato/teste verificáveis estão presentes.
- **Experimental/Beta**: implementação presente, mas cobertura operacional,
  compatibilidade de plataforma ou integração externa permanece condicionada.
- **Interna**: interface destinada às superfícies do próprio Nexus, sem
  compromisso de estabilidade para terceiros.
