# Distribuição e atualização Linux

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
