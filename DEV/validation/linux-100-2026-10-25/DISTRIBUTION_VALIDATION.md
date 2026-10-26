## GoReleaser snapshot e ciclo DEB do overlay atual — 2026-10-26

- Base `release/v1.0.0@7a04f1b4cf5950b97b4ea1e156dda31ac513823b`; fingerprint de fonte `101e6e62b8d4b9903cf4d40309caeea54c809183e5dba017d9e6a8770302669c` (1.430 arquivos). `go run github.com/goreleaser/goreleaser/v2@v2.18.0 release --snapshot --clean --config <goreleaser.yaml isolado>` gerou o CLI Linux AMD64 beta.10 em diretório `dist` isolado no estado local; configuração, temporários e GOCACHE foram separados. GoReleaser selecionou Go 1.27.3. Nenhum conteúdo do `dist/` existente do repositório foi apagado.
- Artefatos fora do Git: `/home/desenvolvedor/.local/state/nexus-homologation/20261026-preview-copy-fix-release-snapshot/dist/nexus_Linux_x86_64.tar.gz` (SHA-256 `ed36483ebe682644988e12b35dbdfff1d63cfc94a4cc85e1b0d06cb020b79afe`) e `nexus_1.0.0-beta.10_linux_amd64.deb` (SHA-256 `18a1962646deac4bca2ebf2ffbfc422385e6a1dbcca30bc6e575b14341868819`). `sha256sum -c checksums.txt` passou; o DEB declara `Package=nexus`, `Version=1.0.0~beta.10`, `Architecture=amd64`; o tar contém CLI, README, LICENSE e changelog.
- O DEB foi instalado, executado (`nexus --version`) e removido em duas raízes dpkg isoladas. A primeira tentativa executou as operações mas dpkg não conseguiu abrir `/var/log/dpkg.log` fora da raiz devido a permissão; a repetição direcionou `--log` para dentro da raiz e concluiu sem esse aviso. Os logs preservam ambos os ensaios: `p5-current-recheck-2026-10-26-final/logs/goreleaser-current-overlay-package-validation-2026-10-26.txt` e `p5-current-recheck-2026-10-26-final/logs/goreleaser-current-overlay-package-validation-clean-2026-10-26.txt`.
- Este é um snapshot de homologação não assinado e não publicado. Não substitui a release estável v1.0.0, assinatura de produção nem o ciclo oficial do updater.

## Build e instalação do overlay FIX-035 — histórico — 2026-10-26

- `make build build-desktop-wails build-hud-wails` passou no overlay `047e6f968d64ee7f51ad389a21d43fe305e9ce9fb98b00c750be526b06c2313c`, sobre HEAD `7a04f1b4cf5950b97b4ea1e156dda31ac513823b`. CLI, Desktop e HUD são ELF x86-64; `ldd` não reporta bibliotecas ausentes.
- Os hashes do build e dos executáveis instalados coincidem: CLI `9349ed4cd2cf0fa124018ac83c49831d7d64521548cda5c6c8c5c200962614fc`, Desktop `c91834c27641f5cff8bb133ea268aded7bca50bfc8cf329e3dc63cd7a5c660ff`, HUD `7667b895cc5c884ab687db1ad982d44ccb7d3e9ed162ba98a366ed54cce65350`. Backup prévio preservado; unidade Desktop/HUD ativa; Core HTTP 200 na porta 36135.
- **Pacote GoReleaser deste fingerprint atual: NOT_RUN.** Os snapshots beta.10/DEB validados nas seções abaixo são de fingerprints anteriores e não devem ser tratados como pacote idêntico a `047e6f…`. A versão continua beta.10; não há release estável `v1.0.0`, assinatura de produção, assets oficiais nem ciclo oficial de updater/rollback validado.
- Evidência: [instalação atômica e hashes](logs/install-hud-wayland-geometry-fix-2026-10-26.txt), [verificação final do build ativo e health](logs/current-installed-wayland-geometry-verification-2026-10-26.txt) e [proveniência](logs/source-provenance-wayland-geometry-fix-2026-10-26.txt).

## Build e instalação locais após FIX-030/FIX-031 — 2026-10-26

