# ADR — Nexus como cliente do Maestro independente

- **Data:** 2026-11-01
- **Decisão de produto:** aprovada; código da release **ainda não está conforme**
- **Owner do contrato:** [Orquestrador-Maestro](https://github.com/IAPro-Community/Orquestrador-Maestro)
- **Issue de migração:** [Nexus #176](https://github.com/IAPro-Community/nexus/issues/176)
- **Contrato de origem:** [Maestro #45](https://github.com/IAPro-Community/Orquestrador-Maestro/issues/45)

## Invariantes

1. **Maestro deve funcionar plenamente sem Nexus.** Projetos, contexto, memória, Skills, composição, workflows, orquestração, runtime de runs e verificação pertencem ao Maestro.
2. Nexus é cliente via Bridge/API pública versionada; não importa módulos internos do Maestro, não copia seus executores nem lê/escreve diretamente seu banco/RunStore.
3. O Composer Nexus é UI para descoberta/revisão do PromptArtifact; o Flow Nexus é UI de plano/DAG, dependências e evidências. **A compilação semântica e as políticas de execução/Goal Loop de trabalhos delegados devem usar serviços do Maestro.**
4. Um run possui `execution_owner` imutável durante sua vida: `MAESTRO` ou `NEXUS`. Quando `MAESTRO`, nunca despachar, retentar, cancelar ou concluir por `MissionRunner` local; encaminhar operações ao Maestro por contrato.
5. Terminais PTY/ConPTY, Browser, janelas, layout, notificações e preferências visuais pertencem ao Nexus. Essa persistência operacional não constitui memória semântica concorrente.
6. Se Maestro estiver indisponível, informar erro/capability degradada; não criar fallback cloud, planner/memória paralelos ou confirmação fictícia. Direct/PTY genuinamente independentes continuam funcionando.
7. Migração de legado exige inventário + backup + digest + CAS/journal/recovery + testes cross-process + rollback antes do cutover. Não excluir memória anterior até prova de paridade.

## Contrato de abertura de projeto

1. Detectar uma instalação **executável** e versão compatível do Maestro; a existência de diretório `.orquestrador` não prova disponibilidade do Bridge.
2. Inicializar Bridge com raiz canônica configurada. Negociar `initialize` e capabilities e selecionar o método suportado: preferir projeção `project.context` após sua adoção; em versões antigas usar `context.brief` apenas quando explicitamente suportado e sem simular dados ausentes.
3. Validar escopo da raiz/branch/worktree/tarefa, manifesto, digest, provenance, budget e redaction antes de apresentar dados. Não presumir que identidades iguais de caminho implicam mesma worktree.
4. Pedir ao Maestro a composição canônica para a sessão; Nexus só apresenta, revisa e envia alterações aprovadas pelos contratos permitidos.
5. Se o Bridge cair ou houver mudança de projeto/worktree, descartar projeções stale e obter estado novo antes de GO. Nunca inferir completion de ausência de evento.

## Inventário para arquivamento/migração (não remover antecipadamente)

| Local existente | Diagnóstico | Destino | Critério de retirada |
| --- | --- | --- | --- |
| `internal/nexus/intelligence_memory.go` | Memória semântica legada | Maestro Memory API | Import, read-back, preservação de escopo, rollback, writers desativados |
| `internal/nexus/flow_decomposition.go` | Heurística/planificação própria | Maestro project composition | Mesma qualidade de plano/DAG, sem fallback genérico |
| `internal/nexus/plan.go` (`GeneratePlanFromIntent`, `MaterializePromptArtifactAsFlow`) | Composição por Nexus Intelligence | Maestro compose/plan + UI Flow | Preservar PromptArtifact, hashes, aprovações e aceites |
| `internal/nexus/mission_executor.go` + MissionRunner | Execução local concorrente se owner MAESTRO | Maestro Run API | Bloquear dispatch local para run Maestro, eventos/evidence/recovery confirmados |
| `MemoryFactsMap` e readers afins | Projeção que pode voltar ao legado | Maestro context/memory Bridge | Consumidores migrados, sem fallback silencioso |

## PRs e issues que devem ser reorientados

- [#168](https://github.com/IAPro-Community/nexus/pull/168) e [#169](https://github.com/IAPro-Community/nexus/pull/169): **HOLD / REPLACE-CANDIDATE** para mecanismo semântico Nexus; reaproveitar validação de DAG, UI e proveniência, sem merge da parte que compete com Maestro. [Issue #166](https://github.com/IAPro-Community/nexus/issues/166).
- [#170](https://github.com/IAPro-Community/nexus/pull/170): **HOLD / REPLACE-CANDIDATE** para retry/goal loop sob MissionRunner. Diagnóstico read-only pode ser reaproveitado como projeção. [Issue #167](https://github.com/IAPro-Community/nexus/issues/167).
- [#155](https://github.com/IAPro-Community/nexus/pull/155), [#156](https://github.com/IAPro-Community/nexus/pull/156), #162–#165: **MIGRATION-ONLY**; componentes de backup, CAS e journal não são um serviço de memória semântica definitivo. [#113](https://github.com/IAPro-Community/nexus/issues/113).
- [#154](https://github.com/IAPro-Community/nexus/pull/154): reconciliar roadmap com este ADR. OMP/NEX não fazem parte deste escopo.

Nenhum desses caminhos é apagado antes de comprovada migração/rollback. Arquivar significa retirar da rota de entrega até adaptar ou substituir; não significa perda de trabalho nem fechamento de issue de negócio.

## Critérios mínimos de homologação

- Maestro isolado sem Nexus, e Nexus + Maestro no mesmo projeto/worktree, contexto/decisões coerentes.
- SHA exato nos dois repos, testes de contratos de contexto, versão e permissões, mais dois clientes.
- Missão delegada sempre com execução e evidência de origem Maestro; nenhuma duplicação local de retry/dispatch.
- Reabertura do Nexus não cria novo run para tarefa já aceita; falhas e dados stale são explícitos.
- Testes Linux e Windows, restauração/migração com registros existentes, sem perda de dados.

**Status atual:** especificação aprovada; integração e migração ainda não homologadas. Não alegar conclusão por PR documental.
