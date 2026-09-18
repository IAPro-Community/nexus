# Pacotes nativos (DEB / RPM / NSIS / winget)

![Pacotes nativos DEB RPM NSIS winget](../assets/help/help-install-packages.png)

Distribuição por **pacote de sistema** ou instalador gráfico. Separado da instalação via script ([installation.md](installation.md)) e do Docker ([docker-compat.md](docker-compat.md)).

## Linux — DEB e RPM

Gerados pelo GoReleaser (`nfpms` em [`.goreleaser.yaml`](../../.goreleaser.yaml)).

- Artefatos típicos na release: `nexus_*_linux_amd64.deb`, `nexus_*_linux_amd64.rpm` (e arm64 quando aplicável).
- Binário em `/usr/local/bin/nexus`.
- O CI faz smoke: `dpkg -i` (Ubuntu) e `rpm -i` (Fedora) → `nexus version`.

### Debian / Ubuntu

```bash
# Baixe o .deb da release correspondente e:
sudo dpkg -i nexus_*_linux_amd64.deb
# se faltar dependência:
sudo apt-get install -f -y
nexus version
```

### Fedora / RHEL

```bash
sudo rpm -i nexus_*_linux_amd64.rpm
nexus version
```

Desktop Wails **não** entra no pacote CLI; use o artefato `nexus-desktop_*` da mesma release ou o instalador de script com desktop.

## Windows — NSIS

Job CI dedicado gera `nexus-setup-Windows_x86_64.exe` ([`packaging/windows/nexus.nsi`](../../packaging/windows/nexus.nsi)).

- Instala em `%LOCALAPPDATA%\Programs\IAPro Nexus`
- Inclui `nexus.exe`; inclui `nexus-desktop.exe` quando o artefato desktop estiver no payload
- Ajusta PATH do usuário e cria atalho no Desktop quando houver desktop
- A release **exige** esse artefato no gate de publicação

```powershell
# Após baixar da GitHub Release:
.\nexus-setup-Windows_x86_64.exe
nexus doctor
nexus web
```

Fallback sem instalador gráfico: [installation.md — Windows](installation.md#windows-powershell).

## Windows — winget

Manifesto stub em [`packaging/winget/IAPro.Nexus.yaml`](../../packaging/winget/IAPro.Nexus.yaml).

Só publique no repositório winget-pkgs **depois** de existir NSIS estável na release com SHA-256 real. Substitua `VERSION` e `InstallerSha256` no manifesto antes do PR.

## Relação com a matriz

Status por OS: [platform-support.md](../operations/platform-support.md).
