# Red-team independente — desacoplamento do Maestro

Data: 2026-09-13  
HEAD auditado: `e53f8352e4c3647632ebac7877cf0a6a3bd62647`  
Escopo: acoplamento oculto entre Nexus e Orquestrador Maestro  
Alterações de código: nenhuma

## Veredictos

### Maestro OFF — FAIL

O caminho **explicitamente configurado** como `OFF` possui boa evidência unitária:

- contexto durável não exige disponibilidade do Maestro quando
  `project.MaestroMode == OFF`;
- skills builtin e de projeto são resolvidas pelo catálogo genérico sem
  Maestro;
- o executor e o runner pertencem ao Nexus e não consultam o Maestro para
  alocação, dispatch ou supervisão de uma Mission;
- o instalador não instala Maestro sem opt-in.

Isso não fecha o requisito de produção **“Maestro OFF → Nexus works”** porque
`OFF` não é o default real. Novos projetos são persistidos em `ASSIST`, tanto
por `CreateProject` quanto pelo schema SQLite. Na ausência do Maestro, esse
default faz `PrepareContext` registrar `FAILED`; em seguida o Composer recusa
planejamento porque exige contexto `READY`. Um usuário que instala apenas o
Nexus recebe, por padrão, um produto parcialmente bloqueado até alterar
manualmente o modo para `OFF`.

### Maestro ON — NOT_CERTIFIED

Há evidência estática e unitária de enriquecimento:

- o Maestro é uma fonte adicional e opcional do catálogo;
- uma fonte offline é ignorada sem invalidar skills builtin/projeto;
- uma fonte online acrescenta contratos e proveniência de skills;
- orientação já materializada em um snapshot é carregada como metadado pelo
  executor do Nexus.

Não há evidência executada suficiente para aprovar **“Maestro ON → Nexus
enriched”** em produto real nem **“Maestro crash during run → Nexus keeps
runtime ownership”**. As campanhas overnight A (OFF), B (ON) e C (recovery)
estão `NOT_STARTED`. Além disso, quando um projeto `ASSIST` perde o Maestro
depois de ficar `READY`, `ObserveContextReadiness` muda o estado para `STALE`;
`ComposerContextData` então bloqueia novo planejamento. O código do executor
indica que uma Mission já congelada permanece sob propriedade do Nexus, mas
não existe teste/campanha de crash no meio da execução que prove continuidade
do runtime, terminal, evidência e conclusão da Mission.

## Achados

### MAESTRO-001 — P0 — default contradiz opcionalidade

Evidência:

- `internal/nexus/store/projects.go:76-78` define
  `p.MaestroMode = MaestroAssist` quando o chamador não informa modo.
- `internal/nexus/store/migrations/0001_init.sql:15` define
  `maestro_mode TEXT NOT NULL DEFAULT 'ASSIST'`.
- `internal/nexus/store/store_test.go:94-96` e
  `internal/control/web/handlers_nexus_test.go:85-87` cristalizam `ASSIST` como
  comportamento esperado.

Impacto: a instalação pode ser independente, mas a primeira experiência de
projeto não é Maestro-OFF por padrão.

### MAESTRO-002 — P0 — readiness falha no default sem Maestro

Evidência:

- `internal/nexus/context_readiness.go:235-237` registra `FAILED` quando o modo
  não é `OFF` e o Maestro está indisponível.
- `internal/nexus/context_readiness.go:270-276` passa pelo gate de readiness.
- `internal/nexus/context_readiness.go:296-299` rejeita qualquer estado
  diferente de `READY`.
- `internal/nexus/context_readiness_test.go:112-126` exige explicitamente o
  fail-closed em `ASSIST`.

Impacto: com o default `ASSIST`, a indisponibilidade do complemento impede o
Composer de produzir o contexto necessário ao planejamento. Isso contradiz a
afirmação do ADR de que a ausência “não falha o Nexus” para a experiência
integrada do produto.

### MAESTRO-003 — P1 — crash torna contexto STALE

Evidência:

- `internal/nexus/context_readiness.go:185-186` marca como `STALE` um contexto
  antes `READY` quando o Maestro se torna indisponível.
- o fingerprint em `internal/nexus/context_readiness.go:58-101` inclui a
  versão do Maestro, acoplando readiness à presença/versão do complemento.

Impacto: uma queda do Maestro não transfere propriedade do processo do runner,
mas interrompe novas operações de Composer no mesmo projeto. A continuidade
de uma Mission já ativa não foi exercitada.

