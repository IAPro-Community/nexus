# Validação de agentes e runtime

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

Este PASS cobre um caminho real `DIRECT` com Codex e uma única tarefa segura. Não homologa `STICKY_LEAD`, `ORCHESTRATED`, decisões KEEP/DELEGATE/MISSION, uso real de Skills, failover, continuidade, retomada, cancelamento ou eventos do Attention Center no HUD.

A importação de sessões para Codex, Gemini, AGY e OpenCode foi validada em perfis temporários na execução anterior ([evidências](../linux-100-2026-10-24/logs/live-host-import-service-after-fix.log)). FIX-009 corrige o falso negativo de `cursor-agent status` (`Login successful`); Cursor permanece `UNSUPPORTED` para cópia/importação porque não há formato público documentado. Claude não tinha tokens locais e retornou `NO_SESSION`.

## Importação pela interface nativa

O botão **Importar sessões existentes** foi acionado no Desktop Wails real em XWayland, com `NEXUS_CONFIG_DIR`, `NEXUS_DATA_DIR` e diretórios XDG isolados em `/tmp`. Na validação anterior, a UI informou 4 importadas, 0 já autenticadas, 1 sem sessão importável, 1 sem suporte e 0 falhas; a correção do timeout de detecção fez a UI reconhecer quatro provedores com conta após reinício. Repeti o fluxo no overlay final `64c48060972e179d021f1406339ce3619389d0321f76ebcffad0414f46cbd904`: 4 importadas, 0 já autenticadas, 1 sem sessão, 1 sem suporte e 0 falhas; após reiniciar o mesmo Desktop e perfil isolado, a tela mostrou 4 provedores com conta. As sessões de origem não foram alteradas; o perfil temporário foi removido. Evidência e hashes das capturas: [smoke nativo do overlay final](logs/native-final-overlay-smoke.txt), [importação](screenshots/native-desktop-final-session-import-xwayland.png), [após reinício](screenshots/native-desktop-final-after-restart-xwayland.png).



## Chamadas reais de provedores via Nexus — overlay final

AGY 1.4.7 e OpenCode 2.1.3 concluíram uma solicitação segura através de `nexus run`, com sessões importadas em perfis descartáveis. Ambos retornaram `READY` e exit 0. O smoke usa a árvore de fontes identificada por `64c48060972e179d021f1406339ce3619389d0321f76ebcffad0414f46cbd904`; ver [resumo sanitizado](logs/live-provider-smokes-final.txt).

Gemini CLI 0.62.0 passou a aceitar o perfil OAuth preparado localmente após FIX-007, mas a chamada real foi recusada pelo serviço com `IneligibleTierError` para Gemini Code Assist individual. Esse provedor segue **BLOCKED** para esta conta; isso não é contabilizado como sucesso de modelo. Ver [correção e regressão](logs/gemini-auth-settings-regression.txt).

O smoke confirma invocação real por AGY e OpenCode, mas não homologa roteamento `STICKY_LEAD`/`ORCHESTRATED`, KEEP/DELEGATE/MISSION, failover, Skills ou continuidade de sessão.

## Anexação de runtime shell existente — FIX-011

O cartão “Anexar Terminal” foi corrigido para reabrir o runtime selecionado. A validação nativa no Desktop Wails atual criou o runtime `shell-06GQ5DD4PV4KZSM7H687M20YKC`, depois anexou essa sessão. O PID e o estado RUNNING persistiram; o contador ficou em 4 runtimes e a superfície de Terminais mostrou o prompt existente, sem iniciar outro shell. Evidências: [log](logs/runtime-attach-validation.txt), [terminal nativo](screenshots/terminal-runtime-attach-current-build-xwayland.png) e [contagem após anexar](screenshots/terminal-runtime-count-after-attach-xwayland.png).

Hosts iniciados por uma imagem anterior foram recusados após substituir o binário no mesmo caminho: a validação de identidade marcou os registros `STALE` e a conexão da interface mostrou `TERMINAL_TRANSPORT_ERROR`, mesmo com o socket do host respondendo ao ping direto. Este cenário após atualização permanece BLOCKED e precisa de validação de ciclo de vida em uma atualização oficial isolada; o gate de compatibilidade não foi alterado.
