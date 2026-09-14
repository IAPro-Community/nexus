# Flow

Flow é o **studio de pipeline** do WorkPlan canônico: Plan → Build → Review → Verify (e fases extras do preset quando existirem). Não é um segundo planner nem uma fonte paralela de verdade — projeta o plano e devolve mutações válidas ao runtime.

## Superfície

- **Home:** pipeline de fases com cards de etapa e inspector à direita.
- **Vista secundária:** grafo de dependências.
- **Histórico:** execuções (Mission Runs) em aba própria — não é a porta de entrada.

Use Flow quando houver dependências, paralelismo, múltiplos agentes, gates, longa duração, recovery ou isolamento. Para uma tarefa curta, Direct ou Composer → Copy/Agent é suficiente.

Criar um draft não inicia Agent, MissionRun ou runtime. Você pode gerar um plano na própria página (**Novo Flow**) ou trazer um prompt refinado do Composer (**Turn into Flow**). Dúvidas críticas reutilizam o mesmo contrato de esclarecimento do Composer e não repregunta o que sessão ou memória já cobriram.

No modo Execution, observe estados, attempts, artifacts, blockers e verification reais.
