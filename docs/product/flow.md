# Flow

Flow é o **studio de pipeline** do WorkPlan canônico: Plan → Build → Review → Verify (e fases extras do preset quando existirem). Não é um segundo planner nem uma fonte paralela de verdade — projeta o plano e devolve mutações válidas ao runtime.

## Superfície

- **Home:** pipeline de fases com cards de etapa e inspector à direita (`FlowDraftEditor`).
- **Vista secundária:** grafo de dependências.
- **Histórico:** execuções (Mission Runs) em aba própria — não é a porta de entrada.

Use Flow quando houver dependências, paralelismo, múltiplos agentes, gates, longa duração, recovery ou isolamento. Para uma tarefa curta, Direct ou Composer → Copy/Agent é suficiente.

Criar um draft não inicia Agent, MissionRun ou runtime. O usuário pode **colar um prompt completo diretamente no Flow** e usar **Gerar atividades**: a Nexus Intelligence configurada produz WorkPackages com objetivos, dependências, papéis e critérios de aceite, que são validados antes da persistência. **Criar manualmente** continua disponível e não depende de IA. O antigo parâmetro `auto_plan` de criação genérica permanece desativado; a geração semântica usa a API própria `POST /api/v1/flow/generate` e não dispara execução.

**Composer → Turn into Flow** usa o mesmo planejador semântico do prompt direto nesta branch de continuidade (#166, dependente do PR #168): parte do PromptArtifact finalizado, preserva ID, versão, hash, decisões e manifesto de contexto congelados; gera WorkPackages com objetivos, dependências e critérios de aceite. **Sem provider, com saída inadequada ou dúvida bloqueante, a conversão falha explicitamente e mantém o artefato inalterado.** Uma nova dúvida bloqueante exige revisão e nova finalização no Composer, em vez de seguir a clarificação genérica e perder a proveniência. O Composer continua produzindo prompts completos e não se torna um executor.

Dúvidas críticas reutilizam o mesmo contrato de esclarecimento do Composer e não repregunta o que sessão ou memória já cobriram.

No modo Execution (`FlowRunSurface`), observe estados, attempts, artifacts, blockers e verification reais.


## Planejamento semântico e limites

A geração direta requer provider Nexus Intelligence explicitamente configurado. Uma saída de modelo com atividades ambíguas, critérios de aceite ausentes, dependências desconhecidas, ciclos ou colapso evidente de trabalho multiárea em uma só atividade é **rejeitada**, sem criar Flow substituto. A validação normaliza dependências por título/ordinal (`step 1`, `etapa 2`) e mantém etapas dependentes em `PENDING`.

O Flow é um rascunho revisável: o usuário deve inspecionar atividades, independência/paralelismo, papéis, evidências e custo no Execution Preview antes de aprovar GO. **Goal Loop** de reavaliação e reparação seletiva permanece pendente, especificado em [#167](https://github.com/IAPro-Community/nexus/issues/167); não confundir verificações atuais com um ciclo autônomo de correção.

## Coordenação dinâmica no Flow Run

Quando uma Mission materializa trabalho auxiliar a partir de uma solicitação
`NEED_INFO`, `ARTIFACT_REQUEST`, `REVIEW_REQUEST` ou `DELEGATE`, o
trabalho dinâmico continua pertencendo à mesma Mission.

O Flow Run apresenta a projeção dessa verdade persistida:

- request e status de coordenação;
- package de origem;
- child package dinâmica;
- lineage;
- materialização pendente/concluída;
- WorkReceipt/resultados quando disponíveis.

Isso não transforma Flow em scheduler. O `MissionRunner` continua sendo a
autoridade da execução.

Veja [Coordenação entre Agents](agent-coordination.md).

## Contrato Composer → Flow v1

O Composer continua sendo consultor de descoberta e planejamento; Flow é o editor
e a projeção da execução canônica. A conversão de um PromptArtifact aprovado
persiste `composer.handoff.v1` em `WorkPlan.StructuredFacts`, com:

- `session_id`, `artifact_id`, `artifact_version` e `artifact_hash` imutáveis;
- objetivo, respostas aprovadas, hipóteses e propostas da IA identificadas;
- perguntas bloqueantes não resolvidas e critérios de aceite;
- recomendação de destino entre Direct/Agent e Flow, explicável e não vinculante.

A interface de Flow Draft e Flow Run exibe o mesmo contrato, sem editar o
Composer por baixo dos panos. Uma proposta da IA continua sendo PROPOSED, e
dados inferidos não viram fatos executivos em `StructuredFacts`. O
`ExecutionPreview` acrescenta um bloqueio se o handoff v1 mantiver questões
`BLOCKING` abertas; a admissão do MissionRunner revalida essa projeção no GO.

O editor usa `@xyflow/react` para o grafo, mas deixa o pipeline como visão
inicial. O `MissionRunner` conserva toda a autoridade de execução, alocação,
retries, isolamento e recovery. Não há scheduler concorrente.

### Descoberta durante a execução e replanejamento

No Flow Run, a ação **Reavaliar planejamento no Composer** cria uma **nova**
elaboração com o ID da Mission, revisão de origem e razão para revisão. A
navegação abre a nova sessão, sem reescrever o plano original, a execução ativa
ou o `ExecutionSnapshot`. A pessoa precisa revisar, aprovar e materializar um
novo artefato/WorkPlan para qualquer execução futura. Nunca é feito um retry,
merge, deploy ou restart implícito.

Esse processo requer Nexus Intelligence configurada e responsiva. Com IA
indisponível, o histórico de Composer continua legível, mas novas elaborações
ficam bloqueadas. Maestro permanece opcional e independente.

### Homologação

Validar, no mínimo, transferência de decisões e proveniência,
ausência de promoção de inferências a fatos, admissões bloqueadas por lacunas,
revisões imutáveis, composição sem IA, persistência de histórico, permissão
do usuário no replanejamento e navegação em abas já montadas.

**Não declarar release estável sem validação local e por plataforma.**


## Excluir Flows

No Flow Studio, na área de rascunhos, cada Flow possui a ação **Excluir**.
O editor também oferece **Excluir Flow**. Ambas apresentam confirmação
explícita e enviam `DELETE /api/v1/plans/{id}?expected_revision=N`.

A exclusão é definitiva apenas para Flows sem Mission, snapshot de execução
ou agendamento. Isso é validado pelo armazenamento em uma transação antes
do delete: um Flow usado em execução passa a integrar a trilha histórica e
não pode ser apagado em cascata, nem mesmo depois de a Mission terminar.
O Nexus retorna HTTP 409 explicando o impedimento ou uma revisão obsoleta.
A lista e a seleção local são atualizadas após exclusão bem-sucedida.

Quando o Flow veio de um Composer, remover esse Flow não remove a elaboração
original automaticamente; o usuário pode excluí-la separadamente após
remover todas as dependências.

## Tour guiado e clareza operacional

O Flow Studio oferece **Conhecer esta área** na biblioteca, no editor e
no acompanhamento de Mission. O tour explica os caminhos
**Planejar → Revisar e aprovar → Executar Mission → Conferir evidências**
sem promover um rascunho manual a planejamento gerado por IA. A experiência
é contextual, acessível e não dispara uma execução. Consulte
[tours guiados Composer/Flow](composer-flow-tour.md) e suas verificações.
