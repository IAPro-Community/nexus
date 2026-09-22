# Entrega autônoma

Entrega autônoma é o uso do pipeline completo para transformar um objetivo em
uma Mission verificada, mantendo planejamento, Agents, routing, handoffs,
review e evidência sob a mesma autoridade de execução.

```text
Objetivo
  → plano
  → implementação
  → review
  → correção quando necessária
  → verificação
  → COMPLETED_VERIFIED
```

## Equipe virtual

O Nexus pode decompor um objetivo substancial em papéis como:

- architect;
- implementer;
- reviewer;
- security reviewer;
- QA/verifier;
- especialistas compatíveis com a tarefa.

Os papéis são requisitos da package. Eles não codificam provider.

## Provider é recurso, não papel

Uma policy pode preferir um recurso para determinada fase, mas o Nexus registra
separadamente:

- Agent escolhido;
- provider/profile/model desejado;
- recurso realmente executado;
- motivo de fallback.

Isso evita descrever uma execução como “Claude review” ou “Codex planning”
quando esse recurso não foi realmente utilizado.

## Tarefas pequenas continuam pequenas

Um typo, rename simples ou ajuste localizado não deve receber uma equipe de
quatro fases. Classificação e custo de coordenação fazem parte da decisão.

## Promoção a partir de sessão supervisionada

Uma sessão pode permanecer Direct ou ser promovida a trabalho estruturado
quando a política identifica um objetivo composto.

A promoção reutiliza:

```text
Conductor/Composer → WorkPlan → Flow → MissionRunner
```

Ela não cria outro orquestrador.

## Review e remediation

Findings de review retornam ao trabalho de implementação quando necessário.
IDs estáveis de finding ajudam a distinguir progresso de repetição do mesmo
problema.

O loop é limitado pelo `AutonomyContract`. Esgotamento pode produzir falha de
no-progress ou intervenção tipada.

## Coordenação dinâmica

Um Agent que precisa de outra informação ou especialista usa o protocolo de
[coordenação entre Agents](agent-coordination.md), nunca spawn livre de
providers/workers.

## Evidência

A entrega autônoma deve ser explicável depois do fato:

- plano autorizado;
- Agents;
- recursos executados;
- receipts;
- artifacts;
- testes;
- findings;
- retries;
- intervenções;
- DoD final.

Essa rastreabilidade é parte do produto, não apenas logging.
