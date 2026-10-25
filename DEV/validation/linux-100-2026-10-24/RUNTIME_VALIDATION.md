# Validação de agentes e runtime

## Provedores encontrados

O diagnóstico do Nexus reconheceu seis CLIs instaladas: Agy, Claude, Codex, Cursor, Gemini e OpenCode. Evidência sanitizada: `logs/cli-doctor-host-sanitized.json`. O arquivo contém caminhos com `${HOME}` e não inclui tokens.

## Sessões autenticadas do host

`ImportHostSessions` foi executado contra os arquivos reais do host usando diretórios Nexus descartáveis em `/tmp`. Depois da correção Gemini, Codex, Gemini, AGY e OpenCode foram copiados para perfis temporários e confirmados pelo verificador local de autenticação. Claude tem um arquivo de credenciais, mas os campos OAuth `accessToken` e `refreshToken` estão vazios; o resultado `NO_SESSION` é correto para essa fonte. Cursor retornou `UNSUPPORTED`, pois o adapter não implementa importação de sessão do host. Nenhum arquivo de origem foi alterado; os perfis temporários foram removidos ao fim do teste.

Evidências: `logs/live-host-session-import.txt`, `logs/live-host-auth-validation.txt`, `logs/gemini-live-host-import-after-fix.log` e `logs/live-host-import-service-after-fix.log`. O smoke de Codex executou uma única solicitação genérica com a sessão importada em `CODEX_HOME` temporário, `--ephemeral`, sandbox `read-only` e sem ferramentas. O retorno foi exatamente `READY`; o arquivo original permaneceu intacto e a sessão temporária foi descartada. Evidência: `logs/live-codex-agent-smoke.txt`.

Essa execução valida a CLI Codex com a sessão copiada, mas não substitui um fluxo de agente iniciado pelo orquestrador Nexus. Não foram feitas chamadas reais a AGY ou OpenCode.

## Resultado por fluxo

| Fluxo | Resultado | Motivo e evidência |
| --- | --- | --- |
| Descoberta local de CLIs | PASS | `logs/cli-doctor-host-sanitized.json` |
| Importação real para perfil temporário — Codex, Gemini, AGY e OpenCode | PASS | `logs/live-host-import-service-after-fix.log`; `logs/gemini-live-host-import-after-fix.log`; fontes comparadas antes/depois |
| Claude com arquivo presente no host | PASS para `NO_SESSION` | O arquivo contém registro OAuth sem tokens; não existe uma sessão autenticada para importar. Evidência estrutural sanitizada: `logs/live-host-auth-validation.txt` |
| Importação de sessão Cursor | BLOCKED | Adapter sem suporte, status `UNSUPPORTED` no relatório de importação |
| CLI Codex com a sessão importada | PASS | `logs/live-codex-agent-smoke.txt`; retorno `READY`, sandbox somente leitura |
| Interface nativa Nexus usando importação real do host | NOT_RUN | O E2E da tela de primeiro uso valida feedback com respostas fixture; o serviço real passou separadamente |
| Início do Desktop e criação do workspace de smoke | PASS | `screenshots/native-single-mascot-main-xwayland.png` e `screenshots/native-functional-project-added-xwayland.png` |
| Comando no terminal, ANSI e Unicode | PASS | `screenshots/native-functional-terminal-command-xwayland.png`, `native-functional-terminal-ansi-xwayland.png` e `native-functional-terminal-unicode-xwayland.png` |
| Resize do PTY quando o painel muda de largura | PASS após correção | Teste de regressão e `screenshots/native-terminal-resize-after-fix-xwayland.png` |
| DIRECT, STICKY_LEAD e ORCHESTRATED com runtime Nexus | BLOCKED | Não foi executada uma sessão orquestrada real; o smoke direto da CLI não prova os caminhos do Nexus |
| KEEP, DELEGATE, MISSION, failover, retomada e cancelamento com provedores reais | BLOCKED | Requer uma sessão real no runtime e mais de um agente configurado |
| Skills invocadas por provedor durante uma sessão | BLOCKED | Nenhuma sessão de agente Nexus autenticada foi iniciada para o fluxo de skills |
| Estados de atenção do HUD produzidos por agente | BLOCKED | Fixtures cobrem modelo e render; não houve runtime Nexus real gerando os estados |
| Soak, recuperação de crash e processos órfãos | NOT_RUN | Não se executou soak de agentes Nexus |

Os testes automatizados Go e frontend passaram nos modelos, serviços e contratos cobertos por eles. O smoke real valida uma sessão Codex de baixo risco, não aprova continuidade, roteamento, delegação, failover ou execução de skills.

## Proteção de credenciais

Os 65 arquivos de texto em logs/ foram examinados por padrões comuns de API keys, JWT e Bearer longo; nenhum padrão foi encontrado. O registro sem valores está em logs/credential-pattern-scan.md. Isso não equivale a uma auditoria completa de todos os formatos ou dados do host.


## Casos CLI isolados adicionais

`nexus config validate` foi exercitado com arquivo ausente, JSON inválido e permissão de leitura negada em `HOME`/`XDG_CONFIG_HOME` descartáveis. Os defaults foram aceitos; JSON e permissão retornaram erros explícitos sem panic. A alteração de idioma `pt-BR` persistiu após novo processo, e o JSON manteve modo `0600`. Evidências: `logs/cli-config-negative-cases-final.log` e `logs/cli-config-persistence-final.log`. A colisão na mesma porta passou no binário CLI final (RES-001). Duas instâncias em portas distintas passaram no reteste exato (RES-002): cada Core respondeu HTTP 200 e gravou `listen-<PID>.json`; encerrar a segunda instância preservou a primeira, e o último encerramento limpou seu próprio registro. SIGTERM encerrou ambos com exit 0 (RES-003). Evidências: `logs/core-multi-instance-first-final-build.log`, `core-multi-instance-second-final-build.log`, `core-multi-instance-state-final-build.log` e `core-port-occupied-final-build.log`.


## Colisão de porta no Core

Um primeiro Core temporário foi iniciado em `127.0.0.1:13211` com HOME/XDG próprios e sem spawn do HUD; `/api/v1/health` respondeu HTTP 200. Um segundo processo tentou a mesma porta, saiu com código 1 e mensagem explícita `address already in use`. Após a tentativa, o primeiro Core continuou respondendo HTTP 200. Os processos e diretórios de teste foram isolados e encerrados; os serviços Nexus preexistentes do usuário não foram tocados. Evidências: `logs/core-port-first-instance-final.log` e `logs/core-port-occupied-final.log`. Uma tentativa preliminar em outro perfil retornou exit 0 sem saída observável; foi tratada como inconclusiva e não contada. O reteste pareado determinístico é o resultado PASS registrado em RES-001.

## Reteste de coexistência e shutdown — 2026-10-25

Dois Cores do binário `f7470fcee9a3f1a2f4eba2bf2dd8f9aac443c5cade2de30c7225561586496c86` foram iniciados com o mesmo perfil temporário em `127.0.0.1:13241` e `:13242`. Ambos responderam HTTP 200 e registraram estado separado por PID. Ao encerrar o processo da porta 13242, somente seu `listen-<PID>.json` foi removido; a porta 13241 manteve health 200 e seu registro. O primeiro processo encerrou depois com exit 0 e limpou o último registro. A segunda instância na mesma porta também falhou claramente com `address already in use`, sem afetar o primeiro.
