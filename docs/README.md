# Documentação do IAPro Nexus

Esta é a entrada pública para entender, instalar e operar o Nexus. O produto
é uma workstation local para coding agents; a documentação parte do caminho
mais simples e só depois apresenta Composer, Flow e Mission.

## Escolha seu caminho

| Quero… | Comece em |
| --- | --- |
| experimentar o Nexus em 5–10 minutos | [Quick Start](quick-start.md) |
| descobrir uma capacidade pelo problema que quero resolver | [Funcionalidades](features/index.md) |
| usar a CLI ou automatizar uma rotina | [Referência da CLI](cli/README.md) |
| configurar o ambiente sem expor segredos | [Configuração](reference/configuration.md) |
| entender o produto | [Visão geral](product/overview.md) e [modelo mental](product/mental-model.md) |
| instalar no host (zero-toolchain) | [Instalação nativa](getting-started/installation.md) |
| instalar via DEB/RPM/NSIS | [Pacotes nativos](getting-started/packages.md) |
| rodar no Docker (CLI + Web) | [Docker compatibilidade](getting-started/docker-compat.md) |
| mapa getting-started | [Getting Started](getting-started/README.md) |
| ver ilustrações de instalação | [Visuais](getting-started/visuals.md) |
| abrir um agente diretamente | [Fluxo Direct](product/direct.md) |
| entender Web, Desktop e CLI | [Superfícies](product/surfaces.md) |
| operar agentes e terminais | [Agentes](product/agents.md) e [terminais](product/terminals.md) |
| usar Composer | [Composer](product/composer.md) |
| revisar um plano no Flow | [Flow](product/flow.md) |
| executar uma Mission | [Missions](product/missions.md) |
| entender Maestro | [Integração Maestro](product/maestro.md) |
| entender a implementação | [Arquitetura](architecture/overview.md) |
| integrar com a API local | [API local](api/README.md) |
| solucionar problemas | [Troubleshooting](operations/troubleshooting.md) |
| verificar suporte por sistema | [Suporte de plataformas](operations/platform-support.md) |
| contribuir | [CONTRIBUTING](../CONTRIBUTING.md) |

## Progressão de uso

```text
DIRECT → ASSISTED → GUIDED → ORCHESTRATED → WORKPLAN → MISSION / AUTOPILOT
```

Esses níveis descrevem uma progressão de supervisão e coordenação quando os
contratos do produto os expõem. Direct continua sendo válido por si só:
Composer, Flow e Mission não são pré-requisitos para abrir uma sessão.

## Referências técnicas

- [Runtime](architecture/runtime.md)
- [Persistência](architecture/persistence.md)
- [Identidade de filesystem](architecture/filesystem-identity.md)
- [Segurança](architecture/security.md)
- [Atualizações](architecture/updates.md)
- [ADR Docker host-parity](architecture/ADR-docker-compat-host-parity.md)
- [Glossário](GLOSSARY.md)
- [Guia editorial](STYLE_GUIDE.md)
- [Matriz de funcionalidades](reference/feature-matrix.md)
- [FAQ](faq.md)
- [Glossário](glossary.md)
- [O que mudou](whats-new.md)

## Público, interno e estado da funcionalidade

Esta pasta documenta o produto para usuários, operadores, integradores e
contribuidores. `DEV/` guarda contexto de engenharia, validações e histórico;
`.omx/` guarda artefatos internos de execução. Eles não são necessários para o
primeiro uso.

Cada página deve distinguir `IMPLEMENTED`, `EXPERIMENTAL`, `PLANNED` e
`UNKNOWN`. Código ou intenção futura não são evidência de uma capacidade
disponível.
