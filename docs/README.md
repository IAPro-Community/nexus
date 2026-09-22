# Documentação do IAPro Nexus v1

Esta é a **entrada pública canônica** para entender, instalar, usar, operar e
estender o Nexus.

![Componentes do Nexus v1](assets/diagrams/nexus-v1-components.svg)

> `README.md` apresenta o produto. Esta pasta explica seu uso e arquitetura.
> `DEV/` é um ledger de engenharia, validação e histórico; não deve ser usado
> como onboarding nem substituir a verdade de produto documentada aqui.

## Comece pelo que você quer fazer

| Quero… | Comece em |
| --- | --- |
| testar em 5–10 minutos | [Quick Start](quick-start.md) |
| entender o produto inteiro | [Como o Nexus funciona](product/how-it-works.md) |
| abrir uma sessão agora | [Direct](product/direct.md) |
| transformar uma ideia em plano | [Composer](product/composer.md) |
| revisar dependências e execução | [Flow](product/flow.md) |
| executar trabalho longo/multi-Agent | [Missions](product/missions.md) |
| entender comunicação entre Agents | [Coordenação entre Agents](product/agent-coordination.md) |
| ver telas reais | [Tour visual](product/visual-tour.md) |
| instalar | [Getting Started](getting-started/README.md) |
| operar/diagnosticar | [Operação](operations/README.md) |
| usar a CLI | [Referência da CLI](cli/README.md) |
| desenvolver provider | [Provider development](provider-development.md) |
| entender a arquitetura | [Arquitetura](architecture/overview.md) |
| contribuir | [CONTRIBUTING](../CONTRIBUTING.md) |

## Mapa da documentação

### 1. Primeiro uso

- [Quick Start](quick-start.md)
- [Getting Started](getting-started/README.md)
- [Instalação nativa](getting-started/installation.md)
- [Primeiro projeto](getting-started/first-project.md)
- [Pacotes nativos](getting-started/packages.md)
- [Docker compatibilidade](getting-started/docker-compat.md)

### 2. Produto

- [Visão geral](product/overview.md)
- [Como funciona](product/how-it-works.md)
- [Modelo mental](product/mental-model.md)
- [Direct](product/direct.md)
- [Agents](product/agents.md)
- [Composer](product/composer.md)
- [Flow](product/flow.md)
- [Missions](product/missions.md)
- [Coordenação entre Agents](product/agent-coordination.md)
- [Conductor](product/conductor.md)
- [Autonomous delivery](product/autonomous-delivery.md)
- [Maestro](product/maestro.md)
- [Providers e usage](product/providers-and-usage.md)
- [Superfícies](product/surfaces.md)
- [Tour visual](product/visual-tour.md)

### 3. Arquitetura

- [Visão geral](architecture/overview.md)
- [Nexus Core](architecture/core.md)
- [Runtime da Mission](architecture/mission-runtime.md)
- [Guardrails](architecture/guardrails.md)
- [Mission Teams](architecture/mission-teams.md)
- [Runtime supervisionado](architecture/runtime-lifecycle.md)
- [Provider contract](architecture/provider-contract.md)
- [Resource routing](architecture/resource-routing.md)
- [Agent routing](architecture/agent-routing.md)
- [Persistência](architecture/persistence.md)
- [Eventos](architecture/events.md)
- [Continuidade](architecture/session-continuity.md)
- [ADRs](architecture/)

### 4. Operação

- [Operação e suporte](operations/README.md)
- [Troubleshooting](operations/troubleshooting.md)
- [Suporte de plataformas](operations/platform-support.md)
- [Release](operations/release.md)
- [Security](security.md)

### 5. Referência

- [CLI](cli/README.md)
- [Configuração](reference/configuration.md)
- [Matriz de funcionalidades](reference/feature-matrix.md)
- [Glossário](glossary.md)
- [FAQ](faq.md)
- [O que mudou](whats-new.md)

## Progressão de uso

```text
Direct
  ↓ quando precisa de clareza
Composer
  ↓ quando precisa revisar estrutura
WorkPlan / Flow
  ↓ GO explícito
Mission
  ↓ gates + evidência + convergência
COMPLETED_VERIFIED
```

A progressão não é obrigatória. Uma correção simples deve continuar simples.

## Fonte de verdade de cada assunto

| Assunto | Fonte canônica |
| --- | --- |
| como usar o produto | `docs/product/` |
| instalação | `docs/getting-started/` |
| operação e suporte | `docs/operations/` |
| arquitetura atual | `ARCHITECTURE.md` + `docs/architecture/` |
| contratos de execução | `docs/architecture/` + código |
| status funcional | `docs/reference/feature-matrix.md` |
| status de release | `DEV/validation/current/RELEASE_STATUS.md` |
| especificação de fechamento v1 | `DEV/SPECS/NEXUS_1_0_CLOSURE.md` |
| histórico/evidência | `DEV/` |

## Convenções de status

A documentação pública usa estes significados:

- **Disponível no source**: implementação e contrato existem no branch/revisão descrita.
- **Beta/Experimental**: implementação existe, mas compatibilidade/certificação ainda é limitada.
- **Opcional**: integração não é necessária ao Core.
- **UNKNOWN**: não há evidência suficiente para afirmar sucesso.
- **Planejado**: não deve ser descrito como funcional.

Status de código e status de release são coisas diferentes. Uma capacidade pode
estar implementada no source e a v1 estável ainda permanecer `NO_GO`.

## Imagens

Existem dois tipos de imagens:

1. **screenshots reais** — `assets/screenshots/`, produzidos por fixture
   sintética e registrados em manifesto;
2. **diagramas conceituais** — `assets/diagrams/`, usados para explicar
   arquitetura e fluxo.

Diagramas não substituem evidência de plataforma, runtime ou provider.

## Documentação interna

`DEV/` contém specs, worklogs, relatórios e campanhas de validação. Muitos
arquivos são históricos e preservam decisões de uma revisão específica.

Para evitar confusão:

- estado público de release: [Status da release](release-status.md);
- contrato interno de fechamento: `DEV/SPECS/NEXUS_1_0_CLOSURE.md`;
- índice interno: `DEV/INDEX.md`.

## Guia editorial

Consulte [STYLE_GUIDE.md](STYLE_GUIDE.md). Regra principal: **fato verificável
antes de marketing**.
