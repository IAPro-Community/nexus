> **HISTORICAL — NOT CURRENT RELEASE EVIDENCE**
>
> Canonical current status: `DEV/validation/current/RELEASE_STATUS.md`.

# IAPro Nexus — Final Platform / Release / Desktop Report

Date: 2026-09-06  
Branch: `feat/nexus-maximum-delivery`  
Starting SHA: `1899ca6334576d859056d48a394e51d03758f313`  
Candidate SHA: same working-tree `HEAD` (no new commit created)  
Remote CI run: `34012236345`

## Verdict

**NO-GO**

The local Linux evidence is strong, but the last remote committed tree at this
SHA failed native Windows and macOS CI. The current worktree is dirty and
contains additional uncommitted fixes, so that remote result is not evidence
against the full local candidate and is not a pass for it either. Frontend and
downstream browser/desktop/snapshot jobs were skipped in that run. Repository
admin permission is required to read the failed job logs; the GitHub API
exposed job/step conclusions but returned HTTP 403 for log download.

## Changes verified in this pass

- Corrigida a seleção do TUI Usage: `Enter` confirma o filtro e seleciona a
  linha destacada no mesmo evento; o cursor também é restaurado após um filtro
  temporariamente vazio. O fluxo tem teste determinístico e passou 20x nesta
  revalidação final (`go test ./internal/tui -run Usage -count=20`).
- Regressão final local passou: `go test ./... -count=1`, `go test -race ./... -count=1`,
  `bun run verify` e `git diff --check`. Os gates direcionados de desktop,
  notificações, update e release também passaram sob teste normal e race.
- `nexus doctor` deixou de marcar capacidades nativas como `PASS` por
  intenção: o shell Desktop agora exige smoke nativo, WebKitGTK usa probe
  read-only, e WebView2/ConPTY permanecem `WARN` até evidência no Windows.
  Testes do doctor passaram 20x e sob race 5x.
- Corrigida a sincronização cross-process do Registry: o cache agora compara
  `ModTime`, tamanho e fingerprint SHA-256 do conteúdo, evitando perder
  atualizações quando o sistema de arquivos mantém timestamp e tamanho. O
  teste concorrente passou 50x, o pacote passou race 10x e a suíte Go completa
  voltou a passar.
- O endpoint Web `/api/v1/system/updates` agora consulta o mesmo `Update
  Service` assinado usado pelo CLI/Desktop e expõe erro de confiança de forma
  honesta; a ação POST continua explicitamente Maestro-only. O contrato Web
  passou 20x e sob vet.
- A tela de Settings deixou de sugerir que o botão de manutenção do Maestro
  aplicaria uma atualização do Nexus; status, instrução e erro do Nexus agora
  são exibidos separadamente e o botão continua explicitamente Maestro-only.
  `bun run verify` passou com o contrato atualizado.
- O texto legado do merged help também foi corrigido nos três idiomas para não
  anunciar uma atualização conjunta; teste de localização passou 20x.
- A mutação Web do Maestro foi movida para `/api/v1/maestro/update` e agora
  exige `product=maestro`, `target_version=latest` e `confirmed=true`; não há
  mais POST genérico de `system/update`. Casos inválidos retornam 400 e o
  contrato válido passou 20x.
- O Axe encontrou `aria-allowed-role` no `<footer>` do status bar dentro da
  aplicação; o elemento foi convertido para um container `role=status` sem
  landmark `contentinfo` inválido. Browser E2E agora reporta **0 minor
  violations** e passou todos os breakpoints/Settings.
- Revalidação agregada posterior passou: `PATH=/tmp/nexus-tools:$PATH make
  quality-full` (incluindo race, security e frontend), com `No vulnerabilities
  found`; `bun run test:e2e` também passou após a correção de Axe.
- O job Frontend agora executa `bun run verify` após o build, validando igualdade
  byte a byte entre `web/dist` e `internal/control/web/embedded`, em vez de
  apenas verificar a existência dos arquivos.
