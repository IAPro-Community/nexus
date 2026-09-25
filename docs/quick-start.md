# Quick Start

## TL;DR

Em 5–10 minutos você pode instalar o Nexus, abrir a Web local, adicionar um
projeto e iniciar uma AI Session. O exemplo usa o fluxo **Direct**, que não
exige Composer, Flow ou Mission.

> Estado: o source candidato é **`1.0.0-beta.2`** e integra CLI/Web,
> Desktop/HUD e os targets Linux amd64, Windows amd64 e macOS arm64. Isso não
> equivale a release publicada/certificada.
>
> Ainda não existe GitHub Release pública do Nexus. A tag histórica
> `v1.0.0-beta.1` no repositório-fonte não contém uma distribuição pública
> assinada em `IAPro-Community/nexus-releases`.
>
> Confirme providers locais com `nexus doctor` e
> `nexus providers status --json`.

## 1. Compile o candidato

Enquanto a primeira release binária pública não for certificada, o caminho
válido para avaliação de engenharia é compilar o source:

```bash
git clone https://github.com/IAPro-Community/nexus.git
cd nexus
bun --cwd web install --frozen-lockfile
make build
./nexus doctor
```

A instalação binária assinada passa a ser recomendada somente quando
[status de release](release-status.md) registrar uma GitHub Release real.

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
