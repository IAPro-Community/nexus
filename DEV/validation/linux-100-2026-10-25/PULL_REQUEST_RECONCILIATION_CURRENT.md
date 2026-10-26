## Snapshot GitHub final — 2026-10-26 08:57 UTC

HEAD e remoto `release/v1.0.0` continuam `7a04f1b4cf5950b97b4ea1e156dda31ac513823b`; fingerprint do build instalado: `047e6f968d64ee7f51ad389a21d43fe305e9ce9fb98b00c750be526b06c2313c`. Reconsulta somente leitura: [PRs](logs/current-gh-prs-final-recheck-2026-10-26.json) e [releases](logs/current-gh-releases-final-recheck-2026-10-26.txt).

| PR | Estado remoto às 08:57 UTC | Tratamento nesta homologação |
| --- | --- | --- |
| #55 | CLOSED, Draft, DIRTY | Companion 2D continua fora do escopo por instrução do usuário; preservar somente o NEX único. |
| #66 | MERGED em `release/v1.0.0` | Arte oficial e sprite do NEX foram integrados; correspondem ao único mascote atual. |
| #67 | MERGED em `release/v1.0.0` | Contrato visual, manifesto e gates de paridade integrados. |
| #57 | OPEN, Draft, UNSTABLE | Função base de worktrees somente leitura existe na release; o diálogo nativo ainda precisa de validação. Sem merge da branch antiga. |
| #58 | OPEN, Draft, DIRTY | Split/contexto já existem na release; interação/ANSI/reconexão PTY continuam parcialmente bloqueados. Sem merge em bloco. |
| #59 | OPEN, DIRTY | Correção equivalente está no overlay atual e foi validada no Linux, mas ainda não tem commit imutável na release. Smoke Windows pendente. |
| #61 | OPEN, Draft, DIRTY | Portal documental em conflito; não bloqueia execução Linux. |
| #64 | OPEN, base `main`, BLOCKED | A correção isolada de aliases hosts foi validada localmente; a PR não foi integrada nem redirecionada para release. |

A API ainda lista cinco releases, todas `Pre-release` e `Unsigned Preview`; `VERSION` é `1.0.0-beta.10`. Não existe release estável `v1.0.0`. Nenhum merge, commit, push, tag ou publicação foi executado nesta homologação. #55 permanece fechado; #66/#67 são as PRs de visual NEX já incorporadas.

# Reconcile de PRs abertas para a homologação Linux

## Snapshot GitHub final — 2026-10-26 08:19 UTC

HEAD local/remoto continua `7a04f1b4cf5950b97b4ea1e156dda31ac513823b`; a árvore de fontes utilizada no build instalado tem fingerprint `047e6f968d64ee7f51ad389a21d43fe305e9ce9fb98b00c750be526b06c2313c`. Consulta somente leitura por `gh pr list`, `gh pr view` e `gh release list`; saídas integrais em `logs/current-gh-prs-and-releases-final-2026-10-26.txt` e `logs/current-gh-pr-details-final-2026-10-26.json`.

| PR | Estado agora | Destino / decisão |
| --- | --- | --- |
| #55 | CLOSED, Draft, DIRTY | Companion 2D foi explicitamente retirado do produto; preservar um único NEX. |
| #57 | OPEN, Draft, UNSTABLE (`3d6d57a21d22ca19c723327a38d599846595223d`) | Núcleo de worktrees somente leitura já está na release; o diálogo nativo ainda não está homologado. Não integrar branch antiga em bloco. |
| #58 | OPEN, Draft, DIRTY (`9b523a42ccc085e9ee618081e9d3ecc2d400436b`) | Split/contexto já têm implementações e testes locais; validação de PTY real segue parcial. Não mesclar conflitos em bloco. |
| #59 | OPEN, DIRTY (`36e7cafd40609eb8424ac770e4d8f5f0a07fdb58`) | Correções Linux equivalentes estão no overlay, com testes/race/vet e HUD Wayland validados; ainda sem commit imutável na branch. |
| #61 | OPEN, Draft, DIRTY (`0b64885b8f58d59c0f19552fcb51b0edaf6d1cfb`) | Portal documental separado e conflito aberto; não bloqueia a homologação do executável. |
| #64 | OPEN, base `main`, BLOCKED (`0caf64085b4d5fd8b39358e69af2a4e8651d6453`) | Correção isolada de aliases validada localmente; sem portar dependências/migrações não relacionadas. |

