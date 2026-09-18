# ADR: migrar o module path para IAPro-Community/nexus

- Status: aceito
- Data: 2026-09-13

## Contexto

O repositório Nexus foi transferido para a organização `IAPro-Community`, mas
o módulo Go ainda declarava `github.com/kivervinicius/ai-cli`. Isso fazia os
binários, metadados de build e documentação apontarem para a origem histórica.

## Decisão

Adotar `github.com/IAPro-Community/nexus` como module path canônico e atualizar
imports internos, ldflags, Makefile, GoReleaser e documentação no mesmo slice.

## Consequências

- O caminho de importação Go muda antes da versão 1.0; a quebra é aceita porque
  não há consumidores Go externos suportados neste estágio.
- URLs e metadados de release passam a apontar para a organização nova.
- Perfis, CLI, flags, JSON público e runtime não sofrem alteração de contrato.
- Consumidores externos que dependam do caminho antigo devem migrar para o novo
  módulo; nenhum redirect de compatibilidade é criado.

## Verificação

`go mod tidy`, `go test ./...`, `make quality` e `git diff --check` devem passar;
`rg` não deve encontrar referências ao module path antigo em código ou artefatos
de build.
