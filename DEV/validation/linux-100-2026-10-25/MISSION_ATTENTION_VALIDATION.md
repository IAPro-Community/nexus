# Validação do Attention Center e abertura de missões

## P2_NATIVE_LIVE_ATTENTION_VALIDATION — transições de estado e restart — 2026-10-25

- **PASS no escopo local single-user:** a mesma janela Wails real em XWayland (PID `2274400`, XID `0x1200004`) exibiu A/D/E; manteve A/D/E após polling enquanto B estava `EXECUTING` e C `REMEDIATING`; e mostrou A/B/C/D/E (1 precisa de você, 3 concluídas, 1 falha) após a fixture SQLite persistir B/C como concluídas.
- **PASS após restart:** o Core isolado encerrou (PID `2271425`) e voltou na mesma porta `43162` (PID `2307030`); health respondeu HTTP 200. Sem F5 e com o mesmo Desktop PID/XID, o drawer recuperou a projeção 1/3/1/5.
- O teste atualiza estados canônicos no banco fixture isolado e observa o polling real da UI. Não chama runner nem provedor; recuperação/execução determinística do runner está coberta em `P2-MISSION-001`. Não houve alteração de dados ou sessão do usuário. ACL multiusuário não é suportada nem homologada pelo workspace local single-user.
- Evidência de sessão e fronteiras: [log](logs/p2-native-live-attention-current-overlay-2026-10-25.txt). Capturas Wails reais com checksums: [índice SHA-256](logs/p2-native-live-attention-screenshots-2026-10-25.sha256), [estado inicial](screenshots/p2-native-live-attention-2026-10-25/attention-drawer-initial.png), [B/C ativos](screenshots/p2-native-live-attention-2026-10-25/attention-drawer-active.png), [B/C concluídos](screenshots/p2-native-live-attention-2026-10-25/attention-drawer-completed.png), [após restart](screenshots/p2-native-live-attention-2026-10-25/attention-drawer-after-core-restart.png).
- O build de teste usou apenas IDs GTK/D-Bus exclusivos e tem SHA-256 `46dc4cb910de307ce2c136175da351761ffa8b97f111da68ba14a01884cbaa31`. Fonte efetiva do produto: overlay `2c23ebb0c7726a200acbde9326c23ad6023d8fad0238c2adfef993677203f2fa` sobre HEAD `7a04f1b4cf5950b97b4ea1e156dda31ac513823b`.

## Follow-up P2: recuperação multi-missão A/B/C/D/E — 2026-10-25

- Branch `release/v1.0.0`; HEAD local/remoto `7a04f1b4cf5950b97b4ea1e156dda31ac513823b`; overlay de fontes `30223d700c7d7395705571ad72d2ea30055d191e181b03cb114a922395aef18b` (1.422 fontes; `logs/source-provenance-p2-multimission-2026-10-25.txt`). Worktree modificada e sem commit imutável.
- O teste `TestAttentionHTTPMultiMissionQuotaRecoverySurvivesRestart` usa executor determinístico, cinco projetos, endpoint HTTP autenticado e banco SQLite real. A recebe resultado de dispatch desconhecido e fica `BLOCKED_NEEDS_USER`; B fica em checkpoint `EXECUTING`; C recebe falha tipada de rate limit, persiste `REMEDIATING`/`RetryFrom=ALLOCATING`; D termina `COMPLETED_VERIFIED`; E termina `FAILED_NO_PROGRESS`.
- Antes do restart, a rota `/api/v1/attention` retorna A/D/E (needs-user 1, completed 1, failed 1). B, ainda executando, e C, durante a recuperação de quota, não aparecem como itens de atenção. Depois de fechar e reabrir SQLite, serviço e servidor HTTP de teste, `RecoverMissionRuns` retoma B/C: ambos concluem `COMPLETED_VERIFIED` e a projeção passa a conter A/B/C/D/E (needs-user 1, completed 3, failed 1). C aparece como concluída após sucesso; a ausência se aplica somente enquanto sua remediação está ativa.
- Os contadores provam ausência de repetição indevida: B faz 1 alocação/execução; C faz 2 alocações/execuções (tentativa limitada por quota e retomada bem-sucedida). O teste de projeção cross-project existente também valida 401 sem sessão e persistência de categorias/filtros após reconstrução.
- Validação real: teste direcionado repetido 10 vezes e teste de projeção repetido junto; race do cenário A/B/C/D/E 5 vezes; pacote `internal/control/web` completo; `go vet ./internal/control/web`. Todos passaram. Logs: `logs/p2-multi-mission-http-restart-e2e-2026-10-25.txt`, `logs/p2-multi-mission-http-restart-race-2026-10-25.txt`, `logs/go-test-web-p2-multi-mission-2026-10-25.txt`, `logs/go-vet-web-p2-multi-mission-2026-10-25.txt`.
- Limite dos testes backend e nativo: o executor é determinístico e não chama provedor externo. O teste nativo cobre as transições da projeção e restart do Core na mesma sessão, mas os estados são escritos por fixture SQLite e não pelo runner. ACL por usuário/projeto não existe no workspace local single-user e não foi homologada. A evidência funcional P2 no escopo local exercitado está concluída; continuidade com provedor real permanece fora desta validação, e o resultado global segue `NO_GO_LINUX`.

