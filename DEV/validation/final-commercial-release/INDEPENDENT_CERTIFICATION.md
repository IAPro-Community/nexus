# INDEPENDENT CERTIFICATION — Nexus v1.0.0-beta.1

Auditor independente. `FINAL_REPORT.md` lido SOMENTE após a investigação
abaixo. Tentativa ativa de reprovar o RC; dois bugs reais foram
encontrados, corrigidos, testados e commitados (regra de certificação);
a auditoria foi reiniciada no SHA final.

## SHA auditado

- branch: `feat/nexus-maximum-delivery` (13 commits à frente de
  `origin/feat/nexus-maximum-delivery` — push pendente)
- SHA: `50608624dba98a7790501f5ed47f188bbc9d4264`
- tree: limpo (`git status --short` vazio)
- versão: `1.0.0-beta.1` (`VERSION` + `web/package.json`)
- data: 2026-09-17 UTC

## P0 (bloqueadores técnicos)

Nenhum P0 remanescente. Dois issues classe-release foram encontrados e
CORRIGIDOS nesta auditoria (commits abaixo); o suite completo foi
re-executado no SHA final:

1. (era P1-gate) `go test ./...` VERMELHO no SHA do report:
   `TestPublicLicenseMetadataMatchesLicenseFile` exigia LICENSE MIT após
   o relicenciamento, e `README.en.md`/`README.es.md` ainda anunciavam MIT.
   O report afirmava suite verde — a tabela de testes estava desatualizada
   em relação ao próprio SHA. Fix: `a35b85e` (teste agora impõe a licença
   proprietária nos 3 READMEs + arquivo).
2. (era P1-exposure) Binários distribuídos sem `-trimpath` embutiam paths
   absolutos da máquina de build (username + handle de perfil derivado de
   Gmail + layout do module cache) — verificado via `strings` no binário
   `dist/`. Fix: `5060862` (`-trimpath` em `.goreleaser.yaml`, nas 3
   targets do `Makefile` e nos 4 builds do `ci.yml`; re-scan com 0
   ocorrências; regressão coberta por
   `TestReleaseBuildsStripLocalPaths`).

## P1

Nenhum P1 técnico remanescente. P1s processuais (owner) listados em
gates externos.

## P2 (não bloqueiam o beta fechado)

- Quota de attachments por sessão tem TOCTOU sob uploads concorrentes
  (pior caso: ~2x do limite; limitado e sem corrupção).
- `EntitlementService` existe e é testado, mas nenhum call site o consome
  ainda: correto para o beta (não há feature premium), porém a primeira
  feature premium deve passar pelo gate server-side, nunca só UI.
- `PRAGMA wal_checkpoint` ignorado no `Close` (pré-existente; close nunca
  falha por isso).
- GOCACHE padrão desta máquina (`~/.cache/go-build`) está corrompido por
  troca de toolchains; auditoria usou `GOCACHE=/tmp/gocache-audit` limpo.
  CI (cache fresco) não é afetado.
- `DEV/CONTEXT.md` contém notas defasadas (ex.: worktree sujo, `yarn dev`);
  irrelevante para o artefato, mas deve ser higienizado.
- Tag local `v1.0.0-beta.1` existe apontando para `be6c793` (30 commits
  atrás) — ver gates externos.

## Regressões

`git log main..HEAD` (159 commits) + suítes de regressão por funcionalidade
verdes; nenhum comportamento existente removido foi detectado. O único
teste quebrado pelo ciclo desta missão foi o de metadados de licença
(corrigido acima).

## Security findings

- Attachments: re-testado independentemente (HTML-como-PNG, JS-como-JPEG,
  SVG, WebP corrompido/truncado, cauda polyglot após IEND, GIF/BMP,
  dimensões gigantes via IHDR, corpo > limite, traversal, MIME spoof,
  401/403/404, CSRF em mutação). Todos rejeitados ou normalizados sem
  vestígio (probes em `audit_probe_test.go`). Servir usa MIME do servidor
  + nosniff + disposition + cache explícitos; `storage_ref` não vaza em
  nenhuma rota (grep confirma ponto único de serialização).
- Auth: bootstrap one-shot em bind privado/tunnel e reutilizável em
  loopback com expiração (12 testes verdes); WS query-token rejeitado sob
  tunnel/remoto; rotate atômico; CORS echo somente com origem validada +
  CSP/nosniff/DENY globais. CSRF exigido em não-GET.
- Update: manifest adulterado, assinatura errada, chave desconhecida,
  downgrade e rollback cobertos (42 testes no pacote); rotação com
  revogação testada; keyring de produção vazio em builds locais.
- Store: migrations transacionais por arquivo (rollback em erro, replay
  idempotente); repairs com ledger e sem perda silenciosa (fixtures de
  upgrade); DB travado falha fechado via `busy_timeout` (sem hang).
- Conductor: PAUSED/BLOCKED nunca reportados como COMPLETED (teste de
  mapeamento); GoalIDs em ULID.
