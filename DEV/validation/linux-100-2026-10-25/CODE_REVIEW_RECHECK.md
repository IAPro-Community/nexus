# Rechecagem dos achados da revisão de runtime

Data: 2026-10-26 UTC  
Branch/base: `release/v1.0.0@7a04f1b4cf5950b97b4ea1e156dda31ac513823b`  
Overlay de fontes: `f033fa678c1fdeb99373d2456347ac152638ef13f699e607ff5b9bdd12113f2b` (`logs/source-provenance-current-single-mascot-2026-10-25.txt`)

## Resultado

Os cinco achados P1/P2 do relatório independente foram rechecados no código corrente. As guardas e regressões já existem na árvore; esta rechecagem não alterou código de produção. Os testes direcionados passaram com `-race`, conforme `logs/p1-code-review-findings-recheck-2026-10-25.txt`.

## Comandos reproduzíveis

```bash
GOCACHE=/tmp/nexus-p1-review-gocache go test -count=1 -race ./internal/nexus/runner -run 'TestResolveInterventionDoesNotReplayCompletedUnsafeCoordination|TestResolveIntervention_ReplanReallocatesAndRecompilesBeforeRetry|TestResolveIntervention_DefersPackageUntilDependenciesAreVerified'
GOCACHE=/tmp/nexus-p1-review-gocache go test -count=1 -race ./internal/nexus -run TestMissionWorkerHandoffRetriesAfterOldWorkerExceedsTimeout
GOCACHE=/tmp/nexus-p1-review-gocache go test -count=1 -race ./internal/control/web -run 'TestResolveInterventionHTTPDecisionSurvivesServerRestartExactlyOnce|TestResolveRunInterventionRejectsMalformedAndNonPositiveVersion'
```

| Achado | Rechecagem no código atual | Evidência | Status |
| --- | --- | --- | --- |
| Replanejar trabalho com dispatch externo concluído podia repetir efeito | `applyInterventionOption` nega replanejamento quando `ReplayProhibited`, preservando a identidade do dispatch; a opção permitida é verificar sem replay. | `TestResolveInterventionDoesNotReplayCompletedUnsafeCoordination` | PASS |
| Retomada podia reutilizar agente, runtime e prompt obsoletos | Replan seguro limpa o estado de dispatch/runtime/prompt e volta para `ALLOCATING`, passando novamente por alocação e compilação antes de execução. Dependências não verificadas mantêm o pacote em `PENDING`. | `TestResolveIntervention_ReplanReallocatesAndRecompilesBeforeRetry`; `TestResolveIntervention_DefersPackageUntilDependenciesAreVerified` | PASS |
| Não havia prova pela rota HTTP autenticada, persistência e restart | O teste exercita a rota autenticada, encerra/reabre o serviço e confirma exatamente uma execução do efeito. | `TestResolveInterventionHTTPDecisionSurvivesServerRestartExactlyOnce` | PASS |
| Timeout de handoff podia deixar a Mission sem nova tentativa em memória | A substituição aguarda a saída do worker antigo e então tenta registrar uma nova geração, sem executar dois workers concorrentes. | `TestMissionWorkerHandoffRetriesAfterOldWorkerExceedsTimeout` | PASS |
| A rota aceitava versão ausente ou não positiva e podia ignorar JSON inválido | O handler rejeita body inválido/trailing JSON e exige `version > 0` antes de chamar o serviço. | `TestResolveRunInterventionRejectsMalformedAndNonPositiveVersion` | PASS |

## Limites

Esta rechecagem cobre os achados listados e as regressões direcionadas; não substitui a suíte Linux completa, o soak P9, um provider real nem a homologação do compositor Wayland. O resultado geral continua `NO_GO_LINUX` até esses gates serem concluídos.
