# Flow semantic planner — validação da release v1.0.0

**Estado: NÃO HOMOLOGADO / PR em draft.** Este roteiro não é um relatório de testes executados.

## Contrato

- O Composer conclui um `PromptArtifact`; não cria ou executa automaticamente uma Mission.
- O Flow aceita um prompt completo independente do Composer, em `POST /api/v1/flow/generate`, e pede a Nexus Intelligence configurada uma lista de WorkPackages específicas. A API persiste somente um WorkPlan revisável.
- O botão **Criar manual** continua independente de provider. O legado `auto_plan: true` permanece rejeitado.
- A geração falha antes de gravar um plano se não houver tarefas, critério de aceite, objetivo específico, resolução inequívoca de dependências, DAG acíclico ou uma quantidade adequada de atividades para um prompt de múltiplas frentes.
- Dependências são resolvidas por título ou ordinal; atividades dependentes ficam `PENDING`.
- Sem GO, nenhum Agent, runtime, scheduler ou Mission inicia. Routing, cost, preview, approval e evidence permanecem nos contratos existentes.
- O caminho Composer → Flow ainda não possui paridade semântica (#166). Goal Loop automático ainda não está implementado (#167).

## Gates obrigatórios (Go 1.25)

```bash
go version
gofmt -l internal/nexus/flow_generated_plan*.go internal/nexus/plan.go internal/control/web/handlers_flow_generate*.go internal/control/web/routes.go internal/nexus/intelligence/{cli_provider,openai_provider}.go
go test -count=1 ./internal/nexus ./internal/control/web ./internal/nexus/intelligence
go test -race -count=1 ./internal/nexus ./internal/control/web
go vet ./internal/nexus ./internal/control/web
cd web && npm run format:check && npm run lint && npm run typecheck && npm run test && npm run build
```

Qualquer falha, formatação não aplicada ou Go abaixo de 1.25 impede aprovação. Registrar SHA, OS, logs dos comandos, snapshot da UI e os resultados no PR.

## E2E em pelo menos dois clientes

1. Cole um prompt full-stack com Backend, Frontend, migrações e QA. Verifique quatro ou mais tarefas **específicas** quando exigidas, dependências exatas, paralelismo e critérios mensuráveis.
2. Com prompt atômico, verifique uma atividade sem inflação artificial.
3. Desconfigure o provider; a UI deve apresentar erro, preservar o prompt digitado e não criar Flow.
4. Provoque deliberadamente referência ausente, dependência circular, tarefas duplicadas, conteúdo placeholder e prompt complexo condensado em uma tarefa; nenhum WorkPlan deve ser persistido.
5. Para pergunta bloqueante, verifique status 409, clarificação visível e revalidação após resposta, sem bypass.
6. Teste o caminho separado **Criar manual**, sem LLM.
7. Em duas abas e dois clientes, gere Flow e valide revisão, alterações concorrentes e edição sem executar.
8. Valide idioma PT/EN/ES, navegação por teclado, 390px, foco, contraste e feature tour.
9. Inspecione Execution Preview, aprovação, entradas do MissionRunner, logs e custo. Gerar Flow não é GO.
10. Não declarar Goal Loop como concluído por existir revisão/testes: #167 exige retry seletivo sob orçamento/limite e evidências reais.

## Evidência de entrega

PR draft, sem merge até evidência em Go 1.25 e browser real. Nenhuma alteração deve ir para `main`; alvo exclusivo `release/v1.0.0`.