## Follow-up nativo P2-UI-MISSION-001 — snapshot anterior antes/depois do restart —

- PASS: janela Wails real em XWayland exibiu cinco missões fixture (1 needs-user, 3 completed, 1 failed) antes e depois do restart do Web Core isolado. O Core mudou do PID 2677424 para 2742197, health e endpoint autenticado voltaram HTTP 200, e a janela `0x140000e` permaneceu no Desktop PID 2681463.
- Capturas reais de 1280×800 e checksums: `screenshots/p2-native-multi-mission-2026-10-25/attention-native-five-missions-before-restart.png`, `.../attention-native-five-missions-after-restart.png` e `logs/p2-native-multi-mission-screenshots-2026-10-25.sha256`.
- Evidência de processo/projeção: `logs/p2-native-five-mission-ui-restart-2026-10-25.txt`. Fixture é um snapshot pós-recuperação; a progressão live de missões na UI e ACL multiusuário seguem pendentes.

## Follow-up E2E de estados e layout — overlay `ee8a19c0` — 2026-10-25

- HEAD de base/remoto: `7a04f1b4cf5950b97b4ea1e156dda31ac513823b`; fingerprint atual: `ee8a19c0cf340086f8f1cfc258d50a98ec29719952b978401be55c977a693b6` (1.415 fontes, manifesto `logs/source-provenance-p2-ui-e2e-2026-10-25.txt`).
- `bun run test:e2e:mission-attention`: PASS no Chromium com servidor fixture, 12 requests. Cobriu loading, needs-user/completed/failed, filtros de projeto+mission, teclado, estado vazio, erro HTTP 503, limpeza de cards obsoletos e abertura do detalhe da missão.
- `ATTN-WEB-STATE-01`: PASS; o detalhe carregou a pergunta e as ações fixture. `ATTN-WEB-MOBILE-01`: PASS em 390×844; drawer e filtros cabem sem scroll horizontal após a animação.
- Bugs corrigidos com regressão no E2E: erro de consulta deixava cards antigos visíveis; o drawer de 440 px transbordava a viewport de 390 px. A correção limpa a projeção no erro e aplica `max-width: 100vw` via `ContextDrawer.module.scss`.
- Capturas: [lista desktop](screenshots/p2-mission-attention-2026-10-25/mission-attention-desktop.png), [missão aberta](screenshots/p2-mission-attention-2026-10-25/mission-attention-opened-mission.png), [drawer mobile](screenshots/p2-mission-attention-2026-10-25/mission-attention-mobile.png). Checksums: `logs/mission-attention-screenshots.sha256`.
- O E2E é Chromium com fixtures e não testa a janela Wails nem o processo real `nexus web` após reiniciar; esses gates continuam abertos.

## Follow-up de backend e UI — HEAD 7a04f1b4

