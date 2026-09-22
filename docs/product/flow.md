# Flow

Flow é o **studio de pipeline** do WorkPlan canônico: Plan → Build → Review → Verify (e fases extras do preset quando existirem). Não é um segundo planner nem uma fonte paralela de verdade — projeta o plano e devolve mutações válidas ao runtime.

## Superfície

- **Home:** pipeline de fases com cards de etapa e inspector à direita (`FlowDraftEditor`).
- **Vista secundária:** grafo de dependências.
- **Histórico:** execuções (Mission Runs) em aba própria — não é a porta de entrada.

Use Flow quando houver dependências, paralelismo, múltiplos agentes, gates, longa duração, recovery ou isolamento. Para uma tarefa curta, Direct ou Composer → Copy/Agent é suficiente.

Criar um draft não inicia Agent, MissionRun ou runtime. Prefira trazer um prompt refinado do Composer (**Turn into Flow**). **Novo Flow** cria um draft vazio editável — `auto_plan` / geração automática de plano na Web foi aposentado; decomposição rica passa pelo Composer (ou pelo Conductor na CLI).

Dúvidas críticas reutilizam o mesmo contrato de esclarecimento do Composer e não repregunta o que sessão ou memória já cobriram.

No modo Execution (`FlowRunSurface`), observe estados, attempts, artifacts, blockers e verification reais.


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
