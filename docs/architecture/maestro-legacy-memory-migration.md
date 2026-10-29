# Etapa 00C — inventário e backup da memória legada

**Estado:** candidato de implementação, sem importação, gravação no Maestro, cutover ou exclusão.  
**Base:** `release/v1.0.0`.  
**Depends on:** Nexus #86 / #113; Maestro #41 / PR #42; Nexus PR #155.

## Por que não migrar automaticamente

A memória legada existe em duas fontes distintas:
1. `config.json`, campo `intelligence_memories` de escopo **INSTALL**, global à instalação do Nexus;
2. SQLite, tabela `intelligence_memories` por **PROJECT**, incluindo possíveis linhas internas `pending.*` que não são memória promovida.

Já o Maestro identifica repositório Git, branch, worktree, commit e task; seu `repositoryId` não é o `projectID` interno do Nexus. Converter INSTALL em repository ou PROJECT em branch sem revisão muda a semântica e pode vazar preferências entre projetos. O primeiro incremento faz apenas **prévia e backup local**.

## Comandos locais de operação (sem importação)

```bash
# Prévia apenas de metadados, sem alterar memória existente:
nexus maestro memory-preview --project-id prj_EXAMPLE --json

# Backup explícito, com confirmação exigida:
nexus maestro memory-backup --project-id prj_EXAMPLE --confirm-local-backup --json
```

O backup só é gerado por invocação do operador. `nexus maestro memory-backup` sem `--confirm-local-backup` falha antes de abrir a store. O CLI abre a Store local diretamente, evitando iniciar o scheduler da instância Nexus Default. Ambos os comandos retornam JSON: a prévia inclui apenas metadados e estados; o recibo do backup informa caminho local, SHA-256 e tamanho, **não** o conteúdo. O `project-id` deve corresponder a um projeto local existente.

**Não existe comando `memory-migrate` nem `memory-restore` neste incremento.** Esses comportamentos dependem de contrato de escrita, mapeamento de escopo, verificação de idempotência e aprovação explícita. Não copiar o arquivo bruto para o repositório nem usá-lo como entrada sem revisar a política de privacidade.

## API interna de migração

- `PreviewLegacyIntelligenceMemoryMigration(ctx, projectID)` lê os dados atuais dos dois storages e retorna somente IDs, chaves, contagens, classes e digest SHA-256 do conjunto fonte; não retorna valores, fontes textuais nem paths de usuário.
- `BackupLegacyIntelligenceMemories(ctx, projectID)` é operação **explícita**, não conectada a rotas HTTP, install, update, login, migration runner ou inicialização. Cria snapshot exato com ambas as origens, inclusive `pending.*`, sob o diretório de dados do Nexus, em `migration-backups/legacy-intelligence-<random>.json`. Permissões 0700 do diretório e 0600 do arquivo; gravação temporária, sync, rename, digest e verificação após a escrita.
- `VerifyLegacyMemoryBackup(filePath, expectedSHA256)` valida digest, schema, tamanho e tipo do arquivo; não faz importação ou alteração de configuração/SQLite.
- `BuildLegacyMemoryMigrationPreview(snapshot)` é função pura, útil nos testes e em uma UI futura que mostre a decisão de escopo antes da aprovação.

**Atenção:** os backups contêm valores de memória legados sem truncamento para permitir restauração exata futura. Devem permanecer exclusivamente no diretório de dados privado, nunca serem anexados a issues, PRs, Support Bundles, logs, analytics, Evidence Bundles ou artefatos remotos.

## Estados da prévia

| Estado | Decisão |
|---|---|
| `SKIP_PENDING` | Preservar no backup; não importar como conhecimento consolidado |
| `BLOCKED_SENSITIVE_VALUE` | Bloquear automaticamente; exigir tratamento humano/redaction |
| `REVIEW_UNRECOGNIZED_KEY` | Chave fora do catálogo atual; não inventar mapeamento |
| `REVIEW_CONFLICT` | INSTALL e PROJECT contêm valores diferentes para a mesma chave; projetar precedência atual, pedir decisão |
| `REVIEW_GLOBAL_TO_PROJECT` | INSTALL não equivale a escopo de repositório no Maestro; exigir decisão |
| `REVIEW_PROJECT_TO_REPOSITORY` | Confirmar vínculo Nexus project → Git canonical repo/worktree no Maestro |

Nenhum desses estados implica que uma observação foi migrada.

## Próximo incremento 00D: contrato de importação após consentimento

1. Resolver `projectID -> CanonicalPath -> Maestro repositoryId` e verificar o remote/branch/escopo antes de emitir observações.
2. Implementar API de escrita no Maestro com aprovação explícita, nonce/idempotency key, origem, escopo, política redaction e validação de conflito. Sem editar `memory.jsonl` diretamente pelo Nexus.
3. Exigir backup validado, snapshot digest igual ao estado atual, dry-run completo e check de capacidades `memory.write` do Bridge.
4. Migrar por operação idempotente de observação com recibo/proveniência; verificar read-back pela API e determinar perdas/conflitos; NÃO apagar as fontes no mesmo passo.
5. Criar ferramenta de restauração **segura, explicitamente autorizada e testada** para repor exatamente o backup antes de desligar escritores legados. Comparar hashes e fazer rollback em falha parcial.
6. Trocar os readers/writers semânticos do Nexus somente com contratos aprovados, guardas de compatibilidade, E2E e snapshot/rollback. Até lá manter compatibilidade legada.

## Testes exigidos

```bash
go test ./internal/nexus -run 'Test(BuildLegacyMemory|LegacyMemory|LegacyMigration)' -count=1
go test -race ./internal/nexus
```

Validar ainda Windows/Linux/macOS, `NEXUS_DATA_DIR`, isolamento entre dois projects, arquivo manipulado, backup em diretório inseguro, parada durante snapshot, memória INSTALL versus PROJECT, chaves com `pending.*` e estratégia de conflito.

**Política de entrega:** nenhuma execução de testes foi afirmada neste documento. O PR continua draft/NO_GO enquanto não houver logs de testes no exact SHA.
