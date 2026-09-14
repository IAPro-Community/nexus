# NEXUS 1.0 FINAL ACCEPTANCE

Audit date: 2026-09-07  
Candidate SHA: `5f51d03985f6ca48186f0cb12b3e97a5294600e3` (working tree dirty)  
Branch: `feat/nexus-maximum-delivery`  
Remote: `origin` → `https://github.com/IAPro-Community/nexus.git`

Fresh remote evidence: CI run [34159526843](https://github.com/IAPro-Community/nexus/actions/runs/34159526843)
completed for the current SHA with Frontend, Linux, Security, Desktop Linux and
Desktop Windows passing. macOS E2E, Browser E2E and Desktop macOS failed; the
GoReleaser Snapshot was skipped. The available GitHub token is invalid/rate
limited for raw logs, so job conclusions are recorded without inventing root
causes.

Root cause confirmed for the Frontend failure: running Prettier against the
exact SHA checkout flags five generated files (`web/src/nexus/api.ts`,
`web/src/wailsjs/wailsjs/go/desktop/App.d.ts`, `models.ts`,
`runtime/package.json`, and `runtime/runtime.d.ts`). The current dirty worktree
contains their formatting correction and local `bun run format:check` passes;
the correction still needs to land in a new immutable commit before CI can
re-run. Windows/macOS failures remain untriaged because GitHub logs require
re-authentication.

This is an evidence ledger, not a feature inventory. `PASS` requires a fresh
executable result in this checkout. Historical reports are marked as context,
not reused as same-SHA proof.

## Product

| P0 | Status | Evidence |
| --- | --- | --- |
| Resume-first UX | CONDITIONAL | Overview now derives Needs You/Continue/Recent lanes from project runtimes and receives project Flow Runs; focused model tests and frontend verify pass, but fresh return-session E2E remains unproven |
| Needs You contract | PASS (local) | Structured `runner.HumanIntervention`; runner tests and durable state paths |
| Terminal continuity | CONDITIONAL | Linux continuity tests pass; native Windows/macOS execution unavailable |

## Autonomy

| P0 | Status | Evidence |
| --- | --- | --- |
| Stagnation recovery | PASS (local) | `go test ./internal/nexus/runner`; remediation persists `StrategyVariant` and no-progress tests pass |
| Artifact communication | PASS (local) | Context capsule/work receipt producer-consumer tests; restart persistence tests |
| Worktree safety | PASS (local) | Isolation and canonical-path regression tests; adversarial live two-writer run not executed |
| Agent crash recovery | BLOCKED_EXTERNAL | Requires killing an authenticated provider process and observing recovery lineage |
| Nexus restart recovery | PASS (local) / CONDITIONAL live | Durable runner restart tests pass; active provider restart scenario not run |
| Provider fallback | BLOCKED_EXTERNAL | Requires provider authentication/quota/outage injection |
| Overnight acceptance | PASS (deterministic sandbox) | `TestOvernightAcceptanceSandbox`: parallel plan, dependency receipt, injected failure, remediation and global DoD |
| Real Nexus dogfooding | FAIL | No recorded real self-hosted mission with zero post-GO intervention |

## Platforms

| Surface | Status | Evidence |
| --- | --- | --- |
| Linux | PASS (local) | `go test ./...`, `go vet ./...`, frontend build/typecheck and embedded sync |
| Windows | UNVERIFIED | Cross-build evidence exists; no native ConPTY/Named Pipe/desktop smoke in this environment |
| macOS | UNVERIFIED | Cross-build evidence exists; no native PTY/keychain/desktop smoke in this environment |
| Desktop Linux | CONDITIONAL | Existing Wails evidence is historical relative to candidate SHA |
| Desktop Windows | UNVERIFIED | Native runner unavailable |
| Desktop macOS | UNVERIFIED | Native runner unavailable |

## Quality

| Gate | Status | Fresh evidence |
| --- | --- | --- |
| Frontend typecheck | PASS | `cd web && bun run typecheck` |
| Frontend tests | PASS | `cd web && bun run test` (61 files / 311 tests) |
| Frontend lint/style | PASS | `bun run lint && bun run lint:styles && bun run check:styles` (pre-existing warning only) |
| Frontend build/embed | PASS | `node web/scripts/build.mjs` and `cd web && bun run verify` (10/10 gates, 2026-09-07T20:49:46Z) |
| Backend tests | PASS | `go test ./...` and `make quality` |
| Backend vet | PASS | `go vet ./...` |
| Runner/store race | PASS | `go test -race ./internal/nexus/runner ./internal/nexus/store` |
| Browser E2E/Axe/visual | PASS (local current checkout) | Hardening E2E plus `node web/scripts/task-preparation-visual-verify.mjs` passed on `ec6badc` + dirty fixes; six breakpoints, Axe, settings and task-preparation screenshots |
| Security | PASS (local) | `make security` — No vulnerabilities found |
| Documentation evidence | CONDITIONAL | `make docs-verify` correctly rejects visual manifest anchored at historical SHA while visual files are dirty; regenerate/commit manifest on candidate SHA |
| Packaging | CONDITIONAL | Existing Linux/package evidence; no same-SHA native installer matrix |

## Release

| Gate | Status | Reason |
| --- | --- | --- |
| Same-SHA matrix | FAIL | Exact-SHA CI run 34159526843: Frontend/Linux/Security/Desktop Linux+Windows PASS; macOS E2E, Browser E2E and Desktop macOS FAIL; Snapshot skipped |
| Clean install | CONDITIONAL | Linux evidence exists; native install smoke unavailable |
| Upgrade path | CONDITIONAL | Existing tests cover policy; no full three-platform install/upgrade run |
| Release artifacts | CONDITIONAL | Local artifacts exist historically; candidate SHA publication not performed |

## Verdict

`NO_GO`

The local control-plane implementation is buildable and the deterministic safety
paths pass, including a new unattended sandbox with injected failure and global
DoD verification. Release Candidate promotion is still blocked by missing
real-world proof: Nexus dogfooding, authenticated crash/provider recovery,
same-SHA CI completion, and native Windows/macOS validation remain outstanding.
The browser result is locally green after fixing bootstrap-state handoff and a
brittle ancestor hit-test assertion, but that fix is uncommitted and therefore
cannot be treated as same-SHA CI proof yet.

The exact-SHA CI run also exposed a workflow portability bug: Desktop macOS
used Bash-4-only `mapfile` and failed with exit code 127. The workflow now uses
Bash-3.2-compatible array reads locally; a new CI run is required to prove the
fix. Windows/macOS diagnostic steps now continue after individual failures and
finish with an explicit aggregate assertion, so the next run will preserve
failure semantics while exposing all platform logs.

The workflow snippets were parsed as YAML and every macOS Bash run block passed
`bash -n` locally. This validates syntax only; native execution remains a CI
responsibility.

The release gate now also compares the CI API's `headSha` with the requested
candidate SHA before accepting the run; mismatches are ignored and retried.

## Fresh remote evidence — pushed SHA `059bb5c` (run `34164359845`)

- `headSha` confirmado pela API: `059bb5cc3ca0f36e8e430182d2e66c51d5993897`.
- Linux E2E e Security passaram.
- Frontend falhou no passo `Format Check` antes de typecheck/test/build.
- Windows E2E executou testes, ConPTY, runtime, build e smoke com sucesso, mas
  o `Assert Windows diagnostics` falhou.
- macOS E2E executou vet, race, PTY, socket, Web, build e installer com sucesso,
  mas o `Assert macOS diagnostics` falhou.
- Desktop, Browser E2E e GoReleaser foram pulados após os gates anteriores.
- Logs detalhados não estão acessíveis por `gh run view --log-failed` (HTTP 403,
  endpoint exige admin); os passos e conclusões foram confirmados pela API.
- O checkout local está em `772bd2e`, um commit à frente do SHA publicado, e
  contém alterações adicionais ainda não publicadas. PASS local posterior não
  pode ser atribuído ao run `34164359845`.
- A falha de formato do SHA publicado foi reproduzida e corrigida localmente em
  `web/src/features/work/ComposerSurface.tsx`; a checagem Prettier usada pela
  CI passa no checkout atual. O workflow também passou a imprimir explicitamente
  os outcomes individuais dos diagnósticos Windows/macOS antes do agregado.

## CI instrumentation pending next run

- O Browser job agora grava logs separados para `test:e2e`, `test:a11y` e
  `test:visual`, publica `browser-diagnostics-${{ github.sha }}` e só então falha
  no agregado se o passo de verificação tiver outcome `failure`.
- O build Desktop macOS agora publica `desktop-macos-build-diagnostics-${{ github.sha }}`
  mesmo em falha. A próxima execução é necessária para identificar a causa
  nativa, pois logs históricos exigem permissão de administração.

## Exact blockers and reproduction

1. Run the authenticated provider-backed overnight scenario and save mission
   timeline/artifacts; the deterministic sandbox is covered locally, but no
   provider-backed harness is configured here.
2. Run the self-hosted Nexus mission and record mission ID, agents, worktrees,
   artifacts, remediation and global verification on one immutable SHA.
3. Execute the native Windows and macOS jobs (ConPTY/PTY, desktop and installer)
   and attach logs/screenshots to this SHA.
4. Re-authenticate `gh` (current token returns HTTP 403/rate-limit for logs), inspect run
   34159526843 failure logs, fix the smallest root causes, and rerun all release
   gates on the resulting immutable SHA.

5. The local direct-work harness now resolves bootstrap through persisted
   `nexus web url`; run it with authenticated provider variables to produce the
   missing provider-backed evidence.

6. Fresh local attempt on 2026-09-07: `go run ./scripts/nexus-e2e-local.go
   -start -port 3101 -browser` started the real server, then returned
   `bootstrap did not establish an authenticated session`; browser smoke also
   reported `NEXUS_BROWSER_SMOKE_NOT_RUN` because no Playwright executable is
   installed. This is recorded as `BLOCKED_EXTERNAL`, not a passing test.
