# Integração Nexus ↔ Orquestrador Maestro

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