- README e instaladores foram alinhados ao fluxo de versão fixada com SHA-256;
  o instalador PowerShell não exibe mais o branding público legado `AI CLI`.
- O instalador PowerShell passou a usar `Programs\\IAPro Nexus` e preserva uma
  instalação legada em `Programs\\ai-cli` sem removê-la automaticamente; há
  teste estático para essa política de migração segura.
- Os jobs nativos Windows/macOS agora preservam logs completos dos testes em
  artifacts com o SHA da execução, mantendo o código de saída original; isso
  melhora a reprodução dos failures sem mascarar CI vermelho.
- O job Windows ganhou um gate explícito de parsing PowerShell para
  `install.ps1`, separado dos testes Go e do smoke runtime.
- A fixture de `TestSessionHost_ListenerFailureTerminatesChild` foi separada por
  plataforma: Unix ocupa um socket com diretório e Windows usa um endpoint com
  NUL rejeitado pelo `go-winio`, removendo a fixture Unix inválida para Named
  Pipes. O teste Linux passou 20x e o pacote Windows compilou.
- A fixture interativa ConPTY foi corrigida para usar delayed expansion do
  `cmd.exe` (`!X!`) e entrada `CRLF`; `%X%` era expandido antes do `set /p` e não
  testava corretamente a entrega de stdin. O pacote Windows compilou.
- `TestFSMkdir` passou a serializar o path com `json.Marshal`; a concatenação
  anterior gerava JSON inválido para backslashes de caminhos Windows e mascarava
  o problema como `path is required`. O teste focado passou 20x e o pacote Web
  compilou para Windows amd64.
- Testes de readiness de `SessionHost` e protocolo deixaram de usar sleeps para
  aguardar endpoint: agora usam `WaitForEndpoint` com timeout ou sincronização
  de conexão real. Host/protocolo passaram 20x e sob race.
- Testes HTTP de bootstrap/session/restart também deixaram de depender de
  sleeps fixos; o listener já é criado por `NewServer` e as primeiras requests
  são a sincronização efetiva. A seleção focada passou 3x e sob race.
- Os E2Es Web, API Nexus e túnel também tiveram sleeps de startup removidos; a
  primeira operação real contra o listener agora fornece a sincronização.
  Pacote Web completo passou com e sem race.
- O teste PTY Unix deixou de dormir para tentar uma segunda leitura; agora usa
  leitor assíncrono com timeout explícito e aguarda o texto observado. Terminal
  passou 20x e sob race.
- A revalidação final dos pacotes terminal/host/protocol/web passou em testes
  normais, race e golangci-lint; `make quality PATH=/tmp/nexus-tools:$PATH`
  também passou.
- A suíte Go completa com race, security, GoReleaser snapshot v2.18.0 e
  actionlint v1.7.7 passou nesta revalidação local.
- Browser E2E encontrou e corrigiu um locator case-sensitive que não reconhecia
  a tradução `Configurações de aparência`; o locator agora usa role/name com
  regex case-insensitive. A execução completa passou com Playwright, Axe,
  deep-links, seis breakpoints e screenshot visual em três execuções
  consecutivas.
- Cross-build local dos entrypoints CLI passou para Linux amd64, Windows amd64,
  macOS arm64 e Desktop Windows amd64; o Desktop Linux exige WebKitGTK nativo
  e foi validado pelo fluxo Wails Linux, não por cross-build CGO.
- Os fixtures Windows de SessionHost/QA deixaram de usar o executável Unix
  `cat`; no Windows eles usam `cmd.exe /D /Q /C more`, evitando
  `CreateProcessW: File not found`. Linux/race passaram e os pacotes Host,
  Terminal e Web compilaram para Windows amd64 e Host/Terminal para macOS arm64.
- Wine não foi classificado como runtime nativo: sua implementação de ConPTY
  entrega a saída no console do Wine, mas não ao reader do pseudo-console; os
  testes ConPTY falham nesse harness específico e continuam pendentes em
  Windows real.
