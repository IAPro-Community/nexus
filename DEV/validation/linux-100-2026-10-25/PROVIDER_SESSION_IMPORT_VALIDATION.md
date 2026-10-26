# Importação real de sessão do usuário — 2026-10-26

- Executei `profile.ImportHostSessions` usando o diretório atual do Nexus, sem iniciar login nem chamar modelo. A primeira execução retornou: Codex/AGY/OpenCode/Cursor `ALREADY_AUTHENTICATED`, Gemini `IMPORTED`, Claude `NO_SESSION`.
- Gemini ficou com um único perfil Nexus autenticado e padrão. O estado reportado pelo próprio Nexus é `Session needs refresh`; portanto, a cópia local passou, mas uma chamada real ao provedor ainda não foi validada.
- A segunda execução retornou `ALREADY_AUTHENTICATED` para Gemini e manteve exatamente um perfil, comprovando idempotência. O arquivo isolado corresponde byte a byte à origem e ambos estão em modo `0600`. Saídas sanitizadas: `logs/host-session-import-real-profile-idempotency-2026-10-26.txt` e `logs/host-session-import-live-profile-state-2026-10-26.txt`.
- Claude não tinha `accessToken` nem `refreshToken` no arquivo local; `claude auth status` saiu com código 1. O importador retornou `NO_SESSION` e não criou perfil. Nenhum novo login foi iniciado. Só há importação possível quando o CLI do host mantém uma sessão válida e exportável.
- A execução não leu nem escreveu texto de credenciais nos logs, e não fez requisição a nenhum modelo. A conta Gemini pode precisar atualizar a sessão; quota/tier e execução real seguem pendentes.

## Rechecagem da sessão Cursor instalada no host — 2026-10-26

