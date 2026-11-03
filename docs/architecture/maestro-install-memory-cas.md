# Etapa 00E — primitivo CAS para memórias INSTALL

**Estado:** interno, sem chamada por CLI, HTTP, Web, processo de migração ou startup; não é um restore apply.

`internal/core/config/install_memory_restore.go` introduz `ReplaceInstallIntelligenceMemoriesIfUnchanged(expectedBefore, replacement)` usando o `configMu` existente, com as seguintes regras:

- Exige `config.json` preexistente e regular; não cria uma instalação nova nem segue symlink do arquivo.
- Valida IDs/chaves/valores e rejeita duplicatas; compara campo a campo o conjunto INSTALL atual com o esperado, ignorando somente a ordem dos registros.
- Lê `config.json` em `map[string]json.RawMessage` e altera SOMENTE `intelligence_memories`; evita perder campos desconhecidos de versões futuras ou opções de contas/sessões.
- Escreve arquivo temporário 0600, sincroniza o conteúdo e renomeia para o arquivo de configuração. Retorna tamanho e checksum SHA-256, sem retornar valores de memória.
- O CAS é serializado por `configMu` somente dentro deste processo. O `SaveConfig` legado de outro processo não participa do mesmo lock; o método NÃO oferece isolamento multiprocesso nem coordenação transacional com SQLite.

## Bloqueios para ativar restore apply

1. Fencing de escritores legados em **todos** os processos (INSTALL e PROJECT) e confirmação de que nenhum writer não participante está ativo.
2. Journal durável para preservar a ordem das etapas; estado anterior com backup validado e hash de origem.
3. Persistência de `config.json` condicionada ao hash do arquivo/geração sob o mesmo mecanismo de exclusão utilizado por todos os writers.
4. Coordenação com `ReplaceProjectIntelligenceMemoriesTransactional` (PR #162) sem assumir transação distribuída; recuperação após crash em qualquer fronteira.
5. Testes em Linux e Windows, race, falhas simuladas após SQL commit e após config rename, E2E com duas instâncias e rollback compensatório auditável.
6. Autorizar apply explicitamente no Nexus apenas após homologação; não copiar os dados semanticamente para outro armazenamento.

## Testes adicionados, execução pendente

```bash
go test ./internal/core/config -run TestInstallMemoryCAS -count=1
go test -race ./internal/core/config -run TestInstallMemoryCAS -count=1
make lint:go
```

Casos: preservação de configuração desconhecida, CAS obsoleto, duas operações simultâneas no mesmo processo, arquivo ausente, symlink, JSON malformado, IDs/chaves inválidos ou duplicados, e remoção para conjunto vazio.

**Arquitetura:** este PR é independente de #156 e #162 e tem como destino exclusivo `release/v1.0.0`. Não há dependência com OMP nem nova memória.

**Validação:** Go 1.25 ainda indisponível no ambiente de execução; sem afirmação de testes passando, sem merge e sem alteração de comportamento ativo.