- O teste de aplicação do Update Service passou a isolar `HOME` em `t.TempDir`,
  evitando receipts de teste no checkout; Update passou 20x e sob race.
- A preparação do processo ConPTY passou a enviar o valor do handle `HPCON` ao
  atributo `PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE`, em vez do endereço da variável
  local que o contém. O pacote Terminal continua compilando para Windows amd64
  e arm64; a execução nativa ainda depende de runner Windows real.
- A fronteira Nexus/Maestro foi tornada explícita no CLI: `nexus update` é
  reservado ao Update Service do Nexus, enquanto `nexus maestro status|doctor|update`
  é a superfície opt-in para a integração Maestro. O help, completions e copy
  do Web foram alinhados, e o status sem Maestro reporta degradação honesta.
- O Update Service agora aceita o detached sidecar `.sig` publicado junto ao
  manifest, além do header de assinatura; ausência, erro ou assinatura inválida
  continuam falhando fechados. O contrato foi coberto por teste normal e race.
- O parser de deep links agora rejeita fragmentos, usuários, segmentos extras,
  IDs com slash/espaço/controle e entradas acima do limite; o autostart Linux
  também escapa o executável conforme o formato `.desktop`. Ambos têm testes
  de regressão no pacote Desktop.
- O `DesktopBridge` Web deixou de inferir capabilities pela mera existência de
  métodos Wails; após bootstrap ele usa a matriz reportada pelo backend Go e
  permanece conservador enquanto essa evidência não chega. O teste cobre um
  binding com picker exposto, mas capability falsa.
- Notificações nativas deixaram de interpolar payload em AppleScript e
  PowerShell: os scripts agora são constantes e recebem título/corpo como
  argumentos. O pacote Notify passou em teste normal, race e cross-compile
  Windows.
- A allowlist estrutural de URLs externas agora é compartilhada pelos bridges
  Web e Desktop, incluindo seus fallbacks `window.open` e `BrowserOpenURL`;
  somente `http`/`https` com hostname chegam ao navegador do sistema.
- Tokens de bootstrap que estavam persistidos em `DEV/HANDOFF.md` foram
  redigidos no working tree, junto com caminhos de instalação locais. Não há
  processo Nexus ativo; ocorrências históricas permanecem no histórico Git e
  exigem rotação/expiração externa se algum token tiver sido usado.
- O workflow CI agora possui um job nomeado `Security` com `govulncheck`, e o
  gate de release exige esse job no mesmo SHA antes da promoção/publicação.
  `actionlint` passou após a alteração; a execução remota ainda aguarda novo
  CI porque não houve push automático.

- Corrected Desktop capability truth: unimplemented tray, native menus, deep
  links, and autostart are no longer advertised; picker/notification claims
  depend on actual bindings or available fallbacks.
- Restricted Desktop external URL opening to structural `http`/`https` URLs.
- Replaced hardcoded dark theme reporting with platform probing and `unknown`
  when no reliable probe exists.
- Removed macOS AppleScript input interpolation from picker/notification
  fallbacks; untrusted labels are passed as process arguments.
- Made unsupported native picker/notification fallbacks return an explicit
  capability error instead of false success.
- Made the shared Update Service reject unsigned manifests and added a
  regression test.
- Made installers require an explicit version, verify the release archive
  against `checksums.txt`, and require explicit source-build intent/reference;
  removed mutable `latest`/`@latest` fallbacks.
- Kept checksum verification portable by supporting both `sha256sum` and
  macOS `shasum -a 256`.
- Fixed signed-manifest generation to emit absolute, versioned HTTPS artifact
  URLs and wired the release workflow to pass the release base URL.
- Fixed manifest signing to cover the exact published bytes, including the
  trailing newline, with a regression test verifying the published file.
