## Follow-up P5 — pedido DIRECT atômico com Mission explícita até MissionRunner — 2026-10-26

HEAD local/remoto `7a04f1b4cf5950b97b4ea1e156dda31ac513823b`; overlay fonte `ab5629f56e014719c0ce89c88b16b986f5bb0da8c310baac9fa079370140da5c` (1.428 arquivos; [manifesto](logs/source-provenance-p5-direct-atomic-mission-2026-10-26.txt)).

- `TestStartDispatchExplicitMissionForAtomicFixUsesSingleWorkPlanAndMissionPackage` verifica que “Corrija este teste quebrado como Mission.” é DIRECT e pede Mission explicitamente, Composer/Flow não são obrigatórios, a decisão e o team selection são persistidos no WorkPlan e a execução explícita gera uma única etapa/pacote.
- O fluxo usa preview/admission e MissionRunner reais do Nexus; credenciais/CLI Codex são sintéticas, executor e reviewer são fixtures. `go test ./...` executa no worktree temporário e a fixture persiste evidência de validação do pacote e global; a missão chega a `COMPLETED_VERIFIED` com um dispatch, sem rede/provider externo.
- PASS: [execução focada](logs/p5-direct-atomic-mission-integration-2026-10-26.txt), [race x3](logs/p5-direct-atomic-mission-race-2026-10-26.txt), [vet](logs/p5-direct-atomic-mission-vet-2026-10-26.txt), [suíte `internal/nexus`](logs/p5-direct-atomic-mission-full-package-2026-10-26.txt).
- Limite: isto cobre a trilha de backend determinística, não o caminho de conversa no navegador/terminal nem a janela Wails ou handoff autenticado real. P5 continua parcial. Resultado global permanece `NO_GO_LINUX`.

## Reteste E2E Composer — edição semântica, objetivo vago e grounding — 2026-10-26

Branch `release/v1.0.0`; HEAD local/remoto `7a04f1b4cf5950b97b4ea1e156dda31ac513823b`. Fingerprint do overlay intermediário do E2E: `1c3aa2c6ca9043f4a43dc3cd7a5ade451930d21c6fe3ff2e07de4cc0ed068693` (1.428 fontes; [manifesto](p5-recheck-2026-10-26T0347Z/logs/source-provenance-composer-e2e-before-backend-fixes.txt)).

- **PASS — spine de execução:** objetivo alterado gerou revisão semântica 2 e token de aprovação novo; o token anterior ficou inválido. Sem provider elegível, nenhum Agent/Mission foi criado e a prévia permaneceu `ready=false`. O estado/sentinel do fixture persistiu; erros de console/página: zero.
- **PASS — objetivo vago/grounding:** dois esclarecimentos resolvidos; referência `package.json/scripts.test` (Vitest) permaneceu no manifesto de contexto com confiança e timestamp; POST `/finalize` respondeu HTTP 201 e a sessão ficou `FINALIZED`. O WorkPlan produziu cinco etapas; um prompt completo não gerou perguntas bloqueantes; erros de console/página: zero.
- Evidências reais: `p5-recheck-2026-10-26T0347Z/logs/composer-spine-retry.txt`, `p5-recheck-2026-10-26T0347Z/logs/composer-spine-retry/north-star.json` e PNGs; `p5-recheck-2026-10-26T0347Z/logs/composer-vague-spine.txt`, `p5-recheck-2026-10-26T0347Z/logs/composer-vague-spine/north-star-vague.json` e PNG.
- A primeira tentativa do spine terminou antes do cenário por falta de espaço no `/tmp` (`no space left on device`). O retry usou `TMPDIR` e `GOCACHE` em ext4, em diretório separado, e passou. Nenhuma árvore/cache compartilhada foi apagada.
- Esses testes não chamam provider externo e não aprovam a janela Wails. `preview_ready=false` é resultado seguro porque nenhum provider elegível está configurado. P5 permanece **PARCIAL**; continua faltando handoff real autenticado e fluxo completo pela janela Desktop.

A homologação continua com um único mascote NEX; Companion 2D não é requisito.

# P5 — Composer aprovado até MissionRunner

## Follow-up corrente — limite de dispatch e continuidade após restart (2026-10-25)

HEAD local/remoto `7a04f1b4cf5950b97b4ea1e156dda31ac513823b`; overlay atual `2c23ebb0c7726a200acbde9326c23ad6023d8fad0238c2adfef993677203f2fa` (1.425 fontes; [manifesto](logs/source-provenance-p5-predispatch-continuity-2026-10-25.txt)). A execução de `executeAgentPrompt` agora marca como comprovadamente não enviada apenas a falha ocorrida antes de chamar `Launcher.Launch`. `nexusPackageExecutor` converte somente esse tipo em `runner.MarkDispatchNotSent`; erros retornados pelo launcher continuam com resultado externo desconhecido e bloqueiam replay.