Há cinco releases, todas previews sem assinatura; a estável `v1.0.0` não existe e `VERSION` continua `1.0.0-beta.10`. Nenhum merge, commit, push, tag ou publicação foi feito. `REL-001` continua P1/FAIL. O pedido de um só mascote prevalece: PR #55 não será reaberta nem incorporada.

Snapshot anterior consultado em 2026-10-26. Branch alvo `release/v1.0.0`; HEAD local/remoto `7a04f1b4cf5950b97b4ea1e156dda31ac513823b`; overlay então corrente `0f9f1c0912788cf913aff1918111cf3e2b10521e22f70c9079edc6f042a08f28` (1.428 arquivos). Estado remoto daquele snapshot em `logs/current-pr-status-final-2026-10-26.json`; manifesto daquele código em `logs/source-provenance-pr64-hosts-alias-fix-2026-10-26.txt`.

| PR | Estado remoto consultado | Escopo e decisão para a release |
| --- | --- | --- |
| #67 | MERGED em `release/v1.0.0` | Contrato visual NEX, manifesto e gates de paridade. Documentos estão na base atual; a identidade do mascote continua um único NEX. |
| #66 | MERGED em `release/v1.0.0` | Sprite NEX frontal e assets oficiais. É o renderer visual usado pelo HUD atual. |
| #62 | MERGED em `release/v1.0.0` | Evidência tipada de falha de runtime. Já incluída na linha atual; não reabrir nem substituir por branch antiga. |
| #55 | CLOSED, Draft, CONFLICTING | Companion 2D está fora do produto por decisão explícita do usuário. Um único mascote NEX é o contrato; não reabrir nem integrar. |
| #57 | OPEN, Draft, UNSTABLE; head `3d6d57a21d22ca19c723327a38d599846595223d` | A funcionalidade principal de worktrees somente leitura já está no HEAD da release (`handlers_git_worktrees`, parser, modal e testes). Comparação dos arquivos do PR com o HEAD encontrou a versão atual equivalente e tipos/traduções mais recentes na release. Testes Go e 3 testes da UI passaram; interação nativa ainda não foi homologada. Não há delta funcional a portar; mesclar a branch antiga pode regredir a UI atual. |
| #58 | OPEN, Draft, DIRTY; head `9b523a42ccc085e9ee618081e9d3ecc2d400436b` | Os recursos centrais também estão no HEAD da release: modo `SPLIT` persistente e seu renderer, Command Palette com escopos e recomendação de conta Direct contextual. O head do PR diverge de melhorias posteriores já na release, incluindo rotas de Mission Attention, estilos de abas PTY e endurecimento do terminal. Os 70 testes Direct/contexto e E2E web de sete geometrias passaram; PTY nativo, persistência/reconexão e entrada ANSI ainda têm gaps. Não integrar a branch antiga em bloco. |
| #59 | OPEN, não Draft, CONFLICTING; head `36e7cafd40609eb8424ac770e4d8f5f0a07fdb58` | O restore em `WindowShow` e `CommandNoConsole` estão no overlay local sujo, mas ainda não no HEAD imutável da release. Testes Linux de `internal/desktop`, `internal/executil` e `internal/hudlink` passaram sob race e `go vet` passou; smoke Windows nativo não foi executado. Checks remotos foram SKIPPED/CANCELLED e a PR conflita. A correção local está validada para Linux, mas ainda não integra um commit da release. |
| #61 | OPEN, Draft, CONFLICTING; head `0b64885b8f58d59c0f19552fcb51b0edaf6d1cfb` | Portal Docusaurus. Build remoto falhou; publish passou em run separado e não compensa a falha. Escopo documental e conflito de merge; manter separado. |
| #64 | OPEN, não Draft, BLOCKED, base `main`; head `0caf64085b4d5fd8b39358e69af2a4e8651d6453` | Não direcionada à release e sem check runs. O defeito localizado de parsing foi corrigido no overlay com preservação de aliases/comentários e regressões sob race; dependências, lockfile, migrações e CSS da PR não foram transportados. A PR não foi mesclada. |