- Hardened release publication: CI now runs on version tags, and the release
  job waits for and requires all named same-SHA Frontend, Browser, Linux,
  Windows, macOS, Desktop, and GoReleaser jobs before obtaining write scope.
- Corrected the Wails project location: the usable config is now adjacent to
  `cmd/nexus-desktop`, and a real local Wails v2.15.0 Linux build generated
  and packaged `cmd/nexus-desktop/build/bin/nexus-desktop`.
- Added native Desktop artifact upload/promotion: Linux, Windows, and macOS
  CI jobs package artifacts, and Signed Release downloads artifacts from the
  exact CI run ID selected by the same-SHA gate.
- Corrected public MIT license references and downgraded platform matrix claims
  to evidence-backed states.
- Preserved pre-existing local work in quota monitoring, notifications, handoff,
  and mission failover files; those changes were not discarded or reset.

## Verification matrix

| Gate | Result | Evidence |
| --- | --- | --- |
| Frontend format/typecheck/lint/stylelint/tests/build/embed/UI markers | PASS | `bun run verify` 10/10 at 2026-09-06 06:02:14Z; `make web-verify` also passes |
| Go formatting | PASS | `gofmt -l . \| grep -v .worktrees` empty |
| Go lint | PASS | `golangci-lint v2.12.2 run ./...`, `0 issues` |
| Go vet | PASS | `go vet ./...` |
| Go unit/integration tests | PASS | `go test ./...` |
| Go race | PASS | `go test -race ./...` |
| Desktop Linux build | PASS | `make build-desktop` |
| Wails native Linux packaging | PASS | `make build-desktop-wails`; Wails v2.15.0 generated bindings, compiled, and packaged the binary |
| Linux CI | PASS | run `34012236345`, Linux job passed |
| Windows native core | FAIL | same run, full Windows test step failed |
| macOS native core | FAIL | same run, race step failed |
| Browser E2E / Axe / visual | PASS locally / NOT REMOTE | `bun run test:e2e` passed three consecutive local runs; the historical remote run skipped it after frontend dependency failure |
| Desktop Windows/macOS | NOT RUN | skipped after dependencies failed |
| GoReleaser snapshot | PASS locally / NOT remote | local v2.18.0 snapshot passed; remote run `34012236345` skipped after dependencies failed |
| Security | PASS | `make security` now uses an installed `govulncheck` or pinned `go run ...@v1.7.0`; result: `No vulnerabilities found`. |
| Quality aggregate | PASS | `PATH=/tmp/nexus-tools:$PATH make quality`; all declared quality targets completed successfully |
| Quality-full aggregate | PASS | `PATH=/tmp/nexus-tools:$PATH make quality-full`; race, security, Go, and frontend gates completed successfully |
| Concurrency stress | PASS | `go test -count=20 ./internal/desktop ./internal/update ./internal/control/host ./internal/control/workspace` |
| Installer signed trust chain | PARTIAL | installers now use pinned version + SHA256 checksums and explicit source intent; detached Ed25519 manifest verification/public key distribution remains unresolved |
| Manifest artifact URLs | PASS | signer now emits absolute versioned HTTPS URLs; `go test ./scripts ./internal/update` passed |
| Manifest byte binding | PASS | `TestWriteSignedManifestSignsPublishedBytes` verifies Ed25519 over the exact file contents; Update Service also verifies a published detached `.sig` sidecar |
| Same-SHA all-platform CI | FAIL | candidate run is not green |
| Release same-SHA workflow gate | PASS (static) | YAML/actionlint validation passes; gate now waits for the newest exact-SHA CI run to complete, rejects non-success conclusions, and enumerates all mandatory jobs; not executed here |
| Native artifact promotion | PASS (static) | `actionlint` v1.7.7 and YAML parse pass; release downloads three named artifacts from the gated CI run; runtime not executed |
| CI embedded bundle equality gate | PASS (static/local) | workflow now runs `bun run verify`; local report passed build and embed-sync |
| Native failure diagnostics | PASS (static) | Windows/macOS jobs upload runner logs with exact SHA and preserve failing exit codes; Actionlint/YAML pass |
| GoReleaser snapshot (local) | PASS | `go run github.com/goreleaser/goreleaser/v2@v2.18.0 release --snapshot --clean`; six CLI archives, checksums, DEB/RPM artifacts generated; artifact matrix verified |
| CLI cross-build matrix | PASS (cross-compiled only) | Linux amd64, Windows amd64, macOS arm64 CLI and Windows amd64 Desktop compiled locally; not native runtime evidence |

