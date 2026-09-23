# Pacotes e artefatos nativos

A distribuição estável do Nexus v1 publica somente artefatos **Linux amd64**.
Windows, macOS e Linux arm64 permanecem fora da matriz suportada até existir
certificação nativa correspondente.

## Linux amd64 — DEB

O GoReleaser gera DEB via [`.goreleaser.yaml`](../../.goreleaser.yaml).

- artefato: `nexus_*_linux_amd64.deb`;
- binário: `/usr/local/bin/nexus`;
- CI instala o DEB em Ubuntu, executa `nexus version`, remove o pacote e
  verifica que o comando desapareceu;
- Desktop Wails não faz parte do DEB CLI.

### Debian / Ubuntu

```bash
sudo dpkg -i nexus_*_linux_amd64.deb
# se faltar dependência:
sudo apt-get install -f -y
nexus version
```

## Archive de CLI

A v1 publica:

- Linux amd64: `nexus_Linux_x86_64.tar.gz`.

O instalador assinado valida manifesto Ed25519, SHA-256 e a versão embutida no
binário antes da substituição.

→ [Instalação por script](installation.md)

## Desktop nativo

A v1 publica somente:

- Linux x86_64: `nexus-desktop_Linux_x86_64.tar.gz`.

Windows e macOS possuem implementação no source, porém seus artefatos não são
publicados como parte da v1 estável.

→ [Suporte de plataformas](../operations/platform-support.md)

## Targets deferidos

Estes targets não fazem parte da promessa da v1:

- Linux arm64;
- Windows amd64/arm64;
- macOS amd64/arm64;
- RPM;
- NSIS;
- winget;
- DMG.

Arquivos preparatórios podem existir no source. Isso não é evidência de
distribuição.

## Regra de evidência

A presença de `packaging/*`, scripts de instalação ou de uma configuração de
build não prova suporte. A fonte de verdade é a combinação do workflow da
revisão, artifact gerado, smoke da mesma revisão e a matriz oficial de
[suporte de plataformas](../operations/platform-support.md).
