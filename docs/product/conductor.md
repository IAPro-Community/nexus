# Conductor

Conductor é a camada de decisão/decomposição que transforma um objetivo composto
em trabalho estruturado usando os contratos já existentes do Nexus.

Ele **não** é um segundo runtime ou uma segunda engine de orquestração.

```text
Objetivo
  ↓
classificação / decomposição
  ↓
WorkPlan
  ↓
Flow / Mission
  ↓
MissionRunner
```

## Responsabilidade

Conductor pode decidir entre:

- manter trabalho simples no caminho Direct;
- decompor um objetivo substancial;
- selecionar um preset de Mission Team;
- produzir packages/dependências que seguem o pipeline canônico.

Depois disso, execução pertence ao `MissionRunner`.

## O que Conductor não faz

- não cria provider diretamente;
- não substitui ResourceScheduler;
- não substitui AgentMatcher;
- não mantém uma segunda cópia do plano;
- não executa uma state machine paralela;
- não concede push/deploy implicitamente.

## Tarefas simples

Uma correção localizada não deve ser inflada para uma Mission multi-Agent.
Classificação considera complexidade, dependências, risco e custo de
coordenação.

## Objetivos compostos

Features, mudanças de arquitetura, release/security work e refactors maiores
podem ser decompostos em fases como:

```text
Planning → Implementation → Review → Verification
```

O preset descreve papéis/requisitos; não fixa providers.

## Superfícies

A disponibilidade concreta dos comandos/superfícies do Conductor pode evoluir
durante o fechamento da v1. Para evitar documentação contraditória, trate
`nexus --help`, a [matriz de funcionalidades](../reference/feature-matrix.md)
e o source da revisão instalada como referência de interface.

O fluxo de produto na Web para planejamento continua centrado em
**Composer → WorkPlan/Flow → GO → Mission**.

## Segurança

Conductor respeita as mesmas boundaries da execução:

- plano aprovado;
- worktree/policies;
- AgentMatcher + ResourceScheduler;
- AutonomyContract;
- gates/evidence;
- intervenção humana quando necessário.

Veja [Como funciona](how-it-works.md), [Missions](missions.md) e
[Guardrails](../architecture/guardrails.md).
