# Pacote final para o owner — Nexus 1.0.1 (2026-11-22)

Estado: **NO_GO**. Base `release/v1.0.0` @ `a5a30a2c` (local; 3 commits Fase A já commitados sem push). Todo o restante está **não commitado** na working tree compartilhada, aguardando decisão do owner.

## 1. Commits candidatos (não commitados, prontos para fatiamento)

### A. Guardas de `execution_owner` (executor único NEXUS|MAESTRO)
- `internal/core/model/types.go`, migration `0040_execution_owner.sql`: coluna persistida, `NEXUS|MAESTRO` only, vazio = `NEXUS` (legado), desconhecido falha fechado; colunas autoritativas sobre `payload_json`; owner imutável pós-criação.
- `internal/nexus/runner/` (`runner.go`, `coordination.go`, `repository.go`, `types.go`): `MissionRunner` recusa run/plano `MAESTRO` ou owner desconhecido em start, step, cancel, pause, resume, intervention e coordenação (`SubmitCoordinationRequest`, `ResolveCoordinationRequest`, `MaterializeCoordinationWork` validam owner antes de lease).
- `internal/nexus/mission_executor.go` + `mission_integration.go`: `nexusPackageExecutor` recusa owner não local em Allocate, Compile, Execute, Review e IntegrateDelivery.
- `internal/nexus/mission_service.go`: takeover/return (`TakeControlMissionRun`, `ReturnMissionRun`) falham antes de parar worker/runtime/supervisor.
- Testes: `internal/nexus/runner/execution_owner_test.go`, `internal/nexus/mission_worker_test.go`, `internal/nexus/mission_executor_test.go` — sem lease/mutação em cenário negativo.

### B. Migração maestro-canonical-memory (#113/#176)
- `internal/nexus/memory_migration.go`: `PlanMemoryMigration` (dry-run read-only) e `MigrateIntelligenceMemoryToMaestro` (backup → journal → export → read-back → gate) + `RecoverMemoryMigration` (resume idempotente; digest divergente recusado). Copy-out: nunca apaga a fonte; ordem de side effects = backup → journal → gate (único efeito persistente, sempre por último).
- `internal/core/memoryrecovery/` (`gate.go`, `journal.go`, `resume.go`): gate fail-closed em `memory-migration-state.json`, journal com estágios, fence `.active.lock` cross-process.
- `internal/nexus/store/legacy_memory_gate.go` + `intelligence_memories.go`: writers falham com `ErrLegacyMemoryWritesDisabled` após o gate; leituras permitidas (histórico).
- `internal/nexus/intelligence_memory.go`: guarda also no caminho INSTALL (config.json).
- `internal/control/handoff/maestro_memory.go`: cliente Bridge `memory.record`/`memory.search` com capability `memory.write` (gated por `ORQUESTRADOR_MAESTRO_BRIDGE_MEMORY_WRITE=1`), redação e idempotência.
- Testes: `memory_migration_test.go` (dry-run, confirmação, happy path, falha parcial, recovery com `REPLAYED`, gate latched), `legacy_memory_gate_test.go`, suite handoff verde.

### C. CLI de migração
- `internal/app/maestro_memory_recovery.go`: `nexus maestro memory-migrate <plan|apply|recover> --workspace <path> [--reason <text>] [--confirm] [--json]`; `apply`/`recover` exigem `--confirm` + `--reason`.
- `internal/app/maestro.go` (dispatch) + `internal/app/app.go` (help).
- Testes: `maestro_memory_recovery_test.go` (invocações inseguras rejeitadas; plan não cria journal/backups).

### D. Fase B/C de CI (evidência por plataforma + montagem do gate)
- `ci.yml`: `native-windows`/`native-macos` produzem e publicam `nexus-v1-workspace-acceptance-windows-amd64|macos-arm64-<sha>` (setup-node adicionado); selo `native-certification` valida e publica `native-acceptance-records.jsonl`.
- `release.yml`: `same-sha-gate` monta o `evidence.json` (3 records + selo nativo) antes de `verify-agent-workspace-v1.mjs --release`; fail-closed em SHA/jobUrl/manifesto divergentes.
- `scripts/workspace-acceptance-e2e.test.mjs` 11/11; docs de aceite atualizados.

## 1b. Fatiamento verificado em worktree isolado e depois commitado na árvore real (2026-11-23)

Cada bloco foi aplicado sozinho e de forma cumulativa em worktree temporária (`/tmp/opencode/blockwt`, criada e removida; worktrees do owner intactas) e, após as decisões do owner, commitado na árvore real **por staging de índice** e enviado a `origin/release/v1.0.0`.

| Ordem | Commit publicado | Escopo | Build | Testes |
|---|---|---|---|---|
| 1 | `8b932b0c feat(go): enforce single execution owner on plans and runs` | 16 arquivos (13 alterados + migration 0040 + 2 testes novos) | `go build ./internal/...` OK | `internal/nexus/runner` 20.1s, `internal/nexus/store` 20.5s, `internal/nexus -run 'Mission\|ExecutionOwner'` 52.0s |
| 2 | `2af8d233 feat(go): add Maestro canonical memory migration with gate and recovery` | 11 arquivos (5 alterados + 6 novos) | OK | `internal/nexus` (migração), `store`, `memoryrecovery`, `core/config`, `handoff` verdes |
| 3 | `7b2a5a41 feat(cli): expose nexus maestro memory-migrate plan\|apply\|recover` | 4 arquivos | OK | `internal/app -run 'MaestroMemoryMigrate\|MaestroRecovery'` verde; `gofmt` limpo |
| 4 | `a90ace4b feat(ci): produce native platform records and assemble release evidence` | 4 arquivos (ci, release, teste, doc) | `go build` OK | `node --test` 11/11 ×3; YAML parseia; `verify-agent-workspace-v1` exit 0 |