## Findings table

| Finding | Platform | Root cause / state | RED evidence | Fix / status |
| --- | --- | --- | --- | --- |
| Desktop capability overclaim | All Desktop | Static defaults claimed unsupported features | `DefaultCapabilities` and `DesktopBridge` returned all `true` | Corrected and unit-tested locally; native smoke still pending |
| Unsafe external URL bridge | All Desktop | Arbitrary schemes reached OS opener | `OpenExternal` had no scheme validation | Restricted to `http/https`; unit-tested |
| macOS fallback input injection | macOS | Picker/notification labels were interpolated into AppleScript | `fmt.Sprintf` constructed scripts from caller input | Values now travel through `osascript` arguments; Go lint and Desktop tests pass |
| Hardcoded system theme | All Desktop | `GetSystemTheme` always returned `dark` | source inspection | Added OS probes and `unknown` fallback |
| Unsigned update manifest accepted | CLI/Web/Desktop service | Missing detached signature fell through to JSON parsing | `fetchManifest` fallback path | Now fail-closed and regression-tested |
| Browser Settings E2E locator | Web | CSS locator was case-sensitive and missed the lower-case `aparência` translation | local E2E timed out waiting for Settings button | Replaced with semantic role/name regex; three complete runs PASS |
| Windows test command fixture | Windows | Host/QA tests launched Unix-only `cat` through Windows `CreateProcessW` | Wine reproduced `CreateProcessW failed: File not found` | Platform-specific `cmd.exe /D /Q /C more`; cross-compile and Linux/race PASS |
| ConPTY attribute handle | Windows | `UpdateProcThreadAttribute` received `&hPC` instead of the `HPCON` value required by the ConPTY attribute | Native CI reported empty ConPTY reader output; code inspection matched the incorrect ABI usage | Pass `hPC` directly with `sizeof(HPCON)`; Windows amd64/arm64 cross-compile PASS, native runtime pending |
| Maestro update boundary | CLI/Web | Public help and settings copy described a combined Nexus/Maestro update surface | `nexus update` used the shared Nexus service, while `/api/v1/system/update` separately mutated Maestro; labels implied one operation | Added explicit `nexus maestro status|doctor|update`, changed CLI completions/help and UI copy; targeted app tests PASS |
| Desktop deep-link/autostart input | Desktop Linux/All | Deep-link parser accepted ambiguous paths and Linux autostart interpolated executable paths | `nexus://project/id/extra` and quoted executable paths were accepted/generated without structural escaping | Strict parser validation and `.desktop` Exec quoting; Desktop normal/race tests PASS |
| Desktop capability evidence | Web/Desktop | Frontend treated bound Wails methods as proof of native availability | `SelectFile` existed even when Go fallback returned capability unavailable | Cache/use `GetCapabilities()` from Go after bootstrap and report conservative pre-bootstrap state; `bun run verify` PASS |
| Native notification injection | macOS/Windows | Notification payload was interpolated into AppleScript/PowerShell source | `fmt.Sprintf` built executable script text from title/body | Static scripts consume process arguments; Notify normal/race/Windows cross-compile PASS |
| Frontend external URL fallback | Web/Desktop | Browser fallbacks bypassed the Go bridge scheme validation | `window.open`/`BrowserOpenURL` received raw caller URLs | Shared structural `http/https` validator; frontend full verify PASS |
| Persisted bootstrap secrets | Documentation/history | Local handoff recorded ephemeral loopback bootstrap URLs with token values | Secret scan found two token-bearing URLs in `DEV/HANDOFF.md` | Current working tree redacted and machine paths removed; historical Git objects are preserved and require external rotation assessment |
| Release security omission | CI/release | Same-SHA release gate did not require a Security job | Required-job list omitted security despite final release criteria | Added CI `Security`/govulncheck job and made it mandatory in release gate; actionlint PASS, remote execution pending |
| Windows native test suite | Windows | Exact failing test unavailable without repository-admin logs | CI job failed at `Test` step | BLOCKED pending native reproduction/log access |
| macOS race suite | macOS | Exact failing test unavailable without repository-admin logs | CI job failed at `Test with Race Detector` step | BLOCKED pending native reproduction/log access |
| Installer trust chain | Linux/Windows | Shell installers lacked pinned artifact/source controls and signed-manifest integration | source audit of `install.sh`/`install.ps1` | Pinned version + SHA256 and explicit source controls fixed; signed-manifest integration remains partial |
| Doctor capability truth | All Desktop | Doctor reported native shell/runtime support as `PASS` without probing availability or native smoke | `BuildReport` hardcoded platform and shell checks | Added read-only platform probes and `SKIPPED`/`WARN` statuses; native smoke remains required |
| Registry cross-process cache | Linux/Windows/macOS | Cache invalidation could miss writes with equal timestamp/size | `TestRegistry_ConcurrentMultiProcessNoLostUpdates` intermittently observed 39/40 sessions | Track `ModTime`, size and content fingerprint; targeted test 50x and race 10x PASS |
| Web update service drift | Web | `/api/v1/system/updates` reported only npm/Maestro status and did not consult Nexus Update Service | API response had no signed Nexus registry result | Web now uses shared Update Service and keeps Maestro mutation explicit; focused contract tests PASS |
| Web update action ambiguity | Web | Settings could show a generic update available state while POST only updated Maestro | UI combined Nexus/Maestro status with one ambiguous action | Display Nexus instruction/error separately; Maestro action remains explicit; frontend verify PASS |
| Localized merged update help | CLI | Legacy localized description still said Nexus and Maestro updated together | `HumanizeHelp` retained the merged claim in pt-BR/es/en | Explicit wording override plus regression test for all three languages |
| Ambiguous Maestro mutation endpoint | Web | Generic `/api/v1/system/update` could trigger Maestro maintenance without product/confirmation fields | Invalid `{}` was accepted by the handler | Dedicated `/api/v1/maestro/update` with explicit contract; negative and positive tests PASS |
| Axe status-bar landmark | Web | `<footer role="status">` exposed conflicting implicit `contentinfo` role inside the app shell | Browser Axe reported `aria-allowed-role` on `footer` | Replaced with `div role="status"`; Browser E2E reports 0 minor violations |

