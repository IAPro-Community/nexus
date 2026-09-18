# Release engineering red-team review

## Escopo e decisão

- **SHA auditado:** `e53f8352e4c3647632ebac7877cf0a6a3bd62647`
- **Versão declarada:** `0.5.0-beta.23` em `VERSION` e `web/package.json`
- **Decisão global:** **NO_GO**

| Área | Veredito |
|---|---|
| Release path | **NO_GO** |
| Updater | **NO_GO** |
| same-SHA CI | **NO_GO — NOT RUN / SHA ausente no remoto** |

## Bloqueadores

### R1 — Não existe CI hospedada para o SHA auditado

**Severidade:** bloqueador

`gh run list --commit e53f8352...` retornou `[]`. A API de check-runs retornou HTTP 422, `No commit found for SHA`, confirmando que o GitHub não conhece o commit. `git branch -r --contains e53f8352...` também não encontrou referência remota.

O gate de `.github/workflows/release.yml` é conceitualmente estrito: procura uma execução de `ci.yml` com `headSha` exatamente igual, exige conclusão global `success` e exige uma única ocorrência bem-sucedida de cada job crítico. Porém, esse gate nunca executou para o candidato e não pode ser substituído por testes locais.

**Conclusão:** o candidato não possui evidência same-SHA de Linux, Windows, macOS, Desktop, instaladores, segurança, browser ou snapshot GoReleaser.

### R2 — Não há release, tag nem artefatos certificáveis da versão

**Severidade:** bloqueador

`gh release view v0.5.0-beta.23` retornou `release not found`; não há tag local apontando para `e53f8352...`; não existe `dist/` local. Logo, não há arquivos publicados cujos digests, assinaturas ou attestations possam ser conferidos.

O workflow exige:

- chave pública Ed25519 de 64 caracteres hexadecimais;
- chave privada protegida;
- key ID exato `nexus-signing-key-2026-v1`;
- correspondência entre chave privada e pública ao gerar o manifesto;
- SHA-256 para artefatos;
- provenance via `actions/attest-build-provenance`.

Esses controles são positivos, mas configuração declarativa não prova que variáveis/segredos existem, que o ambiente protegido autorizou a execução ou que os produtos resultantes correspondem ao SHA auditado.

### R3 — Trust root de produção não está demonstrado nos produtos

**Severidade:** bloqueador

`internal/update.ProductionTrustRootHex` é vazio no código e somente recebe valor por `-ldflags`. Isso é fail-closed, mas torna a confiança dependente do caminho exato de build:

- GoReleaser injeta `NEXUS_UPDATE_PUBLIC_KEY` no CLI;
- builds Desktop de CI consomem a variável da CI, enquanto o job de release usa o environment `beta-signing`;
- o workflow não extrai nem testa os binários promovidos para provar que todos contêm a mesma chave pública usada para assinar o manifesto;
- `make build` e `make build-desktop` não injetam o trust root, portanto builds locais/source ficam sem chave de produção;
- `install.sh` e `install.ps1` usam apenas `NEXUS_UPDATE_PUBLIC_KEY` do ambiente ou um placeholder inválido; o workflow não renderiza/publica instaladores com uma chave fixa.

Consequentemente, o instalador zero-toolchain falha fechado por padrão, e os binários Desktop promovidos não têm prova de trust-root equivalente à chave do manifesto.

### R4 — Cadeia assinada não cobre os artefatos Desktop promovidos

**Severidade:** bloqueador

O manifesto Ed25519 é gerado a partir de `dist/` antes do download dos artefatos nativos. Assim, ele cobre os arquivos GoReleaser, mas não `native-artifacts/`.

Depois, `release-checksums.txt`/`desktop-checksums.txt` são gerados para `dist` e artefatos nativos e recebem attestation. Porém, os instaladores Desktop validam o download usando `desktop-checksums.txt`, sem validar assinatura Ed25519 ou GitHub attestation desse checksum. Um checksum obtido do mesmo canal que o binário fornece integridade de transporte, não uma raiz independente de autenticidade.

Também não foi encontrada configuração de Authenticode para Windows/NSIS nem codesign/notarização para macOS.

### R5 — Candidato local não está congelado

**Severidade:** bloqueador de certificação local

O arquivo de início registra `e53f8352...`, mas durante a auditoria o checkout avançou primeiro para `37f970b...` e depois para `61a9531...`. A árvore permaneceu suja, com arquivos modificados e diretórios de evidência não rastreados.

O SHA Git em si é imutável; o problema é probatório: resultados executados no checkout móvel/sujo não certificam o conteúdo exato de `e53f8352...`. Os 58 testes focados de updater/release passaram, mas são somente evidência suplementar e não same-SHA.

### R6 — Reprodutibilidade não está estabelecida

**Severidade:** alto

Não há `SOURCE_DATE_EPOCH`, comparação de dois builds limpos ou gate que recompile e compare digests. Os builds incluem timestamps variáveis (`BuildDate`, `time.Now()`, `date`, `Get-Date`), e o empacotamento nativo usa `tar`, `Compress-Archive` e `ditto` sem normalização determinística. As GitHub Actions são referenciadas por tags major (`@v2`, `@v4`, `@v5`, `@v6`), não por commits imutáveis.

Attestation de provenance identifica quem produziu um artefato; não demonstra que uma reconstrução do mesmo source gera bytes idênticos.

## Evidência favorável, insuficiente para GO

- Contrato de versão consistente em `0.5.0-beta.23`.
- Manifesto assinado com Ed25519 sobre os bytes exatos publicados.
- Updater rejeita key ID desconhecido, assinatura inválida, expiração, downgrade, target ausente, tamanho/hash incorretos e archives inseguros.
- O release gate exige CI do mesmo SHA e todos os jobs críticos bem-sucedidos.
- Testes focados locais: `go test ./internal/update ./internal/release ./scripts/sign-update-manifest` — 58 testes aprovados.

## Condições mínimas para nova decisão

1. Congelar um checkout limpo no SHA final, publicar esse SHA e criar a tag correspondente.
2. Obter uma execução completa de `ci.yml` no mesmo SHA e deixar o `same-sha-gate` consumi-la com sucesso.
3. Executar a release no environment protegido e verificar a release publicada, seus digests, manifesto/signature e attestations.
4. Provar por inspeção/teste dos binários CLI e Desktop que o trust root embutido corresponde à chave pública do manifesto.
5. Cobrir artefatos Desktop/NSIS por uma cadeia de autenticidade verificada pelo instalador, além de checksum co-hospedado.
6. Produzir dois builds a partir de checkouts limpos equivalentes e documentar/comparar digests, ou declarar precisamente quais artefatos ainda não são reprodutíveis.

Até essas condições serem satisfeitas no mesmo SHA, **release path, updater e same-SHA CI permanecem NO_GO**.
