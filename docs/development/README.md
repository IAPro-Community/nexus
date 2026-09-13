# Documentação para desenvolvimento

## Ambiente local

O build a partir do fonte requer Go 1.25+ e Bun 1.3.9+:

```bash
git clone https://github.com/kivervinicius/ai-cli.git
cd ai-cli
bun --cwd web install --frozen-lockfile
make build
./nexus doctor
```

Comandos de qualidade verificados no repositório:

```bash
make quality
make web-verify
go test ./... -count=1
```

Consulte [BUILD.md](../../BUILD.md) para o contrato de versão e release,
[arquitetura](../architecture/overview.md) para módulos e limites, e
[CONTRIBUTING.md](../../CONTRIBUTING.md) antes de enviar mudanças.

## Mapa de código

| Área | Diretório principal |
| --- | --- |
| CLI e entrypoints | `cmd/`, `internal/app/` |
| Core de projetos, agents, planos e Mission | `internal/nexus/` |
| Web e API local | `internal/control/web/`, `web/src/` |
| providers, perfis e quota | `internal/core/`, `internal/profile/` |
| Desktop | `cmd/nexus-desktop/`, `internal/desktop/` |
| testes e scripts | `internal/**/*_test.go`, `scripts/`, `web/scripts/` |

## Documentação como código

Execute `node scripts/docs-verify.mjs` após alterar links, páginas públicas ou
evidência visual. Para atualizar imagens sintéticas, use
`bun run --cwd web docs:capture`; veja [assets visuais](../assets/README.md).
