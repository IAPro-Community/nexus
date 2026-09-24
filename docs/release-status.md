# Status público da release do Nexus v1

Atualizado em **24 de setembro de 2026**.

| Campo | Estado |
| --- | --- |
| Programa | estabilização do Nexus 1.0 beta |
| Versão do source candidato | `1.0.0-beta.2` |
| Última beta publicada registrada | `v1.0.0-beta.1` |
| Matriz candidata da v1 | Linux amd64 + Windows amd64 + macOS arm64 |
| Release estável v1 | **ainda não publicada** |
| CI do SHA integrado | indisponível no momento por bloqueio de créditos da conta |
| Veredito atual | **candidato integrado; homologação/certificação pendentes** |

## O que mudou

A linha que estava planejada como **1.1** foi incorporada à estabilização da
**1.0 enquanto ela ainda está em beta**. Isso elimina duas linhas concorrentes
antes da primeira release estável.

A `v1.0.0-beta.1` publicada continua sendo evidência histórica Linux-only.
Ela não é alterada retroativamente.

O candidato `v1.0.0-beta.2` passa a concentrar:

- correções de roteamento Direct/Lead versus Mission;
- correção de transporte das teclas de navegação no terminal;
- OpenCode como provider independente em Usage e Agents;
- ExecutionRole / Agent Interop já integrado;
- acessibilidade por voz e ditado PTY;
- Nexus HUD;
- suporte nativo de Windows amd64;
- suporte nativo de macOS arm64.

## Estado da validação

O push integrado mais recente disparou GitHub Actions, mas o run encerrou sem
jobs por indisponibilidade de créditos da conta. Isso é **falha de
infraestrutura**, não PASS e também não evidência funcional de FAIL.

Enquanto o CI remoto estiver indisponível, a homologação local deve usar:

```bash
./scripts/homologate-beta.sh
```

No Windows:

```powershell
.\scripts\homologate-beta.ps1
```

Os scripts registram evidência por SHA em
`DEV/validation/local-beta2/<sha>/<plataforma>/` e executam os gates
automatizáveis localmente. Os checks interativos de terminal, OpenCode, HUD e
voz continuam explicitamente listados como manuais.

## Matriz candidata

| Plataforma | Estado no source | Estado de certificação |
| --- | --- | --- |
| Linux amd64 | integrado | homologação beta pendente |
| Windows amd64 | integrado | certificação nativa exact-SHA pendente |
| macOS arm64 | integrado | certificação nativa exact-SHA pendente |
| Web | integrado | homologação beta pendente |

Implementação não equivale a suporte certificado. A documentação pública só
deve promover a matriz acima após evidência correspondente do mesmo SHA.

## Gates para `v1.0.0-beta.2`

Antes de criar/publicar a tag:

- documentação;
- frontend format/lint/styles/typecheck/unit/build;
- Go fmt/vet/test/race;
- regressões específicas de terminal e roteamento Mission;
- control/runtime E2E;
- build CLI;
- `nexus version --json` e `nexus doctor --json`;
- OpenCode visível em `providers status`, Usage e seleção de Agent quando instalado;
- build Desktop/HUD da plataforma em homologação;
- smoke interativo de setas no Codex;
- smoke Direct/Lead sem promoção indevida para Mission.

Para publicar artefatos Windows/macOS, continuam obrigatórios os gates nativos
de assinatura, instalador e lifecycle.

## Gates para `v1.0.0` estável

Além da beta homologada, o SHA final precisa comprovar:

- CI exact-SHA completo;
- security/govulncheck;
- Browser E2E, Axe e visual smoke;
- Desktop/HUD;
- Windows Authenticode + NSIS;
- macOS Developer ID + notarização + stapling + Gatekeeper;
- lifecycle install/version/doctor/uninstall;
- update manifest Ed25519;
- native certification seal;
- provider smoke autenticado;
- Mission smoke até `COMPLETED_VERIFIED`;
- recovery/restart;
- Stable Certification/soak;
- zero findings BLOCKER/HIGH relevantes.

## Regra

A `v1.0.0-beta.2` pode existir como prerelease depois da homologação beta.
A `v1.0.0` estável continua fail-closed: ausência de CI, assinatura,
notarização ou evidência nativa nunca é convertida em PASS.
