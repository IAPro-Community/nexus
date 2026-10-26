# Terminais e agentes: YOLO e scrollback — 2026-10-26

## Escopo

Revisão solicitada para o modo YOLO do Codex, rolagem no OpenCode e no Oh My Pi, e funcionamento geral do fluxo de terminais/agentes. O usuário confirmou que “OMP” significa Oh My Pi (`omp` 18.11.1).

## Causas e correções

- **YOLO na criação do agente:** a interface gravava `--yolo` em `options.extra_args`, mas o launcher só normalizava aliases nos argumentos-base. O Codex recebia a forma canônica não suportada. Agora as opções persistidas passam pelo normalizador do provedor; Codex recebe `--dangerously-bypass-approvals-and-sandbox`. A normalização preserva argumentos nativos repetíveis, como `--add-dir`.
- **Modo no terminal:** `AgentTerminal` iniciava visualmente em Safe quando o host não fornecia `mode`, mesmo que a configuração persistida estivesse em YOLO. O terminal agora lê a configuração do agente e mantém a ação de mudança de modo desativada enquanto carrega. Criação e alternância Safe/YOLO compartilham a mesma rotina para manter o marcador e as flags alinhados.
- **Scrollback:** `TerminalPane` e `AgentTerminal` chamavam `scrollToBottom()` em todo evento `onData`. Relatórios de roda do mouse e teclas de paginação/Home/End podiam então reposicionar o buffer imediatamente. Os dois terminais agora preservam a posição nesses eventos, incluindo protocolos SGR, X10 e URXVT e sequências Home/End em modo de aplicação. O comportamento cobre os PTYs usados por OpenCode e Oh My Pi.
- **Anexação WebSocket:** a suíte de terminais encontrou que uma origem inválida podia abrir o pipe de controle antes de ser rejeitada pelo upgrade WebSocket. `HandleWebSocket` valida a origem antes de consultar/dialar o runtime, evitando esse efeito colateral e permitindo a anexação seguinte.

## Arquivos principais

- `web/src/nexus/agentExecutionMode.ts` e teste: leitura/persistência uniforme do modo.
- `web/src/nexus/AgentTerminal.tsx`, `web/src/app/WorkspaceSurfaceHost.tsx` e `web/src/features/agents/NewAgentModal.tsx`: modo persistido, criação e alternância.
- `web/src/nexus/terminalInput.ts` e teste, `web/src/components/TerminalPane.tsx` e `web/src/nexus/AgentTerminal.tsx`: preservação da rolagem manual nos dois tipos de terminal.
- `internal/core/provider/spec/flags.go`, `internal/control/flags/normalizer.go` e `internal/control/driver/launch_config.go`: normalização de aliases persistidos sem eliminar argumentos nativos repetidos.
- `internal/control/web/handler_terminal.go`: validação de origem antes do acesso ao pipe.

## Verificações

- PASS — Web: 113 suítes, 706 testes; `bun run typecheck`; `bun run lint:styles`; `bun run build`.
- PASS — Lint ESLint: zero erros; um aviso já existente em `web/src/features/work/FlowRunSurface.tsx:121` sobre dependência de `useCallback` (`plan.current_revision`).
- PASS — Prettier nos oito arquivos Web alterados.
- PASS — testes Go completos de `internal/control/flags`, `internal/core/provider/spec`, `internal/control/driver` e `internal/control/launcher`.
- PASS — `TestTerminalAttachBinaryMismatchReturnsActionableWebSocketError` e suíte completa `go test ./internal/control/web -count=1`.
- PASS — `go vet` nos pacotes Go relacionados e `go build -o %TEMP%\nexus-terminal-yolo-check-20261026-1733.exe ./cmd/nexus`.
- PASS — a flag nativa do Codex foi aceita pelo CLI instalado com `codex --dangerously-bypass-approvals-and-sandbox --help`; nenhuma chamada a modelo foi feita.
- **Gate geral pendente:** `bun run quality:full` parou no Prettier por 15 arquivos preexistentes fora deste escopo: `ComposerBriefOverview.tsx`, `ComposerDestinations.tsx`, `ComposerHandoffSummary.test.ts`, `ComposerHandoffSummary.tsx`, `ComposerReadOnlyHistory.tsx`, `ComposerStart.tsx`, `FlowDraftEditor.tsx`, `FlowGoalBar.tsx`, `flowRunHistoricalPlan.test.ts`, `flowRunHistoricalPlan.ts`, `FlowRunsHistorySurface.tsx`, `FlowRunSurface.tsx`, `useComposerSession.ts`, `useFlowDraftEditor.ts` e `api.test.ts`. Não foram reformatados.
- **Suíte Nexus ampla com falhas fora do diff:** a execução combinada passou em flags/provider/driver/launcher, mas `internal/nexus` reportou falhas de Composer sem IA configurada ou com timing, decomposição de Flow, admissão de Mission sem provedor elegível, resolução de skills e isolamento do fixture Codex (`codex.cmd` global). Não alterei esses caminhos nem estabeleci um baseline separado para eles. A suíte completa de `internal/control/web` foi repetida depois da correção de origem e passou.

## Limites e estado da instalação

- Não foi feita sessão interativa real de modelo com Codex, OpenCode ou Oh My Pi; não houve custo nem alteração em workspaces do usuário. As evidências cobrem configuração/flags, lógica de scrollback, testes de terminal e builds.
- O binário beta.11 instalado continua aberto em `127.0.0.1:13000` e há um runtime Codex ativo. A instalação **não** contém estas correções e não foi reiniciada para evitar interromper a sessão. A build Go foi produzida em `%TEMP%`, não instalada.
- A branch `fix/windows-agent-terminal-reliability` reúne estas alterações; ainda não há PR. Para confirmar a experiência visual em OpenCode/Oh My Pi e o botão YOLO no app, é necessário aplicar a build atual após encerrar ou salvar o runtime ativo.
