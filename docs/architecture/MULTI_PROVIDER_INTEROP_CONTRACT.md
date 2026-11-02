# ADR complementar — interoperabilidade de agentes Nexus ↔ Maestro

- **Data:** 2026-11-02
- **Natureza:** premissa arquitetural obrigatória; **implementação E2E ainda pendente**
- **Rastreabilidade:** [Nexus #81](https://github.com/IAPro-Community/nexus/issues/81), [#166](https://github.com/IAPro-Community/nexus/issues/166), [#176](https://github.com/IAPro-Community/nexus/issues/176), [Maestro #48](https://github.com/IAPro-Community/Orquestrador-Maestro/issues/48), [#53](https://github.com/IAPro-Community/Orquestrador-Maestro/issues/53)

## Invariante do produto

O Nexus permanece o **intermediador de aplicações e agentes**: entrada unificada, Composer, Flow, escolha explícita do executor por atividade, observabilidade, ações do operador, Browser e terminais interativos.

O Orquestrador Maestro continua **independente do Nexus**. Ele é a autoridade canônica de contexto, memória semântica, composição, skills e da execução/orquestração de runs delegados. Oferece a mesma capacidade de execução pelo CLI e pelos contratos externos, mesmo sem Nexus instalado.

A separação por owner é mandatória:

| Operação | Dono da verdade | Nexus |
| --- | --- | --- |
| Memória, contexto, composição do projeto | Maestro | Solicita e projeta; não duplica |
| DAG e atribuição de provider para cada tarefa delegada | Maestro, após revisão do operador | Editor e intermediador de intenções |
| Run delegado a AGY, OpenCode, Codex ou Claude | Maestro | Inicia por API pública e acompanha o run |
| Retry, cancel, resumption e Goal Loop de run Maestro | Maestro | Envia pedido e mostra recibo; não altera diretamente o estado |
| Terminal e sessão Direct de usuário | Nexus (PT Y/ConPTY) | Dono local explícito, independente do Maestro |
| Código e branches/worktrees | Git e checkout real | Visualiza, inspeciona e permite revisão |

**Nexus NÃO pode converter toda tarefa delegada em Codex** nem substituir AGY/OpenCode silenciosamente. A presença de Codex como única opção inicial de **planejamento read-only** no PR Maestro #51 é uma limitação de sandbox do *planejador*, não do *executor*. O provider de planejamento e o provider de execução são dimensões independentes.

## Fluxo de referência obrigatório

1. Usuário pede: «Antigravity implementa o frontend; OpenCode implementa a API; depois revisar a integração».
2. Nexus usa Composer para coletar/editar os requisitos; Maestro fornece o contexto e a composição canônica do projeto.
3. Flow apresenta as WorkPackages, atribuições **AGY/front-end** e **OpenCode/back-end**, dependências, critérios de aceite, permissões e orçamentos.
4. O usuário revisa a composição e as atribuições. Antes de GO, Nexus solicita ao Maestro validar capabilities/instalação dos dois providers, worktree e política.
5. Somente depois da aprovação, Maestro admite runs `execution_owner=MAESTRO`, com `taskId`, `providerId`, `model`, `runId`, `workspaceId`, `artifactSha`, `attemptId`, `approvalRef`, `limits` e `policyVersion` versionados. Atribuições devem sobreviver à reabertura do Nexus.
6. AGY e OpenCode operam em worktrees isoladas. Maestro coordena a execução, provas, bloqueios, cancel, retries e prevenção de conflito de checkout; Nexus agrega visualmente o estado.
7. Uma WorkPackage dependente só é admitida quando as dependências têm evidências aceitas. A fusão dos resultados requer revisão e não é automática.

## Garantias negativas

- Ausência de AGY, OpenCode ou modelo exigido: **BLOCKED** para a atividade afetada, sem fallback para Codex/cloud, sem inventar completion. As demais atividades independentes prosseguem conforme política.
- Sandbox read-only exigido onde o adaptador não o suporta: falhar explicitamente. Hoje `opencode` não anuncia `sandboxControl`; `agy` aceita sandbox genérico, mas não garante modo read-only. Isso não impede execução *autorizada para escrita* nos modos suportados.
- Queda do Nexus não encerra nem reinicia os runs Maestro; reconexão recupera estados do Maestro. Queda do Maestro não pode disparar `MissionRunner` Nexus no lugar dele.
- Nenhum provider ganha acesso a segredos ou outro worktree fora do contrato aprovado. Passar um nome de CLI não constitui validação de permissões.
- O Nexus pode abrir sessões Direct interativas com AGY/OpenCode por mecanismos próprios, mas devem ser exibidas como `owner=NEXUS` e não confundidas com jobs supervisionados pelo Maestro.
- O contrato de `runs.create` ainda é bloqueante no Bridge v1; `runs.subscribe` não está disponível. Não prometer automação durável até concluir Maestro #48.

## Verificação obrigatória e critério de retirada do código legado

E2E com dois providers reais (AGY e OpenCode) e duas worktrees, executor Maestro standalone, Nexus simultâneo, dois clientes, suspensão/retomada do Nexus, indisponibilidade seletiva de provider, cancelamento e terceiro passo dependente. Capturar run IDs, proofs, revisões de artefato e SHA de ambos repositórios.

Somente após essas provas e migração de dados, retirar a execução concorrente do `MissionRunner` para runs de owner Maestro. Não remover suporte a CLI, sessões interativas, providers nem comunicação Nexus ↔ apps em nome dessa migração.