- Executei `timeout 15s cursor-agent status` sem gravar a saída bruta: classificação `AUTHENTICATED`, exit 0, 64 bytes. Email, tokens e conteúdo do banco local não foram lidos nem persistidos. Evidência sanitizada: `logs/cursor-existing-session-status-sanitized-2026-10-26.txt`.
- A documentação oficial atual do Cursor CLI orienta autenticação por `agent login` ou `CURSOR_API_KEY`; não documenta exportação da sessão local para outro perfil ([Cursor CLI help](https://prod.cursor.com/help/integrations/cli)).
- Conclusão: a sessão pessoal está ativa, mas o Nexus ainda não tem um caminho suportado para copiá-la. BLK-004 permanece `BLOCKED`; não vou copiar o banco SQLite de 17 GB nem dados de keychain sem contrato documentado. A correção anterior continua válida: um perfil Nexus Cursor que já esteja autenticado é reconhecido como `ALREADY_AUTHENTICATED`.

## Desktop Wails no overlay atual — Cursor autenticado em fixture isolada — 2026-10-26

- Build `6bbbafdab42309d2657ceed4ccf90f1d87c4381aee5d7b6d4528903a136dc61f` (`0df26de4a35641d1e5d02ded7757b257fa7f4f15747220416574c083b4127208`) executado sob D-Bus/Xvfb e `NEXUS_DATA_DIR` temporário. Um `cursor-agent` fixture retornou `Login successful`; o perfil foi criado sem login e sem qualquer sessão pessoal.
- A janela Wails chamou o Core real. Resultado visual: **0 importadas, 1 já autenticada, 5 sem sessão importável, 0 sem suporte, 0 falharam**. A captura mostra a conta Cursor fixture existente e o único mascote NEX: [Uso/Contas](p5-recheck-2026-10-26T0347Z/screenshots/desktop-existing-cursor-clean/usage-after-import.png), SHA-256 `73e4aa31cb1b3fbe76d3bbb9ad24b01c75cf12a869ba8fa09ade12d9e540e92a`.
- Evidência reproduzível: `tools/native_usage_existing_cursor_fixture_xvfb.sh`; log `p5-recheck-2026-10-26T0347Z/logs/native-usage-existing-cursor-xvfb-retry.txt`. Perfil e HOME fixture ficaram em `/run/user/1000/nexus-native-existing-cursor.wjmdbe/`; não contém dados de conta real.
- Isso confirma o caminho UI+Core fixture, não uma chamada externa nem exportação da sessão global Cursor. Interação do compositor Wayland do usuário permanece BLOCKED.

## Reteste — perfil já autenticado sem importador de sessão — 2026-10-26

- Corrigida a ordem de `ImportHostSessions`: primeiro inspeciona perfis habilitados que já existem; a presença de um perfil autenticado retorna `ALREADY_AUTHENTICATED` mesmo quando o adaptador não implementa `HostAuthImporter`.
- Regressão: `TestImportHostSessionRecognizesAuthenticatedProfileWithoutImporter`; `go test -count=1 ./internal/profile` e `go vet ./internal/nexus ./internal/profile` passaram no overlay `0df26de4a35641d1e5d02ded7757b257fa7f4f15747220416574c083b4127208`.
- Limite honesto: o Cursor CLI não oferece formato de exportação de sessão documentado no adaptador. Não copiei `state.vscdb` nem tokens internos. Se o usuário já tem perfil Cursor autenticado pelo Nexus, o importador agora o reconhece; se não existe, a cópia automática do login global continua não suportada.
- Log e fingerprint: `p5-recheck-2026-10-26T0347Z/logs/go-profile-suite-fixed.txt`, `p5-recheck-2026-10-26T0347Z/logs/go-vet-nexus-profile-fixed.txt`; manifesto `p5-recheck-2026-10-26T0347Z/logs/source-provenance-after-auth-maestro-fixes.txt`. A interação Wails no compositor do host continua BLOCKED separadamente.

# Validação da importação de sessões em Uso/Contas

## Reteste do Desktop Wails corrente — 2026-10-26

**PASS em Xvfb 1440×900:** executei `cmd/nexus-desktop/build/bin/nexus-desktop` do overlay `751cfc1a160c38d950ac5b9807e1febb9746d03981c861c722e5bd965e1ef012` (SHA-256 `a0190540c4aa14227ad4d4ffd5c0a5adb8438a21d42e0602bee50fae8129f6ff`). O Core embutido respondeu HTTP 200. Com HOME, dados Nexus e diretórios Codex/Claude temporários, adicionei um repositório Git vazio, abri Uso/Contas e cliquei em “Importar sessões existentes”. Resultado mostrado na tela: 0 importadas, 0 já autenticadas, 5 sem sessão, 1 sem suporte e 0 falhas. Nenhuma sessão do HOME real foi lida ou copiada.

- [Captura antes](screenshots/native-current-desktop-xvfb-1440-2026-10-26/usage-before-import.png), SHA-256 `f72f5f2c5234cd4283d60f95ec9c6cfa9c1ba1100a8826c14cad36699da97865`.
- [Captura após importação](screenshots/native-current-desktop-xvfb-1440-2026-10-26/usage-after-import.png), SHA-256 `d0a0c9ad3f303e7828997ed5c404a44b5fb71065ca830698772d3df65add4f4c`; checksums em `screenshots/native-current-desktop-xvfb-1440-2026-10-26/SHA256SUMS`.
- Log do procedimento: `logs/native-desktop-current-build-xvfb-1440-2026-10-26.log`; detalhes de classificação da tentativa 1600×1000 inconclusiva: `logs/native-desktop-current-build-xvfb-2026-10-26-notes.txt`.

Xvfb certifica o fluxo nativo Wails e o estado visual capturado, mas não valida transparência, click-through ou o compositor Wayland do host.

## Escopo

O usuário precisa importar sessões locais reconhecidas diretamente em Uso/Contas para não receber um novo fluxo de login quando já autenticou uma CLI. A importação existente no ProjectHub permanece disponível. O serviço registra apenas perfis isolados suportados; não modifica a sessão original.

## Alteração

- `UsageSurface` agora oferece **Importar sessões das CLIs** junto às ações de contas.
- A tela chama `POST /api/v1/profiles/import-host-sessions`, recarrega perfis e cotas, e resume `IMPORTED`, `ALREADY_AUTHENTICATED`, `NO_SESSION`, `LOGIN_REQUIRED`, `UNSUPPORTED`, `NOT_INSTALLED` e `FAILED`.
- A mensagem de resultado contém contagens; não exibe credenciais, nomes de perfil ou dados de conta.
- O resumo é compartilhado pelo ProjectHub e Uso/Contas em `web/src/lib/hostSessionImport.ts`; não há lógica duplicada entre features.
- Textos estão localizados em inglês, português brasileiro e espanhol.

## Evidências

| Caso | Procedimento | Status | Resultado |
|---|---|---|---|
| AUTH-010 | `NEXUS_E2E_ARTIFACTS_DIR=../DEV/validation/linux-100-2026-10-25/screenshots/host-session-import-2026-10-25 bun run test:e2e:first-run-import` | PASS | O E2E executa a UI Usage/Accounts em Chromium com API fixture: uma chamada ao importador, resumo renderizado e o perfil `Codex fixture` aparece após recarregar recursos. Isto valida o fluxo de interface com backend controlado, não a janela Wails nem cópia de uma sessão real. |
| WEB-IMPORT-001 | `bun run quality:full` | PASS | SCSS/allowlist, Prettier, ESLint, Stylelint, TypeScript, 109 arquivos/674 testes e build de produção passaram. |
| BUILD-IMPORT-001 | `make build build-desktop-wails build-hud-wails` | PASS | CLI, Desktop Wails e HUD Wails foram compilados como ELF x86-64 para `1.0.0-beta.10`; `ldd` não encontrou bibliotecas ausentes. |
| NATIVE-IMPORT-XVFB-001 | `xvfb-run -a -s '-screen 0 1440x900x24' bash DEV/validation/linux-100-2026-10-25/tools/native_usage_import_xvfb.sh` | PASS | Aplicativo Wails instalado real, Core embutido e HUD vinculado abriram; clique nativo no importador retornou 0 importadas, 0 já autenticadas, 5 sem sessão importável, 1 sem suporte e 0 falhas. HOME e dados vazios isolados garantiram que nenhuma sessão do host fosse lida ou copiada. Ver [captura antes](screenshots/native-usage-import-xvfb/usage-before-import.png), [depois](screenshots/native-usage-import-xvfb/usage-after-import.png) e [log](logs/current-usage-host-session-import-native-xvfb-2026-10-26.txt). |
| NATIVE-IMPORT-001 | Abrir Uso/Contas e clicar no importador na sessão Wayland/XWayland do host | BLOCKED | O Desktop e HUD vinculado iniciaram; o Core retornou HTTP 200. XWayland reporta root `0×0` e `grim` retorna `no wl_output`, então a interação no compositor do host permanece sem evidência. A interação sob Xvfb está coberta separadamente por `NATIVE-IMPORT-XVFB-001`. |

Captura do E2E de Uso/Contas: [usage-accounts-host-session-import.png](screenshots/host-session-import-2026-10-25/usage-accounts-host-session-import.png), SHA-256 `7ef08d6b3c61a330de982e5c089313797c1a86472020025079be1a9e20a78927`. A captura é Chromium com fixture; não certifica a superfície Wails.

Logs: [E2E](logs/current-first-run-and-usage-host-session-import-e2e-2026-10-25.txt), [quality gate](logs/current-usage-host-session-import-quality-release-final-2026-10-25.txt), [build](logs/current-usage-host-session-import-linux-build-2026-10-25.txt), [instalação](logs/current-usage-host-session-import-local-install-2026-10-25.txt), [startup nativo do usuário](logs/current-installed-desktop-startup-final-2026-10-26.txt), [interação Wails em Xvfb](logs/current-usage-host-session-import-native-xvfb-2026-10-26.txt), [bloqueio do compositor](logs/current-usage-host-session-import-native-2026-10-26.txt) e [checksum da captura Chromium](logs/current-usage-host-session-import-screenshots-2026-10-26.sha256).

## Artefatos e proveniência

- Branch `release/v1.0.0`; HEAD local e remoto `7a04f1b4cf5950b97b4ea1e156dda31ac513823b`.
- Overlay compilado: fingerprint `9a66687715414fc8b42d612fd2b4a531370c7f8c4fbfa86338a81ba9728ddcd1` (1.428 arquivos de produto; árvore modificada, sem commit imutável). Manifesto: [source-provenance-usage-host-session-import-2026-10-25.txt](logs/source-provenance-usage-host-session-import-2026-10-25.txt).
- CLI SHA-256 `47c517d28223ebcdde4bbb9288caf3c7413f9d8058ae20de7f2a672b1ebcbd81`.
- Desktop SHA-256 `362c090ea6972cc1548cf883d78a2d071e9de746e42eb8283a4e10867e15fd0e`.
- HUD SHA-256 `54925c242a76fc204ae646afdfc7762b88c458ef12dcee8f92765f66641435e1`.
- Os três executáveis foram instalados em `/home/desenvolvedor/.local/bin` após backup em `/home/desenvolvedor/.local/state/nexus-homologation/20261026T014019Z-usage-account-import`. Depois do teste isolado, o Desktop beta.10 foi reaberto no perfil padrão do usuário, com HUD vinculado; o Core respondeu HTTP 200. O processo atual está sob a unidade transitória `nexus-homologation-final-restored-20261026.service` (Desktop PID `22800`, HUD PID `22810`). Evidência: [restore do perfil](logs/current-installed-desktop-user-profile-restored-2026-10-26.txt).
- A instância de inspeção usou dados temporários e foi encerrada sem acionar a importação. O diretório `/tmp/nexus-linux-100-host-import-native-20261026T014019Z` foi preservado após a revisão automática de segurança bloquear o comando de limpeza; nenhuma sessão de provedor foi importada nessa instância.

## Limites e veredito

O fluxo de UI passou em E2E controlado e a ação foi clicada na janela Wails real sob Xvfb, com perfil vazio isolado; o Core respondeu HTTP 200 e a UI mostrou o resultado esperado sem criar contas. A importação real do serviço tem evidência separada em `AUTH-002`/`AUTH-007`. A interação na sessão Wayland/XWayland do host continua sem captura porque o root reporta `0×0` e `grim` responde `no wl_output`. Cursor continua sem importador porque o formato local não tem exportação documentada; esse provedor retorna `UNSUPPORTED`.

Este caso não muda a recomendação geral: **NO_GO_LINUX** enquanto a interação Wails deste fluxo e os demais gates obrigatórios permanecerem sem validação.
