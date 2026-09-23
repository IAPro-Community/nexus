# Suporte de plataformas

Build, runtime, terminal nativo, empacotamento e atualização são gates
independentes. Cross-compilation prova somente compilação para o alvo; não
prova execução nativa.

| Capacidade | Linux amd64 | Windows amd64 | macOS arm64 |
| --- | --- | --- | --- |
| CLI/Core build | CI nativo | CI nativo | CI nativo |
| Web frontend | compartilhado | compartilhado | compartilhado |
| PTY/runtime | PTY + UDS | ConPTY + Named Pipe | PTY + UDS |
| Browser E2E/Axe/visual | runner Linux | compartilhado | compartilhado |
| Desktop Wails build | binário ELF | `.exe` | bundle `.app` |
| Desktop runtime smoke | somente com evidência nativa | somente com evidência nativa | somente com evidência nativa |
| Instalação por script | [`install.sh`](../getting-started/installation.md) | [`install.ps1`](../getting-started/installation.md) | `install.sh` |
| Pacotes/artefatos v1 | DEB + CLI/Desktop archives ([packages](../getting-started/packages.md)) | CLI/Desktop ZIP; NSIS/winget ainda não certificados ([packages](../getting-started/packages.md)) | CLI tar.gz + Desktop .app ZIP em arm64; DMG não publicado |
| Docker (compat) | [docker-compat](../getting-started/docker-compat.md) — CLI+Web headless | mesmo (via WSL/Docker Desktop); Desktop nativo separado | mesmo headless |
| Updater | manifesto/checksum/assinatura | manifesto/checksum/assinatura | manifesto/checksum/assinatura |
| Evidência nativa | CI Linux contínuo | certificação self-hosted por SHA | certificação self-hosted por SHA |

### CI contínuo versus certificação nativa

O pipeline contínuo (`.github/workflows/ci.yml`) não depende de minutos pagos de
runners GitHub-hosted. Ele roda em `[self-hosted, Linux, X64, nexus-ci]` e cobre
frontend, segurança, Browser E2E/Axe/visual, Go/race/vet/lint, Desktop Linux,
GoReleaser snapshot e smoke de pacote DEB.

Windows e macOS são gates de **certificação nativa do candidato**, não jobs de todo
commit. O workflow manual `.github/workflows/native-certification.yml` recebe o
`target_sha` completo e exige:

- Windows amd64: `[self-hosted, Windows, X64, nexus-windows]`;
- macOS arm64: `[self-hosted, macOS, ARM64, nexus-macos]`.

As duas máquinas podem permanecer offline durante o desenvolvimento normal. Para
certificar um candidato, inicie os runners, execute **Native Certification** para o SHA
congelado e aguarde os dois jobs nativos. O workflow só sela
`native-certification-<sha>` depois que Windows e macOS passarem. Esse artefato contém
`certified-sha.txt` e os hashes dos binários Desktop produzidos.

`release.yml` não aceita apenas o status do workflow: ele baixa o certificado e
confere o SHA. Os binários Windows/macOS promovidos também precisam vir do mesmo run
nativo certificado. Portanto uma evidência de outro commit não pode autorizar release.

#### Registro dos runners sem custo de minutos hospedados

No repositório/organização, abra **Settings → Actions → Runners → New self-hosted
runner** e execute os comandos gerados pelo próprio GitHub em cada máquina.

Use os labels adicionais:

- Windows: `nexus-windows`;
- Mac Apple Silicon: `nexus-macos`.

Não use `--no-default-labels`: o workflow também exige os labels automáticos de
sistema/arquitetura (`Windows/X64` e `macOS/ARM64`). Para uso eventual, não é
necessário manter o runner como serviço; ele pode ser iniciado apenas durante a
certificação do candidato.

### Arquiteturas de artefato

A matriz de artefatos de CLI/Core é mais ampla que a matriz Desktop:

| Artefato | Linux amd64 | Linux arm64 | Windows amd64 | Windows arm64 | macOS amd64 | macOS arm64 |
| --- | --- | --- | --- | --- | --- | --- |
| CLI/Core archive | publicado pelo GoReleaser | publicado pelo GoReleaser | publicado pelo GoReleaser | publicado pelo GoReleaser | publicado pelo GoReleaser | publicado pelo GoReleaser |
| Desktop nativo | candidato exige CI nativo | **não publicado** | candidato exige CI nativo | **não publicado** | **não publicado** | candidato exige CI nativo |

O instalador não transforma ausência de Desktop em falha do CLI. Em arquiteturas sem
Desktop publicado/certificado, instala o CLI/Core e informa explicitamente que o shell
nativo não está disponível naquele alvo.

O status oficial de cada célula deve vir do CI nativo e do relatório da mesma
revisão. Para diagnóstico, consulte [troubleshooting](troubleshooting.md) e
use `nexus doctor --json`. Não trate uma plataforma como suportada apenas
porque `GOOS=<alvo> go build` passa.

Guias separados (não misturar escopos):

- [Instalação nativa](../getting-started/installation.md)
- [Pacotes](../getting-started/packages.md)
- [Docker](../getting-started/docker-compat.md)
