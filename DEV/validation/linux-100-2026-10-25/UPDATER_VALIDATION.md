# Homologação do atualizador Linux

## Reteste das políticas e fluxos isolados — 2026-10-25

Branch `release/v1.0.0`, HEAD local/remoto `7a04f1b4cf5950b97b4ea1e156dda31ac513823b`; overlay de fontes `2c23ebb0c7726a200acbde9326c23ad6023d8fad0238c2adfef993677203f2fa` (1.425 arquivos).

**PASS técnico:** `go test -count=1 ./internal/update`, `go test -race -count=1 ./internal/update` e `go vet ./internal/update`. A suíte cobre manifesto assinado com chave Ed25519 sintética, assinatura destacada, chave não confiável e manifesto adulterado, versão/canal/expiração/arquitetura, checksum e alvo, restrição por instalação gerenciada ou trabalho ativo, caminho de extração e traversal, limite de arquivo e rollback. O serviço também usa servidor HTTP local para validar discovery e download sem contatar uma release de produção.

Reprodução:

```bash
go test -count=1 ./internal/update
go test -race -count=1 ./internal/update
go vet ./internal/update
```

Log consolidado: [testes, race e vet](logs/p8-updater-package-recheck-2026-10-25.log). As chamadas de compare ao GitHub nos testes são interceptadas por transporte falso; as demais respostas vêm de `httptest`. Nenhuma credencial nem chave de produção foi usada.

## Ciclo oficial ainda bloqueado

O código aponta discovery para GitHub Releases, porém a branch reporta `1.0.0-beta.10`, não há tag/assets estáveis assinados `v1.0.0` nem chave Ed25519 de produção configurada. Portanto manifesto oficial, assinatura de release real, download/update sobre uma instalação isolada da versão estável, interrupção de rede e rollback de produção permanecem **BLOCKED**. O PASS da suíte local não certifica esse ciclo; o resultado global da homologação continua `NO_GO_LINUX`.


## Download interruption fixture — 2026-10-25

A new local regression serves a valid Ed25519-signed `1.0.1` fixture manifest and an artifact response whose declared `Content-Length` exceeds the bytes sent. `TestServiceApplyInterruptedDownloadPreservesBinaryAndConfig` receives `io.ErrUnexpectedEOF` and proves the existing executable and settings file stay byte-for-byte unchanged; no `.tmp`, `.bak` or update receipt is created. **PASS** in the focused test, package suite, race detector and `go vet`. No official release was queried or installed.

Reproduction: `GOCACHE=/tmp/nexus-linux100-update-recheck-gocache go test -race -count=1 ./internal/update -run '^TestServiceApplyInterruptedDownloadPreservesBinaryAndConfig$' -v`. Log: [interrupted download regression](logs/current-update-interrupted-download-regression-2026-10-25.txt); package evidence: `logs/current-update-tests-2026-10-25.txt`, `logs/current-update-race-2026-10-25.txt`, `logs/current-update-vet-2026-10-25.txt`.

This closes only the local interrupted-download failure case. Official signed release A→B update, installation recovery/rollback and live configuration preservation remain **NOT_RUN/BLOCKED** because stable signed `v1.0.0` assets and a production trust key are absent.