- `make build build-desktop-wails build-hud-wails`: PASS em Linux AMD64. Artefatos: CLI `fb45b0a42af24aa3427958983fb9315cbab338c88713b0290c4cb3f708222a8f`, Desktop `6bbbafdab42309d2657ceed4ccf90f1d87c4381aee5d7b6d4528903a136dc61f`, HUD `76aabefd32c3eb88b5693546b3402e46f6006006250e2eff47822fa598e33e15`; `ldd` sem dependências ausentes. CLI e Desktop reportam `1.0.0-beta.10`/base `7a04f1b4`.
- Os três executáveis locais foram substituídos atomicamente após backup em `/home/desenvolvedor/.local/state/ai-manager/nexus-update-backups/20261026T041144Z-import-auth-maestro-fix`. A unidade Desktop/HUD reiniciada está ativa, e Core respondeu HTTP 200.
- Evidência: `p5-recheck-2026-10-26T0347Z/logs/linux-build-cli-desktop-hud.txt`, `p5-recheck-2026-10-26T0347Z/logs/linux-build-artifact-verification.txt`, `p5-recheck-2026-10-26T0347Z/logs/install-current-overlay-restart.txt`.
- Este é um build beta local do overlay de fontes `0df26de4a35641d1e5d02ded7757b257fa7f4f15747220416574c083b4127208`. Nenhum pacote GoReleaser foi gerado para este overlay nesta passagem, e não houve release/tag estável.

# Distribuição e atualização Linux

## Revalidação do snapshot exato P5 — 2026-10-25

Base e remoto `release/v1.0.0`: `7a04f1b4cf5950b97b4ea1e156dda31ac513823b`; overlay validado `2c23ebb0c7726a200acbde9326c23ad6023d8fad0238c2adfef993677203f2fa` (1.425 arquivos). GoReleaser v2.18.0 selecionou Go 1.27.3 e gerou CLI tar.gz e DEB Linux/amd64 beta.10 em `/home/desenvolvedor/.local/state/nexus-homologation/20261025T-p5-predispatch-goreleaser-retry/dist`.

| Artefato | SHA-256 |
| --- | --- |
| CLI tar.gz | `5f008932c9b7b12b030d59ce3769455b60d335d8ff72d1298cb136fac22ec63d` |
| DEB amd64 | `19bc2f01353cdc0de9973abc9cdd841a27c54078e4a02218f6c0d3f7703b1fb4` |
| `checksums.txt` | `58a6799724c9c8496d3ed99be4ec65c3c414f7985c9e4bcb8adddcc83768a361` |
| Payload CLI comum ao tar/DEB | `f8a250eb61095df94691aa12a7a462469d196aebb671bd93c54512f1e493d145` |

`sha256sum -c` passou; o pacote declara amd64/`1.0.0~beta.10`, e o CLI extraído reporta Linux/amd64 e o SHA-base esperado. O DEB foi instalado, consultado, executado e removido em dpkg root isolado sob `/home`; a instalação do host não foi tocada. A primeira tentativa usou temporários em `/tmp` e falhou antes da validação por falta de espaço temporário. A repetição direcionou `GOTMPDIR` e `dist` para estado isolado em `/home` e passou. Logs: [snapshot bem-sucedido](logs/p5-predispatch-goreleaser-snapshot-retry-2026-10-25.log), [validação de checksum, payload e ciclo dpkg](logs/p5-predispatch-package-validation-2026-10-25.txt) e [tentativa anterior com limite de espaço](logs/p5-predispatch-goreleaser-snapshot-2026-10-25.log).

Este é um snapshot unsigned beta.10, não uma release estável; não houve publicação, alteração de tag ou instalação do pacote no host. O updater oficial, assinatura de produção, assets estáveis e rollback continuam sem validação.

## Snapshot atual após P8 — overlay `268a9e9a…`

