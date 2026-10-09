# Smoke isolado do Path do instalador Windows

**Data:** 2026-10-07
**Base:** `release/v1.0.0` em `52f35c04181b85379219ab55fc3192c63a5440a8`
**Versão:** `1.0.0-beta.7`
**Sistema:** Windows AMD64 nativo
**Veredito:** 5/5 cenários PASS

## Escopo

O smoke verificou se o instalador e o desinstalador preservam a existência, o conteúdo e o tipo de `HKCU\Environment\Path`. A fixture usou NSIS portátil 3.13, payload executável fictício e uma subchave de registro isolada. Não alterou o `Path` real do usuário nem criou atalhos no Desktop real. A chave de teste foi removida ao final.

O patch local modifica `packaging/windows/nexus.nsi` e `internal/release/installer_test.go`. Ele captura metadados do Path antes da instalação e restaura `REG_SZ`, `REG_EXPAND_SZ` ou a ausência original durante a desinstalação.

## Resultados

| Estado antes da instalação | Estado após desinstalação | Resultado |
| --- | --- | --- |
| `REG_SZ`, conteúdo não vazio | Valor, conteúdo e tipo preservados | PASS |
| `REG_EXPAND_SZ`, conteúdo não vazio | Valor, conteúdo e tipo preservados | PASS |
| `REG_SZ` vazio | Valor vazio e tipo preservados | PASS |
| `REG_EXPAND_SZ` vazio | Valor vazio e tipo preservados | PASS |
| Valor ausente | Continua ausente | PASS |

Em todos os casos, instalador e desinstalador terminaram com código `0`.

## Verificações complementares

- `go test -count=1 ./internal/release`: PASS no checkout Windows real.
- `go test -count=1 ./...`, `go vet ./...` e `golangci-lint run ./...`: PASS no worktree isolado de `52f35c04` com o mesmo patch local.
- `git diff --check`: PASS.
- Fixture setup SHA-256: `F54349A78908851F4040431B79007A0CA2E2C2B0B8CD8DF7BAE3BE63367B4942`.

Esta fixture não substitui o build do instalador final com os binários reais, a assinatura Authenticode nem o workflow Windows exact-SHA. O patch está aplicado localmente e sem commit; o checkout não é um candidato congelado para tag ou distribuição.

## Smoke complementar com payload Windows beta.7 real — 2026-10-07

O build exato de distribuição foi compilado a partir do script atual com NSIS 3.13, sem assinatura. Em diretório e registry isolados, uma cópia source-equivalent instalou os binários reais CLI/Desktop/HUD e executou a desinstalação.

- PASS — setup e uninstall retornaram 0.
- PASS — CLI instalada informa 1.0.0-beta.7; doctor --json produziu JSON válido e exit 0.
- PASS — Path de teste REG_SZ foi restaurado exatamente; Path e Desktop reais permaneceram inalterados; TEMP e chave de teste foram limpos.
- LIMITE — esse smoke redireciona os alvos HKCU e shortcut para a fixture. Não é evidência de instalação do executável final exato no perfil do usuário.
- Executável exact-source unsigned: 23.917.346 bytes, SHA-256 7f310bf94d318d339ccd332d4c2de6b924404f87574e3adfccdb914fa55f5cf4. Evidência completa de build e smoke: outputs/windows-installer-validation-52f35c04-dirty-20261007/ e outputs/windows-installer-smoke-isolated-52f35c04-dirty-20261007/.
