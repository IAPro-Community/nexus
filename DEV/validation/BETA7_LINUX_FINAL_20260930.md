# Nexus 1.0 Beta.7 — Linux amd64 exact-SHA homologation (automated)

Data: 2026-09-30
Branch canônica: `release/v1.0.0`
Versão: `1.0.0-beta.7`
**SHA validado: `6bf1b9fef578822d2be9d542409a44a2e154ed96`**
Execução: `2026-09-30T21:35:34Z` → `2026-09-30T21:42:16Z` (~6m42s)
Veredito: **PASS — homologação automatizada Linux amd64**

> Este registro cobre **somente** o que `scripts/homologate-beta.sh` executa de
> forma automatizada. Checks manuais interativos, Windows, macOS, GitHub
> Actions e smokes de provider/voz **não** fazem parte deste PASS e continuam
> pendentes (ver “Pendências explícitas”).

## Comando de reprodução

```bash
export PATH="$TC/bin:$HOME/.bun/bin:/home/desenvolvedor/.nvm/versions/node/v22.17.0/bin:$PATH"
export GOTOOLCHAIN=local
export GOCACHE=/projetos/nexus/DEV/validation/local-beta/.gocache
export NEXUS_EXPECTED_SHA=6bf1b9fef578822d2be9d542409a44a2e154ed96
unset NEXUS_UPDATE_PUBLIC_KEY NEXUS_ALLOW_DIRTY NEXUS_SKIP_DESKTOP NEXUS_VERSION
./scripts/homologate-beta.sh
```

Exigências do ambiente: Go `1.25.14` (`GOTOOLCHAIN=local`), Bun `1.3.9`
(`~/.bun/bin` **antes** do `node22/bin`, que traz um Bun 1.4.0 via npm),
GTK4 + WebKitGTK 6.0 dev headers.

## Ambiente certificado

| Campo | Valor |
| --- | --- |
| OS / arch | Linux `x86_64` |
| Go | `go1.25.14` |
| Bun | `1.3.9` |
| `UPDATE_PUBLIC_KEY` | `ee2e268394ce6a0634a9ea2f414ee831dd536d12f7e2b80107f94566e58aa334` |
| Evidência | `DEV/validation/local-beta/1.0.0-beta.7/6bf1b9f…/linux-x86_64/` (gitignored) |

## Gates executados — todos PASS

| # | Gate | Resultado |
| --- | --- | --- |
| 1 | documentation (`docs-verify`) | PASS |
| 2 | frontend dependencies (bun install) | PASS |
| 3 | Playwright Chromium | PASS |
| 4 | frontend quality (format, lint, styles, typecheck, testes) | PASS |
| 5 | Go formatting (gofmt) | PASS |
| 6 | Go vet (sem tags; GTK4/WebKitGTK 6.0) | PASS |
| 7 | Go lint (golangci-lint v2.12.2 pinned) | PASS |
| 8 | Go security (govulncheck v1.7.0) | PASS |
| 9 | Version contract | PASS |
| 10 | provider-native Mission/terminal regression preflight | PASS |
| 11 | Go race suite (isolated profile) | PASS |
| 12 | control/runtime E2E | PASS |
| 13 | Agent Interop / OpenCode projection regressions | PASS |
| 14 | build CLI | PASS |
| 15 | Browser E2E (10 tarefas: e2e, composer, surface, capacity, hud, composer-spine, composer-vague-spine, composer-themes, a11y, visual) | PASS |
| 16 | OpenCode visibility regression | PASS |
| 17 | native Desktop + HUD build | PASS |
| 18 | GoReleaser snapshot (Linux amd64, standalone v2.18.0) | PASS |
| 19 | DEB inspect/extract/execute smoke | PASS |

Artefatos conferidos dentro do DEB/CLI: `version=1.0.0-beta.7`,
`go=go1.25.14`, `commit=6bf1b9fef578822d2be9d542409a44a2e154ed96`,
`arch=amd64`. `nexus doctor` PASS no evidence bundle
(`doctor.json`), além de `providers.json` e `usage.json`.

## Correções que destravaram o PASS

| Commit | Motivo |
| --- | --- |
| `b8483ef` | ignora `DEV/validation/local-beta/` (evidência de homologação) |
| `e2c3b16` | GoReleaser standalone v2.18.0 + `snapshot.version_template` com `NEXUS_VERSION` (Go 1.25.14 não roda o módulo goreleaser, que exige Go ≥ 1.27) |
| `0c60cce` / `2da6892` | gofmt + regra estática De Morgan em teste |
| `83370e9` | saída Prettier dos gates de frontend |
| `ede0b7e` | **a11y crítico** `select-name`: label do seletor de profundidade da entrevista do Composer fora do `<label htmlFor>` (reprovava os 4 temas) |
| `4ea6e9e` | espera do primeiro carregamento de `/settings` de 10s → 30s (flake de cold start sob carga; asserções inalteradas) |

