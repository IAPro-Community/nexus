# Visão geral do produto

IAPro Nexus é uma **workstation local e control plane para coding agents**.

Ele reúne projetos, Agents persistentes, providers, terminais, workspaces,
planejamento, execução durável e evidência em um Core comum, exposto por Web,
Desktop e CLI.

![Visão geral do Nexus v1](../assets/diagrams/nexus-v1-overview.svg)

## O que resolve

Sem um control plane, cada coding CLI tende a manter sua própria sessão,
perfil, contexto operacional e forma de observar disponibilidade.

O Nexus cria uma camada comum para:

- manter identidade de Agent separada do provider;
- alternar recursos sem perder o modelo operacional;
- trabalhar diretamente quando a tarefa é simples;
- planejar visualmente quando a tarefa é complexa;
- coordenar múltiplos Agents;
- isolar escrita em worktrees;
- persistir evidence/recovery;
- concluir apenas por critérios verificáveis.

## Caminhos

### Direct

Para trabalho imediato:

```text
Project → AI Session → Provider → Terminal
```

### Trabalho planejado

Para objetivos que precisam de refinamento:

```text
Objetivo → Composer → WorkPlan → Flow
```

### Trabalho autônomo

Para execução longa ou coordenada:

```text
Flow aprovado → GO → Mission → COMPLETED_VERIFIED
```

## Nexus vs Maestro

Nexus possui execução, estado e UX. Maestro é integração opcional de método,
skills, risco e gates.

Veja [Como funciona](how-it-works.md), [Modelo mental](mental-model.md) e
[Arquitetura](../architecture/overview.md).
