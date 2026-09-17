# Current release status

STATUS: CLOSURE_PATCH_IMPLEMENTED_PENDING_HOSTED_VALIDATION
VERSION: 1.0.0-beta.1
HEAD_AT_WRITE: 19e233c5c4224b46e565b754b4ec92d28a757dd9
GENERATED_AT: 2026-09-17T00:00:00Z

O repositório permanece privado e a distribuição beta é closed-source. O
HEAD acima é a baseline observada antes do commit único de fechamento; se a
branch avançar, a validação hosted deve ser refeita no novo SHA.

## Remaining closure gates

1. Autorizar um único commit de fechamento e fazer push da branch.
2. Confirmar os 14 jobs definidos em `.github/workflows/ci.yml` verdes no mesmo SHA.
3. Reexecutar o Windows no mesmo SHA e obter duas execuções consecutivas verdes.
4. Provisionar o GitHub Environment `beta-signing` com
   `NEXUS_UPDATE_PRIVATE_KEY`, `NEXUS_UPDATE_KEY_ID` e
   `NEXUS_UPDATE_PUBLIC_KEY` como variável.
5. Após merge commit da PR #2 e CI verde na `main`, remover a tag remota
   `v1.0.0-beta.1` que ainda aponta para `cf2eb41` e recriá-la no commit entregue.
6. Executar o Signed Release e verificar checksums, DEB/RPM/NSIS, binários
   desktop, `update-manifest.json`, assinatura e `channel-beta`.
7. Como o plano privado não permite branch protection, mergear somente por
   maintainer autorizado, com checklist manual e sem push direto planejado na
   `main`; o risco residual permanece explicitamente registrado.

Não publicar como `1.0.0` estável.

## Local closure slice

- golangci-lint: misspell PT isolado em `composer_brief.go`; SA1019 no CLI legado
- Composer attach: symlink, LimitReader, UTF-8, detach/persist não rebaixam `FINALIZED`
- Telemetria de teste isolada em `AI_CLI_STATE_DIR`
- README / quick-start / installation apontam para `IAPro-Community/nexus` e `v1.0.0-beta.1`
- CHANGELOG descreve Community Preview, não Unreleased
