# Comparison Lab — evidência parcial

Data: 2026-11-09
Branch: `release/v1.0.0`
Status: **PARTIAL / NO_GO**

## Implementado

Além de comparar execuções concluídas, o Flow Studio agora prepara duas variantes congeladas de um mesmo WorkPlan/revisão/base. Cada variante recebe IDs independentes para fases e pacotes, dependências remapeadas, instruções registradas, preview de execução próprio e confirmações separadas de execução/custo. Ambas seguem pelo RunApplicationService/MissionRunner existente em workspaces gerenciadas; não existe segundo scheduler. A criação dos dois planos e da campanha é atômica.

A campanha registra vínculo de cada run e estado de lançamento. Na recuperação do Core, uma tentativa sem runs adquiridas volta a PREPARED; lançamento unilateral fica PARTIAL para inspeção humana e não inicia trabalho automaticamente. A comparação existente valida workspaces Nexus distintas, base/revisão/origem, e salva resultados de testes e verificação, diffs, custos com proveniência e itens desconhecidos em snapshot imutável. A decisão humana é única, com justificativa. Não escreve no checkout, aplica diff ou faz merge.

A API e a persistência têm testes para escopo/proteção, idempotência da decisão, imutabilidade do snapshot, isolamento de workspace, preparo de variantes, consentimento dos previews, recuperação sem runs e ausência de alteração no checkout. A interface reutiliza os componentes existentes, possui traduções PT-BR/EN/ES e usa o Product Tour compartilhado.

## Lacunas

- Ainda não há evidência de uma campanha realmente executada com dois providers/runs em paralelo; os testes atuais validam preparo, autorização e ligação/comparação de runs persistidas, sem chamar um provider externo.
- Em uma falha unilateral a campanha fica `PARTIAL`; o estado exige inspeção e não há retry automático ou fluxo de recuperação da alternativa faltante.
- Testes e custos são evidências reportadas pelas execuções existentes; não existe instrumentação completa para provar comparabilidade de toda métrica de custo.
- Não foi homologado visualmente em aplicativo distribuído no SHA exato; `make docs-verify` segue bloqueado por manifests sintéticos associados a SHA anterior.
- O critério de alternativas concorrentes e a capability `comparison-lab` permanecem `partial`/`pending`; não contar como uma das 48 homologadas.

## Verificações executadas

- PASS — `go test ./internal/nexus/store ./internal/nexus ./internal/control/web -run 'TestComparison' -count=1` (store, Nexus e handlers), incluindo recovery de campanha sem runs após restart.
- PASS — `npm --prefix web run format:check` e `npm --prefix web run typecheck`.
- PASS anterior — testes Web focados de API, anúncios e âncoras do tour (31 testes) e ESLint (seis warnings existentes); a suíte quality-full ainda precisa ser repetida após esta extensão A/B.

`a02c298a` foi o checkpoint anterior enviado. Esta extensão A/B ainda está em worktree, sem novo SHA imutável. `make docs-verify` segue BLOCKED: o guard visual relata que capturas/manifesto vinculados a `7a26a298…` ficaram obsoletos após nove fontes de renderização alteradas. Falta browser E2E visual no candidato publicado; não se usaram capturas sintéticas para declarar homologação. Ver [discovery frontend](FRONTEND-DISCOVERY.json).
