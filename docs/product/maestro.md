# Integração Nexus ↔ Orquestrador Maestro

## Contrato de CLI v1 — implementado, homologação global pendente

O Maestro expõe transporte `stdin JSON → stdout JSON` por meio de
`capabilities --json`, `skills list --json`, `advise --json` e
`verify --json`. Capabilities anuncia as versões de advice e verification
separadamente do número de versão do pacote. Advice usa o `SkillRouterV3` e
não executa skills. Verify consome checks concluídos pelo owner do runtime
(somente `PASS:` explícito), referências de evidência e sinais de risco; não
executa comandos nem abre as referências. O comando legado de instalação
`verify [flags]` continua disponível fora da forma exata `verify --json`.

O cliente Nexus falha em modo degradado se o contrato respectivo não estiver
anunciado, valida lifecycle/mode/verdict na resposta e rejeita intents com
padrão de segredo. Teste Go, vet e teste cross-repo passaram em `f2f29cd0`;
após a normalização mecânica Go, `make format-check` passou em `b5cfe715` e a
interoperação deve ser repetida nesse SHA. Isso não prova migração
de memória semântica, ownership de execução no caminho completo ou homologação
M9; ver [handoff corrente](../../DEV/HANDOFF.md).

> **Decisão arquitetural de 01/11/2026 (obrigatória para novos PRs):** o Maestro é a autoridade independente de contexto, memória semântica, composição, orquestração e execução dos runs delegados. Nexus consome apenas seu Bridge/API pública, sem integrar código interno, banco ou executor. As seções abaixo descrevem a **beta atual e sua transição**, não autorização para duplicar responsabilidades. Consulte [ADR Maestro externo](../architecture/MAESTRO_EXTERNAL_AUTHORITY_ADR.md) e [issue #176](https://github.com/IAPro-Community/nexus/issues/176).


O [Orquestrador Maestro](https://github.com/IAPro-Community/Orquestrador-Maestro)
é um produto independente: seu CLI, Protocol, Runtime e Bridge funcionam
sem Nexus. O Nexus não pode se tornar uma dependência do Maestro.

## Estado implementado em release/v1.0.0

**A integração ainda é opcional nesta branch.** O Nexus detecta a instalação,
consulta capacidades e skills, e integra determinadas políticas metodológicas.
`MAESTRO_DEGRADED` sinaliza indisponibilidade de capacidades suportadas.
O Nexus possui persistência semântica legada (`IntelligenceMemory`), que **ainda
não foi migrada**. Não declarar a arquitetura alvo como já implementada.

## Arquitetura alvo aprovada — ainda não entregue

- O Maestro permanece utilizável de forma autônoma, sem instalar o Nexus.
- O Nexus passa a distribuir/descobrir uma versão **compatível e obrigatória**
  do Maestro, sem modificar uma instalação independente do usuário sem consentimento.
- O Maestro é a única autoridade para memória semântica, contexto canônico
  (`AGENTS.md`, `DEV/`), skills, metodologia e políticas de verificação.
- Os mecanismos de memória adicionais escolhidos pelo usuário são conectados
  **através de adaptadores do Maestro**, não de novos bancos semânticos no Nexus.
- Nexus continua responsável por layout, terminais PTY/ConPTY, profiles,
  quotas, ResourceScheduler, supervisão das execuções que ele inicia e
  checkpoints **operacionais** de processo/handoff.
- Um Run tem exatamente um dono de execução (`NEXUS` ou `MAESTRO`).
  O Nexus não reinicia, cancela nem marca como concluído um Run criado no
  Runtime do Maestro sem encaminhar uma operação autorizada ao seu dono.
- A integração utiliza um contrato de contexto **versionado**, com escopo
  por projeto/branch/worktree/tarefa, orçamento, proveniência, digest e
  estado de atualização. Não compartilhar bancos internos nem duplicar
  motores de memória/execução.

## Integração incremental disponível na beta

O Maestro Bridge v1 passou a expor `context.brief` como operação aditiva
de leitura. O `ContextHandoff` do Nexus consulta esse método via `stdio`,
aceita somente o manifesto versão 1 com SHA-256 verificável e texto limitado
a 8.000 caracteres, e injeta o conteúdo apenas no prompt efêmero do destino.
O checkpoint operacional **não** passa a armazenar essa memória.

O runtime de destino registra `maestro_context=READY` quando a leitura passa
pelas validações, ou `maestro_context=DEGRADED` em instalações antigas,
ausentes ou indisponíveis. A degradação permanece explícita, e o provedor
destino deve reidratar os arquivos canônicos do projeto. Isso não constitui
prova de continuidade semântica completa nem a migração do
`IntelligenceMemory`.

## Cliente de contexto canônico via Bridge (incremento em avaliação)

A integração de handoff do Nexus foi ajustada para negociar `initialize`
(`PROTOCOL_VERSION=1`) e preferir `project.context` quando anunciado pelo
Maestro externo. O método retorna uma projeção mínima do projeto e reutiliza
o manifesto/digest do contexto canônico. Esta projeção é **efêmera**: Nexus não
a persiste como memória semântica.

Para instalações anteriores do Maestro com `contextBrief=true`, o cliente
pode usar `context.brief` explicitamente anunciado, sem inventar identidade
ou histórico de projeto. Versão desconhecida, manifesto/digest incompatível,
resposta fora do orçamento, indisponibilidade do processo ou método ausente
são erros; o handoff permanece `MAESTRO_DEGRADED` e não usa memória Nexus
como substituto silencioso.

**Limitação:** essa é uma integração de leitura no Context Handoff, não uma
migração completa do Composer/Flow, não habilita despacho pelo Maestro, nem
prova continuidade semântica dos demais caminhos. A adoção geral do projeto
depende de [Maestro #47](https://github.com/IAPro-Community/Orquestrador-Maestro/issues/47),
[Maestro #48](https://github.com/IAPro-Community/Orquestrador-Maestro/issues/48)
e [Nexus #176](https://github.com/IAPro-Community/nexus/issues/176).

## Caminho de migração seguro

1. Estender aditivamente o Bridge v1 do Maestro para operações de contexto
   (`context brief` já existe via CLI); negociar capacidades e versões.
2. Acrescentar adaptador Nexus e testes de compatibilidade sem alterar a
   instalação do Maestro independente.
3. Definir inventário, exportação e migração **reversível** de memórias
   `IntelligenceMemory` existentes. Não apagar hábitos, fatos ou projetos.
4. Ativar leitura via Maestro somente após testes de escopo, redaction,
   compatibilidade e recuperação; depois migrar gravações semânticas.
5. Tornar o componente obrigatório na distribuição do Nexus, com diagnóstico,
   atualização controlada e degradação explícita quando estiver indisponível.

Até concluir as etapas, **não bloquear Direct nem os terminais existentes**
somente porque o Maestro não está instalado: isso seria uma regressão sem
migração. Indisponibilidade de contexto semântico não permite inventar
continuidade confirmada.

## Segurança

A integração recebe somente contexto limitado e sanitizado, sem `.env`,
credenciais, auth files ou transcrições privadas completas por padrão.
Checkpoints do Nexus guardam identidade operacional e referência ao contexto;
não substituem o repositório semântico do Maestro.

Veja [Como funciona](how-it-works.md),
[Arquitetura](../architecture/overview.md) e
[estado de homologação](../release-status.md).
