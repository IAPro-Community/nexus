# Status público da release do Nexus v1

Atualizado em **25 de setembro de 2026**.

| Campo | Estado |
| --- | --- |
| Programa | estabilização do Nexus 1.0 beta |
| Versão do source candidato | `1.0.0-beta.2` |
| Última tag beta no repositório-fonte | `v1.0.0-beta.1` |
| Releases GitHub no repositório-fonte | nenhuma |
| Releases públicas em `IAPro-Community/nexus-releases` | nenhuma |
| Matriz candidata da v1 | Linux amd64 + Windows amd64 + macOS arm64 |
| Release estável v1 | **ainda não publicada** |
| CI do SHA integrado | indisponível no momento por bloqueio de créditos da conta |
| Veredito atual | **candidato integrado; homologação/certificação pendentes** |

## O que mudou

A linha que estava planejada como **1.1** foi incorporada à estabilização da
**1.0 enquanto ela ainda está em beta**. Isso elimina duas linhas concorrentes
antes da primeira release estável.

A tag `v1.0.0-beta.1` continua sendo um marco histórico Linux-only no
repositório-fonte. Ela **não corresponde a uma GitHub Release publicada**: o
repositório-fonte não possui releases e o repositório público
`IAPro-Community/nexus-releases` ainda não possui releases. A primeira
publicação binária pública continua pendente e não deve ser inferida apenas da
existência da tag.

O candidato `v1.0.0-beta.2` passa a concentrar:

- correções de roteamento Direct/Lead versus Mission;
- promoção interativa só consome o prompt após admissão; promoção heurística bloqueada volta ao Lead sem perder a solicitação, enquanto pedido explícito mostra blockers exatos;
- hardening da boundary de promoção: `GoalContract` congelado no snapshot, falhas pre-run tipadas e reconhecimento explícito de Mission sem falsos positivos como `permission`/`submission`;
- isolamento autônomo corrigido para ser efetivo por Agent: `AUTO` ignora Agents inseguros, `CREATE` usa worktree, `EXISTING` inseguro bloqueia, e override `project` não contorna default `worktree`;
- correção de transporte das teclas de navegação no terminal;
- OpenCode como provider independente em Usage e Agents;
- ExecutionRole / Agent Interop Core já integrado;
- Agent Projection nativa do OpenCode em implementação como requisito da V1;
- targets de execução customizados (Docker/VPN Workspace, Docker Compose, WSL, SSH e wrappers) preservam o provider lógico e usam templates argv-only sem shell;
- acessibilidade por voz e ditado PTY;
- Nexus HUD;
- suporte nativo de Windows amd64;
- suporte nativo de macOS arm64.

## Estado da validação

O push integrado mais recente disparou GitHub Actions, mas o run encerrou sem
jobs por indisponibilidade de créditos da conta. Isso é **falha de
infraestrutura**, não PASS e também não evidência funcional de FAIL.

Enquanto o CI remoto estiver indisponível, a homologação local deve usar o
SHA candidato explicitamente para impedir PASS acidental em outro commit:

```bash
NEXUS_EXPECTED_SHA="$(git rev-parse release/v1.0.0-beta.2)" ./scripts/homologate-beta.sh
```

No Windows:

```powershell
.\scripts\homologate-beta.ps1 -ExpectedSha (git rev-parse release/v1.0.0-beta.2)
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
- regressões direcionadas de Agent Interop/OpenCode Projection, receipts, launcher customizado e API;
- build CLI;
- `nexus version --json` e `nexus doctor --json`;
- OpenCode visível em `providers status`, Usage e seleção de Agent quando instalado;
- Agent Projection OpenCode: materialização no `NEXUS_PROFILE`, ownership fail-closed, receipts duráveis, discovery sem LLM e activation probe explícito (`opencode run --agent`) com confirmação de execução externa; activation comprovada muda apenas a estratégia para `NATIVE_ON_LAUNCH` e **não** promove enforcement para `ENFORCED`;
- custom target smoke: provider selecionável + `{provider}/{profile}/{cwd}/{target_cwd}/{args}` + probe de versão no próprio target; VPN Dev Workspace deve aceitar host path diferente do container path e materializar no shared `target_profile_home` quando explicitamente configurado;
- build Desktop/HUD da plataforma em homologação;
- smoke interativo de setas no Codex;
- smoke Direct/Lead sem promoção indevida para Mission;
- smoke de Mission promotion: heuristic admission/preparation failure → Lead fallback; explicit Mission failure → blocker acionável sem execução Direct;
- smoke de classificação: prompts com `permission`, `submission` ou `transmission` permanecem não explícitos para Mission.

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
- Agent Interop OpenCode certificado no SHA final: discovery + activation + enforcement read-only sem falsa promoção de `BEST_EFFORT` para `ENFORCED`;
- custom execution target certificado no SHA final: comando supervisionado sem shell, provider ausente no host permitido, probe bounded, host→target workspace mapping (incluindo Windows `/workspace`), shared profile home explícita e degradação para overlay/BEST_EFFORT quando a projeção nativa não puder ser comprovada;
- Mission smoke até `COMPLETED_VERIFIED`;
- recovery/restart;
- Stable Certification/soak;
- zero findings BLOCKER/HIGH relevantes.

## Regra

A `v1.0.0-beta.2` pode existir como prerelease depois da homologação beta.
A `v1.0.0` estável continua fail-closed: ausência de CI, assinatura,
notarização ou evidência nativa nunca é convertida em PASS.