As regressões cobrem os dois lados do limite: agente sem perfil falha antes do launcher (zero chamadas, `ErrDispatchNotSent`); erro do launcher acontece com uma chamada e não é classificado como seguro. O fluxo OAuth Composer/WorkPlan da seção seguinte injeta uma falha pré-dispatch, fecha e reabre SQLite, recria o runner e completa o plano mantendo a revisão imutável e sem repetir dispatches concluídos.

**PASS:** regressões focadas, `go test -race -count=3` nos três cenários, `go test -count=1 -timeout=4m ./internal/nexus/...` e `go vet ./internal/nexus/...`, com `GOCACHE=/tmp/nexus-linux-100-2026-10-25/gocache-p5-full-current`. Logs: `p5-predispatch-continuity-focused-2026-10-25.log`, `p5-predispatch-continuity-race-2026-10-25.log`, `p5-predispatch-continuity-go-packages-2026-10-25.log` e `p5-predispatch-continuity-go-vet-2026-10-25.log`. A fixture é determinística; não houve acesso a credenciais do host nem chamada real a provedor.

P5 continua **PARCIAL**: os cinco cenários oficiais ainda não estão todos concluídos; handoff real autenticado e fluxo equivalente pela janela Wails continuam sem validação.

## Histórico — prompt OAuth multiárea até MissionRunner (overlay 0c522c88, 2026-10-25)

HEAD local/remoto `7a04f1b4cf5950b97b4ea1e156dda31ac513823b`; overlay `0c522c885e700dc52d6d03a0feb313bc7d5e420915ca0188fe6c554b49fe9841` (1.424 fontes; [`logs/source-provenance-p5-large-feature-mission-fixture-2026-10-25.txt`](logs/source-provenance-p5-large-feature-mission-fixture-2026-10-25.txt)). Ampliei `TestComposerApprovedFlowReachesMissionRunner` de um ajuste pequeno para uma solicitação OAuth grande, cobrindo backend API, frontend React, PKCE/cookies, testes, revisão e critério de aceite. Com perfis/CLI Codex sintéticos e executor determinístico, a solicitação gera um WorkPlan multiárea; uma edição semântica invalida a aprovação antiga, o token anterior não admite run e a aprovação renovada conclui pelo menos quatro pacotes no MissionRunner preservando a revisão no snapshot. Nenhuma conta do host ou provedor externo foi usada.

**PASS:** suíte `go test -count=1 -timeout=4m ./internal/nexus/...`, `go vet ./internal/nexus/...` e o cenário direcionado `go test -race -count=3 -run '^TestComposerApprovedFlowReachesMissionRunner$' ./internal/nexus`. O race inicial mostrou que o contador compartilhado do executor falso não era protegido; preservamos esse log e corrigimos exclusivamente o fixture com `sync.Mutex`. A repetição com race e a suíte completa passaram. Evidências: `logs/p5-large-feature-mission-fixture-focused-2026-10-25.log`, `logs/p5-large-feature-mission-fixture-race-2026-10-25.log`, `logs/p5-large-feature-mission-fixture-race-final-2026-10-25.log`, `logs/p5-large-feature-mission-fixture-go-packages-2026-10-25.log` e `logs/p5-large-feature-mission-fixture-vet-2026-10-25.log`.

O E2E Chromium Flow abaixo continua como evidência da edição visual e foi executado no overlay anterior `ee6a9dd866ae4a938d66d958fe08d3711088f41233358f2c13feee1df015d590`; não substitui a janela Wails nem prova execução de provider real. P5 segue **PARCIAL** até validar handoff/continuidade externa e o fluxo nativo.

## Follow-up anterior — edição semântica e validade da aprovação (2026-10-25)