### MAESTRO-004 — P1 — continuidade durante crash não tem prova

Evidência positiva de arquitetura:

- `internal/nexus/mission_service.go:127-193` cria snapshot imutável e inicia o
  `Runner` do Nexus.
- `internal/nexus/mission_executor.go:27-223` aloca recursos, persiste decisão
  e resolve workspace sem chamada ao cliente Maestro.
- `internal/nexus/mission_executor.go:246-267` apenas projeta a referência de
  orientação Maestro já recebida.

Lacuna: não foi encontrado teste que inicie uma Mission com Maestro ON, derrube
o Maestro durante a execução e verifique que PID/runtime, terminal, eventos,
recovery e conclusão continuam sob controle do Nexus.

### MAESTRO-005 — P2 — adaptador legado ainda exige Maestro

`internal/nexus/maestro_prompt_compat.go:15-39` mantém
`CompileAgentPrompt(...)` com validação estrita pelo catálogo Maestro quando
há skills solicitadas. O teste
`internal/nexus/prompt_compiler_test.go:83-98` exige erro
`MAESTRO_DEGRADED` se o Maestro estiver offline.

Busca no HEAD encontrou esse adaptador apenas em seus testes; o caminho
canônico `CompileAgentPromptForProject` usa o catálogo genérico. Portanto, é
acoplamento residual de API, não bloqueador observado no fluxo canônico.

### MAESTRO-006 — P1 — campanhas obrigatórias não executadas

`DEV/validation/overnight-production-certification/campaign.json` registra:

- `A_maestro_off: NOT_STARTED`;
- `B_maestro_on: NOT_STARTED`;
- `C_recovery: NOT_STARTED`;
- `soak_hours: 0`;
- `verdict: ABORTED_PRECONDITION_FAILED`.

O relatório overnight também declara que nenhuma Mission, troca ON/OFF ou
injeção de recovery foi executada. Testes unitários não substituem essa prova
operacional.

## Controles que passaram

- `install.sh` inicia `WITH_MAESTRO=false` e somente executa
  `npm install -g @iapro/orquestrador-maestro-cli` sob `--with-maestro`.
- `install.ps1` inicia `WithMaestro = $false` e somente instala sob
  `-WithMaestro`.
- `internal/nexus/skills/catalog.go:22-34` ignora fontes indisponíveis.
- `internal/nexus/skill_catalog.go:64-101` sempre inclui builtin e roots
  Nexus/projeto antes de adicionar Maestro como fonte opcional.
- `internal/nexus/skills/catalog.go:109-122` dá prioridade a projeto, usuário e
  builtin acima de Maestro.
- `internal/app/maestro.go:42-65` reporta indisponibilidade sem impedir o
  comando de status.

## Verificação executada

No mesmo HEAD, sem diferenças locais nos pacotes Go auditados:

```text
go test ./internal/nexus -run \
  'TestPrepareContextFailsWhenConfiguredMaestroUnavailable|
   TestGenericSkillIDsResolveWithoutMaestro|
   TestBuiltinCatalogCoversCoreExecutionSkillsWithoutMaestro|
   TestMaestroSkillSourceIsOptionalAndGeneric|
   TestComposerCanApplyBuiltinSkillWithoutMaestro|
   TestMissionRoutingProjectionPersistsSkillAndMaestroGuidanceReferences|
   TestMaestroUnavailableNeverFabricatesAdvice' -count=1

Go test: 7 passed in 1 packages
```

Os testes confirmam simultaneamente o desacoplamento do catálogo canônico e o
fail-closed do contexto em `ASSIST`; eles não constituem campanha overnight ou
injeção de crash.

## Decisão de go/no-go

**NO-GO para a alegação “Maestro é 100% opcional” no HEAD auditado.**

Critérios mínimos para reavaliação:

1. novo projeto nasce em `OFF`, inclusive no schema, API e UI;
2. ausência/queda do Maestro não muda readiness de contexto requerido pelo
   caminho base do Nexus;
3. campanha real OFF comprova criação de projeto, Composer, Flow, Mission,
   runtime, terminal, evidência e recovery sem Maestro instalado;
4. campanha real ON comprova enriquecimento observável, sem transferir
   ownership do runtime;
5. injeção de crash do Maestro no meio da Mission comprova continuidade e
   conclusão sob propriedade do Nexus.