- Base e remote: `7a04f1b4cf5950b97b4ea1e156dda31ac513823b`; fingerprint das 1.421 fontes: `268a9e9a0036e89e39bcff07eca880869534c9c75cc13765384bb3fa3fc30b32` (`logs/p8-source-provenance.txt`). A árvore não está congelada.
- `make release-dry-run`: PASS; frontend, CLI e contratos `VERSION`, `web/package.json`, Wails e GoReleaser passaram.
- GoReleaser v2.18.0 `release --snapshot --clean` gerou CLI tar.gz e DEB AMD64 em `/tmp/nexus-linux-100-p8-20261025.gNjpMg/dist`. O toolchain do GoReleaser selecionou Go 1.27.3; o CLI do pacote identifica HEAD `7a04f1b4…`.
- Checksums: DEB `64e6f74b42f3eec6507a3ac4da1ddd7b867c87736d05ce69d4fab57c41a159cb`; tar.gz `8ef28f6ef88e630b4c70febec1691fc7195dc3cabbcef9f80fd9cb1d08346d0c`; manifest `cf76018ea7f4c2e16a29c4db8ffc206b115bd91ae067ea2f18e52c0da4f03dc3`.
- Payload CLI no tar e no DEB: `d1fa6eca3763efdb62759087fd5731e5a7db20146aaf2dcd67b8f71f07f92d34` em ambos. `sha256sum -c` passou; `file` reportou ELF x86-64 estaticamente linkado; `dpkg-deb` reportou `1.0.0~beta.10`, `amd64`.
- Instalação, consulta da versão e remoção passaram com `dpkg --force-not-root` numa raiz temporária em `/tmp/nexus-linux-100-p8-20261025.gNjpMg/dpkg-root`. A instalação do host permaneceu intacta. O Desktop e HUD Wails instalados coincidem byte a byte com os binários atualmente compilados; o CLI instalado e o novo `make build` têm o mesmo `build_date`, commit e versão, embora os hashes sejam diferentes. A causa dessa diferença não foi investigada nesta rodada.

Reprodução do snapshot: gerar uma configuração temporária com `dist: /tmp/<diretório>/dist` e `.goreleaser.yaml` do checkout; depois executar `NEXUS_VERSION=$(cat VERSION) NEXUS_UPDATE_PUBLIC_KEY= go run github.com/goreleaser/goreleaser/v2@v2.18.0 release --snapshot --clean --config /tmp/<diretório>/goreleaser.yaml`. Verificar com `(cd /tmp/<diretório>/dist && sha256sum -c checksums.txt)`, `dpkg-deb --field`, extração e comparação do payload. O ciclo isolado usado foi `dpkg --force-not-root --root=<raiz> --admindir=<raiz>/var/lib/dpkg --install/--remove`; nenhum `dpkg` do host foi alterado.

Isso valida um snapshot local, não uma release estável. Não há assinatura de produção, assets `v1.0.0` publicados, descoberta do updater GitHub Releases, download/update interrompido ou rollback nesse ciclo. Os binários e o snapshot ficam em `/tmp`; não foram adicionados ao Git.

## Snapshot GoReleaser no overlay corrente P5 — 2026-10-25

Fontes: HEAD-base/remoto `7a04f1b4cf5950b97b4ea1e156dda31ac513823b` + fingerprint d3caa597ae3bc061ad0048ad4d390b1b9bb0011cc496e4d213d0ec6119400796 (1.422 fontes). GoReleaser v2.18.0 criou snapshot CLI tar.gz e DEB `1.0.0~beta.10` Linux/amd64 em `/tmp/nexus-linux-100-p5-current-4d415ihx/dist`; o toolchain do GoReleaser selecionou Go 1.27.3.

| Artefato | SHA-256 |
| --- | --- |
| CLI tar.gz | `fa55abccb732fb0fc5b269cca57befe6ec83f07378d05493bbad319ca81c0b8b` |
| DEB amd64 | `f1d7ba24e8c97c3044718436b7940291a02f89494a324cbc9d8b3d17e6061ab9` |
| `checksums.txt` | `64dbfdc5874caf2f970d1700775629d2e724465fff8e9a44ab08fb5848ad2fa7` |
| Payload CLI comum ao tar/DEB | `92fed5a2fae1896b112032c14dc37fab62417bc398fb22ec91f94ef5c51d9efe` |

