# Corrections made during homologation

All changes are local and uncommitted. The local contract prohibits automatic commits. Base source SHA remains `f62fdec0c571ac6a19a904164d3f580799da6bf3`; `FIXES.patch` SHA-256 is `5557c3bccf606baffcdc8e3db1e44c7cd423d0da570baf69f9acb8744d749b10` and contains F1–F5 only. Follow-up changes F6–F7 are recorded here but are not included in that patch file.

## F1 — Portuguese explicit review delegation

- Observed: `peça ao Codex para revisar isso` resolved to the generic `implementer` role, so a requested read-only review was not delegated to the registered Codex profile.
- Change: recognize `revisar`, `revise`, `revisão`, and `revisao` in the review-task classifier.
- Regression: the existing explicit-provider review test failed before the change and passed afterward. The complete Go suite also passed after the change under the Wayland environment.
- Files: `internal/nexus/task_requirements.go`, covered by `internal/nexus/registered_provider_routing_test.go`.

## F2 — HUD restore test session isolation

- Observed: a test for the `WindowShow` geometry-restore event expected arbitrary X/Y positioning, but the host's real Wayland capability correctly disables global positioning.
- Change: the event-hook unit test explicitly enables the position-capable path it intends to cover. This changes the test fixture only; it does not claim global positioning works on Wayland.
- Regression: the focused test passed with `XDG_SESSION_TYPE=wayland`, and the complete Go suite passed. The separate Wayland capability test remains part of the suite.
- File: `internal/desktop/hud_window_test.go`.

## F3 — Unsupported Fedora/RHEL RPM guidance

- Observed: `install.sh` printed an RPM installation command although `.goreleaser.yaml` produces DEB only and `docs/getting-started/packages.md` excludes RPM from v1.
- Change: remove the unsupported RPM command from installer output.
- Regression: `bash -n install.sh` passed; `rg -i rpm install.sh` returned no matches.
- File: `install.sh`.

## F4 — Unix-only protocol test lint

- Observed: two Windows-only skip branches were unreachable because the entire test file is already guarded by `//go:build !windows`; golangci-lint reported `SA4032`.
- Change: remove the unreachable runtime guards and their `runtime` import.
- Regression: `go test -tags=gtk3 -count=1 ./internal/control/protocol` passed after this change; golangci-lint then reported zero issues. The full Go and race suites had passed after F1/F2, before this test-only cleanup.
- File: `internal/control/protocol/endpoint_path_test.go`.

## F5 — HUD omitted by Linux Desktop installer

- Observed: the native Desktop looks for sibling `nexus-hud`, but the signed Linux installer downloaded and installed only `nexus-desktop`; HUD startup could silently be skipped in an installed layout.
- Change: require the Linux HUD archive to be present in the signed manifest, verify it through the existing signature and SHA-256 path, and install it beside `nexus-desktop`. Update package docs with the two expected archives and sibling layout.
- Regression: `bash -n install.sh`, the manifest artifact key regression, signed-manifest unit tests, and archive extraction/executable sibling-layout checks passed. A published stable manifest is absent, so the full live installer path remains blocked.
- Files: `install.sh`, `docs/getting-started/packages.md`.

No Companion, HUD product code, or release/distribution code was merged from another branch. PR #55 remains a release blocker.

## F6 — Comando de login padrão repetia o nome do provedor

- Observado no screenshot: `nexus login codex codex`. A gramática do CLI exigia provedor e perfil, ambos com o valor `codex` para o perfil padrão.
- Mudança: a UI omite o argumento redundante quando o perfil coincide com o provedor; `login <provider>` agora resolve para o perfil padrão nomeado com o ID do provedor. Perfis explícitos continuam usando os dois argumentos.
- Regressão: dois testes Vitest cobrem a geração padrão e explícita; testes Go cobrem nome padrão, nome explícito, aridades inválidas e uma invocação com CLI Codex fake.
- Evidência: `logs/host-session-import-quality-final.log`; build Wails e startup nativo em `logs/host-session-import-desktop-build-final.log` e `logs/host-session-import-native-smoke-final.log`.
- Arquivos: `web/src/features/usage/AccountLoginModal.tsx`, `web/src/features/usage/providerLoginCommand.ts`, `internal/app/app.go` e testes associados.

## F7 — Limpeza exigida pelo lint completo após importação de sessões

- A primeira execução do `make quality` expôs cinco findings: três em wrappers/testes recém-introduzidos no fluxo de importação e dois no tratamento de leitura OpenCode.
- Mudança: removidos wrappers de validação sem uso, conversões redundantes em fixtures e atribuição de erro não utilizada; comportamento de runtime preservado.
- Regressão: `make quality`, `make vet` e `go test -race -tags=gtk3 ./internal/app ./internal/profile` passaram na execução final. A tentativa inicial está preservada como `logs/host-session-import-quality-first-attempt.log`.

