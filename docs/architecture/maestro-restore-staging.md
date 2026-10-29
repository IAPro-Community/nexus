# Etapa 00E — validação e preparação de restauração de memória legada

**Estado:** candidato; **nenhuma restauração aplicada**, nenhuma migração automática.  
**Base:** PR Nexus #156, branch `feat/maestro-legacy-memory-migration-preview-20261029`.

A memória legado Nexus pode existir em `config.json` (escopo INSTALL) e no SQLite (escopo PROJECT). Não há commit atômico compartilhado entre esses dois storages. Uma ferramenta que sobrescrevesse ambos sem journal, lock e rollback próprio poderia criar um estado parcialmente restaurado.

## Implementação de segurança já registrada

- `readLocalLegacyRestoreBackup(filePath, sha256)`: aceita somente backups previamente criados no diretório privado `<Nexus DataDir>/migration-backups`, com nome esperado, checksum SHA256, 0600 em sistemas POSIX, schema suportado, IDs únicos e todas as linhas PROJECT ligadas ao projeto correto.
- `PreviewLegacyMemoryRestore(ctx, projectID, backupPath, sha256)`: associa backup ao `CanonicalPath` atual do projeto, compara digest de origem com a memória legada ativa e expõe contagens/metadados mais a flag `source_drift`. Nunca retorna valores de memória nem modifica os storages.
- `StageLegacyMemoryRestore(ctx, projectID, backupPath, sha256, confirmed)`: exige confirmação explícita, revalida backup e contexto e grava apenas uma *intenção operacional* em `migration-restore-intents` no diretório de dados local. A intenção contém metadados, não observações, e informa status inequívoco `STAGED_NO_APPLY`.
- Testes unitários de identidade, checksum, drift, autorização, ausência de modificação e contenção de segredos.

**Ainda não existe um comando CLI conectado para invocar a intenção de restore** neste incremento; o ponto de entrada deve ser integrado e testado antes da homologação. Os comandos de inventário e backup de #156 continuam separados.

## Antes de implementar `restore apply`

1. Obter lock exclusivo sobre os escritores de INSTALL e SQLite, impedindo novas gravações dos agentes enquanto o procedimento ocorrer. Não copiar semântica para outra memória.
2. Criar um snapshot de segurança *do estado atual* antes de restaurar o backup antigo.
3. Registrar journal de operação com geração, identificadores de projeto, digests antes/depois e estado `PREPARED`.
4. Implementar transações SQLite para todo o conjunto PROJECT com IDs e timestamps originais; validar read-back antes de avançar.
5. Salvar a configuração INSTALL via mecanismo atômico já existente, com comparação de versão e rollback compensatório. Simular crash entre passos.
6. Só marcar `RESTORED` após os dois storages conferirem os digests esperados. Em qualquer falha, manter journal recuperável.
7. Construir decisão explícita do operador para conflito entre dados atuais e backup; nunca impor precedência silenciosa.
8. Registrar testes reais Go 1.25 em Linux e Windows e E2E com dois clientes. Manter #156 draft até então.

A etapa seguinte de migração para Maestro depende também da escrita opt-in do PR Maestro #43, de mapeamento de escopo e de teste de leitura posterior. A restauração legado não é um substituto da memória canônica.