Correções de ambiente (não versionadas): instalação de `libgtk-4-dev` +
`libwebkitgtk-6.0-dev` (o Wails v3 exige GTK4 sem tag `gtk3`; sem isso
`go vet`/lint/race reprovam) e `GOCACHE` dedicado.

## Incidência conhecida — cache do Go

Duas execuções abortaram em `Go lint` com
`could not import … (…/.cache/go-build/…: no such file or directory)`, ou seja,
`-d` do cache referenciados por `-a` inexistentes. O `trim.txt` do cache mostrou
execução de trim às 17:00 (Go trima no máximo 1×/24h e só entradas com mtime
> 5 dias), então o trim não explica a perda — algo externo removeu os arquivos
sob `~/.cache`.

Mitigação adotada na execução vencedora: `GOCACHE` dedicado em
`DEV/validation/local-beta/.gocache` (gitignored), pré-aquecido com
`go build ./...` + `go vet ./...`. Não é edit `GOFLAGS`; apenas isolamento de
cache. Se recorrer, repetir o pré-aquecimento antes do gate.

## GitHub Actions (FASE 10)

Estado real: **bloqueado pelo runner self-hosted offline**, não por créditos.

- Runner único: `nexus-ci-dev-web-kiver` (Linux/X64/`nexus-ci`) — `status: offline`.
- A job `Sanitize self-hosted workspace` fica `queued` indefinidamente
  (job `110112383061` da run `36781431374` segue `queued`/`runner_id: 0`).
- As runs anteriores aparecem `completed/cancelled` porque a `concurrency`
  cancela a run anterior a cada push; nenhuma delas chegou a executar jobs.
- Portanto o registro histórico “BLOCKED_EXTERNAL_GITHUB_ACTIONS_CREDITS”
  está **superado**: o obstáculo é o runner offline nessa máquina.

Enquanto o runner não subir, a certificação exact-SHA continua sendo a
evidência local destes gates + a homologação Windows nativa.

## Pendências explícitas (não cobertas por este PASS)

1. **FASE 6 — smoke Codex**: `nexus codex`, navegação por setas sem artefatos
   `[A`/`[B`/`[C`/`[D`, prompt trivial e complexo sem promoção a Mission,
   `--lead` com `NEXUS_DELEGATION_MODE` unset, e os casos
   `ASK`/`AUTO`. Requer sessão TTY real e credencial de provider.
2. **FASE 7 — AGY/OpenCode**: mesma matriz de prompts, mais os itens de
   Agents (OpenCode oferecido de forma independente de AGY), Usage (OpenCode
   instalado sem perfil aparece como provider), papéis Review/Investigate e
   Certify/ENFORCED.
3. **FASE 8 — Voice**: permissão de microfone, captura, transcrição, fallback,
   inserção no terminal sem Enter automático e um comando global
   não-destrutivo.
4. **HUD**: show/hide, autostart e projeção em sessão nativa.
5. **Windows amd64** e **macOS arm64**: certificação nativa no mesmo SHA.
6. **GitHub Actions**: subir o runner `nexus-ci-dev-web-kiver`.

A matriz completa e imutável destes checks está em
`DEV/validation/local-beta/1.0.0-beta.7/<sha>/linux-x86_64/MANUAL_CHECKS.md`
(gerada pelo próprio script no SHA validado).

## Segredo fora do repositório (divulgação obrigatória)

Durante a varredura, `scripts/ad-auth/` foi removido do repositório e movido
para **`/home/desenvolvedor/scratch/ad-auth-20260930`**. O material contém
credencial de bind LDAP em texto plano (`bindPassword`) e endpoint
`ldap://192.168.5.4:389`.

- Não versionar, não reintroduzir.
- **Ação do maestro**: rotacionar a credencial e tratar o diretório como
  material sensível local (remover quando não for mais necessário).
- Nenhum segredo de provider foi gravado na evidência.

## Handoff Windows

```text
WINDOWS MUST VALIDATE SHA = 6bf1b9fef578822d2be9d542409a44a2e154ed96
```

Observação: o HEAD de `release/v1.0.0` avançou para `a80e9f4c7a91bab43a9490f3d666d8e4080508c6`,
que é **somente documentação/evidência DEV** (`git diff 6bf1b9f..a80e9f4` toca
apenas `DEV/`). O código do SHA validado é idêntico ao do HEAD.