- Branch/HEAD remoto: `release/v1.0.0` / `7a04f1b4cf5950b97b4ea1e156dda31ac513823b`; fingerprint do overlay `340800c27162b0d340ac10a4d65e32af2efb09f470b1c17a12e9ba594c0ff98f` (1.413 fontes, incluindo o novo teste Go).
- `TestAttentionHTTPProjectionSurvivesRestartWithProjectIsolation`: PASS em 10 execuções e 5 com `-race`. A rota com middleware real retorna 401 sem sessão; cliente autenticado lê 3 categorias em dois projetos; filtros por projeto/mission não atravessam escopos; a projeção permanece após encerrar/reabrir store/serviço/servidor HTTP de teste. Evidências: `logs/attention-http-restart-p2-2026-10-25.log`, `logs/attention-http-restart-p2-race-2026-10-25.log` e `logs/source-provenance-p2-http-restart-2026-10-25.txt`.
- `bun run test:e2e:hud`: PASS no Chromium com fixture; atenção exibida, “Adiar” persistido sem remover a missão do Attention Center, movimento reduzido respeitado e Axe sem violações sérias/críticas em desktop/mobile. Log `logs/attention-hud-web-e2e-p2-2026-10-25.log`; capturas em `screenshots/p2-hud-e2e-2026-10-25/`.
- O teste HTTP não exercita o processo real `nexus web` reiniciado enquanto a UI permanece aberta nem ACL por usuário/projeto. Os estados vazios/loading/erro e responsividade agora têm cobertura Chromium em fixtures, conforme seção acima; o workspace local é single-user e P2 continua IN_PROGRESS.
- Mascote retestado no mesmo HEAD: 27/27 testes, exportação SVG coerente, e HUD Wails mostra uma arte NEX em Xvfb; ver `MASCOT_VALIDATION.md`. Não há opção 2D ou segundo renderer integrado.

- Data: 2026-10-25
- Branch: `release/v1.0.0`
- Base Git local/remota: `bf6adf0c80ac3531d7291844512a14a017ab0002`
- Fingerprint do overlay compilado: `7368531977b7422eccdf00e7c676f57d440e8d736d9e114bcdc071b9d5885fe6` (1.410 arquivos do escopo descrito no manifesto)
- Binário Desktop Wails SHA-256: `f6e1afcf2e7b0e10ed5d75a5d716b22dfe6f6ca894e056c64938599172ee7f15`

Esta é uma validação complementar ao relatório de homologação. O worktree continua sujo e sem commit imutável; estes resultados não mudam o veredito geral **NO_GO_LINUX**.

## Matriz desta rodada