`sha256sum -c` passou; o metadata do DEB declara `amd64`; o CLI extraído reporta beta.10/Linux/amd64/base `7a04f1b4`. O DEB foi instalado, consultado, executado e removido com `dpkg --force-not-root` numa raiz temporária separada; a instalação do host não foi tocada. Logs: [build do snapshot](logs/p5-goreleaser-snapshot-current-overlay-2026-10-25.log) e [validação dos artefatos e ciclo dpkg](logs/p5-goreleaser-package-validation-current-overlay-2026-10-25.txt).

Este é um snapshot local unsigned, não uma release. `v1.0.0` estável, assinatura de produção e ciclo do updater seguem pendentes.

## Snapshot corrente após FIX-012 — `516b706a…`

Os builds finais atuais foram produzidos localmente para Linux/AMD64; nada foi publicado. Os tarballs Desktop/HUD seguem exatamente o formato do workflow CI existente. Payloads extraídos são byte a byte iguais aos executáveis compilados, `file` confirma ELF x86-64 e `ldd` não encontrou bibliotecas ausentes. Os binários brutos foram iniciados nativamente em GNOME/XWayland; isso valida execução local, não uma instalação de release.

| Artefato | Status | SHA-256 |
| --- | --- | --- |
| CLI GoReleaser tar.gz | PASS | `8fc86f6a435a28259e1d7029870f0669858ced731235b38680ea42c76e159598` |
| DEB amd64 GoReleaser | PASS | `f5c4932defd2745d6a256ee66478eb8eda2214089b16ffec1d2484a7ded1a83f` |
| `checksums.txt` | PASS | `a46519ce260fc0bdb5185923ba59d255741e91c6d10d79f8d92cf8e2cea6b17f` |
| Payload CLI no tar/DEB | PASS | `ce232d32c71ca54ce463be4150d8082a60be1ad944b53f46e30ad88e5dcfa8f7` |
| Tar Desktop do workflow CI | PASS | `8ed42fe048300f4069a4cbafb61378080c0aae9d2c7e950053fe7da67cf8f` |
| Tar HUD do workflow CI | PASS | `425e58cf811dc6b7245211bd289e4ab3f90fe908f922197f4e64f6b401f9eb2f` |

`sha256sum -c` passou. O DEB beta.10 amd64 foi instalado, consultado, executado e removido sob uma raiz dpkg temporária via `dpkg --force-not-root`; a instalação do host não foi tocada. Logs: [GoReleaser snapshot](logs/goreleaser-gdk-fix-snapshot.log), [ciclo de instalação/remoção](logs/distribution-gdk-fix-install-remove-rerun.txt), [artefatos e dependências](logs/gdk-popup-fix-build-artifacts.txt) e [validação dos tarballs nativos](logs/native-packages-gdk-fix-validation.txt). Caminho isolado: `/tmp/nexus-linux-active-20261025/`.

O snapshot não é uma release estável: assinatura, updater público, download/rollback e assets oficiais de `v1.0.0` continuam sem validação/publicação.

## Snapshot anterior — overlay de base

Os pacotes descritos abaixo foram criados em modo local `--snapshot` a partir do commit base `bf6adf0c80ac3531d7291844512a14a017ab0002` e do overlay `af7476a2ab2ffeabcbfcaf35051d6d24d61a0a35bd5786b5429dc2ee3a0662e4`; são evidência histórica do fluxo de empacotamento, não contêm o ajuste final do timeout. O overlay final (`825800ba46bab26ac692a8cc5f8cfc82f64ed474cf36459a672d09943c85b62b`) também foi empacotado em um snapshot isolado após FIX-005; os artefatos finais estão documentados abaixo. A versão é beta.10. Os artefatos ficaram em `/tmp/nexus-linux-100-2026-10-25`; não houve publicação.

| Artefato | Status | SHA-256 |
| --- | --- | --- |
| CLI GoReleaser tar.gz | PASS | `a0bf9ef490931ae5b78f445915695bf561915a709453f737c095ea7c27579ede` |
| DEB amd64 GoReleaser | PASS | `52cf2c9cca1a4204c914d7e3d08eb4f3dacc37406959c3f5aaa51e2cd19e943b` |
| Desktop tar.gz | PASS | `624de1a179fb4020c8d2bcbddcc9dde0f098467055da28bec7d996dcc8465de3` |
| HUD tar.gz | PASS | `7438e0b079ca13512b6e2711401f00841b2e865bffbcd2d89d1268e1aedb9c91` |

