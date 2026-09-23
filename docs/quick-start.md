# Quick Start

## TL;DR

Em 5–10 minutos você pode instalar o Nexus, abrir a Web local, adicionar um
projeto e iniciar uma AI Session. O exemplo usa o fluxo **Direct**, que não
exige Composer, Flow ou Mission.

> Estado: **Disponível** para a CLI/Web e Desktop Linux x86_64 no escopo v1.
> Windows/macOS estão deferidos e não são targets suportados desta release.
> Confirme providers locais com `nexus doctor` e `nexus providers status --json`.
>
> Os comandos abaixo pinam `v1.0.0-beta.1` porque essa é a tag beta publicada
> disponível. O source candidato final reporta `1.0.0`, mas não deve ser tratado
> como release publicada até a certificação estável e criação da tag correspondente.

## 1. Instale

O target suportado da v1 é **Linux amd64**:

```bash
curl -fsSL https://github.com/IAPro-Community/nexus-releases/releases/download/v1.0.0-beta.1/install.sh -o install.sh
bash install.sh --version=v1.0.0-beta.1
nexus doctor
```

O installer vem do repositório público de binários e o payload é aceito somente
após manifesto Ed25519 válido, SHA-256 correspondente e versão interna coerente.
Enquanto a linha pública for beta, use a tag beta explícita; `latest` representa
o latest não-prerelease do GitHub.

Windows/macOS não fazem parte da matriz suportada da v1.

Veja os pré-requisitos e opções em [instalação nativa](getting-started/installation.md).

## 2. Abra a workstation local

```bash
nexus web
```

Abra a URL de bootstrap que o comando mostrar. Ela é local e de curta duração;
não a compartilhe. A Web usa loopback por padrão.

## 3. Registre um projeto e abra uma sessão

Na Web:

1. Abra **Projects** e adicione uma pasta de projeto.
2. Selecione **+ New → AI Session**.
3. Escolha um provider que apareça disponível localmente.
4. Trabalhe no terminal da sessão.

Alternativa de terminal:

```bash
cd /caminho/do/projeto
nexus projects add .
nexus codex
```

Substitua `codex` por um provider instalado. Não trate o exemplo como uma
garantia de autenticação: use `nexus providers status --json` para confirmar o
seu ambiente.

## 4. Confirme que funcionou

```bash
nexus ps --json
nexus status
```

Você deve ver o runtime ou o perfil que acabou de usar. Se não aparecer,
consulte [troubleshooting](operations/troubleshooting.md).

## Próximos passos

- [Descobrir funcionalidades](features/index.md)
- [Entender como o Nexus funciona ponta a ponta](product/how-it-works.md)
- [Consultar o modelo mental](product/mental-model.md)
- [Configurar providers e quota](product/providers-and-usage.md)
- [Referência da CLI](cli/README.md)
