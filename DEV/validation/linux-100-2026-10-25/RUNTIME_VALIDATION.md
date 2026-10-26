# Validação de agentes e runtime

## Smoke real Codex no CLI instalado — 2026-10-26

- **PASS:** o CLI Nexus instalado encaminhou uma chamada direta para o Codex autenticado. Em `/tmp`, com `--ephemeral`, `--sandbox read-only` e `--skip-git-repo-check`, o provedor respondeu exatamente `NEXUS_RUNTIME_SMOKE_OK`; exit 0. CLI Codex `0.164.0`, modelo reportado `gpt-5.6-luna`, execução informou 7.968 tokens.
- O prompt proibiu uso de ferramentas. O sandbox não permitia escrita e a execução não usou o checkout. Depois do comando, `nexus ps --json` continuou sem runtimes ativos. Perfil/email não foram registrados.
- Duas tentativas anteriores falharam antes de chegar ao modelo: opções específicas de `exec` antes do alias `--print`, e falta de `--skip-git-repo-check` ao executar em `/tmp`. A forma correta abaixo passou. Isso não valida delegação, retomada, failover, Skills ou continuidade.
- O resultado foi transcrito de forma sanitizada após a execução real; o stdout original não foi salvo como arquivo de log. A linha `AUTH-LIVE-002` aponta para este registro e o procedimento reproduzível, sem inventar uma captura de log.
- Reprodução (substitua `<perfil-autenticado>` por um perfil Nexus Codex autenticado):

  ```bash
  cd /tmp
  nexus codex:<perfil-autenticado> --print --ephemeral --sandbox read-only --skip-git-repo-check \
    'Reply with exactly NEXUS_RUNTIME_SMOKE_OK and do not use tools.'
  ```

- Evidência: este registro operacional e o caso `AUTH-LIVE-002` em `TEST_MATRIX.md`; o stdout original não está preservado. O perfil não é citado para não expor identidade.

## Mission real via Nexus

Uma execução real do comando `nexus run` usou o provedor Codex em perfil e workspace temporários. Após atualizar a quota no perfil isolado, a solicitação `corrija este teste quebrado` terminou com:

- Estado final `COMPLETED_VERIFIED`;
- Intenção `DIRECT`, sem decomposição em Mission Team;
- Uma work package com papel `implementer`;
- Seis iterações e zero pedidos de coordenação;
- Verificação `go test ./...` concluída com exit 0;
- A única alteração ficou em `marker_test.go` do worktree temporário;
- O projeto canônico de teste e o repositório Nexus permaneceram sem alteração por essa execução.

O resumo sanitizado registra o SHA-256 do output bruto. O arquivo bruto ficou em `/tmp/nexus-linux-runtime-live-2026-10-25/private-direct-mission-output.log`; ele não foi copiado porque contém metadados privados do workspace. Nenhum segredo de autenticação foi registrado no dossiê.

## Tentativas anteriores e correções

- A primeira tentativa em perfil isolado foi bloqueada porque a quota ainda estava `UNKNOWN`. O refresh autenticado no cache oficial habilitou a repetição.
- Uma tentativa de Mission mais ampla foi barrada pelo guardrail quando o agente pediu coordenação depois de modificar o workspace. O bloqueio foi correto.
- Um prompt de coordenação copiável produziu placeholders repetidos. As instruções do prompt e o validador agora rejeitam esses placeholders.
- A repetição final para uma solicitação direta foi concluída e verificada. Evidência e testes: [resumo sanitizado](logs/runtime-direct-mission-summary.json), [full quality log](logs/quality-full.log).

## Limites atuais

Os PASS registrados cobrem um smoke direto de texto com Codex e uma execução anterior `DIRECT` com uma tarefa segura. Não homologam `STICKY_LEAD`, `ORCHESTRATED`, decisões KEEP/DELEGATE/MISSION, Skills em provedor real autenticado, failover, continuidade, retomada, cancelamento ou eventos do Attention Center no HUD.

A importação de sessões para Codex, Gemini, AGY e OpenCode foi validada em perfis temporários na execução anterior ([evidências](../linux-100-2026-10-24/logs/live-host-import-service-after-fix.log)). FIX-009 corrige o falso negativo de `cursor-agent status` (`Login successful`); Cursor permanece `UNSUPPORTED` para cópia/importação porque não há formato público documentado. Claude não tinha tokens locais e retornou `NO_SESSION`.

## Importação pela interface nativa

O botão **Importar sessões existentes** foi acionado no Desktop Wails real em XWayland, com `NEXUS_CONFIG_DIR`, `NEXUS_DATA_DIR` e diretórios XDG isolados em `/tmp`. Na validação anterior, a UI informou 4 importadas, 0 já autenticadas, 1 sem sessão importável, 1 sem suporte e 0 falhas; a correção do timeout de detecção fez a UI reconhecer quatro provedores com conta após reinício. Repeti o fluxo no overlay final `64c48060972e179d021f1406339ce3619389d0321f76ebcffad0414f46cbd904`: 4 importadas, 0 já autenticadas, 1 sem sessão, 1 sem suporte e 0 falhas; após reiniciar o mesmo Desktop e perfil isolado, a tela mostrou 4 provedores com conta. As sessões de origem não foram alteradas; o perfil temporário foi removido. Evidência e hashes das capturas: [smoke nativo do overlay final](logs/native-final-overlay-smoke.txt), [importação](screenshots/native-desktop-final-session-import-xwayland.png), [após reinício](screenshots/native-desktop-final-after-restart-xwayland.png).



