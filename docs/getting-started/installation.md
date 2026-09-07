# Instalação

## Compilar do fonte

Requisitos verificados no projeto: Go 1.25+ e Bun 1.3.9+.

```bash
git clone https://github.com/kivervinicius/ai-cli.git
cd ai-cli
bun --cwd web install --frozen-lockfile
make build
./nexus version
./nexus doctor
```

O build produz `./nexus` no checkout. `make install-local` é a etapa separada para instalar o binário no diretório local configurado.

## Executar a Web

```bash
./nexus web --no-open
```

Abra a URL `Bootstrap` exibida no terminal. O servidor escuta em loopback por padrão. Não cole tokens de bootstrap, cookies ou credenciais em issues, screenshots ou logs compartilhados.

## Binários publicados

Use uma release publicada somente quando ela existir e confira o `checksums.txt` correspondente. O fluxo atual usa checksum; não o trate como cadeia de suprimentos assinada enquanto a chave pública e a assinatura do manifesto não estiverem publicadas.

## Diagnóstico

```bash
./nexus doctor --json
```