## Snapshot após FIX-005 — evidência histórica

GoReleaser v2.18.0 criou CLI tar.gz e DEB amd64 do overlay `825800ba46bab26ac692a8cc5f8cfc82f64ed474cf36459a672d09943c85b` em `/tmp/nexus-linux-100-2026-10-25-goreleaser-final/dist`. O checksum manifest passou; o payload `nexus` no tar e no DEB é byte a byte igual. O DEB foi instalado, consultado e removido em um dpkg root temporário, sem tocar na instalação do host. A execução do CLI extraído reportou beta.10, Linux amd64, Go 1.25.14 e commit-base bf6adf0c. Depois do snapshot, FIX-006 alterou o frontend das abas PTY (overlay atual `a6be83416651b6cdb4e0cb77a8c0448c17708d19e4c3b8d6f7e41bb8a5ad0b0a`); o Desktop foi recompilado, mas CLI/DEB/tarballs não foram empacotados novamente desse overlay.

| Artefato histórico | Status | SHA-256 |
| --- | --- | --- |
| CLI GoReleaser tar.gz | PASS | `3c49713e4082b8c9001c6cfdca2520471fe391c084101afc6ca24c8109024e49` |
| DEB amd64 GoReleaser | PASS | `db30dfab46e4cfada61e6b5c8587113d0fd2c18b8add588da896b5b076bb5098` |
| checksums.txt | PASS | `c2d82bf9a85b9168ba957c68d7ecb12e17f7cc8c538edca68816e75c7e45f592` |
| Payload CLI extraído do tar/DEB | PASS | `affe7fed441edfa149427fb79e5fac5a0ef8eeded264ead7dc5947fc0165d748` |

Evidências: [snapshot GoReleaser](logs/goreleaser-final-source-snapshot-retry.log), [checksums e execução do payload](logs/goreleaser-final-package-validation.log), [instalação/remoção isolada](logs/goreleaser-final-deb-install-remove.log). O primeiro comando de snapshot precisou declarar `NEXUS_UPDATE_PUBLIC_KEY` vazio para satisfazer o template; a repetição documentada passou. Nenhuma publicação foi feita.

## Evidência histórica do snapshot anterior

No snapshot de base, GoReleaser validou os checksums. O payload CLI do tar e do DEB era idêntico. Os tarballs Desktop/HUD eram byte a byte iguais aos binários daquele snapshot. O CLI GoReleaser tinha hash diferente do CLI `make build` por causa dos ldflags; ambos informavam beta.10, Linux AMD64 e o commit base bf6adf0c.

O DEB daquele snapshot foi instalado, consultado e removido sob `/tmp/nexus-linux-100-2026-10-25/deb-root`, sem tocar na instalação do host. Ver [verificação de payloads](logs/package-verification.json), [metadados DEB](logs/deb-info.txt), [log de instalação](logs/deb-isolated-install.log), [versão instalada](logs/deb-isolated-installed-version.json), [remoção](logs/deb-isolated-remove.log) e [checksums GoReleaser](logs/goreleaser-checksums.txt).

## GitHub Releases e updater

O build de snapshot local não é release de distribuição. A branch declara beta.10 e não há assets oficiais estáveis 1.0.0 assinados. Descoberta/atualização via GitHub Releases, manifesto de produção, assinatura, download interrompido, rollback e preservação de dados no update seguem sem validação de produção.

Nenhum formato novo de distribuição foi criado. Nenhuma tag ou release foi publicada.



## Snapshot após FIX-007 Gemini — histórico

O snapshot GoReleaser v2.18.0 foi recriado a partir do overlay `64c48060972e179d021f1406339ce3619389d0321f76ebcffad0414f46cbd904`, versão `1.0.0-beta.10`, em `/tmp/nexus-linux-live-gemini-smoke/goreleaser-final/dist`. O checksum manifest passou, os payloads CLI no tar.gz e no DEB são byte a byte idênticos e ambos reportam Linux/amd64 e commit-base `bf6adf0c`.

