# Etapa 00E — restauração SQLite com transação e CAS

**Status:** bloco interno, não conectado à CLI ou Web. Não restaura config.json nem ativa restore apply.

## Entrega

O método `Store.ReplaceProjectIntelligenceMemoriesTransactional(ctx, projectID, expectedBefore, replacement)` executa um restauro limitado ao SQLite:

- Valida vínculo de projeto, IDs, chaves, valores e timestamps, rejeitando duplicatas e linhas de outros projetos.
- Abre uma `sql.Tx`, verifica dentro dela a igualdade integral do estado atual com o snapshot esperado (CAS) e aborta diante de drift.
- Substitui exclusivamente linhas do projeto em uma transação DELETE + INSERT, preservando IDs, timestamps, valores e marcadores `pending.*`.
- Valida o resultado por read-back antes do commit e usa rollback automático em erro/cancelamento.
- Não altera INSTALL (config.json), não armazena memória canônica no Nexus e não acessa o Maestro.

## Bloqueios antes de ativar restauração real

INSTALL fica em config.json; PROJECT fica no SQLite. Transação SQL não engloba o arquivo de configuração. O `restore apply` exigirá:

1. Fencing de todos os escritores de memória, inclusive outros processos, sem confiar em locks desconhecidos pelos caminhos legados.
2. Backup íntegro do estado atual, hash e associação segura ao `projectID` e ao `CanonicalPath`.
3. Journal durável com estados PREPARED / PROJECT_COMMITTED / INSTALL_COMMITTED / VERIFIED / NEEDS_RECOVERY.
4. CAS e atomic rename para atualizar somente `intelligence_memories` no config.json, preservando outras configurações.
5. Testes de crash em cada etapa, compensação segura e impossibilidade de sobrepor concorrência não reconhecida.
6. Confirmação explícita do operador no momento da aplicação, depois da revalidação dos hashes.

Até passar esses gates, `memory-restore-preview` e `memory-restore-stage` permanecem não destrutivos; este primitivo não é invocado pela CLI.

## Testes propostos e implementados, ainda não executados

```bash
go test ./internal/nexus/store -run TestRestoreProjectIntelligence -count=1
go test -race ./internal/nexus/store -run TestRestoreProjectIntelligence -count=1
make lint:go
```

Casos: fidelidade das linhas, isolamento entre projetos, pending.*, CAS obsoleto, duplicação e rollback provocado por colisão de primary key depois do DELETE. Exigir Go 1.25, validação Linux/Windows e registro do SHA antes de merge.

**Integração:** Esta mudança pode entrar isoladamente na `release/v1.0.0`; a implementação de backup/preview do Nexus PR #156 é separada e ainda não homologada.
