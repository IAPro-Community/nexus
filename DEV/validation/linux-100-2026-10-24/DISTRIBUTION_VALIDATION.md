# Distribuição e atualização Linux

## Artefatos de homologação

Os artefatos abaixo foram compilados/reempacotados para Linux AMD64 a partir do commit base `bf6adf0c80ac3531d7291844512a14a017ab0002` mais o overlay final identificado pelo fingerprint `84bef2759af6b975d73e0488ab80e5fced2df70f17855597d512dcdef1f29ec8`. A versão embutida continua `1.0.0-beta.10`.

| Artefato | Resultado | Local temporário | SHA-256 |
| --- | --- | --- | --- |
| CLI Nexus | PASS | `/tmp/nexus-linux-100-2026-10-24/res-001-rebuild-2026-10-25/bin/nexus` | `f7470fcee9a3f1a2f4eba2bf2dd8f9aac443c5cade2de30c7225561586496c86` |
| CLI tar.gz GoReleaser | PASS, snapshot | `/tmp/nexus-linux-100-2026-10-24/goreleaser-final-resilience-2026-10-25/nexus_Linux_x86_64.tar.gz` | `58d1ebf202f5bc4bb2966ecf05f1caa57407ebcbf9094418b61c4a2b21f2d5c1` |
| CLI DEB GoReleaser | PASS, snapshot | `/tmp/nexus-linux-100-2026-10-24/goreleaser-final-resilience-2026-10-25/nexus_1.0.0-beta.10_linux_amd64.deb` | `d0d92dbd74c558c4ad699e0aa4db8b9fb53a527502d4da7062d4592a4e14953d` |
| Desktop Wails | PASS, build/start Core | `/tmp/nexus-linux-100-2026-10-24/res-001-rebuild-2026-10-25/bin/nexus-desktop` | `2d0c4af7b8717776e7b95133dfcdaa8f16e2495c3b243a9925053ad3aef6dcef` |
| HUD nativo | PASS, build/start Core | `/tmp/nexus-linux-100-2026-10-24/res-001-rebuild-2026-10-25/bin/nexus-hud` | `57a30228ed3060c134a91d6ee7cf5202ae941799faae7960729f2ac54b2dca7d` |
| Desktop tar.gz conforme workflow | PASS, payload equivalente | `/tmp/nexus-linux-100-2026-10-24/desktop-packages-resilience-2026-10-25/nexus-desktop_Linux_x86_64.tar.gz` | `f8c40d687cc66df69edb8428ee2c9cc246c75cb733b9d0a2cf923f48647c9049` |
| HUD tar.gz conforme workflow | PASS, payload equivalente | `/tmp/nexus-linux-100-2026-10-24/desktop-packages-resilience-2026-10-25/nexus-hud_Linux_x86_64.tar.gz` | `57d4819b0b6bcf53e92a166ad074f65a832e390ca52213ff3e7f220059d51bb8` |

Desktop e HUD foram extraídos e comparados byte a byte com os executáveis do build final. CLI tar e DEB são do mesmo snapshot GoReleaser e seus binários extraídos coincidem entre si byte a byte; o CLI isolado de `make build` usa ldflags diferentes e por isso tem outro hash. O CLI tar e o DEB passaram pela verificação de checksums; o DEB declara arquitetura `amd64`, e o CLI extraído corresponde ao binário produzido pelo GoReleaser. `nexus version --json` no CLI extraído confirmou `1.0.0-beta.10`, Linux AMD64 e o commit base testado. Evidências: `logs/build-resilience-final-sha256.txt`, `logs/distribution-package-verification-resilience-final.txt`, `logs/deb-info-final.txt` e `logs/artifact-validation.txt`.

Duas tentativas de preparação do snapshot final falharam por configuração do harness (PATH sem Go e variável pública de trust root ausente); seus logs foram preservados. A execução com Go e variável de ambiente explicitados gerou o snapshot final com sucesso em `logs/goreleaser-snapshot-resilience-final.log`. Nenhuma publicação ocorreu.

## GitHub Releases e instalador

`install.sh` e a documentação de pacotes apontam para GitHub Releases. A consulta remota de 2026-10-24 não encontrou release estável `1.0.0`. O preview mais recente é `preview-f23ed9b88725e9bc4df80ded19a89b2e7f8a326a`, publicado em 2026-10-22, anterior ao HEAD homologado. Seus assets não fornecem o conjunto Linux estável CLI/DEB/Desktop/HUD. Evidências: `logs/github-releases-list.txt` e `logs/github-latest-preview.json`.

## Atualizador e publicação

- Pacote CLI DEB local de snapshot: **PASS**; pacote estável `1.0.0`: **FAIL**, não publicado.
- Atualização de produção via GitHub Releases e manifesto oficial assinado: **BLOCKED**, sem artefato estável e sem chave de produção para este teste.
- Verificação de assinatura sobre release publicada, download interrompido, recuperação, rollback e preservação de preferências: **NOT_RUN**.
- Instalação/desinstalação do DEB em `dpkg --root` temporário: **PASS**; `dpkg --force-not-root` instalou Nexus amd64 beta.10, o executável coincidiu byte a byte com o payload e foi removido sem tocar a instalação ativa. Evidência: `logs/distribution-package-verification-resilience-final.txt`.
- Instalação global: não executada.

A primeira tentativa de `dpkg --root` foi recusada por falta de privilégio de superusuário; foi repetida com `--force-not-root` e log redirecionado à raiz temporária, sem escrita no sistema host. Evidências: `logs/deb-isolated-install-remove-final.log` e `logs/distribution-package-verification-resilience-final.txt`. Nenhuma forma de distribuição nova foi criada. Nenhuma tag ou release foi publicada.


O snapshot GoReleaser final foi produzido somente em `/tmp`, sem publicar. `NEXUS_UPDATE_PUBLIC_KEY` estava vazio porque não havia trust root de produção nesta sessão; o artefato beta.10 é um snapshot local, não um instalador de release estável ou uma validação do updater assinado. O DEB final foi instalado e removido em dpkg root temporário; `logs/distribution-package-verification-resilience-final.txt` registra checksums, equivalência do payload, versão, arquitetura e remoção.
