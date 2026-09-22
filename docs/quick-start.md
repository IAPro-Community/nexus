# Quick Start

## TL;DR

Em 5–10 minutos você pode instalar o Nexus, abrir a Web local, adicionar um
projeto e iniciar uma AI Session. O exemplo usa o fluxo **Direct**, que não
exige Composer, Flow ou Mission.

> Estado: **Disponível** para a CLI e Web locais. A evidência de Desktop e de
> providers autenticados varia por plataforma e perfil; confirme com
> `nexus doctor` e `nexus providers status --json`.
>
> Os comandos abaixo pinam `v1.0.0-beta.1` porque essa é a tag beta publicada
> disponível. A versão de fonte `1.0.0-beta.4` está em fechamento e não deve ser
> tratada como release publicada até a certificação e criação da tag correspondente.

## 1. Instale

Para usar um release no Linux/macOS:

```bash
curl -fsSL https://raw.githubusercontent.com/IAPro-Community/nexus/main/install.sh | bash -s -- --version=v1.0.0-beta.1
nexus doctor
```

No Windows PowerShell:

```powershell
irm https://raw.githubusercontent.com/IAPro-Community/nexus/main/install.ps1 -OutFile install.ps1
.\install.ps1 -Version v1.0.0-beta.1
nexus doctor
```

O instalador busca um release e verifica integridade. Para um release
reproduzível, troque `latest` por uma tag explícita, por exemplo `vX.Y.Z`.
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
