# Goal Loop — primeiro contrato seguro (read-only)

**Status: etapa inicial de #167. Não executa correções, não está homologado.**
Base técnica: `internal/nexus/goal_loop_preview.go`, `GoalAssessment`, `runner.MissionRun` e `runner.AutonomyContract`.

## Por que não foi criado outro executor

O Nexus já dispõe de um `MissionRunner` com `RetryController`, `WorkReceipt`, `PackageRun`, `HumanIntervention`, `CoordinationRequest`, despacho idempotente e contratos de autonomia. Um segundo scheduler ou uma nova memória semântica criaria resultados divergentes.

A função `PreviewGoalLoop(run, assessment, failedPackageID, fingerprint)` é pura. Ela não escreve em SQLite, não atualiza um run, não reinicia Agent, não registra coordination request e não altera o snapshot aprovado.

## Contrato do diagnóstico

- `PASS`: somente o `MissionRunner` com estado `COMPLETED_VERIFIED` e todas as WorkPackages `VERIFIED` com recibo de conclusão factual. O texto de uma IA não comprova PASS.
- `FAIL/REDO`: exige o identificador exato da atividade que falhou, fingerprint persistível e violações explícitas. Sem vínculo confiável, retorno `BLOCKED_NEEDS_USER`.
- Análise de impacto: apenas a atividade de origem e suas dependentes transitivas são indicadas como afetadas. WorkPackages independentes, verificadas e com recibo, permanecem preservadas. Não há replay implícito.
- Bloqueios: despacho com `INTENT`/resultado externo incerto, `ReplayProhibited`, falha repetida sem progresso, `MaxRetries`, `MaxTotalIterations`, `MaxNoProgress`, `AutoRemediate=false`, cancelamento e intervenção humana já pendente.
- `REPAIR_CANDIDATE` significa apenas que há uma sugestão de correção com escopo definido. **Nunca equivale à autorização para executá-la**.

## Etapas pendentes antes de chamar de Goal Loop completo

1. Integrar preview aos registros reais de verificação/assessment do `MissionRunner`, sem inferir `failed_package_id` de texto livre.
2. Persistir ledger de tentativas (ID, fingerprint, snapshot, custos reais, evidências) no repositório de runs existente e com idempotência em restart, não em segunda store.
3. Projetar uma `CoordinationRequest` de reparação somente após avaliação de permissão, orçamento, integração, necessidade de nova revisão e dedupe; fazer `MaterializeCoordinationWork` sob admission.
4. Revalidar as atividades descendentes após reparação sem perder receipts independentes. Não mutar WorkPlan congelado; nova revisão somente com aprovação.
5. Limitar custo total, tempo e repetições e detectar regressões de integração; mostrar os bloqueios no Flow Run e Centro de Atenção.
6. Homologar em dois clientes, dois Agents, crash injection (antes/depois de dispatch e receipt), cancel, leases/RBAC, Linux e Windows.

## Gate local (a executar, nenhum resultado presumido)

```bash
go version
gofmt -l internal/nexus/goal_loop_preview*.go
go test -count=1 ./internal/nexus -run '^TestGoalLoopPreview'
go test -race -count=1 ./internal/nexus -run '^TestGoalLoopPreview'
go vet ./internal/nexus
```

O PR permanece draft/NO_GO até Go 1.25, formatação, race e aprovação da arquitetura de coordenação. Não usar `main`. **A memória semântica pertence exclusivamente ao Maestro.**
