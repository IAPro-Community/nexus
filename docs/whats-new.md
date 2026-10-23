# O que mudou recentemente

## 22/10/2026 — reconciliação técnica beta.10

No source da `release/v1.0.0` foram reunidas correções de scroll Web, entrega
de assets no Desktop, entrada do terminal Windows via stream PTY, exceção
explícita de revisão `DELEGATE` em STICKY_LEAD com Mission `OFF`, failover
supervisionado por quota e projeções `ExecutionTarget`/`CapacityDomain`/
`ModelCatalog`. Os respectivos contratos e testes têm links no
[catálogo de 36 recursos](reference/feature-matrix.md).

**Estado:** essas alterações são código de um candidato em beta e ainda não
possuem aprovação integrada/nativa same-SHA. O run Windows executado contra
o commit `e98c711` falhou em `format:check`; veja
[status](release-status.md) e
ledger interno de engenharia (`DEV/validation/current/BETA10_GAP_CLOSURE_20261022.md`).


## Fechamento do Nexus v1 — source candidato

A linha de fechamento da v1 consolidou o fluxo completo:

```text
Objetivo
  → Composer
  → WorkPlan
  → Flow
  → GO
  → MissionRunner
  → Agents
  → Evidências
  → COMPLETED_VERIFIED
```

### Planejamento com uma única fonte de verdade

- Composer refina objetivo, contexto, aceite, riscos e restrições.
- `PromptArtifact` é versionado.
- `WorkPlan` é a fonte de verdade do plano.
- Flow é a projeção visual/editável desse mesmo plano.
- GO autoriza uma revisão específica antes da execução.

### Agents e recursos desacoplados

- Agent é identidade persistente.
- AgentMatcher escolhe **quem** executa.
- ResourceScheduler escolhe **com qual provider/profile/model**.
- Preferência de provider não vira `PIN` automaticamente.
- Capabilities exigidas pela tarefa chegam ao scheduler.

### Mission durável e evidence-first

- `MissionRunner` persiste estado e transições.
- packages executam por dependências e podem usar paralelismo isolado.
- testes/review podem retornar para remediation.
- recovery/restart preserva lineage e evidence.
- o estado final depende do DoD global, não de um “done” textual.

### Coordenação dinâmica entre Agents

A v1 fecha um protocolo tipado:

- `NEED_INFO`;
- `ARTIFACT_REQUEST`;
- `REVIEW_REQUEST`;
- `DELEGATE`.

O Nexus tenta memória/receipts/artifacts antes de criar trabalho novo. Quando
precisa de um especialista, materializa uma child package rastreável, registra
seu `WorkReceipt` e devolve o resultado ao Agent de origem por um
`ContextCapsule` atualizado.

Replay automático só acontece com prova positiva de segurança. Caso contrário,
a Mission bloqueia para intervenção humana.

### Documentação v1

A documentação pública foi reorganizada em PT-BR e ganhou diagramas canônicos
versionados em `docs/assets/diagrams/`:

- visão geral;
- fluxo ponta a ponta;
- componentes;
- coordenação entre Agents;
- lifecycle da Mission;
- guardrails.

Comece por [Como o Nexus funciona](product/how-it-works.md).

> **Importante:** implementação no source candidato não equivale a release
> estável certificada. Consulte
> [status da release](release-status.md).

## Delegação automática do Lead

Uma sessão interativa supervisionada pode avaliar, por política determinística,
se um objetivo composto se beneficia de especialistas persistentes. `AUTO`,
`ASK` e `OFF` controlam a decisão; matching e resource routing continuam
separados.

Tarefas pequenas continuam diretas. Delegação é usada quando dependências,
risco, custo de coordenação e evidence justificam a estrutura adicional.

Veja [Agents](product/agents.md), [Flow](product/flow.md),
[Coordenação entre Agents](product/agent-coordination.md) e
[Matriz de funcionalidades](reference/feature-matrix.md).
