# Suporte de plataformas

A release estável do **Nexus v1** tem um escopo deliberadamente menor que o
source: **Linux amd64 é a única plataforma suportada e certificável nesta linha
de release**.

Windows e macOS continuam com implementação no source e com um modo de
certificação nativa preparado no CI, mas permanecem **UNVERIFIED / DEFERRED**.
Eles não são publicados nem contam como requisito para declarar a v1 Linux
pronta. Cross-compilation nunca deve ser apresentada como certificação nativa.

| Capacidade | Linux amd64 — v1 | Windows amd64 | macOS arm64 |
| --- | --- | --- | --- |
| CLI/Core | **suportado** | source disponível; release deferida | source disponível; release deferida |
| Web frontend | **suportado** | não certificado na v1 | não certificado na v1 |
| PTY/runtime | **PTY + UDS certificado pelo CI** | ConPTY no source; não certificado | PTY + UDS no source; não certificado |
| Browser E2E/Axe/visual | **gate obrigatório** | não é target de release v1 | não é target de release v1 |
| Desktop Wails | **artefato Linux x86_64** | não publicado na v1 | não publicado na v1 |
| Instalação assinada | **`install.sh`** | não publicada na v1 | não publicada na v1 |
| Pacotes/artefatos | **CLI tar.gz + DEB + Desktop tar.gz** | não publicados na v1 | não publicados na v1 |
| Updater | **manifesto/checksum/Ed25519** | sem artifact v1 suportado | sem artifact v1 suportado |
| Evidência nativa | **CI Linux self-hosted** | deferida | deferida |

## CI contínuo da v1

O pipeline contínuo (`.github/workflows/ci.yml`) roda em
`[self-hosted, Linux, X64, nexus-ci]` e cobre os gates de release da v1:

- frontend format/lint/styles/typecheck/unit/build/verify;
- documentação;
- Go fmt/vet/lint/race e E2E;
- security/govulncheck;
- Browser E2E, Surface Isolation, Capacity, Composer, Axe e visual smoke;
- Desktop Linux Wails;
- GoReleaser snapshot;
- instalação/remoção do DEB.

Windows e macOS aparecem como jobs **skipped** no fluxo normal. Isso é
intencional e não é interpretado como PASS.

## Certificação futura de Windows/macOS

O modo manual do `.github/workflows/ci.yml` foi preservado para a linha futura
que reativar suporte nativo. Com `native_target_sha=<SHA completo>`, ele exige:

- Windows amd64: `[self-hosted, Windows, X64, nexus-windows]`;
- macOS arm64: `[self-hosted, macOS, ARM64, nexus-macos]`.

Esse modo pode produzir `native-certification-<sha>`, mas esse certificado não
é requisito da release Linux-only da v1. Quando Windows/macOS voltarem à matriz
oficial, o gate de release deve ser ampliado novamente antes de qualquer anúncio
de suporte.

## Artefatos da v1

| Artefato | Linux amd64 | Demais alvos |
| --- | --- | --- |
| CLI/Core archive | **publicado** | **não publicado na v1** |
| DEB | **publicado** | N/A |
| Desktop nativo | **`nexus-desktop_Linux_x86_64.tar.gz`** | **não publicado na v1** |
| Instalador | **`install.sh`** | **não publicado na v1** |
| Uninstaller | **`uninstall.sh`** | **não publicado na v1** |

A presença de código, scripts PowerShell ou paths de runtime para outros sistemas
não altera essa matriz. Uma plataforma só volta a ser suportada depois de build,
runtime smoke, install/update/uninstall e evidence same-SHA nativos.

Para diagnóstico, consulte [troubleshooting](troubleshooting.md) e use
`nexus doctor --json`.

Guias relacionados:

- [Instalação nativa](../getting-started/installation.md)
- [Pacotes](../getting-started/packages.md)
- [Docker](../getting-started/docker-compat.md)
