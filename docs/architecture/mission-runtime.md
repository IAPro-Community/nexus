# Runtime da Mission

![Ciclo de execução da Mission](../assets/diagrams/nexus-v1-mission-lifecycle.svg)

## Princípio

`MissionRunner` é uma máquina de estados durável. Provider output é entrada da
execução, não autoridade para mudar o estado global.

O caminho principal de uma package é:

```text
READY
  → ALLOCATE
  → COMPILE
  → EXECUTE
  → TESTING
  → REVIEW
  → VERIFIED
```

Depois que todas as packages necessárias estão verificadas, ocorre a
**convergência global**. A Mission só chega a `COMPLETED_VERIFIED` quando o
Definition of Done global também passa.

## Estados e responsabilidades

| Estado | Responsabilidade |
| --- | --- |
| READY | dependências satisfeitas |
| ALLOCATE | selecionar Agent/workspace/recurso |
| COMPILE | produzir prompt imutável e contexto |
| EXECUTE | cruzar o boundary do provider |
| TESTING | executar verificações determinísticas |
| REVIEW | revisão independente quando exigida |
| VERIFIED | package comprovada |
| BLOCKED_NEEDS_USER | decisão humana necessária |
| FAILED_* | falha terminal classificada |
| COMPLETED_VERIFIED | DoD global comprovado |

## Dispatch boundary

Antes de chamar um provider, o runner persiste uma intenção de dispatch. Isso
impede que um restart transforme um resultado externo desconhecido em uma
execução duplicada.

Se o processo cair depois de cruzar o boundary e não houver evidência do
resultado, o estado é `DISPATCH_OUTCOME_UNKNOWN` e exige reconciliação.

## Retry, remediation e no-progress

Falhas de teste/review podem retornar a execução para uma estratégia diferente,
desde que o `AutonomyContract` ainda permita tentativas.

O runner acompanha assinatura da falha e progresso. Repetir indefinidamente o
mesmo erro não é uma estratégia válida.

## Dependências e paralelismo

Packages formam um DAG. Uma package `PENDING` só vira `READY` quando suas
dependências estão em `VERIFIED`.

Grupos realmente independentes podem executar em paralelo, mas cada dispatch
continua tendo identidade, workspace, receipt e evidência próprios.

## Handoff

O handoff entre packages utiliza contratos limitados:

- `ContextCapsule`;
- `WorkReceipt`;
- artifacts;
- decisões e riscos;
- evidência de verificação.

Transcrições brutas do provider não são a memória canônica da Mission.

## Recovery

Após reinício:

1. a Mission é lida do store;
2. evidence é reidratada;
3. leases antigas são reconciliadas;
4. dispatches desconhecidos permanecem bloqueados;
5. o próximo estado seguro é calculado.

## Conclusão global

Uma Mission não pode concluir enquanto houver:

- package obrigatória não verificada;
- gate obrigatório falhando;
- intervenção humana pendente;
- Definition of Done global incompleto.

Isso evita **false DONE**.
