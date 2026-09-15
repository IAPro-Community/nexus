# Current release status

STATUS: CLOSURE_READY_PENDING_FREEZE
VERSION: 1.0.0-beta.1
PRIOR_GO_SHA: 2b9c95675d1727008eec86c58cd6be9e8a44a788
PRIOR_GO_CI: https://github.com/IAPro-Community/nexus/actions/runs/34924168564
HEAD_AT_WRITE: 82f5447c7ef5584df71dbea492e7761ddbe02080
GENERATED_AT: 2026-09-15T10:20:00Z

O SHA `2b9c95693be7bf73a463c6395ec1ea9f131a4c9c` citado em ledgers anteriores
**não existe neste repositório**. O freeze com CI 14/14 é
`2b9c95675d1727008eec86c58cd6be9e8a44a788`.

HEAD `82f5447` adicionou Composer/Flow/i18n depois desse freeze e quebrou
Linux golangci-lint. Esta campanha de fechamento corrige lint, attach,
docs de install e changelog **neste worktree**. O GIT_SHA de freeze da
beta.1 é o commit que incorporar estas correções — anotar depois do
commit autorizado.

## Remaining human gates (not in-repo)

Sequência obrigatória — nenhuma destas etapas é automática:

1. Autorizar **commit** deste worktree (não incluir stash `wip: conversation spike…`).
2. Push da branch; esperar CI **14/14** no SHA do commit (Linux, Windows, macOS, frontend, security, browser, 3 desktops, snapshot, DEB, RPM, NSIS).
3. Provisionar GitHub Environment `beta-signing` com `NEXUS_UPDATE_PRIVATE_KEY`, `NEXUS_UPDATE_KEY_ID=nexus-signing-key-2026-v1`, variable `NEXUS_UPDATE_PUBLIC_KEY`.
4. Tornar o repositório **público** (ou documentar `GH_TOKEN` no install). Sem isso `curl \| bash` anônimo falha.
5. Apagar a tag local stale (`git tag -d v1.0.0-beta.1`) se ainda apontar para `be6c793`. Criar a tag **no SHA verde**, não no HEAD antigo.
6. Push da tag → workflow Signed Release. Verificar assets: checksums, DEB/RPM/NSIS, desktop, `update-manifest.json` + `.sig`, ponteiro `channel-beta`.
7. Só então autorizar merge da PR [#2](https://github.com/IAPro-Community/nexus/pull/2) em `main`.

Não publicar como `1.0.0` estável.

## Local closure slice

- golangci-lint: misspell PT isolado em `composer_brief.go`; SA1019 no CLI legado
- Composer attach: symlink, LimitReader, UTF-8, detach/persist não rebaixam `FINALIZED`
- Telemetria de teste isolada em `AI_CLI_STATE_DIR`
- README / quick-start / installation apontam para `IAPro-Community/nexus` e `v1.0.0-beta.1`
- CHANGELOG descreve Community Preview, não Unreleased