## Explicit external blockers

1. Native Windows/macOS runners or complete CI failure logs are required to
   identify and fix the remaining platform failures without guessing.
2. A production installer trust root/public-key publication contract is not
   wired into `install.sh` and `install.ps1`; current installer verification is
   checksum-only and therefore not a complete signed supply chain.
3. Code signing/notarization credentials and repository-admin branch protection
   permissions were not available and were not simulated.

The public GitHub job metadata was revalidated after the local work: Frontend
failed in `Format Check`, Windows failed in `Test`, and macOS failed in `Test
with Race Detector`; all later dependent steps were skipped. The API exposed
only the generic exit-code annotations and no downloadable diagnostic artifacts
for run `34012236345`, so no more specific remote root cause is claimed.

No release, tag, push, repository transfer, or production publication was
performed.

## Final local hygiene revalidation — 2026-09-06

- `git diff --check`: PASS.
- Secret-pattern scan of the working tree found only deliberately synthetic
  credentials in redaction/E2E tests; no real credential was printed or added
  to the report.
- No generated `node_modules`, `dist`, or `.tempmediaStorage` path is being
  treated as an untracked source change. Local untracked files are intentional
  campaign documentation, platform fixtures, Wails configuration, and the
  shared external-URL validator.
- Targeted Go revalidation passed for TUI Usage, registry, Web/API, Nexus,
  localization, and Doctor packages.