HEAD local/remoto `7a04f1b4cf5950b97b4ea1e156dda31ac513823b`; overlay `ee6a9dd866ae4a938d66d958fe08d3711088f41233358f2c13feee1df015d590` (1.424 fontes; [`logs/source-provenance-p5-semantic-revision-2026-10-25.txt`](logs/source-provenance-p5-semantic-revision-2026-10-25.txt)). A regressão backend altera o objetivo após uma prévia pronta, verifica que o token antigo é rejeitado mesmo com a revisão nova aprovada, confirma que nenhum run é criado, refaz a prévia e valida que o MissionRunner congela o objetivo editado no snapshot. `go test -race -count=3 -run '^TestComposerApprovedFlowReachesMissionRunner$' ./internal/nexus`, `go test -count=1 -timeout=4m ./internal/nexus/...` e `go vet ./internal/nexus` passaram. O E2E Chromium editou o Goal na UI Flow, persistiu revisão 2 antes do preview, observou token novo e confirmou zero Missions/Agents sem provider elegível. Erros de página/console: zero. O cancelamento isolado de HEAD `wails/custom.js` é o probe do harness. Capturas: `screenshots/p5-semantic-revision-e2e-2026-10-25/`; checksums: `logs/p5-composer-semantic-revision-screenshots-2026-10-25.sha256`. Este caso não é validação da janela Wails nativa e não comprova execução por provider. P5 e a homologação global permanecem parciais/NO_GO.

## Identificação

- Branch: `release/v1.0.0`
- HEAD local/remoto: `7a04f1b4cf5950b97b4ea1e156dda31ac513823b`
- Overlay efetivamente testado: `5e39faee13bd663dc56798cffe435e52e2bca081f5332f6d0619c0a61ddce011` (1.424 arquivos)
- Worktree: modificada e não congelada; alterações anteriores preservadas, sem commit ou publicação.
- Manifesto: [proveniência das fontes](logs/source-provenance-p5-composer-mission-fixture-2026-10-25.txt)

## Defeito reproduzido e correção

O preview tratava especialidades da tarefa (`security`, `code_review`) como capacidades obrigatórias do executável do provedor. Um Flow válido ficava bloqueado nas etapas Planning e Review, embora o perfil Codex tivesse as capacidades de controle necessárias (`headless` e `submit_prompt`). Isso misturava seleção de especialidade do Agent com seleção de runtime.

`providerResourceRequirements` agora mantém no hard gate somente capacidades de controle conhecidas e anunciadas pelo driver. Requisitos de tarefa continuam disponíveis para o Agent matcher. Recursos explicitamente pedidos, como `read_only_review`, seguem sujeitos ao hard gate. A prévia e o executor de Mission usam a mesma projeção, inclusive para decidir se o recurso atual pode ser mantido.

## Validação determinística

`TestComposerApprovedFlowReachesMissionRunner` cria HOME, configuração, perfil Codex, token sintético, quota atribuída e CLI de detecção em diretório temporário. Nenhuma credencial do host é lida e nenhum pedido externo é feito. O fluxo testado:

1. Cria e finaliza um prompt Composer.
2. Materializa o artefato como WorkPlan e confere sua proveniência.
3. Define verificações `exit 0` pertencentes à fixture.
4. Obtém uma prévia pronta com cada etapa direcionada ao perfil Codex temporário.
5. Aprova a revisão e confirma que o MissionRun persiste um snapshot imutável.
6. Executa as etapas pelo MissionRunner com executor e reviewer determinísticos até `COMPLETED_VERIFIED`.

O teste de regressão `TestProviderResourceRequirementsKeepTaskSpecialtiesOutOfControlGate` confirma que `security` e `code_review` não bloqueiam o provider, que `read_only_review` permanece uma restrição de controle e que os requisitos originais não são mutados.

## Resultados

| Verificação | Resultado |
|---|---|
| Dois testes focados | PASS |
| `go test ./internal/nexus/... -count=1 -timeout=4m` | PASS |
| Os dois testes focados com `-race -count=3` | PASS |
| `go vet ./internal/nexus/...` | PASS |
| Provedor autenticado real / resposta externa | BLOCKED nesta fixture |
| Cinco cenários oficiais P5 e E2E Browser → runtime nativo real | NOT concluídos |

Logs: [testes focados](logs/p5-composer-mission-fixture-focused-tests-2026-10-25.log), [pacotes Nexus](logs/p5-composer-mission-fixture-go-tests-2026-10-25.log), [resultado race](logs/p5-composer-mission-fixture-go-race-result-2026-10-25.txt), [resultado vet](logs/p5-composer-mission-fixture-go-vet-result-2026-10-25.txt).

## Decisão

Esta fatia P5 está **PASS** para o fluxo determinístico backend e para a regressão de roteamento. Ela não homologa chamada de provedor, handoff real, continuidade externa, os cinco cenários completos, ou a interface Wails nesse fluxo. P5 permanece parcial; o resultado global segue **NO_GO_LINUX**.

A direção do mascote não mudou: o produto permanece com um único NEX, sem personagem ou modo Companion 2D separado.