| Artefato | Status | SHA-256 |
| --- | --- | --- |
| CLI GoReleaser tar.gz | PASS | `65d3797cb896bb52753b19da9de54ae0f40d97f6ecdc6f763f20ff7f9a20c8fb` |
| DEB amd64 GoReleaser | PASS | `571db698e234db4062a95509d661d0f419d5bf39d6e9a7998a2b64b0faa16a87` |
| checksums.txt | PASS | `7d02930340029a898835cf91bbfe7ba1bc9c2f4480eb986085c0d187c8795f5a` |
| Payload CLI idêntico no tar/DEB | PASS | `6ff20faf39bac06e29bfa8e0bcbb5c476c2931b0747648d6fca0339ba25c1ce2` |

Extração com `dpkg-deb -x` e execução do payload passaram. Instalação em raiz dpkg temporária ficou **BLOCKED** porque `dpkg --root` requer privilégio de superusuário nesta sessão; nenhuma instalação no host foi tentada. O log detalhado está em [snapshot final](logs/goreleaser-final-gemini-overlay.txt). O snapshot local não está assinado nem publicado. A versão oficial permanece beta.10, sem assets estáveis assinados e sem ciclo de updater/rollback validado.

## Snapshot atual após FIX-008 — `996e1063540370263158e70903477f4cbbc3035dbb33b93161e603bffbe3c133`

GoReleaser v2.18.0 criou novamente CLI tar.gz e DEB beta.10 Linux/amd64 a partir do overlay corrigido. GoReleaser selecionou Go 1.27.3 via toolchain; o restante do build local usa Go 1.25.14. `sha256sum -c` passou. Os payloads extraídos do tar e do DEB são byte a byte idênticos e os dois executaram reportando Linux/amd64, beta.10 e commit-base bf6adf0c.

| Artefato | Status | SHA-256 |
| --- | --- | --- |
| CLI GoReleaser tar.gz | PASS | `289131adba5bce57ca4ca504a6c05ee37ead13caf8b43a14eee6a880d10cb3db` |
| DEB amd64 GoReleaser | PASS | `112bbd98c11139aa865f09e576239c4c49f0d0d3e508f47d387eac571ad01df0` |
| checksums.txt | PASS | `2ef6c4a47b51d094a975b7f7a36fba77e48e49338999567147f7af23378554aa` |
| Payload CLI idêntico no tar/DEB | PASS | `650d7dc827f8e0ba84f3861376b42d263b1d5402b4148b49d598741449f49c09` |
| Desktop tar do workflow CI | PASS | `46df90ec4409a74e1ac91a3153e6eecba37d7c8590467641914cfc723abaab32` |
| HUD tar do workflow CI | PASS | `3905bc6035d0cd2a7a45da8ffa5754468204933d94fe94e6c54713231e9e68ad` |

Os tarballs Desktop/HUD foram extraídos; os payloads coincidiram com os binários, `ldd` não apontou bibliotecas ausentes e os dois aplicativos iniciaram. O Desktop exibiu a tela inicial com seis provedores; o HUD empacotado mostrou uma arte NEX em 224×158. Evidência de build e execução: [pacotes nativos](logs/linux-native-packages-xwayland-position-fix.txt), [screenshot HUD empacotado](screenshots/native-hud-packaged-linux-position-fix.png), [snapshot CLI/DEB](logs/goreleaser-xwayland-position-fix.txt) e [log GoReleaser](logs/goreleaser-xwayland-position-fix.log).

A afirmação acima descreve a tentativa anterior sem `--force-not-root` e foi superada pela validação atual abaixo. A instalação dpkg isolada do snapshot após FIX-009 passou; nenhuma instalação do host, assinatura ou publicação ocorreu. O snapshot local não é uma release estável.


## Snapshot atual após FIX-009 — `dac0d00b8410f70ed02c8b4c324780349655fed8f7e580d1944da1ec3eaecefa`

GoReleaser v2.18.0 recriou o CLI tar.gz e o DEB AMD64 do overlay atual em `/tmp/nexus-linux-cursor-status-fix/goreleaser-dist`. O executável do GoReleaser foi instalado apenas em `/tmp`; o CLI do produto reporta Go 1.25.14. O snapshot usou `NEXUS_UPDATE_PUBLIC_KEY` vazio e não tem assinatura de release.

