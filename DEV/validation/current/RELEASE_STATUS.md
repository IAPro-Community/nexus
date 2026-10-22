# Current release status

> **Atualizacao 2026-10-22:** popup independente e Mascote controlaveis em
> `f789db63` (HUD SHA-256 `1f3ced07...7929de`). O modo Mascote era
> inalcancavel porque o popup so existia com `NEXUS_HUD_DETACHED_POPOVER=1`;
> agora ha um toggle persistido no painel, aplicado em runtime. Evidencia real
> em `DEV/validation/mascot-toggle-f789db63/homologation/RESULT.json` (PASS,
> duas sessoes com reinicio). Gates verdes (`gofmt`, `go vet`, `go test ./...`,
> Web quality 97/599). O veredito segue **NO_GO_WINDOWS** por assinatura e
> smokes interativos.

> **Atualizacao 2026-10-21 (2):** click-through do HUD corrigido e provado em
> `047f4348` (HUD SHA-256 `19c5e299...a1e6a5`): clique na area transparente
> atravessa para a janela atras e a capsula continua interativa. Gates verdes
> (`gofmt`, `go vet`, `go test ./...`, Web quality 97/598, `docs-verify`
> 192/33). O veredito de release segue **NO_GO_WINDOWS** por assinatura e
> smokes interativos. Detalhes em
> `DEV/validation/local-hud-047f4348/HOMOLOGATION.md`.

> **Atualizacao 2026-10-21:** codigo homologado em `7d92b942` (HEAD de docs
> `42a6971b`), PR #54 integrado. Gates automatizados PASS (9/9), mas o
> veredito segue **NO_GO_WINDOWS**: falta instalador assinado, smokes
> interativos de Codex e o click-through do HUD falhou. Detalhes em
> `DEV/validation/local-hud-7d92b942/HOMOLOGATION.md`.

> **Estado canÃ´nico para a beta.10 (2026-10-08):** cÃ³digo de PR #45
> e PR #46 conciliado na PR #47, aguardando incorporaÃ§Ã£o Ã  branch
> `release/v1.0.0` e execuÃ§Ã£o dos gates no SHA final.
> **NO-GO para tag, instalador assinado e distribuiÃ§Ã£o.**
> HistÃ³rico Windows beta.7/beta.8 abaixo permanece apenas como evidÃªncia
> de SHAs antigos e **nÃ£o certifica beta.10**.
> Matriz detalhada: [convergÃªncia PR45/46](BETA10_PR45_PR46_CONVERGENCE_20261008.md)
> e [auditoria tÃ©cnica](BETA10_ENGINEERING_AUDIT_20261008.md).