## Chamadas reais de provedores via Nexus — overlay final

AGY 1.4.7 e OpenCode 2.1.3 concluíram uma solicitação segura através de `nexus run`, com sessões importadas em perfis descartáveis. Ambos retornaram `READY` e exit 0. O smoke usa a árvore de fontes identificada por `64c48060972e179d021f1406339ce3619389d0321f76ebcffad0414f46cbd904`; ver [resumo sanitizado](logs/live-provider-smokes-final.txt).

Gemini CLI 0.62.0 passou a aceitar o perfil OAuth preparado localmente após FIX-007, mas a chamada real foi recusada pelo serviço com `IneligibleTierError` para Gemini Code Assist individual. Esse provedor segue **BLOCKED** para esta conta; isso não é contabilizado como sucesso de modelo. Ver [correção e regressão](logs/gemini-auth-settings-regression.txt).

O smoke confirma invocação real por AGY e OpenCode, mas não homologa roteamento `STICKY_LEAD`/`ORCHESTRATED`, KEEP/DELEGATE/MISSION, failover, Skills ou continuidade de sessão.

## Skills em sessão Nexus — fixture isolada (SKL-FIXTURE-001)

**PASS técnico com fixture:** seleções explícita e automática de Skills foram compiladas e enviadas pelo caminho `Nexus.executeAgentPrompt` → launcher supervisionado → CLI Codex fixture. O processo recebeu o contrato de Skill completo como argumento, as flags de revisão `--sandbox read-only`/`--ask-for-approval never`, respondeu e encerrou. A repetição passou 5 vezes normalmente e 3 vezes com `-race`; `go test -count=1 ./internal/nexus` e `go vet ./internal/nexus` também passaram.

O teste usou `HOME`, configuração, dados, perfil e workspace temporários; comprovou que o `PATH` resolveu o CLI fixture e que não havia `auth.json` no perfil. Não usou tokens nem acessou serviços externos. Procedimento e limite: [SKILL_SESSION_VALIDATION.md](SKILL_SESSION_VALIDATION.md); código [skill_session_fixture_test.go](../../../internal/nexus/skill_session_fixture_test.go); logs em `logs/skill-session-fixture-repeat-2026-10-25.log`, `logs/skill-session-fixture-race-2026-10-25.log`, `logs/skill-session-internal-nexus-tests-2026-10-25.log` e `logs/skill-session-internal-nexus-vet-2026-10-25.log`.

**BLK-008 permanece BLOCKED:** falta uma sessão com provedor externo realmente autenticado. Fixture comprova o contrato de seleção/execução do Nexus, não a resposta de serviço externo.

## Anexação de runtime shell existente — FIX-011

O cartão “Anexar Terminal” foi corrigido para reabrir o runtime selecionado. A validação nativa no Desktop Wails atual criou o runtime `shell-06GQ5DD4PV4KZSM7H687M20YKC`, depois anexou essa sessão. O PID e o estado RUNNING persistiram; o contador ficou em 4 runtimes e a superfície de Terminais mostrou o prompt existente, sem iniciar outro shell. Evidências: [log](logs/runtime-attach-validation.txt), [terminal nativo](screenshots/terminal-runtime-attach-current-build-xwayland.png) e [contagem após anexar](screenshots/terminal-runtime-count-after-attach-xwayland.png).

Hosts iniciados por uma imagem anterior foram recusados após substituir o binário no mesmo caminho: a validação de identidade marcou os registros `STALE` e a conexão da interface mostrou `TERMINAL_TRANSPORT_ERROR`, mesmo com o socket do host respondendo ao ping direto. Este cenário após atualização permanece BLOCKED e precisa de validação de ciclo de vida em uma atualização oficial isolada; o gate de compatibilidade não foi alterado.

## Crash de processo durante dispatch ativo — NR-007-CRASH

**PASS com fixture determinística, no limite entre processos.** Um processo filho separado iniciou um Mission usando SQLite temporário, persistiu `DispatchIntent` e parou dentro do executor. O processo foi encerrado abruptamente. O processo Nexus de teste reabriu a mesma base, chamou `RecoverMissionRuns`, aguardou a lease original vencer e retomou a inspeção do run.

O recovery mudou o dispatch para `UNKNOWN_EXTERNAL_OUTCOME`, preservou seu ID, criou intervenção `DISPATCH_OUTCOME_UNKNOWN` e deixou o Mission em `BLOCKED_NEEDS_USER`. O executor substituto foi chamado zero vezes. A evidência não demonstra que um provedor externo realmente iniciou ou concluiu a solicitação; nesse estado ambíguo, não repetir a chamada é o comportamento seguro.

