# Evidence ledger da documentação

Este ledger liga afirmações públicas relevantes a implementação e testes
revisáveis. Ele existe para impedir documentação inventada.

> Teste de contrato comprova comportamento do source; não comprova sozinho
> disponibilidade de conta/provider nem certificação nativa de plataforma.

| Afirmação pública | Evidência primária | Testes / evidência operacional | Página pública |
| --- | --- | --- | --- |
| Web, Desktop e CLI consomem um Core comum | `cmd/nexus/main.go`, `cmd/nexus-desktop/main.go`, `internal/nexus/` | desktop/routes tests | [Visão geral](../product/overview.md) |
| Direct não exige Composer/Flow/Mission | `web/src/features/work/directSessionModel.ts` | `directSessionModel.test.ts` | [Direct](../product/direct.md) |
| Agent é separado de runtime/provider | agent/runtime application boundaries | agent/runtime tests | [Agents](../product/agents.md) |
| AgentMatcher escolhe WHO e Scheduler escolhe recurso | `internal/nexus/agent_matching.go`, resource/routing code | matching/routing tests | [Modelo mental](../product/mental-model.md) |
| Quota/capacity possui qualidade explícita | `internal/profile/usage.go`, quota monitor | usage/quota tests | [Providers](../product/providers-and-usage.md) |
| Composer produz planejamento estruturado | composer application/compiler | composer tests | [Composer](../product/composer.md) |
| WorkPlan é o plano canônico e Flow sua projeção | plan/flow code e revision contracts | plan/flow round-trip tests | [Flow](../product/flow.md) |
| GO associa execução a revisão aprovada | execution snapshot / mission start path | preview/mission tests | [Como funciona](../product/how-it-works.md) |
| MissionRunner é máquina de estado durável | `internal/nexus/runner/runner.go` | runner/restart/race tests | [Runtime da Mission](../architecture/mission-runtime.md) |
| ContextCapsule é handoff limitado | `internal/nexus/runner/evidence.go` | `evidence_test.go` | [Missions](../product/missions.md) |
| WorkReceipt registra resultado/evidência | `internal/nexus/runner/evidence.go` | `evidence_test.go` | [Missions](../product/missions.md) |
| DoD global impede false DONE | runner global verification | `global_verification_test.go` | [Runtime da Mission](../architecture/mission-runtime.md) |
| Coordenação possui operações tipadas e estado durável | `internal/nexus/runner/coordination.go` | `coordination_test.go` | [Coordenação](../product/agent-coordination.md) |
| Replay da mesma request é idempotente | coordination request IDs/materialization | coordination persistence/replay tests | [Coordenação](../product/agent-coordination.md) |
| Miss de contexto pode gerar child PackageRun | `materializeCoordinationWorkInRun` | coordination execution tests | [Coordenação](../product/agent-coordination.md) |
| NEED_INFO consulta memória antes de worker | mission executor local resolver | mission coordination execution tests | [Coordenação](../product/agent-coordination.md) |
| ARTIFACT_REQUEST pode reutilizar receipt/artifact | mission executor local resolver | coordination tests | [Coordenação](../product/agent-coordination.md) |
| Resposta da child volta por WorkReceipt/ContextCapsule | runner coordination + evidence | `coordination_execution_test.go` | [Coordenação](../product/agent-coordination.md) |
| Retomada preserva Agent identity | ResumeAgentID/allocation path | coordination execution lifecycle test | [Coordenação](../product/agent-coordination.md) |
| Replay inseguro falha fechado | `coordinationReplaySafety`, runner intervention | mission/runner coordination safety tests | [Guardrails](../architecture/guardrails.md) |
| Fingerprint inclui Git HEAD e workspace state | `workspaceFingerprint` | mission execution tests | [Guardrails](../architecture/guardrails.md) |
| Fan-out/payload de coordenação é limitado | coordination validation ceilings | coordination bound tests | [Coordenação](../product/agent-coordination.md) |
| Required capabilities chegam ao Scheduler | dynamic package TaskRequirements | scheduler capability coordination test | [Coordenação](../product/agent-coordination.md) |
| Preferred provider continua preferência, não PIN | coordination materialization resource policy | preferred-provider test | [Coordenação](../product/agent-coordination.md) |
| Maestro é opcional e não substitui runtime | Maestro boundary + capability integration | integration tests/status handling | [Maestro](../product/maestro.md) |
| API local usa autenticação/bootstrap/middleware | `internal/control/web/` | server/routes tests | [API local](../api/README.md) |
| Screenshots usam fixture sintética | docs capture script + manifesto | docs verification | [Tour visual](../product/visual-tour.md) |
| Diagramas v1 são conceituais e versionados | `docs/assets/diagrams/` | revisão documental | [Assets](../assets/README.md) |
| Update/release possuem verificação de integridade | update/release/installers | release/update negative tests | [Release](../operations/release.md) |

## Convenções

- **Disponível**: implementação e contrato verificáveis presentes.
- **Implementado no source candidato**: código existe, mas a release estável
  ainda depende de certificação.
- **Experimental/Beta**: escopo operacional/compatibilidade ainda restrito.
- **Interna**: interface sem compromisso público de estabilidade.
- **UNKNOWN**: ausência de evidência suficiente.

## Precedência

Em caso de divergência documental:

1. comportamento atual de código/teste;
2. documentação pública canônica;
3. status corrente de release;
4. specs ativas;
5. relatórios históricos.

O status de release continua em
[status público da release](../release-status.md).
