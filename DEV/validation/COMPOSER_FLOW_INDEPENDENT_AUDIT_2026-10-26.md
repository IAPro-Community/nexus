# Auditoria independente — Nexus Composer + Flow v1.0.0

**Data:** 2026-10-26  
**Branch revisada:** `release/v1.0.0`  
**Veredito:** **NO_GO** — integração identificada no código, sem prova de compilação, testes completos ou homologação multiplataforma.  
**Tipo de verificação:** leitura direta dos arquivos da branch via GitHub; o ambiente não permitiu clonar o repositório pela rede.

## Cobertura de código observada (inspeção estática)

| Área | Evidência | Resultado |
| --- | --- | --- |
| Composer: remoção da elaboração ativa | `ComposerWorkspaceHeader.tsx` → `useComposerSession.ts` → API | Implementado, não executado |
| Composer: remoção do histórico salvo | `ComposerStart.tsx` | Implementado, não executado |
| Composer sem IA: remoção | `ComposerReadOnlyHistory.tsx` | Implementado, não executado |
| Composer: CAS por revisão | `handlers_planning.go`, `store/composer.go` | Implementado, não executado |
| Composer: proteção de entregas e fontes | `store/composer.go` | Inclui receipts, source IDs e handoff importado |
| Composer: cascata apenas de dados próprios | SQLite FK schema e `DeleteComposerSessionExpected` | Implementado, testar DB |
| Flow: remoção da galeria | `FlowRunsHistorySurface.tsx` | Implementado, não executado |
| Flow: remoção do editor | `FlowGoalBar.tsx`, `useFlowDraftEditor.ts` | Implementado, não executado |
| Flow: proteção Mission / snapshot / schedules | `store/plans.go` | Implementado, testar DB |
| Flow: contratos DELETE antigos | `handlers_planning.go` | Corrigido: `expected_revision` opcional para cliente legado |
| Erros ao consultar histórico | `FlowRunsHistorySurface.tsx` | Corrigido: não confundir indisponibilidade com zero execuções |
| Composer→Flow imutabilidade | `plan.go`, `plan_application.go` | Integrado, testes ainda não executados |
| Flow→Mission preflight + GO | `flow_decomposition.go`, `mission_service.go` | Contratos de bloqueio presentes, testes ainda não executados |
| Replanejamento | `handleRunPlanningReview` e UI | Nova sessão separada, não reescreve run ativo |

## Problemas detectados e corrigidos nesta auditoria

1. **Quebra de retrocompatibilidade da API:** DELETE de WorkPlan exigia `expected_revision`, apesar de o endpoint antigo aceitar remoções sem essa query. Agora o parâmetro é opcional apenas para o contrato legado. O cliente novo continua fornecendo revisão; ambos passam pelo mesmo guard de evidências.
2. **Histórico incorreto em falha de rede:** a consulta de Missions engolia erros e os convertia em lista vazia. Agora a indisponibilidade é explicitada, sem declarar falsamente que não há execuções.
3. **Teste incompleto de snapshot:** o teste tinha nome de snapshot mas criava apenas Mission. Foram adicionados snapshots reais e cenários em que o snapshot existe sem Mission.
4. **Linhagem importada entre projetos:** a proteção de exclusão verificava IDs achatados, mas não um handoff v1 importado sozinho. Agora consulta ambos, incluindo referências entre projetos.
5. **Lacuna de teste de entrega externa:** adicionado cenário de `composer_destination_receipts`.

## Riscos e pendências (não resolvidos)

- **BLOQUEANTE:** nenhum resultado confiável de `go test`, `bun run typecheck`, `bun run test`, `bun run build` ou E2E nesta auditoria. Código commitado != release homologada.
- **BLOQUEANTE:** sem certificação GUI por plataforma (Linux / Windows / macOS), navegação por teclado e regressões nos terminais.
- **IMPORTANTE:** não foi feita limpeza geral de formatos (`gofmt`, Prettier, ESLint, Stylelint), nem houve prova de compatibilidade por instalação.
- **IMPORTANTE:** exclusão de Flows com histórico é intencionalmente impedida. Para ocultar trabalhos concluídos sem perda de evidência, implementar **Arquivar** em projeto separado antes de prometer remoção universal.
- **IMPORTANTE:** confirmar fluxo end-to-end de novos planejamentos vagos, mudanças de escopo, inteligência configurada/desconfigurada, aprovação e handoff imutável com um provedor real.
- **IMPORTANTE:** cobertura E2E de clique nos dialogs (cancelar, confirmar, 409, reload, persistência após restart) ainda não adicionada; testes unitários novos não substituem GUI.
- **CONHECIDO:** gravações locais de anexos são limpas apenas best-effort após transação concluída, sem erro fatal caso o filesystem esteja indisponível.

## Comandos necessários para o gate

```bash
git fetch origin
git switch release/v1.0.0
git pull --ff-only
git status --short
git rev-parse HEAD
go test ./internal/nexus/store/... ./internal/nexus/... ./internal/control/web/...
cd web
bun install --frozen-lockfile
bun run typecheck
bun run lint
bun run lint:styles
bun run format:check
bun run test
bun run build
bun run test:e2e:composer
bun run test:e2e:composer-spine
bun run test:e2e:composer-vague-spine
```

Depois da suíte, fazer testes manuais no app: criar/deletar Composer sem vínculos, rejeitar Composer com Flow, apagar Flow não executado, rejeitar Flow com Mission/snapshot/schedule, navegar com teclado, repetir com IA desconfigurada, abrir em duas abas para exercitar HTTP 409, reiniciar app e confirmar que a lista persiste.

**Gate:** a versão 1.0.0 não deve ser chamada de validada só porque possui handlers, botões e testes presentes na árvore. Evidência é a execução efetiva desses testes e a validação visual das plataformas.