O primeiro teste vermelho reproduziu que o worker de startup terminava ao receber `ErrLeaseHeld`. A correção agora faz workers em background aguardarem a lease com backoff limitado e cancelável. A regressão passou normalmente (31,238 s), sob `-race` (33,409 s), na suíte `go test ./internal/nexus/... -count=1` (119,315 s) e em `go vet ./internal/nexus/...`. Log completo: [crash e recovery](logs/nr007-process-crash-recovery-current-2026-10-25.txt). Código: [teste de crash](../../../internal/nexus/mission_process_crash_test.go).

O gate composto **NR-007 permanece BLOCKED**. O harness controlado de oito horas passou em P9-003, mas não incluiu uma chamada de provedor externo, recuperação de PTY órfão sob carga prolongada nem inspeção persistente dos processos da aplicação após uso longo. O teste de crash durante dispatch foi realizado separadamente em fixture determinística; nenhum dos casos externos foi inferido a partir dela.

## Soak P9 de oito horas — concluído em 2026-10-26

- `make overnight-soak DURATION=8h` concluiu. O Go test registrou `ok github.com/IAPro-Community/nexus/internal/nexus/runner 28800.004s`; o processo de teste já não estava presente na observação terminal.
- A fixture é um harness determinístico do runner. Proveniência do código testado: HEAD/base `7a04f1b4cf5950b97b4ea1e156dda31ac513823b`, overlay `b874bec02f83066daed6f913b4d357fb411dfff9fe1ff97e8290a2f8b4f0ca8d`; SHA-256 do executável de teste preservado durante a execução: `ccfa49e1af5cf08fc5fae32339b788b7c8cb788b6914ff2a41abf0a63f096bbe`.
- P9-003 é **PASS** para o harness de soak. NR-007 continua **BLOCKED** para o gate composto: continuidade real de provider, recuperação de PTY órfão durante uso prolongado e inspeção de restart/processos no dia seguinte continuam sem validação.
- Evidências: `logs/p9-overnight-soak-8h-current-2026-10-25.log` e `logs/p9-overnight-soak-8h-final-observation-2026-10-26.txt`.

## Mission Codex real em workspace descartável — 2026-10-26

- O binário da CLI foi construído com `make build` no checkout `release/v1.0.0`, HEAD `7a04f1b4cf5950b97b4ea1e156dda31ac513823b`, fingerprint de fontes do produto `101e6e62b8d4b9903cf4d40309caeea54c809183e5dba017d9e6a8770302669c`; SHA-256 do executável `nexus`: `437ccf9a14b04a3bd463ce994cc36ba6fcf947f98d8e452ba990fcd4b5fad51c`.
- O harness `tools/runtime_isolated_smoke.py` importou a sessão Codex por endpoint suportado para perfil descartável, limpou variáveis de API key e criou um pequeno Git fixture dentro de `~/.cache`. A atualização de quota retornou `CACHED`, elegível e autenticado. O fixture exige uma alteração exata de `README.md` e `npm test`; nenhuma credencial ou identidade de conta foi registrada.
- **Tentativa A — resultado do produto concluído, evidência incompleta:** `nexus run` retornou exit 0; API reportou `COMPLETED_VERIFIED`, um package `VERIFIED`, dispatch `COMPLETED`, sem erro e sem runtime ao final. A primeira versão do harness verificou incorretamente o workspace canônico em vez do worktree de entrega e classificou BLOCKED; o processo então removeu a raiz temporária. Portanto, o conteúdo final do worktree e os comandos do recibo não foram preservados. Log: [primeira execução](logs/runtime-real-nexus-run-isolated-2026-10-26.log).
- **Tentativa B — BLOCKED:** após corrigir o harness para inspecionar worktree, recibo e isolamento do fixture canônico, o provedor não respondeu em 180 s. O grupo da CLI foi encerrado; o runtime ativo foi parado explicitamente; a inspeção final mostrou zero runtime. Core encerrou com exit 0 e os arquivos temporários foram removidos. Log: [repetição com inspeção do worktree](logs/runtime-real-nexus-run-isolated-worktree-2026-10-26.log).
- **Classificação RT-002: BLOCKED**, sem atribuir defeito ao agente nem declarar a tarefa homologada: houve uma conclusão real reportada pelo Nexus, porém o artefato correspondente não foi retido/inspecionado; a repetição excedeu o timeout. Uma aprovação futura exige registrar worktree, recibo e DoD global no mesmo ciclo.
- Reprodução: `make build`, depois `PATH="$PWD:$PATH" TMPDIR=/home/desenvolvedor/.cache/nexus-build-tmp python3 DEV/validation/linux-100-2026-10-25/tools/runtime_isolated_smoke.py`. O harness remove seu perfil/workspace descartável apenas após confirmar o Core parado e zero runtime; timeout ou evidência ausente termina como BLOCKED.
