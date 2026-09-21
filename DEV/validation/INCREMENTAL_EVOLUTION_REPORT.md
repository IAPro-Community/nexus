# Relatório — evolução incremental (autopilot 2026-09-19)

Branch: `feat/capacity-monitor-revival` @ base `483f12e` (+ working tree desta rodada).
Base de merge: `origin/main` (`b8d9962`). PR #3.

## Baseline → final

- Baseline: T0–T3 prontos (probe contract, SQLite, loop com histerese), T4–T9
  abertos, monitor novo sem callers de produção, 0021 duplicada, dual-publish
  sem coordenação, sem API/UI de capacity.
- Final: T4–T9 implementados e testados; dispatcher único; API+UI; autostart
  consent-gated; routing/fallback/affinity/explain; memory E2E; Composer;
  Missions projection. Nenhum subsistema novo.

## Problemas encontrados → corrigidos

| Classe | Achado | Ação |
|---|---|---|
| `FIX_EXISTING` | 0021 duplicada (segundo arquivo nunca aplicado) | Fundida em 0021 única + repair (decisão: fundir) |
| `EXTEND_EXISTING` | Sem bridge de recomendação no monitor novo | `CapacityRecommender` sobre `RecommendResources` |
| `EXTEND_EXISTING` | Sem dedup/receipts/ack/snooze | T5 completo + enforcement |
| `EXTEND_EXISTING` | Dual-publish sem coordenação | `Coordinator` owner-tagged (decisão: single dispatcher) |
| `REGRESSION_THIS_BRANCH` | `DeriveRoutingSwitchReason` com substring fora do classificador (guardrails falharam) | Reescrito sobre `classifier.Classify` |
| `REGRESSION_THIS_BRANCH` | Dispatcher v1 auto-suprimia milestones do legado | Owner tags + `Forget` (review próprio) |
| `PREEXISTING` | `make docs-verify` bloqueado (manifesto visual stale) | Não mascarado, registrado |
| `ENVIRONMENTAL` | `go build ./...` puro: VCS stamping no worktree | `make build` canônico PASS |
| `NOT_VERIFIED` | Providers reais, Windows/macOS, overnight | Registrado, fora do ambiente |

## Módulos

- **Capacity Monitor**: T4–T8 + dispatcher + T6 API + settings/alertas/UI. Monitor
  passa a ser utilizável de verdade (status/history/prefs/ack/snooze).
- **Terminal hardening**: preservado, intocado por esta rodada.
- **Routing/fallback**: Retry-After > heurística; fallback limitado por elegíveis;
  sticky affinity; motivos tipados via classificador.
- **Memory**: E2E produtor→persistência→consumidor; PROJECT > INSTALL; secrets
  rejeitados. Sem segundo memory engine.
- **Composer**: reusa fatos duráveis; pedido atual vence.
- **Web/UX**: settings, alert cards com dedup por fingerprint, faixa de monitor
  no usage, execution summary no flow. Sem tray.

## Inspirações externas

| Ideia | Fonte | Veredito |
|---|---|---|
| Sticky affinity (sessão saudável não rebalanceia) | OpenCodex | ADAPTED (`ShouldRetainCurrentResource`) |
| Retry-After > heurística | OpenCodex | ADAPTED (`EffectiveCooldown`) |
| Tentativas limitadas pelos elegíveis | OpenCodex | ADAPTED (fallback bound, anti ping-pong) |
| Explicação de routing | OpenCodex | ADAPTED (`DeriveRoutingSwitchReason`, projection) |
| Redline (linhas provider/profile/remaining/state/reset) | Overclock | ADAPTED (faixa de monitor no `UsageAccountCard`, visual Nexus) |
| Harness/Receitas como apresentação | Overclock | ADAPTED como execution summary; entidade Harness/Recipe DEFERRED |
| OverMemory como referência | Overclock | ALREADY_EXISTED (fortalecido `IntelligenceMemory`, sem engine novo) |
| Universal proxy/gateway, 40 providers | OpenCodex | DEFERRED (ver FUTURE doc) |
| SpendLedger/billing/price catalog | — | DEFERRED |
| Marketplace/remote registry | — | DEFERRED |
| MCP server/voice/multimodal | — | DEFERRED |
| Nova arquitetura Engine/Backend, novo banco, daemons | — | REJECTED nesta rodada |

## Deliberadamente NÃO implementado

Proxy/gateway, 40+ adapters, SpendLedger, Marketplace, MCP, Voice, multimodal,
Recipes/Harness persistentes, Engine≠Backend split, novo banco, novos daemons,
tray, battery/activity detector, probes de produção (registry vazio documentado).

## Testes / gates

- `go test ./internal/...`: PASS (zero FAIL após correção do classificador).
- `-race`: capacitymonitor, store, cooldown, fallback, events: PASS.
- `go vet` + `gofmt` + `git diff --check`: limpos.
- `make build`, `make web-verify` 10/10, `make test-e2e`: PASS.
- Web: 31 testes nas áreas tocadas + 5 novos do mapper: PASS; typecheck/lint/stylelint/i18n/build: PASS.

## Dívida restante

Probes de produção por provider; `make docs-verify`; CI same-SHA; evidências
nativas/live/overnight; throttle de `onCapacityChange` (reload cheio após ack).

## GO / NO_GO (merge do branch)

**GO local**: base correta, sem duplicação arquitetural, gates verdes,
adendo honesto do que é UNVERIFIED. Falta: revisão humana do diff, commits,
push autorizado, CI same-SHA, e o título novo do PR #3.

## Adendo 2026-09-20 — merge closure (sem features)

Rodadas posteriores (surface isolation, deep review, merge closure) não
alteraram a arquitetura Capacity: T0–T9 intactos, sem subsistema novo.
Terminologia mantida: monitor infrastructure = implemented; production
per-provider probes = pending/deferred (nunca "monitoramento real
completo"). CI #143 vermelho foi fechado localmente (topbar @1280,
SA9003 ×2, E2E surface/capacity ligados ao Browser CI, h1 sr-only);
push + CI same-SHA do HEAD final pendem de credencial. Estado e veredito
atuais em `DEV/validation/SURFACE_ISOLATION_REPORT.md` (§11–§12).