| Artefato | Status | SHA-256 |
| --- | --- | --- |
| CLI GoReleaser tar.gz | PASS | `381cc71579572ee98a9843c3671994c11a7ddad158c713a52fa224c2b12f8772` |
| DEB amd64 GoReleaser | PASS | `9ebb10f22ac3939b2b35abe050e38ec298eac0da09fd64ca645575cc36d79108` |
| `checksums.txt` | PASS | `944ea1ec1abc75afa9ecb13470dc98a1dc6bc190f1f433f620b0a74875bcc003` |
| Payload CLI idêntico no tar/DEB | PASS | `eb11ecb19cc7d380ab5edf219b78aa29768b5bae19c3749fa97d6248fbdb7521` |

`sha256sum -c` passou. O pacote declara `amd64` e versão `1.0.0~beta.10`. `dpkg --force-not-root` instalou o DEB em `/tmp/nexus-linux-cursor-status-fix/dpkg-root-verified`, `dpkg-query` confirmou o estado instalado, o CLI executou e reportou Linux/amd64, beta.10, Go 1.25.14 e commit-base bf6adf0c. A remoção passou e a instalação do host permaneceu intacta. Logs: [snapshot](logs/goreleaser-cursor-status-snapshot.log) e [validação limpa de instalação/remoção](logs/distribution-current-overlay-install-remove.log).

O primeiro comando de snapshot precisou receber `NEXUS_UPDATE_PUBLIC_KEY` vazio; o log do erro de template está em `logs/goreleaser-cursor-status-snapshot-missing-env.log`. Uma primeira consulta do pacote também teve erro somente na formatação shell após a instalação; o ciclo completo foi repetido sem esse erro e está no log de validação limpa.


## Revalidação do overlay atual — FIX-009

O snapshot GoReleaser v2.18.0 do overlay `dac0d00b8410f70ed02c8b4c324780349655fed8f7e580d1944da1ec3eaecefa` (1.401 fontes, versão beta.10) passou nos checksums. CLI tar e DEB contêm o mesmo payload SHA-256 `eb11ecb19cc7d380ab5edf219b78aa29768b5bae19c3749fa97d6248fbdb7521`. O DEB foi instalado, consultado e removido em raiz temporária com `dpkg --force-not-root`; a instalação do host não foi alterada.

Os tarballs Desktop/HUD gerados conforme o workflow CI correspondem aos payloads, são ELF x86-64 e `ldd` não encontrou dependências ausentes. O smoke do HUD esperou a janela real identificada pelo PID e pelo título X11 `nexus-hud` alcançar 224×158; a captura exibe um NEX e o processo encerrou com exit 0. A primeira tentativa usava o título incorreto `Nexus HUD` e capturou a janela inicial de 80×48 antes da restauração; a repetição corrigida passou, portanto esse falso negativo não representa defeito do aplicativo.

Evidências: [snapshot](logs/goreleaser-cursor-status-snapshot.log), [instalação isolada](logs/distribution-current-overlay-install-remove.log), [smoke do HUD empacotado](logs/native-packaged-hud-current-overlay.txt), [captura do HUD](screenshots/native-hud-packaged-current-overlay-xwayland.png).


## Snapshot após FIX-023 — 2026-10-25

- Fonte: `release/v1.0.0@7a04f1b4cf5950b97b4ea1e156dda31ac513823b` + overlay `a2ed858e013e8e43391cec7c1bb6d6e00324a3170ea9804d0d068bf231e73dfd` (1.422 fontes). GoReleaser v2.18.0, Go 1.27.3, Linux AMD64, versão `1.0.0-beta.10`.
- Artefatos: tar SHA-256 `2c7c2ee8f4d6a1b4b82eed2a5bed869d0dcae55e7939c49c5177a2c731ca7b8f`; DEB `7a5e6363026e90767f059e8fdee5141c73f4c172f8cd9434206dc4871138abde`; checksums `18705a56bdb982d37498436af172de5da98ee09a795fde516869d46b58523c08`. Payload tar/DEB idêntico: `12318eda12de10921f368666a23743ce59a590b5575c52ec0308d668b74521b0`.
- O DEB foi instalado/consultado/executado/removido em raiz temporária; host intocado. `web --help` do tar e flag desconhecida do DEB passaram. Snapshot unsigned, não publicado. Log: `logs/p2-cli-web-help-package-validation.log`; artefatos em `/tmp/nexus-linux-100-p2-cli-fix.67mSpN/dist`.
- A CLI instalada fora do pacote tem hash `51b58fe5972f27fcdb55c8284388b3f37998dc57db384b11223719ad6a132905`; veja `logs/p2-cli-web-help-current-build-smoke.txt`.


