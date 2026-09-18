# Red-team independente — Autopilot, Missions e Multi-agent

## Escopo

- Repositório: `ai-manager`
- HEAD auditado: `e53f8352e4c3647632ebac7877cf0a6a3bd62647`
- Critério: provar o fluxo de produto público `intent → plan → execution → verification → remediation → Definition of Done`, não aceitar testes unitários/sandboxes como certificação.
- Nenhum código foi alterado.

## Veredito executivo

| Capacidade | Veredito | Razão determinante |
| --- | --- | --- |
| Autopilot | **FAIL** | O CLI público abandona a execução no primeiro erro remediável e usa o caminho não autônomo; a UI não fecha o ciclo de intervenção nem apresenta a evidência canônica de validação. |
| Mission | **FAIL** | A Mission pública é um CRUD separado do `MissionRunner` e aceita `COMPLETED` por PATCH sem execução, verificação ou evidência. |
| Multi-agent | **FAIL** | Agentes autônomos escrevem em worktrees distintos sem etapa de integração; o DoD global roda no checkout canônico, que não contém essas mudanças, e pode certificar apenas o baseline. |

**Resultado da prova:** rejeitado. O HEAD contém componentes internos importantes, mas não entrega o fluxo solicitado como produto verificável e livre de falso sucesso.

## Evidência de produto disponível

Há um caminho web real para criar plano, iniciar `MissionRun`, executar providers reais, testar, revisar, remediar e aplicar um gate global:

- Web inicia execução autônoma por `POST /api/v1/plans/{id}/run` (`internal/control/web/handlers_planning.go:472-529`).
- `MissionRunner` só promove um pacote após comandos de verificação e revisão independente (`internal/nexus/runner/runner.go:326-392`).
- O estado final nominal é `COMPLETED_VERIFIED`, após `verifyGlobalDefinition` (`internal/nexus/runner/runner.go:522-590`).
- Resultados são gravados em stream append-only, com `PASS`, `FAIL` ou `NOT_VERIFIED` (`internal/nexus/validation_evidence.go:92-171`).

Isso prova estrutura e alcançabilidade estática, não o fluxo como produto. Os próprios artefatos do repositório dizem que execução Mission real com provider/worktree não foi testada (`DEV/validation/FINAL_MISSION_AUTONOMY_REPORT.md:7-9`) e que não existe run autenticado com stream durável para este candidato (`DEV/validation/production-finalization/FINAL_REPORT.md:119-123`).

## Findings

### AM-001 — P0 — DoD global verifica o checkout errado

Uma execução autônoma exige worktree e aloca cada Agent em seu worktree estável (`internal/nexus/worktree.go:28-32`, `internal/nexus/worktree.go:56-101`). Porém:

1. `MissionRun.Workspace` recebe `project.CanonicalPath` na criação (`internal/nexus/mission_service.go:181-187`).
2. Cada pacote executa e testa em `pkg.Workspace` (`internal/nexus/runner/runner.go:326-340`).
3. O DoD global executa em `run.Workspace`, não nos worktrees/result revisions (`internal/nexus/runner/runner.go:541-565`).
4. Não há operação de merge, cherry-pick ou montagem de workspace integrado no runner.

Consequência: mudanças de Agent podem passar nos testes de pacote em branches isoladas, enquanto o DoD global testa somente o checkout canônico inalterado. Se o baseline já estiver verde, a Mission pode terminar `COMPLETED_VERIFIED` sem testar o produto produzido.

### AM-002 — P0 — Multi-agent não integra entregas

O handoff entre dependências leva apenas `WorkReceipt` textual e metadados (`internal/nexus/runner/evidence.go:264-303`, `internal/nexus/runner/evidence.go:432-445`). O próximo Agent não recebe a árvore de arquivos nem o commit da dependência em seu worktree. Para grupos paralelos, Agents distintos trabalham em branches distintas; não existe integrador que materialize os resultados em uma árvore comum.

Consequência: a topologia multi-agent coordena estados, mas não entrega necessariamente um artefato de software integrado. `ChangedFiles` e `ResultRevision` são recibos, não aplicação das mudanças.

### AM-003 — P0 — Mission pública pode declarar sucesso sem verificação

`/api/v1/missions/{id}` é uma superfície pública diferente de `MissionRun`. O PATCH copia `body.Status` diretamente para `Mission.Status` (`internal/control/web/handlers_missions.go:91-112`; `internal/nexus/mission_application.go:99-129`). O store persiste o status sem transição, DoD ou evidência (`internal/nexus/store/missions.go:57-62`).

As tasks seguem o mesmo modelo CRUD e suas estatísticas contam `COMPLETED` por valor persistido (`internal/nexus/store/missions.go:229-238`). Não há ligação de execução entre `Mission`, `MissionTask`, assignments e `MissionRunner`; `MissionApplicationService` declara explicitamente que só possui o lado de planejamento (`internal/nexus/mission_application.go:10-12`).

Consequência: o produto mostra Missions concluídas sem prova de execução ou verificação.

### AM-004 — P1 — CLI público interrompe a remediação

`nexus run "<goal>"` cria Flow/WorkPlan e usa `StartMissionRun(..., autonomous=false)` (`internal/app/app.go:1572-1619`). O loop chama `ExecuteNextStep`, mas retorna imediatamente em qualquer `stepErr` (`internal/app/app.go:1621-1629`).