- Scan: `govulncheck ./...` limpo; `golangci-lint run ./...` 0 issues.

## Distribution findings

- Instalação beta não exige git/go/bun/node: caminho padrão de
  `install.sh`/`install.ps1` usa só curl|wget, sha256sum|shasum, tar;
  git/go/bun/make/winget aparecem SOMENTE no ramo `--build-from-source`
  (interno). Manifest assinado verificado (Ed25519) + SHA-256 vindo só do
  manifest; downgrade recusado.
- `dist/` local é ignorado pelo git (snapshot, não distribuído); release
  oficial sai do workflow com same-SHA gate (verificado em
  `release.yml`: espera o CI do SHA exato, falha caso contrário).
- Updater desacoplado do repo source (`ArtifactRegistry`; backend estático
  via `NEXUS_UPDATE_REGISTRY_URL`).

## Source exposure findings

- Bundle web: minificado, sem `.map`, sem `sourceMappingURL`, sem secrets/
  chaves/licensing-authority privada (grep). Extraível ≠ source original:
  requisito atendido.
- Binário Go: vazamento de paths locais ELIMINADO (`-trimpath`,
  verificado com `strings`: 0 ocorrências de `home/desenvolvedor` e do
  handle de perfil). Nenhum `.git`, teste, `.env`, token de CI ou chave
  privada no binário ou no bundle.
- Chave privada de release: somente ambiente protegido (nunca no repo;
  grep confirma ausência); trust root pública injetada via ldflags.

## Platform findings

- Linux: build + suite + race evidenciados aqui.
- Windows/macOS E2E, instaladores (NSIS/DEB/RPM), desktop WebKitGTK/
  WebView2 smoke, Authenticode e notarization: NÃO executáveis neste
  ambiente → `EXTERNAL_RELEASE_GATE` (não mascarados; comandos abaixo).
- Providers: contratos herméticos verdes (6 adapters + specs); twice:
  comportamento real de provider segue UNVERIFIED-por-mocks por definição.
  Maestro: degradação explícita e testada (fail-closed, sem fabricar
  advice).

## Test evidence (SHA 5060862, rerun independente)

- `go build ./...`, `go vet ./...`: PASS
- `go test ./...`: PASS (inclui o teste de licença corrigido)
- `-race`: release, store, update, control/web: PASS
- `golangci-lint run ./...`: 0 issues
- `govulncheck ./...`: No vulnerabilities found
- Frontend: frozen-lockfile, format, lint, styles, allowlist, typecheck,
  409 testes, build: PASS (re-executados)
- Playwright/axe/visual: NOT RUN (libatk ausente, sem sudo) — gate da CI
- SBOM CycloneDX gerado; `GO_MODULES.txt` pinado; licenças de terceiros
  indexadas (plist de verificação pelo owner no release).

## Confronto com FINAL_REPORT.md

O report é materialmente honesto (veredito condicional, Playwright NOT
RUN, gates externos listados), mas estava desatualizado em dois pontos no
próprio SHA: afirmava suite Go verde (estava vermelha — item P0-1 acima)
e não detectava o vazamento de paths no binário (item P0-2 acima). Ambos
corrigidos e cobertos por testes de regressão nesta certificação. Todo o
resto do report foi confirmado por evidência independente.

## Classificação

`CONDITIONAL_CERTIFICATION`

Únicos gates restantes são realmente externos. Para tornar definitiva:

1. `git push origin feat/nexus-maximum-delivery` + abrir/atualizar PR #2;
   aguardar CI verde no SHA `5060862` (inclui browser, Windows, macOS).
2. Mover a tag: `git tag -d v1.0.0-beta.1 && git tag v1.0.0-beta.1 5060862`
   (somente se a tag antiga não foi publicada; se foi, delete remoto +
   re-tag + comunique; nunca reutilize tag stale em silêncio).
3. Assinatura Windows: prover certificado Authenticode; ligar secrets no
   workflow de release (`EXTERNAL_SIGNING_GATE`).
4. macOS: prover Developer ID + credenciais de notarization; ligar
   signing/stapling (`EXTERNAL_NOTARIZATION_GATE`).
5. Distribuição: prover storage/CDN ou repo só-binários; definir
   `NEXUS_UPDATE_REGISTRY_URL` (ou `NEXUS_RELEASE_REPO`); injetar
   `NEXUS_UPDATE_PUBLIC_KEY` no build; guardar
   `NEXUS_UPDATE_PRIVATE_KEY` só no ambiente de release.
6. `GITHUB_TOKEN=<admin> ./scripts/apply-branch-protection.sh`
   (conferir nomes dos checks com `ci.yml`).
7. Confirmar com o owner: nome do titular do copyright em `LICENSE` e
   clearance de relicenciamento (sem CLA externo arquivado).
8. Rodar o release via `.github/workflows/release.yml` (same-SHA gate já
   impõe CI verde) seguindo `docs/release/BETA_PLAYBOOK.md`.
