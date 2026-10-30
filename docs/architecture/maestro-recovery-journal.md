# Etapa 00F — journal operacional de recuperação

**Status:** componente de recuperação preventiva, PR draft, sem `restore apply`, sem migração, sem substituição da memória do Maestro.

## Escopo implementado

- O pacote `internal/nexus/memoryrecovery` usa exclusivamente o DataDir já existente do Nexus. Ele não cria nenhuma memória semântica nem índice de conhecimento.
- `Begin` verifica nome e localização do backup legado, tamanho, permissões POSIX, SHA-256 do conteúdo, versão e projeto do snapshot. Exige também um digest de origem informado pelo chamador; esse segundo digest **não é revalidado no armazenamento ativo nesta etapa**.
- Uma única operação de recovery por diretório de dados pode possuir o arquivo `.active.lock`, criado com `O_EXCL` e vinculado a um ID aleatório de operação. O lock é **cooperativo entre coordenadores de recuperação**, não bloqueia automaticamente `UpsertIntelligenceMemory`, `SaveConfig` ou outros escritores legados de outros processos.
- Os eventos são arquivos imutáveis, sequenciais e privados (`0000.json`, `0001.json` etc.), com hash encadeado para detectar corrupção acidental ou edição sem recomputar a cadeia. Esse hash **não é assinatura/autenticação**.
- Transições permitidas: PREPARED → PROJECT_COMMITTED → INSTALL_COMMITTED → VERIFIED, com NEEDS_RECOVERY em pontos intermediários e ABORTED apenas antes de qualquer commit. Os estados registram alegações operacionais do futuro coordenador, não evidência automática de que SQL ou config foram realmente escritos.
- `Close` retira o lock apenas após VERIFIED/ABORTED, e somente se o lock ainda pertence à sessão. Para estados intermediários, lock permanece e toda operação nova falha; não há desbloqueio/repair silencioso.
- `Inspect` é somente leitura e marca bloqueio para lock órfão, journal incompleto ou corrompido. O comando `nexus maestro memory-recovery-status --json` expõe esse diagnóstico local, sem alterar nenhum registro.
- O journal **não inclui valores de memória**, apenas identificadores, etapas, horários e digests. Os backups privados existentes continuam sendo a única origem para recuperação.

## Limites de segurança obrigatórios

1. Não ativar restore apply: o arquivo INSTALL e o SQLite PROJECT não compartilham transação. Os componentes #162 e #163 fazem apenas operações CAS locais.
2. O atual lock não é universal: escrituras legadas continuam possíveis sem consultá-lo. Requer modo manutenção/fencing dos escritores em TODAS as instâncias, inclusive clientes remotos, e confirmação de quiescência antes de aplicar.
3. Exigir rollback automático/compensatório testado para crash entre SQL commit e config rename. O journal sozinho não executa rollback.
4. Retenção de locks órfãos é proposital: somente recuperação humana explicitamente autorizada, com verificação dos dois storages e geração do runtime, poderá liberar o lock.
5. O evento VERIFIED não pode ser registrado pelo futuro coordenador sem verificar hashes pós-commit e os dados lidos de ambas as origens. O pacote atual ainda não implementa essa integração.
6. Antes de produção, validar fsync e permissões no Windows, Linux e macOS; evitar NFS/unreliable filesystems se atomicidade do `O_EXCL` não puder ser garantida.

## Testes

```bash
go test ./internal/nexus/memoryrecovery -count=1
go test ./internal/app -run TestMaestroRecoveryStatus -count=1
go test -race ./internal/nexus/memoryrecovery ./internal/app
gofmt -w internal/nexus/memoryrecovery/*.go internal/app/maestro_memory_recovery*.go
```

Testes adicionados para encadeamento e replay, bloqueio de recuperações interrompidas, rejeição de transição ilegal, alteração de evento, vínculo do backup ao projeto, arquivo desconhecido, CLI read-only e argumentos inválidos. **Ainda não executados** no ambiente real do Nexus/Go 1.25.

## Atribuição de responsabilidades

- Maestro mantém a propriedade da memória semântica, contexto e compaction.
- Nexus mantém somente estado operacional de UI/runtime e o journal temporário para eventual migração/reversão.
- Nenhuma política de memória do Maestro é sobreposta por esse pacote.

## Próxima etapa 00G

Implementar e provar writer fencing em todos os pontos legados, incorporar recibos de operações reais nos eventos (com hashes), recuperar falhas intercaladas com crash injection, e só então expor comando de aplicação explicitamente confirmado. PRs de dependência: Nexus #156, #162 e #163; Maestro #42/#43; Nexus #155.
