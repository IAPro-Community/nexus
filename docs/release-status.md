# Status público da release do Nexus v1

Atualizado em **23 de setembro de 2026**.

| Campo | Estado |
| --- | --- |
| Programa | fechamento do Nexus 1.0 |
| Versão reportada pelo source | `1.0.0` |
| Última beta publicada registrada | `v1.0.0-beta.1` |
| Escopo estável da v1 | **Linux amd64** |
| Windows/macOS | **deferidos; não suportados/certificados na v1** |
| Release estável v1 | **ainda não publicada** |
| Veredito atual | **candidato em fechamento final** |

## O que esse status significa

A v1 foi reduzida a um escopo de release que pode ser comprovado com a
infraestrutura disponível: Linux amd64. Isso não remove o código Windows/macOS
do source; apenas impede que a documentação ou o pipeline os anunciem como
plataformas suportadas sem evidência nativa.

Uma release estável só pode ser anunciada depois que o mesmo SHA candidato
passar todos os gates Linux/Web/release e a certificação estável.

## Plataforma suportada

A promessa pública da v1 é:

- Linux amd64 CLI/Core;
- Web local;
- Desktop Wails Linux x86_64;
- archive tar.gz;
- DEB;
- install/uninstall assinados;
- update manifest Ed25519.

Windows, macOS e Linux arm64 ficam para uma linha posterior de certificação.

→ [Suporte de plataformas](operations/platform-support.md)

## Gates para stable

O SHA final precisa comprovar:

- formatting, lint, typecheck, unit e build do frontend;
- Go fmt/vet/lint/race e E2E;
- security/govulncheck;
- Browser E2E, Composer, Surface Isolation, Capacity, Axe e visual smoke;
- Desktop Linux nativo;
- GoReleaser snapshot Linux amd64;
- smoke de instalação/remoção DEB;
- cadeia de assinatura/checksum/update;
- provider smoke autenticado;
- Mission smoke até `COMPLETED_VERIFIED`;
- recovery/restart;
- stable certification/soak definida no workflow;
- zero findings BLOCKER/HIGH relevantes.

## Como ler a documentação

- **Disponível**: implementação existe no source.
- **Suportado na v1**: faz parte da matriz Linux amd64 acima.
- **Certificado**: possui evidência do SHA/plataforma correspondente.
- **UNVERIFIED / DEFERRED**: existe código ou preparação, mas não é promessa de release.

## Regra

Não anunciar Windows/macOS como suportados pela Nexus v1. A futura inclusão
dessas plataformas exige nova certificação nativa e ampliação explícita do gate
de release.