Árvore publicada `3867a1fd` revalidada em worktree limpa: build OK, `gofmt` limpo, `runner`/`store`/`memoryrecovery`/`nexus`/`app`/`handoff` verdes, `node --test` 23/23, YAML válido, `verify-agent-workspace-v1` exit 0.

**Dependências:** A e B independentes; C depende de B; D independente. Ordem publicada A→B→C→D.

**Arquivos contaminados por outros workstreams (staging parcial aplicado, nunca `git add <arquivo>`):**

| Arquivo | Hunk alheio | Workstream | Tratamento |
|---|---|---|---|
| `internal/app/app.go` | help `handoff --no-attach`, aviso Codex quota, bloco `profilesCmd` | continuidade/quota | só as 2 linhas de help do `memory-migrate` entraram em C |
| `internal/nexus/store/autonomy.go` | `Seq int64` + `ListMissionRunEvents(runID, afterSeq, limit)` | paginação de eventos | só os hunks de owner + import `errors` entraram em A |
| `internal/nexus/mission_intervention_persistence_test.go` | `ListMissionRunEvents(run.ID, 0, 100)` | paginação de eventos | ficou fora de A (depende do paginador) |
| `.github/workflows/ci.yml` | step `terminal-golden-e2e` | #196 | fora de D; commitado à parte pelo owner em `3867a1fd` |

Commits concorrentes do owner na mesma janela: `d6b1d72e` (fix do produtor) e `3867a1fd` (step `terminal-golden-e2e`).

## 2. Estado das 48 capacidades

`docs/reference/agent-workspace-v1.json`: **43 `partial/pending`, 3 `present/pending`, 2 `missing/pending`** (`two-client-real-e2e`, `cross-platform-exact-sha`). Zero certificações; mapa de evidência: 21 `audited` / 76 `partial`, `coverage: unverified`. Verificadores: `verify-agent-workspace-v1.mjs` exit 0 (scope 48), `verify-agent-workspace-evidence-map.mjs` exit 0 (97 cláusulas).

## 3. Verificação executada (local, Linux)

- `go build ./internal/...`, `go vet ./internal/nexus/... ./internal/core/memoryrecovery/... ./internal/control/handoff/... ./internal/app/...`, `gofmt -l internal/` — limpos.
- `go test` verde: `internal/nexus` (migração), `internal/nexus/store`, `internal/core/memoryrecovery`, `internal/core/config`, `internal/control/handoff`, `internal/app` (suíte completa, 59.5s).
- Race verde: `internal/app` (CLI, 4.8s) e conjunto migração/gate/journal (`internal/nexus` 18.0s, `internal/nexus/store` 11.5s, `internal/core/memoryrecovery` 2.3s).
- Node: 23/23 nos scripts de evidence/contract; `docs-verify --links-only` exit 0 (256 Markdown).
- Dry-run do gate de release: 3 records sintéticos + `assemble` → 48 capacidades/3 plataformas; verificador recusou só por `source/integration not completed` (contrato ainda pendente) — a montagem não fabrica aceite.
- **Flake corrigido nesta rodada**: `internal/app` falhou 1/8 no filtro `MaestroMemoryMigrate|MaestroRecovery` (`unable to open database file (14)`) porque o singleton `nexus.Default()` guardava o store de um `NEXUS_DATA_DIR` temporário removido. Isolamento com `nexus.ResetDefaultForTest()` + `t.Cleanup` nos testes do CLI; reexecução **60/60 verdes**.

## 4. Decisões do owner e action items externos restantes

**Decidido pelo owner em 2026-11-23 e já executado:**
1. **Fatiamento + commits** — A→B→C→D, um commit por bloco, sem arrastar hunks alheios (staging por índice). Publicados: `8b932b0c`, `2af8d233`, `7b2a5a41`, `a90ace4b`.
2. **Push** — `git push origin release/v1.0.0` aceito (`6c322427..3867a1fd`, fast-forward); tag `v1.0.0` intacta; CI do push: run `45165787696`.
3. **Cutover da memória** — adiado para o SHA publicado com Bridge Maestro ativo; **nenhum cutover local executado**.

**Ainda bloqueado em fatores externos (owner/runners):**
4. **Workflow_dispatch nativo** — `native_target_sha=3867a1fd` + `native_platform=all` nos runners `nexus-windows`/`nexus-macos`; sem runner Windows o selo nativo não fecha (bloqueio #216). Sem os 3 registros reais, o `same-sha-gate` não produz `evidence.json` válido.
5. **Fechamento das 48 capacidades** — `present`+`verified` só com os 3 registros do SHA publicado; decisão do owner no SHA final, sem edição automática.
6. **Demais bloqueios intocados** — quota real (evidência atual é sintética), NEX 2/11, assinatura/`maestroSha`, credencial do Named Tunnel, aceite visual HUD+Site, sessões físicas do maestro (kits 1–5).

## 5. Faixa de risco assumida

- Working tree compartilhada com ~145 arquivos modificados de vários workstreams: o fatiamento precisa ser por índice (`git add -p` / `update-index --cacheinfo`), nunca `git add` de arquivo inteiro (precedente: `b035aa04`/`b81e63bf`).
- Nenhum status de capacidade foi alterado para fechar aceite nesta onda; nenhuma tag movida; nada publicado.