- Final local HEAD remains `1899ca6334576d859056d48a394e51d03758f313`; the
  working tree is intentionally dirty and has not been reset, committed, or
  pushed by this campaign.
- Final state revalidation: branch and remote both still point to
  `1899ca6334576d859056d48a394e51d03758f313`; the worktree has 97 preserved
  modified/untracked paths. Targeted release/TUI tests 20x, frontend format,
  `git diff --check`, and all required validation artifacts passed presence
  checks. No new remote CI run exists.
- The release gate was hardened so a completed failed run cannot be selected
  while a newer CI run for the same SHA is still in progress; workflow
  validation passed with `actionlint`.
- The SAME-SHA required-job list now matches the actual matrix-expanded
  Windows/macOS job names (`(1.25.14)`); the previous static names would have
  rejected an otherwise green run.
- The matrix check was then generalized safely: functional job prefixes are
  matched with exactly one successful matrix-expanded job, preventing both
  patch-version drift and ambiguous duplicate successes.
- GitHub Action runtime warnings were addressed by updating `checkout` to v5
  and `setup-go` to v6, both Node 24 editions; static workflow validation
  passes. Native execution on the next CI run remains required.
- Post-workflow-change aggregate verification: `PATH=/tmp/nexus-tools:$PATH
  make quality-full` passed, including 58 frontend test files/289 tests,
  Go tests/race, lint, security (`No vulnerabilities found`), and build/embed
  checks.
- Public license metadata was corrected from stale Apache-2.0 claims to the
  vigente MIT `LICENSE` in English and Spanish README surfaces; a release test
  now prevents this mismatch from returning.
- Added public Community Preview files with truthful scope: Code of Conduct,
  Security, Support, Roadmap, Governance, Changelog, and pull-request
  template. No maintainer ownership or repository transfer was invented.
- Added `.github/CODEOWNERS` for the repository owner and a Nexus-specific
  redacted bug-report issue form; no broader maintainer roster was inferred.
- Updated `FINAL_LOCAL_VALIDATION_PROMPT.md` to require the public-readiness
  audit, dirty-worktree/remote-tree distinction, and explicit checksum-only
  installer limitation.
- Final public-surface scan found no remaining Apache-2.0, signed-supply-chain,
  production/enterprise-readiness, or falsely verified Windows/macOS claims
  outside the explicitly historical support matrix entries.
- Installation examples now explicitly warn that scripts fetched from `main`
  must be reviewed/pinned by commit and that current artifact verification is
  checksum-only, not a published Ed25519 supply chain.
- Public README requirements now distinguish Linux local runtime evidence from
  pending native Windows/macOS candidate evidence and link to the support
  matrix instead of implying equal platform support.
- The public platform matrix now labels Windows/macOS as `UNVERIFIED` with a
  historical failed CI reference, rather than claiming the current dirty
  candidate is definitively `BROKEN`.
- The updater column now distinguishes the implemented Ed25519 service from
  the still-pending public keyring publication, avoiding an implied signed
  production channel.
- README platform tables now separate native-tested architecture rows from
  cross-compiled-only rows (Linux/Windows/macOS amd64 and arm64), so an
  architecture is not inferred from another architecture's runner.
- Additional Wine diagnostics were run for Windows test binaries: ConPTY still
  produced an empty reader stream and timed out, while SessionHost fixtures
  could not resolve `cmd.exe` through Wine's Windows process environment. This
  is useful reproduction evidence only; Wine is not counted as native Windows
  verification and no Windows PASS is claimed.
