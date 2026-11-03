# Central de consumo Nexus — estudo comparativo com Oh My Pi (03/11/2026)

**Branch única vigente:** `release/v1.0.0` no Nexus; PR #195 já integrado e encerrado. Sem novo PR, banco de contabilização ou motor de sincronização paralelo. O Maestro permanece dono dos Runs duráveis; PR #59 já integrado à sua branch `v1`. Estar na release não implica homologação funcional ou publicação.

## Fontes estudadas

- Código/documentação do [OMP stats](https://github.com/can1357/oh-my-pi/blob/main/packages/stats/README.md) e [CLI](https://github.com/can1357/oh-my-pi/blob/main/docs/cli-reference.md), consultados em 03/11/2026.
- [Documentação dos pacotes do OMP](https://github.com/can1357/oh-my-pi/blob/main/docs/user-facing-packages.md), seção `packages/stats`.
- Fonte do Nexus: `web/src/features/usage/UsageSurface.tsx`, `CostSummaryStrip.tsx`, `internal/nexus/store/cost.go`, `internal/nexus/cost_application.go`, `internal/core/quota/*` e `docs/architecture/usage-quota.md`.

O OMP dispõe de **duas visões distintas**: `omp usage` para limites dos provedores/contas e `omp stats` para análise local de sessões, solicitações e custos. Sua implementação usa transcrições JSONL incrementais e agregados SQLite; tem recortes temporais, projeto, modelo, cache, TTFT, tokens/s, latência, erros, traces e ferramentas. O OMP também diferencia a estimativa de preço público equivalente à API do débito da assinatura.

O Nexus já tem uma camada rica de **cotas** (`UsageSnapshot`, janelas, resets, perfis, monitor de capacidade, scheduler) e **ledger de observações de execução** (`cost_observations`, projetos, missões, agentes, geração, provider, modelo e indicadores de origem). A interface mostrava cotas por conta e custo resumido em sete dias, mas sem série histórica e comparação por modelo na tela principal.

## Matriz de evolução

| Capacidade | OMP de referência | Nexus na release base | Entrega deste PR | Risco/remanescente |
| --- | --- | --- | --- | --- |
| Cotas por conta/provedor | `omp usage`, janelas e resets | Já presente por provedor/perfil | Reutilizada sem alterações de collector | Sem fontes nativas não mostrar limite inventado |
| Consumo por intervalo | Séries horários e recortes de período | Resumo fixo sete dias | API `cost/analytics`, 7/30/90d; buckets diários | Não equivaler observações a requests faturados |
| Custo por modelo | Gráficos e tabelas por modelo | Ledger guarda modelo | Agregação limitada a 25 pares provedor/modelo | Sem normalizar preços/modelos por heurística |
| Tipos de custo | Estimativa API equivalente explícita | `cost_kind` já existe; UX somava valores | Reportado/estimado/inferido separados | NÃO são valores de fatura |
| Cache | Cache rate + savings | Tokens cached opcionais | Cache rate apenas com par input/cache observado + cobertura | Savings em USD requer rate card provado |
| Erros e desempenho | Erros, TTFT, tokens/s, latência | Duração/status e telemetry parcial | Fora do primeiro incremento | Padronizar evento e duração antes de calcular |
| Projetos/agents/tools | Projetos, sessões, tools, traces | Project/Agent/RuntimeGeneration/Run IDs no ledger | Agregação local respeita escopo de projeto | Correlacionar tools e traces sem duplicidade |
| Atualização | Watcher/incremental SQLite e status ao vivo | UI consulta cotas a cada 60s | Painel consome API a cada 60s quando visível | Não implementar segundo watcher ou ler credenciais/transcrições sem consentimento |
| Rastreamento de origem | Logs e sync status | `source`, `cost_kind` e evidência de nulidade | Indicadores de cobertura e `LOCAL_COST_OBSERVATIONS` | Mais explicação de ausência de dados por provider |

## Contrato implementado neste PR

`GET /api/v1/cost/analytics?days=7&project_id=...` reutiliza a autenticação HTTP e o `CostApplicationService`. Ele executa consultas SQL parametrizadas sobre `cost_observations`, sem importar transcrições externas, sem guardar prompts, sem carregar mais de 25 agrupamentos por modelo e sem presumir que valores ausentes sejam zero. `ByDay` é limitado naturalmente ao intervalo de até 365 dias; o frontend oferece 7, 30 e 90 dias.

As somas de `provider_reported_microusd`, `api_equivalent_microusd`, `inferred_microusd` e `unclassified_microusd` são independentes. Cada classe mantém também seu **contador de observações com valor conhecido** para distinguir ausência de dados de zero válido. `CacheRate` calcula `cached_read/(plain_input+cached_read)` **somente nas observações onde ambos os campos foram medidos**, com cobertura explícita e `null` para denominador zero ou sem evidência.

A tela mantém a seção de cotas e budget atual e acrescenta um painel de análise com períodos, atividade diária (contagem de observações) e detalhamento por modelo. O refresh é desacelerado ao ocultar a página. As traduções PT-BR/EN/ES e testes são parte do mesmo PR.

## Risco P0 encontrado no Nexus legado

O resumo anterior `CostSummaryStrip` usava `aggregate.cost_microusd` do `SummarizeCostObservations`, que agrega **todos os `cost_kind`**, e a política legada também usava esse valor. O P0 foi corrigido no código conforme o bloco abaixo, mas a homologação financeira continua pendente.

Critérios de correção: política explícita por tipo de custo, provas de fatura/receita quando existirem, comportamento conservador sob incerteza, testes de valores mistos e garantias de idempotência na origem. Não reduzir o orçamento nem alterar bloqueios existentes sem aprovação de semântica.

## Atualização do P0 de orçamento na release (03/11/2026)

**Correção de código aplicada, aguardando validação completa:** o resumo de custos agora deriva o orçamento da mesma agregação de evidências tipadas da analítica (`SummarizeCostEvidence`), sem criar outro ledger. O valor de `known_cost_microusd`, mantido para clientes legados, passa a corresponder **exclusivamente ao valor reportado pelo provedor**; campos novos mostram separadamente reportado, estimado pela API, inferido e não classificado, além da cobertura de observações reportadas.

- Somente o valor **reportado pelo provider** pode comprovar que um threshold de orçamento foi ultrapassado. Isso ainda **não é uma fatura validada**.
- Custo equivalente à API, custo inferido e observação sem custo não podem comprovar margem disponível. Orçamento com limite hard/emergency continua exigindo confirmação na incerteza, inclusive depois de ultrapassar um limite soft.
- O token de confirmação agora incorpora a composição das evidências: alterações no montante estimado ou na cobertura invalidam confirmação anterior; é uma mudança intencional e conservadora.
- `CostSummaryStrip` não mostra mais a soma mista como gasto; valores por provedor no resumo foram substituídos pela contagem de observações, mantendo o detalhamento por classe na analítica.
- Testes cobrem custo reportado vs estimado, limites soft/hard/emergency, token de confirmação, isolamento por projeto e API equivalente acima do hard limit.

**Ainda não entregue:** certificação financeira com faturas, conversão monetária, garantias de completude de provider, integração visual homologada e limites por assinatura. A política continua sendo **um guard operacional local**, não um serviço contábil.

## Gates para declarar a central de consumo concluída

1. **Integridade e isolamento**: testes de janela, projeto, unknown, estimativas, cache e truncamento de modelos; autenticação existente e rejeição de mutações.
2. **CI**: frontend (typecheck, lint, stylelint, Prettier, Vitest/build), Go tests, Windows TypeScript e `docs-verify` no SHA exato.
3. **Visual**: captura real das novas telas em desktop e mobile + revisão de acessibilidade; não inventar screenshots ou atualizar apenas o SHA do manifesto.
4. **Semântica de custo**: resolver P0 do `CostSummaryStrip`/budget e revisar cobertura dos provedores, inclusive contratos de assinatura.
5. **Funcionalidade avançada**: TTFT/erros/tokens/s e tracing somente após suporte explícito dos adaptadores, com fonte e confiança visíveis. Nenhum número sintético para preencher dashboard.

**Veredito atual:** incremento de analítica e separação de evidências do orçamento implementados, **não homologados**. Não confundir frontend construído/CI com evidência de faturamento real, sessões OMP conectadas ou release aprovada.
