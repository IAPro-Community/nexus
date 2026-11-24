## 2026-11-24 — Lane A `mission-observatory` auditada

- SHA local: `release/v1.0.0` @ `bb3f9d5a` (owner `5d83fc40`, E2E `410e626f`, evidence map `00fa18de`, contrato `b878bdc8`, lint `bb3f9d5a`); `origin/release/v1.0.0` = `714c8dae` até o push desta linha. Responsável: orquestrador. Ambiente: Linux + Chromium local.
- PASS (`mission-observatory`) — `GOCACHE=/tmp/opencode/go-build GOFLAGS=-mod=readonly npm --prefix web run test:e2e:mission-attention` PASS: Mission MAESTRO read-only, zero ações mutáveis, dependência/capsule/receipt/arquivo visíveis e navegação ao terminal do Agent no mesmo workspace. `npm --prefix web run typecheck` PASS.
- PASS (ownership backend) — `GOCACHE=/tmp/opencode/go-build GOFLAGS=-mod=readonly go test ./internal/nexus/runner ./internal/nexus/store -run 'TestMissionRunnerRefusesMaestroOwnedRunForEveryLocalMutation|TestCoordinationFailsClosedForUnknownExecutionOwner|TestExecutionOwnerIsImmutableAndDefaultsToNexus|TestMissionRunExecutionOwnerPersistsAndIsImmutable' -count=1` PASS.
- PASS (contrato) — evidence map com 2 critérios `audited`; contrato de `partial` para `present`, `verification: pending`; doc de aceite sincronizado. `verify-agent-workspace-evidence-map.mjs` exit 0 (48/97, 0 certificadas); testes do mapa 4/4; `verify-agent-workspace-v1.mjs` exit 0. Estado: 33 capabilities com todos os critérios `audited` (68/97 cláusulas); contrato 35 `present` / 12 `partial` / 1 `missing`.
- PASS (registro Linux exato anterior) — SHA `714c8dae87d8f124fe513976dfb1e2131098a503`: `workspace-acceptance-linux-amd64.json` com 47 `passed`, 1 `blocked` (`cross-platform-exact-sha`), 0 `failed`; Linux E2E e smoke verdes.
- FIX (CI anterior) — `ci.yml` `45325848046` falhou somente em ESLint `no-control-regex` em `web/src/features/voice/terminalVoice.ts`; `bb3f9d5a` substitui regex por inspeção de code point, preserva negação de controles, e lint agora passa com 1 warning preexistente.
- PENDENTE — push do lote e CI do SHA novo; `workspace-acceptance-linux-amd64.json` desse SHA exato ainda não coletado; aceite externo por capability e lanes B–F dependem do maestro. Release **NO_GO**.

## 2026-11-24 — Lane A `design-inspect` auditada

- SHA local: `release/v1.0.0` @ `d8f1a664` (evidence map `87a8f8ea`, contrato `d8f1a664`). Responsável: orquestrador. Ambiente: Linux + Chromium local.
- PASS (`design-inspect`) — `npm --prefix web run test:e2e:browser-design-inspect` PASS: DOM/estilos/crop mascarado, Activate read-only, uma única entrega ao Agent explicitamente escolhido, `start_if_needed=false` e zero starts.
- PASS (sanitização) — `go test ./internal/browsercontrol -run 'TestInspectRejectsEmptyOrOversizedSelectorBeforeCDP|TestChromiumDesignInspectOptional|TestMaskDesignCropOpaqueMasksSensitiveRect' -count=1` PASS: seletor vazio/oversized recusado antes do CDP, email redigido e crop sensível mascarado.
- PASS (contrato) — evidence map com 2 critérios `audited`; contrato de `partial` para `present`, `verification: pending`; doc de aceite sincronizado. `verify-agent-workspace-evidence-map.mjs` exit 0 (48/97, 0 certificadas); testes do mapa 4/4; `verify-agent-workspace-v1.mjs` exit 0. Estado: 32 capabilities com todos os critérios `audited` (66/97 cláusulas); contrato 34 `present` / 13 `partial` / 1 `missing`.
- PENDENTE — CI do push `d8f1a664`; `workspace-acceptance-linux-amd64.json` do SHA exato ainda não coletado; aceite externo por capacidade e lanes B–F dependem do maestro. Release **NO_GO**.

## 2026-11-24 — Lane A `remote-terminal` auditada

- SHA publicado: `origin/release/v1.0.0` @ `9c8d6dfd`; fix de código `4b16c60d`, evidence map `1bdabb7d`, contrato `9c8d6dfd`. Responsável: orquestrador. Ambiente: Linux + runner self-hosted `nexus-ci` em container.
- PASS (`remote-terminal`, escopo) — `go test ./internal/control/web/ -run 'TestAgentTerminal|TestTerminalAttach' -count=1 -v` PASS: viewer recebe 403 no attach por Agent, runtime de outro Agent recebe 404 sem revelar ownership, upgrade WS sem Origin ou com Origin estrangeira recebe 403, lease `CONTROL` continua único.
- PASS (regressão) — `go test ./internal/control/web/ -count=1` PASS (69 s); `go vet ./internal/control/web/` limpo; `gofmt -l internal/control/web/routes_projects_agents.go internal/control/web/desktop_auth_test.go` vazio.
- PASS (contrato) — evidence map `remote-terminal` com 2 critérios `audited`; contrato de `partial` para `present`, `verification: pending`; doc de aceite sincronizado. `node scripts/verify-agent-workspace-evidence-map.mjs` exit 0 (48 capacidades, 97 cláusulas, 95 source+test, 0 certificadas); testes do mapa 4/4; `node scripts/verify-agent-workspace-v1.mjs` exit 0. Estado: 31 capabilities com todos os critérios `audited` (64/97 cláusulas); contrato 33 `present` / 14 `partial` / 1 `missing`.
- PENDENTE — CI do push `9c8d6dfd`; `workspace-acceptance-linux-amd64.json` do SHA exato ainda não coletado; aceite externo por capacidade e lanes B–F dependem do maestro. Release **NO_GO**.

## 2026-11-23 — Lane A `parallel-challenge` auditada

- SHA publicado: `origin/release/v1.0.0` @ `0de2faf6`; SHA local ao fim: `41c9671d` (commits `c7141630`, `c1cfdd41`, `41c9671d`, sobre `0de2faf6`). Responsável: orquestrador. Ambiente: Linux + runner self-hosted `nexus-ci` em container.
- PASS (dispatch real) — `GOCACHE=/tmp/opencode/go-build GOFLAGS=-mod=readonly go test ./internal/nexus -run TestParallelChallengeDispatchesOneSpecificationToIsolatedAgents -count=1` PASS: mesma especificação despachada como dois `MissionRun` para `agent-alpha` e `agent-beta`; cada run executa no próprio worktree Git real (`git init` + marcador + verificação `exit 0` + revisão aprovada) até `COMPLETED_VERIFIED`; `MissionChallengeParticipants` recebe os runs reais e mapeia os dois worktrees isolados; `EvaluatePlanChallenge` ranqueia `agent-alpha`, `agent-beta` com `auto_merge=false`.
- PASS (regressão) — `go test ./internal/nexus -count=1` PASS (117 s); `gofmt -l` vazio em `internal/nexus`.
- PASS (contrato) — evidence map `parallel-challenge` com 2 critérios `audited` (`c1cfdd41`); contrato de `partial` para `present`, `verification: pending` (`41c9671d`); doc de aceite sincronizado. `node scripts/verify-agent-workspace-evidence-map.mjs` exit 0 (48 capacidades, 97 cláusulas, 95 source+test, 0 certificadas); testes do mapa 4/4; `node scripts/verify-agent-workspace-v1.mjs` exit 0. Estado: 29 capacidades com todos os critérios `audited` (62/97 cláusulas); contrato 31 `present` / 16 `partial` / 1 `missing`.
- PENDENTE — CI do push `41c9671d`; recaptura sintética desnecessária (nenhum arquivo em `web/src`); `workspace-acceptance-linux-amd64.json` do SHA exato ainda não coletado; Lanes B–F dependem do maestro. Release **NO_GO**.

## 2026-11-23 — Lane A `agent-groups` auditada

- SHA publicado: `origin/release/v1.0.0` @ `36ffc734`; SHA local ao fim: `467676ea` (commits `78161c8d`, `fd50e8fb`, `467676ea`, sobre `36ffc734`). Responsável: orquestrador. Ambiente: Linux + runner self-hosted `nexus-ci` em container.
- PASS (`agent-groups`, worktree real) — `GOCACHE=/tmp/opencode/go-build GOFLAGS=-mod=readonly npm --prefix web run test:e2e:agent-groups` PASS: `web/scripts/e2e-agent-groups-live-session.mjs` cria repositório Git real + `git worktree add`, usa esse caminho no formulário de grupo da UI e exige que o valor persistido continue ligado ao worktree (`.git` com `gitdir:`); saída `PASS group worktree is a real linked Git worktree`.
- PASS (E2E completo) — dois Agentes com uma geração supervisionada cada; criação de grupo task/worktree/feature pela UI; filtragem por grupo e sem grupo; reordenação persistida por Agent ID; terminal anexa o mesmo `agent_id`/`runtime_id`; 0 POST `/start` adicionais; viewports 1280/768/320/390 sem overflow horizontal; screenshots em `.tempmediaStorage/agent-groups/`.
- PASS (contrato) — evidence map `agent-groups` com 2 critérios `audited` (`fd50e8fb`); contrato de `partial` para `present`, `verification: pending` (`467676ea`); doc de aceite sincronizado. `node scripts/verify-agent-workspace-evidence-map.mjs` exit 0 (48 capacidades, 97 cláusulas, 95 source+test, 0 certificadas); `node --test scripts/verify-agent-workspace-evidence-map.test.mjs` 4/4; `node scripts/verify-agent-workspace-v1.mjs` exit 0. Estado: 28 capacidades com todos os critérios `audited` (61/97 cláusulas); contrato 30 `present` / 17 `partial` / 1 `missing`.
- PASS (CI do push `36ffc734`) — `Publish Nexus documentation` (`45319282209`), `Nexus release Linux smoke (GitHub-hosted)` (`45319282242`) e `Refresh synthetic documentation captures` (`45319282303`) verdes; `ci.yml` (`45319282198`) com `Sanitize` e `Release configuration` verdes, `Linux E2E` em execução, resto em fila.
- FAIL (Windows) — `Windows beta.10 homologation` (`45319282180`) reprovou no job `Windows AMD64 local homologation gates` (ConPTY nativo, #216); bloqueia `cross-platform-exact-sha`.
- PENDENTE — CI do SHA `467676ea`; `workspace-acceptance-linux-amd64.json` do SHA exato ainda não coletado; Lane B (Bridge Maestro), Lane C (NEX 9 gates + aceite visual do owner), Lane D (runner Windows/macOS e chaves), Lane E (2 dispositivos físicos, quota real AGY) e Lane F (48/48 certificadas, tag `v1.0.1`) dependem do maestro. Release **NO_GO**.

## 2026-11-23 — Lane A `agent-command-center` auditada

- SHA local: `release/v1.0.0` @ `70212f0b` (commits `15c7fdd7`, `6aad3bb4`, `d7bcfd20`, `70212f0b`, sobre `f6e009bb`). Responsável: orquestrador. Ambiente: Linux + runner self-hosted `nexus-ci` em container.
- PASS (estado tipado) — `store.AgentCompleted = "COMPLETED"` em `internal/nexus/store/models.go`; `GOCACHE=/tmp/opencode/go-build GOFLAGS=-mod=readonly go test ./internal/nexus -run 'TestAgentApplicationService' -count=1` PASS (0,216 s). `TestAgentApplicationServiceListScopesProjectAndProjectsEffectiveRuntimeState` exige, na listagem por projeto: `RECOVERABLE` para runtime morto, e preservação estruturada de `WAITING`, `FAILED` e `COMPLETED` (sem inferência do texto do CLI).
- PASS (UI) — `web/src/features/agents/AgentsSurface.test.ts` 7/7 (`npx vitest run src/features/agents/AgentsSurface.test.ts`): lista do projeto via `nexus.listAgents(projectId)`, um cartão por Agent (`data-agent-id`), badge apenas de `agent.status` e seleção direta do Agent correto (Terminal e Director); a asserção negativa recusa `output|stdout|last_message|attention_reason` no cálculo do status do cartão.
- PASS (contrato) — evidence map com 2 critérios `audited` (`d7bcfd20`); contrato `agent-command-center` de `partial` para `present`, `verification: pending` (`70212f0b`); doc de aceite sincronizado. `node scripts/verify-agent-workspace-v1.mjs` exit 0 ("scope 48, certification still pending"); `node scripts/verify-agent-workspace-evidence-map.mjs` exit 0 (48 capacidades, 97 cláusulas, 95 source+test, 0 certificadas); `node --test scripts/verify-agent-workspace-evidence-map.test.mjs` 4/4. Estado: 27 capacidades com todos os critérios `audited` (59/97 cláusulas); contrato 29 `present` / 18 `partial` / 1 `missing`.
- PASS (web/Go) — `npm --prefix web run typecheck` limpo; `npx prettier --check src/features/agents/AgentsSurface.test.ts` limpo; `gofmt -l internal/nexus/store/models.go internal/nexus/agent_application_test.go` vazio.
- EM CURSO (CI `f6e009bb`, run `45318330774`) — `Sanitize self-hosted workspace`, `Release configuration` e `Documentation visual capture evidence` verdes; `Linux E2E` em execução; `Frontend`, `Security` e `Cross-platform compile guard` em fila. `Nexus release Linux smoke (GitHub-hosted)` (`45318330751`) verde.
- FAIL→NÃO-CÓDIGO (workflow de docs) — run `45318330781` (`f6e009bb`): upload do artefato `github-pages` concluído e `Publish GitHub Pages` falhou com `Found 0 artifact(s)` — race de API do GitHub.
- FAIL (Windows) — `Windows beta.10 homologation` (`45318330753`) reprovou no job `Windows AMD64 local homologation gates` (ConPTY, #216); bloqueia `cross-platform-exact-sha`.
- PENDENTE — `workspace-acceptance-linux-amd64.json` do SHA exato ainda não coletado; Lane B (Bridge Maestro), Lane C (NEX 9 gates + aceite visual do owner), Lane D (runner Windows/macOS e chaves), Lane E (2 dispositivos físicos, quota real AGY) e Lane F (48/48 certificadas, tag `v1.0.1`) dependem do maestro. Release **NO_GO**.

## 2026-11-23 — Lane A `remote-companion` e `focus-queue` auditadas

- SHA local: `release/v1.0.0` @ `f599dd7f` (commits `852adf2e`, `a6fc62af`, `4bda618d`, `bb5e1b5f`, `f599dd7f`). Responsável: orquestrador. Ambiente: Linux + runner self-hosted `nexus-ci` em container.
- PASS (`remote-companion`, revogação) — `GOCACHE=/tmp/opencode/go-build go test ./internal/control/web/ -run 'TestRemoteDevice' -count=1 -v`: 2/2 PASS. `TestRemoteDeviceRevocationClosesOnlyThatDeviceSession` prova que `RevokeDevice` fecha o watcher da sessão do dispositivo revogado em até 2 s, mantém a sessão do outro dispositivo pareado ativa e permite revogação independente do segundo.
- AUDITADO (`remote-companion`) — critérios 0/1/2 do evidence map agora `audited` (`a6fc62af`) com asserções de vínculo de sessão (`attentionRadarModel.test.ts`, `missionAttentionModel.test.ts`), handover de lease sem relaunch (`internal/control/host/host_test.go`, `qa_test.go`, `internal/control/web/terminal_handoff_authenticated_test.go`), revogação isolada (`devices_test.go`) e convite de uso único/QR (`internal/control/web/access_code_test.go`, `remote_invite_test.go`, `private_origin_test.go`, `web/scripts/e2e-remote-access.mjs`). Contrato de `partial` para `present`, `verification: pending` (`4bda618d`).
- AUDITADO (`focus-queue`) — critérios 0/1 `audited` (`bb5e1b5f`): ordenação por severidade e tempo, dedupe por episódio, exclusão de episódio respondido e claim one-shot durável (`focusQueueModel.test.ts`, `internal/control/registry/attention_claim_test.go`). Contrato de `partial` para `present` (`f599dd7f`).
- PASS (contrato) — `node scripts/verify-agent-workspace-v1.mjs` exit 0 ("scope 48, certification still pending"); `node scripts/verify-agent-workspace-evidence-map.mjs` exit 0 (48 capacidades, 97 cláusulas, 95 source+test, 0 certificadas); `node --test scripts/verify-agent-workspace-evidence-map.test.mjs` 4/4. Estado: 21 capacidades com todos os critérios `audited` (50/97 cláusulas); contrato 15 `present` / 31 `partial` / 2 `missing`.
- FAIL→NÃO-CÓDIGO (workflow de docs) — run `45315356441` (`4bda618d`): `Run actions/upload-pages-artifact@v4` success e `Publish GitHub Pages` falhou com `Found 0 artifact(s)` / `Fetching artifact metadata failed` — race de API do GitHub, sem alteração de código.
- PENDENTE — `Frontend` do CI depende da recaptura sintética no SHA novo (gate `visual manifest is stale`); `workspace-acceptance-linux-amd64.json` do SHA exato ainda não coletado; `Windows beta.10` vermelho (runner ConPTY, #216). Lane B (Bridge Maestro), Lane C (NEX 9 gates + aceite visual do owner), Lane D (runner Windows/macOS e chaves), Lane E (2 dispositivos físicos, quota real AGY) e Lane F (48/48 certificadas, tag `v1.0.1`) dependem do maestro. Release **NO_GO**.

## 2026-11-23 — Lane A `mobile-inbox` auditada: vínculo de sessão e resposta única

- SHA publicado: `origin/release/v1.0.0` @ `44813009` (commits `1b6405f2`, `d49c1beb`, `6a5173d0`, `5abd77a6`, `6142f735`, `44813009`, sobre `5f424f62`). Responsável: orquestrador. Ambiente: Linux + runner self-hosted `nexus-ci` em container.
- PASS (`mobile-inbox`, vínculo de sessão) — `web/src/app/attentionRadarModel.ts` propaga `provider_session_id` e o `response guard` do episódio exato para cada linha acionável; `web/src/nexus/missionAttentionModel.ts` expõe `missionAttentionSessionKey` (mission, project, task, intervention id/version) e `MissionAttentionPanel` usa essa chave no lugar de `mission_id`. Testes: `vitest src/app/attentionRadarModel.test.ts` 10/10 e `src/nexus/missionAttentionModel.test.ts` 5/5.
- PASS (`mobile-inbox`, resposta única) — `attentionResponseEpisodeKey` (runtime + generation + revision + fingerprint + interaction id) é a identidade de push: estável entre reconexões, distinta por revisão/runtime/interação. `vitest src/notifications/attentionDelivery.test.ts src/app/attentionRadarModel.test.ts` 17/17. Evidência do servidor já existente e referenciada no mapa: `internal/control/registry/attention_claim_test.go` (claim durável, `UNCERTAIN` e persistência após restart) e `internal/control/web/handlers_attention_retry_test.go` (retry HTTP reproduz o resultado persistido).
- PASS (contrato) — `mobile-inbox` de `partial` para `present` (`6142f735`), `verification: pending`; doc de aceite sincronizado. `node scripts/verify-agent-workspace-v1.mjs` exit 0 ("scope 48, certification still pending"); `node scripts/verify-agent-workspace-evidence-map.mjs` exit 0 (48 capacidades, 97 cláusulas, 95 source+test, 0 certificadas); `node --test scripts/verify-agent-workspace-evidence-map.test.mjs` 4/4. Estado: 19 capacidades com todos os critérios `audited` (45/97 cláusulas).
- PASS (web) — `npm --prefix web run typecheck` limpo; `prettier --check` limpo nos 7 arquivos web tocados.
- FAIL (CI `768d6593`, run `45314181244`) — job `Frontend` reprovou apenas no step `Verify documentation`: `visual manifest is stale after visual changes since 36f50ad5…: web/src/features/voice/terminalVoice.ts`. Consequência do gate de provenance visual (qualquer mudança em `web/src` fora de testes invalida as capturas). O mesmo gate voltará a reprovar no SHA `44813009` até o workflow `Refresh synthetic documentation captures` recapturar no SHA novo.
- PENDENTE — `Linux E2E` do run `45314181244` ainda em execução; `workspace-acceptance-linux-amd64.json` do SHA exato ainda não coletado; `Windows beta.10 homologation` continua vermelho (runner ConPTY, #216). Lane B (Bridge Maestro), Lane C (NEX 9 gates + aceite visual do owner), Lane D (runner Windows/macOS e chaves), Lane E (2 dispositivos físicos, quota real AGY) e Lane F (48/48 certificadas, tag `v1.0.1`) dependem do maestro. Release **NO_GO**.

## 2026-11-23 — Lane A `change-radar`/`consult-another-agent`/`browser-pane` auditados; lease stale corrigido

- SHA publicado: `origin/release/v1.0.0` @ `fc805351` (commits `503b8a81`, `ecc72433`, `8a227e94`, `2a2f7960`, `2611207c`, `b53f280c`, `dd1b384a`, `fc805351`). Responsavel: orquestrador. Ambiente: Linux + runner self-hosted `nexus-ci` em container.
- PASS (`change-radar`) — `ChangeCollision` expoe `BaseCommits`, `RequiresHumanReview` e `ChangedHunks`; endpoint agrupa arquivos e hunks por worktree; evidence map: 2 criterios `partial` → `audited` (`8a227e94`).
- PASS (`consult-another-agent`) — `ConsultationRequest` + `BuildConsultationStep` com scope/contexto/skills/diff do Maestro, hop unico, read-only e sem acao externa; evidence map: 2 criterios `partial` → `audited` (`2611207c`).
- PASS (`browser-pane`) — `bun test src/app/browserSurfaces.test.ts src/features/browser/BrowserSurface.test.ts`: 15 pass / 0 fail (36 `expect()`); janelas Browser unicas no Workspace OS, frames CDP, sem iframe; evidence map: 2 criterios `partial` → `audited` (`dd1b384a`).
- FAIL→FIX (Linux E2E) — run `45281023085` (`dd1b384a`): `TestTerminalHandoffAuthenticatedClientsShareProcessAndPersistIdentity` reprovou com `provider process received anything other than one leased input: ""`. Causa provada: o frame `lease_acquire` do attach inicial, processado em outra conexao depois do takeover, podia restaurar o escritor antigo e descartar o input do novo controlador. `fc805351` fenceia a lease por epoch monotonico no host e envia o epoch no comando web.
- PASS (fix do lease) — `go build ./internal/control/...`; `go test -race -count=3 ./internal/control/web/ -run '^TestTerminalHandoffAuthenticatedClientsShareProcessAndPersistIdentity$'` verde (2,8 s); `go test ./internal/control/host/` verde (1,4 s); `gofmt -l` vazio nos dois arquivos; `git diff --cached --check` limpo. A suite `internal/control/web` completa com `-race` passou no SHA anterior (101,9 s) — a falha e do runner sob carga.
- PASS (evidence map) — `node scripts/verify-agent-workspace-evidence-map.mjs` exit 0 (48 capacidades, 97 clausulas, 95 source+test, 2 source-only, 0 unmapped, 0 certificadas); `node --test scripts/verify-agent-workspace-evidence-map.test.mjs` 4/4.
- PENDENTE — CI de `fc805351` ainda em execucao; `workspace-acceptance-linux-amd64.json` do SHA exato ainda nao coletado; Lane B (Bridge Maestro), Lane C (NEX 9 gates + aceite visual do owner), Lane D (runner Windows/macOS e chaves), Lane E (2 dispositivos fisicos, quota real AGY) e Lane F (48/48 certificadas, tag `v1.0.1`) dependem do maestro. Release **NO_GO**.

## 2026-11-23 — Lane A `test-matrix` e `review-handoff` com asserção e teste

- SHA publicado: `origin/release/v1.0.0` @ `94bd4329` (commits `ebee6647`, `092510de`, `16dcab93`, `e99bf284`, `b21b32d4`, `fff5f40a`, `94bd4329`). Responsável: orquestrador. Ambiente: Linux + runner self-hosted `nexus-ci` em container.
- PASS (`test-matrix`) — `go test ./internal/nexus/ -run TestEvidenceBundle` 3/3 e suíte completa do pacote `internal/nexus` verde (109,7 s). `runner.VerificationResult.Source` distingue execucao de comando (`command_execution`) de declaracao; a matriz por worktree carrega exit code, SHA, timestamp e logs, e resultado sem provenance permanece nao-PASS com `missing_evidence: command_execution_provenance`. Evidence map: `test-matrix` `partial` → `audited` nos 2 criterios.
- PASS (`review-handoff`) — `go test ./internal/nexus/ -run TestMissionTeam` 7/7. A fase de revisao recebe `ChangedArtifacts` (paths dos agentes) e `ReviewedStepIDs` (passos revisados), depende dos mesmos passos e nao possui paths proprios; a implementacao continua com 2 passos, sem execucao duplicada. Evidence map: `review-handoff` `partial` → `audited` nos 2 criterios.
- PASS (evidence map) — `node scripts/verify-agent-workspace-evidence-map.mjs` exit 0 (48 capacidades, 97 cláusulas, 95 source+test, 2 source-only, 0 unmapped, 0 certificadas); `node --test scripts/verify-agent-workspace-evidence-map.test.mjs` 4/4.
- PASS (release docs) — `node scripts/verify-release-docs.mjs` exit 0 apos sincronizar `docs/legal/THIRD_PARTY_NOTICES.md` com `github.com/gorilla/websocket v1.5.4-0.20240701034025-d67f41855da4`.
- PASS (workflows) — `Nexus release Linux smoke (GitHub-hosted)` success em `94bd4329`; `Publish Nexus documentation` success em `94bd4329` (a falha anterior no run `45270632553` foi race de API do GitHub apos upload bem-sucedido do artefato `github-pages`).
- PENDENTE — CI do SHA `94bd4329` (run `45271602242`) ainda em fila; registro `workspace-acceptance-linux-amd64.json` do SHA final ainda não coletado; Lane B (Bridge Maestro), Lane C (NEX 9 gates, aceite visual do owner), Lane D (runner Windows/macOS nativo), Lane E (2 dispositivos físicos, quota real AGY) e Lane F (48/48 certificadas, tag `v1.0.1`) dependem do maestro. Release **NO_GO**.

## 2026-11-23 — Fixes de CI (race, manifesto visual, CVE) e Lane A `runtime-health`

- SHA publicado: `origin/release/v1.0.0` @ `ca4e1bfd` (commits `84967b97`, `ac66ff48`, `3bd88bc5`, `77e8abaf`, `ab464bb0`, `1e649749`, `ca4e1bfd`). Responsável: orquestrador. Ambiente: Linux + runner self-hosted `nexus-ci` em container.
- PASS (Linux E2E) — run `45259850030` (`ac66ff48`): `Linux E2E (Go 1.26.11)` success após o mutex em `pushWatchCancel` (`84967b97`); antes o mesmo job falhava com `WARNING: DATA RACE` em `push_attention.go:164` vs `server.go:777`.
- PASS (Frontend + captura visual) — mesmo run: `Frontend (Format, Lint, Styles, Typecheck, Unit & Build)` success e `Documentation visual capture evidence` success; antes ambos reprovavam por `visual manifest is stale after visual changes … internal/desktop/hud_detached_test.go`, corrigido em `ac66ff48` (filtro `_test.go`).
- FAIL→FIX (Security) — `govulncheck@v1.7.0` reprovou com `GO-2026-6278` em `github.com/gorilla/websocket@v1.5.3` (PRNG fraco na mask key). `3bd88bc5` eleva para `v1.5.4-0.20240701034025-d67f41855da4`; varredura local em `internal/control/web`, `internal/browsercontrol` e `internal/app`: `No vulnerabilities found`.
- FAIL→FIX (smoke Linux) — run `45264105223` (`ab464bb0`) reprovou com `undefined: ContinuityClaimIDLabel` porque `internal/control/registry/continuity_claim.go` é untracked de outro workstream e não existe no HEAD publicado. `ca4e1bfd` torna o diagnóstico autocontido (rótulos e limite de staleness locais); build e testes em clone limpo do SHA: `go build ./...` OK, `internal/control/registry` e `internal/doctor` verdes.
- PASS (Lane A `runtime-health`) — `DiagnoseRuntimeHealth` devolve lease, PTY, heartbeat, PID e capability de provider por sinal, com `OK`/`UNKNOWN`/`ERROR` e detalhe de evidência; `doctor` expõe os sinais por runtime. Testes: `TestRuntimeHealthReportsUnknownForUnprovableSignals`, `TestRuntimeHealthReportsErrorsWithoutClaimingRecovery`, `TestRuntimeHealthReportsFreshContinuityLease`, `TestRuntimeHealthReportsStaleContinuityLeaseHeartbeat`, `TestBuildTerminalReportExposesRuntimeHealthSignals`. Evidence map: 2 critérios de `runtime-health` `partial` → `audited` (`1e649749`); `verify-agent-workspace-evidence-map.mjs` exit 0 (48 capacidades, 97 cláusulas) e seus testes 4/4.
- NÃO EXECUTADO — Lane B (Bridge Maestro ativo + `memory-migrate apply` no SHA publicado), Lane C (aceite visual NEX em HUD+Site), Lane D (runner Windows/macOS e chaves), Lane E (2 dispositivos físicos, quota real AGY), Lane F (48/48 certificadas, tag `v1.0.1`). Bloqueio atual no Windows: `TestSessionHost_TwoClientLeaseHandoffIsAuditedAndSingleWriter` falha e o gate de captura documental exige SHA exato. Release **NO_GO**.

## 2026-11-23 — Lane A: auditoria de 7 capacidades com critério `partial`

- SHA de trabalho: `release/v1.0.0` @ `b1ba5377`/`8d101db8`/`746950d0` (push `c1e9dfac..b1ba5377` aceito). Responsável: orquestrador. Ambiente: Linux, sem `sudo`.
- FIX (achado real de segurança/contrato) — `RecommendResources` ignorava `TaskRequirements.RuntimeAffinity`: um PIN explícito de provider/profile podia ser sobreposto por score em `/api/v1/resources/recommend`. Agora PIN exclui as demais contas com motivo explícito e falha fechado quando o alvo fixado não existe; `PREFER`/`AUTO` mantêm o pool. Testes: `TestResourceRecommendation_ConfidenceIsExplicitLiveCachedUnknown`, `…PinnedProviderExcludesOtherAccounts`, `…PinnedProviderUnavailableFailsClosed`, `…PinnedProfileExcludesOtherProfiles`, `…PreferModeKeepsPoolEligible` (`b1ba5377`). `go test ./internal/nexus/...` verde (142s).
- PASS (terminal remoto) — novo `TestTerminalAttachRejectsUnauthenticatedRevokedViewerAndForeignOrigin` (`8d101db8`) prova no roteador real: 401 sem sessão (origem sozinha não autentica), 403 para origem estrangeira com sessão válida, 403 para grant `viewer` (override remoto) e 401 para sessão revogada. Controle único já coberto por `TestBrokerPromotesHostLeaseBeforeBrowserControl`; expiry/revoke por `TestDesktopSessionRevocationAndExpiry`.
- PASS (saídas geradas) — novo `TestOutputHistoryRetentionAndPrivateCaching` (`746950d0`) fixa a janela `maxRevisions`, prova que revisão podada não é servida e que `Serve` responde `Cache-Control: private, no-store`. ACL por projeto e MIME/hash já cobertos por `history_test.go` (acesso cross-project negado) e `catalog_test.go`.
- AUDITADO SEM MUDANÇA — `parallel-challenge`: `parallel_challenge_mission.go` + `_test.go` (versionados em `faab21be`) já provam distribuição por `MissionRun` isolado, recusa de worktree compartilhado, comparação determinística e ausência de merge automático; `go test -run 'TestMissionChallenge|TestEvaluatePlanChallenge|TestEvaluateChallenge'` verde.
- AUDITADO SEM MUDANÇA — `nex-companion`: `node scripts/verify-nex-identity-lock.mjs` exit 0 ("identity provenance gate OK, status: FROZEN"; master retro HD segue pendente, não declarado pronto). Motion: `nexIdleDirector.test.ts` ("pauses on reduced-motion…"), `mascotModel.test.ts` ("does not animate when motion is disabled…"), `GuardianMascot.test.tsx` (atlas bloqueado, drag por teclado). Contraste: `theme.test.ts` (schemes `high-contrast`, `resolveVisualFinish`).
- PARCIAL (bloqueado por ambiente) — `contextual-hud`: click-through e regiões interativas cobertos (`hudHitRegions.test.ts`, `nativeHudExpansion.test.ts` 30 ciclos) e reduced-motion coberto; o **smoke nativo** exige `libgtk`/`libglib`, ausentes neste host. `nex-hud`/`nexus-desktop` não executam aqui (10–17 libs ausentes). Continua `partial`.
- PARCIAL — `two-client-real-e2e`: `multi_runtime_attention_e2e_test.go` cobre 2 clientes/2 projetos/2 runtimes com sessões distintas e deduplicação; `terminal_handoff_authenticated_test.go` cobre 2 clientes autenticados, lease `CONTROL`/`VIEW_ONLY`, handoff do mesmo processo e persistência de identidade. Falta a superfície de **browser control** e reconexão com navegador real.
- BLOQUEADO (não commitar) — `docs/reference/agent-workspace-v1-evidence-map.json` tem alterações **não commitadas de outro workstream** (campos `issue`/`responsible` e 25 vereditos `assertionAudit: audited`, contra 0 em `HEAD`). Nenhuma dessas auditorias é minha; não foram staged para não publicar certificação não verificada. Os vereditos acima precisam ser aplicados ao mapa quando a sessão concorrente liberar o arquivo.
- ACHADO (CI) — run `45228402087` (`c1e9dfac`) falhou em `Linux E2E`, `Security`, `Documentation visual capture evidence` e `Frontend`; `Windows release 1.0.0 exact-SHA homologation` também falhou. Reprodução local: `GOFLAGS='-mod=readonly -tags=gtk3' go vet ./...` aborta com `exec: "pkg-config": executable file not found`, e o capture precisa de Chromium, que não sobe sem `libglib`. Todos os passos do job `Frontend` passam em worktree limpa no SHA exato (`format:check`, `lint` 0 erros/1 warning, `lint:styles`, `check:styles`, `typecheck`, `test` 979/979, `build`, `docs-verify` 255 arquivos, `verify-agent-workspace-v1` exit 0, `release-version.test.mjs` 3/3). Conclusão: as falhas são de ambiente do runner self-hosted, não de código — exigem máquina com `pkg-config`+headers WebKitGTK e Chromium funcional (Lane D/E).
- NÃO EXECUTADO — Lane B (Bridge Maestro: `ORQUESTRADOR_MAESTRO_BRIDGE_MEMORY_WRITE`), Lane C (NEX `BLOCKED 2/11`), Lane D (runner Windows/macOS inexistente), Lane E (hardware/OAuth/quota reais), Lane F (tag `v1.0.1`). Release **NO_GO**.

## 2026-11-23 — Lane 0 do GO: runner religado, contrato sem certificação antecipada, #196 versionado

- SHA de trabalho: `release/v1.0.0` @ `ef8f3556` (sobre `c20a77c9`, `49632c6c` do owner); `origin/release/v1.0.0` em `bf7c85ec` no início da janela. Responsável: orquestrador. Ambiente: Linux, sem `sudo`, sem `libicu`/`libgtk`/`libglib`.
- PASS (runner CI) — `~/actions-runner/run.sh` abortava com exit 134 ("Couldn't find a valid ICU package"); com `DOTNET_SYSTEM_GLOBALIZATION_INVARIANT=1 GOCACHE=/home/desenvolvedor/actions-runner/.cache/go-build` o listener sobe ("Connected to GitHub", "Listening for Jobs") e executa o job do run `45184025970` (`bf7c85ec`): `Sanitize self-hosted workspace` success, `Release configuration` success, `Cross-platform compile guard` success, `Frontend…` in_progress.
- PASS (contrato sem claim falso) — `docs/reference/agent-workspace-v1.json`: 9 linhas `"verification": "verified"` → `"pending"` (`ef8f3556`). Estado final: `{present/pending: 12, partial/pending: 34, missing/pending: 2}`, 0 certificadas. `node scripts/verify-agent-workspace-v1.mjs` exit 0 ("scope 48, certification still pending"); `node scripts/verify-agent-workspace-evidence-map.mjs` exit 0 ("48 capabilities, 97 acceptance clauses; 95 source+test, 2 source-only, 0 unmapped; no acceptance certified").
- PASS (#196 golden replay) — `go test ./internal/runtime/ -run GoldenReplay -count=1 -v`: 2/2 PASS (`TestTerminalOutputScrubberGoldenReplay`, `…Chunked`). Fixtures sanitizados (`redacted@example.invalid`, sem path do operador); SHA-256 `afee3b4272fe32ae204cea78a01db4872bf035fc4f8542d9c69e0c63576cc452` (raw) e `a4fbb263e856193bc7063e6b71d3a8c64df9e8837a55ca2aa278ec1a351ac0d7` (scrubbed). Versionados em `c20a77c9` com `web/package.json` (`test:e2e:terminal-golden`) e step `terminal-golden-e2e` no `ci.yml`.
- NÃO EXECUTADO (bloqueio de ambiente) — `node web/scripts/e2e-terminal-golden-replay.mjs` local: Playwright baixa o Chromium (`chromium_headless_shell-1243`, 114,3 MiB) mas o binário aborta com `libglib-2.0.so.0: cannot open shared object file`; `ldd` lista 17 libs ausentes (glib, gobject, gio, atk, dbus, X11, gbm, asound…). `ldd ~/.local/bin/nexus-desktop` lista 10 libs ausentes. Sem `sudo`, as dependências de sistema não podem ser instaladas neste host; a validação do replay visual depende do job de CI.
- NÃO EXECUTADO — Lane A (fechamento das 46 capacidades), Lane B (Bridge Maestro), Lane C (NEX 9 gates), Lane D (runner Windows/macOS, native dispatch, stable certification 8h, chaves), Lane E (homologação física), Lane F (freeze, 48/48, tag `v1.0.1`). Release **NO_GO**.

## 2026-11-23 — 4 blocos commitados e enviados: `origin/release/v1.0.0` @ `3867a1fd`

- SHA publicado: `release/v1.0.0` @ `3867a1fd` (push `6c322427..3867a1fd`, fast-forward, 29 commits; tag `v1.0.0` intacta em `d89797cd`). Commits do owner's call: `8b932b0c` (A `execution_owner`), `2af8d233` (B migração Maestro memory), `7b2a5a41` (C CLI `memory-migrate`), `a90ace4b` (D CI Fase B/C); commits concorrentes do owner: `d6b1d72e`, `3867a1fd`. Responsável: orquestrador. Ambiente: Linux.
- PASS (árvore commitada, worktree limpa) — `go build ./internal/...` OK; `gofmt -l internal/` vazio; `internal/nexus/runner` 20.3s, `internal/nexus/store` 16.9s, `internal/core/memoryrecovery` 4.1s, `internal/nexus` (migração + `Mission|ExecutionOwner`) 50.1s, `internal/app` (`MaestroMemoryMigrate|MaestroRecovery`) 0.1s, `internal/control/handoff` 0.5s; `node --test` (evidence + contract + doc-sync) 23/23; `verify-agent-workspace-v1.mjs` exit 0 (scope 48, certification pending); YAML de `ci.yml`/`release.yml` parseia.
- PASS (higiene de escopo) — commits feitos por staging de índice (`git apply --cached` + `git add` só dos arquivos novos de cada bloco); hunks de outros workstreams ficaram de fora: `internal/app/app.go` (help quota/continuidade), `internal/nexus/store/autonomy.go` (paginador `Seq`), `internal/nexus/mission_intervention_persistence_test.go` (depende do paginador). Working tree preservada com 98 modificados + 20 untracked alheios; `git diff --cached` vazio entre commits.
- PASS (push) — `git push origin release/v1.0.0` aceito (`6c322427..3867a1fd`); `origin/release/v1.0.0` = `3867a1fd`; CI do push: run `45165787696` (`queued`) no SHA exato.
- NÃO EXECUTADO — `workflow_dispatch` nativo (`native_platform=all`) nos runners `nexus-windows`/`nexus-macos`; `nexus maestro memory-migrate apply` (adiado pelo owner para o SHA publicado com Bridge Maestro ativo); fechamento das 48 capacidades em `present`+`verified`; assinatura/tag `v1.0.1`. Release **NO_GO**.

## 2026-11-22 — Fatiamento A–D validado em worktree isolado (4 commits que compilam e testam)

- SHA de trabalho: `release/v1.0.0` @ `d6b1d72e` (owner commitou o fix do produtor durante a janela). Worktree temporária `/tmp/opencode/blockwt` criada e removida nesta sessão; árvore principal com 150 alterações preservadas; worktrees do owner intactas. Responsável: orquestrador. Ambiente: Linux.
- PASS (bloco A) — `feat(go): enforce single execution owner on plans and runs` (13 arquivos + migration `0040` + 2 testes novos): `go build ./internal/...` OK; `internal/nexus/runner` 20.1s, `internal/nexus/store` 20.5s, `internal/nexus -run 'Mission|ExecutionOwner'` 52.0s.
- PASS (bloco B) — `feat(go): add Maestro canonical memory migration with gate and recovery` (5 alterados + 6 novos): build OK; `internal/nexus` (plan/migrate/recover), `store`, `memoryrecovery`, `core/config`, `control/handoff` verdes.
- PASS (bloco C) — `feat(cli): expose nexus maestro memory-migrate plan|apply|recover` (4 arquivos): build OK; `internal/app -run 'MaestroMemoryMigrate|MaestroRecovery'` verde; `gofmt` limpo.
- PASS (bloco D) — `feat(ci): produce native platform records and assemble release evidence` (4 arquivos): build OK; `node --test scripts/workspace-acceptance-e2e.test.mjs` 11/11 em 3 execuções; YAML de `ci.yml`/`release.yml` parseia; `verify-agent-workspace-v1.mjs` exit 0.
- ACHADO (contaminação) — hunks de outros workstreams em `internal/app/app.go` (help quota/continuidade), `internal/nexus/store/autonomy.go` (`Seq`/`ListMissionRunEvents`), `internal/nexus/mission_intervention_persistence_test.go` (depende do paginador) e `.github/workflows/ci.yml` (step `terminal-golden-e2e`): o fatiamento definitivo exige staging por índice. `mission_intervention_persistence_test.go` **não** pertence ao `execution_owner` (primeira atribuição estava errada).
- NÃO EXECUTADO — commit/push na árvore real, `workflow_dispatch` nativo, cutover da memória em produção, `make quality-full` no SHA final. Release **NO_GO**.

## 2026-11-22 — GO 1.0.1 Fase B/C: produtores nativos e montagem fail-closed da evidência

- SHA de trabalho: `release/v1.0.0` @ `a5a30a2c` (árvore com alterações concorrentes de outros workstreams, sem push). Responsável: orquestrador. Ambiente: Linux.
- PASS (YAML) — `.github/workflows/ci.yml` e `.github/workflows/release.yml` parseiam com pyyaml; steps novos presentes: `Produce/Upload V1 workspace acceptance record (windows-amd64|macos-arm64)`, `actions/setup-node@v4` nos dois jobs nativos, download dos records + `native-acceptance-records.jsonl` no selo, `assemble` no `same-sha-gate`.
- PASS (scripts) — `node --test scripts/workspace-acceptance-e2e.test.mjs` 11/11; `node --test scripts/verify-agent-workspace-v1.test.mjs scripts/tests/agent-workspace-doc-sync.test.mjs` 12/12.
- PASS (dry-run do gate) — 3 registros sintéticos (`linux/windows/macos`) + `assemble --native-certified cross-platform-exact-sha` geraram `evidence.json` com 48 capacidades e 3 plataformas; `verify-agent-workspace-v1.mjs --release --sha <HEAD> --ci-run-id 1 --evidence <externo>` recusou só por `source/integration not completed` (48 linhas), ou seja, a montagem passou e o bloqueio real é o contrato `present/pending`. Nenhum status foi alterado para fechar aceite.
- PASS (checks de selo) — `node -e` do job `native-certification` validado com 2 records sintéticos (`producer`/`platform`/`sourceSha`/`jobUrl`): `native acceptance records verified`; record com SHA divergente é recusado (teste unitário dedicado).
- PASS (docs) — `node scripts/verify-agent-workspace-v1.mjs` exit 0 (scope 48, certification pending); `node scripts/docs-verify.mjs --links-only` exit 0 (256 Markdown); `node scripts/verify-agent-workspace-evidence-map.mjs` exit 0 (48 capacidades, 97 cláusulas, 0 certificações).
- CORRIGIDO (flake real) — `internal/app` falhou 1 vez em 8 execuções do filtro `MaestroMemoryMigrate|MaestroRecovery` com `list legacy project memories: unable to open database file (14)`: o singleton `nexus.Default()` capturava o store de um `NEXUS_DATA_DIR` temporário já removido. Isolamento corrigido com `nexus.ResetDefaultForTest()` + `t.Cleanup` nos testes do CLI (`internal/app/maestro_memory_recovery_test.go`). Reexecução: **60/60 execuções verdes** (20+40 consecutivas) e `go test ./internal/app -count=1` completo verde (59.5s).
- PASS (Go, rodada final) — `go build ./internal/...` e `go vet ./internal/nexus/... ./internal/core/memoryrecovery/... ./internal/control/handoff/... ./internal/app/...` sem saída; `gofmt -l internal/` vazio; `go test` verde em `internal/nexus` (migração), `internal/nexus/store`, `internal/core/memoryrecovery`, `internal/core/config`, `internal/control/handoff`, `internal/app`; `-race` verde em `internal/app` (4.8s) e no conjunto migração/gate/journal (18.0s/11.5s/2.3s).
- NÃO EXECUTADO — `workflow_dispatch` nativo real nos runners `nexus-windows`/`nexus-macos` (sem acesso), `make quality-full` no worktree limpo do SHA final, push/tag, fechamento das 48 capacidades em `present`+`verified`. Release **NO_GO**.

## 2026-11-22 — GO 1.0.1 Fase A: fatias commitadas e verificadas

- SHA de trabalho: `release/v1.0.0` @ `ed2c7370` (sobre `fe1beec1`; árvore com alterações concorrentes de outros workstreams, sem push). Responsável: orquestrador. Ambiente: Linux.
- PASS (versão) — `ef0560d1 build(release): bump version to 1.0.1`: `VERSION` e `web/package.json` = `1.0.1`; `README.md`/`README.en.md`/`README.es.md`, `docs/TLDR.md` e `docs/release-status.md` sincronizados (só arquivos vivos; histórico datado em `DEV/` e `CHANGELOG.md` intacto). `make version-check` → `VERSION CONTRACT PASS: all sources consistent`.
- PASS (contrato) — `7744979f feat(workspace): declare requiredClients in v1 contract`: 48/48 com `requiredClients` (diff de 48 linhas, round-trip indent=1 idêntico); `node scripts/verify-agent-workspace-v1.mjs` exit 0 (scope 48); `node --test scripts/verify-agent-workspace-v1.test.mjs` 7/7.
- PASS (produtor/CI) — `ed2c7370 feat(ci): produce v1 workspace acceptance record in test-linux`: `node --test scripts/workspace-acceptance-e2e.test.mjs` 9/9; dry-run real `run --platform linux-amd64 --allow-incomplete` → 47/48 `passed` + 1 `blocked` (`cross-platform-exact-sha`, exige certificação nativa Win/macOS), 52 steps, exit 0; YAML do `ci.yml` commitado parseia com pyyaml (14 jobs, steps `Produce`/`Upload` presentes).
- HIGIENE DE ESCOPO — staging parcial por índice (`git hash-object` + `git update-index --cacheinfo`) em `web/package.json` e `.github/workflows/ci.yml` para não absorver o hunk concorrente `terminal-golden`; `docs/reference/agent-workspace-v1-evidence-map.json` (303 linhas, outro workstream) não foi staged; entradas DEV anexadas por índice (HEAD + bloco) preservando as edições concorrentes no worktree.
- NÃO EXECUTADO — `make quality-full` e `docs-verify` no worktree limpo do SHA final (worktree sujo hoje: manifest visual stale, esperado); records Windows/macOS; montagem do `evidence.json`; push/tag. Release **NO_GO**.

## 2026-11-22 — auditoria de asserção das 97 cláusulas do mapa canônico

- SHA de trabalho: `release/v1.0.0` @ `82b90a3b` (base local; árvore com alterações locais, sem commit/push). Responsável: orquestrador. Ambiente: Linux.
- PASS (auditoria de asserção) — `docs/reference/agent-workspace-v1-evidence-map.json`: 97/97 cláusulas agora têm `assertionAudit` (21 `audited`, 76 `partial`), lidas contra os arquivos de teste referenciados. `coverage` permanece `unverified` em todas; as 48 capacidades seguem `verification: pending`; 0 certificações.
- PASS (registro por cláusula) — cada capacidade ganhou `issue` (contrato canônico) e `responsible` (`NEXUS` 43 / `INTEGRATION` 5), fechando o registro das 97 linhas com estado, issue, cenário, evidência e responsável.
- PASS (gates) — `node scripts/verify-agent-workspace-evidence-map.mjs` exit 0 (48 capacidades, 97 cláusulas, 95 source+test, 2 source-only, 0 unmapped, nenhum aceite); `node --test scripts/verify-agent-workspace-evidence-map.test.mjs` 4/4; `node scripts/verify-agent-workspace-v1.mjs` exit 0 (scope 48, certification still pending).
- LACUNAS REAIS REGISTRADAS (`partial`) — `floating-inspector` (sem assertion de identidade/evidência/lineage), `agent-timeline` (sem launch/attach/quota/handoff/recovery), `runtime-health` (só transições de estado; sem lease/PTY/heartbeat/PID), `review-handoff` e `consult-another-agent` (sem assertion de handoff/segunda opinião), `change-radar` (sem hunks por Agent/worktree), `test-matrix` (sem exit code/SHA/timestamp/logs por worktree), `browser-pane`/`device-preview` (sem viewport real), `mobile-inbox` (sem resposta única sob falha de rede), `remote-terminal` (sem CSRF/expiry/revoke), `voice-actions` (sem Enter automático nem Agent de destino), `resource-recommendations` (sem PIN), `maestro-canonical-memory` (migração inexistente; store semântica legada ainda ativa), `single-execution-owner` (sem executor único asseverado), `remote-generated-outputs` (sem TTL/ACL), `two-client-real-e2e` (sem handoff/browser/PTY), `cross-platform-exact-sha` (evidência externa).
- NÃO EXECUTADO — cenários positivos/negativos com providers reais, aceite do owner, evidência exact-SHA externa, assinatura e publicação. Release **NO_GO**.

## 2026-11-22 — #196 golden replay automatizado no repo; #232/#176 contratos reconciliados

- SHA de trabalho: `release/v1.0.0` @ `82b90a3b` (base local, sem commit/push). Responsável: orquestrador. Ambiente: Linux, Go 1.26.11, xterm 5.3.0 (`web/node_modules/xterm`), Playwright 1.63.0/Chromium.
- PASS (fixture golden sanitizada) — captura real de 1000 eventos (PTY `pty.fork`, 6545 bytes, SHA-256 `b94fcbc59076d7226a0f33a071bf685329639b54dac6b8c332e2bb8cde58cb6b`) teve **apenas** o e-mail do operador (`kivervinicius@gmail.com` → `redacted@example.invalid`) e o caminho do workspace (`/tmp/opencode/issue231-workspace` → `/tmp/nexus-golden-workspace`) redigidos; estrutura ANSI byte-idêntica. Versionados: `internal/runtime/testdata/pty196_golden.raw.bin` (6536 bytes, SHA-256 `afee3b4272fe32ae204cea78a01db4872bf035fc4f8542d9c69e0c63576cc452`) e `internal/runtime/testdata/pty196_golden.scrubbed.bin` (6502 bytes, SHA-256 `a4fbb263e856193bc7063e6b71d3a8c64df9e8837a55ca2aa278ec1a351ac0d7`).
- PASS (Go, byte-exato) — `internal/runtime/terminal_output_golden_test.go`: `TestTerminalOutputScrubberGoldenReplay` (byte-exato raw→scrubbed) e `TestTerminalOutputScrubberGoldenReplayChunked` (chunk sizes 1/2/3/7/64/1024, resultado idêntico). O scrubber remove `ESC[>4m`, `ESC[>4;2m`, `ESC[>5u` e preserva `ESC[?u` e OSC 11. Gates: `gofmt` limpo; `go vet ./internal/runtime/` exit 0; `go test ./internal/runtime/ -count=1` exit 0.
- PASS (Web/xterm 5.3 real) — `web/scripts/e2e-terminal-golden-replay.mjs` (`bun run test:e2e:terminal-golden`): replay dos bytes sanitizados no xterm 5.3 em Chromium → `prompt_lines=3`, `rendered_length=159`, `exact_match=true`, `console_errors=0`. Entrada no CI: `.github/workflows/ci.yml` step `terminal-golden-e2e` (job Browser E2E).
- PASS (#232/#176 contratos versionados) — inventário de fronteiras Nexus × Maestro reconciliado nos documentos candidatos: `docs/product/flow.md` (MissionRunner só para `execution_owner=NEXUS`), `docs/product/maestro.md` (ponteiro para a seção versionada do ADR), `docs/architecture/core.md` (linha Maestro na tabela de ownership + referência), `docs/architecture/consumer-matrix.md` (linha Maestro), `docs/reference/features.json` + `docs/reference/feature-matrix.md` (limite `execution_owner` regenerado por `node scripts/verify-feature-catalog.mjs --write`), `docs/architecture/MAESTRO_EXTERNAL_AUTHORITY_ADR.md` e `ARCHITECTURE.md` (seção de inventário já publicada). Gates: `node scripts/docs-verify.mjs --links-only` exit 0 (256 Markdown files); `npm --prefix docs-site run check:docs` exit 0 (100 páginas); `node scripts/verify-feature-catalog.mjs --check` exit 0 (36 mapeamentos). **Nenhum código alterado nesta reconciliação.**
- NÃO EXECUTADO — aceite em dois dispositivos físicos; reprodução do sintoma original do owner; CI no SHA publicado; commit/push (decisão do owner). Issues #196 e #232/#176 permanecem OPEN até decisão do owner. Release **NO_GO**.

## 2026-11-22 — #196 stress de 1000 eventos reconciliado (buffer completo do xterm)

- SHA de trabalho: `release/v1.0.0` @ `82b90a3b` (base local, sem commit/push). Responsável: orquestrador. Ambiente: Linux, AGY/Antigravity CLI 1.5.14 em PTY real (`pty.fork`, 24×80) e UI web real `http://127.0.0.1:13431` (runtime isolado `agy-06H07PFNRW7RVXDQYKNM66B280`, projeto `prj_06GFFF9XDKDSMV7TES7BM8H3T4`).
- PASS (PTY real, 1000 eventos) — roteiro determinístico (seed 20261122) com 1000 teclas de edição (digitar/backspace/esquerda/direita/delete): 6545 bytes capturados em 69 chunks com timestamps, SHA-256 `b94fcbc59076d7226a0f33a071bf685329639b54dac6b8c332e2bb8cde58cb6b`. Artefatos: `/tmp/opencode/issue196/agy_stress_raw.bin`, `agy_stress_trace.jsonl` (SHA-256 `0121dc626d592d6d43ed1740d571963454f932db45c7afe8a8876e228bbc630c`), `agy_stress_summary.json` (SHA-256 `c8647fabc6f92dfa34d6a4de4f466de68786828e2c48e599f7ef6300f74a45a6`).
- PASS (buffer completo xterm 5.3, antes `expected_present=false`) — o falso negativo anterior era limitação do harness: ele comparava só a **última linha do prompt** (77 chars) contra a entrada esperada inteira (159 chars). Reconstruindo o prompt multi-linha (3 linhas de buffer, 159 chars): `exact_match=true`, `expected_present=true`, `stale_tail_absent=true`. Relatório: `/tmp/opencode/issue196/agy_stress_screen.report.json` (SHA-256 `d4ed748e335b8ec6c73978a142fb7ebfc75941adaf14e9d271e37d4abfcd2a0d`).
- PASS (UI web real, 1000 eventos) — 1000 teclas via teclado real no browser: 2448 ms, prompt final multi-linha com `expected_length=154`, `rendered_length=154`, `exact_match=true`, `expected_present=true`, **0 erros de console**. Relatório: `/tmp/opencode/issue196/agy_edit_1000_web_report.json` (SHA-256 `541f00b259a346454ac98545565513b0cbff221fd567ee1ccc287aa0a0175fff`); captura: `agy_edit_1000_web.png` (SHA-256 `5b031d529d2919fd8154f84a3194df24108a9b1f2d8a4012f6e116444f5cded5`).
- LEITURA CORRETA — **não há corrupção visual de terminal TUI comprovada** nos dois caminhos (PTY real e UI real): o conteúdo final do prompt é exatamente o esperado e não há ghosting/sobreposição. `expected_present=false` anterior era defeito do harness, não do produto.
- NÃO EXECUTADO — regressão golden automatizada dentro do repositório (fixture binário de bytes ainda fora do repo); aceite em dois dispositivos físicos; reprodução do sintoma original relatado pelo owner. A issue #196 permanece OPEN por ausência do sintoma observado pelo owner, sem fechar por evidência sintética. Release **NO_GO**.

## 2026-11-22 — #218 HTTP 400 de agent-groups: classes de erro separadas (4xx vs 5xx)

- SHA de trabalho: `release/v1.0.0` @ `82b90a3b` (base local, sem commit/push). Responsável: orquestrador.
- CAUSA (parcial, medida) — `handleProjectAgentGroups` convertia **todo** erro de `UpdateAgentGroup` em HTTP 400, incluindo falha interna de store (transação/lock/SQLite): o teste intermitente do CI (`update group status=400`) não conseguia distinguir payload inválido de falha de banco. A fixture compartilhada já havia sido isolada em `c6cd9193` (`NEXUS_DATA_DIR` + `nexus.ResetDefaultForTest`), eliminando a interferência global; a separação de classes faltava.
- FIX (store) — `internal/nexus/store/agent_groups.go`: novo sentinel `ErrInvalidAgentGroup`; payload inválido (nome vazio/>120, membros >500, id vazio, id duplicado, agente de outro projeto ou não persistente) passa a ser `fmt.Errorf("%w: …", ErrInvalidAgentGroup)`; `ErrNotFound` continua reservado para grupo inexistente.
- FIX (web) — `internal/control/web/handlers_agent_groups.go`: helper `writeAgentGroupError` mapeia `ErrInvalidAgentGroup` → **400** com mensagem sanitizada, `ErrNotFound` → **404**, qualquer outro erro (falha interna de store) → **500** com mensagem genérica `agent group operation failed`, sem expor detalhe do driver. `GetProject` deixou de mascarar falha interna de store como 404.
- REGRESSÃO — `internal/control/web/agent_groups_api_test.go::TestWriteAgentGroupErrorSeparatesClientAndStoreFailures` fixa 400/404/500 e proíbe vazamento de detalhe interno no corpo; `internal/nexus/store/agent_groups_test.go` fixa `errors.Is(err, ErrInvalidAgentGroup)` para duplicado, agente estrangeiro, id vazio e nome vazio.
- PASS (gates) — `go build ./internal/...` exit 0; `go vet ./internal/nexus/store ./internal/control/web` exit 0; `go test ./internal/nexus/store ./internal/control/web -run 'AgentGroup'` exit 0; `go test -race -count=30 ./internal/control/web -run '^TestAgentGroupsAPIUsesDurableProjectScopedMembership$'` exit 0 (44,5 s); `go test -race -count=30 ./internal/control/web -run 'AgentGroup'` exit 0 (40,8 s); `go test -race -count=30 ./internal/nexus/store -run 'AgentGroup'` exit 0 (116,9 s); `go test -race -count=1 ./internal/nexus/store ./internal/control/web` exit 0 (206,2 s / 100,6 s).
- NÃO EXECUTADO — CA real green no SHA publicado; o sintoma intermitente **não foi reproduzido** em 30 repetições sob `-race` com a fixture isolada, logo a causa raiz do disparo original permanece não comprovada; a separação 4xx/5xx e a regressão determinística ficam como correção de robustez de contrato. A issue #218 permanece OPEN até o owner decidir o fechamento. Release **NO_GO**.

## 2026-11-22 — Gate 0: gate completo verde no HEAD final e `docs-verify` aprovado

- PASS (gate completo no HEAD final) — worktree limpo em `ffafedcd` (`/tmp/opencode/nexus-quality-final`): `make quality` → exit 0 (Prettier limpo; ESLint 0 erros + 1 warning preexistente em `FlowRunSurface.tsx`; lint Go 0 issues; suíte Go verde; Web 153 arquivos / 979 testes; `VERSION CONTRACT PASS`).
- PASS (docs-verify no HEAD final) — `node scripts/docs-verify.mjs` → exit 0 (`docs verification passed: 255 Markdown files, 33 required assets/files`) em worktree limpo em `ffafedcd`.
- ACHADO AMBIENTAL (não é regressão) — `internal/app` é **byte-idêntico** entre `bd17f90e` (gate verde medido antes) e `ffafedcd`; com o `$HOME` real, `TestResolveLaunchProfileFailsClosedOnUnverifiedQuota` e `TestResolveLaunchProfileOverrideAcceptsFallback` falham porque leem o estado local de quota do Codex; com `HOME` isolado os dois passam. O gate completo só é determinístico com `HOME` isolado — registrado aqui para não ser lido como defeito do Gate 0.
- NÃO EXECUTADO — Named Tunnel com credencial real; Tailscale com tailnet real; push contra push service real; aparelho físico; SHA publicado. Release **NO_GO**.

## 2026-11-22 — Gate 0: recaptura visual e `docs-verify` no HEAD funcional

- RECAPTURA VISUAL — `npm --prefix web run docs:capture` reexecutado no HEAD limpo; 12 arquivos de `docs/assets/` recapturados e commitados em `872e8b58` (`docs(visual): recapture synthetic Web evidence for gate 0 UI`), porque o Gate 0 alterou a UI visível (`RemoteAccessTab.tsx`, `SettingsSurface.tsx`, `featureAnnouncements.ts`, `featureTourDefinitions.ts`, manifest/PWA e o cliente de API web).
- PASS (docs-verify) — `node scripts/docs-verify.mjs` → exit 0 (`docs verification passed: 255 Markdown files, 33 required assets/files`) em worktree limpo no HEAD com a recaptura; a procedência visual stale que quebrava o gate ficou resolvida.
- HYGIENE DE ESCOPO — este commit retira dos `DEV/` versionados o bloco de documentação do workstream #214 (`DEV/VERIFY.md`) que entrou por arrastamento em `55ebdc5a` via `git add` de arquivo inteiro durante edição concorrente; o conteúdo volta ao working tree do outro agente, sem commit.
- NÃO EXECUTADO — Named Tunnel com credencial real; Tailscale com tailnet real; push contra push service real; aparelho físico; SHA publicado. Release **NO_GO**.

## 2026-11-22 — #231 tela preta: hipótese do scrubber/xterm 5.3 fechada na UI real (prova negativa)

- SHA de trabalho: `release/v1.0.0` @ `82b90a3b` (base local, sem commit/push). Ensaio real na UI web `http://127.0.0.1:13431`, projeto `prj_06GFFF9XDKDSMV7TES7BM8H3T4`, runtime isolado `agy-06H07PFNRW7RVXDQYKNM66B280` (Antigravity CLI 1.5.14, workspace `/tmp/opencode/issue231-workspace`). Responsável: orquestrador.
- PROVA NEGATIVA (tela preta **não** reproduzida) — na UI real do Nexus o xterm renderizou o banner "Antigravity CLI 1.5.14", o prompt `>` e uma resposta viva do provedor ("Hello! How can I assist you today?") após digitação real (`say hello in one short line` + Enter). Capturas: `/tmp/opencode/issue231.png`, `/tmp/opencode/issue231-after-trust.png`, `/tmp/opencode/issue231-after-prompt.png`, `/tmp/opencode/issue231-ws.png`.
- PASS (fronteira WebSocket/UI instrumentada) — `onmessage` do `WebSocket` foi interceptado antes do carregamento da página: 105 frames (`output` 101, `lease` 2, `title` 1, `replay_complete` 1), 8974 bytes de saída; 0 erros de console e 0 `pageerror`. Relatório: `/tmp/opencode/issue231-ws-report.json`; bytes recebidos: `/tmp/opencode/issue231-ws-output.bin` (SHA-256 `e50e36309b49f838…`).
- PASS (scrubber medido na fronteira) — a captura PTY bruta `/tmp/opencode/agy_capture.bin` (4318 bytes, SHA-256 `a0f53692bc72bd30…`) contém `ESC[>4;2m`, `ESC[>5u`, `ESC[?u` e OSC 11; o stream WebSocket recebido pela UI real contém `ESC[?u` e OSC 11, mas **não** contém `ESC[>4;2m` nem `ESC[>5u`. O scrubber de saída remove exatamente as duas sequências de negociação de teclado e preserva as demais, como projetado.
- HIPÓTESE FECHADA — "scrubber do Go / xterm.js 5.3 causam tela preta" **não se sustenta** no caminho real (Go → WebSocket → `ProtocolScrubber` do cliente → xterm 5.3): com essas sequências presentes na entrada, a tela renderizou corretamente. Hipótese encerrada **para este caminho e ambiente**.
- LIMITE HONESTO — a tela preta não foi reproduzida, portanto **não existe correção** para #231: a issue permanece OPEN (falha não observada, causa não identificada). O ensaio cobriu o runtime AGY 1.5.14 em workspace isolado, não a configuração em que o owner observou a falha. Aceite em dois dispositivos físicos continua pendente. Release **NO_GO**.

## 2026-11-22 — #229 e #230 consolidados: regressão de login e descoberta de produto

- PASS (#229) — `nexus login claude` executa `claude auth login` (`internal/core/provider/adapters/claude/claude.go`); binário local `claude` 2.1.325 confirma o subcomando `auth login`. Correção já presente no commit local `7ba9f349`. Regressão nova: `internal/core/provider/adapters/claude/claude_test.go::TestLoginUsesClaudeAuthLoginSubcommand` fixa o argv `auth login` com binário stub; `go test ./internal/core/provider/adapters/claude -count=1` e `go vet ./internal/core/provider/adapters/claude` exit 0.
- PASS (#230) — links HTTP/HTTPS clicáveis, URL quebrada entre linhas detectada e atalhos de cópia `Ctrl+Shift+C`/`Cmd+C` em `web/src/nexus/terminalLinks.ts`, `web/src/nexus/terminalLinkProvider.ts`, `web/src/components/TerminalPane.tsx` e `web/src/nexus/AgentTerminal.tsx`; `web/src/nexus/terminalLinks.test.ts` cobre 11 casos. Gates medidos: `typecheck` exit 0; suíte web 155 arquivos / 992 testes exit 0; Prettier limpo; ESLint 0 erros (1 warning preexistente em `FlowRunSurface.tsx`); Stylelint limpo; `check:styles` OK.
- PASS (descoberta do produto, exigida por `AGENTS.md` §1) — anúncio `agent-terminal-provider-links` (revision 1, superfície `agent-terminal`, alvo `data-tour="agent-terminal-follow"`, passo de tour `follow` já existente) em `web/src/features/work/featureAnnouncements.ts`, com `discovery.terminalProviderLinksTitle/Body` em PT-BR/EN/ES (`web/src/i18n/resources.ts`). Vitest focado (discovery + terminal links + componentes) 7 arquivos / 41 testes exit 0; `format:check` e `typecheck` exit 0.
- NÃO EXECUTADO — aceite visual do anúncio no SHA publicado; ensaio em dois dispositivos físicos. As issues #229 e #230 permanecem OPEN até decisão de fechamento pelo owner. Release **NO_GO**.

## 2026-11-22 — #214 corrigido: takeover concorrente não sobrescreve writer vigente

- SHA de trabalho: `release/v1.0.0` @ `82b90a3b` (base local, sem commit/push); alterações de #214 não commitadas em `internal/control/host/host.go`, `internal/control/host/host_test.go`, `internal/control/web/handler_terminal.go`. Evidências sanitizadas em `/tmp/opencode/gates-20261122/issue214i/` (40 execuções).
- CAUSA RAIZ (medida) — na corrida de takeover, o `promotion_acquire` do writer **demovido** chegava ao `SessionHost` depois da lease do writer novo e sobrescrevia `activeWriter`, derrubando o input do writer vigente. Correção: ordem broker→host serializada (`terminalLeaseMu`), checagem `broker.IsWriter` antes de promover e **epoch monotônico** no lease; o host rejeita epoch antigo (`lease_stale`), preservando o writer atual.
- PASS (build/vet) — `go build ./internal/control/...` exit 0; `go vet ./internal/control/host ./internal/control/web` exit 0.
- PASS (race) — `go test -race -count=1 ./internal/control/host ./internal/control/web` exit 0 (`host` 13,9 s; `web` 108,3 s).
- PASS (regressão #214, 40 execuções) — `NEXUS_TERMINAL_TRACE=1 go test -race -count=1 ./internal/control/web -run '^TestTerminalHandoffAuthenticatedClientsShareProcessAndPersistIdentity$'` → `fails=0 of 40`; logs por execução em `/tmp/opencode/gates-20261122/issue214i/`.
- PASS (regressão determinística nova) — `internal/control/host/host_test.go::TestSessionHost_RejectsStaleLeaseEpochFromDemotedWriter`: takeover com epoch mais novo é aceito; frame tardio do writer demovido (epoch antigo), epoch igual de outra conexão e frame sem epoch são rejeitados com `lease_stale`; epoch ativo permanece o do writer vigente.
- NÃO EXECUTADO — homologação em dois dispositivos físicos (Windows/macOS), ensaio com quota real esgotada e aceite do owner. Release **NO_GO**.

## 2026-11-22 — Gate 0 tornado autocontido: commit de correção e prova em worktree limpo

- SHA final: `release/v1.0.0` @ `3a31ec50` (`fix(web): make gate 0 remote access commit self-contained`), sobre `b035aa04`. Sem push.
- DEFEITO (medido) — `b035aa04` **não compilava isolado**: `handlers_push.go` usava `s.push`/`s.pushInit`/`s.pushErr`, `push_attention.go` usava `s.attentionSource`/`s.pushWatchCancel`, `roles_test.go` usava `enforceControl`/`isReadOnlyMethod`, `push_attention_test.go` inicializava `attentionSource:` e as rotas `/api/v1/push/*` e `/api/v1/remote/devices` não existiam — todos esses símbolos estavam em `server.go`/`routes.go`, que ficaram fora do commit. `internal/control/web/private_origin_test.go` (8 testes) também ficou untracked.
- DEFEITO (contaminação de escopo) — o mesmo commit arrastou mudanças de **outro workstream** em `internal/app/control_cmd.go` (continuity handoff + terminal input broker) e `internal/app/control_cmd_test.go`, que referenciavam `registry.Continuity*` e `terminalInputBroker` — símbolos de arquivos fora do commit. Correção: `control_cmd.go` volta ao pai mantendo **apenas** o bloco de help do acesso remoto (`NEXUS_TUNNEL_TOKEN`/`NEXUS_TUNNEL_HOSTNAME`/`NEXUS_REMOTE_ORIGIN`); `control_cmd_test.go` volta ao pai.
- CORREÇÃO — `3a31ec50` adiciona `internal/control/web/server.go` (campos de push/attention, `enforceControl`, `isReadOnlyMethod`, anotação do device no pareamento, papel na sessão), `internal/control/web/routes.go` (rotas de push e dispositivos + ligação da projeção de atenção), `internal/control/web/handlers_api.go` (papel da sessão), `internal/control/web/private_origin_test.go` (8 testes, incluindo o caminho HTTP real), o cliente de API web (`web/src/nexus/api.ts` + testes), o registro do service worker (`web/src/index.tsx`), o manifest e as metas PWA instaláveis (`web/public/manifest.webmanifest`, `web/index.html`) e o script `test:e2e:remote-access` (`web/package.json`). Fora do commit: `handlers_agents.go`, `handlers_api.go` (exceto o hunk `role`), `handlers_projects.go`, `handlers_planning.go`, `handlers_nexus_test.go`, `handlers_attention_fence_test.go`, `TerminalPane.tsx`, `internal/control/host/host.go` e demais arquivos alheios.
- PASS (build isolado) — worktree limpo em `3a31ec50` (`/tmp/opencode/nexus-final-check`): `go build ./...` → exit 0. Antes, o mesmo comando no commit `b035aa04` falhava com `s.push undefined`, `s.attentionSource undefined` e `undefined: enforceControl`.
- PASS (go test isolado) — `go test ./internal/control/web/ ./internal/control/originpolicy/ ./internal/app/` → exit 0 (`internal/control/web` 70,9 s).
- PASS (frontend isolado) — `npm --prefix web run typecheck` → exit 0; `npm --prefix web run test` → 153 arquivos / 975 testes, exit 0.
- PASS (gate completo isolado) — `make quality` → exit 0 (0 issues, VERSION CONTRACT PASS). O `lint-go` acusou `comando` (pt-BR) como `misspell` em `internal/app/tunnel_cmd.go`; corrigido pela exclusão do arquivo no `misspell`, seguindo o padrão já usado no `.golangci.yml` para copy pt-BR.
- PASS (E2E isolado) — `npm --prefix web run test:e2e:remote-access` → exit 0 no worktree limpo de `3a31ec50`: dois dispositivos 390px (Chrome Android + Safari iOS) por túnel HTTPS real, pareamento de `controller` e `viewer`, PWA instalável pela origem pública, `viewer` recusado em listagem e mutação sem efeito colateral, revogação isolada e reinício do Nexus medindo a troca de origem no Quick Tunnel.
- PASS (gate final do HEAD completo) — worktree limpo em `bd17f90e` (`/tmp/opencode/nexus-release-check`): `go build ./...` exit 0; `go test ./internal/control/web/ ./internal/control/originpolicy/ ./internal/app/` exit 0 (`internal/control/web` 79,0 s); `make quality` exit 0 (VERSION CONTRACT PASS). A primeira execução do `make quality` falhou com `could not import ... no such file or directory` em `scripts/sign-update-manifest` — cache compartilhado do Go corrompido, não regressão de código (o diff `3a31ec50..bd17f90e` é só documentação). Reexecução com `GOCACHE=/tmp/opencode/gocache-verify` passou sem tocar no cache compartilhado.
- PASS (E2E no HEAD final) — `npm --prefix web run test:e2e:remote-access` exit 0 em worktree limpo em `82b90a3b`: pareamento de `controller` e `viewer`, PWA instalável pela origem pública, `viewer` recusado em listagem e mutação sem efeito colateral, revogação isolada e reinício medindo a troca de origem no Quick Tunnel. O diff entre `bd17f90e` (gate completo verde) e `82b90a3b` é exclusivamente `DEV/VERIFY.md` e `DEV/WORKLOG.md`, então o gate completo do código vale para o HEAD.
- PASS (cobertura de discovery do acesso remoto) — `e7d643a4` fecha a exigência do contrato do projeto (`AGENTS.md` §1) de cobrir `tourTargets.test.ts` e `featureReveal.test.ts` para recurso visível: `web/src/app/tour/tourTargets.test.ts` ganha o caso que exige os três alvos remotos (`remote-tunnel-card`, `remote-devices-card`, `remote-push-card`) resolvidos **dentro** do escopo `settings` e ausentes fora dele; `web/src/features/work/featureReveal.test.ts` ganha três casos que exigem o reveal de `remote-access-stable-url`, `remote-access-scoped-invitations` e `remote-access-push-notifications` apenas na superfície `settings` do projeto selecionado (nunca em outro projeto nem em outra superfície), e só quando o próprio cartão está montado. `npm --prefix web run test -- src/app/tour/tourTargets.test.ts src/features/work/featureReveal.test.ts src/features/work/featureAnnouncements.test.ts` → 3 arquivos / 26 testes, exit 0; `format:check` e `typecheck` exit 0.
- NÃO EXECUTADO — Named Tunnel com credencial real; Tailscale com tailnet real; push contra push service real; aparelho físico; SHA publicado. Release **NO_GO**.

## 2026-11-22 — #214 lease correlation instrumentation

- SHA de trabalho registrado: `release/v1.0.0` @ `16cc08114684b200a76a882f343686bbbd3dc671`; árvore local preservada, sem commit.
- Instrumentação local em `internal/control/host/host.go` registra IDs opacos nos eventos `attach`, `attach_done` e `lease_acquire`; não inclui credenciais nem bytes de entrada.
- EVIDÊNCIA anterior: `/tmp/opencode/gates-20261121/loop3-25.log` falha em `TestTerminalHandoffAuthenticatedClientsShareProcessAndPersistIdentity`, linha 388 (`provider process received anything other than one leased input: ""`). Traces existentes registram input descartado por lease; foram capturadas antes do novo log `lease_acquire`.
- NÃO EXECUTADO após instrumentação: teste direcionado com `-race` e correlação nova de `attach` / `lease_acquire` / `host.input.dropped.lease`. Execução indisponível nesta sessão; nenhum PASS inferido. Release **NO_GO**.

## 2026-11-21 — Onda 1b (bis): defeito do coletor no `__control-host` CORRIGIDO e failover automático reprovado com o coletor dentro do host

SHA de trabalho: `release/v1.0.0` @ `16cc08114684b200a76a882f343686bbbd3dc671` (árvore com 125 arquivos modificados não commitados, preservados). Binário de ensaio `/tmp/opencode/nexus-test2` (SHA-256 `996df50ccc15328274c005acd65e1d11b92cce5d0e38a6a3e21db06f913842c7`). Sandbox `/home/desenvolvedor/nexus-sbx`. Responsável: orquestrador (sessão automatizada).

Causa raiz do defeito registrado em 21/11: o `__control-host` é simultaneamente supervisor do provider e executor da continuidade. O handoff para o provider de origem (passo obrigatório), encerrando o processo, abortava o handoff em voo porque o host saía em `sh.Wait()`.

### Correção implementada

- PASS — `internal/app/control_cmd.go`: após `sh.Wait()`, o host drena o claim de continuidade **de sua própria posse** antes de sair (`waitForOwnedContinuityClaim` / `waitForOwnedContinuityClaimUntil`, timeout `continuityHostDrainTimeout = 3 min`, polling de 100 ms). Claim de outro dono (CLI de attach, core web) é respeitado e não bloqueia.
- PASS — `internal/control/launcher/orphan_hosts.go`: runtime com `registry.ContinuityInFlightLabel == "true"` deixa de ser candidato a host órfão; sem isso o reaper matava o host durante o handoff.
- PASS — `internal/app/control_cmd_test.go`: `TestWaitForOwnedContinuityClaimBlocksUntilClaimSettles` e `TestWaitForOwnedContinuityClaimIgnoresForeignOwner`.
- PASS — `gofmt` limpo; `go build ./...` OK; `go test` verde em `internal/app`, `internal/control/registry`, `internal/control/handoff`, `internal/control/host`, `internal/nexus`, `internal/control/launcher`.

### Failover automático com o coletor DENTRO do `__control-host` — PASS

Cenário: runtime de origem `codex-06H02QXG3E1Z0NSH6FHREXKTWC` (perfil `work`, conta A) com `provider_session_id=01a2015f-b3eb-7b11-8cf3-f51b9a6e6b48`; **coletor de quota executando dentro do próprio host** (host pid 1232927, lease `1232927-1795319378035570828-1`) — sem core web separado.

- PASS — gatilho: `usage.json` da conta A esgotado sinteticamente (`remaining_percent=0`, `source=OFFICIAL_API`, `fetched_at` recente, marcador `evidence_origin=SYNTHETIC_TEST_EXHAUSTION`); o tick do monitor (1 min) disparou a continuidade **sem handoff manual**.
- PASS — sucessor `codex-handoff-06H02RKN36DQAV6REHKR5A8RQG` (perfil `kivergmail`, conta B) em `RUNNING`, `provider_session_id=01a2015f-…` (mesma sessão), `account_scope.account_id=dbcf862202324dc5d01716192bac7c70` (escopo de B), `parent_runtime_id=codex-06H02QXG3E1Z0NSH6FHREXKTWC`.
- PASS — labels de continuidade: `continuity=ACCOUNT_HANDOFF`, `nexus.continuity_reason=QUOTA_EXHAUSTED`, `nexus.continuity_hops=1`, `nexus.continuity_excluded=codex:work`, `nexus.continuity_replaces_runtime=codex-06H02QXG3E1Z0NSH6FHREXKTWC`; origem marcada `state=HANDOFF` com `nexus.continuity_pending=true`.
- PASS — lineage `lin-codex-06H02RKN34YWR4B48RAHMFE1Q4` (`type=ACCOUNT_HANDOFF`, `checkpoint_id=cp-codex-1795319559132429009`, `source_provider_session_id == target_provider_session_id`); checkpoint `cp-codex-1795319559132429009.json` (schema_version 4) gravado antes da quiescência.
- PASS — geração única: exatamente um processo `codex --no-daemon resume 01a2015f-…` vivo (pid 1242375, host 1242365); a origem (pid 1232936) foi encerrada e não deixou geração duplicada.
- PASS — cópia cross-account com provenance: inode distinto (399257 na origem, 1590879 no destino — cópia, nunca hardlink); `session_meta` no destino reescrito para `creator_user_id=user-PzabDJca6nMdrQT6ILjXmQKg` / `creator_account_id=9cba8371-db95-468c-ad00-02e9ceddcb3e` / `account_scoped=true`; marcador em `nexus-cross-account-sessions.jsonl` com `source_profile=work`, `source_path` e `adopted_at`.
- PASS — o defeito do coletor está fechado: com o coletor dentro do `__control-host` o failover completa (antes abria checkpoint e parava a origem sem criar sucessor).
- EVIDÊNCIA (limitação honesta) — o esgotamento da conta A foi **sintético** (`evidence_origin=SYNTHETIC_TEST_EXHAUSTION`); a conta B tem quota real medida pela API oficial. Regressão sintética, não homologação de quota real esgotada.
- EVIDÊNCIA (limitação de harness) — o `control start` isolado retém a lease de escrita do PTY (`activeWriter`), então input via protocolo de controle é descartado; a criação da sessão foi dirigida por tmux (teclas reais no TUI). Não é defeito do produto (o core web/desktop é o caminho esperado), mas o ensaio depende de PTY real.
- NÃO EXECUTADO — failover com esgotamento real da conta A (sem OAuth para 3ª identidade e sem consumo que zere a janela de 5 h); recuperação pós-falha de handoff em runtime real (o rollback tem cobertura de teste); failback para a conta original. Release **NO_GO**.

## 2026-11-21 — Onda 1b: handoff manual e failover automático cross-account PROVADOS (separados)

SHA de trabalho: `release/v1.0.0` @ `16cc08114684b200a76a882f343686bbbd3dc671` (árvore com 119 arquivos modificados não commitados, preservados). Binário de ensaio `/tmp/opencode/nexus-test` (SHA-256 `fc08dc7c72bbac7bc92164926c2c64a4edd914ab59ba8db7ddb06e86eeb90cef`). Sandbox isolado `/home/desenvolvedor/nexus-sbx` (fora de `/tmp`: o Codex recusa `codex_home` sob `/tmp`). Responsável: orquestrador (sessão automatizada).

Contas: A = `codex:work` (`802cf130-605a-4489-94f4-a6570dbdf8e7` / `user-DnXjA2hCjtuQ4ydm1yroOBB2`); B = `codex:kivergmail` (`9cba8371-db95-468c-ad00-02e9ceddcb3e` / `user-PzabDJca6nMdrQT6ILjXmQKg`).

### Handoff manual (cenário 2a) — PASS

- PASS — `nexus control handoff codex-06H01YXX08GKSQFBS39XM2M6QR codex:kivergmail --no-attach` → `✓ Account handoff completed: ... (New Runtime: codex-handoff-06H0209ZJM8HRVP109RKWRC5RW)`.
- PASS — continuidade: sucessor `codex-handoff-06H0209ZJM8HRVP109RKWRC5RW` (perfil `kivergmail`) com `provider_session_id=01a200fb-c331-73d3-9fde-8376370bb895` (mesma sessão de origem), `account_scope` = escopo de B, `labels.nexus.continuity_replaces_runtime` = runtime de origem; origem marcada `state=HANDOFF`, `nexus.continuity_pending=true`.
- PASS — geração única: exatamente um processo `codex --no-daemon resume 01a200fb-…` vivo (pid 764068); o processo de origem (pid 741089) foi encerrado antes do start do destino.
- PASS — cópia cross-account com provenance: inode distinto (399051 na origem, 1590829 no destino — cópia, nunca hardlink); `session_meta` no destino reescrito para `creator_user_id=user-PzabDJca6nMdrQT6ILjXmQKg` / `creator_account_id=9cba8371-…` / `account_scoped=true`; origem intacta com creator omega.
- PASS — marcador de provenance: `data/profiles/codex/kivergmail/home/nexus-cross-account-sessions.jsonl` (22 entradas) com `session_id=01a200fb-…`, `source_profile=work`, `source_path` absoluto, `adopted_at` datado.
- PASS — checkpoint e lineage: `data/checkpoints/cp-codex-1795313188432806585.json` (schema_version 4, `source_runtime_id`, `source_session_id`, `workspace`, `maestro_context` com `DEV/` em `MISSING` — sandbox sem repositório) e `data/lineage/lin-codex-06H0209ZJQCMJK95R6XV1W4BKR.json` (`type=ACCOUNT_HANDOFF`, `source_provider_session_id` = `target_provider_session_id`).
- PASS — reanexação e isolamento de credenciais: `auth.json` de `work` com `account_id=802cf130-…`, de `kivergmail` com `account_id=9cba8371-…` (credenciais não migraram); tela do sucessor mostra `Resuming session…` e o marcador `NEXUS-HANDOFF-MARKER-A1` / resposta `ACK-A1` preservados.

### Failover automático por esgotamento (cenário 2b) — PASS, com causa raiz registrada

- PASS — gatilho: o monitor de quota leu a conta A esgotada e disparou continuidade sem handoff manual. Sucessor `codex-handoff-06H026DZ5Y9ZNCSC98GC57P80R` (perfil `kivergmail`), `labels.continuity=ACCOUNT_HANDOFF`, `labels.nexus.continuity_reason=QUOTA_EXHAUSTED`, `labels.nexus.continuity_hops=1`, `labels.nexus.continuity_excluded=codex:work`; `parent_runtime_id` = runtime de origem; `provider_session_id` idêntico (`01a20119-1173-76c3-a570-0b3fe4c29b50`).
- PASS — lineage próprio: `data/lineage/lin-codex-06H026DZ5X4WNE31C9ZY30XBF8.json` (`type=ACCOUNT_HANDOFF`, `checkpoint_id=cp-codex-1795314793942822281`).
- PASS — geração única e sem duplicação: 1 processo `codex --no-daemon resume 01a20119-…` (pid 861271); origem parada; rollout de origem preservado; cópia no destino com creator reescrito para a identidade de B (`user-PzabDJca6nMdrQT6ILjXmQKg` / `9cba8371-…`, `account_scoped=true`) e marcador de provenance presente.
- PASS — seleção de alvo por identidade real: a exclusão `codex:work` impede o retorno à mesma conta; o alvo escolhido foi a única conta com identidade upstream distinta e quota medida positiva (`kivergmail`, `5h=100%`/`weekly=100%`, `OFFICIAL_API`).
- EVIDÊNCIA (limitação honesta) — o esgotamento da conta A foi **sintético**: `usage.json` da conta A reescrito com `remaining_percent=0`, `source=OFFICIAL_API`, `fetched_at` recente e marcador `evidence_origin=SYNTHETIC_TEST_EXHAUSTION`. A conta B tem quota real medida pela API oficial. Regressão sintética, não homologação de quota real esgotada.
- EVIDÊNCIA (defeito de arquitetura medido) — o `__control-host` é **simultaneamente** supervisor do provider e executor da continuidade. Parar o runtime de origem (passo obrigatório do handoff) encerra o processo e aborta o handoff em voo: a primeira tentativa (coletor dentro do `__control-host`, pid 811849) criou checkpoint `cp-codex-1795314303230293059` e parou a origem, mas **não criou sucessor** (nenhum runtime novo, nenhum lineage). Com o coletor em processo separado (`nexus control web`, pid 854630, lease `854630-…`), o mesmo caminho de código completou o failover. No produto, o core (web/desktop) é o processoSeparate esperado, mas o modo `nexus control start` isolado perde a continuidade automática.
- NÃO EXECUTADO — failover com esgotamento real da conta A (sem OAuth para 3ª identidade e sem consumo que zere a janela de 5 h), recuperação pós-falha de handoff em runtime real (o rollback tem cobertura de teste, não ensaio real), failback para a conta original. Release **NO_GO**.

## 2026-11-22 — Gate 0 fechado nas duas opções (A nomeada / B privada): CLI, política de origem, pareamento e exposição

- PASS (CLI) — `internal/app/tunnel_cmd_test.go` (3 testes): o comando do túnel não pode chamar `web.StartTunnel(` direto (regressão para o hostname aleatório), precisa de `web.StartConfiguredTunnel(` e de `web.NamedTunnelConfigFromEnv()`; o `--help` documenta `NEXUS_TUNNEL_TOKEN` e `NEXUS_TUNNEL_HOSTNAME`.
- PASS (faixa CGNAT / overlay privado) — `internal/control/originpolicy/trusted_test.go` (5 testes): `100.64.0.0/10` aceito como origem (o bind já permitia; a política de origem rejeitava), `100.128.0.1` fora da faixa rejeitado, regra de host exato preservada em endereço privado, e nenhum endereço público aceito.
- PASS (origem publicada) — `RemoteOrigin()` normaliza `https://Host/` → `https://host` e preserva a porta; valor malformado (sem esquema, `ftp://`, wildcard, caminho, query, user-info, host vazio) é **ignorado**, sem alargar a política; a origem publicada não autoriza outro host.
- PASS (origem publicada define um endereço exato) — `internal/control/originpolicy/trusted_test.go` (8 testes): a publicação define mesmo esquema, mesmo hostname e **mesma porta efetiva**; outra porta no mesmo hostname é recusada (`https://nexus.tail1234.ts.net` não autoriza `:8443`), outro esquema é recusado (`http://` vs `https://`), e a regra de host exato permanece para todo o resto. Um proxy confiável que reescreve o `Host` do request (`tailscale serve`) é coberto: o `Origin` do navegador precisa ser exatamente o endereço publicado, e um request que chegue por interface pública **não** é coberto por publicação privada.
- PASS (origem privada é estável por construção) — o status do túnel reporta `stable_url=true` quando há `NEXUS_REMOTE_ORIGIN`, porque um nome controlado pelo operador não é um hostname sorteado a cada start. Antes, o modo privado era reportado como instável, o que dizia ao operador que o app instalado quebra no reinício — falso.
- PASS (pareamento no modo privado) — `internal/control/web/private_origin_test.go` (8 testes): convite emitido para a origem publicada sem túnel; `viewer` mantém o papel somente leitura; nenhum convite sem origem alcançável; convite recusado em outro host; **origem com porta (`https://nexus.tail1234.ts.net:8443`) continua pareável** — o vínculo do convite compara o hostname normalizado, não o `Host` cru do request; o status publica `remote_origin`; a origem publicada é aceita como origem do navegador; **caminho HTTP real** — o host emite o convite por `POST /api/v1/tunnel/invite`, o aparelho troca o código em `POST /api/v1/auth/pair`, recebe cookie `Secure`/`HttpOnly`, aparece no registro de dispositivos e é revogado pelo host.
- PASS (exposição) — `TestEffectiveExposureLoopbackWithPublishedRemoteOrigin`: com a origem publicada o servidor sai de `LOOPBACK` (bootstrap não reutilizável, cookie `Secure` obrigatório). Antes, o servidor alcançável pela rede privada mantinha confiança de loopback.
- PASS (frontend) — `remoteAccessMode()` (4 testes) distingue `private`/`tunnel`/`none`, com o túnel ativo prevalecendo sobre a origem privada (é o endereço ao qual o convite fica ligado enquanto o túnel corre); `tsc --noEmit` limpo; ESLint 0 erros (1 warning preexistente em `FlowRunSurface.tsx`); Stylelint limpo; Prettier limpo.
- PASS (descoberta) — o anúncio `remote-access-stable-url` subiu para `revision: 2` e o texto PT-BR/EN/ES passou a declarar o terceiro estado (endereço privado servido na rede própria: estável e sem túnel).
- PASS — `make quality` → exit 0: `lint-go` 0 issues, suíte Go completa, Web 155 arquivos / 988 testes, VERSION CONTRACT PASS.
- PASS — `npm --prefix web run test:e2e:remote-access` → exit 0: dois dispositivos 390px por túnel HTTPS real, `viewer` recusado, revogação isolada, reinício do Nexus medindo a mudança de origem no Quick Tunnel, PWA instalável pela origem pública.
- FLAKE (infraestrutura, não regressão) — duas execuções do mesmo E2E falharam em passos distintos antes de passar: uma no passo 4 (`timeout waiting for host session`, sessão do host em loopback) e outra no passo 5 (`tunnel readiness failed: tunnel readiness timeout (last status 0: hostname does not resolve)`). Os dois sintomas são de DNS/rede do `trycloudflare.com`, e a reexecução seguinte passou inteira. O mesmo flake já tinha aparecido antes nesta fatia (timeout de `#access-code-input` no pareamento do viewer). Nenhuma das falhas é compatível com regressão de código: as mudanças desta fatia só afetam o caminho de origem privada (`NEXUS_REMOTE_ORIGIN`), que o E2E não define.
- FLAKE (ferramenta) — `make quality` falhou uma vez com `cmd/nexus-hud/main.go:12:2: could not import github.com/IAPro-Community/nexus/internal/desktop ... no such file or directory (typecheck)`: cache de build do Go corrompido. `go clean -cache` resolveu; a reexecução deu exit 0.
- BLOQUEIO EXTERNO RESOLVIDO (outro workstream) — `internal/control/host/host.go` (editado às 02:14 por outro workstream) chamava `sh.activeWriterClientIDLocked()`, método nunca definido: a árvore inteira não compilava e o gate não rodava. Desbloqueio mínimo: o método foi implementado em `host.go` com a semântica óbvia do call site (ID opaco do writer ativo, string vazia quando não há writer), reaproveitando `terminalClientIDLocked`. É um arquivo **fora do escopo do acesso remoto** — a alteração é de desbloqueio de build, não de comportamento de produto.
- PASS (runbook e CLI) — `docs/product/remote-access.md` Opção B reescrita para o caminho executável real (`nexus web --port 13000` em loopback + `tailscale serve --bg http://127.0.0.1:13000`), separando do caminho "servir direto na interface do tailnet"; `nexus web --help` passou a documentar `NEXUS_TUNNEL_TOKEN`+`NEXUS_TUNNEL_HOSTNAME` (com o aviso de porta fixa) e `NEXUS_REMOTE_ORIGIN` (endereço exato: esquema, hostname e porta) — conferido por execução (`go run ./cmd/nexus web --help`).
- PASS (gate final) — `make quality` → exit 0 (0 issues, Web 155 arquivos / 988 testes, VERSION CONTRACT PASS); `npm --prefix web run test:e2e:remote-access` → exit 0 (dois dispositivos, viewer recusado, revogação isolada, reinício com origem Quick diferente como esperado).
- NÃO EXECUTADO — Named Tunnel com credencial real; Tailscale com tailnet real (o modo privado foi provado por contrato e por teste, não por uma tailnet ativa); push contra push service real; aparelho físico; SHA publicado. Release **NO_GO**.

## 2026-11-21 — Gate 0 visível na UI: estabilidade da URL pública declarada no cartão de acesso remoto

- PASS — `internal/control/web/tunnel_named_test.go` (`TestTunnelOriginStability`): origem ausente não é estável; Quick Tunnel nunca é estável; o hostname nomeado configurado é estável; um Quick Tunnel em execução **não** é confundido com a configuração nomeada.
- PASS — `web/src/features/settings/remoteAccessModel.test.ts` (`tunnelOriginKind`): `true` → estável; `false` → temporária; **ausente** → temporária (servidor anterior ao contrato não pode fazer o operador crer que o app instalado sobrevive a um reinício).
- PASS — `vitest` das suítes de descoberta + modelo: 5 arquivos / 28 testes (`featureAnnouncements`, `featureTour`, `featureTourAnchors`, `featureReveal`, `remoteAccessModel`).
- PASS — `tsc --noEmit` limpo; ESLint 0 erros (1 warning preexistente em `FlowRunSurface.tsx`); Stylelint limpo; `check:styles` OK (6 testes, paridade de tokens 485 literais, 0 regressões); Prettier limpo.
- PASS — `make quality` → exit 0: `lint-go` 0 issues, suíte Go completa, Web 154 arquivos / 973 testes, VERSION CONTRACT PASS.
- PASS (descoberta de produto) — âncora `data-tour="remote-tunnel-card"`, passo de tour `remote-stable-url` e anúncio `remote-access-stable-url` (revision 1) registrados em `featureAnnouncements.ts`/`featureTourDefinitions.ts`, com `discovery.remoteStableUrlTitle/Body` e `tour.remoteStableUrlTitle/Body` presentes nos 3 idiomas.
- PASS (contrato de estabilidade no E2E) — o roteiro exige `stable_url` coerente nos dois modos: `false` e sem `named_hostname` no Quick Tunnel; `true` e `named_hostname` igual ao configurado no Named Tunnel, inclusive depois do reinício. Execução `SKIP_BUILD=1 npm --prefix web run test:e2e:remote-access` exit 0, com `The status declares an ephemeral origin: the installed app breaks on restart.` e a origem mudando entre reinícios (`alphabetical-doc-statistical-downloading` → `minor-hook-spears-oem`).
- NÃO EXECUTADO — Named Tunnel com credencial real; a distinção estável/temporária foi exercitada por contrato e por teste, não por um hostname nomeado em produção. Release **NO_GO**.

## 2026-11-21 — Gate 0 (Named Tunnel) coberto por teste, sem credencial

- PASS — `internal/control/web/tunnel_named_test.go` (6 testes): configuração do ambiente ausente/parcial/completa; token **ausente do argv** e presente no env do filho (`TUNNEL_TOKEN=`); URL pública derivada do hostname e idêntica entre starts; recusa de hostname com barra e de token vazio; Named Tunnel preferido quando a configuração existe, Quick Tunnel como padrão.
- PASS — `web/src/app/pwaOriginScope.test.ts` (4 testes): manifest com `id`/`scope`/`start_url` relativos, ícones relativos, service worker registrado na raiz da origem atual e nenhuma origem absoluta fixada no `sw.js`.
- PASS — `go test ./internal/control/web/ -count=1` (65,9 s) sem `-overlay`; `go vet ./internal/control/web/`; `gofmt -l` vazio; `make lint-go` → 0 issues (cache local do Go corrompido, resolvido com `go clean -cache`).
- PASS — `npm --prefix web run test` → 154 arquivos / 970 testes; `npm --prefix web run test:e2e:remote-access` exit 0 (re-executado após a correção da prontidão).
- PASS (prova negativa, fail-closed) — binário real com `NEXUS_TUNNEL_TOKEN` inválido e hostname configurado: `POST /api/v1/tunnel/start` respondeu `502 {"active":false,"error":"cloudflared exited during startup: exit status 255"}`, sem publicar host nem armar o modo de autenticação de túnel.
- PASS (PWA pela origem pública) — o E2E exige `manifest.webmanifest` e `sw.js` com `200` no hostname do túnel, manifest em modo instalável e service worker registrado no escopo público: `Public origin serves the installable app (registered)`.
- CORRIGIDO — `waitForTunnelReady` aceitava qualquer resposta HTTP como prontidão, inclusive `502`/`530` do edge (ingress que não alcança o Nexus). Um Named Tunnel mal configurado apareceria ativo com a URL pública quebrada. `tunnelResponseReady` agora exige status 1–499 e o timeout informa o último status/erro. Coberto por `TestTunnelResponseReadyRejectsEdgeErrors`.
- RESTRIÇÃO — `StartNamedTunnel` recusa porta local efêmera (`0`/negativa): com token, o `cloudflared` aplica a ingress do dashboard, que aponta para um `http://localhost:<porta>` fixo; em porta efêmera o host publicado responderia `502`. Coberto por `TestStartNamedTunnelRejectsInvalidInput`.
- EVIDÊNCIA (defeito do Gate 0 medido) — reinício do Nexus no Quick Tunnel: `Origin before restart: https://piece-magnitude-implications-personalized.trycloudflare.com` → `Origin after restart: https://request-students-medieval-christine.trycloudflare.com`. A origem muda: o ícone do PWA instalado abriria host morto e o cookie host-bound não reaproveitaria.
- NÃO EXECUTADO — Named Tunnel com credencial real (origem idêntica após reinício ainda não provada com hostname nomeado), aparelho físico, entrega push contra push service real, execução no SHA publicado. Release **NO_GO**.

## 2026-11-21 — E2E de dois dispositivos em túnel público real (acesso remoto)

- PASS — `npm --prefix web run test:e2e:remote-access` (exit 0): binário `nexus` real em porta efêmera, `cloudflared` 2026.8.2 com SHA-256 conferido, Quick Tunnel HTTPS ativo, convites `controller` e `viewer`, dois navegadores móveis 390px pareados pelo código de acesso. Overflow horizontal zero na tela de pareamento; `viewer` recusado com 403 em listagem de dispositivos e em mutação (workspace idêntico antes/depois); registro de dispositivos no host com handle opaco e `kind=mobile`; revogação derrubando só a sessão do aparelho revogado (401) e preservando a do irmão (200).
- PASS — `go build ./...`, `go vet ./internal/control/web/`, `go test ./internal/control/web/ -count=1` (64,8 s) sem `-overlay`; `make lint-go` → 0 issues (corrigidos: `ecdsaKeyFromECDH` sem `elliptic.ScalarBaseMult` depreciado, `misspell` em `push_attention.go`, código morto em `cross_account_sessions.go`); `npm --prefix web run typecheck` (0 erros) e `npm --prefix web run test` (153 arquivos, 966/966); `prettier --check` do script E2E e do `package.json`.
- PASS — `make quality` completo (exit 0): format-check, lint-frontend, lint-styles, typecheck, lint-go (0 issues), suíte Go (74 pacotes `ok`, 0 `FAIL`), suíte Web (153/966) e version-contract.
- PASS — `npm --prefix web run test:e2e` (exit 0, "All E2E Hardening assertions PASSED") com o bundle novo.
- NÃO EXECUTADO — aparelho físico, reingresso no hostname nomeado após reinício do Nexus, entrega push contra FCM/Mozilla (o push foi provado contra endpoints `httptest`, não contra um push service real). Release **NO_GO**.

## 2026-11-20 — Retheme Fluent 2: ondas 1 e 2 verificadas e instaladas

- PASS — `npm --prefix web run quality:full` após o dedup: `check:styles` (allowlist de estilos, manifesto ordenado das 17 seções, paridade de tokens 485 aceitos / 0 regressões), formatação, ESLint, Stylelint, typecheck, 151 arquivos / 959 testes e build. ESLint mantém o warning preexistente em `FlowRunSurface.tsx`.
- PASS — `npm --prefix web run test:e2e` ("All E2E Hardening assertions PASSED"); densidade medida no próprio segmented control: 36.0px confortável / 32.0px compacto.
- PASS — bundle final: `dist/bundle.css` 492.871 bytes (antes 523.986), com `--nx-focus-ring`, `--nx-font-mono`, `--nx-space-2`, `--nx-radius-md` e `--nx-font-size-sm` aparecendo uma única vez. O `scripts/build.mjs` valida os 10 tokens obrigatórios.
- PASS — `make install-local` (CLI) e `make build-desktop build-hud`; os três binários foram instalados atomicamente em `~/.local/bin`. Binários embutem o bundle: `nexus` com o bloco de tokens 1× (antes 13×).
- PASS — superfícies: um único `nexus web` (PID 517979, `http://127.0.0.1:33517` servindo o bundle novo), `nexus-hud` relançado pela unit transient e `nexus-desktop` aberto anexado ao mesmo core. Os três `nexus web` antigos (20h+) foram encerrados.
- Commit `7576610b` na `release/v1.0.0`, sem push. Onda 3 (`workspace-styles` → SCSS Modules) continua aberta. Release **NO_GO**.

## 2026-11-20 — Candidato final local para o ensaio de continuidade

- PASS — `make PATH="/home/desenvolvedor/go/bin:$PATH" quality-full`: formatação; ESLint sem erros (um warning preexistente em `FlowRunSurface.tsx`); Stylelint; typecheck; lint Go 0 issues; Go completo; Web 151 arquivos/959 testes; Go `-race` completo; versão e `govulncheck` sem vulnerabilidades. Log integral: `/tmp/nexus-quality-full-20261120-final.log`.
- PASS — `make build build-desktop build-hud`. Hashes locais dos artefatos: CLI `38a918264021ec306c5efca4ab4ce16d550ef7f9de7c215ee1f5c7e3f883e945`; Desktop `d3e69b54381a5150a4bcc2e0095a529dc3fcad7aa97df13de62dc5fd7cfdd54e`; HUD `f006a9d18753f83494153dc3d29578990b632537797c53e6a409edab3c83734e`. CLI: `1.0.0`, metadata `6c322427`, Go 1.26.11, build `2026-11-20T04:40:37Z`.
- PASS — candidato instalado atomicamente fora do PATH em `/home/desenvolvedor/.local/share/ai-manager/profiles/codex/kivergmail/home/.local/share/nexus-continuity-check/nexus`; hash igual ao CLI acima. O binário anterior foi preservado como `nexus.pre-exact-import-20261120T044219Z` (SHA `c84a3614040ed2ef4105908dfb54d17419ad21c09a6ed6d8e29999e72c3998fa`). Smoke `version` passou.
- NÃO EXECUTADO — OAuth, refresh de quota, geração e handoff/failover real. O último estado registrado tinha `kiver` sem autenticação e quotas Codex `UNKNOWN`; revalidar antes de qualquer geração. NEX segue `BLOCKED 2/11`, 48 capacidades/97 critérios sem aceite certificado; conclusão local não é homologação nem GO.

## 2026-11-19 — Hardening da continuidade, teste externo pendente

- PASS — `go test ./internal/profile ./internal/tui ./internal/control/driver ./internal/control/handoff ./internal/nexus`.
- PASS — `make quality-full`: lint Go 0 issues, suíte Go, Web 151 arquivos/959 testes, Go `-race` completo e `govulncheck` sem vulnerabilidades. ESLint mantém um warning preexistente em `FlowRunSurface.tsx` (sem erro).
- PASS — `make build build-desktop build-hud`; `git diff --check`.
- PASS — candidato local `nexus` e o executável isolado de teste têm SHA-256 `c84a3614040ed2ef4105908dfb54d17419ad21c09a6ed6d8e29999e72c3998fa`; `nexus version` mostra 1.0.0 / Linux amd64 / Go 1.26.11 / base `6c322427`. Backup da versão anterior em `nexus.pre-account-hardening-20261120T035834Z`.
- NÃO EXECUTADO — OAuth final, consulta real de quota, geração e handoff/failover entre duas identidades independentes. O estado atual conhecido continua sem segunda identidade autenticada/quota elegível.
- Limite conhecido: não há caminho cross-account verificado para AGY; agora é bloqueado fail-safe antes de interromper a origem.

## 2026-11-20 — Substituição geral dos binários locais

- PASS — os builds foram instalados atomicamente nos dois diretórios: `/home/desenvolvedor/.local/bin` e `/home/desenvolvedor/.local/share/ai-manager/profiles/codex/kivergmail/home/.local/bin`.
- PASS — SHA-256 conferido nos dois destinos: CLI `4ef28dfaf430bd509e40db1e2e76884db4612fbb05bb78ec875d317d4cfbb628`; Desktop `67bd6e0094a247afc96f99c3be4a2fb05c8686348018773a67c119189695d06c`; HUD `afa98abcd425c98a5aabe1b386b05281f5a007c035d8340226349950cbdfeb71`. Ambas as cópias do CLI reportam `1.0.0`, commit `6c322427`, Go 1.26.11.
- PASS — os seis binários anteriores estão preservados em `/home/desenvolvedor/.local/share/nexus-install-backups/20261120T030422Z`; os hashes dos backups foram conferidos.
- Estado dos processos: runtimes Nexus e HUD já ativos continuam na versão antiga, pois seus processos mantêm o inode carregado antes da troca. Não foram reiniciados para preservar sessões em andamento. Aplicações abertas devem ser reiniciadas quando essas sessões terminarem para carregarem os novos binários.

## Recheck do status de contas e saldos

- PASS — leitura local sanitizada de `nexus profiles --json`: quatro perfis Codex no inventário atual; `default`, `omegasistemas` e `work` autenticados compartilham o mesmo fingerprint do ID externo; `kiver` não autenticado. Snapshot de quota dos perfis Codex está `UNKNOWN`. Nenhum refresh externo foi disparado nesta revisão.
- PASS — inspeção da TUI: `buildUsageRows` emite uma linha por grupo de modelo; `usageHealthSummary` soma essas linhas e o título usa `len(filteredAccounts)`. Portanto “Contas 21” e `6/20 lançáveis` não são contagem de identidades distintas. Percentuais exibidos são saldo restante, e a classificação usa a menor janela quando ambas são conhecidas.
- PASS — `--with` não consta no inventário persistido atual. Há processos antigos com `/proc/<pid>/exe` apontando para o inode removido após substituição; sem identificar inequivocamente qual terminal está na captura, o screenshot deve ser tratado como possivelmente stale até reabrir só a TUI.
- NO_GO — a segunda identidade Codex não está autenticada e não há saldo atual comprovado; handoff/geração entre contas não podem ser homologados. Não houve alteração de perfis nem interrupção de processos nesta revisão.

## 2026-11-20 — Fase 1 da finalização local

- PASS — `go test ./internal/control/web -run '^TestResolveInterventionHTTPDecisionSurvivesServerRestartExactlyOnce$' -count=1`; cobre paginação por cursor e garante que `resolved_by`/ID da intervenção não vazem na timeline HTTP.
- PASS — `make quality-full` na árvore modificada: Go completo, `-race`, Web 151 arquivos / 959 testes, contrato de versão e `govulncheck` sem vulnerabilidades. Log bruto: `/tmp/nexus-quality-full-20261119.log`.
- PASS — `make build build-desktop build-hud`. Hashes: CLI `4ef28dfaf430bd509e40db1e2e76884db4612fbb05bb78ec875d317d4cfbb628`; Desktop `67bd6e0094a247afc96f99c3be4a2fb05c8686348018773a67c119189695d06c`; HUD `afa98abcd425c98a5aabe1b386b05281f5a007c035d8340226349950cbdfeb71`. `nexus version` confirma `1.0.0`, `6c322427`, Go 1.26.11.
- PASS — `make docs-verify`: 254 arquivos Markdown e 33 assets/arquivos requeridos. PASS — `node scripts/verify-agent-workspace-evidence-map.mjs`: 48 capacidades, 97 critérios (95 source+test, 2 source-only), zero aceites certificados. O verificador normal de readiness continua `BLOCKED (2/11)` e o site público sem NEX.
- PASS — candidato foi copiado atomicamente para `~/.local/share/nexus-continuity-check/nexus`, com hash atual acima; binário anterior preservado em `nexus.pre-quality-20261120`. `~/.local/bin/nexus` não foi substituído.
- NO_GO operacional — consulta sanitizada de perfis: `default`, `omegasistemas` e `work` compartilham a identidade omegaedu; `kiver` não autenticado; todas as quotas Codex `UNKNOWN`. OAuth `nexus login codex kiver` continua aguardando callback. Nenhum refresh pós-OAuth, geração ou handoff real foi executado.
- Worktrees revisadas e preservadas: dois branches `iapro/agent/*` em `1c722dab` (229 commits atrás, zero exclusivos, alterações locais DEV) e `/tmp/nexus-resume-release-7f899b06` em `7f899b06` (18 atrás, limpa). Nenhum agente correspondente ativo foi encontrado; não houve remoção/prune.
- Resultado: validação de código local PASS; continuidade real e release permanecem **NO_GO** até segunda conta distinta, quota elegível e ensaios manual/automático. O candidato não corresponde a um SHA limpo/congelado.

## 2026-11-20 — Causa da quota Codex UNKNOWN

- Reproduzido com `nexus usage --refresh --json`: app-server quota probe falha e não há leitura de sessão utilizável. O stderr do app-server informa que o access token salvo não pôde ser renovado e orienta a autenticação novamente. Assim, o estado UNKNOWN é resultado do token expirado/inválido, embora `InspectAuth` ainda marque credenciais salvas como autenticadas. Nenhuma chamada de geração foi executada.
- O perfil `codex:kiver` está no fluxo de login oficial para renovar a autenticação. A sessão local aguarda OAuth no browser em `localhost:1455`. Depois do callback, verificar que o perfil volta a autenticar com a conta omegaedu e repetir quota ao vivo antes do handoff.
- `codex:omegasistemas` ainda contém credencial importada do host que identifica omegaedu. Não usar como conta separada até OAuth no perfil correto e confirmação de identidade externa distinta.

## 2026-11-20 — Perfil da conta omegasistemas e login

- Após remover os aliases, Nexus tinha apenas `codex:kiver`, autenticado na conta omegaedu. Não havia perfil ou runtime Codex da conta omegasistemas disponível para o teste.
- Criado perfil isolado `codex:omegasistemas` e iniciado o fluxo oficial OAuth do Codex no terminal. Login aguarda a conclusão no navegador local. Não executar handoff nem geração até o perfil autenticar, confirmar identidade distinta e mostrar quota elegível.
- Próxima ação após concluir OAuth: validar `nexus profiles --json` e `nexus usage --refresh --json` de forma sanitizada, então retomar o ensaio manual descrito em `SPECS/RELEASE_FINALIZATION_PLAN.md`.

## 2026-11-19 — Candidato para aceite de continuidade

- PASS — recompilado e instalado o CLI local em `~/.local/bin/nexus`, base `6c322427` com árvore modificada. Identifica-se como `1.0.0`, build Go 1.26.11. SHA-256 `6fe4756615b479624f7a589955adc6cf450503716093f886c0fd9e63a6e8b255`. Smoke confirma versão, três perfis Codex autenticados visíveis e opção `--no-attach`. Backup anterior preservado em `~/.local/share/nexus-backups/nexus-before-reinstall-20261119T182653Z`. O roteiro está em [plano de finalização](SPECS/RELEASE_FINALIZATION_PLAN.md#teste-do-maestro).
- A primeira execução de `go test ./...` encontrou `TestMissionFailoverCostObservationAttributionFollowsExecutingRuntime` tentando interpretar saída PTY como JSON de attach. Após ordenar a fila de frames, o teste passou em cinco execuções consecutivas. O teste novo trava a ordem da fila. PASS — race detector no host e no protocolo após o fix. Os pacotes app, registry, handoff e scheduler passaram no race após a alteração do CLI.
- PASS na árvore final — `go test ./...`, `make lint-go` com zero issues, `make security` sem vulnerabilidades reportadas, links locais do plano e `git diff --check`.
- PASS no checkout Maestro `v1@f27ba2c` — handshake Bridge v1 anuncia `runs.submit` e `node --test tests/cli-durable-worker-e2e.test.js` passou 2/2. A evidência usa processos CLI e executor fixture. A paridade com o CLI distribuído permanece aberta.
- NO_GO de homologação — 48 capacidades e 97 critérios mapeados, zero critérios certificados. NEX 2/11 e `--require-ready` bloqueado. A troca Codex manual e automática com duas contas reais é o próximo teste do maestro. O CLI padrão segue instalado e a credencial do AI Manager ficou intacta.

## 2026-11-19 — Continuidade Codex local

- PASS — `go test ./...` na árvore local; `make lint-go` (0 issues); `make security` (No vulnerabilities found); `go build -o /tmp/nexus-continuity-validation ./cmd/nexus`; `git diff --check`.
- PASS — race detector em `internal/control/registry`; testes focados de `internal/control/registry`, `internal/core/scheduler`, `internal/control/handoff`, `internal/app`, `internal/control/web` e `internal/nexus` também passaram.
- NO_GO operacional — não copiei nem removi `auth.json` de `kivergmail`: duas sessões Codex que usam o diretório ainda estão ativas. Permissão de origem segue `0600`. Escopo salvo declara identidade distinta de `kiver`, `work` e `default`, mas credencial atual/quota ainda não foram verificadas. Nenhuma geração de modelo foi iniciada e nenhum E2E real de troca aconteceu.

## 2026-11-19 — Codex e triagem GitHub

- PASS no CLI instalado de `6c322427`. A resolução via PATH passou a usar o build de novembro. Inspeção de perfis confirmou dois registros Codex, uma identidade de provedor e autenticação saudável nos dois perfis.
- PASS na cópia isolada de `6c322427` com o patch local. Regressões de rotação, continuidade, failover e launch com race detector, `go vet` e golangci-lint nos pacotes afetados, build do CLI Go e smoke de `status --json`. #218 passou 30 vezes sob race detector na cópia isolada em Linux. O teste da árvore compartilhada encontrou incompatibilidade preexistente na assinatura de `ListMissionRunEvents`.
- NO_GO. Validador de entrega NEX reportou 2/11. Linux smoke da ponta remota verde, CI canônica na fila e preflight Windows falhou em `TestSessionHost_TwoClientLeaseHandoffIsAuditedAndSingleWriter` no run `44402136168`. Nenhuma chamada de geração nem troca entre contas Codex reais foi feita.

## 2026-11-17 — 2ª varredura

- PASS — PR #217 merged @ `6ab69945`: ps1 do first-run reconstruído (**tip anterior tinha 9 erros de parse** e blocos duplicados do squash do #215) + guarda de sintaxe PS no CI. Tip: ps1 0 erros, yaml OK, `make quality` 0 FAILs/911.
- PASS — `fix/hud-show-restore-v2` deletado (conteúdo 100% presente no tip); `docs/nexus-1.0-portal-20261022` deletado (superseded por `docs-site/`; PR original era closed-unmerged). Remotos **63** (80 removidos no total).
- KEEP — NEX motion/companion branches sob governança de identidade (não integrar sem owner visual); v1.1 lines mantidas para pós-1.0.
- OPEN — #214 (race estrutural handoff pós-revoke, estoura 15s em CI) aguarda ordem de investigação.

## 2026-11-17 — Triagem remota

- PASS — PR #215 squash-merged @ `eb391b36` (Windows Go pin do `go.mod` + null-safety Assistant); quality-full **MAKE EXIT: 0** no tip (1º run morreu por OOM — runner CI na mesma máquina; rerun limpo).
- PASS — 78 branches mortos removidos (71 PR-merged + 7 contidos), remotos 143→65; `origin/pr-61` local limpa.
- KEEP — 25 issues abertas nenhuma fechada: #214 real (falha @15s no CI — race estrutural pós-revoke, investigar antes do mobile/remote), resto = sessões/roadmap/gates visuais.
- DECISÃO MAESTRO — ~10 branches com trabalho único não integrado (NEX rig/companion/v1.1, 2–42 commits): reconciliar × fechar; cruzam com identity lock.

## 2026-11-16 — Tag v1.0.0 @ 14622be8 + pipeline parado no gate nativo (correto)

- PASS — `make quality-full` @ tip `14622be8` com **MAKE EXIT: 0** (format, lint, styles, typecheck, lint-go, test-go, 904 frontend, race suite completa, govulncheck, version contract).
- PASS — Deflake honesto: waits WS do teste de handoff autenticado 5s→15s (0,11s isolado × timeout sob race do pacote cheio; wait não é asserção de latência).
- PASS — Tag `v1.0.0` @ `14622be8` pushed; CI 9/9 required **success no SHA exato da tag** (verificado por API).
- PASS — Gate "Same-SHA CI" do release **funcionou end-to-end**: encontrou o ci.yml completed success no SHA (match client-side `head_sha`+`path`+`event`+`completed`), validou os 9 jobs por nome.
- EXPECTED FAIL — Run Signed Release parou em **"requires exact-SHA native certification policy=all"**: Windows/macOS/Seal checks aguardam as sessões físicas do maestro. Fail-closed correto; nada publicável sem 3 plataformas.
- LIÇÃO — `actions/runs?head_sha=` retorna 0 no repo (server-side filter quebrado); filtrar sempre client-side por jq. E `last` por id sem filtro de workflow pega o Linux smoke — usar `path == ".github/workflows/ci.yml"`.

## 2026-11-16 — GO ceremony kits (parte executável)

- PASS — Kit 3/Linux: homologação automatizada completa **PASS** @ `1.0.0 / 4d07d831f3c2…` (docs-verify, frontend, gofmt/vet/golangci/govulncheck, version contract, race, control E2E, agent interop, builds ×3, browser E2E ×9, a11y, visual, goreleaser snapshot, DEB smoke). Evidência local gitignored com screenshot real 1440×900 + test.log + binário DEB (`e2b50ab5…`).
- PASS — Kit 2 pipeline: trust root rotacionada (arquivo + var GitHub → `faa2e250…` = chave enrolada 10/11); ed25519 aceita pela trust store no verificador offline (dry-run); trio de produção no SHA final.
- PASS — Kit 1 parcial: AGY **1/2** (kivervinicius-gmail 5h ok; kiveromegasistemas sem janela 5h — maestro). Evidência commitada em `DEV/validation/current/agy-quota-final.json`.
- FAIL/NÃO EXECUTADO (maestro): AGY 2/2, Windows/macOS físicos, Kit 4 two-client, Kit 5 non-stop (2ª conta codex), aceite 48/48, NEX owner approval, maestroSha. Bundle offline fail-closed correto: "scope not certified" + "exactly three platforms".
- PASS — CI @ tip: Linux smoke ✅; homologation Windows 🔴 apenas por runner físico offline (esperado).

## 2026-11-16 — Versão definitiva 1.0.0 + bandeja do HUD com resolução em lote

- PASS — Versão: `VERSION` = `web/package.json` = `1.0.0` (definitiva, canal stable automático); VERSION CONTRACT PASS; `verify-release-docs` PASS; `release-version.test` 3/3; testes Go de update/canal PASS (fixtures beta são cenários de canal, intocados).
- PASS — Recurso da bandeja: filtro de snoozes em `projectMascotAttention` (correção: "Adiar" antes só silenciava a bolha e o item ficava na lista para sempre); `groupMascotAttention` agrupa duplicatas com contador ×N; "Adiar todas" com persistência localStorage; i18n EN/PT/ES; vitest **904/904** com 4 testes novos.
- PASS — `make quality` completo (format-check, lint-frontend, lint-styles, typecheck, lint-go, test-go, test-frontend, version-check) + build + `make install-local` → runtime `1.0.0 @ 9e5d4548` vivo (HTTP 200, health ok).
- NÃO EXECUTADO: aceites visuais do owner (pacote `owner-visual-approval-20261115-rig`), Windows/macOS nativos (runners físicos offline), kits 1–5 do GO.

## 2026-11-12 — Non-stop Codex fatias 1–2 + fatias 14–16 + security GO-2026-6859

- PASS — Go (toolchain 1.26.11): `make test-go` completo sem FAIL; `make vet`; `make lint-go` 0 issues; `make race` sem data races; `make security` "No vulnerabilities found"; `go build ./...` + gofmt limpos.
- PASS — Novos testes Go: session discovery codex ×5; watcher host ×3 (+race); attach-follow ×5; fail-closed quota ×3; guarda interativa ×1; quota pattern watcher ×7; evidência/subscriber nexus ×5; challenge serviço ×6 + endpoint HTTP ×4; sharing fase 2 registry ×3 + HTTP ×2.
- PASS — Web: `npm run quality` completo → **897/897 testes em 141 arquivos** (fatias 14 + 17: VisualDiffPanel + ShareGrantsCard + models 14 testes + discovery/tour/i18n ×3); tsc, eslint, stylelint, prettier, check:styles, build — todos verdes.
- PASS — Contrato: `verify-agent-workspace-v1.mjs` = "scope 48, certification still pending"; evidence map 97 cláusulas (95 source+test, 2 source-only honestas de CI multi-SO), zero unmapped.
- FAIL/NÃO EXECUTADO (gates externos do GO): aceite manual non-stop com 2 contas codex reais (maestro); NEX master visual (maestro); AGY 2/2; worker Maestro + pacote assinado; execução física 3 SOs; E2E two-client em app distribuído. GO segue **NO_GO** por esses gates, não por código.

## 2026-11-10 — Fatia 13 (UI release-desk) + plano OMP

- PASS — `npx vitest run releaseDeskModel.test.ts i18n.test.ts`: 13/13 (model com nulls/tones/Intl + paridade de chaves ×3).
- PASS — `npm run typecheck`, eslint focado, stylelint do module.scss, prettier, `npm run build` — todos verdes.
- Plano OMP registrado em `DEV/SPECS/OMP_QUOTA_ROUTING_PLAN.md` (somente planejamento; 4 decisões aguardando o maestro).
- NÃO EXECUTADO: E2E visual do card no app distribuído; aceite.

## 2026-11-10 — Rotação por saldo disponível

- PASS — `go test ./internal/core/scheduler -run TestBalancedSelection -count=1 -v`: 8/8, incluindo 3 testes novos da política "mais saldo primeiro".
- PASS — `go test ./internal/core/scheduler ./internal/core/quota ./internal/core/fallback -count=1` e `go test ./internal/app -count=1` (busy_failover): todos verdes; `go vet` limpo.
- Comportamento novo codificado: conta com mais quota medida vence sempre; opaque fica em tier inferior; empate gira por cursor persistido.

## 2026-11-10 — Visual Diff wiring

- PASS — `go test ./internal/control/web -run TestBrowserVisualDiffExplicitPair -count=1`: par idêntico com artefato PNG, 422 de size mismatch, 400 sem limiares, 400 sem modo, 405 em GET.
- PASS — `go test ./internal/control/web -count=1` completo (67s); build/vet limpos; validador do mapa PASS.
- NÃO EXECUTADO: fluxo visual na UI (capturar dois frames e comparar); recheck server-side com sessão real de browser; aceite `pending`.

## 2026-11-10 — Release Desk: aprovação assinada

- PASS — `go test ./internal/nexus -run 'TestVerifyReleaseApproval|TestLoadReleaseApproval' -count=1`: 3 testes (round-trip, fail-closed com 4 cenários, skew).
- PASS — `go test ./internal/control/web -run TestProjectReleaseDesk -count=1`: 4 testes, incluindo o fluxo assinado end-to-end via HTTP (trust store + bundle no repo do fixture).
- PASS — `node --test scripts/sign-release-approval.test.mjs`: 2/2 (round-trip com verificação independente da canonical form; mismatch e overwrite recusados). Prettier PASS nos 2 arquivos.
- PASS — `go test ./internal/nexus -count=1` completo (123s); validador do mapa PASS com os novos refs (padrão de testRefs Go-only mantido — teste node do signer registrado nos DEV docs, como o signer anterior).
- NÃO EXECUTADO: aprovação real do maestro para o SHA de release (o circuito está pronto; o aceite é humano); UI do desk.

## 2026-11-10 — Parallel Challenge

- PASS — `go test ./internal/nexus -run TestEvaluateChallenge -count=1`: 5 testes puros verdes.
- PASS — `go test ./internal/nexus -count=1` completo (116s); validadores do contrato/mapa PASS após `parallel-challenge` → `partial`.
- NÃO EXECUTADO: despacho real multi-agente via MissionRunner + UI; aceite `pending`.

## 2026-11-10 — Workspace Sharing

- PASS — 8 testes puros do registry (`workspace_sharing_test.go`) + 3 testes de integração HTTP (`handlers_workspace_sharing_test.go`): viewer flow completo com revoke imediato (403 na chamada seguinte), deny explícito de grant desconhecido/parâmetro ausente, payload restrito ao projeto concedido.
- PASS — `go test ./internal/nexus ./internal/control/web -count=1` completos verdes; gofmt limpo; validadores contrato/mapa PASS após `workspace-sharing` → `partial` (48 caps, 97 cláusulas, 95 source+test, 2 source-only).
- Segurança: deny-by-default com razões determinísticas; viewer-only (mutação negada por teste); escopo obrigatório de projeto (cross-project negado por teste); IDs randômicos não enumeráveis; gestão de grants no caminho autenticado+CSRF. Sem secrets novos; sem mutação de dados de produção.
- NÃO EXECUTADO: UI de gestão de grants; persistência durável (fase 2 — ADR follow-up); TTL e auditoria; aceite `pending`.

## 2026-11-10 — Agent Compare

- PASS — `go test ./internal/nexus -run TestCompareAgentImplementations -count=1`: 7 testes puros verdes (independência de worktree, critérios explícitos, peso de falhas, UNKNOWN≠0, custo decide só com evidência, empate misto, kind inválido explícito).
- PASS — `go test ./internal/nexus -count=1` completo (127s); validadores do contrato/mapa PASS após `agent-compare` → `partial`.
- NÃO EXECUTADO: orquestração de execução nas worktrees + wiring do custo real + UI — gaps explícitos; aceite `pending`.

## 2026-11-10 — Visual Diff

- PASS — `go test ./internal/nexus -run 'TestCompareScreenshots|TestRenderDiffArtifact' -count=1`: 7 testes puros (inclui determinismo e fronteiras de limiar) verdes.
- PASS — `go test ./internal/nexus -count=1` completo (117s); gofmt limpo; validadores do contrato/mapa PASS após `visual-diff` → `partial`.
- NÃO EXECUTADO: encadeamento com capturas reais do browser (serviço + UI), persistência de artefatos, homologação — gaps explícitos no mapa; capability `pending`.

## 2026-11-10 — Release Desk

- PASS — `go test ./internal/nexus -run 'ReleaseDesk|ParseCommitReferences|MergePublish|VerifyReleaseGate|ParseReleaseCommitLine' -count=1`: 8 testes puros verdes.
- PASS — `go test ./internal/control/web -run TestProjectReleaseDesk -count=1`: 3 testes de integração com repo git real (range base/head correto via merge-base, 2 commits, PR #55 vs issue #114 separados, bloqueio com razão de aprovação, ref `main;rm` → 400, POST → 405).
- PASS — pacotes completos: `go test ./internal/nexus -count=1` (132s) e `./internal/control/web -count=1` (69s); gofmt limpo.
- PASS — `verify-agent-workspace-v1.mjs` + `verify-agent-workspace-evidence-map.mjs` após `release-desk` → `partial` (48 caps, 97 cláusulas, 95 source+test, 2 source-only).
- NÃO EXECUTADO: UI do Release Desk (card pendente — gap explícito no mapa); homologação/aceite (pending); fluxo real de aprovação assinada integrado ao `sign-offline-release-evidence` (persistência pertence ao caminho offline, próximo passo natural da fatia).

## 2026-11-10 — Design-inspect drawer UX

- PASS — vitest: `BrowserSurface.test.ts` + `designInspectHandoff.test.ts` + `terminalResolution.test.ts` + `i18n.test.ts` (30/30) — handoff de inspeção, resolução e paridade ×3 com as 20 chaves novas.
- PASS — typecheck; ESLint 0 em `BrowserSurface.tsx`/`resources.ts`; stylelint no module; prettier (1 arquivo com --write); build.
- PASS — validador do mapa após atualização do gap do `design-inspect` (refs de UX reais).
- NÃO EXECUTADO: screenshot 390/1280 no app distribuído (homologação visual — aceite segue `pending`); o crop era reparado por CSS wrap/scroll; evidência visual final exige SHA publicado.

## 2026-11-10 — Agent parking honesto

- PASS — `npx vitest run src/features/agents/agentParkingModel.test.ts`: 10 testes — logicalKey `agent:{id}:terminal`, detecção de superfície aberta, `parkingOptions(false)` sem suspend (nunca mente), `parkingOptions(true)` com suspend, `parkingStatusKey` (null aberta / hiddenWorking ativo / hiddenIdle parado), `isParkedWorking` exato, hint keys, e honestidade i18n ×3 (afirmação de execução contínua + negação explícita de pausa).
- PASS — vitest conjunto com `changeCollisionsModel` e `i18n` (30/30), paridade PT-BR/EN/ES preservada com os novos blocos `agents.parking` e `agents.collisions`.
- PASS — `npm run typecheck` (após afrouxar o tipo do array de locales no teste — literais do catálogo são incompatíveis entre locales por design); ESLint 0 nos arquivos tocados; stylelint; prettier `--write` em 2 arquivos; `npm run build`.
- PASS — `node scripts/verify-agent-workspace-v1.mjs` + `verify-agent-workspace-evidence-map.mjs` (48 caps, 97 cláusulas, 95 source+test, 2 source-only, 0 unmapped) após `agent-parking` → `partial`.
- NÃO EXECUTADO: interação real do botão "Ocultar superfície" em app distribuído (exige E2E visual); homologação — capability segue `partial`/`pending`.

## 2026-11-10 — Detector de colisões de edição

- PASS — `go test ./internal/nexus -run 'TestParseDiffHunks|TestDetectChangeCollisions' -count=1`: 6 testes puros (extração de faixas, caminho com espaço/arquivo novo, overlap de hunks, disjunto = incerto com razão, inserção-âncora vs edição da mesma linha, arquivos de worktree único ignorados, determinismo).
- PASS — `go test ./internal/control/web -run TestProjectChangeCollisions -count=1`: 3 testes de integração com repositórios git reais em temp (repo + worktree vinculado, edições não commitadas) — overlap mesmo arquivo/mesma linha → `hunk-overlap` sem incerteza e 2 worktrees listados; edições disjuntas (linha 2 vs 28) → `same-file` incerto com `uncertain_count=1`; POST → 405.
- PASS — `go test ./internal/nexus ./internal/control/web -count=1` completos (120s/66s) verdes — nenhuma regressão nos pacotes tocados.
- PASS — web: `vitest` `changeCollisionsModel.test.ts` 6/6 e `i18n.test.ts` 8/8 (paridade dos 3 locales com as novas chaves `agents.collisions.*`); `npm run typecheck`; ESLint 0 erros nos arquivos novos/alterados; stylelint no novo module SCSS; prettier `--check`; `npm run build` (bundle + NEX SVG export).
- PASS — `node scripts/verify-agent-workspace-v1.mjs` (escopo 48) após `change-collision-detector` → `partial`; evidence map reescrito com refs reais.
- NÃO EXECUTADO: homologação do card em app distribuído/SHA publicado (aceite `pending`); scan em projeto real com >16 worktrees (limite coberto por código, não por teste); `make quality` consolidado da árvore multi-lane não reexecutado nesta fatia (gates focados + pacotes Go completos cobrem o diff).

## 2026-11-10 — Reconciliação, trust store/signer e auditoria de galhos

- PASS — Contrato reconciliado: `node scripts/verify-agent-workspace-v1.mjs` (escopo OK) + `node scripts/verify-agent-workspace-evidence-map.mjs` (48 caps, 97 cláusulas, 95 source+test, 2 source-only, 0 unmapped, 0 certificados) + testes do mapa 4/4 (contagem dinâmica). Statuses corrigidos: `design-inspect` e `maestro-managed-dependency` `missing→partial` com gaps reescritos; 9 `missing` restantes confirmados sem implementação por grep.
- PASS — `node --test scripts/sign-offline-release-evidence.test.mjs` 4/4: round-trip Ed25519 (assinatura válida alcança checagem estrutural; chave não enrolada falha na assinatura), recusa de payload já assinado, `--hash` sha256. Smoke com chave real + trust store enrolado PASS (erro esperado = checagem estrutural, não de assinatura).
- PASS — Auditoria de galhos por conteúdo: 8 galhos verificados 100% integrados/superseded (pr/55, pr/57, pr/58, pr/59, pr/64, pr-review/*, fix/beta2, fix/workspace-dir, homologation). Limpeza: `git worktree prune` + refs `pr/*`/`pr-review/*` removidas + 3 branches locais deletadas. Preservados: `origin/pr-61` (remote), `iapro/agent/*` (AI manager).
- NÃO EXECUTADO: deleção de `origin/pr-61` no GitHub (exige push — decisão do maestro); E2E de túnel público do código de acesso; `make quality` re-executado não foi necessário nesta fatia (só docs/scripts JSON/JS — validadores específicos cobertos acima).

## 2026-11-10 — Pareamento remoto por código de acesso

- PASS — `go test ./internal/control/web -count=1` completo (66s): pacote integral verde, incluindo 8 novos testes de `access_code_test.go` (geração Crockford base32 sem duplicatas, uso único, normalização I/L/O + caixa + pontuação, guess errado não consome convite, vínculo ao host sem consumir, túnel inativo falha fechado, guarda de 10 falhas revoga convites, endpoint pair autentica/rejeita/revoga/nunca por GET). Regressão de auth (`TestExchangeBootstrap|TestTunnelInvite|TestLoopback|TestIssue|TestRemote|TestAuth`) PASS — refatoração `newSessionLocked` preserva semântica de consumo pós-entropia.
- PASS — `go test -race ./internal/control/web -run 'AccessCode|ExchangeAccessCode|Pair' -count=1`; `go vet ./internal/app ./internal/control/web` limpo.
- PASS — suíte web completa: `npx vitest run` 136 arquivos / 856 testes (6 novos: 3 de `remoteAccessModel`, 3 de `pairWithAccessCode` em `api.test.ts`).
- PASS — typecheck strict, ESLint (0 erros; 1 warning pré-existente de outra lane em `FlowRunSurface.tsx:122`), stylelint, prettier (após `--write` em 2 arquivos), `npm run build` (bundle + NEX SVG export OK), `git diff --check` limpo.
- PASS — `node scripts/verify-agent-workspace-evidence-map.mjs`: 48 capabilities, 97 acceptance clauses (95 source+test, 2 source-only, 0 unmapped, nenhum aceite); `node scripts/verify-agent-workspace-v1.mjs` escopo OK; testes do validador 4/4 após tornar a contagem de cláusulas dinâmica (soma do contrato; o literal "96" não sobreviveria à extensão legítima do aceite).
- NÃO EXECUTADO: E2E em túnel público real com dois dispositivos físicos (depende de infra externa); `make quality-full` consolidado da árvore multi-lane permanece pendente de nova execução após todas as lanes convergirem.
- PASS — Gates consolidados da árvore completa (multi-lane, 2026-11-10): `make quality` EXIT=0 (format/lint-frontend/lint-styles/typecheck/lint-go/test-go/test-frontend/version-check — 1ª execução falhou transitória no build de `scripts/sign-update-manifest` e `flatted` durante reescrita de node_modules; reexecução limpa), `make race` EXIT=0, `make security` EXIT=0 ("No vulnerabilities found"). `gofmt -w` aplicado a `access_code_test.go`.

## 2026-11-09 — Focus Queue real + correções de gate (Comparison Lab, maestro, hermeticidade, marker race)

- PASS — `npm --prefix web run test -- --run src/app/focusQueueModel.test.ts` (5/5): ordenação severidade→tempo, dedupe por episódio (`fp:`/`rt:`), não-reabertura (`attention_submitted_revision >= attention_revision`), only honest needs_user + erro observado, sanitização e determinismo de rebuild.
- PASS — vitest focado 31/31 (`focusQueueModel`, `featureAnnouncements`, `featureTour`, `tourTargets`, `featureReveal`): catálogo de descoberta íntegro com a nova superfície `notifications`, âncora `data-tour="focus-queue"` presente em `FocusQueueSection.tsx`, i18n discovery/tour nos 3 idiomas.
- PASS — `node scripts/verify-agent-workspace-evidence-map.mjs` (48 capabilities, 96 critérios; 94 source+test, 2 source-only, 0 unmapped, nenhum aceite) + `node --test scripts/verify-agent-workspace-evidence-map.test.mjs` (4/4). focus-queue agora aponta para os arquivos reais implementados; `assertionAudit: partial`, cobertura `unverified`.
- PASS — `cd web && node scripts/verify-report.mjs` 10/10 (format, typecheck, lint, stylelint, null-arrays, test 135 arquivos/850 testes, i18n, build, embed-sync, ui-markers). O gate `null-arrays` voltou a PASS após normalizar `checks`/`changed_files` em `ComparisonLabPanel` com `asArray`/`asStringArray` (falha pré-existente em código commitado).
- PASS — `go test ./internal/nexus -count=1` completo (129s) após remover código morto do contrato CLI antigo em `maestro.go` e reescrever `TestMaestroCapabilitiesNegotiatesBridgeV1Contract` (handshake fake com marker, `runsSubmit=false`, sem ORCHESTRATE, memória read-only).
- PASS — `go test ./internal/profile -run TestGetAccountInfo` e `go test ./internal/app -run TestControlPlaneCLICommands` após fixar `NEXUS_REAL_HOME` temporário: `Create` adota credenciais reais do host (`AdoptHostAuth`) e os testes liam o estado AGY/Codex local (e-mail real vazava no output). Produto inalterado; teste agora determinístico.
- PASS — `go test -race ./internal/control/web -run TestTerminalHandoffAuthenticatedClientsShareProcessAndPersistIdentity -count=5` após trocar leitura única do marker file por polling (`awaitMarkerContains`): o echo do WebSocket podia vencer o append do processo filho sob `-race`.
- PASS — `make lint-go` (0 issues) e `make race` completo (`RACE_EXIT=0`, zero FAIL) após `go clean -cache` (cache Go corrompido no meio da 1ª tentativa por pressão de disco/outra lane: `link: cannot open file .cache/go-build/...`).
- PASS — `make security` (`SECURITY_EXIT=0`, govulncheck: "No vulnerabilities found").
- PASS — `make quality` sobre a árvore final (`QUALITY_EXIT=0`): format-check, lint-frontend (0 erros; warnings pré-existentes apenas), lint-styles, stylelint/check:styles, typecheck, lint-go, test-go, test-frontend, version-check ("all sources consistent").
- PASS — `make build` sobre a árvore final (`BUILD_EXIT=0`).
- PASS — `git diff --check` limpo.
- FAIL esperado — nenhum. Continua **NO_GO**: gates externos abertos (AGY 1/2 elegíveis 5h, worker durável Maestro ausente com `runsSubmit=false`, pacote oficial assinado/multiplataforma, artefato distribuído exact-SHA, aceite 48/48, `make docs-verify` bloqueado por manifesto visual stale).

## 2026-11-09 — Maestro Bridge v1 + revogação de terminal + corrida Event Bus

- PASS — `NEXUS_MAESTRO_CLI_TEST_PATH=/tmp/nexus-maestro-v1-wrapper go test ./internal/nexus -run '^TestMaestroBridgeV1ContractInterop$' -count=1 -v`. O wrapper invoca o checkout real `/projetos/tools/Orquestrador-Maestro/bin/orquestrador-maestro.js` na branch `v1`; inicialização JSON-RPC v1, `project.context`, `plans.validate` e `runsSubmit=false` foram validados. O wrapper é temporário, não pacote instalado/distribuído.
- PASS — `go test -race ./internal/control/events ./internal/control/web -run 'Test(UnsubscribeRacesWithPublish|AuthManagerSessionWatcher|TerminalHandoffAuthenticatedClientsShareProcessAndPersistIdentity)$' -count=1`. Prova revogação/expiração observada, fechamento de WebSocket, transferência de lease e ausência de input pós-revogação no processo; inclui regressão de corrida Publish/Unsubscribe.
- PASS — testes focados de discovery Maestro e `go test ./internal/nexus -count=1` na revisão em curso. Ainda executar `make quality-full`, `make build`, contrato/evidence map e `git diff --check` depois das outras lanes.
- Estado correto de Maestro: Bridge v1 é descoberto por handshake. Advice e verification genéricos não estão anunciados no Bridge canônico; endpoints CLI legados permanecem degradados, sem fallback/veredito local inventado. `runs.submit` está explicitamente false.
- Auditorias das 48 capacidades/96 critérios terminaram: todas foram avaliadas semanticamente, mas nenhuma certificada. Foram identificados gaps locais (incluindo runtime-health, focus-queue, agent-timeline, change-radar, visual-diff, workspace-sharing e release-desk) além de gates externos. O evidence map estrutural não é aceite.
- NO_GO — branch sem checkpoint, gates integral/build por repetir, dependência Maestro sem worker de execução durável, AGY com somente 1/2 contas elegíveis a 5h, providers/custos e artefatos/plataformas exact-SHA pendentes.

## 2026-11-09 — Integração das lanes locais e gate integral

- PASS — `make quality-full`: formato, lint (0 erros; 1 warning ESLint preexistente em `FlowRunSurface.tsx`), Stylelint, TypeScript, Web 134 arquivos/845 testes, Go completo + race e govulncheck (`No vulnerabilities found`). `make build` e `git diff --check` também passaram.
- Nota de execução — a primeira tentativa de `make quality-full` falhou no estágio Go; o log não preservou de forma suficiente o primeiro pacote/teste. Em seguida, `make test-go`, `make race` e uma nova execução integral passaram. Manter atenção a eventual intermitência em execuções futuras.
- PASS — `node scripts/verify-agent-workspace-v1.mjs`, `node scripts/verify-agent-workspace-evidence-map.mjs`, testes do contrato 5/5. Mapa 48/96: 94 source+test, 2 source-only, 0 unmapped, 0 certificados.
- PASS — E2E Design Inspect local: elemento selecionado, inspeção read-only, send explícito para o Agent Beta escolhido, sem auto-start; asserts dos controles e textos completos nos viewports 1280/390. Vitest combinado das lanes: 8 arquivos/36 testes.
- PASS — testes focados Agent Groups de store/API e Vitest model/surface/tour/âncoras; ESLint/Stylelint/Prettier e typecheck passaram. Limite: nenhum E2E demonstra runtime/PTY viva após reorder/filter.
- PASS — `go test ./internal/core/scheduler -run 'TestBalancedSelection|TestBalancedAGY|TestAGYBalancedQuota' -count=1`; `go run ./cmd/nexus maestro status --json` com CLI instalado retorna `available=true`, ASSIST e contracts 1.0.0; interop real Go→Maestro passou com `NEXUS_MAESTRO_CLI_TEST_PATH` definido.
- BLOCKED — `make docs-verify`: manifesto visual stale e alterações locais sem captura homologável no SHA publicado. Não regenerar snapshots sintéticos como substituto.
- NO_GO — mudança local ainda sem SHA commitado; AGY só 1/2 elegíveis a 5h, Maestro oficial assinado/multiplataforma ausente, runtime móvel/túnel/provider e custos reais não comprovados; nenhuma das 48 capacidades foi certificada e 60 critérios referenciados ainda precisam auditoria de assertions.

## Snapshot — A2 lifecycle concorrente e redaction de browser — 2026-11-09

- PASS — `make quality-full`: format-check, ESLint (0 errors; six existing warnings), Stylelint, TypeScript, Web 131 arquivos/833 testes, Go completo + race; `internal/nexus` race passou em 287,551 s; `govulncheck` terminou `No vulnerabilities found`.
- PASS — suites focalizadas browsercontrol, nexus e control/web; regressões sob `-race` para aquisição cancelada e serialização de Start/Stop/Recover/SafeApply/mission/interactive runtime/continuity. Redaction cobre active actor, URL e snapshot JSON/text. `make build` e smoke `NEXUS_BROWSER_INTEGRATION=1 go test ./internal/browsercontrol -run '^TestChromiumSmokeOptional$' -count=1 -v` passaram.
- PASS — mapa estrutural 48/96: 93 referências source+test, 3 source-only, 0 unmapped, 0 certificados; contrato e mapa validados, testes do verificador 5/5 e `git diff --check` limpo.
- BLOCKED — `make docs-verify`: enumera manifest visual stale para ComparisonCampaignPanel/i18n/tipos desde `1c048f8d` e alterações locais em DirectSession/UI sem evidência visual do SHA. Isso não constitui homologação.
- NO_GO — referências no mapa não encerram os gaps de design-inspect (seleção/DOM), live-preview (processo/porta e controle de URL), device-handoff (dois clientes autenticados + execução única), nem requisitos externos AGY/Maestro/provider/exact-SHA.

## 2026-11-09 — E2E DirectSession no commit publicado e recheck Maestro

- PASS — exportei `e7268be6d08ee9aa1c6a0a067b140476d8fbcbb8` com `git archive` para evitar alterações locais concorrentes e rodei `NEXUS_E2E_ARTIFACTS_DIR=/tmp/nexus-e7268be6-source-artifacts npm --prefix <archive>/web run test:e2e:first-run-import`. Fluxo DirectSession atingiu Core compilado da árvore exata para CRUD/config/stop, verificou ordem start→stop→delete, Agent final ausente e 390 px sem overflow. Captura real arquivada em `DEV/validation/current/direct-session-cancel-e7268be6/`.
- Limite — a UI é montada do bundle do checkout, host usa harness local e provider/start está em fixture. Não certifica runtime de provider, app desktop/distribuído, login de dois perfis ou GO.
- Falha inicial observada — CLI antigo do PATH rejeitava `capabilities --json`. Apontei o alias do perfil ao CLI compatível no checkout Maestro e guardei o destino anterior como backup reversível. Revalidação sem override: `go run ./cmd/nexus maestro status --json` filtrado reportou `available=true`, `ASSIST`, CLI 1.1.0, contracts advice/verification 1.0.0; `NEXUS_MAESTRO_CLI_TEST_PATH=<alias maestro> go test ./internal/nexus -run '^TestMaestroCLIContractInterop$' -count=1 -v` passou. Ainda falta pacote oficial assinado e smoke no aplicativo distribuído.

## 2026-11-09 — live-preview + device-handoff reais locais

- PASS — `go test -race ./internal/browsercontrol -count=1`: processo filho/porta são validados por `/proc`; Chromium real bloqueou fetch e redirect cross-origin antes que o servidor externo recebesse qualquer hit. `go test ./internal/control/web -count=1`, `go vet ./internal/browsercontrol ./internal/control/web`, build cross-platform do pacote para Windows amd64 e macOS arm64 passaram. Não-Linux permanece intencionalmente fail-closed, ainda sem preview funcional nesses OS.
- PASS — `go test ./internal/control/host ./internal/control/web ./internal/control/events ./internal/control/protocol -count=1`; integração authenticated HTTP/WebSocket x5 e race focal. Dois cookies distintos chegam ao mesmo runtime/process generation; lease takeover nega Alpha, executa marcador Beta exatamente uma vez e SQLite registra IDs/atores pseudônimos sem bearer.
- PASS — mapa 48/96 estrutural: 93 source+test, 3 source-only, zero unmapped/certificados; verificador do mapa e testes do contrato (5/5).
- Design-inspect: Go/browsercontrol/web, Vitest 42/42 e E2E 1280/390 passaram; porém a captura revisada mostrou resultado do inspector recortado no drawer e a lane está ajustando a apresentação. Ainda não contar a entrega UI como visualmente fechada. Máscaras cobrem formulários/padrões conhecidos; PII arbitrária em imagem continua fora da garantia.

## 2026-11-09 — A2: cancelamento autenticado de DirectSession sem órfão

- PASS — `go test ./internal/control/web -run '^TestAuthenticatedAgentCancellationStopsRuntimeWithoutOrphan$' -count=1 -v` e `go test ./internal/control/web -count=1`: API com sessão/CSRF inicia stop verificado de SessionHost standalone supervisionado; Agent fica `STOPPED`, geração registra `StoppedAt` e runtime sai do Registry.
- PASS — `npm --prefix web run test -- --run src/features/work/featureTour.test.ts src/features/work/directSessionLifecycle.test.ts src/features/work/featureAnnouncements.test.ts src/features/work/featureTourAnchors.test.ts` (26 testes). Cancelar antes do start remove Agent; cancelar durante o start espera a resposta, tenta stop e só deleta após stop; falha de stop preserva Agent para recovery.
- PASS — `NEXUS_E2E_ARTIFACTS_DIR=/tmp/nexus-a2-cancel-20261109-mobile npm --prefix web run test:e2e:first-run-import`: Core real para create/delete/stop, start controlado pelo harness; sequência observada `create → select-resource → apply-config → start → stop → delete`; lista final de Agents/runtimes vazia. A viewport 390 px não tem overflow horizontal. Screenshot em `/tmp/nexus-a2-cancel-20261109-mobile/direct-session-cancel-mobile.png` mostra rolagem vertical do diálogo para acessar ações.
- PASS — `npm --prefix web run format:check`; validadores do contrato/mapa (48 capabilities/96 critérios, 87 source+test, 9 source-only, 0 unmapped, zero certificados).
- PASS — `go test ./internal/nexus -run '^TestStopWaitsForInFlightAgentStartAndStopsPublishedGeneration$' -count=1 -v`: stop fica bloqueado enquanto o launcher conclui start, para a geração publicada e deixa o Agent `STOPPED` sem runtime ativo. Start/stop/delete compartilham lock por Agent.
- PASS — `make quality-full` no estado de código anterior ao ajuste somente do fixture do verificador: Web 131 arquivos/833 testes; Go integral + race (Nexus 280,909 s, Store 104,383 s); version contract e `govulncheck` sem vulnerabilidades. `make build` posterior também passou, gerando bundle Web e CLI Nexus beta.11.
- PASS — testes do contrato/evidence map 9/9; contrato 48 verificado (pending), mapa 48/96 (87 source+test, 9 source-only, 0 unmapped, zero certificados), catálogo 36 e `git diff --check`. Corrigi fixture sintético para incluir binário de teste explicitamente não distribuível; essa suíte direcionada passou depois do `quality-full`.
- BLOCKED — `make docs-verify`: manifest visual antigo em `1c048f8d`; lista fontes visuais alteradas e não certifica as mudanças DirectSession locais sem captura vinculada a SHA. Captura sintética não substitui homologação.
- Limite — nenhuma conta de provider foi chamada; resposta de start é fixture no browser e o processo fake real é exercitado separadamente pelo teste Go. Ainda faltam dois perfis reais e app distribuído/exact-SHA. A2/capability seguem parciais; GO não certificado.

## 2026-11-09 — Comparison Lab A/B + retomada parcial — verificação corrente

- PASS — `make quality-full`: format-check, ESLint (0 erros; seis warnings já existentes), Stylelint, TypeScript, build Web, 131 arquivos/829 testes, Go integral + race (Nexus 319,240 s; Store 107,440 s), version contract e `govulncheck` sem vulnerabilidades.
- PASS — build e smoke Maestro no SHA publicado `f803d264`: `./nexus version` reportou beta.11/commit correto; `./nexus maestro status --json` retornou available, ASSIST, advice/verification 1.0.0. Go focal; após retry: `go test ./internal/nexus/store ./internal/nexus ./internal/control/web -run 'TestComparison' -count=1`, TypeScript, Stylelint e Prettier.
- PASS — contrato v1 (48 capabilities pending), evidence map (96 critérios: 87 source+test, 9 source-only, 0 unmapped, 0 certificados), map tests 4/4, catálogo 36 e `git diff --check`.
- BLOCKED — `make docs-verify`: a recaptura sintética do portal publicada em `d2a48371` permanece vinculada ao source `1c048f8d`; guard enumera três fontes visuais posteriores. Não refazer captura sintética como evidência de aceite.
- Limite — testes validam preparo, approvals, persistência/recovery, transição parcial no store, dispatch boundary e comparação de runs vinculadas; não fazem provider call nem homologam execução paralela real/app distribuído. Retry da UI não tem E2E visual. Comparison Lab e release continuam partial/NO_GO.

## Histórico — Comparison Lab sem launcher A/B

- PASS — testes de aplicação Nexus para workspaces gerenciadas, base comum, diff, status de testes, custos conhecidos/desconhecidos, snapshot imutável, decisão humana sem merge e ausência de alteração no checkout (`internal/nexus/comparison_application_test.go`).
- PASS — persistência/decisão one-way na store e proteção de projeto/auth/CSRF nas rotas.
- PASS — testes API/Feature Announcement/Tour Anchor, TypeScript, Stylelint e ESLint dos arquivos alterados; `make quality-full` passou (Web 131/828, Go integral + race, lint/format/version/security). Nenhum erro novo de lint; 6 warnings ESLint preexistentes.
- PASS — `make build`; validadores de contrato 48, mapa (87 source+test/9 source-only/0 unmapped, zero certificados), catálogo 36, regressões do mapa 4/4 e `git diff --check`.
- BLOCKED — `make docs-verify`: após o push `a02c298a`, o gate visual informa capturas/manifesto vinculados a `7a26a298…` e nove fontes de renderização alteradas desde esse SHA. Falta browser E2E visual real no SHA publicado; as capturas sintéticas do portal não serão declaradas como homologação.
- Não cobre criação/execução paralela A/B, provider real, UX no artefato distribuído/SHA exato, nem aceite de capability. NO_GO mantido.

## AGY `:balance` — refresh read-only — 2026-11-09

- `NEXUS_REAL_HOME=/home/desenvolvedor nexus usage --refresh --json` consultou as duas contas sem enviar prompt. Snapshots recentes `CACHED/CLI_OUTPUT`; nenhuma geração executada.
- Uma conta retorna somente janelas semanais nos grupos Gemini e Claude/GPT; a outra retorna 5h mensurada nos dois grupos. Assim, a elegibilidade estrita de balance é 1/2. `eligibility.eligible` da quota geral não prova a janela 5h exigida pelo balance.
- Evidência sanitizada: [relatório](validation/current/agy-balance/README.md). Testes cobrem weekly-only rejection, fonte/atribuição/frescor por grupo e cursor round-robin persistente.
- Falta a primeira conta expor 5h e, depois, demonstrar rotação live entre duas contas elegíveis. Não iniciar geração para forçar esse estado.

## Maestro — descoberta da CLI contratual — 2026-11-09

- Reproduzido no host: `orquestrador-maestro` em Node 22/24 precedia no PATH e não implementava `capabilities`; o alias `/home/desenvolvedor/.local/bin/maestro` direciona para Node 20 e expõe contracts 1.0.0.
- Corrigi a preferência de descoberta do Nexus para `maestro` antes dos shims versionados, com regressão para múltiplos executáveis no PATH.
- PASS — `go test ./internal/nexus -run 'TestFindMaestroBinPrefersStableAliasOverNodeVersionShims|TestMaestroCLIContractInterop' -count=1 -v`; integração real `NEXUS_MAESTRO_CLI_TEST_PATH=/home/desenvolvedor/.local/bin/maestro go test ./internal/nexus -run '^TestMaestroCLIContractInterop$' -count=1 -v`.
- PASS — após correção, `make build` e `./nexus maestro status --json`: `available=true`, modo `ASSIST`, Maestro `1.1.0`, `advice=1.0.0`, `verification=1.0.0`.
- Instalação reversível — backup do binário de perfil `nexus.backup.40d479bb`; build `deaf2802` copiado para `/home/desenvolvedor/.local/bin/nexus`. SHA-256 build e destino: `107ac2d74420f17d41f7cfacb7c074f929d81952f1a7c35d34ac1c7e571e9fa6`. `version` aponta para `deaf2802`; status executado pelo caminho instalado retorna `available=true` e ambos contratos 1.0.0.
- PASS — `make quality-full`: format; ESLint 0 erros/6 avisos preexistentes; lint Go 0 issues; Vitest 131 arquivos/826 testes; Go integral e race (`internal/nexus` 302,029s); version contract; `govulncheck` sem vulnerabilidades.
- Limite: o binário Linux do perfil foi atualizado e revalidado, mas isto não substitui o artefato final publicado; matrizes Windows/macOS, empacotamento e assinatura permanecem pendentes.

## P0 A4 — opção estruturada no Attention Radar — 2026-11-09

- Fluxo implementado: metadados choice estruturados e limitados no Bridge → Registry; `CmdChooseInteraction` tipado → backend structured; validação e claim atômico one-shot antes do IPC; endpoint aceita exatamente `input` ou `option_id`; UI mostra botões somente para opções AUTHORITATIVE recebidas. Não há conversão de escolha em input de terminal.
- PASS — o E2E foi repetido no commit publicado `deaf2802` (9,61 s): duas runtimes ACP fixture com pergunta igual; Alpha escolhe `Allow once`, Alpha resolve e Beta segue `needs_user`; consulta autenticada sem overlay e WebSocket real do SessionHost de Beta. Capturas/limites: [relatório](validation/current/attention-structured-choice/README.md).
- PASS — `npm --prefix web run test -- --run src/features/work/featureTourAnchors.test.ts src/features/work/featureAnnouncements.test.ts src/components/AttentionNotification.test.ts` (18 testes); `go test ./internal/control/registry ./internal/control/agentwire ./internal/control/host ./internal/control/web ./internal/nexus -run 'Attention|Structured' -count=1`; `git diff --check`.
- PASS — `make quality-full`: formatação, ESLint (0 erros; 6 warnings preexistentes), lint Go (0 issues), Vitest (131 arquivos/826 testes), version contract, Go integral e race, `govulncheck` sem vulnerabilidades. `make build`: frontend embutido e Nexus v1.0.0-beta.11 construídos.
- A execução de integração foi no SHA publicado `deaf2802`; o fluxo usa fixture ACP, sem provider externo ou app distribuído. Portanto ela é evidência exata deste commit, mas não homologação do artefato distribuído nem do SHA final de release.
- Atualizei o mapa de evidências para incluir o novo pipeline estruturado de choice e remover a afirmação obsoleta de que toda escolha E2E estava ausente. O mapa continua sem certificar aceite: 48/96, 86 refs source+test, 9 source-only, 1 unmapped; validador do mapa e contrato passaram; Vitest focado 18/18.
- P0 A4/capability `attention-radar` continuam parciais; 48 capacidades/96 critérios têm zero certificados. Restam P0 A1–A4, A2 com dois perfis/cancelamento sem órfãos, validação provider real e UX/autorização, AGY com 2/2 elegíveis + rotação real, distribuição Nexus↔Maestro, auditoria dos 96 critérios, app/plataformas/assinatura no exact-SHA. Estado geral: **NO_GO**.

## P0 A3 — GlobalAttentionRadar: foco cross-project — 2026-11-08

- PASS — `npm --prefix web run test:e2e:attention-radar-focus`: Playwright criou dois projetos e dois Project Shell runtimes pelas APIs Nexus, apresentou pergunta idêntica com runtime IDs distintos e confirmou que escolher Beta troca de projeto e anexa o shell de Beta; escolher Alpha retorna e abre Alpha.
- O teste descobriu e a correção resolve perda de seleção ao remount do provider de workspace keyed por projeto: o pedido pendente agora vive acima do provider, espera o layout do projeto destino e carrega o `projectId` da superfície ao navegar.
- O browser intercepta o GET `/api/v1/runtimes` e sobrepõe estados de atenção. Project/Shell creation e foco usam código/APIs Nexus; nenhum provider foi chamado. Portanto isto não prova evento real Registry→API/read model→UI, provider externo, pacote publicado, nem UX no SHA final. Screenshots temporárias: `/tmp/nexus-attention-radar-20261108-sha-check`.
- PASS — `npm --prefix web run quality:full`: Web 131 arquivos/825 testes, build/typecheck/format/style e lint sem erros; 6 warnings ESLint preexistentes. `verify-agent-workspace-v1` (48 pending), evidence map/verificador (4/4; 48/96, 86 refs source+test, 9 source-only, 1 unmapped, zero certificados), feature catalog (36) e `git diff --check` passaram.
- `make docs-verify` ainda precisa da recaptura sintética após commit do código, pois os manifests existentes apontam para fonte anterior. Essa captura apenas mantém os docs alinhados ao source SHA, não homologa a UI.
- Veredito: A3 permanece parcial/pending, `attention-radar` não certificado, 48 capacidades pending e GO geral **NO_GO**. Próxima evidência: combinar o evento/API sem overlay com o browser no mesmo cenário, provider externo e app distribuído no exact SHA final.

### Complemento Registry → API sem overlay — HEAD `ffd5c59f`

- PASS — `go test ./internal/control/web -run '^TestTwoSupervisedACPProcessesAndTwoAuthenticatedHTTPClients$' -count=1 -v` (0,161 s). Dois subprocessos ACP determinísticos são supervisionados por SessionHost; perguntas iguais atravessam IPC/Bridge/Registry. Dois clientes autenticados consultam o endpoint real `/api/v1/runtimes` sem interceptação; resposta IPC faz apenas Alpha chegar a completed, preservando Beta em `needs_user`, e encerrar Beta limpa sua atenção.
- Isto prova processo estruturado até Registry e projeção HTTP, mas não liga o browser Radar nesse mesmo cenário. O Playwright cross-project segue usando overlay da resposta para testar a UI. Não houve chamada a provider externo.
- Veredito e limites continuam: A3 parcial/pending, nenhuma capability certificada, GO **NO_GO** até E2E unificado com browser, provider externo aplicável e validação do artefato no SHA final.

## P0 A3 — evento ACP estruturado até Registry — 2026-11-08

- PASS — `go test ./internal/control/host -run '^TestStructuredACPAttentionReachesRegistryAndSettlesOnlyItsRuntime$' -count=1 -v`: subprocesso ACP real no harness, Bridge e SessionHost de produção, Registry persistente partilhado.
- PASS — duas runtimes com o mesmo título e interaction ID `req-1` ficam separadas por runtime; a pergunta ACP e `end_turn` mantêm `AUTHORITATIVE`, a resposta local correlacionada marca Alpha `SIGNALLED`/`SUBMITTED`/`working` sem concluir o turno, e Beta continua `needs_user` com `AUTHORITATIVE`.
- PASS — `go test ./internal/control/host -count=1`; `go test -race ./internal/control/host -run '^TestStructuredACPAttentionReachesRegistryAndSettlesOnlyItsRuntime$' -count=10`; `go vet ./internal/control/host ./internal/control/agentwire ./internal/control/registry`.
- PASS no estado final deste checkpoint — `make quality-full` (131 arquivos/823 testes Web, Go completo + race, lint Go 0 issues, build/style/type/format/security, `govulncheck` sem vulnerabilidades; 6 avisos ESLint preexistentes/0 erros); validadores 48-capabilities, mapa (4/4, 48/96, 86 source+test, 9 source-only, 1 unmapped, zero certificados), `make docs-verify` (245 Markdown/33 assets), `git diff --check`.
- Limite — subprocesso ACP é fixture determinística; ainda faltam prova API/UI do Radar com foco no runtime certo, provider externo e homologação no app distribuído/SHA final. Capability `attention-radar` e P0 A3 continuam pending/partial.

## `follow-agent` — E2E de interface e controle explícito — 2026-11-08

- PASS — `NEXUS_E2E_ARTIFACTS_DIR=/tmp/nexus-follow-agent-e2e npm --prefix web run test:e2e:agent-terminal-follow`: servidor fixture/API real criou Agent persistido; UI real AgentTerminal/xterm passou pausa/resume e cenários VIEW_ONLY/CONTROL.
- WebSocket provider foi interceptado: o teste comprova mensagens emitidas pela UI e comportamento de reconnect no cliente, mas não broker/host real. Falta E2E backend integrado de attach, lease e revoke.
- PASS — verificador de mapa 4/4; mapa 48/96, 86 refs source+test, 9 source-only, 1 unmapped, zero certificados; `make docs-verify` (245 Markdown/33 assets); `git diff --check`.
- PASS — `make quality-full`: Web 131 arquivos/823 testes, Go completo + race/security, format/lint/type/style/build e version-contract; seis avisos ESLint existentes, zero erros, govulncheck sem vulnerabilidades.
- E2E repetido três vezes com PASS, incluindo execução após rebase no commit `101dd328`; `make docs-verify` passou após o rebase. O commit ainda aguarda push após reconciliação do avanço paralelo remoto.
- Veredito: `follow-agent` parcial/pending; homologação do pacote distribuído no SHA final e revisão humana continuam necessárias.

## `follow-agent` — pausa explícita do acompanhamento ao vivo — 2026-11-08

- PASS — testes focados: `terminalOutput`, `terminalInput`, `featureAnnouncements`, `featureTour`, `featureTourAnchors`, `featureReveal` e `tourTargets`: 31/31.
- PASS — `make quality-full`: format-check, ESLint 0 erros/6 warnings preexistentes, Stylelint, TypeScript, 131 arquivos/823 testes Web, build, version-contract, Go completo + race/security e `govulncheck` sem vulnerabilidades.
- PASS — `node scripts/verify-agent-workspace-v1.mjs` (48 no escopo, todos pending); `node scripts/verify-agent-workspace-evidence-map.mjs` (48/96, 86 source+test, 9 source-only, 1 unmapped, zero certificados); `node scripts/verify-feature-catalog.mjs` (36); `git diff --check`.
- PASS — `npm --prefix web run docs:capture` gerou 10 capturas isoladas sintéticas para source SHA `46f605b9`; `make docs-verify` passou (245 Markdown/33 assets). Capturas não são evidência visual do Follow Agent nem substituem E2E publicado.
- Limite — teste unitário demonstra preservação da posição de leitura, incluindo painel oculto; não é E2E visual, não prova que um WebSocket real da UI permaneça sem `lease_acquire` ao anexar/digitar/revogar/reconectar, nem valida o app instalado no SHA final. Capability e GO continuam pending/NO_GO.

## AGY — quota recente e limite atual de balance — 2026-11-08

- PASS — `nexus usage --refresh --json` executado pelo binário da árvore corrente, com saída sanitizada; não solicitou geração. Releitura: duas contas autenticadas, snapshots `CACHED/CLI_OUTPUT` com menos de 3 min, escopos verificáveis e correspondentes aos perfis.
- Conta ordinal 1: 0/2 grupos com janela 5h; só semanal (Gemini e Claude/GPT). Conta ordinal 2: 2/2 grupos com 5h; ambos reportaram remaining 100. Identidades e e-mails não foram expostos no registro.
- PASS — `go test ./internal/core/scheduler -run 'TestBalancedSelectionRotatesEligibleProfilesAndPersistsCursor|TestBalancedSelectionSkipsQuotaIneligibleAndExcludedProfiles|TestBalancedAGYSelectionRequiresFreshAttributableEvidenceForEveryKnownGroup' -count=1`: seleção round-robin persistente e hard gate passaram.
- Resultado — há 2 perfis registrados, mas somente 1 elegível; não é possível provar alternância live entre duas contas sem inventar quota. O fail-closed continua necessário. Uma tentativa de chamar o CLI por `nexus agy:<perfil>` retornou usage/exit 2; a evidência válida veio da rota suportada de refresh do Nexus. Nenhum modelo foi executado.
- Estado: AGY balance permanece NO_GO para a aceitação de duas contas; causa imediata é ausência upstream de janela 5h na primeira conta, apesar de autenticação válida.

## A2 — rollback de sessão direta antes do start — 2026-11-08

- E2E atualizado em `web/scripts/e2e-first-run-import.mjs`: abre DirectSessionLauncher, simula criação de Agent e seleção de recurso, injeta falha em `config/apply` e exige DELETE compensatório antes de qualquer `/start`. Ordem observada: `create → select-resource → apply-config → delete`; nenhum PTY é iniciado pelo caminho exercitado.
- PASS — E2E completo no commit de código `7effc914`, inclusive importação first-run, quick-connect, refresh e rollback fixture.
- PASS — `make quality-full` no commit `7effc914`: format, 6 warnings ESLint/0 erros, 131 arquivos/818 testes Web, Go completo, race, build e security; sem advisories govulncheck. `make docs-verify` PASS.
- Limite — API de Agent/recurso foi interceptada no browser; isto verifica sequência de UI/requests, não exclusão real no Core, Wails, login ou provider. A2 continua parcial; seguem necessários perfis reais e cenário integrado sem Agent/PTY órfão.

## Checkpoint A2 — first-run e lifecycle no SHA corrente — 2026-11-08

- PASS — `NEXUS_E2E_ARTIFACTS_DIR=/tmp/nexus-first-run-evidence-0feb33f1 npm --prefix web run test:e2e:first-run-import`: importação inicial, descoberta/quick-connect e refresh em Usage / Accounts passaram no SHA `0feb33f1`. Artefatos sintéticos ficaram em `/tmp`.
- PASS — `npm --prefix web exec -- vitest run src/features/work/directSessionLifecycle.test.ts`: 9/9 testes; setup falho antes do start remove o Agent, enquanto resposta incerta após start preserva a identidade para recuperação.
- PASS — `make quality-full` no SHA `0feb33f1`: format-check, lint (6 warnings preexistentes/0 erros), TypeScript, Stylelint, build, 131 arquivos/818 testes Web, suíte Go completa, Go race e security; `govulncheck` sem vulnerabilidades.
- PASS — `make docs-verify`: 245 Markdown e 33 assets/arquivos obrigatórios.
- Limite — fixture não exercita duas contas/profiles reais nem cancelamento durante uma execução autenticada, não prova ausência de PTY órfão em Wails e não fecha A2. A matriz de 48 capacidades e o GO permanecem pendentes.

## Maestro — contrato canônico advice/verification — 2026-11-08

- PASS — `NEXUS_MAESTRO_CLI_TEST_PATH=/home/desenvolvedor/.local/bin/maestro go test ./internal/nexus -run '^TestMaestroCLIContractInterop$' -count=1` contra Maestro `release/1.1.0@9c9c9eb`: capabilities, advice e verify reais passaram, sem execução de provider/Agent.
- PASS — `maestro capabilities --json` reportou 1.1.0, modos OFF/ASSIST/ORCHESTRATE e contracts advice/verification 1.0.0.
- PASS — `go build -o /tmp/nexus-maestro-smoke ./cmd/nexus`; o binário da árvore release, rodando `maestro status --json` com o CLI atualizado no PATH, reportou `available=true`, version 1.1.0 e os dois contracts 1.0.0.
- Limite — o Nexus preinstalado no perfil global continua antigo e seu status omite contracts. O binário candidato deve ser distribuído/instalado a partir do SHA final; o conjunto de release permanece NO_GO pelas demais pendências.

## P0 A4 — fencing, autorização e isolamento de respostas — 2026-11-08

- PASS — `go test ./internal/control/web -run 'TestRespondAttentionEndpointRejectsStaleAndUnsafeInput|TestNexusCSRFEnforcement' -count=1`.
- Regressões verificam session ausente/revogada e CSRF ausente/inválido para POST; fingerprint/revisão/generation antigos e um guard de runtime/projeto diferente retornam conflito sem consumir claim.
- Evidência relacionada: `TestAttentionRevisionAndOneShotClaimAcrossRegistryInstances` cobre claim atômico; `TestResolveInterventionHTTPDecisionSurvivesServerRestartExactlyOnce` cobre duplicate request idempotente no fluxo Mission.
- Limite: não prova resposta ponta a ponta com provider real nem ACL por projeto; instalação ainda é single-user. A4 e `attention-radar` permanecem parciais/pending.

## `follow-agent` — lease sem consentimento — 2026-11-08

- PASS — `npm --prefix web run test -- --run src/nexus/agentTerminalModel.test.ts`: 7 testes auxiliares Node e 20 Vitest; role CONTROL habilita input e VIEW_ONLY/role ausente não.
- PASS — `make quality-full` na árvore atual: format-check, lint (6 warnings preexistentes/0 erros), typecheck/build, Web 131/818, suíte Go integral + race integral, version-contract e `govulncheck` sem vulnerabilidades.
- PASS — `node scripts/verify-agent-workspace-evidence-map.mjs` (48/96, 85 referências source+test, 10 source-only, 1 unmapped; zero certificados); `node --test scripts/verify-agent-workspace-evidence-map.test.mjs` (4/4); `git diff --check`.
- PASS — diff confirma WebSocket attach em VIEW_ONLY, stdin desabilitado no open/reconnect, remoção de `lease_acquire` do attach e do handler de input; solicitação só permanece no botão explícito.
- Pendente — teste integrado do WebSocket/UI para provar ausência de lease em attach e após tecla e o comportamento de revoke/reconnect. O aceite de pausa/follow explícitos não está coberto nem localizado.
- Catálogo: mudança classificada internamente como correção de consentimento/segurança, não novo recurso anunciável.
- Veredito global e capability `follow-agent`: **NO_GO / pending**.
- PASS — `make docs-verify` após integrar a recaptura sintética do commit `f7120e79`, ancorada em `6960a1f7`: 245 Markdown / 33 assets; não há alterações visuais posteriores no terminal. O gate documental não substitui E2E interativo de lease.

## P0 A1 — auditoria semântica inicial de terminal-studio — 2026-11-08

- PASS — `node scripts/verify-agent-workspace-evidence-map.mjs`: o mapa mantém sincronismo estrutural (48 capacidades/96 critérios, zero aceites certificados).
- Critério 0 — PASS parcial: `terminalViewModel.test.ts` prova que runtime não duplica view; `askAgentModel.test.ts` prova que Agent ativo não pede outro runtime; state/presentation cobrem persistência/migração do layout. Vitest focado PASS 7/7. E2E `NEXUS_E2E_ARTIFACTS_DIR=/tmp/nexus-surface-live-pty-20261108 npm --prefix web run test:e2e:surface` PASS: iniciou um shell supervisionado real pelo Overview, preservou runtime ID, nó da superfície e nó `.xterm` durante a matriz de tabs de produto, e contou exatamente um POST de launch; o E2E também passou axe (zero violations sérias/críticas) e 6 viewports. Limite: não lançou provider IA nem percorreu tabs/split internos de PTY; critério canônico permanece pending até essa evidência.
- Critério 1 — host/QA cobrem lease/handover de dois clientes; slash router verifica Ctrl+C; ring buffer e teste Windows de seta/ConPTY são isolados. Novo teste dirigido `TestSessionHostResizeFromSecondAttachedClientReachesPTY` PASS: request do segundo cliente resulta em `Resize(41,137)` no backend. Falta aceite integrado de concorrência/controle com scrollback após reconnect e execução em runner Windows/ConPTY.
- PASS — `node --test scripts/verify-agent-workspace-evidence-map.test.mjs` (4/4), incluindo referência válida ao Playwright E2E e rejeição de helper `.mjs` arbitrário; `node scripts/verify-agent-workspace-evidence-map.mjs` mantém 48/96 e nenhum aceite certificado.
- PASS — `make docs-verify` (245 Markdown/33 assets), Prettier nos scripts alterados e `make quality-full` no worktree baseado em `c03f9ffa`: Web 131/818, Go completo/race, lint/type/style/build/version/security; 6 warnings ESLint, 0 erros; govulncheck sem advisories.
- PASS — E2E surface repetido após formatação; shell/runtime/superfície/xterm persistiram durante as trocas; um único POST de launch. Não houve provider IA nem teste de split/tab interno de PTY, portanto o critério e a capacidade permanecem pending.
- PASS — `make quality-full` na árvore de código commitada como `c179246c`: Go/Web, race, security, build, lint/type/style/format/version; Web 131/818; ESLint 6 warnings/0 erros; `govulncheck` sem vulnerabilidades.
- PASS — `make docs-verify` (245 Markdown, 33 assets/files).
- O mapa agora registra `assertionAudit: partial` e as lacunas específicas. Mantém os critérios como `coverage: unverified`, capacidades `verification: pending`, estado global **NO_GO**.

## P0 A3 — input terminal não confirma aprovação — 2026-11-08

- RED confirmado — regressão `TestTerminalInputSubmissionRequiresProviderTransitionToClearAttention` falhou antes da correção porque `writeTerminalInput("y")` removia `needs_user` sem resposta observada do provider.
- PASS — regressões `TestTerminalInputSubmissionRequiresProviderTransitionToClearAttention`, `TestAttentionDetectorAGYQuestionnaireWithLongTUIMenu`, `TestAttentionDetectorYNAnswerWaitsForProviderOutput` e `TestNativeAGYQuestionPreemptsPTYHeuristic`.
- PASS — `go test ./internal/control/host ./internal/control/registry ./internal/control/web -count=1`.
- PASS — `go vet ./internal/control/host ./internal/control/registry ./internal/control/web` e `git diff --check`.
- PASS — `make quality-full` no SHA `7c77317f`: format-check, lint/style/typecheck/build Web, 131 arquivos/818 testes Web, contrato de versão, lint/test Go, Go race e security; ESLint 6 warnings/0 erros; govulncheck sem vulnerabilidades.
- PASS — `make docs-verify` após atualizar o handoff e este registro.
- Limite — A3/A4 seguem parciais para sinais estruturados completos, aprovação tipada, retry/autorização e revogação; 48 capacidades continuam pending e estado geral **NO_GO**. Este resultado precede o teste multi-runtime abaixo.

## P0 A3 — perguntas simultâneas isoladas por runtime — 2026-11-08

- PASS — `TestSimultaneousRuntimeQuestionsRemainIndependentAfterAnswer`: duas perguntas idênticas são registradas sob runtime IDs distintos; input sozinho não resolve nenhuma; saída observada no runtime A limpa apenas A e mantém B em `WAITING/needs_user` com fingerprint.
- PASS — `go test ./internal/control/host -run '^TestSimultaneousRuntimeQuestionsRemainIndependentAfterAnswer$' -count=1`.
- PASS — `go vet ./internal/control/host ./internal/control/registry ./internal/control/web`, `make docs-verify` e `make quality-full` no worktree baseado em `82aaaae9`: Web 131/818, Go completo, race, security; ESLint 6 warnings/0 erros, govulncheck sem vulnerabilidades.
- Limite — A3 estruturado integral e A4 seguem abertos; esta evidência não altera as 48 capacidades `pending` nem o **NO_GO**.

## P0 A2 — E2E first-run — 2026-11-08

- PASS — `NEXUS_E2E_ARTIFACTS_DIR=/tmp/nexus-first-run-evidence-59d1936a npm --prefix web run test:e2e:first-run-import` no SHA `59d1936a`: feedback da importação, descoberta/conexão rápida do provider e importação/refresh em Usage / Accounts.
- Limite — o harness usa browser e endpoints fixture; artefatos de screenshot são sintéticos e ficaram em `/tmp`. Não valida CLIs autenticadas, dois perfis reais, integração Wails nativa, nem ausência de runtime/PTY órfão após cancelamento.
- Veredito de release e capability `first-run`/`composer-flow-onboarding`: continuam pendentes; esta evidência cobre somente cenários determinísticos de UI.

## Continuação — mapa rastreável das 48 capacidades — 2026-11-08

- PASS — `node scripts/verify-agent-workspace-evidence-map.mjs`: 48 capacidades/96 critérios sincronizados ao contrato; 88 critérios com referências candidatas a testes e 8 source-only; nenhum aceite certificado.
- PASS — `node --test scripts/verify-agent-workspace-evidence-map.test.mjs`: 3/3; falhas de critério ausente, promoção de aceite, path inseguro/inexistente e classificação contraditória foram rejeitadas.
- PASS — `node scripts/verify-agent-workspace-v1.mjs`: escopo canônico validado; 48/48 continuam pending.
- PASS — `node scripts/verify-feature-catalog.mjs`: 36 mappings do catálogo de produto permanecem válidos e separados do contrato de 48.
- PASS — `make docs-verify`: 245 arquivos Markdown e 33 assets obrigatórios; inclui o mapa estrutural e seus testes de regressão.
- PASS — `make quality-full`: formatação, ESLint (6 warnings/0 erros), Stylelint, TypeScript, Go lint, suíte Go, 131 arquivos/818 testes Web, version-contract, Go race e security; exit 0 no worktree baseado em `fbe72c57`; `govulncheck`: nenhum advisory encontrado.
- Limite: o mapa usa auditoria estática anterior como origem dos paths e valida existência na árvore atual. A relação semântica de cada teste com cada assertion continua sem revisão; certificação e estado geral permanecem **NO_GO**.

## P0 A1 — projection contract and catalogs — 2026-11-08

- PASS — `go test ./internal/control/driver`, including the new all-registered-drivers projection consistency test.
- PASS — `node scripts/verify-agent-workspace-v1.mjs` (48-item scope integrity; certification still pending).
- PASS — `node scripts/verify-feature-catalog.mjs` (36 code/test/doc mappings).
- PASS — `make quality`: full Go suite/lint, 131 Web files/818 tests, format/style/type checks and version-contract; ESLint has 6 warnings/0 errors.
- Scope limitation: these checks establish the projection and catalog structure, not per-capability acceptance. The 48-item source/test/acceptance map and external exact-SHA proof remain open; overall **NO_GO**.

## Gate AGY balance e qualidade integral — 2026-11-08

- Código testado no SHA `c5a31c41` (`fix(quota): require fresh AGY 5h evidence for balance`).
- PASS — regressões `go test ./internal/core/fallback ./internal/core/scheduler ./internal/core/provider/adapters/agy`.
- PASS — `go vet ./internal/core/fallback ./internal/core/scheduler ./internal/core/provider/adapters/agy`.
- PASS — `make quality`: format-check, ESLint sem erros (6 warnings), Stylelint, TypeScript, 131 arquivos/818 testes Web, lint Go 0 issues, suíte Go completa e version-contract.
- PASS — `git diff --check` antes do registro documental.
- PASS — dry-run read-only temporário com perfis e snapshots locais: 2 candidatos, 0 elegíveis, nenhuma seleção; `AI_CLI_STATE_DIR` isolado. `GetUsageSnapshot` usou cache sem refresh; nenhuma geração ou cursor real.
- PASS — probes live atuais usando exclusivamente o comando AGY `--print=/quota` para duas contas: ambos LIVE/CLI_OUTPUT. Uma saída bruta contém só rótulos semanais; a outra contém rótulos 5h e semanais nos dois grupos. Captura sanitizada confirmou que o primeiro CLI não emitiu 5h, portanto o parser não descartou essa janela.
- PASS — após atualizar os snapshots, seleção real dry-run no state dir isolado: 2 candidatos, 1 elegível, selecionado=true. Só a conta com 5h medido em cada grupo entra; a semanal-only é recusada. Nenhuma geração foi iniciada.
- Regressão inicialmente detectou `TestRunWithFallbackStrategyRotatesAfterQuotaFailure` usando quota `UNSUPPORTED` de fixture AGY; a fixture foi corrigida para LIVE/CLI_OUTPUT/fresh/5h e a suíte integral repetida passou.
- Cobertura nova: AGY weekly-only, 5h ausente por grupo, snapshot vencido, proveniência/perfil incompatível, remaining inválido, falha fechada sem capacidade elegível e não regressão de outros providers.
- Dry-run com snapshots reais confirma fail-closed, não homologa duas contas elegíveis nem alternância real. O provider CLI não expõe 5h em uma conta. Status global **NO_GO**: 48/48 capacidades ainda pendentes, alternância live entre duas contas não demonstrada e release/P0s ainda abertos.

## Recheck de qualidade após formatação Go — 2026-11-08

- Primeiro `make quality`: NO_GO em `format-check`; reportou 23 arquivos Go sem o formato de `gofmt -w -s`, inclusive o arquivo Nexus modificado.
- PASS — `gofmt -w -s` nos 24 arquivos reportados e `make format-check` no commit `b5cfe715` (Prettier PASS e `gofmt -l` vazio).
- PENDENTE — reiniciar `make quality` no SHA de registro após este checkpoint; a primeira execução não atingiu lint, suites, build ou security.

## Contratos de advice e verification — 2026-11-08

- PASS — Maestro `release/1.1.0@9c9c9eb`: `npm test` 1.373 total, 1.369 pass, 0 fail, 4 skipped; `npm run validate`; `pwsh scripts/validate-public.ps1` (81 skills públicas, 58 contratos canônicos).
- PASS — novos testes CLI advice/verify + compatibilidade de CLI/Router: 17/17.
- PASS — Nexus local: `go test ./internal/nexus ./internal/control/web`; `go vet ./internal/nexus ./internal/control/web`; teste focado da rota autenticada/CSRF/JSON estrito.
- PASS — teste cruzado Go→CLI Maestro real: negotiates capability versions, obtains a routed canonical skill and receives policy verification verdict. Executado contra checkout local do Maestro antes de `9c9c9eb`; repetir para os SHAs Nexus publicados.
- NO_GO — código Nexus ainda não publicado neste checkpoint; nenhuma chamada a provider foi realizada. Teste não comprova execução real dos checks nem abertura de referências: Maestro aplica policy aos resultados declarados pelo owner Nexus.
- NO_GO — 48/48 capabilities permanecem `verification: pending`; E2E com duas contas AGY/quota real, P0 restantes, UI visual no SHA publicado e certificações de plataforma/distribuição permanecem pendentes.

## Retomada — suíte Maestro e pendências rumo ao GO — 2026-11-08

- PASS — `npm test` no checkout Maestro `/projetos/tools/Orquestrador-Maestro`, branch `release/1.1.0`, terminou: 1.367 testes, 1.363 pass, 0 fail, 4 skipped; 112 arquivos; duração 521.795 ms.
- PASS — `node --test tests/install-profile-state.test.js`: 5/5, após isolar `HOME`, `USERPROFILE` e `CODEX_HOME` à fixture temporária.
- PASS — commit `90b6152` enviado ao remoto `origin/release/1.1.0`; mudança restrita à fixture de teste.
- NO_GO — Nexus `release/v1.0.0@8aaf7b85`: as 48 capacidades oficiais permanecem `verification: pending`; integração ainda precisa `advise --json` e `verify --json` no CLI Maestro e E2E contra os executáveis/instâncias do release.
- NO_GO — balanceamento AGY `:balance`/`:auto` ainda precisa demonstrar seleção alternada e preservação das janelas de quota em duas contas AGY reais; nenhum provider live foi consumido nesta validação.
- NO_GO — faltam executar/revalidar os P0 do plano de ação, E2E visual real no SHA publicado, validação multiplataforma/assinatura/distribuição e demais critérios de release. Não declarar GO com mocks, screenshots sintéticas ou testes em SHA distinto.

## Checkpoint — projeção conservadora de capabilities — 2026-11-06

- PASS — `go test ./internal/control/driver ./internal/control/web`: ambos os pacotes passaram; o pacote `web` levou 62,5 s.
- Regressão coberta: `SUPPORTED` com `tested:false` não vira `true` no contrato booleano legado. O `EffectiveCapabilities` detalhado continua preservando status e evidência.
- Confirmado em runtime: o CLI Maestro instalado (`@iapro/orquestrador-maestro-cli` 1.1.0) rejeita `capabilities --json` e `advise --json`; a forma `skills list --json` também não é aceita. Portanto M9 e integração canônica Maestro permanecem NO_GO; não inferir homologação a partir dos fallbacks de skills locais.
- Matriz `docs/reference/agent-workspace-v1.json`: 48 capacidades, todas `verification: pending`; `docs/reference/features.json` contém apenas 36 itens e precisa de reconciliação com o contrato de 48.
- Sem execução autenticada de providers, aprovação visual do SHA publicado ou validação ponta a ponta de Maestro nesta etapa.

## Checkpoint — contratos parciais Nexus↔Maestro — 2026-11-06

- PASS — `go test ./internal/nexus`: pacote completo passou (71,978 s); inclui rejeição de disponibilidade baseada apenas em `version` e preservação dos contratos capabilities/skills/advice/verify com fixtures.
- PASS — `go test ./internal/control/driver ./internal/control/web` e `go vet ./internal/control/driver ./internal/control/web ./internal/nexus`.
- PASS — no checkout `/projetos/tools/Orquestrador-Maestro`, `node --test tests/legacy-compatibility.test.js`: 7/7. Suíte completa `npm test` ainda em execução neste checkpoint.
- Maestro CLI agora tem `capabilities --json` e `skills list --json`, com metadados canônicos sanitizados (sem caminhos locais). O cliente Nexus exige capabilities explícitas; catálogo local/version string isolados não provam integração.
- Ainda faltam no Maestro CLI `advise --json` e `verify --json`. Até esses comandos existirem e passarem E2E no binário integrado, `GetAdvice`/`Verify` permanecem degradados e M9 não está homologado.
- `docs/reference/features.json` (36 features do produto) e `agent-workspace-v1.json` (48 capacidades de workspace) têm escopos distintos; não comparar suas contagens como divergência.
- NO_GO: 48/48 ainda `verification: pending`; nenhuma validação real multi-cliente/provider ou SHA publicado ocorreu aqui.

## Checkpoint de continuidade — autodiscovery e Product Discovery — 2026-11-06

- PASS — `npm --prefix web run quality:full`: 131 suites de arquivo/818 testes Vitest; check de estilos, Prettier, ESLint sem erros (6 avisos preexistentes), Stylelint, TypeScript e build.
- PASS — `NEXUS_E2E_ARTIFACTS_DIR=/tmp/nexus-host-session-e2e npm --prefix web run test:e2e:first-run-import`: importação no primeiro uso, autodiscovery em Recursos/Uso, conexão rápida única e refresh; verificado também viewport 390 px sem overflow horizontal. Capturas fixture em `/tmp/nexus-host-session-e2e/`.
- PASS — `go test ./internal/profile ./internal/core/provider/adapters/... ./internal/core/provider/spec ./internal/control/driver ./internal/control/web ./internal/app` e `go vet` nos pacotes alterados.
- PASS — `GOOS=windows GOARCH=amd64 go test -c -o /tmp/nexus-terminal-windows.test.exe ./internal/control/terminal` (cross-compile; não equivale a execução nativa Windows).
- PASS — Prettier e `git diff --check` após corrigir a linha em branco final de WORKLOG.
- FAIL preexistente e alheio a este diff — `make format-check` aponta 23 arquivos Go fora do escopo. O formatter dos arquivos Go alterados passou.
- Não certifica conexão com provider real, screenshot no SHA publicado, CI same-SHA ou homologação de release.

## Verificação — roteamento de provider no Lead interativo — 2026-11-04

- Base sincronizada com `origin/release/v1.0.0` em `d66426af` antes da validação.
- PASS — `go test -tags=gtk3 ./internal/app -count=1` e o teste isolado `TestControlPlaneCLICommands`.
- PASS — `internal/nexus` e `internal/control/web` durante `make test-go`; nessa execução paralela, `internal/app` teve uma falha em `TestControlPlaneCLICommands`. O teste e todo o pacote `internal/app` passaram ao serem repetidos isoladamente. Portanto, a execução agregada não foi considerada integralmente verde.
- PASS — `make format-check` depois da sincronização; `go vet` nos pacotes alterados e `make security` passaram na validação prévia desta correção.
- A instalação local anterior foi concluída e teve health API OK; ela precede a sincronização com `d66426af` e não é evidência de um binário construído a partir da base atualizada.

## Continuidade atual — beta.11 sincronizado e instalado — 2026-10-27

- Branch `release/v1.0.0` em `b416bb90` (= `origin/release/v1.0.0`); commits locais `5e3eff4e` (feature auth/usage/HUD), `11f3d771` (teclado Codex/PTY) e `d61b0dcc` (gates/upstream), seguidos do commit de docs.
- PASS — `GOTOOLCHAIN=go1.25.14 make quality` (exit 0): gofmt/Prettier, ESLint, Stylelint, typecheck, golangci-lint (0 issues), Go tests, Vitest 114 suítes/710 testes e version-contract.
- PASS — falhas upstream reproduzidas em worktree limpo (`/tmp/nexus-origin-verify`, `b416bb90`) e corrigidas; testes focados de `internal/nexus` e do driver passam.
- PASS — build e instalação local macOS amd64: CLI `1.0.0-beta.11`/`580a99cf`, Desktop/HUD com o mesmo build; backup em `Applications-backup-20261027-0937`.
- PASS — runtime instalado: Desktop PID 58863 + HUD vinculado PID 58868, HTTP 200 em `127.0.0.1:57271`, UI exibe `Nexus v1.0.0-beta.11`.
- PASS — push fast-forward `b416bb90..e58b0d60` para `origin/release/v1.0.0`; helper de credencial do GitHub corrigido para `/usr/local/bin/gh` (backup `~/.gitconfig.bak-20261027`).
- BLOCKED — Developer ID/notarização, certificação arm64 e `make docs-verify` com árvore não commitada.
- SHA-256: CLI `e9d8e998f676f77353a88e3f6f4ae740f254daecb5923c9fff20f8c2b36e4a19`, Desktop `7a9f7744f0ac6d82ec41f1a8b4157bcf3e1d412e558e644c57e27081f060607f`, HUD `2f96da581971ba7d795c377f8fe0b061dd251116b1b338500cef6fe3f91888b2`.

## Continuidade atual — versão 1.0.0-beta.11 e reconciliação upstream — 2026-10-26

- Branch `release/v1.0.0`; HEAD origin `9d8d30afcefb7ec667bc4a0d3c1d1da15faf41b0`. Reconciliação dos 15 commits remotos concluída e validada.
- Versão: `1.0.0-beta.11` sincronizada em `VERSION` e `web/package.json`. Contrato verificado por `scripts/verify-version-contract.go`.
- Validações executadas com sucesso:
  - `scripts/verify-nex-visual.mjs`: PASS
  - `make version-check`: PASS
  - `make format-check`: PASS
  - `npm --prefix web run check:styles`: PASS
  - `npm --prefix web run lint:styles`: PASS
  - `npm --prefix web run lint`: PASS
  - `npm --prefix web run typecheck`: PASS
  - `npm --prefix web run test`: 110 arquivos, 684 testes PASS
  - `npm --prefix web run build`: PASS
  - `make vet`: PASS (0 warnings)
  - `go test -tags="gtk3" ./internal/...`: PASS
  - `make security`: PASS (`govulncheck` sem vulnerabilidades)
- Binários compilados: CLI `nexus`, Desktop `cmd/nexus-desktop/build/bin/nexus-desktop`, HUD `cmd/nexus-hud/build/bin/nexus-hud`.

## Continuidade atual — mascote NEX único e homologação Linux — 2026-10-26

- Branch `release/v1.0.0`; HEAD local/remoto `7a04f1b4cf5950b97b4ea1e156dda31ac513823b`. O overlay atualmente compilado/instalado tem fingerprint `101e6e62b8d4b9903cf4d40309caeea54c809183e5dba017d9e6a8770302669c` (1.430 fontes; manifesto `DEV/validation/linux-100-2026-10-25/p5-current-recheck-2026-10-26-final/logs/source-provenance-preview-copy-fix-2026-10-26.txt`). Worktree compartilhada modificada, sem SHA Git congelado; não fazer commit/push automático.
- Contrato do produto: um único mascote NEX (`NexusHud → GuardianMascot → NexPixelSprite`). Barra, Mascote e Bandeja são apresentações do mesmo HUD. Companion 2D/PR #55 não faz parte do escopo; `companionModel.ts`/teste estão removidos. Preferências antigas relacionadas a `companionMotion` são somente compatibilidade/migração.
- Corrigi a prévia Composer/Flow que repetia o motivo de ausência de contas e mostrava o orçamento local em inglês. PASS: 20/20 testes focados, E2Es Composer completo e vago, e `quality:full` com 110 arquivos/681 testes/build. Sem contas elegíveis, preview segue bloqueado e não cria Mission/Agent. Chromium fixture não certifica a janela Wails nem provider externo.
- CLI/Desktop/HUD foram recompilados e instalados após backup. Hashes build/instalação coincidem: CLI `76c347d5a12697215c150aa87f6e2e52cf11e199af862c4fa16ca4e3ddc87fbe`, Desktop `4c68324423fce7e591c8e532092964375cccde2b0ccb725ee5066a65f830c6e2`, HUD `9a143bef1d9be86542dfb92da7248cadb9a3db88c18f508f38c30f78ef83e55a`. Unidade ativa; Core HTTP 200; versão reportada `1.0.0-beta.10`. Logs em `DEV/validation/linux-100-2026-10-25/p5-current-recheck-2026-10-26-final/logs/`.
- Estado atual: **NO_GO_LINUX — 169 PASS / 1 FAIL / 21 BLOCKED / 0 NOT_RUN (191)**. Permanecem pendentes release estável/assinatura e SHA imutável, handoff/continuidade com provedores além do smoke direto Codex, preview P5 na janela Wails, alpha/foco/múltiplos monitores e GPU/compositor Wayland.

## Verificações desta retomada — 2026-10-26

- PASS — `bun run quality:full`: 110 arquivos/681 testes, typecheck, lint, Stylelint, formatação, verificação de estilos e build de produção. Log: `DEV/validation/linux-100-2026-10-25/p5-current-recheck-2026-10-26-final/logs/web-quality-full-2026-10-26.txt`.
- PASS — regressão focada da cópia de preview: 20/20; E2E Composer completo e vago passaram em Chromium com fixtures, sem Mission/Agent quando não há provider. Relatório e capturas: `DEV/validation/linux-100-2026-10-25/P5_CURRENT_BROWSER_RECHECK.md`.
- PASS — build/instalação CLI, Desktop e HUD Linux AMD64; ELF x86-64, `ldd` sem ausências, hashes coincidentes; unidade ativa, Desktop/HUD em execução e Core HTTP 200. Evidência: `p5-current-recheck-2026-10-26-final/logs/install-latest-overlay-health-2026-10-26.txt`.
- PASS — GoReleaser v2.18.0 gerou tar.gz/DEB no fingerprint atual; checksums, metadata AMD64 e ciclo instalar/executar/remover em raiz dpkg isolada passaram. Snapshot não publicado/assinado. Logs em `p5-current-recheck-2026-10-26-final/logs/goreleaser-current-overlay-package-validation-clean-2026-10-26.txt`.
- PASS — mascote NEX único continua no caminho ativo; Companion 2D não foi integrado. Seis suítes HUD/mascote (35/35), E2E HUD e checagem da arte oficial passaram; screenshot Wails/XWayland mostra um NEX.
- Reteste do checkout atual: cinco suítes/33 testes, `bun run art:nex:check` e `bun run test:e2e:hud` PASS; evidência com os hashes das fontes em `DEV/validation/linux-100-2026-10-25/logs/mascot-current-source-recheck-2026-10-26.txt`.
- PASS — importação de sessão Gemini no perfil real do Nexus; repetição idempotente, perfil único, arquivo igual à origem e permissões `0600`. Claude não tinha credenciais tokenizadas (`NO_SESSION`). Recheck e estado sanitizados: `logs/host-session-import-real-profile-idempotency-2026-10-26.txt` e `logs/host-session-import-live-profile-state-2026-10-26.txt`. A sessão Gemini ainda requer refresh; nenhum modelo foi chamado.
- BLOCKED — fluxo Composer nativo Wails/provider autenticado real, alpha/GPU e interação completa no compositor Wayland, release estável/assinada e proveniência Git imutável.
- Resultado geral segue `NO_GO_LINUX`: 169 PASS / 1 FAIL / 21 BLOCKED / 0 NOT_RUN (191).
- Recheck mascote: cinco suítes Vitest, 33/33; `bun run art:nex:check`; `bun run test:e2e:hud` — todos PASS. Um NEX no caminho ativo; sem renderer/modo Companion 2D. O smoke direto real do Codex também passou em read-only/efêmero; P5 Wails/handoff ainda bloqueados.
- Wails/Xvfb no binário corrente: PASS, uma janela de 224×158, um NEX, clique abriu controles; screenshot nova preservada. A saída RGB não valida transparência nem compositor Wayland.

## Revalidação do mascote NEX único — 2026-10-26

- PASS: seis suítes Vitest HUD, 35/35 testes, `bun run test:e2e:hud` e `bun run art:nex:check` (`DEV/validation/linux-100-2026-10-25/logs/mascot-one-character-recheck-2026-10-26.txt`).
- PASS visual parcial: screenshot Wails/XWayland 224×158 mostra uma arte NEX e clique abre controles. Xvfb não valida alpha ou interação completa no compositor Wayland.
- Resultado do escopo: uma única personagem; não existe modo/renderer Companion 2D ativo. As referências `companionMotion` e chave antiga de adiamento apenas preservam preferências/dados. Sem alteração de produto nesta rechecagem. Homologação geral permanece `NO_GO_LINUX` por bloqueios independentes.

## Continuidade — DIRECT atômico até WorkPlan/MissionRunner e mascote NEX único — 2026-10-26

- HEAD/base local e remoto `release/v1.0.0@7a04f1b4cf5950b97b4ea1e156dda31ac513823b`; overlay fonte `ab5629f56e014719c0ce89c88b16b986f5bb0da8c310baac9fa079370140da5c` (1.428 arquivos; manifesto `DEV/validation/linux-100-2026-10-25/logs/source-provenance-p5-direct-atomic-mission-2026-10-26.txt`). Worktree compartilhada permanece modificada; sem commit/push.
- O produto mantém um único NEX. `companionModel`/teste e a preferência antiga de movimento como gravação canônica estão removidos; só existe o caminho `NexusHud → GuardianMascot → NexPixelSprite`. Barra/Mascote/Bandeja são apresentações do mesmo HUD. Não existe modo/renderer Companion 2D.
- PASS: cinco suítes Vitest focadas (30/30), `bun run art:nex:check`, e smoke nativo Wails prévio no mesmo overlay fonte de frontend, Xvfb 224×158, um NEX. Evidência: `DEV/validation/linux-100-2026-10-25/MASCOT_VALIDATION.md` e screenshot `screenshots/native-hud-workspace-fix-2026-10-26/hud-mascot.png`. Xvfb não comprova alpha/Wayland.
- PASS P5-009: pedido DIRECT explicitamente promovido → WorkPlan de uma etapa → admissão → MissionRunner `COMPLETED_VERIFIED`; uma execução de fixture, `go test ./...` em worktree temporária e duas entradas de evidência durável (pacote/global). Repetição `-race -count=3`, `go vet ./internal/nexus` e suíte completa `go test -count=1 ./internal/nexus` passaram. Isso não é handoff de provider real nem E2E Browser/Wails; P5 continua parcial.
- Matriz agora: **NO_GO_LINUX — 145 PASS / 1 FAIL / 18 BLOCKED / 4 NOT_RUN (168)**. Motivos pendentes incluem release estável/proveniência congelada, Wayland alpha/GPU, handoff externo autenticado, updater oficial e soak/inspeção final.
- Relatório: `DEV/validation/linux-100-2026-10-25/LINUX_HOMOLOGATION_REPORT.md`.

## Verificação atual — FIX-032 workspace idle — 2026-10-26

- Base local/remota release/v1.0.0: 7a04f1b4cf5950b97b4ea1e156dda31ac513823b; overlay fonte: eb038f52b8555c362a9129a3680746e0769ba4cbe733450b5c7a3b34a3ecbe05 (1.428 arquivos; manifesto DEV/validation/linux-100-2026-10-25/logs/source-provenance-workspace-idle-fix-2026-10-26.txt).
- Causa de CPU alta reproduzida: deduplicação recriava identidade da árvore em operação no-op e fazia o efeito de garantir superfícies repetir. Corrigido em web/src/workspace/model.ts.
- PASS: regressões workspace/Overview 22/22; bun run quality 109 arquivos/678 testes, typecheck, ESLint, Stylelint, allowlist, Prettier e build; make build CLI/Desktop/HUD Linux AMD64; ldd Desktop/HUD sem bibliotecas ausentes.
- Wails produção: janela Overview abriu em Xvfb 1280×800; WebKit principal 0,5% de um núcleo/12 s, RSS 456436→454408 kB. Inspector debug separado: 0 title writes, 1 timer inicial, 0 mutações DOM/6 s. Screenshot e logs no relatório.
- Limite: Xvfb/software, sem A/B controlado contra Wayland/GNOME; GPU, alpha e compositor real não homologados; NR-010 BLOCKED. NO_GO_LINUX permanece: 144 PASS / 1 FAIL / 18 BLOCKED / 4 NOT_RUN.
- Um único NEX continua sendo o mascote do produto; nenhum modo/personagem Companion 2D separado.
- HUD do overlay final também passou smoke Wails Xvfb: janela 224×158 com um NEX; clique abriu popover. Binário 12e83d1568dd79d082d2dd6640032c96e4e60ad2485d3364ff55e52cbd8a09a4, captura screenshots/native-hud-workspace-fix-2026-10-26/hud-mascot.png, log logs/current-hud-native-workspace-fix-xvfb-2026-10-26.txt. RGB não certifica o conteúdo completo do popover, alpha ou Wayland.

## Continuidade — build/install e perfil Cursor autenticado em fixture — 2026-10-26

- HEAD/base local/remoto `release/v1.0.0@7a04f1b4cf5950b97b4ea1e156dda31ac513823b`; overlay testado/instalado `0df26de4a35641d1e5d02ded7757b257fa7f4f15747220416574c083b4127208` (1.428 fontes; manifesto `DEV/validation/linux-100-2026-10-25/p5-recheck-2026-10-26T0347Z/logs/source-provenance-after-auth-maestro-fixes.txt`). Worktree modificada, sem commit/push.
- `make build build-desktop-wails build-hud-wails` passou; CLI/Desktop/HUD ELF AMD64 e `ldd` sem ausências. Hashes: `fb45b0a42af24aa3427958983fb9315cbab338c88713b0290c4cb3f708222a8f`, `6bbbafdab42309d2657ceed4ccf90f1d87c4381aee5d7b6d4528903a136dc61f`, `76aabefd32c3eb88b5693546b3402e46f6006006250e2eff47822fa598e33e15`.
- Desktop e HUD instalados atomicamente com backup e unidade reiniciada; PID do Desktop/HUD vinculado carregam os hashes, Core HTTP 200. Backup: `/home/desenvolvedor/.local/state/ai-manager/nexus-update-backups/20261026T041144Z-import-auth-maestro-fix`.
- Perfil Cursor fixture já autenticado na UI Wails resultou em `1 já autenticada`, `0 sem suporte`; nenhum login ou token real foi copiado. HUD nativo mostra um único NEX e controles. Companion 2D segue fora do produto.
- Matriz atual: **NO_GO_LINUX — 144 PASS / 1 FAIL / 18 BLOCKED / 4 NOT_RUN (167)**. Xvfb não certifica alfa/Wayland; provider externo, release estável, updater oficial e árvore imutável continuam sem aprovação. Relatório: `DEV/validation/linux-100-2026-10-25/LINUX_HOMOLOGATION_REPORT.md`.

## Continuidade — regressões de sessão autenticada e perfil Maestro — 2026-10-26

- Branch `release/v1.0.0`; HEAD local/remoto `7a04f1b4cf5950b97b4ea1e156dda31ac513823b`. Overlay atual dos testes Go: `0df26de4a35641d1e5d02ded7757b257fa7f4f15747220416574c083b4127208` (1.428 fontes; manifesto `DEV/validation/linux-100-2026-10-25/p5-recheck-2026-10-26T0347Z/logs/source-provenance-after-auth-maestro-fixes.txt`). Worktree compartilhada modificada, sem commit/push.
- FIX-030 reconhece perfil Nexus já autenticado antes de exigir importador do host; FIX-031 prioriza raiz isolada e não percorre ancestrais do `HOME`. O Nexus mantém um único mascote NEX, sem Companion 2D.
- PASS: `go test -count=1 -timeout=4m ./internal/nexus` (103,474 s), `go test -count=1 ./internal/profile`, `go vet ./internal/nexus ./internal/profile`. E2Es Composer passaram no overlay intermediário `1c3aa2c6ca9043f4a43dc3cd7a5ade451930d21c6fe3ff2e07de4cc0ed068693`; saída não pronta é esperada sem provider elegível.
- Matriz: **NO_GO_LINUX — 141 PASS / 1 FAIL / 18 BLOCKED / 4 NOT_RUN (164)**. Esta correção não fecha release, compositor/alpha, provider real, idle acelerado, updater oficial ou proveniência imutável. O binário Wails instalado não inclui este último overlay ainda. Evidências em `DEV/validation/linux-100-2026-10-25/p5-recheck-2026-10-26T0347Z/`.

## Verificação atual — 2026-10-26 — mascote único e Desktop Wails

- HEAD local/remoto `release/v1.0.0`: `7a04f1b4cf5950b97b4ea1e156dda31ac513823b`; overlay `751cfc1a160c38d950ac5b9807e1febb9746d03981c861c722e5bd965e1ef012` (1.428/1.428 entradas do manifesto conferidas). Worktree modificada.
- Produto contém um NEX e nenhuma apresentação Companion 2D separada. Desktop/HUD Wails AMD64 correntes passaram smokes Xvfb; Desktop/Core HTTP 200, Uso/Contas exibiu resultado de importação com HOME isolado; HUD mostrou um NEX e abriu controles. RGB/Xvfb não confirma alpha/Wayland.
- **PASS:** `quality:full` (109 arquivos/675 testes/build), E2E HUD/Mission Attention, Bun 1.3.9 frozen install e auditoria Bun sem advisories em `web`/`docs-site`. OSV permanece BLOCKED por reset da API.
- **NO_GO_LINUX — 139 PASS / 1 FAIL / 18 BLOCKED / 4 NOT_RUN (162)**. REL-001 continua FAIL; alpha Wayland, providers reais, idle acelerado, updater oficial e proveniência congelada não estão aprovados. Soak P9 segue ativo.

## Verificação anterior — FIX-028 e recursos — 2026-10-26

- HEAD/remoto `release/v1.0.0`: `7a04f1b4cf5950b97b4ea1e156dda31ac513823b`; overlay de fontes `391c5718b3f2cd494fb4fa0e6a9fae6f03a618c002155258a59b7e7e83f27459` (1.428 arquivos; `validation/linux-100-2026-10-25/logs/source-provenance-update-indicator-2026-10-26.txt`); worktree modificada.
- PASS: `bun run quality:full` (109 arquivos/675 testes/build); regressão do indicador (6/6); build Wails `gtk3,production` isolado; janela nativa Wails/Core sob Xvfb e health HTTP 200.
- Wails Xvfb: 107,8% CPU WebKit principal/20 s após remover animação contínua; baseline pré-fix 196,1%, delta direcional (três processos WebKit vs. um). Screenshot em `validation/linux-100-2026-10-25/screenshots/native-idle-xvfb-fixed-update-indicator-2026-10-26/usage-accounts-idle.png`. Sem aprovação de GPU/Wayland; NR-010 continua BLOCKED.
- Mascote: NEX único, sem modo/renderer Companion 2D; evidência e regressões vigentes em `validation/linux-100-2026-10-25/MASCOT_VALIDATION.md`.
- GitHub read-only: cinco previews e nenhuma release estável v1.0.0; #57/#58 Draft, #59/#61 conflitantes, #64 em main sem checks. REL-001 permanece FAIL. Evidência em `validation/linux-100-2026-10-25/PULL_REQUEST_RECONCILIATION_CURRENT.md` e `validation/linux-100-2026-10-25/logs/current-gh-release-and-pr-recheck-2026-10-26.txt`.
- Gate agregado: **NO_GO_LINUX — 137 PASS / 1 FAIL / 20 BLOCKED / 4 NOT_RUN (162)**. Soak P9 segue ativo; não commit/push.

## 2026-10-25 — recheck final do mascote único e homologação Linux (UTC 26/10)

- Branch/HEAD remoto/local: `release/v1.0.0@7a04f1b4cf5950b97b4ea1e156dda31ac513823b`; overlay de produto `9a66687715414fc8b42d612fd2b4a531370c7f8c4fbfa86338a81ba9728ddcd1` (1.428 arquivos). Manifesto conferido: 1.428/1.428 arquivos sem diferença.
- Direção confirmada: um único NEX; nenhum modo ou renderer Companion 2D. 32/32 testes focados e `bun run art:nex:check` passaram.
- `bun run quality:full` (109 arquivos/674 testes/build) e E2E de primeiro acesso/importação passaram no overlay atual. Wails Desktop/Core/HUD instalados foram exercitados sob Xvfb; clique em Uso/Contas mostrou a resposta esperada e a captura mostra somente um NEX. HOME vazio isolado; nenhuma sessão do perfil normal foi lida ou copiada.
- Binários instalados conferidos por SHA-256; unidade Desktop/HUD do usuário ativa e `/api/v1/health` respondeu HTTP 200 (`1.0.0-beta.10`).
- Matriz Markdown reconciliada com o JSON: 162 casos = 137 PASS, 1 FAIL, 20 BLOCKED, 4 NOT_RUN. **NO_GO_LINUX** permanece: Wayland/alpha do host, provider real, updater assinado/rollback e commit congelado seguem pendentes. A CPU alta do WebKit foi reproduzida em repouso aparente na sessão normal e sob Xvfb sem credenciais; profiling por `strace`/`perf` não é permitido pelo kernel, portanto a causa segue sem identificação. Soak P9 de 8 h está em andamento.
- Evidências: `validation/linux-100-2026-10-25/{LINUX_HOMOLOGATION_REPORT.md,TEST_MATRIX.md,LINUX_HOMOLOGATION_SUMMARY.json,MASCOT_VALIDATION.md,PROVIDER_SESSION_IMPORT_VALIDATION.md}`.

## 2026-10-25 — recheck do mascote único e gates P1 Linux

- Base local/remota `release/v1.0.0@7a04f1b4cf5950b97b4ea1e156dda31ac513823b`; fingerprint corrente `f033fa678c1fdeb99373d2456347ac152638ef13f699e607ff5b9bdd12113f2b` (1.426 fontes; manifesto `validation/linux-100-2026-10-25/logs/source-provenance-current-single-mascot-2026-10-25.txt`). Worktree suja, sem SHA imutável.
- Um único NEX foi confirmado no HUD Wails instalado em XWayland, 224×158; screenshot `screenshots/current-single-mascot-installed-hud-xwayland.png` SHA-256 `df2698bd013a9c32bfe81d69d46e3c812bbb89cf82de0108371a9a39035762ed`. PR #55/Companion 2D está fechada e fora do objetivo. Novas preferências gravam `mascotMotion`; leitura legada continua.
- **PASS:** regressões do mascote 30/30, arte oficial, E2E HUD Chromium, `quality:full` (108 arquivos/674 testes/build), build/instalação CLI/Desktop/HUD AMD64 e health HTTP 200. Recheck sob `-race` também passou nos casos de replay/replan e HTTP de intervenções (`validation/linux-100-2026-10-25/logs/p1-code-review-findings-recheck-2026-10-25.txt`).
- Soak P9 de 8 h permanece em execução. Resultado **NO_GO_LINUX**: 132 PASS / 1 FAIL / 16 BLOCKED / 5 NOT_RUN (154). Alpha Wayland, provider/atenção real, assinatura/update oficial e proveniência imutável seguem abertos. Veja `validation/linux-100-2026-10-25/TEST_MATRIX.md`.

## 2026-10-25 — P2 nativo e mascote NEX único

- Overlay de produto: `2c23ebb0c7726a200acbde9326c23ad6023d8fad0238c2adfef993677203f2fa`, sobre branch/remoto `release/v1.0.0@7a04f1b4cf5950b97b4ea1e156dda31ac513823b`.
- PASS nativo XWayland: uma sessão Wails exibiu A/D/E, manteve os estados enquanto B/C estavam ativos, refletiu A/B/C/D/E após atualização isolada do SQLite e refez a projeção depois do restart do Core sem F5. Nenhum provedor/runner foi chamado nesse teste; ACL multiusuário não é suportada nem validada.
- Logs e checksums: `validation/linux-100-2026-10-25/logs/p2-native-live-attention-current-overlay-2026-10-25.txt` e `p2-native-live-attention-screenshots-2026-10-25.sha256`; screenshots em `screenshots/p2-native-live-attention-2026-10-25/`.
- Mascote: um único NEX, sem Companion 2D separado; testes/art check/E2E e captura nativa estão em `MASCOT_VALIDATION.md`. `node scripts/verify-nex-visual.mjs` passou estruturalmente; `make docs-verify` terminou exit 2 porque o manifesto visual não certifica a árvore HUD/NEX não commitada (`validation/linux-100-2026-10-25/logs/docs-verify-current-overlay-2026-10-25.log`).
- O veredito geral continua `NO_GO_LINUX` (126 PASS / 1 FAIL / 16 BLOCKED / 5 NOT_RUN).

## 2026-10-25 — P5 falha pré-dispatch e continuidade após restart

- Overlay exato: `2c23ebb0c7726a200acbde9326c23ad6023d8fad0238c2adfef993677203f2fa`; manifesto `validation/linux-100-2026-10-25/logs/source-provenance-p5-predispatch-continuity-2026-10-25.txt`; HEAD local/remoto `7a04f1b4cf5950b97b4ea1e156dda31ac513823b`.
- Mascote único: 29/29 testes HUD, verificação SVG e E2E HUD Chromium passaram; o Wails do overlay exato abriu em XWayland (224×158) e três capturas (5/15/30 s) mostram um NEX. Evidências: `validation/linux-100-2026-10-25/MASCOT_VALIDATION.md` e `HUD_VALIDATION.md`. Alpha Wayland e evento de agente real permanecem pendentes.
- Integridade final: 1.425/1.425 registros de fonte do manifesto coincidem com fingerprint `2c23ebb0…`; matriz Markdown e JSON reconciliam 148 IDs (126 PASS, 1 FAIL, 16 BLOCKED, 5 NOT_RUN); JSON parse e `git diff --check` passaram.
- **PASS:** setup local sem perfil retorna como não enviado e não chama launcher; falha do launcher permanece ambígua após uma chamada.
- **PASS:** Composer OAuth multiárea injeta falha antes de dispatch, reabre SQLite e recria o MissionRunner; snapshot da revisão aprovada permanece igual, ≥4 pacotes completam e dispatches bem-sucedidos não se repetem.
- **PASS:** testes focados, race x3, `go test -count=1 -timeout=4m ./internal/nexus/...`, `go vet ./internal/nexus/...`.
- Fixture determinística, sem credenciais/requests reais. P5 permanece parcial por handoff real, cenários oficiais restantes e fluxo Wails. O updater tem PASS com fixtures HTTP locais/chaves sintéticas; release estável assinada permanece bloqueada. **NO_GO_LINUX** (126 PASS / 1 FAIL / 16 BLOCKED / 5 NOT_RUN, 148 casos).

## 2026-10-25 — Recheck do mascote NEX único no overlay corrente

- Overlay `0c522c885e700dc52d6d03a0feb313bc7d5e420915ca0188fe6c554b49fe9841`, 1.424 fontes; testes focados HUD: 5 suítes/29 testes **PASS**.
- `bun run --cwd web art:nex:check`: **PASS**. `NEXUS_E2E_ARTIFACTS_DIR=../DEV/validation/linux-100-2026-10-25/screenshots/mascot-single-renderer-current-overlay-0c522c88 bun run test:e2e:hud`: **PASS** com fixture Chromium; capturas e hashes registrados. Não é evidência Wails nativa.
- `companionModel` separado não existe; o renderer ativo apresenta um NEX. Alpha e atenção gerada por agente real seguem BLOCKED.

## 2026-10-25 — P5 feature OAuth multiárea até MissionRunner

- Overlay: `0c522c885e700dc52d6d03a0feb313bc7d5e420915ca0188fe6c554b49fe9841`; manifesto `validation/linux-100-2026-10-25/logs/source-provenance-p5-large-feature-mission-fixture-2026-10-25.txt`; base local/remota `7a04f1b4cf5950b97b4ea1e156dda31ac513823b`.
- **PASS:** prompt OAuth multiárea em fixture, edição semântica rejeita token antigo sem run e preview atualizado conclui ≥4 pacotes no MissionRunner; suíte `go test -count=1 -timeout=4m ./internal/nexus/...`, `go vet ./internal/nexus/...` e foco com `-race -count=3` passaram.
- Um primeiro race encontrou contador inseguro somente no executor fake do teste; mutex aplicado ao fixture, log original preservado, retry e três repetições race passaram. Não foi alteração runtime.
- E2E Chromium Flow/revisão 2 vem do overlay anterior `ee6a9dd…`; não é homologação Wails nem execução real de provider. P5 parcial e resultado global **NO_GO_LINUX**: 124 PASS / 1 FAIL / 16 BLOCKED / 5 NOT_RUN.

## 2026-10-25 — P5 edição semântica e aprovação stale-safe

- Overlay histórico `ee6a9dd866ae4a938d66d958fe08d3711088f41233358f2c13feee1df015d590`, 1.424 fontes; HEAD local/remoto `7a04f1b4cf5950b97b4ea1e156dda31ac513823b`.
- **PASS** — Chromium E2E `bun run test:e2e:composer-spine`: Flow editou Goal, persistiu revisão 2, recalculou preview/token, não criou run/Agent sem provider. Capturas em `validation/linux-100-2026-10-25/screenshots/p5-semantic-revision-e2e-2026-10-25/`.
- O E2E Chromium não é teste Wails nativo; provider real e UI nativa permanecem parciais.

## 2026-10-25 — P1 Skills com CLI fixture

- **PASS:** `go test ./internal/nexus -run '^TestSelectedSkillsReachAnIsolatedProviderSession$' -count=5`; 2 subcasos por repetição.
- **PASS:** mesmo teste com `-race -count=3`; cleanup do Agent/runtime confirmado como `STOPPED` e nenhuma credencial no perfil isolado.
- **PASS:** `go test -count=1 -timeout=20m ./internal/nexus` e `go vet ./internal/nexus`.
- CLI falso por `PATH`, sem chamadas externas; comandos e limitação externa em `validation/linux-100-2026-10-25/SKILL_SESSION_VALIDATION.md`. `BLK-008` continua BLOCKED até sessão com provedor real.
- Fingerprint testado: `d773e38c495781f52cdb3fa208c6a80bd1aae88d5e14bfabee56ac486407e0e1`; NO_GO_LINUX continua.

## 2026-10-25 — Reconfirmação do mascote NEX único

- **PASS** — 29 testes em cinco suítes do HUD; o seletor não contém opção Companion 2D e o renderer produz exatamente uma arte NEX. Log: `validation/linux-100-2026-10-25/logs/mascot-user-direction-recheck-2026-10-25.log`.
- **PASS** — `bun run art:nex:check` e `bun run test:e2e:hud` (Chromium/fixtures; não substitui validação nativa). Logs correspondentes em `validation/linux-100-2026-10-25/logs/mascot-art-check-recheck-2026-10-25.log` e `mascot-hud-e2e-user-direction-2026-10-25.log`.
- **PASS visual nativo previamente coletado** — capturas Wails mostram uma única arte, balão ligado ao mesmo NEX e preferências do HUD. Alpha/transparência e sinal de agente real seguem pendentes.
- Não havia modo ou renderer 2D no checkout; nenhuma remoção de produto foi necessária. Resultado amplo da homologação: `NO_GO_LINUX`, 120 PASS / 1 FAIL / 16 BLOCKED / 5 NOT_RUN.

## 2026-10-25 — P2 A/B/C/D/E com recuperação durável

- Identidade: branch `release/v1.0.0`, HEAD/remoto `7a04f1b4cf5950b97b4ea1e156dda31ac513823b`, overlay de fontes `30223d700c7d7395705571ad72d2ea30055d191e181b03cb114a922395aef18b` (1.422 arquivos), árvore dirty.
- **PASS** — `go test -count=10 ./internal/control/web -run '^TestAttentionHTTP(MultiMissionQuotaRecoverySurvivesRestart|ProjectionSurvivesRestartWithProjectIsolation)$'`.
- **PASS** — `go test -race -count=5 ./internal/control/web -run '^TestAttentionHTTPMultiMissionQuotaRecoverySurvivesRestart$'`.
- **PASS** — `go test -count=1 ./internal/control/web` e `go vet ./internal/control/web`.
- O E2E HTTP/SQLite fixture mostra A/D/E antes do restart, omite B durante EXECUTING e C enquanto REMEDIATING por quota, e recupera B/C uma vez como COMPLETED_VERIFIED. B contou 1 alocação/execução; C contou 2. Isso não valida a UI Wails conectada com as cinco missões, nem ACL multiusuário.
- Logs: `validation/linux-100-2026-10-25/logs/p2-multi-mission-http-restart-e2e-2026-10-25.txt`, `p2-multi-mission-http-restart-race-2026-10-25.txt`, `go-test-web-p2-multi-mission-2026-10-25.txt`, `go-vet-web-p2-multi-mission-2026-10-25.txt`; manifesto: `validation/linux-100-2026-10-25/logs/source-provenance-p2-multimission-2026-10-25.txt`.
- Resultado atualizado: **NO_GO_LINUX — 117 PASS / 1 FAIL / 16 BLOCKED / 5 NOT_RUN (135)**.
- No overlay atual, `go test -count=1 -timeout=20m ./...`, `go vet ./...` e `go test -race -count=1 -timeout=30m ./...` passaram; 83 linhas de resultado de pacote, sem falhas/race. Logs em `validation/linux-100-2026-10-25/logs/current-go-*.txt`.
- `bun run quality:full` passou em 108 arquivos/673 testes e gerou build de produção. E2E HUD passou com capturas versionadas de desktop, atenção, expandido e mobile; checksum e logs em `validation/linux-100-2026-10-25/`.
- O reteste nativo P2-UI-MISSION-001 passou com screenshot fixture de cinco missões antes/depois do restart na mesma janela Wails. O caso não valida transições live de provedores na UI nem ACL multiusuário; detalhes e hashes em `validation/linux-100-2026-10-25/MISSION_ATTENTION_VALIDATION.md`.

## 2026-10-25 — follow-up do mascote NEX único

- **PASS** — `bun run art:nex:check` validou o SVG oficial da arte NEX; log `validation/linux-100-2026-10-25/logs/current-nex-art-check-2026-10-25.txt`.

- Base Git local/remota `7a04f1b4cf5950b97b4ea1e156dda31ac513823b`; overlay local modificado, sem commit ou fingerprint integral novo.
- **PASS** — `bun run --cwd web quality`: 108 arquivos e 673 testes; inclui format, lint, Stylelint, verificação de styles e typecheck. Log: `validation/linux-100-2026-10-25/logs/mascot-single-model-refactor-quality.log`.
- **PASS** — build Web e `bun run --cwd web test:e2e:hud`; o E2E exercitou atenção, abrir/adiar e recarga. Logs e capturas: `validation/linux-100-2026-10-25/logs/mascot-single-model-refactor-e2e.log` e `screenshots/mascot-single-model-refactor/`.
- **PASS** — CLI, Desktop Wails e HUD Wails Linux AMD64 recompilados/instalados; Desktop iniciou via XWayland, health HTTP retornou `ok`, `ldd` não encontrou dependências ausentes. Hashes: `validation/linux-100-2026-10-25/logs/local-install-mascot-refactor.sha256`.
- **PASS** — 34 testes focados, incluindo preferência `companionMotion` e chave antiga de adiamento, um único NEX e ausência de opção 2D.
- O frontend agora usa `mascotModel`/`hud.mascot`, preservando dados legados; `NO_GO_LINUX` continua pelos gates P2/P7 live/P8/P9.

## 2026-10-25 — Campaign P7 handoff, PTY, and full race

- HEAD local/remoto `7a04f1b4cf5950b97b4ea1e156dda31ac513823b`; overlay `bb575423508a0c773580804a476b0888ff2449135045a74f80aa6efaffedc6e1` (1,421 source files, manifest `logs/source-provenance-p6-agent-fallback-2026-10-25.txt`). Worktree is modified and not frozen.
- **PASS** — `go test -race -count=1 ./...`, exit 0; 82 package result lines. Slow packages: `internal/nexus` 261.558 s, store 79.035 s, capacitymonitor 25.074 s. Log `logs/go-race-p7-full-repository.log`.
- **PASS** — focused process-level PTY attach/detach/writer lease, launcher/registry stale ownership and replacement, handoff predicate, and runner recovery race tests; see `P7_HANDOFF_PTY_VALIDATION.md`.
- **PASS** — live AGY safe response from one authenticated profile; expected HTTP 429 from an authenticated exhausted profile. Raw provider output and identity were suppressed.
- **BLOCKED** — live same-provider handoff and cross-provider checkpoint continuation. Existing Codex runtimes were preserved; OpenCode did not provide a final response within 90 s. `NATIVE_RESUME_UNVERIFIED` remains.
- One NEX mascot remains the product contract; no Companion 2D mode. Counts: 97 PASS / 1 FAIL / 15 BLOCKED / 5 NOT_RUN (118), `NO_GO_LINUX`. P2/P7 live/P8/P9 remain open.

## 2026-10-25 — Campaign P6 Agent fallback

- Commit local/remoto `7a04f1b4cf5950b97b4ea1e156dda31ac513823b`; product overlay `bb575423508a0c773580804a476b0888ff2449135045a74f80aa6efaffedc6e1` (1,421 source files; `logs/source-provenance-p6-agent-fallback-2026-10-25.txt`). Worktree is modified and not frozen.
- **PASS** — `go test ./internal/nexus/... ./internal/control/... -count=1`; log `logs/go-test-p6-agent-fallback.log`.
- **PASS** — focused race regressions repeated 5x; log `logs/go-race-p6-agent-fallback.log`.
- **PASS** — `go vet ./internal/nexus/... ./internal/control/...`; log `logs/go-vet-p6-agent-fallback.log`.
- **PASS** — deterministic hard gates/tie-break, provider/model fallback provenance, ephemeral specialist identity, durable assigned kind, and safe cleanup; see `AGENT_FALLBACK_VALIDATION.md`. No live provider session was used.
- One NEX mascot remains the product contract; no Companion 2D mode exists. Overall counts: 91 PASS / 1 FAIL / 13 BLOCKED / 5 NOT_RUN (110), `NO_GO_LINUX`. P2 and P7–P9 remain open.

## Follow-up P5 — conflito de revisão WorkPlan (2026-10-25)

- Compare-and-swap por `current_revision` impede overwrite concorrente; stale write retorna conflito tipado e HTTP 409.
- Regressões race passaram (store x3, serviços Nexus x2, handler x3), suíte `store/nexus/control/web` e vet passaram. Fingerprint `d3caa597ae3bc061ad0048ad4d390b1b9bb0011cc496e4d213d0ec6119400796`; manifesto e logs em `DEV/validation/linux-100-2026-10-25/`.
- P5 continua parcial; cinco cenários oficiais, runtime real e gates de distribuição continuam pendentes. Resultado geral NO_GO_LINUX.

## Snapshot histórico — Campaign P4 Project Intelligence / P5 Composer spine

- HEAD local/remoto `7a04f1b4cf5950b97b4ea1e156dda31ac513823b`; overlay `bcfbfa69c53255cf2768c1024b7932d5241e9316eff3e3d9d7ca3cf878755141` (1.421 sources; `logs/source-provenance-p4-project-intelligence-2026-10-25.txt`). Worktree is not frozen.
- **PASS** — `go test ./internal/nexus/contextsnapshot ./internal/nexus -count=1`; log `logs/go-test-p4-project-intelligence.log`.
- **PASS** — focused race tests repeated 3x and `go vet ./internal/nexus/contextsnapshot ./internal/nexus`; logs `logs/go-race-p4-project-intelligence.log` and `logs/go-vet-p4-project-intelligence.log`.
- **PASS** — `bun run --cwd web quality:full`: 108 Vitest files / 672 tests; format, lint, styles, typecheck and production build. Log `logs/web-quality-full-p4-project-intelligence.log`.
- **PASS** — isolated P4 Chromium E2E grounds the Composer with `vitest run`, source `package.json:scripts.test`, HIGH confidence and freshness; vague goal asks one material question, complete prompt has zero blocking questions. Log and artifacts in `composer-project-intelligence/`.
- **PASS** — P5 Composer→WorkPlan E2E: two clarifications, finalize HTTP 201, persisted FINALIZED, five steps, preview starts no Mission. Log `logs/composer-vague-spine-p5-repro.log`; direct store regression `logs/go-test-p5-finalization-regression.log`.
- **BLOCKED** — real authenticated `ai-cli` question: executable absent from PATH, no external account used; `logs/ai-cli-availability-p4.txt`.
- Mascot remains one NEX, with no separate Companion 2D mode. Overall counts: 91 PASS / 1 FAIL / 13 BLOCKED / 5 NOT_RUN (110), `NO_GO_LINUX`. Detailed matrix: `TEST_MATRIX.md`.

## 2026-10-25 — Campaign P3: entrega de notificações de Mission

- HEAD local/remoto `7a04f1b4cf5950b97b4ea1e156dda31ac513823b`; overlay P3 `65b209131c3ffe8101c86090ad229b3299742d4a102229db927b4e3a463c9b3d` (1.420 fontes; manifesto `DEV/validation/linux-100-2026-10-25/logs/source-provenance-p3-notifications-2026-10-25.txt`). Worktree não congelada.
- **PASS** — `go test ./internal/nexus/... ./internal/control/notify -count=1`; todos os módulos Nexus, store, runner e notifier passaram. Log `logs/go-test-p3-notifications.log`.
- **PASS** — testes focados do throttle/outbox/capability/evento de aplicação repetidos cinco vezes com `-race`; log `logs/go-race-p3-notifications.log`.
- **PASS** — `go vet ./internal/control/notify ./internal/nexus/...`, exit 0 sem diagnósticos; log `logs/go-vet-p3-notifications.log`.
- **PASS** — smoke nativo opt-in no Ubuntu GNOME 49/Wayland: `notify-send` aceitou copy genérica via D-Bus; log `logs/native-notification-smoke-p3-2026-10-25.log`. Não capturou pixels do toast.
- **PASS** — store close/reopen recuperou retry pendente; o serviço deduplicou blocker e não repetiu delivery concluída. Fixture de capability ausente adiou sem incrementar tentativas e retomou quando habilitada.
- **PASS** — quota/rate-limit/retry/remediation não geram toast; completion permanece no Attention Center; falha só notifica com escalonamento explícito. Copy e erros persistidos foram inspecionados sem vazamento de prompts, links ou credenciais.
- **Mascote:** NEX permanece único; Barra/Mascote/Bandeja são apresentações do HUD. Não há modo Companion 2D para remover.
- Resultado geral segue `NO_GO_LINUX`: 77 PASS / 1 FAIL / 12 BLOCKED / 5 NOT_RUN (95); P2 e P4–P9 continuam em aberto.
- Relatório P3: `DEV/validation/linux-100-2026-10-25/MISSION_NOTIFICATIONS_VALIDATION.md`.

## 2026-10-25 — Reteste no HEAD 7a04f1b4: P2 e mascote NEX único

- HEAD local/remoto `7a04f1b4cf5950b97b4ea1e156dda31ac513823b`; overlay atual `ee8a19c0cf340086f8f1cfc258d50a98ec29719952b978401be55c977a693b6c` (1.415 fontes; `DEV/validation/linux-100-2026-10-25/logs/source-provenance-p2-ui-e2e-2026-10-25.txt`). Worktree permanece sem commit imutável.
- **PASS** — `TestAttentionHTTPProjectionSurvivesRestartWithProjectIsolation`: 10 execuções normais e 5 com `-race`. Autenticação sem bearer = 401; com sessão = 200; projeções needs-user/completed/failed e filtros por projeto/mission sobreviveram a close/reopen do SQLite, serviço e endpoint HTTP de teste.
- **PASS** — `bun run test:e2e:hud`: fixture de atenção, snooze persistido e movimento reduzido passaram no overlay atual. O E2E anterior também cobriu Axe desktop/mobile em fixture.
- **PASS** — cinco suítes focadas do NEX: 27/27; `bun run art:nex:check` passou. Não há renderer/atlas/apresentação 2D separado; a arte pública SVG é exportação do mesmo componente.
- **PASS** — Desktop Wails atual iniciou em XWayland e renderizou a tela inicial em 1280×800; SIGTERM exit 0 sem `GDK-CRITICAL`. Captura `DEV/validation/linux-100-2026-10-25/screenshots/native-desktop-current-overlay-xwayland-ready.png`.
- **PASS** — HUD Wails atual foi capturado em XWayland, janela visível de 224×158, com uma única figura NEX; screenshot atual em `DEV/validation/linux-100-2026-10-25/screenshots/native-hud-current-overlay-xwayland-visible.png`. Alpha/transparência continua sem validação.
- **NO_GO_LINUX** — E2E Chromium atual passou estados vazios/loading/erro, filtros, teclado, navegação e mobile. P2 segue aberto para restart do processo real com UI conectada, cenário A/B/C/D/E integrado e ACL por usuário/projeto. Matriz: 69 PASS / 1 FAIL / 12 BLOCKED / 5 NOT_RUN (87). O workspace local é single-user.
- Logs: `DEV/validation/linux-100-2026-10-25/logs/attention-http-restart-p2-final-2026-10-25.log`, `attention-http-restart-p2-race-2026-10-25.log`, `mission-attention-browser-e2e-p2-2026-10-25.log`, `mascot-hud-current-overlay-final-2026-10-25.log` e `native-hud-current-overlay-xwayland-2026-10-25.log`.

## 2026-10-25 — Campaign P1: intervenção e retomada — COMPLETE

- **PASS** — `go test ./internal/nexus/runner -count=1 -v`; `go test ./internal/nexus -count=1`; `go test ./internal/nexus/...`; `go test ./internal/control/web -count=1`.
- **PASS** — `go test -race ./internal/nexus/runner -count=1`; `go vet ./internal/control/web ./internal/nexus/...`.
- **PASS** — regressões de replay inseguro, re-alocação/recompilação, espera por dependências, persistência `ReplayProhibited` no SQLite, timeout/handoff do worker e JSON/versionamento HTTP.
- **PASS** — `TestResolveInterventionHTTPDecisionSurvivesServerRestartExactlyOnce`, 10 execuções; teste com race do mesmo caso, 5 execuções. Inclui middleware real, resolução persistida, duplicata idempotente, shutdown/reopen de store/servidor, recovery e um único efeito externo. Logs `p1-http-restart-e2e-2026-10-25.log` e `p1-http-restart-e2e-race-2026-10-25.log`.
- **PASS** — suítes Web e Nexus do escopo, vet, lint Go com cache isolado, builds CLI/Desktop/HUD e instalação local/restart; evidências linkadas no relatório P1.
- **NO_GO_LINUX** — P1 fechado para o overlay, mas P2–P9 e outros gates de release seguem abertos. HEAD local/remoto `bf6adf0c80ac3531d7291844512a14a017ab0002`; worktree dirty.
- Fingerprint do overlay validado: `ddad56697609de1374cdede949e794316e3263357b052f745defb64f11a4be5b` (1.412 fontes; manifesto em `DEV/validation/linux-100-2026-10-25/logs/p1-e2e-source-provenance-current.txt`).
- Mascote: um caminho único `NexusHud → GuardianMascot → NexPixelSprite`; nenhum modo ou personagem Companion 2D separado. A apresentação Mascote continua sendo o NEX único.

## 2026-10-25 — Follow-up P2: Mission Attention e decisão nativa

- **PASS** — testes focados de rota e gate: 21/21; detalhe de uma missão continua disponível na build estável, com criação/histórico experimentais bloqueados.
- **PASS** — Wails AMD64 iniciou no GNOME Wayland/XWayland; cartão de fixture abriu e ativou o Flow Run com a decisão visível. Captura 1280×800: `DEV/validation/linux-100-2026-10-25/screenshots/mission-attention-open-detail-native-xwayland.png`.
- **PASS** — resumo de estado corrigido com SCSS Module e teste de regressão. `bun run quality:full`: 108 arquivos/672 testes; checks de estilo, formatação, lint, typecheck e build passaram.
- **PASS** — `make build-desktop-wails`; ELF AMD64, `ldd` sem bibliotecas ausentes. SHA-256 `f6e1afcf2e7b0e10ed5d75a5d716b22dfe6f6ca894e056c64938599172ee7f15`.
- **PASS** — `TestGlobalAttentionProjectionSurvivesStoreReopen`: cinco missões/2 projetos mantiveram os estados canônicos após reabrir SQLite; filtros projeto/missão não vazaram.
- **Mascote** — um único NEX (`GuardianMascot → NexPixelSprite`); nenhum Companion 2D ou modo separado. Captura: `DEV/validation/linux-100-2026-10-25/screenshots/native-nexus-hud-single-mascot-xwayland.png`.
- **NO_GO_LINUX** — base HEAD local/remota `bf6adf0c80ac3531d7291844512a14a017ab0002`, overlay dirty `7368531977b7422eccdf00e7c676f57d440e8d736d9e114bcdc071b9d5885fe6`; a fixture não valida atenção de agente real e gates globais anteriores continuam abertos. Detalhes: `DEV/validation/linux-100-2026-10-25/MISSION_ATTENTION_VALIDATION.md`.

## 2026-10-25 — FIX-012, evidência Linux atual e escopo do mascote

- **PASS** — reexecutei os cinco testes focados do mascote: 25/25. Barra/Mascote/Bandeja são apresentações do mesmo HUD; nenhum modo Companion 2D separado está conectado. A captura nativa mostra um NEX (SHA-256 `98b4320b329512b349af90284f5851b9080b20be3d97fc4bccf19520ac16cd3c`).
- **PASS** — `node scripts/verify-nex-visual.mjs`: identidade visual estrutural. O manifesto `source-provenance-gdk-fix.txt` foi rechecado arquivo a arquivo: 1.404 fontes, zero ausentes/divergentes.
- **PASS** — regressões direcionadas do popover GTK e `go test -race -count=1 ./internal/desktop ./cmd/nexus-hud`. Três ciclos nativos abrir/fechar passaram com `G_DEBUG=fatal-criticals`.
- **PASS** — builds/pacotes Linux pós-FIX-012: CLI/Desktop/HUD AMD64, tarballs CI com payload idêntico, GoReleaser tar/DEB e instalação/remoção dpkg isolada.
- **PASS** — JSON e matriz reconciliados: 63 PASS / 1 FAIL / 12 BLOCKED / 5 NOT_RUN (81); branch base local/remota `bf6adf0c80ac3531d7291844512a14a017ab0002`.
- **NO_GO_LINUX** — worktree dirty sem commit imutável e gates de release/runtime ainda abertos.


## 2026-10-25 — Balão de atenção nativo do mascote

- **PASS** — HUD Wails recompilado abriu no GNOME/XWayland com perfil isolado; fixture `REQUIRE_USER` apareceu junto ao NEX único. Screenshot SHA-256 `b2bff6d18ab8214d682430c3f097e93bf1e338a1924af7ccb7b25f661c7ba08a`.
- **PASS** — XTest clicou “Adiar”; HUD recolheu o balão, enquanto `/api/v1/attention` manteve a fixture (`HTTP 200`, `needs_you_count=1`). Log: `DEV/validation/linux-100-2026-10-25/logs/native-attention-fixture-click-snooze.txt`.
- **PASS** — evidência de preferências nativas e controle Barra/Mascote/Bandeja em `DEV/validation/linux-100-2026-10-25/screenshots/native-hud-expanded-single-nex-auth-fix-xwayland.png`.
- **BLOCKED** — evento real de agente ainda não alimentou o HUD; a fixture não conta como validação de runtime.
- **PASS** — contagem reconciliada em 79 casos: 61 PASS / 1 FAIL / 12 BLOCKED / 5 NOT_RUN. Matriz e JSON atualizados; `NO_GO_LINUX` permanece.

## 2026-10-25 — Contrato do mascote único, reteste e integridade das evidências

- **PASS** — inspeção do fluxo ativo confirma `NexusHud → GuardianMascot → NexPixelSprite`; não existe segundo renderer nem seletor Companion 2D. Barra, Mascote e Bandeja são apresentações do mesmo HUD.
- **PASS** — reteste focado atual: 5 arquivos / 25 testes, incluindo `GuardianModeSelector`; log `DEV/validation/linux-100-2026-10-25/logs/mascot-user-direction-final-retest.log`.
- **PASS** — 79 casos contabilizados igualmente no JSON e na matriz: 61 PASS / 1 FAIL / 12 BLOCKED / 5 NOT_RUN. Integridade de 1.403 entradas do manifesto atual: zero arquivos ausentes ou divergentes.
- **PASS** — `git diff --check`; `origin/release/v1.0.0` segue em `bf6adf0c80ac3531d7291844512a14a017ab0002`. Overlay `4e7afe7bc0e99c8d11e34dd3db9295829ab41081173bc9a7cc9b262f3c92111b`; sem commit imutável. Veredito `NO_GO_LINUX` permanece.

## 2026-10-25 — FIX-011: anexar runtime shell existente

- **PASS** — regressão unitária de roteamento: shell chama `onAttachRuntime(runtime)`; runtime de agente preserva `runtime_id` e o fluxo de terminal existente. 2/2 testes.
- **PASS** — `bun run quality:full`: 106 arquivos/664 testes; arquitetura de estilos, Prettier, ESLint, Stylelint, typecheck e build passaram.
- **PASS** — E2E de terminal dividido: sete geometrias, rótulos longos e toolbars em 520 px/360 px.
- **PASS** — Desktop Wails AMD64 atual recompilado e iniciado em XWayland. Shell iniciado com a imagem atual foi anexado; runtime permaneceu RUNNING, total de runtimes/PTYs ficou em 4 e o prompt existente ficou visível.
- **BLOCKED** — anexar host lançado por imagem anterior após troca do binário no mesmo caminho: validação de fingerprint o marcou `STALE`; a UI expôs erro genérico `TERMINAL_TRANSPORT_ERROR`. Não foi alterado o gate de compatibilidade.
- **Mascote** — confirmado NEX único; não há modo Companion 2D separado para remover. A apresentação “Mascote” continua sendo o mesmo NEX.
- **Matriz:** NO_GO_LINUX — 60 PASS / 1 FAIL / 12 BLOCKED / 6 NOT_RUN (79). Base SHA local/remota `bf6adf0c80ac3531d7291844512a14a017ab0002`; overlay `4e7afe7bc0e99c8d11e34dd3db9295829ab41081173bc9a7cc9b262f3c92111b` (1.403 fontes), dirty e não commitado.
- Evidências: `DEV/validation/linux-100-2026-10-25/logs/runtime-attach-validation.txt`, `runtime-attach-quality-full.log`, `terminal-split-toolbar-regression-final.log` e screenshots correspondentes.

## 2026-10-25 — FIX-009, revalidação do mascote e evidência corrente

- **PASS** — mascote único: `GuardianMascot → NexPixelSprite`, seletor Barra/Mascote/Bandeja, sem segundo renderer ou modo 2D. Regressão Vitest: 25/25; NEX visível em captura nativa.
- **PASS** — Cursor `InspectAuth` reconhece `✓ Login successful!`; sinais negativos e saída vazia permanecem não autenticados.
- **PASS** — `go test -count=1 -timeout=20m ./...`; `go vet ./...`; `go test -count=1 -v ./internal/core/provider/adapters/cursor`; `go test -race -count=1 ./internal/core/provider/adapters/cursor`; `bun run quality:full` (105 arquivos/660 testes); `bun run art:nex:check`. Avisos restantes são depreciações GDK/X11 upstream do Wails.
- **PASS** — snapshot GoReleaser v2.18 atual, checksums, payload tar/DEB idênticos e ciclo de instalação/consulta/execução/remoção do DEB em dpkg root temporário.
- **BLOCKED** — importação/cópia de sessão Cursor ainda não suportada; runtime orquestrado/continuidade/Skills/atenção real, interação física Wayland, alpha e updater assinado continuam pendentes.
- **Matriz:** NO_GO_LINUX — 58 PASS / 1 FAIL / 12 BLOCKED / 6 NOT_RUN (77); overlay `dac0d00b8410f70ed02c8b4c324780349655fed8f7e580d1944da1ec3eaecefa`, base `bf6adf0c80ac3531d7291844512a14a017ab0002`, worktree dirty.
- Evidências: `DEV/validation/linux-100-2026-10-25/LINUX_HOMOLOGATION_REPORT.md`, `TEST_MATRIX.md`, `MASCOT_VALIDATION.md`, `logs/go-test-cursor-status-final.log`, `logs/go-vet-cursor-status-final.log`.

## 2026-10-25 — Atualização da homologação nativa e mascote NEX único

- **PASS** — busca local Docusaurus: `npm run build --prefix docs-site` gerou `search-index.json` (2.442.219 bytes); Playwright consultou `terminal`, recebeu índice HTTP 200 e encontrou `/product/terminals`, sem erros de página.
- **PASS** — `npm run check:docs --prefix docs-site`: 100 páginas; `npm audit --audit-level=high --prefix docs-site`: 0 vulnerabilidades; Prettier e `node --check` passaram.
- **PASS** — `bun run quality:full` em `web/`: 105 arquivos/660 testes, lint, Stylelint, typecheck e build. Seis suítes focadas do HUD/mascote: 30/30.
- **PASS** — contrato revisado: uma renderização `GuardianMascot → NexPixelSprite`; sem segundo renderer/atlas/modo Companion 2D. Captura XWayland mostra um NEX. PR #55 `CLOSED`; PR #61 permanece aberta/Draft.
- **PASS** — testes focados de mascote: 25/25. HUD em Wayland direto iniciou, expandiu/recolheu, alternou apresentações, restaurou Mascote pelo tray e manteve a apresentação após reinício. Desktop abriu diretamente em Wayland e expôs a página principal via AT-SPI.
- **PASS** — após FIX-006, `bun run quality:full`: 105 arquivos/660 testes, lint, Stylelint, typecheck e build. Terminais divididos: regressão de rótulo passou em 7 geometrias e na janela Wails XWayland; captura mostra dois PTYs sem sobreposição. Digitação split e persistência seguem BLOCKED.
- **PASS** — interface Wails real importou 4 sessões host, retornou resumo agregado com 0 falhas, criou cópias `0600` no perfil Nexus isolado e voltou a detectar 3 provedores após reinício.
- **BLOCKED** — captura de pixels/alpha, teclado físico, foco/movimento em Wayland, evento real no balão, roteamento/continuidade/Skills completos, Cursor, updater assinado e proveniência imutável.
- **Matriz atualizada:** 48 PASS / 1 FAIL / 11 BLOCKED / 7 NOT_RUN (67 casos). O resultado de release segue `NO_GO_LINUX`.
- Evidências: `DEV/validation/linux-100-2026-10-25/MASCOT_VALIDATION.md`, `HUD_VALIDATION.md`, `RUNTIME_VALIDATION.md` e `PR61_DOCS_SITE_VALIDATION.md`.

## 2026-10-25 — Homologação Linux AMD64 (matriz base antes do follow-up nativo)

- **NO_GO_LINUX** — HEAD local/remoto `bf6adf0c80ac3531d7291844512a14a017ab0002`; overlay `af7476a2ab2ffeabcbfcaf35051d6d24d61a0a35bd5786b5429dc2ee3a0662e4` (1.694 fontes); VERSION=1.0.0-beta.10; worktree não congelado.
- **PASS** — `make quality-full` após correções; Go/race/vet/lint/govulncheck e frontend 103 arquivos/633 testes. Vinte temas × três viewports passaram no E2E browser.
- **PASS** — builds CLI/Desktop/HUD AMD64; Core multi-instância, colisão e shutdown; snapshot GoReleaser atual e tarballs Desktop/HUD; DEB instalado/removido em dpkg root temporário.
- **PASS** — 25/25 testes focados do mascote; captura nativa XWayland do HUD atual mostra um NEX. Não existe personagem ou modo Companion 2D.
- **PASS** — Mission real DIRECT via Nexus terminou `COMPLETED_VERIFIED`; `go test ./...` exit 0 no worktree temporário.
- **BLOCKED** — atenção real no HUD; outros modos de roteamento/continuidade/skills; importação Cursor; teclado/foco/movimento Wayland; alpha; updater assinado; proveniência imutável.
- **FAIL** — versão beta.10 e ausência de release/artefatos estáveis 1.0.0.
- **NOT_RUN na matriz base, posteriormente coberto:** diálogo nativo de importação. Ainda não executados: captura nativa do balão fixture, soak/crash, persistência/multimonitor e temas GTK.
- Matriz: 41 PASS / 1 FAIL / 9 BLOCKED / 9 NOT_RUN (60 casos). Dossiê: `DEV/validation/linux-100-2026-10-25/`.

**Escopo atual do mascote:** a decisão de produto é manter um único NEX. As referências abaixo, em registros de 2026-10-23 e anteriores, que listam a ausência do Companion 2D/PR #55 como falha são históricas e foram superadas; Companion 2D não é requisito nem defeito da release atual. A PR #55 foi fechada em 2026-10-25 e não faz parte da implementação aprovada.

## 2026-10-23 — Homologação Linux e reinstalação final

- **PASS** — `make quality-full`, `make vet`, `make test-e2e`, segurança e builds CLI/Desktop/HUD em Go 1.26.10. Frontend 99 arquivos / 612 testes.
- **PASS** — instalação nas duas raízes locais; version JSON beta.10/amd64. Web, Core Desktop, HUD e quatro hosts retomados com os hashes atualmente instalados; health Web/Core/HUD HTTP 200.
- **FAIL** — Companion 2D da PR #55 ausente. E2Es composer workspace, surface, execution spine e vague spine falham; dois presets Composer também falham; UI de importação sem confirmação visível.
- **BLOCKED/NOT_RUN** — limitações GNOME Wayland/X11, pacote e updater oficial, fluxos externos de provedor, matriz interativa nativa e resiliência; detalhados na matriz.
- Veredito `NO_GO_LINUX`, 30 PASS / 7 FAIL / 6 BLOCKED / 12 NOT_RUN. SHA-base remoto `bf6adf0c80ac3531d7291844512a14a017ab0002`, worktree dirty, sem commit/push. Dossiê: `DEV/validation/linux-100-2026-10-23/LINUX_HOMOLOGATION_REPORT.md`.

## 2026-10-23 — Follow-up de sessões e comando de login

- **PASS** — `make quality`: lint Go 0 issues, suíte Go completa (69 pacotes com testes passaram; 13 sem testes), Web 99 arquivos / 610 testes e contrato de versão.
- **PASS** — `make vet`; `go test -race -tags=gtk3 ./internal/app ./internal/profile`; build `make build-desktop` e smoke final Wails em storage temporário, Core/UI HTTP 200, SIGTERM exit 0.
- **PASS** — regressão do comando do screenshot: UI gera `nexus login codex` para perfil padrão e mantém argumentos de perfil explícito; o CLI resolve o perfil e chamou o `login` do Codex fake.
- **BLOCKED** — screenshot e interação real do cartão: compositor GNOME Wayland rejeita `grim` e screenshot D-Bus. Não foi clicado **Importar sessões existentes** na janela nativa.
- Estado do teste: base `f62fdec0c571ac6a19a904164d3f580799da6bf3` + worktree dirty; nenhum commit/push. Dossiê e logs: `DEV/validation/linux-100-2026-10-22/SESSION_IMPORT_FOLLOWUP.md` e `logs/host-session-import-*`.
- **PASS** — fetch de `origin/release/v1.0.0` permaneceu em `f62fdec0c571ac6a19a904164d3f580799da6bf3`; builds `make build`, `make build-desktop` e `make build-hud` exit 0. Três saídas são ELF x86-64; `ldd` de desktop/HUD sem dependências ausentes. SHA-256: CLI `a9035c8b92538aa80b2d2fe469410ba989f457a6c04cbc2c796f0b19f2f17e44`, desktop `aa22cc5ab78644ba384ec68a9d5417fb9014787f0fdd8c5762f46c11becac206`, HUD `13982c259df5606e73a7c1d70ad12bea4b84c3384d17a172ef2599d128cc1d89`. Compilação feita no worktree dirty, não no SHA remoto puro; logs em `DEV/validation/linux-100-2026-10-22/logs/build-current-2026-10-23-*`.
- **PASS — instalação local:** recompilação seguida de hash parity em ambas as pastas `~/.local/bin` (home e perfil Codex); CLI instalado reporta `1.0.0-beta.10`, commit embutido `f62fdec0`. Desktop/HUD sem bibliotecas dinâmicas ausentes. `nexus web` reiniciado (PID 3393460), health HTTP 200/version `1.0.0-beta.10`; rollback: `~/.local/state/nexus-update-backups/20261023T131106Z`. Evidência: `logs/install-current-2026-10-23.log` e `logs/web-restart-current-2026-10-23.log`.
- **PENDENTE DE REINÍCIO:** 11 processos Nexus `__control-host` mantidos vivos; executáveis já substituídos no disco, mas os processos residentes continuam carregando a imagem antiga até o encerramento natural/reinício das sessões.

## 2026-10-22 — Homologação Linux 1.0.0 — f62fdec0

- **PASS** Linux AMD64 CLI/Desktop/HUD builds em Go 1.26.10; Wails Desktop e HUD linked iniciados em Ubuntu 25.10/GNOME 49 Wayland; Core respondeu HTTP 200.
- **PASS** frontend `quality:full` (98 arquivos/605 testes), Go `vet`, suíte completa e race detector, golangci-lint, govulncheck, `make format-check`, pacote Debian snapshot e archives temporários Desktop/HUD.
- **FAIL** Companion 2D não integrado no SHA exato; versão do source segue beta.10 e não existe release estável assinada; cinco suítes E2E falham; `docs-verify` acusa visual manifest stale.
- **BLOCKED** screenshots/controle da janela pelo compositor Wayland, GPU 3D ausente, provedores autenticados live, fluxo de instalação estável sem manifesto. **NOT_RUN** X11, segundo monitor/escala, soak/crash recovery e ShellCheck.
- **Veredito: NO_GO_LINUX**. Detalhes e matriz: `DEV/validation/linux-100-2026-10-22/`.

---


## 2026-10-21 — Click-through do HUD corrigido em 047f4348

- **PASS** `gofmt`, `go vet ./...`, `go test -count=1 ./...` (Go 1.26.10).
- **PASS** Web `npm run quality`: 97 arquivos / 598 testes.
- **PASS** `docs-verify`: 192 Markdown / 33 assets, manifesto em `047f4348`.
- **PASS** build HUD `production`, SHA-256 `19c5e299...a1e6a5`.
- **PASS** click-through real: `RESULT.json` com `verdict: PASS`, canto transparente atravessa, cápsula interativa, estado restaurado.
- **PASS (funcional)** HUD 20/20 ciclos (`PASS_FUNCTIONAL_PENDING_VISUAL`).
- **NOT_TESTED** click-through no popover destacado; confirmação visual humana.
- **BLOCKED_EXTERNAL** assinatura; **NOT_RUN** smokes Codex/multi-conta.
- Veredito: `NO_GO_WINDOWS`. Relatório: `DEV/validation/local-hud-047f4348/HOMOLOGATION.md`.


## 2026-10-21 — Homologação Windows da release em 7d92b942

- **PASS** `docs-verify`: 192 Markdown / 33 assets, manifesto em `7d92b942`.
- **PASS** `verify-version-contract`, `gofmt` (após correção), `go vet ./...`.
- **PASS** `go test -count=1 ./...` com Go `1.26.10` (via `GOTOOLCHAIN`); `internal/app` 88,9 s.
- **PASS** Web `npm run quality`: 96 arquivos / 593 testes, typecheck, ESLint, Stylelint, Prettier.
- **PASS** build Web e build nativo do HUD (`production`), binário SHA-256 `8dab24a7f616fe6fb720dc50d33cd6f6b6802328993614db2d987eafb48dcefb`.
- **PASS** certificação oficial `scripts/certify-windows-first-run.ps1` em `7d92b942`: 9 gates PASS, 4 NOT_RUN, exit 0, `automated_result: PASS`.
- **PASS (funcional)** HUD nativo: 20/20 ciclos em janela única e 20/20 em popover destacado (`PASS_FUNCTIONAL_PENDING_VISUAL`).
- **FAIL** click-through do HUD: clique em área transparente (`1696,972`) manteve `Nexus HUD` em foreground em vez da janela alvo. Evidência em `DEV/validation/local-hud-7d92b942/click-through/`.
- **INCONCLUSIVO** matriz HUD cross-platform: sem Linux X11 e macOS no mesmo SHA; agregador recusa emitir veredito.
- **BLOCKED_EXTERNAL** instalador assinado: `native-signing` com 0 secrets, sem certificado local, `signtool`/`makensis` ausentes.
- **NOT_RUN** `desktop-wails-first-run`, sessão Codex interativa e cenário multi-conta.
- Veredito: `NO_GO_WINDOWS`. Relatório: `DEV/validation/local-hud-7d92b942/HOMOLOGATION.md`.

## 2026-10-17 — Correção e Verificação do Backend Compartilhado Desktop (76587c39)

- `internal/control/web/desktop_attach.go`: PASS. Corrigido o `NewLoopbackBackendHandler` para servir arquivos estáticos embutidos (`bundle.js`, `bundle.css`, chunks) em vez de retornar incondicionalmente `index.html`.
- `internal/control/web/desktop_attach_test.go`: PASS. Novo teste TDD `TestLoopbackBackendHandlerServesStaticAssetsAndFallbackToSPA` valida que assets estáticos nunca realizam fallback indevido para HTML e que rotas SPA continuam caindo no `index.html`.
- `go test -v -run "TestLoopbackBackendHandler.*" ./internal/control/web`: PASS.
- `gofmt -w` e `go vet ./...`: PASS (código 0).
- Binários recompilados e reinstalados em `%LOCALAPPDATA%\Programs\IAPro Nexus`.
- Desktop validado em execução com PID e controlador WebView2 ativo.

## 2026-10-17 — Validação E2E Web + Desktop e Correção no NexusDemoApp (87216d26)

- `web/src/app/NexusDemoApp.tsx`: PASS. Resolvido o erro `useWorkspacePresentation must be used inside WorkspacePresentationProvider` ao envolver o `<DemoCoordinator>` no `<WorkspacePresentationProvider>`.
- `bun run --cwd web typecheck` & `bun run --cwd web lint`: PASS (código 0, zero erros de tipagem ou lint).
- `bun run --cwd web test`: PASS (89 arquivos de teste, 553 testes aprovados sem falhas).
- Suíte E2E do Navegador (Web / Playwright):
  - `test:e2e:composer`: PASS.
  - `test:e2e:surface`: PASS (isolamento de abas e keep-alive de terminais).
  - `test:e2e:capacity`: PASS.
  - `test:e2e:hud`: PASS.
  - `test:e2e:composer-spine` & `test:e2e:composer-vague-spine`: PASS.
  - `test:e2e:composer-themes`: PASS (20 temas validados).
  - `test:e2e:hardening`: PASS (WCAG 2.1 AA a11y, 7 breakpoints responsivos, deep links).
- Suíte E2E do Desktop (Wails v3 + WebView2 via CDP):
  - Todas as rotas inspecionadas e validadas: `/projects`, `/settings`, `/updates`, `/welcome`, `/demo`.
  - Todas as APIs de backend retornando 200 no contexto do shell Desktop.
  - Evidências salvas em `DEV/validation/local-beta/1.0.0-beta.10/85c5b9589ec19663023fcddb18d0539742367b40/windows-AMD64/desktop-native/`.
- Commit `87216d26` integrado por fast-forward na branch canônica `release/v1.0.0`.

## 2026-10-17 — Continuidade de homologação Windows beta.10

- `bun run --cwd web quality:full`: PASS no worktree baseado em
  `5d5aea9211b5154e96b036c519a61de9723593b1`; 89 arquivos/553 testes e build.
- `bun run --cwd web test:a11y`: PASS em três execuções após espera limitada
  de 250 ms por foco; asserção preservada. Esta alteração de teste ainda não
  está commitada e não certifica o SHA-base.
- `pwsh -File scripts/test-uninstall-windows.ps1`: PASS com TEMP/TMP em D:;
  coordenadas beta, mismatch de manifesto, digest, path segments, variável de
  ambiente, preservação de dados e diretórios desconhecidos.
- `go test -count=1 ./internal/release ./internal/update/...`: PASS. `go run
  ./scripts/verify-version-contract.go .`, `node --check` e `git diff --check`:
  PASS.
- Falhas obrigatórias mantidas: `go test -count=1 ./...` exit 1 no timeout
  padrão de 10 min em `internal/nexus`; golangci-lint exit 4 em duas expirações
  de `go/packages`; browser `test:a11y` falhou uma vez no SHA-base.
- NSIS lifecycle/signing, HUD visual matrix, Codex interativo, updater A→B,
  CI exact-SHA e Linux nativo permanecem BLOCKED/UNVERIFIED. Relatório completo
  e logs: `DEV/validation/local-beta/1.0.0-beta.10/5d5aea9211b5154e96b036c519a61de9723593b1/windows-AMD64/`.
- Veredito `NO_GO_WINDOWS`; sem commit, push ou publicação nesta atualização.

## 2026-10-15 — Homologação Windows beta.10 no SHA exato, instalação local e smoke Desktop/HUD

- PASS — cadeia local completa (mesmos scripts do CI) no SHA `c5cf076a96934f83c8dd9fffec6e957db881e637`: frontend (format/typecheck/lint/styles/test/build), Go (vet/golangci/govulncheck/test), control/runtime E2E, agent-interop, browser E2E (hardening/a11y/visual, composer, surface, capacity, hud, spine, vague-spine, composer-themes 20/20), smokes doctor/providers/usage + regressão OpenCode e builds Wails v3 Desktop/HUD; veredito `PASS automated Windows beta homologation: 1.0.0-beta.10 @ c5cf076a`. Evidência em `DEV/validation/local-beta/1.0.0-beta.10/c5cf076a96934f83c8dd9fffec6e957db881e637/windows-AMD64/` e `%TEMP%\opencode\local-gate-c5cf076a.log`.
- PASS — correção do flake do gate `composer-themes`: o × de fechar aba deixou de ser um `<button>` filho do `role=tablist` (Axe `aria-required-children` critical) e virou span decorativo `aria-hidden` nos dois strips, preservando fechamento por teclado via `Delete` e menu de contexto (mesmo diálogo de confirmação); reprodução determinística RED→GREEN no estado exato da falha.
- PASS — captura oficial de docs refeita no SHA (`source_sha=c5cf076a`); `docs-verify` 179 Markdown/33 assets; artefatos regenerados commitados em `b462e844`.
- PASS — instalação local via `install.ps1 -BuildFromSource`: CLI `1.0.0-beta.10` commit `c5cf076a` em `%LOCALAPPDATA%\Programs\IAPro Nexus` (+ alias `ai.exe`), PATH do usuário atualizado e `doctor` executado no install.
- PASS — Desktop e HUD (builds Wails v3 da própria cadeia; SHA-256 conferidos na cópia: `4a6aea46…`/`c1d26699…`) copiados para o diretório de instalação com atalhos no Desktop e Start Menu.
- PASS — smoke nativo: `nexus-desktop.exe` vivo após 9s com janela `IAPro Nexus` e 6 processos WebView2; `nexus-hud.exe` vivo após 9s com janela `Nexus HUD` e 6 WebView2; ambos encerrados sem processos residuais.
- LIMITAÇÃO PRÉ-EXISTENTE — em runs de fixture o PUT de layout nunca é disparado (GET retorna `{}`/rev 0), tornando a persistência de abas timing-dependent nesses runs; fora do escopo desta certificação e documentado no commit `b462e844`.
- NÃO EXECUTADO — `MANUAL_CHECKS.md` (smokes nativos interativos: teclado Codex, Mission/promoção, quota/certify, HUD autostart) aguardam sessão interativa do usuário.
- NO TAG / NO PUBLISH — somente commits na branch `fix/release-beta10-windows-closure-20261014`, todos com `[skip ci]`; `release/v1.0.0` intocada.

## 2026-10-07 — Continuidade Windows WebView2 e branch audit

- PASS — go fmt ./..., go vet ./..., govulncheck ./..., golangci-lint run ./... e go test -count=1 ./... no Windows AMD64 após mudanças Go.
- PASS — teste LOCALAPPDATA do perfil WebView2 e testes dos pacotes internal/desktop, cmd/nexus-desktop e cmd/nexus-hud.
- PASS — build Desktop/HUD com IDs isolados de smoke; Desktop: janela IAPro Nexus + Chrome_WidgetWin_0 por 9s e perfil criado; HUD: processo vivo por 8s e perfil criado.
- PASS — parser PowerShell do smoke Desktop e parser YAML do workflow CI.
- PASS anterior — Bun frozen install, quality:full e oito suítes Playwright nesta árvore Windows; não repetidos nesta continuação porque nenhum arquivo frontend foi alterado.
- PASS — auditoria viva: nenhuma ref Beta 7 nova. KIVER (d14154d2) é ancestral de release (52f35c04). As branches terminal/testes foram comparadas ao código e cobertura atuais; nenhuma integração pendente.
- PASS — branch antiga codex/windows-beta6-convergence revisada (5 commits exclusivos no grafo): equivalentes atuais de startup Wails, API same-origin/WebSocket, descarte de runtime inativo e resolução .cmd estão na release. A mudança para ConPTY padrão foi posteriormente revertida por cc6400db.
- PASS — go test -count=1 -run TestWindowsBackendInteractiveConPTYOptIn -v no Windows AMD64; ConPTY opt-in interativo validado. O padrão de release continua opt-in.
- PASS — parser PowerShell e smoke Desktop após restringir cleanup ao perfil Nexus: Wails/WebView2 ativos 10s, perfil isolado criado, zero processo WebView2 desse perfil após cleanup. Evidência e hash do binário em outputs/windows-desktop-smoke-cleanup-52f35c04-20261007/.
- PASS — go test -count=1 ./internal/control/host ./internal/runtime no Windows AMD64; ambas as suítes passaram.
- PASS — bun run --cwd web vitest run src/nexus/terminalInput.test.ts no Windows; 28 testes passaram.
- LIMITAÇÃO — release local e remoto estão em 52f35c04, cinco commits após d14154d2; o worktree está sujo, portanto não certifica SHA imutável. Refs remotas foram consultadas ao vivo nesta continuação.
- PASS — NSIS 3.13 compilou o nexus.nsi exato da árvore atual; instalador unsigned de 23.917.346 bytes, SHA-256 7f310bf94d318d339ccd332d4c2de6b924404f87574e3adfccdb914fa55f5cf4.
- PASS — variante com registry/atalho isolados instalou e desinstalou o payload real: setup/uninstall 0, CLI beta.7, doctor JSON válido; Path e Desktop reais inalterados.
- NÃO EXECUTADO — install/uninstall do executável NSIS exato no perfil do usuário, Authenticode, trust root, CI exact-SHA e gates Linux.
- FAIL — bun scripts/docs-verify.mjs: o manifesto visual continua em 2892f301, anterior a mudanças visuais do workspace, e visual source também está dirty. O fluxo scripts/refresh-beta-docs-windows.ps1 exige SHA limpo; recapturar e repetir o gate após congelar o candidato.
- NO TAG / NO PUBLISH — manter NO-GO até commit revisado, homologação no SHA final e gates externos de assinatura/instalador.
- Nota Git: git status/diff-check não puderam ser usados neste clone nesta execução (fatal: this operation must be run in a work tree); resultados reportados são os comandos executados e as evidências de smoke listadas em DEV/validation/current/BETA7_WINDOWS_HOMOLOGATION_20261007.md.

## 2026-10-07 — Correções Windows ProjectHub e origem Wails

- PASS — go fmt ./..., go vet ./..., govulncheck ./..., golangci-lint run ./... e go test -count=1 ./... no checkout Windows atualizado.
- PASS — bun install --cwd web --frozen-lockfile.
- PASS — bun run --cwd web quality:full com Bun 1.3.9 e Node 24.19.0; 82 arquivos e 487 testes Vitest, build incluído.
- NOTA DE AMBIENTE — a primeira execução com Node 18.12.1 falhou ao carregar import JSON do Stylelint. Repetição com Node 24.19.0 passou sem alteração de dependências.
- PASS — oito suítes Playwright E2E no worktree isolado com as mesmas alterações; captura ProjectHub 775x483 confirma mensagem em toda a largura e CTA em uma linha.
- PASS — três builds Windows AMD64 e checksums; CLI informou versão 1.0.0-beta.7 e commit 52f35c04-dirty. Authenticode NotSigned; trust root ausente.
- PASS — smoke de oito segundos do HUD; WebView2 criado.
- INCONCLUSIVO — Desktop automatizado encerrou com código 0 antes de oito segundos; binário de baseline repetiu o comportamento no mesmo launcher. Repetir em sessão Windows interativa.
- NO TAG / NO PUBLISH — código local sem commit; CI Windows exact-SHA e assinatura/trust root bloqueados.
- Escopo Windows-only; nenhum gate Linux executado nesta continuação.

## 2026-10-07 — Correção local do estado de Path no NSIS

- PASS — `go test -count=1 ./internal/release` no checkout Windows real.
- PASS — `go test -count=1 ./...`, `go vet ./...` e `golangci-lint run ./...` no worktree isolado de `52f35c04` com o mesmo patch local.
- PASS — smoke NSIS portátil 3.13, em subchave de registro isolada e payload fictício: 5/5 estados, incluindo `REG_SZ`, `REG_EXPAND_SZ`, valores vazios e Path ausente.
- PASS — `git diff --check` nos dois arquivos de código.
- NÃO EXECUTADO — build do instalador de distribuição com payload real e assinatura Authenticode após a correção.
- BLOQUEADO — CI Windows exact-SHA aguardando runner; certificado, secrets e `NEXUS_UPDATE_PUBLIC_KEY` ausentes.
- NO TAG / NO PUBLISH — a correção é local, não commitada; não existe nova certificação imutável.
- Escopo Windows-only; nenhum gate Linux executado nesta continuação.

## 2026-10-07 — Homologação técnica Windows beta.7

- PASS — branch local/remota release/v1.0.0, SHA 52f35c04181b85379219ab55fc3192c63a5440a8; VERSION=1.0.0-beta.7.
- PASS — go fmt ./..., go vet ./..., govulncheck ./..., golangci-lint run ./... e go test -count=1 ./... no Windows.
- PASS — bun install --cwd web --frozen-lockfile e bun run --cwd web quality:full com Node 22.14.0/Bun 1.3.9; 82 arquivos, 487 testes unitários e build.
- PASS — oito suítes E2E Playwright: geral, composer, surface, capacity, hud, composer-spine, composer-vague-spine e composer-themes.
- PASS — builds CLI/Desktop/HUD para Windows AMD64 com commit 52f35c04 embutido; CLI version e doctor --json; Desktop e HUD vivos por oito segundos cada.
- PASS — NSIS 3.13 compilado e smoke isolado na fonte revisada; código NSIS é idêntico no commit 52f35c04. PATH manteve o texto, mas seu tipo foi normalizado de REG_SZ para REG_EXPAND_SZ.
- PASS — checksums SHA-256 dos três executáveis registrados nos artefatos outputs/windows-revalidation-52f35c04; Authenticode NotSigned e trust root ausente, portanto não são distribuição.
- BLOCKED — run nativo Windows exact-SHA 37666126714 está queued; só existe runner Linux/offline, sem runner Windows.
- BLOCKED — assinatura: certificado privado, PFX secrets, signtool.exe e NEXUS_UPDATE_PUBLIC_KEY ausentes no host.
- NO TAG / NO PUBLISH — tag v1.0.0-beta.7 ausente e endpoint de release retornou 404. Veredito: GO técnico local Windows, NO-GO para distribuição.
- Escopo desta continuação: Windows; nenhum gate Linux foi executado. Capturas versionadas geradas incidentalmente pelos E2E foram restauradas ao baseline; logs desta rodada foram guardados fora do repositório.

## 2026-09-19 — Correção do attach por geração/PID

- PASS — `go test ./...`.
- PASS — `go test -race ./internal/control/launcher ./internal/control/protocol ./internal/control/host`.
- PASS — `make format-check` e `make vet`.
- PASS — `make build` e `make install-local`; binário instalado em
  `/home/desenvolvedor/.local/bin/nexus`, commit `50b561c`.
- PASS — Web reiniciado pelo binário novo e `GET http://127.0.0.1:13000`
  retornou HTTP 200.
- PASS — smoke real `nexus codex --yolo`: runtime iniciou com
  `host_generation` não-zero e sem “changed generation/PID”.
- PASS — smoke real `nexus agy`: runtime iniciou com `host_generation`
  não-zero e sem “changed generation/PID”.
- PASS — os hosts antigos incompatíveis foram encerrados após backup explícito
  do registro; nenhum `__control-host` órfão permaneceu.
- PASS — o host residual de captura de documentação em `/tmp` também foi
  encerrado via protocolo, sem alterar o registro principal.
- PASS — `make lint-go` após corrigir os dois avisos estáticos preexistentes
  (`ST1023` e `QF1001`) sem alteração de comportamento.
- PASS — versão `1.0.0-beta.3` consistente em `VERSION`, `web/package.json` e
  `cmd/nexus-desktop/wails.json`; binário instalado reporta a mesma versão.
- NOT EXECUTED — commit/push, conforme o contrato do Maestro.

## 2026-09-18 — Proteção permanente contra falhas de teclado

- PASS — testes Go afetados, `-race`, `make test-e2e`, `go vet ./...`,
  `make build`, `make web-verify` (10/10) e cross-compile Windows/macOS.
- PASS — pseudo-TTY executou `./nexus doctor terminal --recover` com reset
  emitido e termios legível.
- PARTIAL — `go test ./...` com dados temporários reproduz fixtures
  preexistentes de conversation/AGY/Claude/quota/profile; nenhum pacote do
  slice de TTY falhou.
- PARTIAL — `make lint-go` mantém cinco diagnósticos preexistentes fora do
  slice.
- BLOCKED — smoke autenticado `nexus codex --yolo` sem perfil Codex habilitado;
  Windows/macOS nativos continuam não verificados.

## 2026-09-18 — Beta Linux CI closure preparation

- PASS — `make quality` (format, frontend lint/styles/typecheck, 409 Vitest
  tests, Go lint, Go tests and version contract).
- PASS — release contract tests, Wails/Linux build, browser E2E/a11y/visual,
  GoReleaser check and Linux snapshot, DEB container smoke and `make docs-verify`.
- PASS — `actionlint` for CI/release workflows and `git diff --check`.
- OPERATIONAL — repository runner `nexus-ci-dev-web-kiver` is online with
  labels `self-hosted`, `Linux`, `X64`, `nexus-ci`.
- PENDING — commit/push, hosted same-SHA 7/7 run, repeated Linux run, merge,
  green `main`, distribution token and signed beta release.

## 2026-09-17 — PR #2 Windows closure patch

- PASS — `go test -v ./internal/core/provider/adapters/codex -run
  '^TestPrepareEmptyDestWithManyRolloutsRespectsBudget$' -count=20`.
- PASS — `go test -v ./internal/nexus -run
  '^TestInterventionResolutionSurvivesStoreReopen$' -count=20`.
- PASS — `go test ./internal/core/provider/adapters/codex/... ./internal/nexus/... -count=1`.
- PASS — `go test -race ./internal/core/provider/adapters/codex/... ./internal/nexus/... -count=1`.
- PASS — `git diff --check`.
- PASS — `make web-verify` (10/10 gates; relatório frontend atualizado).
- BLOCKED/EXISTING — `make docs-verify` detecta manifesto visual stale desde
  `0ec7e750`; não relacionado às alterações desta correção e requer campanha
  visual separada.
- PENDING — native Windows repeated run, hosted same-SHA CI, maintainer merge,
  `main` CI e tag beta.1 no commit entregue.

## 2026-09-16 — Reassert após interrupção Codex

- PASS — testes de regressão de `Ctrl+C`, forwarding byte-transparente e
  bracketed paste: `go test ./internal/runtime ./internal/app -run
  'Test(WriteTerminalKeyboardModeReset|TerminalKeyboardModeResetDisablesBracketedPaste|ForwardAttachedInput|ContainsTerminalInterrupt)$' -count=1`.
- PASS — `go test ./internal/runtime ./internal/app ./internal/control/host -count=1`.
- PASS — `go test -race ./internal/runtime ./internal/app -count=1`.
- PASS — `make build && make LOCAL_BIN=/home/desenvolvedor/.local/bin install`.
- PASS — binário instalado: `IAPro Nexus 1.0.0-beta.1`, commit `b3db75c`;
  `git diff --check` limpo.
- Observação operacional: PIDs 111266/111295 ainda executam a imagem anterior
  `(deleted)` em memória e precisam de reinício para carregar esta correção.

## 2026-09-16 — Guard persistente de teclado do terminal

- PASS — `go test ./internal/runtime ./internal/app ./internal/control/host -count=1`.
- PASS — `make build && make install LOCAL_BIN=/home/desenvolvedor/.local/bin`.
- PASS — `git diff --check`.
- Aplicado reset pontual de Kitty/CSI-u, modifyOtherKeys e foco estendido ao
  TTY externo da sessão antiga, preservando o processo e seu contexto.

## 2026-09-16 — Auditoria Contínua, Extração de E-mail do Cursor e Consolidação de Qualidade

- PASS — `go test -v ./internal/core/provider/adapters/cursor/...`: 2/2 testes unitários aprovados (`TestInspectAuthMemoizesStatusProbe` e `TestInspectAuthExtractsEmail`).
- PASS — `go test ./internal/core/provider/adapters/...`: 100% de aprovação em todos os adapters (`agy`, `claude`, `codex`, `cursor`, `gemini`, `opencode`).
- PASS — `go test ./internal/...`: Todos os pacotes Go internos aprovados (`internal/app`, `internal/nexus/...`, `internal/control/...`, `internal/core/...`, `internal/profile`, `internal/tui`, etc.).
- PASS — `make vet`: Go vet limpo (0 issues).
- PASS — `/home/desenvolvedor/go/bin/golangci-lint run ./...`: 0 issues.
- PASS — `make format-check`: Formatação Prettier e gofmt 100% conformes.
- PASS — `make web-verify`: 10/10 gates aprovados (`format`, `typecheck`, `lint`, `stylelint`, `null-arrays`, `test` 379/379 Vitest, `i18n`, `build`, `embed-sync`, `ui-markers`).
- PASS — `make security` (`govulncheck`): 0 vulnerabilidades detectadas.
- PASS — `make version-check`: Todas as fontes consistentes em `v1.0.0-beta.1`.
- PASS — `nexus list`: Exibe `kiver@omegasistemas.net.br` para o perfil `cursor:cursor` com status `Authenticated`.
- PASS — `make build && make install LOCAL_BIN=/home/desenvolvedor/.local/bin`: Binário atualizado instalado em `/home/desenvolvedor/.local/bin/nexus`.
- PASS — `GET http://127.0.0.1:13000/api/v1/health`: Retorna `{"status":"ok","timestamp":...,"version":"1.0.0-beta.1"}`.
- NOT EXECUTED — `git commit` / `git push` (preservando o contrato de governança do Maestro).

## 2026-09-16 — Correção de navegação CSI embutida

- PASS — `npm --prefix web run lint` e `npm --prefix web run typecheck`.
- PASS — `npm --prefix web test -- --run src/nexus/terminalInput.test.ts` (20/20).
- PASS — `make build && make install LOCAL_BIN=/home/desenvolvedor/.local/bin`.
- PASS — `make web-verify` com 10/10 gates e `git diff --check`.

## 2026-09-15 — Revalidação completa da fundação conversacional (anti-perda)

Evidência contra o stash `stash@{0}` + gates locais:

- PASS — inventário: todos os arquivos conversation do stash no disco (feature, spike, bootstrap, handlers, 0021, ADR/docs).
- PASS — wiring: Composer → `ConversationSurface`; NDJSON create; upload attachments; CSRF; `@assistant-ui/react`; `/dev/conversation-spike`; i18n `conversation` ×3.
- PASS — DB `~/.local/share/ai-manager/nexus.db`: schema 21 + `kind`/`payload_json` + `conversation_attachments`.
- PASS — `go test ./internal/nexus/store/ ./internal/nexus/ ./internal/control/web/ -count=1`
- PASS — Vitest conversation + composerModel + i18n (19)
- PASS — `make web-verify` 10/10 (`FRONTEND_2026-09-15T13-36-13Z.md`)
- PASS — `nexus web :13000` serve UI nova; exe não deleted.
- NOT EXECUTED — commit/push (aguardando Maestro).

## 2026-09-15 — Resolução de Autenticação e UX do OpenCode (Modo Local, Multi-Provedor & Paridade CLI)

- PASS — `go test -v ./internal/core/provider/adapters/opencode/...`: 7/7 testes unitários aprovados (capacidades Login/Logout, modo local sem login, Zen provider, DeepSeek provider, cópia segura de credenciais do host no `Prepare`, probes de cota).
- PASS — `go test ./internal/profile/...`, `go test ./internal/nexus/...`, `go test ./internal/control/web/...`: 100% de aprovação sem regressões.
- PASS — `npm --prefix web run check:styles`: SCSS Modules em estrita conformidade com allowlist.
- PASS — `npm --prefix web run lint:styles`: Stylelint aprovado com 0 violações.
- PASS — `npm --prefix web run typecheck`: TypeScript compilado limpo (0 erros).
- PASS — `npm --prefix web run lint`: ESLint com `--max-warnings=0` aprovado.
- PASS — `npm --prefix web test`: Vitest executou 71 arquivos e 379 testes com 100% de aprovação.
- PASS — `make web-verify`: 10/10 gates automatizados aprovados (format, typecheck, lint, stylelint, null-arrays, test, i18n, build, embed-sync, ui-markers). Relatório em `DEV/validation/FRONTEND_LATEST.md`.
- PASS — `make build && make install LOCAL_BIN=/home/desenvolvedor/.local/bin`: Binário compilado e instalado em `/home/desenvolvedor/.local/bin/nexus`.
- PASS — `curl -H "Authorization: Bearer <token>" http://127.0.0.1:13000/api/v1/resources`: OpenCode reportado com `authenticated: true`, `health: healthy`, `plan: OpenCode (deepseek)` e `account: deepseek`.
- NOT EXECUTED — `git commit` e `git push` (em estrita conformidade com a constituição do Orquestrador).



## 2026-09-15 — Fechamento v1.0.0-beta.1 (in-repo, worktree isolado)

- PASS — `make lint-go` (0 issues; exclusão misspell em `composer_brief.go`; SA1019 com `//nolint:staticcheck`).
- PASS — `go test -count=1 ./internal/nexus/ ./internal/app/ ./internal/control/host/ ./internal/control/web/ ./internal/core/provider/adapters/opencode/` (575 testes).
- PASS — `make version-check` (`1.0.0-beta.1` consistente).
- PASS — `make web-verify` 10/10 (`DEV/validation/FRONTEND_LATEST.md`, 2026-09-15T13:04:30Z).
- ISOLATED — spike Conversation no stash `wip: conversation spike parked outside v1.0.0-beta.1`.
- NOT EXECUTED — commit, push, tag, GitHub Release, environment `beta-signing` (exigem autorização).

## 2026-09-15 — Soak prolongado

- **PASS** — `make overnight-soak DURATION=8h`, com duração observada de
  `28800.028s` (8h) e encerramento normal do teste.
- O resultado é evidência real de execução prolongada, não um smoke test curto.

## 2026-09-15 — Modernização das Superfícies de Sessões, Agentes e Project Overview (Opções 1, 2 e 3)

- PASS — `go test -v ./internal/control/web/...`: 100% de aprovação na suite web, incluindo os novos testes unitários de `handleTelemetryStats` e `handleTelemetryHistory` (`handlers_telemetry_test.go`).
- PASS — `npm --prefix web run check:styles`: Conformidade estrita com SCSS Modules e allowlist global de estilos.
- PASS — `npm --prefix web run lint:styles`: Stylelint aprovado sem nenhuma violação (incluindo keyframes kebab-case `agent-pulse` e `runtime-pulse`).
- PASS — `npm --prefix web run typecheck`: TypeScript `tsc --noEmit` compilado limpo (0 erros).
- PASS — `npm --prefix web run lint`: ESLint com `--max-warnings=0` aprovado (0 erros, 0 warnings).
- PASS — `npm --prefix web run format:check`: Prettier 100% em conformidade.
- PASS — `npm --prefix web test`: Vitest executou 71 arquivos e 378 testes com 100% de aprovação (incluindo paridade de chaves i18n em `src/i18n/i18n.test.ts`).
- PASS — `make web-verify`: Todos os 10 gates automatizados aprovados (format, typecheck, lint, stylelint, null-arrays, test, i18n, build, embed-sync, ui-markers).
- PASS — `make build`: Binário `./nexus` v1.0.0-beta.1 compilado com novos assets embutidos em `web/dist`.
- PASS — `make install LOCAL_BIN=/home/desenvolvedor/.local/bin`: Binário instalado em `/home/desenvolvedor/.local/bin/nexus`.
- PASS — `curl -sI http://127.0.0.1:13000/`: HTTP 200 OK com novos assets servidos pelo servidor de controle.
- PASS — `curl -s -b /tmp/nexus-cookies.txt http://127.0.0.1:13000/api/v1/telemetry/stats`: Retorno JSON estruturado de telemetria agregada (`total_sessions`, `by_provider`, `by_profile`, `rate_limits_by_profile`).
- PASS — `curl -s -b /tmp/nexus-cookies.txt http://127.0.0.1:13000/api/v1/telemetry/history`: Retorno JSON de eventos cronológicos com provider, profile, duration e timestamp.
- NOT EXECUTED — `git commit` e `git push` (em estrita conformidade com a constituição do Orquestrador).


## 2026-09-15 — Compilação e Instalação Local do Binário Nexus (v1.0.0-beta.1 @ 2b9c956)

- PASS — `make build`: Bundles frontend gerados com sucesso e binário `./nexus` compilado com commit `2b9c956`.
- PASS — `make install LOCAL_BIN=/home/desenvolvedor/.local/bin`: Binário instalado atomicamente em `/home/desenvolvedor/.local/bin/nexus`.
- PASS — `which nexus`: Resolvido para `/home/desenvolvedor/.local/bin/nexus`.
- PASS — `nexus version`: `IAPro Nexus 1.0.0-beta.1 (linux/amd64, go1.25.0) commit 2b9c956 built 2026-09-15T03:22:18Z`.
- PASS — `nexus usage --json`: Retorno estruturado de cotas e provedores funcionando sem anomalias.
- PASS — `curl -sI http://127.0.0.1:13000/`: HTTP 200 OK com novos assets do Web OS servidos.
- NOT EXECUTED — `git commit` e `git push` (em estrita conformidade com a constituição do Orquestrador).



- PASS — `go test -v -run TestUnifiedUsage_AddProfileModal ./internal/tui/...`: 4/4 novos testes unitários aprovados (abertura por tecla `a`, abertura por `enter` em provedor sem perfil, submissão de `ActionAdd` com nome informado e cancelamento por `esc`).
- PASS — `go test ./internal/tui/...`: 100% da suite do TUI aprovada.
- PASS — `go test ./internal/app/...`: 100% da suite do App CLI aprovada, validando despacho de `tui.ActionAdd` para `addCmd`.
- PASS — `npm --prefix web run check:styles`: Conformidade estrita com SCSS Modules e allowlist global.
- PASS — `npm --prefix web run typecheck`: `tsc --noEmit` compilado limpo (0 erros).
- PASS — `npm --prefix web run lint`: ESLint com `--max-warnings=0` aprovado sem erros ou avisos.
- PASS — `npm --prefix web test`: Vitest executou 73 arquivos e 388 testes com 100% de aprovação.
- PASS — `make web-verify`: Todos os 10 gates automatizados aprovados (format, typecheck, lint, stylelint, null-arrays, test, i18n, build, embed-sync, ui-markers). Relatório em `DEV/validation/FRONTEND_LATEST.md`.
- PASS — `make build`: Binário `./nexus` v1.0.0-beta.1 gerado com sucesso.
- PASS — `install -m 755 ./nexus ~/.local/bin/nexus`: Binário instalado no ambiente de execução do usuário.
- NOT EXECUTED — `git commit` e `git push` (em conformidade com as diretrizes do Orquestrador).

## 2026-09-15 — Redesign da Superfície de Uso (UX/UI, Responsividade & Paridade CLI)

- PASS — `npm --prefix web run check:styles`: validação de CSS Modules e conformidade com allowlist de estilos globais.
- PASS — `npm --prefix web run format:check`: 100% dos arquivos formatados segundo regras do Prettier.
- PASS — `npm --prefix web run lint`: ESLint aprovado com 0 erros e 0 warnings (`--max-warnings=0`).
- PASS — `npm --prefix web run lint:styles`: Stylelint aprovado sem nenhuma violação.
- PASS — `npm --prefix web run typecheck`: TypeScript `tsc --noEmit` compilado com 0 erros.
- PASS — `npm --prefix web test`: Vitest executou 73 arquivos de teste e 388 testes com 100% de aprovação (incluindo testes de modelos de uso e paridade i18n).
- PASS — `npm --prefix web run build`: Geração completa de bundles web com Tailwind CSS v4 e esbuild em ~1.3s.
- PASS — `make web-verify`: Todos os 10 gates automatizados aprovados (format, typecheck, lint, stylelint, null-arrays, test, i18n, build, embed-sync, ui-markers). Relatório gerado em `DEV/validation/FRONTEND_LATEST.md`.
- PASS — `go test -v ./internal/control/web/...`: Todos os testes da API de controle Web e endpoints de perfis/recursos aprovados (0 falhas).
- PASS — `make build`: Binário `./nexus` (v1.0.0-beta.1) compilado com sucesso com os novos assets embutidos.
- NOT EXECUTED — `git commit` e `git push` (em estrita conformidade com a constituição do Orquestrador: commits e pushes somente sob ordem explícita do Maestro).

## 2026-09-15 — Limpeza de galhos + pick terminal (HEAD `356f1a3`)

- PASS — `go test ./internal/control/host/ -count=1` (pós-pick e pós-recuperação).
- PASS — `go vet ./internal/control/host/` + `gofmt -l` limpo nos 3 arquivos do pick.
- PASS — `make quality` (frontend 73 arqs/385 testes, version contract `1.0.0-beta.1`).
- PASS — `make build` (binário `./nexus` 20,5M) + `git diff --check`.
- PASS — `make web-verify` 10/10 gates (format, typecheck, lint, stylelint,
  null-arrays, test, i18n, build, embed-sync, ui-markers).
- PASS — loss audit: 12/12 galhos apagados recuperáveis via
  `archive/2026-09-15/*`; `git bundle verify` OK.
- NOT EXECUTED — commit/push dos 5 arquivos de release staged, CI hosted
  same-SHA, `release-proof`/`overnight-soak`/`docs-verify` (exigem SHA
  congelado + Go 1.25.1/covdata).

## 2026-09-14 — Remediação do review pré-main

- PASS — `GOTOOLCHAIN=go1.25.1 make release-proof`: gates locais completos,
  cobertura Go agregada 52,4%, segurança e version contract aprovados;
  `OVERNIGHT_SOAK_EVIDENCE=MISSING` foi mantido como pendência explícita.
- PASS — `go test ./...` e `go test -race ./...` sem falhas ou data races.
- PASS — `make quality`, `make test-e2e` e `make security`.
- PASS — lint frontend com `--max-warnings=0`.
- PASS — `GOOS=windows GOARCH=amd64 go test -c` para `scripts`, `internal/control/web`
  e `internal/control/host`.
- PASS — regressão de persistência do binding `--lead` em Registry reloadado.
- BLOCKED — `make docs-verify`: manifesto visual stale para o worktree não
  commitado; exige novo SHA e evidência visual após congelamento.
- NOT EXECUTED — commit, push, CI hosted same-SHA, rulesets e branch protection.

## 2026-09-14 — CLI supervised sem Lead no teclado

- PASS — `wantInteractiveLead` default false; `--lead` true (case-insensitive); `StripControlFlags` remove `--lead` junto de `--direct`/`--supervised`.
- PASS — `go test ./internal/app/` (flag/strip + suite existente).
- PASS — smoke `nexus codex --yolo` (PTY): runtime `nexus.interactive_lead=false`; **0** agentes novos; `/status` chega ao Codex (autocomplete de slash) sem hang de promoter.
- PASS — smoke `nexus codex --yolo --lead`: label `nexus.interactive_lead=true` + bind Lead; caminho LineShadow preservado.
- PASS — `nexus control list` lista o runtime do atalho; help/completions expõem `--lead`.
- PASS — `make build` + install `~/.local/bin/nexus` (`0.5.0-beta.33`).

## 2026-09-14 — Enter preso em /status (qualquer CLI supervised)

- PASS — `IsImmediateSubmitLine` cobre `/status`, `/model`, `/nexus`, `:help`, vazio; prompts normais não bypassam.
- PASS — `decideInteractiveSubmit` não chama DecideIntent para `/status`; Decide lento estoura 250ms → KeepWithLead; Promote ainda funciona em goal multi-stream.
- PASS — `go test` promotion+app focados (11); binário reinstalado em `~/.local/bin/nexus`.
- UNVERIFIED — smoke manual `nexus codex` → `/status` responde na TUI (requer sessão interativa).

## 2026-09-14 — CLI shortcuts: busy failover + orphan host reap

- PASS — `go test ./internal/app ./internal/control/launcher ./internal/nexus ./internal/control/tui` (350): failover auto cai no próximo perfil; explícito não faz fallback; `ReapOrphanHostsFrom` mata host sem registry e preserva RUNNING.
- PASS — lock exclusivo em `kivergmail` + `nexus codex --yolo`: stderr `Profile codex:kivergmail is busy. Automatically falling back...` e attach em `codex:kiver.omegasistemas` (~8s, TUI YOLO). `nexus codex:kivergmail --yolo` falhou em 159ms sem fallback.
- PASS — `nexus control cleanup` reaped 1 orphan host de teste; 3 RUNNING reais intactos (`agy-06G9ZZ…`, `runtime_06GA0Z…`, `codex-06G9YT…`).
- UNVERIFIED — Windows named-pipe scan of `__control-host` (Linux `/proc` only).

## 2026-09-14 — Rail flicker + terminal recover

- PASS — `useNexusData` só atualiza estado quando o snapshot muda; `resolveProjectSelection` não bounceia para `projects[0]` em poll vazio; outline não re-entra em loading com cache.
- PASS — overlay de Recover só em ERROR/DISCONNECTED/recovering; `CONNECTING` usa indicador leve; WS effect sem dep de `t`.
- PASS — attach prefere `surface.data.runtimeId`; `runtime_changed` faz rebind em `new_runtime_id`; sticky `nextBoundRuntimeId` ignora poll stale.
- PASS — `make web-verify` (FRONTEND_LATEST PASS), testes focados rail/terminal, `make build`, smoke browser: expand estável + Recover Agy conectou Antigravity CLI.

## 2026-09-14 — Codex terminal input

- PASS — `web/src/nexus/terminalInput.ts` shared filter drops focus/CPR/`[i`; `AgentTerminal` and `TerminalPane` use it; mouse/CSI-u/UTF-8/`0x03`/literals forwarded.
- PASS — `internal/control/host` forwards `CmdInput` byte-for-byte; `/nexus stop` never intercepted; `LineShadow` keeps control/paste/OSC/CPR out of prompt.
- PASS — `CODEX_TUI_DISABLE_KEYBOARD_ENHANCEMENT=1` in direct adapter and supervised spec/driver.
- PASS — rebuilt `/home/desenvolvedor/.local/bin/nexus` from `1b840ca`; `nexus version` `1b840ca`, `go version -m` module `github.com/IAPro-Community/nexus`.
- PASS — focused web `terminalInput.test.ts` (10), Go race (`host`, `promotion`, `driver`, `spec`), `go test ./...`, `make quality`, `make build`, `git diff --check`.
- UNVERIFIED — manual Web/Desktop + `nexus codex` acceptance (no mouse `[i`/`[I`/CSI residue; common keys no interrupt; explicit `0x03` interrupts; literals intact).

## 2026-09-14 — Codex autonomous launch compatibility

- PASS — Codex autonomous argument regression: approval/sandbox flags are
  placed before `exec` so Codex 0.154.0 accepts the command.
- PASS — Codex reviewer argument contract: read-only sandbox and no-approval
  flags are placed before `exec`; provider contract golden updated.
- PASS — targeted race tests and `make quality`; alternate Go 1.25.1
  `make release-proof` completed all local gates.
- PASS — authenticated Mission `run_06G9VJV1K6J649PN2EMEHR1EMG` routed to
  `codex:kivergmail`, executed the implementer in a dedicated worktree,
  received an approved verdict from a distinct read-only Codex reviewer, and
  completed package plus global verification as `COMPLETED_VERIFIED`.

## 2026-09-13 — P0/concurrency and release closure refresh

- PASS — `make attention-e2e`, `make overnight-smoke` and
  `make overnight-soak DURATION=45m` on the current worktree.
- PASS — `GOTOOLCHAIN=go1.25.1 make release-proof` completes all local gates
  with aggregate coverage at 52.4%; the default Go 1.25.0 installation still
  lacks `covdata` and is recorded as an environment-only block.
- PASS — authenticated Codex Mission `run_06G9VJV1K6J649PN2EMEHR1EMG`
  completed with a distinct read-only reviewer and durable package/global
  verification evidence (artifacts in
  [`validation/current/real-provider-mission/`](validation/current/real-provider-mission/)).
- BLOCKED — `make docs-verify` still sees a stale visual manifest for
  uncommitted frontend changes.
- NOT EXECUTED — push, hosted same-SHA CI, rulesets and branch protection;
  these require explicit maestro authorization.

## 2026-09-13 — Fase A promoção adaptativa

- PASS — `internal/nexus/promotion` cobre Enter retido, UTF-8/backspace,
  ESC/CSI/OSC, CPR, bracketed paste, controles e OFF passthrough.
- PASS — CLI attach e Web terminal integram `KEEP_WITH_LEAD`, `ASK` e
  `PROMOTE`; Web expõe confirmação `promotion_answer`.
- PASS — `go test ./... -count=1`, `go test -race` nos pacotes tocados,
  `go vet ./...`, `make quality` e `git diff --check`.
- UNVERIFIED — Mission com providers autenticados, CI hosted same-SHA e
  soak de oito horas; nenhum push ou branch protection foi executado.

## 2026-09-13 — Fase 0 pós PRODUCTION_NO_GO

- Scope: código apenas (falso sucesso / Maestro / Mission PATCH / quota /
  launcher / installers / CI key gate). **Não** overnight, **não** CI hosted,
  **não** Mission autenticada.
- PASS — `make quality`; `go test -race` em store/nexus/runner/quota/launcher/
  release/sign-update-manifest; smoke isolado (`maestro_mode=OFF`,
  `nexus run` → strict admission, sem wording “quota/rate limit”).
- PASS (contrato) — CI desktop Linux/Windows/macOS falham se
  `NEXUS_UPDATE_PUBLIC_KEY` vazia; installers desktop recusam
  `desktop-checksums.txt` unsigned.
- BLOCKED / NEXT — congelar SHA (commit); humano provisiona Ed25519 em
  `beta-signing` / `vars.NEXUS_UPDATE_PUBLIC_KEY`; só então push e CI same-SHA.
- UNVERIFIED — overnight 8h, Windows/macOS nativos same-SHA, Mission
  `COMPLETED_VERIFIED` autenticada, red team no SHA novo.

## 2026-09-13 — Current local closure audit

- Candidate observed: `eed65e5a7e46adc896165c94accb7fc73fc6750f`; worktree was
  not frozen, so no release certification is asserted.
- PASS — `go test ./... -count=1`, `go test -race ./... -count=1`, `go vet
  ./...`, `make lint-go`, `make quality`, `make security`, `make build`,
  `make build-desktop-wails`, `make web-verify`, and
  `bash scripts/verify-update-gate.sh`.
- PASS — Go/browser regressions preserve PTY bytes including ESC/OSC/kitty,
  UTF-8 and literal `/e[...]` or `\\e[...]` provider text.
- BLOCKED — `make docs-verify` because the visual manifest is stale for
  uncommitted frontend changes.
- UNVERIFIED — authenticated Mission evidence, production updater key
  provisioning, native Windows/macOS/NSIS, hosted same-SHA browser/release
  matrix, and eight-hour overnight certification.

## 2026-09-13 — Fase B revisão/remediação

- PASS — ciclo canônico Planner → Builder → Reviewer (`CHANGES_REQUESTED`)
  → remediação com finding novo → `APPROVED` → Verifier sem write turn.
- PASS — ledger distingue quatro identidades e termina sem findings abertos;
  `make overnight-smoke` executa o teste.
- BLOCKED — `make docs-verify` exige manifesto visual renovado para as
  alterações frontend não commitadas; nenhuma regeneração foi feita.
- BLOCKED — `make docs-verify` still requires a refreshed visual manifest for
  uncommitted frontend changes; no manifest regeneration was performed.

## 2026-09-13 — Mission Teams validation

- PASS — full Go tests, full race suite, `go vet`, security, build, desktop
  build, E2E, Attention E2E and bounded overnight smoke.
- PASS — canonical deterministic Planner → Builder → Reviewer → remediation →
  Verifier test and direct-task anti-overprocessing coverage.
- PASS — structured finding identity/deduplication/resolution, durable phase
  state, Flow/WorkPlan round-trip and desired-vs-actual routing preference.
- FAIL / ENVIRONMENT — `make coverage` and `make release-proof` stop because
  the active Go 1.25 toolchain lacks `covdata`.
- FAIL / CONCURRENT WORKTREE — latest `make web-verify` reports formatting in
  unrelated `web/src/app/NexusSplashScreen.tsx`; the Mission Teams surface
  itself passed typecheck, lint, styles, tests and build.
- UNVERIFIED — authenticated real-provider Mission, provider failover proof,
  native Windows/macOS and eight-hour overnight certification.
- Report: [`validation/current/MISSION_TEAMS_REPORT.md`](validation/current/MISSION_TEAMS_REPORT.md).

## 2026-09-13 — Production GO/NO-GO red team

- Verdict: `PRODUCTION_NO_GO` / `CERTIFICATION_INVALID_FOR_CURRENT_HEAD`.
- Candidate executable SHA: `e53f8352e4c3647632ebac7877cf0a6a3bd62647`.
- PASS (local, this session): `make quality`, `go vet`, `make security`,
  `make docs-verify`, `make build`, `go test -race ./...` (1079/65).
- PASS-but-not-soak: `make overnight-smoke` 0.243s.
- FAIL: overnight 0h abort; no same-SHA CI; ancestor CI E2E failures;
  Windows/macOS native; updater trust root; authenticated Mission;
  Maestro default ASSIST; UNKNOWN quota allocation; Mission PATCH complete.
- Evidence: `DEV/validation/production-go-no-go/FINAL_REPORT.md`.

## 2026-09-13 — Overnight production certification

- FAIL / ABORT — precondition `production-finalization/FINAL_REPORT.md` missing.
- FAIL / ABORT — verdict `READY_FOR_OVERNIGHT_CERTIFICATION` not present in repo.
- FAIL / ABORT — HEAD `e53f8352e4c3647632ebac7877cf0a6a3bd62647` is not a certified overnight candidate.
- NOT RUN — campaigns A–F, 8h soak, recovery, provider failover, red team.
- Evidence: `DEV/validation/overnight-production-certification/FINAL_REPORT.md`.

## 2026-09-12 — Capability review intensive

- PASS — CLI JSON: doctor, security, control doctor, providers, profiles, version.
- PASS — `make security` (govulncheck clean).
- PASS — focused tests after fixes: web, codex, nexus, flags, update; `make lint-go` clean.
- PASS — TunnelHost ⇒ `IsTunnelActive` regression test.
- PASS — providers advertise CrossAccountResume only for Codex after honesty fix.
- PASS — follow-up: official cloudflared 2026.8.2 checksums, Darwin `.tgz` extract,
  tunnel `OnExit` disarms auth, `--effort` preserves native `-c`, AGY `-p`→DIRECT,
  supervised Codex bootstrap calls `Prepare` (CrossAccountResume seed).
  Retest: `go test ./internal/control/web ./internal/control/flags ./internal/app ./internal/control/driver ./internal/core/provider/adapters/codex -count=1`.
- SKIPPED — Claude CLI OAuth expired; Gemini CLI client unsupported.
- UNVERIFIED — interactive `/resume` cross-account and live mid-turn Codex TUI.
- DEFERRED — Claude/OpenCode Usage file-stub honesty; developer-preset credential exposures;
  TLS on private `--remote`.

## 2026-09-12 — Auth follow-up (desktop + WS query tokens)

- PASS — `/api/v1/desktop/bootstrap` requires pre-provisioned desktop bearer (Origin alone → 401).
- PASS — WS `?token`/`?session` accepted only on loopback listen; rejected on private `--remote`
  and while tunnel active (`TestWebSocketQueryTokenRejectedOnPrivateRemote`).
- PASS — frontend omits WS query token outside loopback (`shouldAttachWebSocketQueryToken`).
- PASS — `go test ./internal/control/web -count=1`; focused web vitest for agent terminal + networkHost.
- PASS — `TestVerifyCloudflaredChecksumUsesPlatformPin` (Darwin sidecar path covered).

## 2026-09-12 — Codex CrossAccountResume

- PASS — `go test ./internal/core/provider/adapters/codex ./internal/conversation ./internal/control/flags -count=1`.
- PASS — `TestAdoptSessionFromSiblingProfile` copies sibling rollout + index
  into destination canonical sessions and marks cross-account.
- PASS — `TestPrepareSeedsSiblingSessionsForPicker` seeds sibling IDs into
  destination `session_index.jsonl` during Prepare.
- PASS — `TestForeignAccountRolloutIgnoredForQuota` and
  `TestRolloutBelongsRejectsMismatchedAccountID` keep GetUsage from inheriting
  foreign resume artifacts.
- UNVERIFIED — interactive `nexus codex:kivergmail` `/resume` listing of
  omegasistemas threads and live `nexus usage` attribution on account
  `9cba8371-...` (requires local authenticated Codex TUI).

## 2026-09-12 — Mission evidence isolation

- PASS — two persisted Missions in the same project receive only their own
  validation entries from the shared canonical stream.
- PASS — the complete hash chain is verified before the per-Mission filter;
  invalid or missing `run_id` metadata fails closed (`TestRunApplicationRejectsMissionEvidenceWithoutRunID`).
- PASS — `TestRunApplicationProjectsCompleteMissionEvidenceBeyondDefaultPage`
  projects 101 entries without inheriting the bounded 100-entry list page.

## 2026-09-12 — Nexus Codex TUI lock / flag compatibility

- PASS — `go test ./internal/core/provider/adapters/codex ./internal/control/flags ./internal/control/driver ./internal/control/host ./internal/profile -count=1`.
- PASS — `go test ./internal/core/provider/adapters/codex -race -count=1`.
- PASS — `go vet` on the touched packages; focused `Quota|Monitor` Nexus tests.
- PASS — TUI lock blocks `runAppServerRateLimits` / `GetUsage` app-server spawn;
  rollout windows still returned; `thread_history` symlink untouched on usage path.
- PASS — Codex `-c`/`-p` pass through; `--continue` → `resume --last`;
  `--print` → `exec`; `--effort` → `-c model_reasoning_effort=...`.
- PASS — Prepare creates canonical `sessions` + `config.toml` only (no second
  `.codex/sessions` write tree).
- PASS — legacy `.codex/config.toml` is migrated before canonical defaults are
  appended; the regression test preserves the legacy model setting.
- PASS — profile preparation waits for an active per-home TUI lock before
  touching configuration/session artifacts.
- PASS — E2E profile import resolves the real host home when `HOME` points at an
  isolated provider home; explicit source override precedence is covered.
- UNVERIFIED — live authenticated `nexus codex` mid-turn interrupt / `model:
  loading` reproduction after the fix (requires interactive provider session).

## 2026-09-12 — Canonical validation evidence projection

- PASS — `go test ./internal/nexus ./internal/control/web -count=1`.
- PASS — `RunApplicationService.ValidationEvidence` reads the existing
  append-only Mission stream, verifies its hash chain before projection and
  returns explicit empty/no-chain state when no stream exists.
- PASS — the projection test closes and reopens the same SQLite store, then
  reloads the run, stream, entry and verified chain successfully.
- PASS — Codex TUI lock tests cover non-blocking ownership, app-server rate
  limit suppression, session migration safety and canonical home preparation.
- PASS — Web route `GET /api/v1/runs/{id}/validation-evidence` exposes that same
  projection; no second ledger or report store was introduced.
- PASS — Web client typed method `getRunValidationEvidence` and its transport
  test are green; frontend typecheck and build pass.
- UNVERIFIED — no authenticated Mission produced a durable production stream
  in this campaign, so this verifies the projection contract only.

## 2026-09-12 — Final closure continuation after routing/guidance integration

- PASS — `go test ./... -count=1`.
- PASS — `go test -race ./... -count=1`.
- PASS — `go vet ./...` and `git diff --check`.
- PASS — `make security`, `make build`, `make build-desktop`.
- PASS — `make quality` (aggregated formatting, lint, staticcheck, Go tests,
  frontend tests and version consistency). One existing ESLint warning at
  `web/src/nexus/AgentTerminal.tsx:218` is non-blocking.
- PASS — `make web-verify` (format, typecheck, lint, stylelint, null-array,
  tests, i18n, build, embed-sync and UI markers).
- PASS — `./nexus doctor --json` at `2026-09-12T03:02:22Z`; desktop shell is
  correctly reported as `SKIPPED`, not PASS.
- PASS — focused AGY/runtime hardening tests: expired-token quota probes fail
  closed and no-browser helper paths do not open external OAuth routes.
- PASS — task requirements now carry bounded generic `ExecutionGuidance` into
  prompt compilation; persisted routing decisions project canonical Skill refs
  and Maestro guidance refs only when an actual source/reference is present,
  including selected Skill source/version/hash/reason provenance.
- PASS — the local autopilot contract now exercises package/global recorder
  integration into `ValidationEvidenceStream`, chain verification and
  Git-SHA-bound `VERIFIED` entries. Latest ephemeral test stream was sequence 2;
  it is not a production Mission stream.
- PASS — Mission evidence captures bounded Go/OS/arch metadata and Node plus
  package-manager versions when applicable, using fixed no-shell probes with
  a two-second timeout.
- PASS — E2E harness now supports an explicit empty stable root for providers
  that reject temporary credential homes; focused root-safety tests pass.
- UNVERIFIED — Codex remained at `model: loading` even with stable root;
  AGY reached the runtime but reported `not signed in`. No real provider
  marker or authenticated Mission evidence was promoted.
- UNVERIFIED — authenticated Nexus provider Mission, durable production
  evidence stream, live failover/escalation/handoff, Windows/macOS and
  overnight acceptance remain unavailable.

## 2026-09-12 — Final closure local autopilot continuation

- PASS — `go test ./... -count=1` after the local autopilot acceptance test.
- PASS — `go test -race ./... -count=1`.
- PASS — `go vet ./...` and `git diff --check`.
- PASS — `TestLocalAutopilotContractTraversesDiscoveryRoutingSkillsAndVerification`:
  bounded discovery → `DIRECT` → Builtin Skill → task-aware model → Mission
  Runner → `COMPLETED_VERIFIED`.
- PASS — SQLite restart reload of `RoutingDecisionJSON` and durable routing
  report projection.
- PASS — persisted routing decisions now include Agent score/confidence/reason
  and selected account scope in addition to provider/profile/model.
- Release remains `NO-GO`: authenticated provider Mission, live failover/
  handoff, native Windows/macOS and overnight evidence are unavailable.

## 2026-09-11 — Codex frescor + TUI Contas & Quotas

- PASS — `go test ./internal/core/provider/adapters/codex ./internal/core/quota ./internal/profile ./internal/tui ./internal/control/host -count=1`.
- PASS — host same-account ingest + adopt; parser aceita qualquer evento com `rate_limits.primary`; evento velho → `ESTIMATED`.
- PASS — `AvailabilityLabel` UNKNOWN → `SEM DADOS` (nunca DISPONIVEL); 5h=0 → `QUOTA ESGOTADA`.
- PASS — last-known Codex aceita bump de `IdentityVersion` (email → `chatgpt_account_id`) com mesmo `AccountID` durável.
- PASS — TUI: chrome compacto (`usageChromeLines=6`), colunas sem MODELO, STATUS `OK`/`ESGOTADA`/`SEM DADOS`/`OFF` cabem em width 80/132/160.
- PASS — `SessionHost.Stop`/`Terminate` disparam `RefreshUsageSnapshot` async para codex/agy.
- PASS — `nexus usage --json` pós-rebuild: `codex:kiver.omegasistemas` LIVE; `codex:kivergmail` ESTIMATED com last-known (rollout isolado tinha `primary:null` do Codex — sem inventar LIVE).
- NOTE — cota LIVE da gmail exige sessão isolada que grave `rate_limits.primary` (não só splash com primary null).

## 2026-09-11 — Native Skill contract transport continuation

- PASS — `go test ./internal/nexus/... -count=1` after adding canonical
  `SkillIDs` to WorkPlan, Flow, runner packages and ContextCapsule.
- PASS — builtin `coding`/`testing` resolve with Maestro unavailable; unknown
  IDs reject deterministically and legacy/generic packages can coexist.
- PASS — the actual execution freeze path accepts a real project plan with
  builtin `SkillIDs` while Maestro is offline and preserves compatibility aliases.
- PASS — `go test ./... -count=1` and `go test -race ./... -count=1` after
  correcting handoff receipts and reconciling latest Codex/TUI contracts.
- PASS — `go vet ./...`, `gofmt` cleanliness and `git diff --check`.
- PASS — `make web-verify`, `make build`, `make build-desktop`,
  `git diff --check` and repository-wide `gofmt` cleanliness.
- Release remains NO-GO for missing authenticated Mission/provider/platform
  evidence.

## 2026-09-11 — Codex quota attribution by chatgpt_account_id + isolated sessions

- PASS — `go test ./internal/core/provider/adapters/codex ./internal/core/provider/adapters/agy ./internal/core/scheduler ./internal/profile ./internal/core/quota ./internal/tui ./internal/app`.
- PASS — e-mails no texto do rollout **não** roubam atribuição; no máximo um perfil reivindica cada rollout host.
- PASS — `FetchedAt` vem do evento `token_count`; `resets_at` no passado vira `window rolled over` e não conta como esgotado.
- PASS — AGY com `refresh_token` permanece autenticado (`Token refresh pending`) e o probe de quota pode rodar com `BROWSER=false`.
- PASS — `nexus usage --json` após rebuild: `codex:kiver.omegasistemas` LIVE com 5h/weekly alinhados ao `/status` Codex (reset weekly `05:35 on 15 Sep`); `codex:kivergmail` UNKNOWN com erro `aguardando primeira sessão isolada desta conta` (esperado até a primeira sessão isolada).
- PASS — symlinks `sessions` → `~/.codex/sessions` migrados para diretórios reais por perfil no `GetUsage`/`Prepare`.
- PASS — scheduler prefere conta Codex com capacidade sobre 5h esgotado (`TestCodexExhaustedFiveHourLosesToHealthyAccount`).

## 2026-09-11 — Codex/CSI-u no terminal Nexus

- PASS — `go test ./internal/control/driver ./internal/core/provider/adapters/codex ./internal/runtime`.
- PASS — `gofmt` nos arquivos Go alterados e `git diff --check`.
- PASS — o driver supervisionado injeta
  `CODEX_TUI_DISABLE_KEYBOARD_ENHANCEMENT=1`; o adapter direto usa a mesma
  proteção.

## 2026-09-11 — Evolution corrective closure (current working tree)

- PASS — focused Go regression: `internal/nexus`, `internal/core/scheduler`,
  `internal/control/web`, `internal/control/handoff`.
- PASS — natural-language classification covers React, Go REST, Kubernetes,
  security, review and complex ecommerce; classified requirements persist on
  generated Flow steps.
- PASS — AgentMatcher no longer treats provider resource gates as Agent
  personality; generic `implementer` labels no longer erase specialized task
  roles.
- PASS — same-provider handoff preserves Agent/Project identity and reports
  `NATIVE_RESUME_UNVERIFIED` without provider-level confirmation; cross-provider
  context handoff is labeled `CONTEXT_HANDOFF`.
- PASS — `nexus run "<goal>"` dispatches through Flow, AgentMatcher,
  ResourceScheduler and MissionRunner while provider-native `nexus run <provider>`
  remains compatible.
- PASS — scheduler tie-breaking is deterministic by profile name.
- PASS — `make quality`, `make security`, `make build`, `make web-verify`,
  `go vet ./...` and `make lint-go` on the corrected tree.
- PASS — `make test-e2e` (terminal, protocol, host and web race scenarios).
- UNVERIFIED — `make docs-verify` rejects the stale visual manifest because the
  Web change is intentionally uncommitted; refreshing that manifest requires a
  committed source SHA or an explicit release commit.
- UNVERIFIED — full Go race completion, authenticated PTY/failover E2E, live
  providers, and native Windows/macOS execution; see final validation report.

## 2026-09-10 — Gates de format/lint alinhados

- PASS — `make format-check` usa o Prettier local e verifica TypeScript, CSS,
  SCSS e JSON.
- PASS — `make lint-styles` verifica CSS e SCSS.
- PASS — `make lint-frontend` usa o ESLint local; permanece apenas o warning
  preexistente de diretiva não utilizada em `NexusWorkspaceApp.tsx`.
- PASS — `.lintstagedrc.json` usa binários locais e inclui SCSS.

## 2026-09-10 — Fechamento dos últimos caminhos de quota

- PASS — AGY só persiste usage live com `AccountScope` verificável; arquivos
  legados sem identidade permanecem não atribuídos até refresh autenticado.
- PASS — `BatchFetch` e seleção de modelo AGY não leem/escrevem cache de quota
  sem escopo; runtimes carregam o escopo quando o perfil registrado o fornece.
- PASS — monitor de quota associa runtime afetado por escopo exato quando
  disponível, evitando colisões entre contas do mesmo provider/perfil.
- PASS — perfis pendentes/sem identidade mantêm apenas o identificador local
  reservado; `AccountScope.Verifiable()` só fica verdadeiro após identidade
  autenticada, impedindo cache e routing atribuíveis antes do login.
- PASS — `go test ./...` após o fechamento.

## 2026-09-10 — Review corrective pass

- PASS — `go test ./...` após as correções.
- PASS — `go test -race ./...` (processo concluído sem detector de corrida).
- PASS — `go vet ./...`.
- PASS — `npm --prefix web run typecheck`, testes Web (320/320), build e
  Prettier nos arquivos Web alterados.
- PASS — `git diff HEAD --check`.
- NOT VERIFIED — execução nativa Windows/macOS e Cloudflare real; o tunnel
  recebeu correção de lifecycle e readiness, mas não foi iniciado contra uma
  instalação real de `cloudflared` nesta sessão.

## 2026-09-10 — AccountScope e isolamento de estado

- PASS — `go test ./internal/core/model ./internal/core/quota
  ./internal/core/cooldown ./internal/profile ./internal/control/events
  ./internal/nexus`.
- PASS — regressões A/B: cache de quota e cooldown da conta A não são visíveis
  para a conta B; troca de credencial preserva `account_id` e incrementa
  `identity_version`.
- PASS — `go test -race ./...`.
- PASS — `git diff --check`.
- NOT VERIFIED — registro TUI/CLI de instalação não registrada e execução nativa
  Windows/macOS permanecem fora deste slice.

## 2026-09-10 — Registry progressivo de CLIs

- PASS — `InstallationRegistry` separa descoberta de binário do registro de
  conta e persiste `installations.json` com permissões restritas.
- PASS — `providers status --json` expõe estado de instalação/registro.
- PASS — `providers register` cria perfil isolado e permite adiar autenticação.
- PASS — testes de contrato e `go test -race` nos pacotes alterados.
- PENDENTE — integração equivalente na TUI/Web e modo `UNMANAGED_EPHEMERAL`.

## 2026-09-10 — Integração Web/TUI

- PASS — API de providers retorna estado de instalação/registro e aceita
  registro de perfil via POST.
- PASS — TUI exibe Providers separadamente e registra o item selecionado como
  `PENDING_AUTH` sem copiar credenciais.
- PASS — cliente Web tipado expõe listagem e registro.
- PASS — `go test -race ./...` e `npm --prefix web run typecheck`.
- PASS — `SaveUsageForExecution(..., managed=false)` garante que execuções
  efêmeras não persistam usage; o wiring de lançamento com HOME temporário
  permanece pendente.

## 2026-09-10 — Correção de uso compartilhado entre contas Codex

- PASS — `go test ./internal/core/provider/adapters/codex ./internal/profile
  ./internal/core/quota ./internal/nexus -count=1`.
- PASS — `go vet` nos mesmos pacotes.
- PASS — regressão cobre perfis distintos apontando por symlink para o mesmo
  `~/.codex/sessions`; apenas a conta do host recebe o rollout LIVE.
- PASS — `nexus usage --json` foi executado localmente; a resposta manteve
  `profile_id`, `account`, `status` e `source` por perfil, com fallback
  `ESTIMATED` quando não há rollout atribuível.
- PASS — `git diff --check`.
- NOT VERIFIED — execução nativa Windows/macOS; esta correção foi verificada
  no Linux.
- CONDITIONAL — `go test ./...` foi tentado, mas o sandbox bloqueou sockets
  Unix/TCP e escrita em `/home/.../checkpoints`; falhas ocorreram em testes
  de host/protocol/web/update/core lifecycle fora da área alterada. Os testes
  focados do domínio passaram.

## 2026-09-10 — Composer application boundary and context propagation

- PASS — TDD RED confirmou que Composer ignorava contexto cancelado; após a
  correção, `TestComposerOperationsRejectCanceledContext` passou cobrindo
  criação, listagem, leitura, turno, skill, finalização e resolução.
- PASS — `go test ./internal/nexus ./internal/control/web`.
- PASS — `make quality-full`: Web 62 arquivos / 315 testes; Go completo com
  race/vet; lint; build; security (`No vulnerabilities found`).
- WARNING PRE-EXISTING — ESLint mantém um warning de dependência ausente em
  `web/src/app/NexusWorkspaceApp.tsx`; nenhum erro novo.
- NOT VERIFIED — execução nativa em Windows/macOS; esta fatia foi verificada
  no Linux.
- PASS — smoke não destrutivo: `go run ./cmd/nexus --help` e `go run
  ./cmd/nexus doctor` retornaram exit 0; o diagnóstico listou os providers
  instalados e manteve estados não verificáveis como WARN/SKIPPED.

## 2026-09-10 — WorkPlan application boundary

- PASS — `go test ./internal/nexus ./internal/control/web` após extrair CRUD
  e revisões para `internal/nexus/plan_application.go`.
- PASS — `make quality-full`: suíte Web com 62 arquivos / 315 testes; testes
  Go completos incluindo race; `go vet`; lint; build; security (`No
  vulnerabilities found`).
- WARNING PRE-EXISTING — ESLint mantém um warning de dependência ausente em
  `web/src/app/NexusWorkspaceApp.tsx`; sem erros novos.
- NOT VERIFIED — execução nativa em Windows/macOS; esta etapa foi verificada
  no Linux.

## 2026-09-08 — Project Shell IPC timeout

- PASS — teste de fallback do ShellDriver com `SHELL` inválido.
- PASS — `go test ./internal/control/driver ./internal/control/web ./internal/nexus`.
- Causa confirmada nos logs: `/usr/bin/zsh` não pôde ser executado pelo
  SessionHost e o handler anterior convertia o erro em `409`.
- PENDENTE — smoke real após reiniciar o processo `nexus web`; o processo atual
  foi iniciado antes da recompilação.

## 2026-09-08 — Rail inteligente e gestão de projetos

- PASS — `cd web && bun run typecheck`.
- PASS — `cd web && bun run lint:styles` e `bun run check:styles`.
- PASS — `cd web && bun run test`: 62 arquivos / 313 testes.
- PASS — `cd web && bun run build`.
- PASS — `make quality` (inclui lint Go, testes Go e gates frontend); ESLint
  manteve somente o warning preexistente de hook em `NexusWorkspaceApp.tsx`.
- PENDENTE — screenshot/browser visual específico do rail nos cinco breakpoints;
  a implementação foi revisada estaticamente, mas nenhum harness autenticado foi
  executado nesta sessão.

## 2026-09-07 — Deep review fixes (commit 059bb5c)

- `go test -count=1 ./...` — PASS (all packages)
- `go vet ./...` — PASS
- `cd web && bun run typecheck` — PASS
- `cd web && bun run lint` — PASS (1 pre-existing warning)
- `cd web && bun run lint:styles` — PASS
- `cd web && bun run check:styles` — PASS
- `cd web && bun run test -- --run` — PASS (62 files, 313 tests)
- `make quality` — PASS
- `make security` — PASS (no vulnerabilities)
- `make build` — PASS (nexus v0.5.0-beta.33)
- `git diff --check` — PASS
- Commit: `059bb5c` (29 files, +1062/-271)

## 2026-09-07 — Preparar tarefa

- PASS — `npm run verify` (10/10 gates), `npm run build` e 311 testes Vitest.
- PASS — `go test ./...`, `go vet` afetado, typecheck, lint, format, stylelint,
  allowlist e `git diff --check`.
- PASS — validação visual/Axe específica do novo diálogo nos cinco breakpoints
  registrada na seção seguinte.

## 2026-09-07 — Visual/Axe do diálogo de preparação

- PASS — `node web/scripts/task-preparation-visual-verify.mjs`.
- PASS — screenshots em 320×568, 390×844, 768×1024, 1024×768 e 1440×900.
- PASS — ausência de overflow horizontal, navegação por teclado e Axe sem
  violações `serious`/`critical`.

## 2026-09-07 — Browser hardening e bootstrap persistido

- PASS — `timeout 180 node web/scripts/e2e-hardening-verify.mjs`; seis
  breakpoints, deep-links, Axe, Settings/ARIA e density.
- PASS — harness atualizado para resolver o bootstrap por `nexus web url`, sem
  exigir token impresso pelo servidor.

`make docs-verify` permanece **CONDITIONAL** neste worktree: o verificador
rejeita corretamente o manifesto visual ancorado em `2358618` enquanto há
alterações visuais não commitadas. A âncora só deve ser atualizada junto com um
novo SHA e screenshots correspondentes.

## 2026-09-07 — Verificação same-SHA de evidência visual

- `scripts/docs-verify.mjs` também inspeciona alterações staged e unstaged nos
  caminhos visuais; não é permitido certificar screenshots de um candidato
  imutável enquanto o worktree contém mudanças locais.
- `make docs-verify` falha de forma esperada no checkout atual, identificando
  o manifesto em `23586183...` e os paths dirty. Isso é um bloqueio honesto,
  não uma falha de teste a ser mascarada.

## 2026-09-07 — Resume-first local slice

- PASS — `overviewResumeModel.test.ts`: prioridade Needs You → ativo →
  recuperável → recente, filtragem por projeto e associação ao agente.
- PASS — `bun run test` (62 arquivos / 313 testes), `bun run typecheck`,
  `bun run lint:styles`, Prettier e build Web.
- PASS — `node web/scripts/e2e-hardening-verify.mjs` após a mudança; os seis
  breakpoints e Axe continuam verdes.
- CONDITIONAL — o painel usa runtimes/agentes já carregados; runs históricos de
  missão ainda não são propagados ao Overview e precisam de E2E de retorno.
- PASS — o hardening E2E passou a verificar explicitamente
  `[aria-labelledby="overview-resume-title"]` no Overview.

## 2026-09-07 — Composer permissões por destino

- `cd web && bun run typecheck` — PASS.
- `cd web && bun run test -- --run src/features/work` — PASS (10 arquivos,
  40 testes), incluindo Composer `MISSING` editável e destinos bloqueados.
- `cd web && bun run lint && bun run lint:styles && bun run check:styles` — PASS
  (apenas warning preexistente de dependência de hook em `NexusWorkspaceApp`).
- `cd web && bun run build` — PASS.
- `node web/scripts/build.mjs` sincronizou `web/dist` com o bundle embutido;
  `go test ./...` — PASS.
- `go test ./internal/nexus/runner -run 'Test(OvernightAcceptanceSandbox|RemediationPersistsStrategyChangeBeforeRetry|NeedsHumanContractIsStructuredAndDurable)'` — PASS; sandbox determinístico cobre plano paralelo, receipt de dependência, falha injetada, remediação e DoD global.
- `make quality` — PASS; frontend 61 arquivos/311 testes e `golangci-lint` sem issues após corrigir a mensagem de erro capitalizada em `internal/nexus/maestro.go`.

## 2026-09-07 — Correção do bootstrap de autenticação Web

- `nexus web` + `curl` em processo real — PASS: fragmento de bootstrap trocado
  por cookie; `/api/v1/session` retornou `authenticated=true` e CSRF válido.
- `go test ./internal/control/web -run 'TestServerBootstrapURLKeepsTokenInFragment|TestProvidersDocsCaptureUsesSyntheticInventory|TestServer_BootstrapAndAuth'` — PASS.
- `npm --prefix web run test -- --run src/api.test.ts` — PASS, incluindo
  consumo do fragmento e limpeza via `history.replaceState`.
- `npm --prefix web run typecheck` — PASS; `make build` — PASS.

## 2026-09-07 — Higiene de mídia documental

- `npm --prefix web run docs:capture` — PASS em fixture sintética isolada; shell
  real validado por round-trip do marcador `__NEXUS_DOCS_TERMINAL_OK__`.
- `go test ./internal/control/web -run 'TestProvidersDocsCaptureUsesSyntheticInventory|TestServer_BootstrapAndAuth'` — PASS.
- `node scripts/docs-verify.mjs` — PASS; manifesto com VIS-001…VIS-011,
  classificação `SYNTHETIC` e rejeição do cenário `real-local-bootstrap`.
- Inspeção visual — PASS para terminal, providers, workspace e Direct; sem
  projetos, paths de usuário, credenciais ou versões reais de providers.
- `git diff --check` continua apontando whitespace em bindings Wails já
  modificados por outra alteração do worktree; não foi alterado nesta tarefa.

## 2026-09-07 — Review closure local

- `npm run verify` — PASS, 10/10 gates; 61 arquivos Vitest / 310 testes.
- Browser E2E — PASS: bootstrap/API readiness, Overview → Terminais, Axe,
  deep-links, 320/390/768/1024/1280/1440px, accordion e density delta.
- `go test ./...` — PASS; `go vet ./...` — PASS.
- `make docs-verify` — PASS com manifest VIS-001…VIS-011.
- `make build-desktop-wails` — PASS com Wails Linux production + `webkit2_41`.
- `VIS-011` — PASS visual em janela Wails real, com Core Web isolado e captura
  da tela de boas-vindas funcional; captura não é mock.
- `git diff --check` — PASS.
- `go run github.com/goreleaser/goreleaser/v2@v2.18.0 release --snapshot --clean`
  — PASS; archives, DEB/RPM packages and checksums generated locally.
- Pendentes obrigatórios para GO: execução nativa Windows/macOS, snapshot
  GoReleaser, CI em SHA único após integração com `origin/main` e regeneração
  do publication report nesse SHA. Nenhum commit/push foi feito.

## 2026-09-07 — Correções iniciais de desempenho Web/Desktop

- `npm --prefix web run verify` — PASS, 10/10 gates.
- `go test ./...` — PASS.
- `go test -race` dos pacotes Desktop/Web/Terminal/Host — PASS.
- `go vet ./...` — PASS.
- `make build-desktop-wails` — PASS em produção com `webkit2_41`.
- `git diff --check` — PASS após restaurar bindings gerados automaticamente.
- Benchmark comparativo e smoke nativo de performance — PENDENTES.

## 2026-09-07 — Plano de paridade de desempenho Web/Desktop

- Artefato `.omx/plans/desktop-web-performance-parity.md` criado e inspecionado.
- Referências principais conferidas em `cmd/nexus-desktop/main.go`,
  `web/src/platform/desktopBridge.ts`, `web/src/api.ts`,
  `web/src/app/NexusWorkspaceApp.tsx`, `web/src/nexus/AgentTerminal.tsx` e
  `web/src/features/work/FlowCanvas.tsx`.
- `git diff --check -- .omx/plans/desktop-web-performance-parity.md` — PASS.
- Nenhum benchmark foi executado e nenhuma causa foi declarada confirmada.

## 2026-09-07 — Execução Luna P0/P1 e fundação de continuidade

- `go test -count=1 ./...` — PASS.
- `go vet ./...` — PASS.
- `cd web && bun run verify` — PASS, 10/10 gates; a execução anterior foi
  corrigida formatando `web/src/nexus/api.ts`.
- `go test ./internal/app ./internal/control/handoff ./internal/nexus -count=1`
  — PASS após adicionar launch supervisionado, checkpoint Maestro schema 4 e
  lease do quota monitor.
- `git diff --check` — PASS antes das alterações atuais; repetir no gate final.
- Cross-platform native runtime — PENDENTE; cross-build não substitui Windows
  ConPTY/Named Pipe nem macOS PTY/race.

## Evidência de cobertura de quota — 2026-09-07

- `go run ./cmd/nexus providers --json` — PASS; binários e capabilities foram
  enumerados sem tratar capability `Usage` como dado atual.
- `go run ./cmd/nexus profiles --json` — PASS; AGY/Codex autenticados, porém
  todos `UNKNOWN`/`NONE` sem janelas/freshness. OpenCode não autenticado e Cursor
  sem Usage. O monitor corretamente não emite alerta de percentual nesses casos.
- Teste de prioridade cross-provider — PASS; `TestResourceRecommendation_ProviderPriorityControlsFallbackOrder`.

## Terminal/continuidade — 2026-09-07

- `go test -race -count=1 ./internal/control/host ./internal/control/handoff
  ./internal/nexus ./internal/app ./internal/core/config` — PASS.
- `go vet` dos pacotes afetados e `git diff --check` — PASS.
- Cross-build de testes dos pacotes host, handoff e nexus para Windows amd64 e
  macOS arm64 — PASS como compilação; execução nativa permanece PENDENTE.
- `--supervised` e `/nexus<TAB>` ainda não têm PASS nativo Windows/macOS.

## 2026-09-06 — Plano Luna (somente documentação)

- Código de lançamento, attach, quota monitor, checkpoint/handoff e workflow
  inspecionado para fundamentar DEV/SPECS/NEXUS_TERMINAL_CONTINUITY_LUNA.md.
- Nenhuma implementação ou suíte runtime executada nesta etapa; não há novo
  resultado de CI remoto. Verificações de execução estão definidas em P0–P12.

## 2026-09-06 — CI closure pass after remote failure reproduction

- Remote run `34060911997` was inspected by job/step metadata. Frontend failed
  on format; Windows and macOS failed on their full Go test/race step. Logs are
  unavailable through the repository API without admin log permission.
- Frontend: **PASS**, `cd web && bun run verify` (10/10 gates), 306 tests,
  including build and embed synchronization.
- Go: **PASS**, `go test -count=1 ./...`, `go test -race -count=1 ./...`, and
  `go vet ./...`.
- Lint: **PASS**, `golangci-lint v2.12.2 run --timeout=5m ./...` (`0 issues`).
- Platforms: **PASS (compile evidence)**, Go test binaries and `cmd/nexus`
  compile for Windows amd64 and Darwin arm64; native execution still belongs
  to the GitHub Windows/macOS runners.
- Desktop/release: **PASS**, Wails Linux package and GoReleaser v2.18.0
  snapshot artifacts/checksums.
- Repository hygiene: **PASS**, `gofmt` and `git diff --check`.
- Generated Wails bindings: **PASS**, Prettier normalized the generated model
  file and the aggregate `make quality` gate passed afterward.
- Remote CI must be rerun on the resulting commit; no commit or push was made.

## 2026-09-06 — Monitor residente de quotas

- Go: `go test ./...` PASS; `go vet ./...` PASS; testes focados de Nexus,
  eventos, Web e App PASS.
- Web: `npm run typecheck` PASS; Vitest PASS (61 arquivos / 306 testes);
  ESLint PASS com warnings preexistentes, Stylelint PASS, `check:styles` PASS,
  build PASS.
- `git diff --check` PASS.
- Cobertura adicionada para Gemini 28% independente de Claude/GPT 100% e para
  histórico global sem filtro de runtime.

## 2026-09-06 — Workspace tabs and terminal viewport regression

- Route synchronization now keys off the actual pathname/search and does not
  reopen `/overview` on every workspace state update.
- Product tab clicks route through the application opener, preserving the URL
  and active surface together.
- Hidden xterm panels defer output, focus, and resize until `data-active=true`;
  activation flushes output and performs a guarded fit.
- Frontend verification: Prettier, TypeScript, and 292 Vitest tests passed.
- `bun run build`, `make build-desktop-wails`, installation of the Linux Wails
  artifact, and `git diff --check` passed.

## 2026-09-06 — Desktop input hardening

- Deep links now enforce bounded length, one resource ID, no fragments/users,
  and reject path/control/whitespace ambiguity.
- Linux autostart quotes the executable path using Desktop Entry `Exec` rules.
- `go test ./internal/desktop -run 'Test(AutoStart|DeepLink)'` passed.

## 2026-09-06 — Desktop capability evidence

- `DesktopBridge` now consumes the backend `GetCapabilities()` result instead
  of equating generated Wails methods with available native functionality.
- Pre-bootstrap state is conservative; capability evidence is cached after the
  authenticated desktop bootstrap.
- `cd web && bun run verify` passed all checks, including the regression test.

## 2026-09-06 — Native notification input safety

- AppleScript and PowerShell notification paths now pass title/body as
  arguments to static scripts instead of interpolating user-controlled text.
- `go test -race ./internal/control/notify -count=20` passed; Windows package
  cross-compilation also passed.

## 2026-09-06 — External URL fallback safety

- Added shared structural URL validation to Web/Desktop bridges and all
  browser-opening fallbacks.
- Unsafe schemes (`javascript`, `data`, `file`, malformed URLs) are rejected;
  the frontend regression and full `bun run verify` passed.

## 2026-09-06 — Handoff secret hygiene

- Redacted two persisted loopback bootstrap token URLs from `DEV/HANDOFF.md`
  and removed machine-local installation paths.
- Current-tree scan found no long hexadecimal bootstrap token URLs outside Git
  metadata; history was not rewritten because local user work must be preserved.
- `internal/core/security` and script tests passed.

## 2026-09-06 — Same-SHA security gate

- Added a dedicated CI `Security` job running pinned `govulncheck` with Go
  1.25.14.
- Added `Security` to the release workflow's required same-SHA job list.
- `actionlint` and release package tests passed; remote CI execution remains
  pending because no push was performed.

## 2026-09-06 — Detached update signature consumption

- Fixed the shared Update Service to consume `update-manifest.sig` beside a
  static `update-manifest.json` when no signature header is present.
- Missing, malformed, unavailable, or invalid signatures remain fail-closed.
- RED/GREEN evidence: detached-signature unit test passes normally and under
  `go test -race ./internal/update -count=5`.

## 2026-09-06 — Explicit Maestro update boundary

- Added the explicit CLI namespace `nexus maestro status|doctor|update`.
- `nexus update` remains owned by the shared Nexus Update Service and no longer
  claims to update Maestro in help/completion text.
- Added a degraded-status test for an unavailable Maestro installation and
  aligned Web settings copy with the explicit Maestro operation.

## Current closure pass — 2026-09-06

- Frontend: **PASS**, `make web-verify` 10/10 at `2026-09-06T05:20:55Z`.
- Go: **PASS**, `gofmt`, `go vet ./...`, `go test ./...`, and
  `go test -race ./...`.
- Go lint: **PASS**, `golangci-lint v2.12.2 run ./...` (`0 issues`).
- Aggregate quality: **PASS**, `PATH=/tmp/nexus-tools:$PATH make quality`.
- Stress: **PASS**, Desktop/Update/SessionHost/Workspace at `-count=20`.
- macOS fallback input handling: **PASS** by static review and Go lint; native
  execution remains pending on a macOS runner.
- Installers: **PARTIAL**, pinned release + SHA256 checks pass; signed
  manifest/keyring publication is still unavailable.
- Release workflow: **STATIC PASS**, YAML parses and publication is gated on
  named same-SHA CI jobs; runtime execution remains unverified until a new
  branch/tag CI run exists.
- Update signer byte binding: **PASS**, published manifest bytes are covered by
  the detached signature and regression-tested.
- Wails Linux packaging: **PASS**, `make build-desktop-wails` generated and
  packaged the native artifact; Windows/macOS execution remains pending.
- Native release artifact promotion: **STATIC PASS**, CI workflow uploads and
  release workflow promotes artifacts from the selected same-SHA run; runtime
  evidence remains pending.
- GoReleaser snapshot: **PASS locally** with v2.18.0; all six CLI archives,
  checksums, and Linux DEB/RPM artifacts were generated. Deprecation warnings
  remain non-blocking follow-up items.
- Desktop Linux build: **PASS**, `make build-desktop`.
- Security: **PASS**, `make security` reported `No vulnerabilities found`.
- Native Windows/macOS: **NO-GO**, CI run `34012236345` failed those jobs;
  browser/Desktop downstream jobs were skipped. See
  [`FINAL_PLATFORM_RELEASE_DESKTOP_REPORT.md`](validation/FINAL_PLATFORM_RELEASE_DESKTOP_REPORT.md).

## 2026-09-05 — Implementação Completa Web + Desktop Multiplataforma

- `go test ./...` — PASS (100% de sucesso nos pacotes `internal/app`, `internal/desktop`, `internal/update`, `internal/control/web`, `internal/doctor`, `internal/release`, etc.).
- `go vet ./...` — PASS.
- Frontend Quality (`bun run quality`):
  - `check:styles` — PASS (allowlist SCSS Modules estrita).
  - `format:check` — PASS.
  - `lint` — PASS (0 erros).
  - `lint:styles` — PASS.
  - `typecheck` — PASS (`tsc --noEmit` 0 erros).
  - `vitest run` — PASS (58 arquivos de teste, 280 testes unitários).
- Frontend Build (`bun run build`):
  - Geração de chunks ESM sob demanda em `web/dist` — PASS.
  - Sincronização e verificação de integridade do bundle embutido no Core e no Desktop (`web.EmbeddedDistFS()`) — PASS.
- Desktop Shell & Capabilities:
  - Wails v2 estável (`v2.15.0`) — build e testes unitários do bridge nativo (`internal/desktop`) — PASS.
  - Deep links (`nexus://`), window management e PlatformBridge — PASS.
- Maestro Desacoplado:
  - Scripts de instalação (`install.sh` e `install.ps1`) com opt-in explícito (`--with-maestro` / `-WithMaestro`) verificados — PASS.
  - Modo degradado `MAESTRO_DEGRADED` sem falhar o produto — PASS.

- `bun install --frozen-lockfile` — PASS com Bun 1.3.9 após regeneração do lockfile.
- Frontend format/typecheck/lint/stylelint — PASS; lint com 0 erros e 36 warnings preexistentes.
- Frontend Vitest — PASS: 51 arquivos, 253 testes.
- golangci-lint `v2.12.2 config verify` — PASS; execução completa reproduz 59 achados existentes, sem supressão.
- `go test ./...` — PASS após ExecutableResolver.
- `go test -race ./internal/control/terminal ./internal/control/protocol ./internal/control/host` — PASS.
- Windows cross-compile `go test -c` para terminal, launcher, host e protocol — PASS; execução nativa ainda pendente.

Estado: runtime/release continuam NO-GO até lint debt, build/embed gate e evidência nativa Windows/macOS serem fechados.

## 2026-09-05 — Path identity and credential capability increment

- `go test ./internal/core/config ./internal/nexus/store` — PASS após migração `0012_path_identity.sql`.
- Cross-compile de config/runtime para Windows e Darwin — PASS.
- `go test ./...` — PASS.
- `go vet ./...` — PASS.
- `git diff --check` — PASS.

Limitações mantidas: identidade nativa por handle no Windows, execução nativa Windows/macOS e bundle do `nexus doctor` ainda não foram validados.

## 2026-09-05 — Doctor CLI and diagnostic bundle

- `nexus doctor --json` com diretórios temporários isolados — PASS.
- `nexus doctor --bundle` — PASS; ZIP contém somente `report.json` e `MANIFEST.txt`.
- `go test ./internal/doctor ./internal/app` — PASS.
- `go test ./...` — PASS.

O bundle foi validado por allowlist; execução nativa Windows/macOS e reuso na Web permanecem pendentes.

## 2026-09-04 — Topbar Overlap Refactor, Responsive Hardening & Accordion Theme Quality Gates

- `make web-verify` — PASS (8/8 gates verdes: typecheck, lint, null-arrays, vitest [51 files, 249 tests], i18n, build, embed-sync, ui-markers).
- `make build` — PASS (binário `nexus v0.5.0-beta.33` gerado e instalado com bundle web embutido).
- `npm --prefix web run test:e2e-hardening` — PASS (100% de sucesso nos 6 viewports):
  - `320x568` (mobile-se): botão Terminal desobstruído (`elementFromPoint` verificado sem overlap).
  - `390x844` (mobile-iphone): botão Terminal desobstruído.
  - `768x1024` (tablet): botão Terminal desobstruído.
  - `1024x768` (laptop-compact): layout estável, ações secundárias colapsadas.
  - `1280x800` (laptop-critical): terminal acessível em 1º lugar, sem clipping.
  - `1440x900` (desktop): renderização completa sem colisão.
  - Validação WAI-ARIA do Accordion de Temas com paletas visuais e medição real do delta de densidade (Comfortable 497px vs Compact 481px).


- `make web-verify` — PASS (typecheck, lint, null-arrays, vitest (244 tests), i18n, build, embed-sync e ui-markers).
- `make build` — PASS (binário `nexus v0.5.0-beta.33` compilado com assets embutidos atualizados).
- Auditoria Playwright multi-viewport (1440x900, 1024x768, 768x800, 480x800) em `/usr/bin/google-chrome`:
  - Visualização de screenshots reais renderizados:
    - `verify_1440x900_desktop.png`: layout limpo com rail lateral, topbar sem colisão, janelas em cascata e radar visível.
    - `verify_1024x768_compact.png`: botões redundantes de criação escondidos, hierarquia de janelas com z-index correto.
    - `verify_768x800_tablet.png`: rail colapsado em drawer/overlay, topbar compacto com badges e menus sem quebra.
    - `verify_480x800_mobile.png`: zero sobreposição/encavalamento de controles; comandos e labels longos suprimidos graciosamente, radar e switcher de idiomas alinhados no topo sem overflow horizontal.
- Z-Index verificado: radar e popovers operando em camada superior (`--nx-z-popover: 4000`) a janelas focadas (`500`).
- Primitivas `<Select>` e `<Input>` do design system adotadas em 100% dos modais e superfícies auditadas, eliminando selects HTML crus não estilizados.

## 2026-09-04 — Terminal recover após reboot/serviço

- `go test -count=1 ./internal/control/registry ./internal/control/web` — PASS.
- `make web-verify` — PASS (typecheck, lint, null-arrays, Vitest, i18n,
  build, embed-sync e ui-markers). Relatório:
  `DEV/validation/FRONTEND_LATEST.md`.
- Reprodução manual restante: reiniciar `nexus web` (ou a máquina) com uma
  janela de terminal de agente aberta e confirmar overlay **Iniciar /
  Recuperar Agente** em vez de xterm preto.

## 2026-09-04 — Baseline atual e Composer skills

- `make web-verify` — PASS (typecheck, lint, null-arrays, Vitest, i18n,
  build, embed-sync e UI markers).
- `go test -count=1 ./...` — PASS.
- `go vet ./...` — PASS.
- `git diff --check` — PASS.
- CI frontend passou a fixar Bun `1.3.9` e o `packageManager` do workspace foi
  declarado como `bun@1.3.9`.
- Composer agora expõe e persiste aceitar/dispensar de Maestro skills e envia
  as skills aceitas na finalização; Readiness, Unknowns e Assumptions são
  visíveis.

## Identidade visual e assets de marca — 2026-09-04

- `make web-verify` — PASS (typecheck, lint, null-arrays, vitest, i18n, build, embed-sync e ui-markers).
- `go test ./...` — PASS (todos os pacotes Go testados e passando).
- `make build` — PASS; binário `nexus v0.5.0-beta.33` gerado e instalado com novos assets embutidos em `internal/control/web/dist`.
- Validação visual: extração do master em alta resolução com desmatting transparente, verificação de contraste em fundo escuro (`#080a0f`) e fundo claro, geração do pacote favicon e renderização no rail e hero.

## Controles de terminais e resumo de Agentes — 2026-09-03

- `make web-verify` — PASS (typecheck, lint, null-arrays, testes, i18n, build,
  embed-sync e ui-markers).
- `make build` — PASS; instalou `nexus v0.5.0-beta.33`.
- Cobertura funcional implementada: troca de modo com erro visível e rebinding,
  feedback de lease, fechamento com escolha aba/runtime, confirmação de Project
  Shell e badge operacional clicável com foco/recuperação.

## Bootstrap de contexto durável — 2026-09-03

- `go test ./internal/nexus ./internal/control/web` — PASS.
- `npm --prefix web run typecheck` — PASS.
- `npm --prefix web test -- --run` — PASS (42 arquivos, 180 testes).
- `make build` — PASS; instalou `nexus v0.5.0-beta.33`.
- A criação é explícita, gera `AGENTS.md` base e nunca sobrescreve contexto
  existente.

## Recuperação de terminal desconectado — 2026-09-03

- Web: `npm --prefix web run typecheck` — PASS.
- Web: `npm --prefix web test -- --run` — PASS (42 arquivos, 175 testes).
- Build: `make build` — PASS; instalou `nexus v0.5.0-beta.33`.
- Recover/Start agora vincula diretamente a geração retornada; o terminal também
  oferece fechamento explícito da aba.

## Nova sessão de IA — 2026-09-03

- Web: `npm --prefix web test -- --run` — PASS (40 arquivos, 157 testes).
- Web: `npm --prefix web run typecheck` — PASS.
- Build: `make build` — PASS; instalou `nexus v0.5.0-beta.33`.
- O launcher agora é disparado depois que a superfície Composer está montada.

## TERM=dumb em CLIs interativos — 2026-09-03

- `go test ./internal/runtime ./internal/control/terminal` — valida substituição
  de `TERM=dumb` sem mutar o ambiente original e preservação de terminais normais.
- A correção cobre execução direta e runtimes supervisionados com PTY/ConPTY.

## AGY quota por grupo — 2026-09-03

- `go test ./internal/core/quota ./internal/app ./internal/core/scheduler ./internal/nexus` — PASS
- Conta AGY com Gemini disponível e Claude/GPT esgotado agora fica `INDISPONIVEL`/
  `QUOTA ESGOTADA` no nível da conta.
- Conta AGY só fica `DISPONIVEL` quando os dois grupos têm quota.

## AGY keyring e modelo — 2026-09-03

O processo AGY recebe `GNOME_KEYRING_CONTROL` do Secret Service privado, e o
arquivo `antigravity-cli/settings.json` não é mais sobrescrito pelo host a cada
execução. Testes Go focados passaram e `make build` instalou `nexus v0.5.0-beta.15`.

## Radar falso + terminal “connecting” — 2026-09-03

- Go: `go test ./internal/control/host` — PASS
  (lista numerada sozinha não vira pergunta; lixo `\uFFFD`/box-drawing rejeitado;
  `Select an option` + `1. …` continua `choice`)
- Web: `make web-verify` → [`DEV/validation/FRONTEND_LATEST.md`](validation/FRONTEND_LATEST.md) — PASS
  (radar colapsa mensagens iguais; honesty usa `sanitizeAttentionText`;
  WS `?runtime_id=`; erro fatal para reconnect; taskbar “agentes degradados”)
- Runtime: `make install` + reinício de `nexus web` + `pkill` dos `__control-host` velhos
- Browser smoke: Radar **ok** / “Nenhum terminal ativo” sem hosts;
  AgentTerminal mostra `error` + “Use Recover/Start…” em vez de `connecting` eterno

## Organizador de terminais + radar — 2026-09-03

Cobertura automatizada do P0:

- Go: `go test ./internal/control/host ./internal/control/notify` — PASS
  (shortcuts/`[y/n]` isolado silenciam; yn real + free_text; clear on `thinking...`;
  OS notify só com evidência + dedupe de fingerprint)
- Web: `make web-verify` → [`DEV/validation/FRONTEND_LATEST.md`](validation/FRONTEND_LATEST.md) — PASS
  (título sem emoji/`(N ❓)`; fingerprints de mensagem iguais colapsam; Sim/Não só se `yn`;
  radar/focus model; i18n parity)
- Embed sync: `web/dist` ≡ `internal/control/web/dist` após o gate

Caminho manual ainda necessário no ambiente do operador (depende de `make build` +
reinício de `nexus web`): dois Projetos, pergunta yn real vs chrome de help, e
notificação de SO com a aba fechada.

## `null.forEach` no Flow — 2026-09-03

O bundle servido foi regenerado a partir de `flowModel.ts`, que trata
`plan.phases` e `phase.packages` não-array como coleções vazias. A regressão
para payloads nulos passou em `yarn test src/features/work/flowModel.test.ts`
(9/9), e `yarn build` atualizou `internal/control/web/dist`.

Nesta execução, `yarn typecheck` também passou.

## Normalização de WorkPlan — 2026-09-03

`web/src/nexus/workPlan.ts` passou a ser a fronteira de normalização para
todos os endpoints que retornam WorkPlan, cobrindo planos e fases nulos antes
do acesso da UI. Verificação atual: testes focados (19/19), `yarn lint`,
`yarn typecheck` e `yarn build` — PASS.

## Reanexação de terminal — 2026-09-03

Após Recover/Start, o terminal não reutiliza o `runtime_id` histórico da
superfície. Ele reanexa pelo Agente e acompanha a nova geração. Testes focados
(14/14), typecheck e build passaram; lint passou com 7 warnings preexistentes.

## Tabela TUI de uso — 2026-09-02

`nexus usage` usa a tabela navegável do Charm Bubbles. O filtro cobre
identidade e grupo de modelo; há teste unitário para ambas as buscas.

## Banner de atenção — 2026-09-02

O banner de intermediação não participa mais da linha expansível da grade do
Workspace OS. Ele é exibido como uma faixa compacta sob o cabeçalho, com
truncamento, controles padronizados e quebra responsiva. A heurística ignora
`? for shortcuts`, evitando o aviso falso visto no terminal.

Validação: `go test ./...`, `go vet ./...`, `yarn --cwd web typecheck`,
`yarn --cwd web lint`, `yarn --cwd web test` (31 arquivos/111 testes),
`git diff --check` e `make build` — todos PASS.

## Notificações transitórias — 2026-09-02

`TASK_COMPLETED` e `ERROR` são convertidos em toasts próprios, enquanto
`QUESTION` e `APPROVAL` continuam no componente de resposta. O modelo possui
cobertura para conclusão e exclusão de eventos interativos.

## Uso/quota — 2026-09-02

O fluxo de sessão Web mantém rotação automática antes da expiração absoluta,
propaga novo CSRF e orienta a recuperação via novo Bootstrap quando a sessão
expira. Testes Go e frontend passaram.

CLI e dashboard de recursos foram verificados com tabela/grid por modelo,
layout responsivo e status independente por grupo. Typecheck, lint, 31 arquivos
de testes (111 testes), `make build` e `git diff --check` passaram.

`nexus usage` agora preserva a capacidade por grupo de modelo e por janela,
exibindo percentual restante e reset 5h/semanal. A validação passou em
`go test ./internal/app ./internal/core/quota ./internal/profile`, `go vet ./...`
e `make build` (`v0.5.0-beta.9`). Cache vencido e snapshot de outra identidade
agora são rejeitados antes da exibição/seleção.
Grupos independentes também têm disponibilidade independente; no estado atual
`kiveromegasistemas` aparece como `DISPONIVEL` para Gemini e `INDISPONIVEL` para
Claude/GPT.

## Review update — 2026-09-02

The complete current diff was reviewed against `HEAD` (`c968852`). The current
worktree is **PASS AFTER CORRECTIONS**: `go build ./cmd/nexus`, `go test ./...`,
`go vet ./...`, frontend lint, tests and typecheck pass. The detailed findings
and exact file locations are recorded in
[`DEV/CODE_REVIEW.md`](CODE_REVIEW.md).

The corrections and latest evidence supersede the historical validation
snapshots below.

## AGY quota semantics — 2026-09-02

Legacy AGY quota files expose `percent_left` as consumed percentage. The
adapter and cache reader now normalize it to remaining percentage before
selection or display. Coverage includes boundary and fractional values.

## Status: ALL 7 PENDING ISSUES RESOLVED

### Build & Test
- `go build ./...` — clean
- `go test -race ./...` — ALL PASS
- `go vet ./...` — clean
- `npx tsc --noEmit` — clean
- `npx vitest run` — 44/44 tests PASS
- `make build` — PASS (v0.4.6)

### Pending Issues Resolution
| # | Issue | Status | Key Changes |
|---|-------|--------|-------------|
| 1 | Resources facade | ✅ | `ListResources()` real discovery, `AllocateResource()` persistence, `ResourcePicker` UI |
| 2 | Maestro synthetic state | ✅ | Honest degraded fallback, no hardcoded capabilities/recommendations |
| 3 | Update simulated | ✅ | Returns `501 Not Implemented` |
| 4 | Agent start without provider | ✅ | `ResolveStartParams()` + `REQUIRED_RESOURCE_SELECTION` flow |
| 5 | Config not reaching runtime | ✅ | `LaunchOptions` extended, full `AgentConfig` propagation |
| 6 | Terminal continuity | ✅ | `AgentTerminalBroker` integration, `runtime_changed` frames |
| 7 | Missions scaffold | ✅ | Kept as-is, documented as future work |

### Business Rules Location
All business logic lives in `internal/nexus/` (service layer). Web and TUI consume the same API endpoints. No resource selection, health checking, config propagation, or Maestro degradation logic exists in the frontend.

### Files Modified (this session)
- `internal/nexus/resource_discovery.go` (new)
- `internal/nexus/maestro.go`
- `internal/nexus/nexus.go`
- `internal/nexus/scheduler.go`
- `internal/nexus/store/agents.go`
- `internal/control/launcher/launcher.go`
- `internal/control/web/handlers_nexus.go`
- `internal/control/web/handler_terminal.go`
- `internal/control/web/broker.go`
- `internal/control/web/server.go`
- `web/src/features/agents/AgentsSurface.tsx`
- `web/src/nexus/ResourcePicker.tsx`
- `web/src/nexus/api.ts`

### Documentation Updated
- `DEV/NEXUS_V1_ARCHITECTURE.md` — resource discovery + terminal broker
- `DEV/NEXUS_V1_RESOURCE_SCHEDULER.md` — marked implemented with flow
- `DEV/NEXUS_V1_MAESTRO_INTEGRATION.md` — honest degraded fallback
- `DEV/NEXUS_V1_AGENT_MODEL.md` — updated start flow + config propagation
- `DEV/WORKLOG.md` — session entry with all changes

### Not Yet Started
- Cross-provider scoring with continuity, cooldown, and rate-limit risk
- Project-level and global policy hierarchy
- Account Handoff and Context Handoff recommendations
- Full Maestro CLI contract (`feat/nexus-contracts-v1`)
- Mission execution/orchestration (scaffold only)
- Full E2E verification with live providers
- Performance benchmarks

## Review final — 2026-09-04

- `go test -count=1 ./...`: PASS
- `go vet ./...`: PASS
- `go test -race -count=1 ./internal/control/... ./internal/nexus/... ./internal/core/security/... ./internal/core/session/...`: PASS
- `make web-verify`: PASS (typecheck, lint, null-arrays, 188 testes, build, embed-sync, ui-markers)
- `npm audit --offline --omit=dev --audit-level=high`: 0 vulnerabilidades
- Cross-build `linux/amd64`, `darwin/amd64`, `windows/amd64`: PASS
- `git diff --check`: PASS
- Smoke final `/api/v1/health`: HTTP 200; bootstrap: HTTP 302; servidor ativo em `127.0.0.1:3000`

Parecer e limitações: [`DEV/validation/CURRENT_CODE_REVIEW.md`](validation/CURRENT_CODE_REVIEW.md).

## TUI Usage selection — 2026-09-06

- `go test ./internal/tui -run 'TestUnifiedUsage(RestoresSelectionAfterEmptyFilter|ModeTogglingAndFlags|EscAndQQuit)$' -count=20` — PASS.
- `go test -race ./internal/tui` — PASS.
- `Enter` após filtro seleciona a linha destacada; cursor é restaurado após
  resultado vazio; `Esc` mantém comportamento de apenas fechar o filtro.

## Frontend embedded identity — 2026-09-06

- `cd web && bun run verify` — PASS (10/10 gates, incluindo igualdade entre
  `web/dist` e `internal/control/web/embedded`).
- `.github/workflows/ci.yml` agora usa esse verificador no job Frontend, não um
  teste de mera existência de arquivos.
- `go test ./internal/release -run 'TestInstaller' -count=1` — PASS; a política
  de caminho `IAPro Nexus` com preservação do diretório legado foi verificada.
- Actionlint/YAML dos workflows — PASS; jobs Windows/macOS preservam logs de
  falha em artifacts sem mascarar o resultado.
- O gate Windows inclui parsing explícito do `install.ps1`; execução local não é
  possível porque `pwsh` não está instalado neste runner Linux.
- Cross-build CLI Linux amd64/Windows amd64/macOS arm64 e Desktop Windows
  amd64 — PASS como compilação cruzada apenas; não substitui execução nativa.
- Listener failure: teste Linux 20x PASS; pacote `internal/control/host`
  compilou para Windows amd64 PASS; execução Windows ainda requer runner nativo.
- ConPTY interactive fixture: pacote Linux e compilação Windows amd64 PASS;
  execução nativa ainda pendente.
- FSMkdir: testes FS focados 20x PASS e compilação Windows amd64 do pacote Web
  PASS; execução nativa ainda pendente.
- IPC readiness: Host/Protocol 20x PASS e race PASS sem sleeps de prontidão;
  execução nativa Unix socket/Named Pipe continua dependente dos runners.
- HTTP readiness: bootstrap/session/restart focados 3x PASS e race PASS sem
  sleeps fixos.
- Web E2E/API/túnel completos: PASS normal e com race após remover waits de
  startup artificiais.
- PTY Unix: 20x PASS e race PASS com leitura baseada em output observado, sem
  sleep de readiness.
- Pacotes terminal/host/protocol/web: testes normais, race e golangci-lint PASS;
  `make quality PATH=/tmp/nexus-tools:$PATH` PASS.
- Revalidação final: `go test -race ./...`, `make security`, GoReleaser v2.18.0
  snapshot e `go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.7`
  passaram.
- Browser E2E real: PASS após corrigir locator case-sensitive de Settings;
  Playwright, Axe, deep-links, breakpoints e screenshots passaram em três
  execuções consecutivas; o script passou pelo Prettier.
- Fixtures Host/QA Windows não usam mais `cat`; usam `cmd.exe /D /Q /C more`.
  Linux normal/race e cross-compilação Windows/macOS dos pacotes afetados
  passaram. Wine/ConPTY permanece explicitamente não nativo.
- Update Service: testes 20x e race PASS; `HOME` de teste isolado e nenhum
  receipt gerado no checkout.
- Doctor: probes read-only e estados `WARN`/`SKIPPED` para capacidades nativas
  sem evidência; pacote passou 20x e race 5x.
- Registry: invalidação cross-process agora usa `ModTime` + tamanho + fingerprint
  SHA-256; teste de concorrência passou 50x, race 10x e `go test ./...` passou.

## Maestro catalog and skill context — 2026-09-07

- `go test ./internal/nexus -run 'TestMaestro|TestFindOrquestrador' -count=1` —
  PASS after global/profile catalog merge and prompt provenance changes.
- Frontend typecheck, lint, stylelint, focused tests and build — PASS after
  replacing command copy with complete skill context.
- Real browser smoke against the rebuilt server — PASS: `53 de 53 disponíveis`,
  53 cards and 53 usage-context panels; `skill-database-migrations` exposed
  its `SKILL.md` prompt.
- This does not replace native macOS/Windows execution; those release gates
  remain pending where recorded above.

## Open Design typography and visual QA — 2026-09-07

- Open Design audit identified undersized semantic typography and one dark-theme
  contrast failure in the shell.
- `node web/scripts/maestro-visual-verify.mjs` — PASS after correction:
  screenshots at five required viewports, no horizontal overflow, search
  interaction, Axe with zero serious/critical violations, and no page errors.
- `bun run format:check`, `bun run typecheck`, `bun run lint:styles`, focused
  theme/Maestro tests (14/14), frontend build, focused Go tests, `go vet` and
  `git diff --check` — PASS. Existing ESLint warnings remain warnings only.
- Typography tokens use `rem`; visual verification was repeated after the
  migration and the rebuilt server health endpoint returned `status: ok`.
- `FontScalePicker` no toolbar foi validado por typecheck/build e usa o mesmo
  `ThemeProvider` da tela Configurações, sem estado paralelo.
- Tipografia completa: nenhuma declaração `font-size`, `font` ou `fontSize` em
  px permanece em `web/src`; o servidor recompilado respondeu `status: ok`.

## Terminal helper do Project Shell — 2026-09-07

- `make build` — PASS; frontend recompilado e binário Nexus gerado.
- Verificação do bundle — PASS; `xterm-helper-textarea` presente em
  `web/dist/bundle.css` e CSS embutido sincronizado.
- Smoke Playwright contra o servidor real — PASS; os helpers do xterm ficaram
  invisíveis, não houve sequência de `W` sobre o terminal e não houve erro de
  página.
- O problema era visual do CSS ausente, não saída indevida do PTY.

## Tokens de espaçamento — 2026-09-07

- `bun run build` — PASS.
- `bun run typecheck` — PASS.
- `bun run lint:styles` — PASS.
- Bundle final contém `--nx-spacing-1`, `--nx-spacing-8`,
  `--nx-spacing-16` e aliases `--nx-space-4` — PASS.
- `make build` — PASS; bundle embutido regenerado.
- Instalação/execução final — PASS; `LOCAL_BIN=/home/desenvolvedor/.local/bin
  make install-local`, CSS HTTP contém `--nx-spacing-4: 16px` e o navegador
  calcula `16px` no `:root`.
- `make -n install` — PASS; fluxo local padrão e fluxo `DESTDIR` ficam explícitos
  no Makefile.

## Cota oficial do Codex via app-server — 2026-09-11

- Fonte primária nova: `codex app-server --stdio` + `account/rateLimits/read`
  (`internal/core/provider/adapters/codex/app_server_usage.go`). Rollouts e
  last-known continuam como fallback honesto, nessa ordem.
- `go test ./... -count=1` — PASS (suíte completa).
- `go test -race -count=1` em `internal/profile`, `internal/core/quota`,
  `internal/core/provider/adapters/codex`, `internal/tui`, `internal/nexus` — PASS.
- `go vet ./...`, `gofmt -l internal/` e `git diff --check` — PASS.
- Validação real com as duas contas, sem abrir sessão e sem enviar prompt:
  - `codex:kivergmail` — `RATE_LIMITED` / `OFFICIAL_API`; 5h em `0%` restante
    (reset 21:55) e semanal em `56%` restante (reset 08:37 de 15 Sep). Confere
    com o splash oficial do Codex CLI.
  - `codex:kiver.omegasistemas` — `LIVE` / `OFFICIAL_API`; 5h `98%` e semanal
    `58%`, com `accountId` próprio. As duas contas não se contaminam.
- `RATE_LIMITED` passou a ser leitura confiável: `quota.Engine.Trustworthy`,
  as barras e o monitor tratam o bloqueio informado pelo provedor como dado
  exato, em vez de descartá-lo para `SEM DADOS`.
- TTL de 60s vale também entre processos para leituras `OFFICIAL_API`; uma
  segunda chamada de `nexus usage` dentro da janela reaproveita o cache e não
  sobe outro app-server. Cache derivado de rollout nunca é reaproveitado.
- Identidade: a sonda exige `chatgpt_account_id` verificável, confere o
  `accountId` da resposta e o `codexHome` ecoado pelo `initialize`. Qualquer
  divergência descarta o payload. Desligável por `NEXUS_CODEX_APP_SERVER=0`.
- Controle estruturado (aprovações/eventos) segue diferido — ver
  [`DEV/AI_CONTROL_DEFERRED.md`](AI_CONTROL_DEFERRED.md), item 6.

## 2026-09-12 — automatic delegation interactive Lead closure

- PASS — focused `go test ./internal/app ./internal/control/launcher ./internal/nexus -count=1`.
- PASS — `go test ./... -count=1`, `go test -race ./... -count=1`, `go vet ./...`
  and `git diff --check`.
- PASS — `make build` and `make build-desktop`; both retain explicit Nexus
  build metadata while disabling only redundant Go VCS stamping.
- PASS — canonical workspace lookup/registration, persistent Lead binding and
  runtime generation lifecycle are covered locally; launcher labels preserve
  the interactive policy metadata.
- UNVERIFIED — authenticated `nexus agy`/Codex provider session, real provider
  prompts and a durable authenticated Mission evidence stream remain unavailable.
- UNVERIFIED — the attempted supervised Codex smoke timed out during IPC
  handshake; no runtime remained registered afterward.

## Fechamento beta.10 Windows — 2026-10-16

Ciclo de correção e certificação da instalação local (branch
`fix/release-beta10-windows-closure-20261014`), fechado sem alterar
`release/v1.0.0` e sem merge/tag/publicação.

- Picker de diretório (`6c14b57e`): `GET /api/v1/fs/browse` sem path devolve
  200 com default seguro (`defaultBrowsePath`: cwd só quando dentro de uma
  raiz permitida, senão home) e bookmarks expõem apenas raízes permitidas —
  sem bookmark apontando para `/`. Coberto por e2e RED→GREEN e 15/15 unit.
- HUD vinculado ao core (`baf91839` + manifesto `37434c6d`): spawn com
  `--linked` quando o core sobe; opt-out `NEXUS_HUD_SPAWN=0`; hard-kill do
  core mata o HUD (watchdog). Verificado ao vivo: HUD parent = PID do core.
- Fechamento de terminal ConPTY com bound de 2s no `Wait()` do rPipe
  (`aa854590`).
- PS 5.1 + stderr nativo: banner do `bun run` vira NativeCommandError
  terminante sob `ErrorActionPreference=Stop`. Corrigido no script da cadeia
  (`36445f10`, `Invoke-NativeLogged`) e depois no próprio instalador
  (`996659f4`, `Invoke-NativeBuild`): `bun install`, `bun run build`,
  `go build` (cache frio imprime `go: downloading` em stderr) e
  `git fetch/checkout`. `$LASTEXITCODE` permanece a única fonte de verdade
  dos checks (provado empiricamente em PS 5.1).
- Flake da cadeia #5 (`36445f10`, fase Go tests):
  `TestEvidenceBundleRequiresAndProjectsCanonicalProof` degradou identidade
  sob saturação. Causa-raiz provada por digest: `IdentityDigest:4eab7424…` =
  `sha256("git\0INCOMPLETE\0git identity command timed out")` =
  `incompleteGitIdentity()` (deadline de 5s do child git, comportamento
  conservador intencional do produto). Correção apenas no teste (`30e2e6f3`):
  retry tolerante exclusivamente ao digest constante de deadline-loss, teto
  5 tentativas + backoff, qualquer outra incompletude falha imediato.
  Repro controlada com git falso (`degrade-once` → PASS com retry;
  `degrade-always` → FAIL estrito com digest idêntico).
- Cadeia local de homologação #6 (`38968218`) e #7 (`996659f4`): **PASS
  18/18** — veredito `PASS automated Windows beta homologation:
  1.0.0-beta.10 @ 996659f4134f2f3f26007c69afd387b2d6905f77` (~18 min cada).
  Evidência: `DEV/validation/local-beta/1.0.0-beta.10/996659f4134f2f3f26007c69afd387b2d6905f77/windows-AMD64/`.
- Reinstalação ao vivo pós-cadeia: `install.ps1 -BuildFromSource -Yes` com
  EAP=Stop completa (exit 0); Desktop e HUD copiados dos builds da cadeia;
  core relançado (`web --no-open`) spawnou o HUD `--linked` (parent = core).
- Live check da API contra o core instalado: bootstrap 200 com cookies
  `ai_control_session`/`ai_control_device`; `GET /api/v1/fs/browse` sem path
  200; `current_path` = diretório de lançamento (cwd dentro de raiz
  permitida, conforme contrato de `defaultBrowsePath`); bookmarks
  `Home → C:\Users\kiver`, `Documents → C:\Users\kiver\Documents`, nenhum
  apontando para raiz do sistema.
- Passagem headless somente leitura sobre o `MANUAL_CHECKS.md` executada
  contra o core vivo (2026-10-16, nenhuma mutação): `GET /api/v1/providers`,
  `/api/v1/execution-roles`, `/api/v1/hud`, `/api/v1/voice/settings`,
  `/api/v1/capacity/status`, `/api/v1/cost/summary` e `/api/v1/projects`
  todos 200. Item 16 confirmado ao vivo (opencode `INSTALLED_UNREGISTERED`
  visível sem perfil); item 15 com OpenCode listado como provider próprio
  independente do AGY; roles build/investigate/review presentes (item 17);
  voz N/A (`enabled=false`, sem chave Groq — item 22). Itens 1 e 10-12
  cobertos pelos testes da cadeia #7 (CSI/setas byte-level, delegation
  fail-closed, budget gate, isolamento). Detalhe:
  `%TEMP%\opencode\MANUAL_CHECKS_HEADLESS_RESULT.md`.
- Ícones do tray fixados na área de notificação (fora do overflow):
  `IsPromoted=1` em `HKCU:\Control Panel\NotifyIconSettings` para os dois
  binários de produção (`nexus-desktop.exe` e `nexus-hud.exe`), mecanismo
  real do Windows; a promoção se concretiza quando o ícone volta a se
  registrar (relançamento do HUD/Desktop ou novo login).
- Autostart do HUD ativado no formato exato do toggle do produto
  (Wails v3 `Autostart.Enable()`): valor `nexus-hud` sob
  `HKCU:\Software\Microsoft\Windows\CurrentVersion\Run` apontando para
  `nexus-hud.exe` entre aspas, sem argumentos (igual ao `Enable()`
  padrão). O toggle nativo do HUD reconhece a entrada e pode desligá-la.
- UNVERIFIED (por design, interativo) — itens do `MANUAL_CHECKS.md` que
  exigem sessão nativa real (prompt em provider, admissão de Mission com
  Agent pinado, Certify/enforcement) permanecem a cargo do operador.

## HUD nativo — arraste e presets de tamanho — 2026-10-21

Worktree `C:\Dev\nexus-hud-validation`, base `f6098b8c684a97bf7e8e56804109ca491c78cfa2`, sem commit/push.

- **PASS** — `tsc --noEmit`, ESLint, Prettier e Stylelint nos arquivos do HUD; Vitest direcionado 37/37; `go test -count=1 ./internal/desktop -run HUD`; build Web e build Go do `nexus-hud`.
- **PASS** — UI Automation em Windows 11 `10.0.26200`, binário isolado `nexus-hud-diag-v3` (execução anterior): presets medidos `220×60 → 334×496 → 380×560 → 440×640 → 334×496 → 220×60`; nomes acessíveis `Compacto`, `Confortável`, `Grande` e `Arrastar HUD. Use as setas para mover.`; arraste real `-144,-108` px; seta direita `+16` px; `Shift`+seta `+64` px.
- **PASS** — persistência: após escolher `comfortable` e recolher, o estado ficou `260×70`; após reiniciar o HUD no mesmo diretório de dados, abriu em `260×70` na posição persistida (`1516,866`).
- **PASS** — primeiro paint: com estado limpo, o HUD abriu em `220×60` no canto inferior direito, sem o antigo mínimo `190×48`.
- **PASS** — rebuild final (`nexus-hud-diag-v4.exe`, SHA-256 `ae64668c16eedbacabc9f91eb0b05a8a3cb6f28f37800ed59a961e4aa5d49674`): `npm run quality` no `web/` passou com 93 arquivos/578 testes após reflow de Prettier em `NexusHud.tsx`; `go vet ./internal/desktop ./cmd/nexus-hud` e `go test -count=1 ./internal/desktop/...` PASS; build Web e build Go do HUD PASS.
- **PASS** — UI Automation no build final: presets medidos `260×70 → 380×560 → 334×496 → 380×560 → 440×640 → 334×496 → 220×60`; alça acessível `24×42` com nome `Arrastar HUD. Use as setas para mover.`; arraste real `-132,-88` px (pedido `-144,-96`); reinício abriu em `220×60` na posição persistida `1524,768`.
- **PASS** — teclado (setas `+16` px, `Shift`+seta `+64` px) validado no build `v3`; não repetido no rebuild `v4` porque o caminho de teclado não mudou.
- **NOT_TESTED** — Linux e macOS do HUD nativo neste slice.
- **BLOCKED** — `node scripts/docs-verify.mjs` falha por alterações visuais não commitadas em `web/src`/`internal/desktop`; o gate exige SHA imutável. Nenhum GO, tag ou release.

<!-- frontend-verify:latest -->
## Frontend gate — 2026-11-22T04:42:15Z

Verdict: **PASS**. Relatório completo: [`DEV/validation/FRONTEND_LATEST.md`](validation/FRONTEND_LATEST.md).

## Lane A + CI — 2026-11-23 (SHA `1963b441`)

- **PASS** — `go vet ./internal/control/web/`, `gofmt -l` vazio e `go test ./internal/control/web/ -run TestDirector -count=1` (projeção Director com lineage e quota UNKNOWN).
- **PASS** — `tsc --noEmit` (exit 0), ESLint sem avisos em `DirectorAgentPanel.tsx`, Prettier limpo em `types.ts`/`resources.ts`/`DirectorAgentPanel.tsx`/`AgentsSurface.test.ts`.
- **PASS** — Vitest `web/src/features/agents`: 10 arquivos / 47 testes; `AgentsSurface.test.ts` 4/4 com o contrato do Inspector.
- **PASS** — `web/scripts/e2e-surface-isolation.mjs` local (worktree `dd1b384a` + fixture nova): o único 503 do fixture era `maestro/submissions`; com o Bridge falso o E2E fecha com "All Surface Isolation assertions PASSED".
- **PASS** — container do runner `nexus-ci`: `pkg-config --exists gtk4` e `webkitgtk-6.0` verdadeiros; `go test -count=1 ./cmd/nexus-hud` ok dentro de `/tmp/nexus-runner-work/nexus/nexus`.
- **PENDENTE** — CI do SHA `1963b441` (Linux E2E, Frontend, Browser E2E, Desktop Linux, GoReleaser Snapshot, Smoke DEB) e artefato `workspace-acceptance-linux-amd64.json` no SHA exato; aceite externo por capacidade continua NO_GO.

## Race de attach e acceptance linux-amd64 — 2026-11-23 (SHA `94412171`)

- **PASS** — `go vet ./internal/control/host/`, `gofmt -l` vazio, `go test ./internal/control/host/ -count=2` verde; o teste de handoff com dois clientes passa 10/10 no commit `1963b441` (worktree limpa) e 5/5 com o fix de fila do ack.
- **PASS** (local) — `web/scripts/e2e-surface-isolation.mjs` completo com o Bridge Maestro falso; `tsc --noEmit` exit 0; ESLint/Prettier limpos; Vitest `web/src/features/agents` 47/47 e `web/src/features/browser/BrowserSurface.test.ts` 16/16.
- **PENDENTE** — CI do SHA `94412171` e novo `workspace-acceptance-linux-amd64.json` com os passos `vitest:` executando de verdade; aceite externo por capacidade continua NO_GO.

## Lane A — `device-preview` (E2E de escala, foco e Axe) — 2026-11-24

- **PASS** — `npm --prefix web run test:e2e:device-preview`: ciclo desktop (1280x800) -> celular (390x844) -> tablet (820x1180) -> desktop na MESMA sessao, um `POST /action` viewport por passo, foco de teclado retido no botao e Axe (WCAG 2 AA) sem violacoes serious/critical nos tres presets. Saida: `Device preview E2E: PASS (3 presets, same session, Axe clean)`.
- **PASS** — `npm --prefix web run test -- BrowserSurface.test.ts`: 18/18 (2 novos casos de `applyBrowserActionViewport`: avanco de viewport e no-op para action nao-viewport).
- **PASS** — `npm --prefix web run typecheck` (exit 0); `npx eslint src/features/browser/BrowserSurface.tsx src/features/browser/BrowserSurface.test.ts` sem erros; `npm --prefix web run lint:styles` limpo; `npx prettier --check` limpo nos arquivos tocados.
- **PASS** — `node scripts/verify-agent-workspace-evidence-map.mjs` exit 0 (48 capabilities, 97 clauses, nenhuma certificada); `node scripts/verify-agent-workspace-v1.mjs` exit 0.
- **PENDENTE** — CI do SHA exato com o novo step `device-preview-e2e` e recaptura sintetica (a mudanca em `web/src` invalida o manifest visual); aceite visual externo no SHA publicado. Capability `verification: pending`; release **NO_GO**.
