# Validação 00G — Writer fencing / CAS de memória

**Status: NO_GO.** Este arquivo é uma matriz de testes **a executar** em uma cópia isolada da instalação, não um relatório de testes executados. Branch candidata: `feat/maestro-memory-writer-fencing-20261029`; destino exclusivo: `release/v1.0.0`. Dependência: PR #164 (journal).

## Pré-condições

Use Go 1.25 conforme `go.mod`, checkout do SHA exato e um DataDir/ConfigDir temporário. Nunca executar testes que escrevem memórias em dados reais de usuário. Não usar GitHub Actions no processo de homologação; registrar resultados no PR.

```bash
go version
gofmt -l internal/core/memoryrecovery/*.go internal/core/config/intelligence_memory_baseline*.go internal/core/config/memory_recovery_fence_test.go internal/core/config/config.go internal/nexus/store/intelligence_memories*.go internal/nexus/store/store.go internal/app/maestro_memory_recovery*.go
go test -count=1 ./internal/core/memoryrecovery ./internal/core/config ./internal/nexus/store
go test -race -count=1 ./internal/core/memoryrecovery ./internal/core/config ./internal/nexus/store
go test -count=1 ./internal/app -run '^TestMaestroRecoveryStatus'
go vet ./internal/core/memoryrecovery ./internal/core/config ./internal/nexus/store ./internal/app
```

Qualquer arquivo listado pelo `gofmt -l`, comando com exit != 0, teste falho ou ausência de Go1.25 bloqueia merge. Registrar SO, SHA, versão Go, resultados completos e diff de formatação aplicado.

## Matriz mínima de duas instâncias

| Cenário | Evidência exigida |
| --- | --- |
| Duas instâncias no mesmo DataDir | Exclusão entre processos comprovada; uma única gravação por vez |
| SAVE de config durante PREPARED | `SaveConfig` falha sem modificar `config.json` |
| SQL Upsert/Delete durante PREPARED | Falha sem alterar linhas em `intelligence_memories` |
| Config carregada antes de atualização legítima | Snapshot CAS rejeita sobrescrita tardia; conteúdo atual preservado |
| Lock órfão ou evento corrompido | Operação bloqueada, nenhum desbloqueio automático, CLI retorna código não zero |
| Lock escrito por outro processo | `Close()` não remove lock com owner diferente |
| Falha entre Write/Sync/Rename | Nenhum arquivo parcialmente gravado aceito como config válida |
| Linux e Windows | Sem bypass de exclusão sob regras distintas de filesystem |
| Instância Nexus antiga ou SQL externo | Detectar explicitamente que o fence NÃO é universal; proibir restore apply |
| ConfigDir e DataDir distintos/alias | Garantir correspondência do store ativo e fence; proibir restore apply quando divergentemente configurados |

## Gate para 00H (coordenador real)

1. Implementar modo manutenção com confirmação, interrupção e prova de quiescência de todos os writers, inclusive binários de versões anteriores.
2. Integrar backup do estado ativo antes da aplicação e recibos de commit SQL/config com comparação de hashes.
3. Implementar journal de compensação com crash injection e recuperação entre os estados PROJECT_COMMITTED e INSTALL_COMMITTED.
4. Registrar testes em Linux, Windows e macOS. Verificar proteção de dados sensíveis/escopos e ausência de nova store semântica.
5. Só depois oferecer `restore apply` com autorização explícita e rollback verificado. Sem fallback silencioso.

**A memória canônica continua no Maestro.** O Nexus armazena exclusivamente seu estado operacional, até conclusão da migração aprovada.