Isso conflita com o contrato do runner: erros remediáveis são persistidos como `REMEDIATING` e `RunToTerminal` deve continuar enquanto houver orçamento (`internal/nexus/runner/runner.go:799-825`). `nexus plan run` repete o mesmo erro de controle (`internal/app/nexus_cli_cmds.go:100-124`).

Consequência: a remediação existe internamente, mas não funciona ponta a ponta na principal superfície CLI.

### AM-005 — P1 — UI de Run não fecha intervenção `BLOCKED_NEEDS_USER`

`FlowRunSurface` trata `BLOCKED_NEEDS_USER` como “resumable”, mas chama `returnToMission` (`web/src/features/work/FlowRunSurface.tsx:154-155`, `web/src/features/work/FlowRunSurface.tsx:333-347`). Esse endpoint termina em `ResumeRun`, que só aceita `PAUSED` (`internal/nexus/runner/runner.go:611-626`). A resolução correta exige uma opção versionada em `/resolve-intervention` (`internal/control/web/handlers_planning.go:578-606`).

Consequência: no ponto em que remediação humana é necessária, a tela principal oferece uma ação que falha, em vez das opções duráveis do checkpoint.

### AM-006 — P1 — Evidência canônica não é consumida pela UI de Run

O cliente possui `getRunValidationEvidence` (`web/src/nexus/api.ts:526-529`), mas nenhuma tela TSX o chama. `FlowRunSurface` carrega apenas capsules/receipts (`web/src/features/work/FlowRunSurface.tsx:73-83`) e apresenta `COMPLETED` a partir do estado da run.

Além disso, a entrada de validação usa a identidade Git de `project.CanonicalPath` (`internal/nexus/validation_evidence.go:103-114`), embora o comando de pacote tenha rodado em `pkg.Workspace`. Isso reforça AM-001: o SHA exibido na stream pode não identificar os bytes verificados no worktree do Agent.

Consequência: a API canônica evita inferir PASS quando a stream não existe (`internal/nexus/validation_evidence.go:36-89`), mas o produto não mostra esse estado nem correlaciona corretamente a identidade do trabalho isolado.

### AM-007 — P1 — É possível concluir sem DoD global

`verifyGlobalDefinition` chama `completeRun` quando `GlobalVerificationCommands` está vazio (`internal/nexus/runner/runner.go:541-547`). A admissão strict exige verificação por step, porém não exige um comando global (`internal/nexus/flow_decomposition.go:413-435`). A normalização só copia comandos detectados por convenções de `go.mod`, `package.json`, Python ou Cargo (`internal/nexus/mission_service.go:281-321`).

Consequência: projetos com gates válidos não detectados, ou planos manuais com verificação por step, podem terminar sem uma verificação integrada real.

### AM-008 — P2 — “Overnight” é sandbox, não certificação

`TestOvernightAcceptanceSandbox` declara que usa executor determinístico (`internal/nexus/runner/runner_durable_test.go:636-640`), `fakeExecutor`, comandos triviais e workspace temporário (`internal/nexus/runner/runner_durable_test.go:641-654`). `TestOvernightSoakHarnessBounded` dura 15 segundos por padrão, usa repositório em memória e comandos `true` (`internal/nexus/runner/fault_injection_test.go:103-123`).

Esses testes são úteis para a máquina de estados, mas não provam provider real, CLI/web, worktrees integrados, restart de processo ou soak noturno. Não foram usados como PASS nesta revisão.

## Checagem específica de `ValidationEvidence`

Não encontrei a regressão “stream ausente inferida como PASS” no HEAD:

- ausência retorna `entries=[]` e `chain_verified=false`;
- entradas sem resultado ou sem SHA viram `NOT_VERIFIED`;
- falha de comando vira `FAIL`;
- falha ao persistir evidence derruba a run.

Portanto, essa hipótese específica foi **rejeitada no serviço**. O risco remanescente é de identidade/workspace incorreto e de não consumo pela UI, não de fabricação direta de `PASS` nesse projector.

## Estado da campanha e dos relatórios

- O relatório overnight não pertence ao HEAD auditado; o arquivo presente no working tree é untracked e registra `ABORTED_PRECONDITION_FAILED`, zero Missions e campanhas A–F não iniciadas.
- Ao contrário da premissa inicial, `DEV/validation/production-finalization/FINAL_REPORT.md` **existe no HEAD `e53f835...`**. Seu veredito é `NOT_READY_FOR_OVERNIGHT_CERTIFICATION`, portanto continua sendo bloqueador.
- O relatório overnight untracked afirma que esse arquivo não existia quando abortou. Essa afirmação é histórica/stale em relação ao HEAD auditado e não pode ser usada como fotografia atual.

## Definition of Done desta revisão

Para promover qualquer uma das três capacidades acima, é necessário demonstrar por CLI e web, contra o mesmo SHA:

1. intenção e plano aprovável;
2. execução autenticada em provider real;
3. integração material dos worktrees multi-agent;
4. verificação de pacote e DoD sobre a árvore integrada exata;
5. falha induzida seguida de remediação automática e humana pelo produto;
6. stream canônica exibida na UI, com SHA/identity da árvore efetivamente testada;
7. reinício de processo e retomada sem redispatch;
8. estado final sem possibilidade de PATCH administrativo fabricar conclusão.

O candidato não satisfaz esses requisitos. **GO/NO-GO: NO-GO.**