## Download interruption fixture — 2026-10-25

A new local regression serves a valid Ed25519-signed `1.0.1` fixture manifest and an artifact response whose declared `Content-Length` exceeds the bytes sent. `TestServiceApplyInterruptedDownloadPreservesBinaryAndConfig` receives `io.ErrUnexpectedEOF` and proves the existing executable and settings file stay byte-for-byte unchanged; no `.tmp`, `.bak` or update receipt is created. **PASS** in the focused test, package suite, race detector and `go vet`. No official release was queried or installed.

Reproduction: `GOCACHE=/tmp/nexus-linux100-update-recheck-gocache go test -race -count=1 ./internal/update -run '^TestServiceApplyInterruptedDownloadPreservesBinaryAndConfig$' -v`. Log: [interrupted download regression](logs/current-update-interrupted-download-regression-2026-10-25.txt); package evidence: `logs/current-update-tests-2026-10-25.txt`, `logs/current-update-race-2026-10-25.txt`, `logs/current-update-vet-2026-10-25.txt`.

This closes only the local interrupted-download failure case. Official signed release A→B update, installation recovery/rollback and live configuration preservation remain **NOT_RUN/BLOCKED** because stable signed `v1.0.0` assets and a production trust key are absent.
## Build e snapshot mais recentes — Linux AMD64 — 2026-10-26

- Proveniência: base local/remota `7a04f1b4cf5950b97b4ea1e156dda31ac513823b` + fingerprint de fontes `b73161ffd4afe7d566bd00cfdc6d6e090e95b0c99e13dbd0653aadd40fdf6eac` (1.428 arquivos; `logs/source-provenance-hud-idle-motion-fix-2026-10-26.txt`). Verificação de tamanhos e SHA-256 dos 1.428 arquivos passou; a árvore não está commitada/congelada.
- `make build build-desktop-wails build-hud-wails` passou. CLI/Desktop/HUD são ELF x86-64/AMD64; `ldd` não encontrou bibliotecas ausentes. SHA-256: CLI `5949669ab400fc63ca4b9e28c96b780e229430d68a7367ea67c99151e92e4e69`; Desktop `570f89398777bd93c36aa9f85d1494efc29aa25ff3bf0222c5d260205f6b8598`; HUD `fbe093cffa25e53a118bb5880a0087645d70ed73776678d55c588f549095c7b3`.
- GoReleaser v2.18.0 snapshot beta.10 gerou tar.gz e DEB `amd64`; checksums passaram, pacote instalado/consultado/executado/removido em dpkg root isolado. SHA-256 DEB `1d1e45099a49b5df57cc81bd5c842cbf7674d73c2dc0616fe3b6753ad8e72c9e`; tar.gz `123ea3276eb37c18be7e5de3368a01a4944056480d0899ddee3e35f616be8784`; checksums `478e0bda25e930e9f77dbdfc59c7242fab066d8da3c19707edef941fb4de875c`.
- Os executáveis foram instalados após backup em `/home/desenvolvedor/.local/bin`; hashes instalados coincidem com o build. A unidade de usuário está `active`, Desktop e HUD estão em execução e health retornou HTTP 200. Log: `logs/install-hud-idle-fix-restart-2026-10-26.txt`.
- A unidade era transitória: depois de parada, o primeiro `systemctl start` não a encontrou porque o systemd a havia descarregado. Recriei-a com `systemd-run`; nenhum binário foi restaurado do backup, e o estado `active`/health 200 foi confirmado. A ocorrência está descrita como nota suplementar no log de instalação.
- O snapshot é unsigned e local. Não valida assinatura de produção, atualização a partir de GitHub Releases nem rollback contra release estável. Não houve publicação/tag; a distribuição oficial e updater permanecem BLOCKED.
