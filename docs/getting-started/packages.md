# Pacotes e artefatos nativos

A distribuição v1 separa três caminhos: instalação por script assinado,
pacote DEB para Linux e archives nativos de CLI/Desktop. Não trate arquivos de
packaging presentes no source como artefatos publicados sem um gate correspondente
no CI/release.

## Linux — DEB

O GoReleaser gera **DEB** via `nfpms` em
[`.goreleaser.yaml`](../../.goreleaser.yaml).

- artefatos: `nexus_*_linux_amd64.deb` e `nexus_*_linux_arm64.deb` quando o build correspondente existir;
- binário: `/usr/local/bin/nexus`;
- CI atual: instala o DEB em Ubuntu, executa `nexus version`, remove o pacote e
  verifica que o comando desapareceu;
- Desktop Wails não faz parte do DEB CLI.

### Debian / Ubuntu

```bash
sudo dpkg -i nexus_*_linux_amd64.deb
# se faltar dependência:
sudo apt-get install -f -y
nexus version
```

## Archives de CLI

O GoReleaser produz archives de CLI para:

- Linux amd64 e arm64: `.tar.gz`;
- Windows amd64 e arm64: `.zip`;
- macOS amd64 e arm64: `.tar.gz`.

O instalador assinado seleciona o archive correto, valida manifesto Ed25519,
SHA-256 e a versão embutida no binário antes da substituição.

→ [Instalação por script](installation.md)

## Desktop nativo

Os artefatos Desktop atualmente exigidos pelo pipeline de fechamento são:

- Linux x86_64: `nexus-desktop_Linux_x86_64.tar.gz`;
- Windows x86_64: `nexus-desktop_Windows_x86_64.zip`;
- macOS arm64: `nexus-desktop_Darwin_arm64.zip`.

Arquiteturas de CLI sem Desktop correspondente continuam instaláveis; o
instalador informa explicitamente que o shell nativo não está publicado naquele
alvo.

→ [Suporte de plataformas](../operations/platform-support.md)

## RPM, NSIS e winget

Existem arquivos preparatórios no source para alguns formatos futuros, mas o
pipeline v1 atual **não publica nem certifica** RPM, NSIS ou winget.

Em particular:

- o CI atual verifica explicitamente que o snapshot GoReleaser contém zero RPMs;
- não há job NSIS no workflow de CI atual;
- o manifesto winget é apenas material preparatório e não deve ser publicado
  até existir um instalador Windows estável, assinado e com SHA-256 real.

Esses formatos só podem mudar para “publicados” quando houver geração,
assinatura quando aplicável, smoke nativo e gate de release correspondente.

## Regra de evidência

A presença de `packaging/*` ou de um arquivo de configuração não é evidência de
distribuição. A fonte de verdade é a combinação do workflow da revisão, artefato
gerado e smoke da mesma revisão.