CURRENT_CANDIDATE_VERSION: 1.0.0-beta.10
CURRENT_CANDIDATE_BRANCH: release/v1.0.0
CURRENT_CANDIDATE_CODE_SHA: f789db63c8ec9f3498c3822a94ee7c823ec7086c
CURRENT_CANDIDATE_DOCS_HEAD: pending docs/evidence commit after f789db63
CURRENT_CANDIDATE_NATIVE_GATES: PASS_AUTOMATED_EXACT_SHA (9 PASS, 4 NOT_RUN)
CURRENT_CANDIDATE_DISTRIBUTION: NO-GO (sem certificado Authenticode; native-signing com 0 secrets)
CURRENT_CANDIDATE_CI: RED_BY_INFRA (Playwright CDN TLS) + runs anteriores cancelados por push
CURRENT_CANDIDATE_HUD_NATIVE_WINDOWS: FUNCTIONAL_PASS_20_20 / CLICK_THROUGH_PASS
CURRENT_CANDIDATE_HUD_MASCOT_TOGGLE: PASS_EXACT_SHA (f789db63 code / 955b197f build, HUD 1f3ced07 -> deployed 8364a942)
CURRENT_CANDIDATE_HUD_CROSS_PLATFORM: INCONCLUSIVE (sem Linux X11 e macOS no mesmo SHA)
LOCAL_WINDOWS_BETA10_REFERENCE_SHA: 047f43481dfdcd6f2f77072cc2002ea8ee5209df (integral)
LOCAL_WINDOWS_BETA10_AUTOMATED_GATES: PASS_ON_REFERENCE_SHA_ONLY
LOCAL_WINDOWS_BETA10_MANUAL_PROVIDER_SMOKES: NOT_RUN
LOCAL_WINDOWS_BETA10_EVIDENCE: DEV/validation/local-hud-047f4348/HOMOLOGATION.md
LATEST_HUD_MASCOT_EVIDENCE: DEV/validation/mascot-toggle-f789db63/HOMOLOGATION.md
LOCAL_HUD_DEPLOY: 955b197f installed on 2026-10-22 (server PID 24184, HTTP 200; HUD PID 21168 --linked) - see DEV/validation/mascot-toggle-f789db63/DEPLOYMENT.md
LOCAL_HUD_REAL_PROFILE: PASS (toggle + mascot + tray restore in the user's own profile) - DEV/validation/mascot-toggle-f789db63/real-profile/REAL_PROFILE.md
CURRENT_CANDIDATE_COMMIT: 8d100e96
LOCAL_HUD_TOP_TOGGLE: PASS (toggle first-fold in inline panel and popup; installed build 9b78fb10) - DEV/validation/mascot-toggle-8d100e96/
CURRENT_RELEASE_HEAD_WINDOWS_EXACT_SHA_GATES: PASS (certify-windows-first-run, Go 1.26.10)
PREVIOUS_VALIDATION_RECORDS: HISTORICAL_ONLY

## Legacy Windows beta.7 baseline (historical)

STATUS: WINDOWS_AUTOMATED_LOCAL_GATES_PASS_RELEASE_NO_GO
OFFICIAL_RELEASE: NO-GO
WINDOWS_DESKTOP_RETEST: PASS_RUNTIME (Wails window and WebView2 child persisted 9s on validated source SHA; isolated profile created; visual rendering unverified)
SOURCE_VERSION: 1.0.0-beta.7
CANDIDATE_REF: release/v1.0.0
REQUESTED_WINDOWS_SHA: d14154d2490dfdd3966c7fddc990a1d9127edd39
VALIDATED_WINDOWS_SOURCE_SHA: d840797bbd975a5c7edc54ebe23f3dfef56ff947
CURRENT_WORKTREE_BASE_SHA: d840797bbd975a5c7edc54ebe23f3dfef56ff947
CURRENT_WORKTREE_GATES: PASS (Windows AMD64 exact-SHA local automated gates; worktree was clean at start)
LAST_FROZEN_WINDOWS_SHA: d840797bbd975a5c7edc54ebe23f3dfef56ff947
REMOTE_TRACKING_SHA_AT_AUDIT: 52f35c04181b85379219ab55fc3192c63a5440a8
LIVE_REMOTE_SHA_AT_AUDIT: CONFIRMED (origin/release/v1.0.0 = 52f35c04181b85379219ab55fc3192c63a5440a8 before this continuation)
REMOTE_RELEASE_PUSH: PASS (fast-forward 52f35c04 -> f1928d0f; code validation SHA d840797b is an ancestor)
LIVE_REMOTE_SHA_AFTER_PUSH: f1928d0f75cf49a0968f248bd7dc9b8e54655dcb (confirmed by git ls-remote)
REMOTE_STATUS_SYNC_PUSH: PASS (follow-up documentation-only fast-forward f1928d0f -> 5a08a1807c4489395dca86ed198f268ea676d2e4)
LIVE_REMOTE_SHA_AT_FINAL_AUDIT: 5a08a1807c4489395dca86ed198f268ea676d2e4 (confirmed before this status-record commit; validated source d840797b remains an ancestor)
REMOTE_BETA7_TAG: ABSENT (live origin has no refs/tags/v1.0.0-beta.7; local tag absent)
PUBLIC_GITHUB_RELEASE: NOT_REVALIDATED (not queried)
WINDOWS_LOCAL_GATES: PASS (PowerShell/docs, Bun frozen + quality:full, Go fmt/vet/lint/security/tests, control/runtime and Agent Interop regressions, 8 browser E2E suites + accessibility/visual, CLI/Desktop/HUD builds and isolated Desktop/HUD native smokes)
WINDOWS_NATIVE_CI: NOT_REVALIDATED (gh unavailable; unauthenticated Actions API returned 404 after push; exact-SHA workflow not confirmed)
WINDOWS_SIGNING: BLOCKED_EXTERNAL
WINDOWS_SIGNING_ENV: NOT_CONFIGURED
WINDOWS_SIGNING_TOOLS: signtool.exe, Authenticode certificate/secrets, makensis.exe, NEXUS_UPDATE_PRIVATE_KEY and NEXUS_UPDATE_KEY_ID unavailable on this host; checked-in public UPDATE_PUBLIC_KEY is valid
LOCAL_ARTIFACTS: CLI/Desktop/HUD binaries and ZIPs are NotSigned; checksums at DEV/validation/local-beta/1.0.0-beta.7/d840797bbd975a5c7edc54ebe23f3dfef56ff947/windows-AMD64/distribution-artifacts/CHECKSUMS.md
WINDOWS_INSTALLER_BUILD: NOT_RUN on validated SHA (makensis.exe unavailable; prior NSIS evidence is historical)
WINDOWS_EXACT_INSTALLER_SMOKE: NOT_RUN
WINDOWS_CONPTY_OPT_IN_SMOKE: NOT_RUN on d840797b (historical smoke exists on earlier Windows SHA)
WINDOWS_MANUAL_PROVIDER_MISSION_VOICE_SMOKES: NOT_RUN on d840797b
WINDOWS_VISUAL_DESKTOP_RENDER: NOT_VERIFIED (Wails/WebView2 smoke passed in hidden mode; no interactive frame capture)
WINDOWS_DOCS_VERIFY: PASS (177 Markdown files, 33 required assets/files)
LOCAL_SOURCE_DELTA: bounded Git identity subprocesses and isolated short TEMP for Windows Go tests; commits d1208720 and d840797b on release/v1.0.0

WORKTREE_FROZEN: YES_FOR_VALIDATED_SOURCE (d840797b was clean at gate start; status/report updates are documentation-only)

LIVE_BETA7_BRANCH_AUDIT: COMPLETE (no new branch created; KIVER d14154d2 is an ancestor of validated release candidate; no merge/cherry-pick required)
UPDATED_AT: 2026-10-08

## Windows exact-SHA local validation â€” 2026-10-08

The complete automated Windows AMD64 homologation passed on source SHA `d840797bbd975a5c7edc54ebe23f3dfef56ff947`. It used Go 1.25.14, Bun 1.3.9 and Node 22.14.0. Go tests reported 65 passing package results and zero failures. The literal `bun install --cwd web --frozen-lockfile` and `bun run --cwd web quality:full` passed with 82 files and 487 tests. Eight browser E2E suites plus accessibility/visual tasks passed. Desktop and HUD builds and isolated native launch smokes passed.

Evidence directory: `DEV/validation/local-beta/1.0.0-beta.7/d840797bbd975a5c7edc54ebe23f3dfef56ff947/windows-AMD64/`. Distribution candidate executables and ZIPs are unsigned; see `distribution-artifacts/CHECKSUMS.md`.

The live remote check found `origin/release/v1.0.0` at `52f35c04181b85379219ab55fc3192c63a5440a8` before this continuation. No Beta 7 tag was found. The local KIVER branch at d14154d2 is an ancestor; no new branch requires merge. GitHub Actions, GitHub Releases inventory, NSIS installer build, Authenticode, provider/Mission/Voice manual smokes and visible Desktop rendering remain unverified. The release is **NO-GO for tagging/publication** until these gates close. This continuation ran no Linux gates.

## Continuidade Windows â€” perfil WebView2 e fechamento de smokes â€” 2026-10-07

Base do checkout: release/v1.0.0 em 52f35c04181b85379219ab55fc3192c63a5440a8; VERSION=1.0.0-beta.7. A Ã¡rvore de trabalho contÃ©m alteraÃ§Ãµes locais e nÃ£o Ã© um SHA imutÃ¡vel homologado.

PASS â€” apÃ³s os ajustes de perfil e smoke, go fmt ./..., go vet ./..., govulncheck ./..., golangci-lint run ./... e go test -count=1 ./... no Windows AMD64. O teste desktop tambÃ©m confirmou que WindowsWebviewUserDataPath usa LOCALAPPDATA.

PASS â€” workflow ci.yml validado como YAML e scripts/smoke-desktop-windows.ps1 validado pelo parser PowerShell. O workflow constrÃ³i binÃ¡rios de smoke com IDs Wails separados; os IDs de produÃ§Ã£o permanecem com seus valores originais. Os smokes exigem janela/controlador WebView2 e diretÃ³rio de perfil efetivamente criado.

PASS â€” Desktop Wails IAPro Nexus e filho Chrome_WidgetWin_0 permaneceram ativos por 9 segundos; perfil isolado criado em LOCALAPPDATA\IAPro\Nexus\desktop\WebView2. HUD permaneceu ativo por 8 segundos e criou LOCALAPPDATA\IAPro\Nexus\hud\WebView2. Artefatos de smoke foram construÃ­dos com IDs exclusivos de teste, nÃ£o sÃ£o binÃ¡rios de distribuiÃ§Ã£o.

A execuÃ§Ã£o local com o ID de instÃ¢ncia de produÃ§Ã£o terminou com cÃ³digo 0. O caminho Wails para uma segunda instÃ¢ncia encerra com o cÃ³digo configurado, mas o proprietÃ¡rio do mutex nÃ£o pÃ´de ser inspecionado nesta sessÃ£o; nenhum processo do usuÃ¡rio foi encerrado. O binÃ¡rio de smoke com ID temporÃ¡rio isolado confirmou o caminho nativo.

Auditoria de branches locais/refspecs em cache: fix/beta7-windows-KIVER-20261007 estÃ¡ em d14154d2490dfdd3966c7fddc990a1d9127edd39 e Ã© ancestral de release/v1.0.0 em 52f35c04181b85379219ab55fc3192c63a5440a8. A release contÃ©m cinco commits posteriores; a branch fix tem zero commits exclusivos. NÃ£o hÃ¡ merge pendente. Os refs origin locais nÃ£o foram atualizados ao vivo.

A release continua NO-GO: HEAD nÃ£o Ã© o SHA d14154d2 informado no pedido original, mudanÃ§as nÃ£o foram commitadas, estado remoto/CI nÃ£o foi revalidado, assinatura/trust root e rebuild final do instalador seguem pendentes. Nenhum gate Linux foi executado nesta continuaÃ§Ã£o.

## RevalidaÃ§Ã£o Windows do worktree com correÃ§Ãµes locais â€” 2026-10-07

O HEAD permanece em 52f35c04181b85379219ab55fc3192c63a5440a8 na branch release/v1.0.0, e VERSION contÃ©m 1.0.0-beta.7. O worktree estÃ¡ sujo: estas correÃ§Ãµes ainda nÃ£o pertencem a um SHA imutÃ¡vel nem foram commitadas.

### CorreÃ§Ãµes reproduzidas

- Wails v3 encaminha http://wails.localhost ao Core embutido. A polÃ­tica reconhecia a origem desktop, mas rejeitava o Host correspondente e retornava invalid origin. Agora aceita somente o host exato wails.localhost junto de uma origem desktop reconhecida.
- Foram adicionados testes negativos para hosts lookalike e origens externas e um teste HTTP autenticado para bootstrap e criaÃ§Ã£o de projeto via host Wails.
- Os estilos do ProjectHub foram migrados para SCSS Module. O erro ocupa a largura do formulÃ¡rio, recebe role=alert e usa o token de perigo; o botÃ£o mantÃ©m Adicionar Projeto em uma linha.
- O hardening E2E reproduz HTTP 403 no viewport Windows 775x483, verifica o layout e captura evidÃªncia.

### EvidÃªncias Windows desta continuaÃ§Ã£o

- PASS â€” go fmt ./..., go vet ./..., govulncheck ./..., golangci-lint run ./... e go test -count=1 ./... no checkout Windows.
- PASS â€” bun install --cwd web --frozen-lockfile.
- PASS â€” bun run --cwd web quality:full com Bun 1.3.9 e Node 24.19.0: allowlist, Prettier, ESLint, Stylelint, TypeScript, 82 arquivos/487 testes Vitest e build.
- PASS â€” as oito suÃ­tes Playwright passaram no worktree Windows isolado com as mesmas alteraÃ§Ãµes de frontend/origin; o caso visual mediu 775x483.
- PASS â€” builds locais CLI, Desktop e HUD para Windows AMD64; a CLI reporta 1.0.0-beta.7 e 52f35c04-dirty.
- PASS â€” o HUD permaneceu ativo por oito segundos e registrou criaÃ§Ã£o do ambiente WebView2.
- INCONCLUSIVO â€” o Desktop encerrou com cÃ³digo 0 antes de oito segundos. O binÃ¡rio Desktop do baseline local e o artefato isolado anterior tiveram o mesmo comportamento no launcher disponÃ­vel, sem erro nos logs. Repetir smoke em sessÃ£o Windows interativa.
- PASS â€” checksums calculados; todos os binÃ¡rios sÃ£o NotSigned e nÃ£o incluem trust root de atualizaÃ§Ã£o.

### Estado de fechamento

A validaÃ§Ã£o de cÃ³digo Windows passou, mas o worktree nÃ£o estÃ¡ congelado. Os binÃ¡rios e checksums sÃ£o evidÃªncia local, nÃ£o pacotes de distribuiÃ§Ã£o. NÃ£o criar tag nem publicar: o workflow exact-SHA continua sem runner compatÃ­vel, faltam certificado/secrets Authenticode e NEXUS_UPDATE_PUBLIC_KEY, e o smoke Desktop precisa ser repetido em sessÃ£o interativa. Nenhum gate Linux foi executado nesta continuaÃ§Ã£o.

> **Current release truth moved to `docs/release-status.md`.** This DEV file
> remains an evidence ledger. The 2026-09-23 SHA/run records below are historical
> and must not override the current `VERSION`, candidate ref, platform policy,
> tag inventory, release inventory or current CI state.

## Canonical beta integration status â€” 2026-09-30

`release/v1.0.0` is the only integration line for Nexus 1.0 retesting. Beta 7 is the current internal candidate. Wails v3 has already been selectively ported onto this line while preserving newer Agent Projection, ExecutionRole, Direct/Mission, OpenCode and homologation contracts. Future beta increments change the internal version/manifest, not the release branch.

The authoritative open-gap checklist is GitHub issue #27. Source integration remains **UNVERIFIED** until native same-SHA evidence closes the required platform gates.

Beta 7 preserves the Usage/Capacity hardening and adds the Composer Interview Engine plus the latest Codex/delegation fail-closed corrections. The public quota field is literal remaining capacity, while multi-pool breadth stays an internal scheduler concern. Release automation now accepts only the exact HEAD of `release/v1.0.0`.

## Closure sweep â€” 2026-09-30

The final source/admin sweep was completed against audited source SHA `32b0cc3611850b744ba4c3191d35835d0744ce82`.
No functional delta was ported from stale PRs #31, #33, #34 or #35; all four were
closed unmerged after review because their remaining deltas are superseded,
deferred, or conflict with the current fail-closed routing/visual-evidence
contracts.

GitHub Actions exact-SHA execution is currently **BLOCKED_EXTERNAL** because the
available Actions credits cannot start the queued self-hosted gate. This is an
operational dependency, not a technical PASS or FAIL. The source candidate must
still complete real visual recapture, native homologation and authenticated
provider smoke on the final immutable SHA before certification/publication.

## Current candidate summary

- source version: `1.0.0-beta.7`;
- candidate ref: `release/v1.0.0`;
- source matrix integrated: Linux amd64 + Windows amd64 + macOS arm64;
- public binary release inventory: empty in both the private source repository
  and `IAPro-Community/nexus-releases`;
- `v1.0.0-beta.1` is a historical source tag, not a published binary release;
- pull-request CI is available; it validates the synthetic merge result only;
- exact-SHA push/workflow_dispatch CI on the final `release/v1.0.0` commit remains mandatory;
- beta publication remains fail-closed until local homologation plus the
  exact-SHA CI/native certification policy defined by the current workflows.

For actionable release state, gates and commands use
[`docs/release-status.md`](../../../docs/release-status.md).

## Historical 2026-09-23 closure evidence

PR #4 (`feat/nexus-1.0-closure`) was squash-merged into `main` on 2026-09-23.
The source-closure head `0d6e9c667af39b7534002d060848acfc9eb8e265` passed CI #668 before merge.
The resulting `main` SHA `5a61d3152b23b002b39e8b4b8988924db39956b2` then passed exact-SHA push CI #669
(run `35849031522`) with the Linux amd64 v1 matrix green: Frontend, Linux E2E,
Security, Release configuration, Browser E2E/Axe/visual, Desktop Linux Wails,
GoReleaser Snapshot and Smoke DEB. Windows amd64 and macOS arm64 remained
DEFERRED / UNVERIFIED and were skipped by policy, not counted as PASS.

CI #669 produced exact-SHA artifacts named with `5a61d3152b23b002b39e8b4b8988924db39956b2`.
No stable `v1.0.0` publication is authorized yet: Stable Certification, signing,
authenticated provider/Mission checkpoints and the live 8-hour soak are still pending.
Any commit after `5a61d315...` creates a new candidate SHA and requires fresh exact-SHA
CI before Stable Certification can authorize publication.

### Historical baseline

PR #3 was merged into `main` as
`7605f702cce220b4111d51d06a34b37f7e0a9a8d` after same-SHA CI #149 completed
successfully on the PR head `300ccbeee4f145f80ebce13c9cefafcb01acefc3`.

The verified PR-head gates were:

1. Frontend format/lint/styles/typecheck/unit/build/verify â€” PASS
2. Linux Go E2E, full race suite, vet and golangci-lint â€” PASS
3. Security / govulncheck â€” PASS
4. Browser hardening â€” PASS
5. Surface Isolation + terminal continuity â€” PASS
6. Capacity API/browser E2E â€” PASS
7. Axe and visual smoke â€” PASS
8. Desktop Linux Wails build â€” PASS
9. GoReleaser Linux snapshot â€” PASS
10. DEB install smoke â€” PASS

The final candidate source version now reports `1.0.0`. This does NOT mean a public
`v1.0.0` tag or stable release has already been certified or published. Repository
tag inventory still exposes `v1.0.0-beta.1` as the published beta tag, so installation
documentation that pins beta.1 remains an artifact reference until stable certification
authorizes the final tag.

The post-merge `main` push CI run `35547840452` is also retained as evidence:
Linux and Security passed, Frontend was cancelled, and dependent jobs were skipped.
That run is therefore NON-GREEN/PARTIAL and must not replace the fully green PR-head
CI #149 baseline.

## PR merge-result identity

GitHub pull-request CI is intentionally merge-result evidence: the workflow run is
associated with the PR head, while the default checkout executes GitHub's synthetic
merge commit for that head against the target branch. This validates mergeability but
must not be described as literal exact-HEAD execution.

Exact-SHA release certification is a separate gate: the final release/tag workflow must
run and pass on the immutable release SHA itself. Artifact provenance and stable-release
claims must use that exact release SHA, not a pull-request synthetic merge SHA.

## Active release program

The source-closure PR has been merged. The active program is now the Linux amd64
stable-certification and publication path. The canonical closure contract remains
`DEV/SPECS/NEXUS_1_0_CLOSURE.md`.

The program must close, with evidence:

- production Capacity probes;
- Agent Modes and canonical AgentMatcher;
- Skills v1;
- cost attribution, budgets and anomaly detection;
- unified Work Mode;
- Mission 1.0 live certification and Evidence Bundles;
- Memory 1.0 on the existing IntelligenceMemory;
- Maestro contract v1 and graceful degradation;
- server-side entitlement enforcement for future paid capabilities;
- native Linux amd64 release certification;
- Windows/macOS certification tracked as post-v1 platform expansion, not a v1 blocker;
- signed install/update/upgrade/uninstall evidence;
- real dogfooding and soak.

## Platform truth

The beta.6 source integrates **Linux amd64, Windows amd64 and macOS arm64**.

Public support remains evidence-bound: each platform advertised by a release must carry
fresh same-SHA native evidence for its applicable CLI/Core, Browser/Web, Desktop/HUD,
packaging and install/update lifecycle gates.

Windows and macOS are therefore **UNVERIFIED until native same-SHA certification passes**;
source integration is not itself a support claim. A public beta/stable release must not
advertise a platform whose required native evidence is missing.

Cross-compilation still must not be described as native certification.

## Distribution truth

The repository is private and the intended beta/customer distribution path is binary
artifacts, not source access. Release/update infrastructure exists, but a stable 1.0
release is NOT authorized by this document.

Before any stable release:

- final candidate CI must be green on the exact release SHA;
- every platform advertised by the release must complete its native same-SHA gates;
- update manifest/checksum/signature chain must be verified;
- install/upgrade/uninstall must be executed on supported platforms;
- release artifacts must be produced from the final clean candidate;
- external signing/notarization limitations must be stated truthfully.

## Stable certification contract

Stable release now has an executable certification boundary rather than a checklist-only
claim. `.github/workflows/stable-certification.yml` must complete successfully on the
exact candidate SHA before a non-prerelease version can pass `release.yml`.

The stable certification performs:

- exact-SHA and canonical update trust-root checks;
- exact-SHA candidate build and release/update contract tests;
- authenticated direct-provider probes;
- a black-box autonomous Mission that must end in `COMPLETED_VERIFIED`;
- an 8-hour process-level Web soak with HTTP health probes;
- a deliberate Web restart during the soak;
- a second provider/Mission checkpoint after the soak window.

The historical in-memory `TestOvernightSoakHarnessBounded` remains a state-machine
test only and is not accepted as stable-release soak evidence.

## Closure hardening â€” 2026-09-22

The closure branch was re-audited against the executable product and release
boundaries. The following source-level blockers were corrected rather than
deferred to certification:

- human confirmation of an unknown external dispatch no longer fabricates
  `VERIFIED`; the package returns to `TESTING` and must produce real verification
  evidence;
- Evidence Bundle no longer treats an already resolved human intervention as
  unresolved evidence;
- Composer Stop now aborts the active request, including attachment transfer, and
  asynchronous errors are announced accessibly;
- Composer narrow-layout overflow, mobile touch targets and the full theme/device
  visual matrix were hardened;
- CLI `nexus run` / `plan run` now remain alive with the canonical Mission worker
  instead of abandoning an in-process goroutine; terminal/blocked/failure states
  preserve non-zero CLI semantics;
- project registration can explicitly request `--isolation worktree`, avoiding an
  autonomous-Mission/default-isolation mismatch;
- GoReleaser v1 publication scope is Linux amd64 only; Windows/macOS source support
  remains in-tree but is not published or advertised as certified v1 support;
- Windows and Unix uninstallers are explicit/idempotent, preserve data by default,
  and the lifecycle includes DEB removal;
- the updater trust root and key ID have versioned sources of truth
  (`UPDATE_PUBLIC_KEY`, `UPDATE_KEY_ID`) and release secrets must match them;
- install scripts verify detached Ed25519 manifest signature, signed artifact digest
  and embedded binary version before replacement;
- the macOS installer correctly extracts the verified Desktop ZIP/`.app`; unsupported
  Desktop architectures no longer turn a valid CLI install into a failure;
- `nexus update check` now fails closed on registry/signature errors, update apply
  failures return a non-zero result, and channel migration is explicit via
  `--channel beta|stable`;
- alpha/beta/RC channel classification is consistent between signer, updater and
  release workflow;
- stable certification now exercises signed install, installer upgrade, self-update,
  tamper rejection, uninstall/purge, explicit betaâ†’stable promotion, authenticated
  provider/Mission execution and an 8-hour cumulative process-level soak with restart;
- release provenance rejects pull-request synthetic-merge CI as exact-SHA evidence;
  only exact-SHA `push` or `workflow_dispatch` CI can authorize artifacts;
- public installation/package documentation now matches the private-source /
  binaries-only distribution model and no longer advertises RPM/NSIS/winget as
  produced v1 artifacts.

The 8-hour soak is split into two four-hour execution segments because GitHub limits an
individual Actions step to 360 minutes. Both segments run on the same exact candidate
SHA and retain separate evidence.

### Evidence still external

The source-level closure and CI #669 are not equivalent to a published stable release.
Before stable `v1.0.0` can be published, the final immutable candidate SHA still needs:

1. green exact-SHA Linux/Web/release-preparation CI on that final SHA;
2. the protected signing environment with the canonical key/key-id pair;
3. successful Stable Certification on that same SHA, including signed lifecycle,
   authenticated provider/Mission checkpoints and both live-soak segments.

CI #669 is exact-SHA evidence for `5a61d3152b23b002b39e8b4b8988924db39956b2` only.
If this status synchronization or any later change is merged, #669 becomes historical
post-merge evidence and the new `main` SHA must pass exact-SHA CI again before certification.

Windows/macOS remain blockers for any release that advertises them until their native same-SHA certification is complete.

## Current verdict

`NEXUS_1_0_CANDIDATE = NO_GO`

Reason: beta.7 source closure is still evolving, and this newer candidate includes changes after
the historical CI #669 baseline. The final immutable candidate still needs fresh exact-SHA
homologation/certification for every platform it advertises, plus Stable Certification.
Windows/macOS remain UNVERIFIED until their native same-SHA evidence is complete.

## Reteste Windows do Desktop com cleanup WebView2 restrito â€” 2026-10-07

O script `scripts/smoke-desktop-windows.ps1` foi ajustado para encerrar apenas processos `msedgewebview2.exe` cuja linha de comando referencia o perfil isolado do Nexus; nÃ£o encerra processos WebView2 de outros aplicativos.

- PASS â€” script PowerShell parseado e build Desktop de smoke com ID de instÃ¢ncia temporÃ¡rio.
- PASS â€” janela Wails `IAPro Nexus` e controlador `Chrome_WidgetWin_0` ativos por 10 segundos; perfil em `LOCALAPPDATA\IAPro\Nexus\desktop\WebView2` criado.
- PASS â€” apÃ³s o cleanup, nenhum processo WebView2 apontava para o perfil isolado do teste.
- EvidÃªncia: `outputs/windows-desktop-smoke-cleanup-52f35c04-20261007/desktop-smoke.txt`; executÃ¡vel de smoke SHA-256 `86d93ae2854852d880e43fd3a81dd1e68a6b4eba109599184ad2e8fd0280623b`.
- Sem alteraÃ§Ã£o do ID de instÃ¢ncia de produÃ§Ã£o. O executÃ¡vel Ã© de smoke e nÃ£o Ã© artefato de distribuiÃ§Ã£o.

## Auditoria ao vivo de branches Beta 7 â€” 2026-10-07 (revisada)

Consulta ao vivo e fetch das refs remotas em 2026-10-07: `origin/release/v1.0.0` estÃ¡ em `52f35c04181b85379219ab55fc3192c63a5440a8`; a tag remota `v1.0.0-beta.7` nÃ£o existe. O conjunto de refs Beta 7 Ã© o mesmo da auditoria anterior; nÃ£o hÃ¡ branch Beta 7 nova que precise ser ligada Ã  release. A branch KIVER (`d14154d2490dfdd3966c7fddc990a1d9127edd39`) Ã© ancestral da release e nÃ£o tem commits exclusivos.

As branches abaixo tÃªm commits exclusivos no grafo, mas comparaÃ§Ã£o do cÃ³digo e das regressÃµes da release nÃ£o encontrou mudanÃ§a Beta 7 pendente que justifique merge ou cherry-pick:

| Branch | SituaÃ§Ã£o | DecisÃ£o |
| --- | --- | --- |
| `origin/fix/beta7-codex-terminal-protocol-p0` | 4 commits exclusivos no grafo; `terminal_output.go` e seu teste coincidem por blob com a release. A release tambÃ©m cobre rajadas de navegaÃ§Ã£o e continuaÃ§Ã£o de ESC. | Sem integraÃ§Ã£o pendente. |
| `origin/fix/beta7-terminal-trace-diagnostics` | 24 commits exclusivos no grafo; `terminal_trace.go` coincide por blob, e a release tem trace redigido, coalescÃªncia final e cobertura atualizada. | Sem integraÃ§Ã£o pendente. |
| `origin/test/beta7-terminal-final-boundary-regression` | 3 commits exclusivos no grafo; a release contÃ©m cobertura de raw-attach CSI/SS3, fragmentaÃ§Ã£o no limite final e privacidade do trace. | Sem integraÃ§Ã£o pendente. |
| `origin/fix/beta7-windows-autopilot-20261001` | EvidÃªncia documental Windows mais antiga, superada pelo relatÃ³rio KIVER. | NÃ£o integrar. |

TambÃ©m foi revisada `origin/codex/windows-beta6-convergence` (head de 2026-10-01; 5 commits exclusivos no grafo; branch anterior, fora do conjunto Beta 7). A release jÃ¡ contÃ©m os comportamentos relevantes de startup Wails, API same-origin/WebSocket no Core, descarte de runtime inativo e resoluÃ§Ã£o Windows de launchers `.cmd`. A Ãºnica diferenÃ§a de polÃ­tica identificada Ã© o commit antigo que torna ConPTY padrÃ£o para runtimes de provedores. A release restaurou ConPTY opt-in em `cc6400db` para evitar regressÃµes de TUI; o teste interativo opt-in passou no Windows nesta auditoria. A branch tambÃ©m inclui seleÃ§Ã£o OpenCode nÃ£o relacionada Ã  homologaÃ§Ã£o Beta 7. NÃ£o mesclar a branch inteira nem alterar o padrÃ£o ConPTY com base nela.

ConclusÃ£o da auditoria: nenhuma branch nova ou mudanÃ§a de cÃ³digo das branches examinadas precisa ser vinculada Ã  release neste momento. A release continua NO-GO para tag/distribuiÃ§Ã£o porque o worktree ainda nÃ£o foi congelado em um SHA homologado e os gates externos de assinatura/trust root e CI Windows exact-SHA continuam abertos. Nenhuma branch foi mesclada; nenhum commit, push, tag ou publicaÃ§Ã£o foi feito.

## RevalidaÃ§Ã£o final do Project Hub e dos gates Windows â€” 2026-10-07

A captura fornecida pelo usuÃ¡rio foi reproduzida no Project Hub em 775Ã—483. autoFocus fazia o painel rolÃ¡vel abrir deslocado e esconder o tÃ­tulo; o formulÃ¡rio tambÃ©m ultrapassava a altura disponÃ­vel. A correÃ§Ã£o remove o foco automÃ¡tico, mantÃ©m rolagem interna, compacta somente janelas de atÃ© 600px de altura e usa tÃ­tulo/descriÃ§Ã£o distintos e localizados nos modelos. O E2E valida abertura, formulÃ¡rio com erro 403, rolagem, overflow horizontal e os trÃªs textos. Captura: outputs/windows-e2e-layout-52f35c04-dirty-20261007/e2e_project_hub_error_775x483.png.

- PASS â€” bun install --cwd web --frozen-lockfile; Bun 1.3.9, sem mudanÃ§as de dependÃªncias.
- PASS â€” bun run --cwd web quality:full com Node 22.14.0: allowlist CSS, Prettier, ESLint, Stylelint, TypeScript, 82 arquivos/487 testes Vitest e build.
- PASS â€” oito suÃ­tes Playwright no bundle final: test:e2e, composer, surface, capacity, hud, composer-spine, composer-vague-spine e composer-themes. Logs finais em outputs/windows-e2e-layout-52f35c04-dirty-20261007.
- PASS â€” builds Windows AMD64 CLI/Desktop/HUD. CLI informa 1.0.0-beta.7, 52f35c04-dirty e Go 1.25.14; doctor --json retorna JSON vÃ¡lido e exit 0.
- PASS â€” smoke Desktop: janela IAPro Nexus e filho Chrome_WidgetWin_0 por 9s; HUD por 8s; ambos com IDs e perfis WebView2 isolados.
- PASS â€” SHA-256 dos trÃªs binÃ¡rios em outputs/windows-final-artifacts-52f35c04-dirty-20261007/CHECKSUMS.md e SHA256SUMS.txt.
- LIMITAÃ‡ÃƒO â€” executÃ¡veis Authenticode NotSigned, sem update trust root. A API GitHub sem autenticaÃ§Ã£o retornou 404 ao consultar run 37666126714; o estado remoto do run nÃ£o foi confirmado. Instalador de distribuiÃ§Ã£o nÃ£o recompilado nesta continuaÃ§Ã£o.

### SHA e branches

O checkout local e origin/release/v1.0.0 estÃ£o em 52f35c04181b85379219ab55fc3192c63a5440a8, cinco commits apÃ³s o SHA d14154d2490dfdd3966c7fddc990a1d9127edd39 solicitado. O worktree estÃ¡ sujo; os resultados sÃ£o do worktree, nÃ£o certificaÃ§Ã£o exact-SHA. VERSION contÃ©m exatamente 1.0.0-beta.7.

A consulta ao vivo via git ls-remote confirmou as mesmas refs Beta 7 da auditoria anterior; nenhum ramo novo foi criado desde entÃ£o. fix/beta7-windows-KIVER-20261007 (d14154d2490dfdd3966c7fddc990a1d9127edd39) Ã© ancestral de origin/release/v1.0.0 (52f35c04181b85379219ab55fc3192c63a5440a8), com cinco commits posteriores na release e zero pendentes nessa branch. Embora trÃªs branches de terminal/testes tenham commits exclusivos no grafo, a Ã¡rvore atual da release jÃ¡ contÃ©m as correÃ§Ãµes e regressÃµes relevantes: coalescÃªncia ConPTY/DECCKM, filtro de rajadas de navegaÃ§Ã£o, trace redigido, teste raw-attach CSI/SS3 e testes de privacidade Go/Web. A branch fix/beta7-windows-autopilot-20261001 Ã© somente documental e sua evidÃªncia antiga foi superada pelo relatÃ³rio KIVER. NÃ£o hÃ¡ conteÃºdo Beta 7 pendente que justifique merge ou cherry-pick. A tag remota v1.0.0-beta.7 continua ausente.

**Veredito oficial: NO-GO para tag e distribuiÃ§Ã£o.** A Ã¡rvore nÃ£o estÃ¡ congelada no SHA solicitado, signing/trust root nÃ£o estÃ£o disponÃ­veis e o estado do CI privado nÃ£o foi revalidado. Nenhuma alteraÃ§Ã£o foi commitada, enviada, mesclada ou publicada. Nenhum gate Linux foi executado.
## Aditamento â€” auditoria final de branches e reteste terminal Windows â€” 2026-10-07

A consulta ao vivo de refs confirmou origin/release/v1.0.0 em 52f35c04181b85379219ab55fc3192c63a5440a8 e o mesmo conjunto de branches Beta 7 da auditoria anterior; nenhum ramo Beta 7 novo foi criado. fix/beta7-windows-KIVER-20261007 em d14154d2490dfdd3966c7fddc990a1d9127edd39 Ã© ancestral da release, cinco commits atrÃ¡s, sem commits exclusivos. O conteÃºdo terminal/teste relevante das branches protocol-p0, trace-diagnostics e final-boundary jÃ¡ estÃ¡ na Ã¡rvore da release; a branch windows-autopilot contÃ©m apenas evidÃªncia documental anterior e superada pelo relatÃ³rio KIVER. Nenhuma integraÃ§Ã£o de branch Ã© necessÃ¡ria.

- PASS â€” Windows AMD64: go test -count=1 ./internal/control/host ./internal/runtime.
- PASS â€” Windows: bun run --cwd web vitest run src/nexus/terminalInput.test.ts (28 testes).
- ConfirmaÃ§Ã£o de conteÃºdo: os arquivos Go de coalescÃªncia/trace e os testes de raw-attach CSI/SS3 e privacidade estÃ£o presentes na release; os arquivos de runtime terminal-output coincidem por blob entre release e branch protocol-p0.
- A Ã¡rvore permanece dirty e baseada em 52f35c04; os retestes direcionados nÃ£o mudam o veredito oficial NO-GO para tag/distribuiÃ§Ã£o.

## Aditamento â€” build NSIS e smoke isolado do payload real â€” 2026-10-07

O compilador NSIS 3.13 portÃ¡til compilou a cÃ³pia exata de packaging/windows/nexus.nsi da Ã¡rvore atual (SHA-256 do script d581f0f86513b5d64568ce0bc1f54c65e41d4ab68b9b9a527db36562680dc789). O instalador de validaÃ§Ã£o nexus-setup-Windows_x86_64.exe tem 23.917.346 bytes, SHA-256 7f310bf94d318d339ccd332d4c2de6b924404f87574e3adfccdb914fa55f5cf4 e Authenticode NotSigned. A compilaÃ§Ã£o emitiu apenas o aviso nÃ£o fatal de ai.exe ausente, pois essa entrada Ã© File /nonfatal no script.

Uma variante source-equivalent redirecionou as chaves HKCU para namespace GUID de teste e a shortcut para dentro do diretÃ³rio de instalaÃ§Ã£o. Com os trÃªs binÃ¡rios beta.7 reais, setup e uninstall retornaram 0; a CLI informou 1.0.0-beta.7; doctor --json foi vÃ¡lido; REG_SZ do Path de teste foi restaurado exatamente; Path/Desktop reais permaneceram inalterados; diretÃ³rio temporÃ¡rio e namespace de teste foram limpos. O instalador exato compilado nÃ£o foi executado no perfil real do usuÃ¡rio.

EvidÃªncias locais em outputs/windows-installer-validation-52f35c04-dirty-20261007/ e outputs/windows-installer-smoke-isolated-52f35c04-dirty-20261007/. O worktree continua dirty e os binÃ¡rios estÃ£o sem assinatura/trust root. O veredito oficial segue NO-GO para tag e distribuiÃ§Ã£o; CI nativo exact-SHA segue sem revalidaÃ§Ã£o.
