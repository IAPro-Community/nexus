# Verification: Nexus V1 (post-pending-issues)

## 2026-09-05 — Autopilot Platform Stabilization T0–T4

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
- `make build` — PASS (binário `nexus v0.5.0-beta.23` gerado e instalado com bundle web embutido).
- `npm --prefix web run test:e2e-hardening` — PASS (100% de sucesso nos 6 viewports):
  - `320x568` (mobile-se): botão Terminal desobstruído (`elementFromPoint` verificado sem overlap).
  - `390x844` (mobile-iphone): botão Terminal desobstruído.
  - `768x1024` (tablet): botão Terminal desobstruído.
  - `1024x768` (laptop-compact): layout estável, ações secundárias colapsadas.
  - `1280x800` (laptop-critical): terminal acessível em 1º lugar, sem clipping.
  - `1440x900` (desktop): renderização completa sem colisão.
  - Validação WAI-ARIA do Accordion de Temas com paletas visuais e medição real do delta de densidade (Comfortable 497px vs Compact 481px).


- `make web-verify` — PASS (typecheck, lint, null-arrays, vitest (244 tests), i18n, build, embed-sync e ui-markers).
- `make build` — PASS (binário `nexus v0.5.0-beta.23` compilado com assets embutidos atualizados).
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
- `make build` — PASS; binário `nexus v0.5.0-beta.23` gerado e instalado com novos assets embutidos em `internal/control/web/dist`.
- Validação visual: extração do master em alta resolução com desmatting transparente, verificação de contraste em fundo escuro (`#080a0f`) e fundo claro, geração do pacote favicon e renderização no rail e hero.

## Controles de terminais e resumo de Agentes — 2026-09-03

- `make web-verify` — PASS (typecheck, lint, null-arrays, testes, i18n, build,
  embed-sync e ui-markers).
- `make build` — PASS; instalou `nexus v0.5.0-beta.23`.
- Cobertura funcional implementada: troca de modo com erro visível e rebinding,
  feedback de lease, fechamento com escolha aba/runtime, confirmação de Project
  Shell e badge operacional clicável com foco/recuperação.

## Bootstrap de contexto durável — 2026-09-03

- `go test ./internal/nexus ./internal/control/web` — PASS.
- `npm --prefix web run typecheck` — PASS.
- `npm --prefix web test -- --run` — PASS (42 arquivos, 180 testes).
- `make build` — PASS; instalou `nexus v0.5.0-beta.23`.
- A criação é explícita, gera `AGENTS.md` base e nunca sobrescreve contexto
  existente.

## Recuperação de terminal desconectado — 2026-09-03

- Web: `npm --prefix web run typecheck` — PASS.
- Web: `npm --prefix web test -- --run` — PASS (42 arquivos, 175 testes).
- Build: `make build` — PASS; instalou `nexus v0.5.0-beta.23`.
- Recover/Start agora vincula diretamente a geração retornada; o terminal também
  oferece fechamento explícito da aba.

## Nova sessão de IA — 2026-09-03

- Web: `npm --prefix web test -- --run` — PASS (40 arquivos, 157 testes).
- Web: `npm --prefix web run typecheck` — PASS.
- Build: `make build` — PASS; instalou `nexus v0.5.0-beta.23`.
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

<!-- frontend-verify:latest -->
## Frontend gate — 2026-09-06T00:10:11Z

Verdict: **PASS**. Relatório completo: [`DEV/validation/FRONTEND_LATEST.md`](validation/FRONTEND_LATEST.md).

