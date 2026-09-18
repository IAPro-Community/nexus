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
| Pacotes | DEB/RPM ([packages](../getting-started/packages.md)) | NSIS; winget após release ([packages](../getting-started/packages.md)) | app/DMG quando publicados |
| Docker (compat) | [docker-compat](../getting-started/docker-compat.md) — CLI+Web headless | mesmo (via WSL/Docker Desktop); Desktop nativo separado | mesmo headless |
| Updater | manifesto/checksum/assinatura | manifesto/checksum/assinatura | manifesto/checksum/assinatura |
| Native CI | obrigatório | obrigatório | obrigatório |

O status oficial de cada célula deve vir do CI nativo e do relatório da mesma
revisão. Para diagnóstico, consulte [troubleshooting](troubleshooting.md) e
use `nexus doctor --json`. Não trate uma plataforma como suportada apenas
porque `GOOS=<alvo> go build` passa.

Guias separados (não misturar escopos):

- [Instalação nativa](../getting-started/installation.md)
- [Pacotes](../getting-started/packages.md)
- [Docker](../getting-started/docker-compat.md)
