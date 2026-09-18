# Red team independente — Providers / quota

- HEAD auditado: `e53f8352e4c3647632ebac7877cf0a6a3bd62647`
- Escopo: adapters Codex, AGY, Claude, Gemini e OpenCode; `Trustworthy()`; descoberta e roteamento de recursos; `RATE_LIMITED`, `UNAVAILABLE` e failover.
- Integridade da revisão: os caminhos de implementação auditados foram comparados com `HEAD` e não têm diferenças locais.
- Teste focado: `go test ./internal/core/quota ./internal/core/scheduler ./internal/core/fallback ./internal/core/provider/adapters/codex ./internal/core/provider/adapters/agy ./internal/nexus -count=1` passou (`374` testes em `6` pacotes). Isso comprova apenas os cenários codificados; não elimina as falhas de contrato abaixo.

## Veredictos

| Área | Veredicto | Motivo decisivo |
| --- | --- | --- |
| Provider routing | **REPROVADO** | quota `UNKNOWN` permanece elegível e pode ser selecionada por LRU; `UNAVAILABLE` não é um hard gate uniforme no recomendador de Mission. |
| Quota | **REPROVADO** | métricas ausentes podem virar `0%`; três adapters anunciam usage sem leitura live e aceitam `usage.json` declarativo; `Trustworthy()` não valida a medição de cada janela. |
| Failover | **REPROVADO** | a Mission perde a classificação tipada do provider, infere quota por substring e publica `QuotaFailoverCompleted` antes de qualquer execução bem-sucedida no destino. |

## Achados bloqueadores

### PQ-01 — `UNKNOWN` é tratado como recurso disponível e sucesso de admissão

`QuotaView.ComputeAvailability()` inicializa `Available=true` e, para `UNKNOWN`, apenas marca `UnknownQuota`; não fecha a disponibilidade (`internal/core/quota/view.go:295-320`). Quando não há janelas, `BuildQuotaView()` cria uma janela sintética `unknown` e chama essa mesma lógica (`internal/core/quota/view_builder.go:23-33`).

O estado chega sem correção ao roteamento:

- `ListResourcesContext()` copia `qv.IsAvailable()` para `ProviderAccount.Available` (`internal/nexus/resource_discovery.go:78-103`);
- `evaluateCandidate()` só bloqueia `!Available`, rate limit, cooldown, falta de autenticação e capabilities; `UNKNOWN` recebe inclusive `+10` de score (`internal/nexus/resource_recommendation.go:195-264`);
- quando todos os candidatos elegíveis têm quota desconhecida, `RecommendResources()` seleciona um deles por LRU (`internal/nexus/resource_recommendation.go:91-126`);
- o selector CLI repete a política e retorna “quota UNKNOWN; LRU among healthy authenticated profiles” (`internal/core/scheduler/scheduler.go:119-140`).

Logo, `UNKNOWN` não é apenas “não aferido”: ele autoriza alocação automática e execução. Isso viola o critério desta revisão de não converter `UNKNOWN` em zero, ilimitado ou sucesso.

### PQ-02 — ausência de percentual pode ser convertida em quota medida de `0%`

O modelo distingue corretamente `nil` (desconhecido) de zero medido (`internal/core/model/types.go:100-115`), mas `BuildQuotaView()` destrói essa distinção: inicia `remaining := 0` e só o substitui quando `RemainingPercent != nil` (`internal/core/quota/view_builder.go:43-54`). Uma janela `LIVE`, `CACHED`, `ESTIMATED` ou `RATE_LIMITED` com percentual ausente passa a `Remaining=0` e pode ser marcada como esgotada por `ComputeAvailability()`.

`Engine.Trustworthy()` exige source, timestamp, status e ao menos uma janela, mas não exige que uma janela tenha qualquer medição (`RemainingPercent`, `UsedPercent`, valores absolutos ou reset atribuível), nem exige `AccountScope` verificável (`internal/core/quota/quota.go:44-70`). Os próprios testes persistem e aceitam snapshots com janela sem métricas (`internal/core/quota/scope_test.go:12-17`).

Os parsers legados também materializam zeros ausentes: campos numéricos não ponteiro começam em zero e são convertidos em janelas 5h/weekly sempre que algum outro campo legado satisfaz o detector (`internal/core/quota/quota.go:149-176,180-213`; `internal/core/provider/adapters/agy/agy.go:515-560`). Portanto, ausência parcial pode se tornar esgotamento inventado.

### PQ-03 — Claude, Gemini e OpenCode anunciam quota sem fonte live