Nesta homologação não executei merge, commit, push, alteração de tag ou publicação. O estado consultado também mostra #67/#66/#62 já MERGED upstream em `release/v1.0.0`; #57 Draft/UNSTABLE, #58 Draft/DIRTY, #59 não Draft/DIRTY, #61 Draft/DIRTY e #64 base `main`/BLOCKED/sem checks; #55 CLOSED por estar fora do escopo do mascote. A comparação de árvore confirma que os recursos centrais de #57/#58 já estão no HEAD; o overlay local acrescenta o fix #59 ainda sem commit. A worktree compartilhada mantém alterações simultâneas. O overlay `0f9f1c09…` é fingerprint de fonte, não commit de release imutável.

Os check rollups atuais foram consultados somente como contexto: #57/#58/#59 mostram os gates de build/teste como SKIPPED e sanitize como CANCELLED; #61 tem build FAILURE e publish SUCCESS em runs distintos; #64 não tem check runs. A homologação local continua independente dos workflows. Saída completa: `logs/current-gh-release-and-pr-recheck-2026-10-26.txt`.

## Rechecks executados nesta atualização

- PR #57: `go test ./internal/control/web -run '^(TestParseGitWorktreesPorcelain|TestListProjectGitWorktreesReadsLinkedCheckouts)$'` e `bun x vitest run src/features/projects/gitWorktrees.test.ts` passaram no código do HEAD da release; comparação direta confirmou que a funcionalidade já está presente. Logs `pr57-worktree-go-current-2026-10-26.txt` e `pr57-worktree-web-current-2026-10-26.txt`.
- PR #58: cinco suítes Direct/contexto passaram (70 testes); split E2E Chromium passou sete geometrias e toolbars compactas. A inspeção do HEAD confirmou `SPLIT`, o renderer dividido, a paleta e a recomendação contextual Direct. Logs `pr58-workspace-direct-current-2026-10-26.txt` e `pr58-terminal-split-current-2026-10-26.txt`.
- PR #64: parsing seguro de hosts foi aplicado como correção pontual; race x3, pacote `internal/control/web` e vet passaram. Logs `pr64-hosts-regression-race-final-2026-10-26.txt`, `pr64-hosts-web-package-final-2026-10-26.txt` e `pr64-hosts-vet-final-2026-10-26.txt`.
- `PLAYWRIGHT_EXECUTABLE_PATH=/usr/bin/google-chrome bun run test:e2e:terminal-split`: PASS, sete geometrias/orientações e toolbars compactas, mais abas com identificadores longos. Log `logs/current-pr58-terminal-split-e2e-2026-10-26.txt`. Chromium valida layout Web; não substitui teclado/PTY em Wails.
- PR #59: `TMPDIR=/home/desenvolvedor/.cache/nexus-p5-recheck/tmp GOCACHE=/home/desenvolvedor/.cache/nexus-p5-recheck/go-build go test -race -count=1 ./internal/executil ./internal/hudlink ./internal/desktop` e `go vet` nos mesmos pacotes passaram. Fingerprint de fontes `0f9f1c0912788cf913aff1918111cf3e2b10521e22f70c9079edc6f042a08f28`; veja `logs/p9-recheck-hud-race-vet-2026-10-26.txt`. A correção está apenas no overlay não congelado; o teste nativo Windows continua pendente.
- PR #57: `quality:full` passou no mesmo overlay, incluindo suites unitárias de worktrees; cobertura de interação no diálogo nativo continua sem evidência.

## Estado de distribuição consultado em 2026-10-26

A lista da API GitHub mostrou cinco releases, todas com tags `preview-*`; nenhuma release estável `v1.0.0` foi publicada. `VERSION` continua `1.0.0-beta.10`. O gate REL-001 permanece FAIL. A homologação não cria release ou tag. Evidência: `logs/current-gh-release-and-pr-recheck-2026-10-26.txt`.

## Reprodução

```bash
cd web
PLAYWRIGHT_EXECUTABLE_PATH=/usr/bin/google-chrome bun run test:e2e:terminal-split
cd ..
TMPDIR=/home/desenvolvedor/.cache/nexus-p5-recheck/tmp \
GOCACHE=/home/desenvolvedor/.cache/nexus-p5-recheck/go-build \
go test -race -count=1 ./internal/executil ./internal/hudlink ./internal/desktop
TMPDIR=/home/desenvolvedor/.cache/nexus-p5-recheck/tmp \
GOCACHE=/home/desenvolvedor/.cache/nexus-p5-recheck/go-build \
go vet ./internal/executil ./internal/hudlink ./internal/desktop
```
