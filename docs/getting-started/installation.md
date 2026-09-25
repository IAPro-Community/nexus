# Instalação nativa por release assinada

A release estável Nexus v1 ainda não foi publicada. A última beta pública
registrada é **v1.0.0-beta.1 / Linux amd64**. O candidato seguinte
**v1.0.0-beta.3** integra Linux amd64, Windows amd64 e macOS arm64; suporte
público depende da certificação exact-SHA correspondente.

| Quero… | Documento |
| --- | --- |
| DEB e matriz de artefatos | [Pacotes e artefatos](packages.md) |
| Rodar no Docker (CLI + Web) | [Docker compatibilidade](docker-compat.md) |
| Compilar o source como desenvolvedor | [Build](../../BUILD.md) |
| Ver a matriz de suporte | [Suporte de plataformas](../operations/platform-support.md) |

## Release pública atualmente documentada

A documentação pública mantém `v1.0.0-beta.1` como última beta publicada
registrada. A versão do source candidato pode ser superior e não implica que o
mesmo tag já esteja distribuído.

Enquanto a linha pública for beta, use uma tag explícita. O valor `latest`
não deve ser usado como sinônimo de “beta mais nova”.

## Linux amd64

Baixe o instalador da própria release e pine a mesma versão:

```bash
curl -fsSL \
  https://github.com/IAPro-Community/nexus-releases/releases/download/v1.0.0-beta.1/install.sh \
  -o install.sh

bash install.sh --version=v1.0.0-beta.1
nexus doctor
nexus web
```

Pré-requisitos:

- Linux amd64;
- `curl` ou `wget`;
- `tar`;
- `sha256sum` ou `shasum`;
- Python 3 com `cryptography` para Ed25519;
- WebKitGTK apenas para o Desktop Linux.

Parâmetros principais:

```text
--no-desktop
--no-path
--install-dir=<path>
--with-maestro
--version=vX.Y.Z
```

`NEXUS_INSTALL_DIR` é o equivalente por ambiente. `NEXUS_RELEASE_REPO` e
`NEXUS_RELEASE_BASE_URL` existem para distribuição interna/certificação.

## Windows e macOS

O trabalho anteriormente planejado para a linha 1.1 foi incorporado à própria
**v1.0 enquanto ela ainda está em beta**. O candidato `v1.0.0-beta.3` inclui
Linux + Windows + macOS no source. A publicação dos artefatos nativos continua
bloqueada até o mesmo SHA concluir a certificação correspondente.

### Windows amd64

O caminho de distribuição passa a ser o instalador NSIS assinado:

```text
nexus-setup-Windows_x86_64.exe
```

Windows amd64 faz parte do candidato `v1.0.0-beta.3`. A certificação exige
Authenticode válido, timestamp, build Wails nativo, ConPTY/Named Pipe E2E e
smoke completo de install/version/doctor/uninstall. O NSIS instala também o
`nexus-hud.exe`, certificado e assinado no mesmo SHA. `install.ps1`
continua disponível para automação, mas o instalador nativo é o bootstrap
recomendado para usuários Windows.

### macOS arm64

O caminho de distribuição passa a ser o pacote notarizado:

```text
nexus-installer_Darwin_arm64.pkg
```

macOS arm64 faz parte do candidato `v1.0.0-beta.3`. A certificação exige
Developer ID, notarização Apple, stapling, Gatekeeper, build Wails nativo,
PTY/socket E2E e smoke completo de install/version/doctor/uninstall. O pacote
instala o CLI em
`/usr/local/bin/nexus`, o app em `/Applications/IAPro Nexus.app`, o HUD em
`/Applications/IAPro Nexus HUD.app` e `nexus-uninstall` para remoção
preservando dados.

Consulte [Suporte de plataformas](../operations/platform-support.md) para os
runners, secrets e gates exact-SHA.

## O que o instalador verifica

A instalação é fail-closed:

```text
update-manifest.json
  + update-manifest.sig
          |
          v
Ed25519 com trust root embutida
          |
          v
SHA-256 do artifact
          |
          v
extração segura
          |
          v
nexus version --json
          |
          v
versão interna == versão solicitada
          |
          v
substituição do binário
```

Se assinatura, hash, artifact, extração ou versão interna divergirem, o binário
não é aceito.

A chave pública e o key ID usados pelo build de release são versionados em
`UPDATE_PUBLIC_KEY` e `UPDATE_KEY_ID`; o workflow de publicação falha quando
os secrets protegidos divergem desses contratos.

## Upgrade

```bash
nexus update check
nexus update
```

Instalações gerenciadas por pacote não são sobrescritas silenciosamente pelo
self-updater.

Para migrar deliberadamente de beta para stable, use a versão stable pinada com
o instalador assinado ou `nexus update --channel stable` quando o cliente
suportar troca de canal.

## Desinstalação

```bash
curl -fsSL \
  https://github.com/IAPro-Community/nexus-releases/releases/download/v1.0.0-beta.1/uninstall.sh \
  -o uninstall.sh

bash uninstall.sh
```

Por padrão, os dados do usuário são preservados. Para removê-los também:

```bash
bash uninstall.sh --purge-data
```

## Desenvolvimento a partir do source

O source é um caminho de engenharia, não o bootstrap recomendado de cliente.

Requisitos: Go 1.25+, Bun 1.3.9+, Git e Make.

```bash
git clone https://github.com/IAPro-Community/nexus.git
cd nexus
bun install --cwd web --frozen-lockfile
make build
./nexus version
./nexus doctor
```

Builds de desenvolvimento e o source integrado não transformam uma plataforma
em suportada. A matriz pública muda somente depois da evidência exact-SHA.

## Diagnóstico

```bash
nexus doctor --json
```

No Docker, probes de Desktop e runtimes nativos podem aparecer como N/A.

## Próximo passo

[Primeiro projeto](first-project.md)


## Homologação do candidato beta.3

Enquanto o GitHub Actions estiver indisponível por créditos, execute a
homologação automatizável localmente:

```bash
./scripts/homologate-beta.sh
```

No Windows:

```powershell
.\scripts\homologate-beta.ps1
```

A evidência é gravada por SHA em
`DEV/validation/local-beta3/<sha>/<plataforma>/`. O script também gera a
lista de checks manuais que não podem ser simulados com segurança.