| ID | Área | Procedimento | Resultado observado | Status | Severidade | Evidência |
| --- | --- | --- | --- | --- | --- | --- |
| ATTN-API-01 | Backend | `go test ./internal/nexus/runner ./internal/nexus ./internal/control/web -count=1` | Filtros por projeto e missão passaram nos testes do Attention Center e do handler. | PASS | — | `logs/go-mission-attention-regression.log` |
| ATTN-API-02 | Backend | `go vet ./internal/nexus/runner ./internal/nexus ./internal/control/web` | Sem diagnósticos; exit code 0. | PASS | — | `logs/go-mission-attention-vet.log` |
| ATTN-API-03 | Persistência | `go test ./internal/nexus -run '^TestGlobalAttentionProjectionSurvivesStoreReopen$' -count=1 -v` | Cinco missões determinísticas sobreviveram ao close/reopen do SQLite; A/D/E continuaram em Needs You/Completed/Failed, B/C em execução permaneceram fora e os filtros não cruzaram projetos. | PASS | — | `logs/go-mission-attention-persistence-reopen.log`; `logs/go-mission-attention-persistence-vet.log` |
| ATTN-API-04 | Autenticação HTTP | Teste do endpoint `/api/v1/attention` sem sessão e com cliente autenticado usando filtros sem correspondência | Sem autenticação, a API respondeu 401; com cliente autenticado, respondeu 200 e a projeção filtrada veio vazia. | PASS | — | `logs/go-mission-attention-auth-endpoint.txt`; `logs/go-mission-attention-auth-endpoint-vet.txt`; `logs/mission-attention-auth-test-provenance.txt` |
| ATTN-WEB-01 | Frontend | `cd web && bun run quality:full` | 108 arquivos/672 testes passaram; arquitetura de estilos, formatação, ESLint, Stylelint, TypeScript e build passaram. | PASS | — | `logs/web-quality-full-mission-attention-route-layout.log` |
| ATTN-NATIVE-01 | Desktop Linux | Iniciar o Wails em perfil isolado no GNOME Wayland/XWayland e abrir a Central de Atenção | A aba Missões mostrou a fixture controlada, contagem 1, filtros e estado “Precisa de você”. | PASS | — | `screenshots/mission-attention-drawer-native-xwayland.png` |
| ATTN-NATIVE-02 | Navegação | Clicar no cartão da fixture na janela Wails real | A aba “Flow Run” tornou-se ativa e mostrou a missão `BLOCKED_NEEDS_USER`, pergunta e opção disponível. Nenhuma chamada a provedor foi feita. | PASS | — | `screenshots/mission-attention-open-detail-native-xwayland.png`; `logs/native-mission-attention-route-open-retry-xwayland.log` |
| ATTN-NATIVE-03 | Layout | Inspecionar valores e rótulos do detalhe após aplicar SCSS Module | Resumo de estado e evidência agora usa colunas e espaçamento; pergunta e ações permanecem legíveis em 1280×800. | PASS | — | Screenshot acima; `web/src/features/work/FlowRunSurface.style.test.ts` |
| ATTN-NATIVE-04 | Atualização live e restart | Conectar Desktop Wails a Web Core isolado; inserir B/C ativos, concluir ambos, reiniciar Core na mesma porta e aguardar o refresh automático da janela | B/C ativos ficam fora da lista; ao concluir, a projeção A/B/C/D/E aparece sem reiniciar o Desktop; a mesma janela mantém a projeção após restart do Core. | PASS: a mesma janela/PID/XID mostrou A/D/E, omitiu B/C em EXECUTING/REMEDIATING, depois mostrou 1/3/1/5 após concluir B/C e manteve essa projeção após restart do Core. Atualizações vieram da fixture SQLite, sem agente/provedor. | PASS | `logs/p2-native-live-attention-current-overlay-2026-10-25.txt`; `logs/p2-native-live-attention-screenshots-2026-10-25.sha256`; `screenshots/p2-native-live-attention-2026-10-25/` |
| MASCOT-SINGLE-01 | Mascote | Inspecionar o HUD nativo e executar regressões focadas existentes | Há uma única arte NEX. Barra, Mascote e Bandeja são apresentações do mesmo HUD; não existe modo ou renderer Companion 2D separado. | PASS | — | `screenshots/native-nexus-hud-single-mascot-xwayland.png`; `MASCOT_VALIDATION.md` |

## Correções reproduzidas

- O clique no cartão de atenção fechava o drawer, mas não deixava uma aba de decisão ativa. O callback agora abre a superfície `flow-run` identificada no workspace e sincroniza a rota. A rota `missions/:id` é permitida, enquanto Composer, criação e histórico continuam sob o gate experimental.
- Os cartões `.nx-flow-run-status` não tinham estilo registrado; `strong` e `span` apareciam colados. O resumo usa `FlowRunSurface.module.scss` com grid, gap e quebra segura de texto.
- Foi preservada a decisão de produto: um mascote NEX único. Nenhum renderer ou modo Companion 2D adicional foi integrado, e a apresentação Mascote existente não foi removida.

## Artefato e ambiente

- Build: `make build-desktop-wails`; ELF Linux AMD64. `ldd` não reportou bibliotecas ausentes.
- Janela Wails real: 1280×800; GNOME em sessão Wayland, aplicação em XWayland. O processo foi iniciado com `G_DEBUG=fatal-criticals`, perfil de dados isolado e HUD spawn desativado.
- O log registra avisos de aceleração DRI/VMware do host, sem `GDK-CRITICAL` durante esta validação.
- Checksums do binário e capturas: `logs/mission-attention-and-single-mascot-sha256.txt`.
- Lista e fingerprint das fontes usadas: `logs/source-provenance-p2-mission-attention.txt`.

