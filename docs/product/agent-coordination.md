# Coordenação entre Agents

> **Objetivo:** permitir que um Agent peça informação, artefato, revisão ou
> delegação durante uma Mission sem criar uma malha peer-to-peer nem um segundo
> orquestrador.

![Coordenação entre Agents no Nexus v1](../assets/diagrams/nexus-v1-agent-coordination.svg)

## Contrato

Agents não iniciam providers, Agents, Missions ou Flow nodes diretamente. Eles
submetem uma solicitação tipada ao `MissionRunner`, que mantém a autoridade
sobre estado, policy, persistência e materialização de trabalho.

As operações fechadas da v1 são:

| Operação | Uso |
| --- | --- |
| `NEED_INFO` | obter uma informação necessária para continuar |
| `ARTIFACT_REQUEST` | reutilizar ou produzir um artefato delimitado |
| `REVIEW_REQUEST` | solicitar revisão independente |
| `DELEGATE` | executar trabalho filho claramente delimitado |

## Resolução local primeiro

Antes de acordar outro Agent, o Nexus tenta resolver o pedido com contexto já
durável:

1. estado da Mission;
2. `WorkReceipt` existente;
3. artifact conhecido;
4. `IntelligenceMemory`;
5. somente então uma child package.

Essa ordem reduz custo, latência e ruído visual no Flow.

## Quando uma child package é criada

A child package:

- mantém lineage para a package e Agent de origem;
- carrega requisitos de capacidade para o scheduler;
- pode registrar preferência de provider sem transformá-la em `PIN`;
- aparece na execução do Flow;
- produz seu próprio `WorkReceipt`;
- fecha a solicitação de coordenação quando termina.

O Agent de origem passa a depender desse trabalho e recebe a resposta através
do novo `ContextCapsule`.

## Continuação segura

O Nexus v1 não depende de “reabrir” magicamente a mesma chamada do provider.
Após uma solicitação concluída, a retomada usa uma nova execução sob a **mesma
identidade persistente de Agent**, com o receipt incorporado ao contexto.

A retomada automática só é aceita quando existe prova positiva de segurança:

- fingerprint do workspace não mudou;
- Git HEAD não mudou;
- não há tracked/index/untracked delta novo;
- o contrato proíbe side effects externos relevantes.

Se essa prova não existir, a Mission entra em intervenção humana em vez de
repetir trabalho potencialmente perigoso.

## Idempotência

O ID semântico da solicitação impede que restart/replay crie duas child
packages equivalentes. A persistência mantém:

- request;
- source package;
- source Agent;
- child package;
- receipt;
- resolução;
- timestamps.

## Limites

Para evitar fan-out acidental por saída de modelo, o runner aplica limites de
quantidade e tamanho de payload. Exceder o limite falha fechado.

## Mudança de plano

Coordenação dinâmica pode resolver execução, mas não pode alterar silenciosamente:

- objetivo;
- escopo;
- critérios de aceite;
- arquitetura autorizada;
- permissões/side effects relevantes.

Mudanças materiais precisam de revisão do plano ou intervenção.

## Especificação técnica

A especificação detalhada está em
[`DEV/SPECS/CONVERSATIONAL_MISSION_COORDINATION.md`](../../DEV/SPECS/CONVERSATIONAL_MISSION_COORDINATION.md).
