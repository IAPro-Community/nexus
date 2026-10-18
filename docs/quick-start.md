# Quick Start

## TL;DR

Em 5–10 minutos, em um ambiente de desenvolvimento já preparado, você pode
compilar o candidato, abrir a Web local, adicionar um projeto e iniciar uma AI Session. O exemplo usa o fluxo **Direct**, que não
exige Composer, Flow ou Mission.

> Estado em 18/10/2026: `release/v1.0.0` com source `1.0.0-beta.10`.
> A versão **não equivale a v1 estável, instalador público ou plataforma homologada**.
> Enquanto privado, o clone exige acesso ao GitHub. Veja [release status](release-status.md).
> License: [PolyForm Strict](../LICENSE), uso não comercial sem alterações/redistribuição.
> Confirme providers com `nexus doctor` e `nexus providers status --json`.

## 1. Compile o candidato

Para avaliação por quem já tem acesso ao repositório, use a compilação do source. Não presuma que exista binário público certificado:

```bash
git clone https://github.com/IAPro-Community/nexus.git
cd nexus
bun install --cwd web --frozen-lockfile
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