## Reprodução

```bash
cd /projetos/nexus
make build-desktop-wails
GDK_BACKEND=x11 G_DEBUG=fatal-criticals \
  NEXUS_DATA_DIR=/tmp/nexus-mission-attention-p2.zDpzg5/data \
  NEXUS_CONFIG_DIR=/tmp/nexus-mission-attention-p2.zDpzg5/config \
  NEXUS_STATE_DIR=/tmp/nexus-mission-attention-p2.zDpzg5/state \
  XDG_CACHE_HOME=/tmp/nexus-mission-attention-p2.zDpzg5/cache \
  NEXUS_HUD_SPAWN=0 ./cmd/nexus-desktop/build/bin/nexus-desktop
```

O perfil nativo contém uma fixture local controlada. Ela valida lista, filtros, navegação, renderização, refresh por estados persistidos e recuperação do drawer após restart real do Core; não valida eventos reais de agentes, integração externa ou decisões de provedor. O close/reopen de SQLite, a rejeição HTTP sem autenticação e os estados vazio/loading/erro têm cobertura automatizada. O produto local testado é single-user; isolamento entre usuários não foi implementado nem homologado.

## Follow-up nativo — transições live e restart do Core — 2026-10-25

- **PASS** — O Desktop Wails real em XWayland (PID `2274400`, XID `0x1200004`) foi conectado ao `nexus web` isolado em `127.0.0.1:43162`. O primeiro Core (PID `2271425`) e o Core reiniciado (PID `2307030`) responderam HTTP 200 em `/api/v1/health`; a mesma janela e o drawer permaneceram abertos após o restart. Sem F5, a consulta periódica da Central recuperou a projeção final em até 17 segundos.
- **PASS** — Em cinco projetos fixture pré-carregados, a captura inicial mostrou A/D/E (1 precisa de você, 1 concluída, 1 falha). B em `EXECUTING` e C em `REMEDIATING` permaneceram fora da Central após o ciclo de 15 segundos. Depois de persistir B/C como `COMPLETED_VERIFIED`, a mesma UI mostrou A/B/C/D/E (1 precisa de você, 3 concluídas, 1 falha). A projeção 1/3/1/5 permaneceu após reiniciar o Core.
- A sequência UI atualiza diretamente o SQLite isolado via `store.UpsertMissionRun`; ela comprova o polling/projeção da interface e sobrevivência ao restart, não executa um agente ou uma transição do runner. A semântica de execução e recuperação realista permanece coberta separadamente por `P2-MISSION-001`, que usa executor determinístico, HTTP autenticado, SQLite/recovery e contadores de execução. Nenhum provedor externo foi chamado.
- Ambiente: Ubuntu 25.10, GNOME 49 em Wayland; Desktop executado em XWayland, 1280×800. CLI e Desktop foram compilados do overlay efetivo `2c23ebb0c7726a200acbde9326c23ad6023d8fad0238c2adfef993677203f2fa` (1.425 fontes de produto). O Desktop usa somente IDs GTK/ApplicationID exclusivos em overlay temporário de build; binário SHA-256 `46dc4cb910de307ce2c136175da351761ffa8b97f111da68ba14a01884cbaa31`; CLI SHA-256 `b149f5145e9ef3337102573d281b71299e8450a450f3d21e8ef405ae8c8f0b6c`.
- Capturas reais de 1280×800 e hashes em `screenshots/p2-native-live-attention-2026-10-25/` e `logs/p2-native-live-attention-screenshots-2026-10-25.sha256`. Log de processo, estados observados e reprodução: `logs/p2-native-live-attention-current-overlay-2026-10-25.txt`.
- O teste multi-missão nativo e restart da UI fica **PASS** para o contrato local single-user suportado. Isso não certifica isolamento entre usuários, provedores reais, alpha do HUD ou os demais gates da release. P2 está concluído no escopo funcional registrado; o veredito geral permanece `NO_GO_LINUX` por gates independentes.