Os três adapters declaram `Usage: true`, porém seus `GetUsage()` apenas desserializam `<profile>/home/usage.json` e devolvem o conteúdo sem validar origem, frescor, identidade ou completude:

- Claude: `internal/core/provider/adapters/claude/claude.go:28-39,200-223`;
- Gemini: `internal/core/provider/adapters/gemini/gemini.go:28-39,158-181`;
- OpenCode: `internal/core/provider/adapters/opencode/opencode.go:29-42,167-190`.

A camada `profile` aplica parte das validações depois, mas um arquivo local recente pode autodeclarar `Source=OFFICIAL_API`, `Status=LIVE` e janelas, omitir `Account`, passar por `Trustworthy()`, receber o scope atual e ser persistido (`internal/profile/usage.go:184-233,302-334`). Isso não prova que a observação veio do provider. A capability `Usage=true` é, para esses adapters, uma leitura de cache injetável, não quota autenticada.

### PQ-04 — failover de Mission não recebe falha tipada e pode fabricar conclusão

O classificador comum produz `FailureRateLimit`, `FailureQuota`, `FailureProvider` etc. (`internal/core/classifier/classifier.go:46-119`) e o failover do comando interativo usa esses tipos (`internal/core/fallback/fallback.go:72-112`). A execução de Mission segue outro caminho: `executeAgentPrompt()` recebe falha do processo como erro genérico — por exemplo, “provider process exited with failure” — e `nexusPackageExecutor.Execute()` apenas a repassa (`internal/nexus/mission_execution.go:295-314`; `internal/nexus/mission_executor.go:608-625`). O output capturado, onde normalmente está o 429/quota, não é classificado quando a execução retorna erro.

O runner decide re-alocação por busca textual em `err.Error()` (`internal/nexus/runner/runner.go:308-319,1135-1147`), e o executor repete a busca textual no `PackageRun.ErrorMessage` (`internal/nexus/mission_executor.go:101-103,501-509`). Consequências:

1. um 429 real presente apenas no output pode virar falha genérica e repetir o mesmo provider em vez de fazer failover;
2. qualquer erro não-quota que contenha palavras como `quota`, `credits` ou `exhausted` pode abrir o pool de fallback;
3. após escolher outra conta, o código publica `EventQuotaFailoverCompleted` durante `Allocate()`, antes de `SafeApply`, resolução do workspace, lançamento do runtime ou sucesso do provider destino (`internal/nexus/mission_executor.go:171-223`).

Esse evento comprova apenas uma nova seleção, não failover completado. É evidência de failover falso.

## RATE_LIMITED e UNAVAILABLE

- **RATE_LIMITED:** Codex transforma bloqueio reportado pelo app-server em `UsageRateLimited` (`internal/core/provider/adapters/codex/app_server_usage.go:163-167`), e quota/resource discovery o bloqueiam. Essa parte unitária está correta. Não há prova de que a Mission preserve essa classificação no caminho real descrito em PQ-04.
- **UNAVAILABLE:** `eligibleAccounts()` bloqueia `Available=false` e rate limit (`internal/nexus/runtime_routing.go:192-199`), mas `RecommendResources()` não usa `Health=unavailable` como hard gate; depende de `Available`. Como quota `UNKNOWN` produz `Available=true`, um registro autenticado com health indisponível pode permanecer elegível para Mission (`internal/nexus/resource_recommendation.go:195-223,268-280`). A validação manual tem um gate mais estrito (`internal/nexus/resource_discovery.go:204-217`), criando semânticas divergentes.

## Evidência live autenticada para este SHA

**NOT_VERIFIED.**

Não existe evidência de Mission autenticada, provider-backed e concluída para `e53f8352e4c3647632ebac7877cf0a6a3bd62647`. O relatório de finalização deste SHA declara explicitamente que authenticated Mission e live failover/handoff/escalation não foram verificados (`DEV/validation/production-finalization/FINAL_REPORT.md:119-133`). A campanha overnight deste SHA abortou antes de iniciar e executou zero Missions e zero campanha de provider failover/quota (`DEV/validation/overnight-production-certification/FINAL_REPORT.md:18-46,81-99`).

Artefatos históricos `UNVERIFIED`, probes de binário, status local de autenticação, unit tests e smoke direto de CLI não são promovidos a evidência de Mission.

## Decisão de red team

Os três gates permanecem fechados. Não promover este SHA para GO de Providers/quota enquanto `UNKNOWN` autorizar execução, percentuais ausentes puderem virar zero, adapters sem leitura live anunciarem quota verificável e a Mission não transportar falha tipada até um failover cuja conclusão seja registrada somente após sucesso real no destino.
