# Documentação do IAPro Nexus

Esta é a entrada pública para entender, instalar e operar o Nexus. O produto
é uma workstation local para coding agents; a documentação parte do caminho
mais simples e só depois apresenta Composer, Flow e Mission.

## Escolha seu caminho

| Quero… | Comece em |
| --- | --- |
| entender o produto | [Visão geral](product/overview.md) e [modelo mental](product/mental-model.md) |
| ter meu primeiro resultado | [Getting Started](getting-started/installation.md) |
| abrir um agente diretamente | [Fluxo Direct](product/direct.md) |
| entender Web, Desktop e CLI | [Superfícies](product/surfaces.md) |
| operar agentes e terminais | [Agentes](product/agents.md) e [terminais](product/terminals.md) |
| usar Composer | [Composer](product/composer.md) |
| revisar um plano no Flow | [Flow](product/flow.md) |
| executar uma Mission | [Missions](product/missions.md) |
| entender Maestro | [Integração Maestro](product/maestro.md) |
| entender a implementação | [Arquitetura](architecture/overview.md) |
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
- [Glossário](GLOSSARY.md)
- [Guia editorial](STYLE_GUIDE.md)

## Público, interno e estado da funcionalidade

Esta pasta documenta o produto para usuários, operadores, integradores e
contribuidores. `DEV/` guarda contexto de engenharia, validações e histórico;
`.omx/` guarda artefatos internos de execução. Eles não são necessários para o
primeiro uso.

Cada página deve distinguir `IMPLEMENTED`, `EXPERIMENTAL`, `PLANNED` e
`UNKNOWN`. Código ou intenção futura não são evidência de uma capacidade
disponível.
