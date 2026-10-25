# Homologação Linux — Nexus 1.0.0

**Veredito: `NO_GO_LINUX`**  
**Gate determinante:** `NO-GO — COMPANION_2D_NOT_INTEGRATED`  
**Data:** 2026-10-23  
**Branch:** `release/v1.0.0`  
**HEAD remoto consultado:** `bf6adf0c80ac3531d7291844512a14a017ab0002`  
**Versão compilada:** `1.0.0-beta.10` — a branch ainda não declara uma release estável `1.0.0`.

## Proveniência e instalação

O fetch confirmou que o checkout estava no HEAD remoto acima. O build foi feito com Go 1.26.10 sobre esse SHA **mais um worktree já dirty**, preservado sem reset, commit ou push. A revisão local do `Makefile` para fixar o scanner de segurança em Go 1.26.10 também entrou no build. Portanto, o SHA do Git identifica a base, enquanto os hashes abaixo identificam os binários efetivamente instalados.

Para identificar as alterações de entrada do build: o hash do diff das fontes rastreadas é `dc565b4d87f498f69bc0016fedb8a26e94aa0d0c50aee3b2348d158e3ce839c5`; os arquivos de build não rastreados têm hashes individuais em [source-provenance-go1.26.10-bf6adf0c.txt](logs/source-provenance-go1.26.10-bf6adf0c.txt).

`make build build-desktop build-hud` passou. CLI, Desktop Wails e HUD foram substituídos atomicamente em:

- `/home/desenvolvedor/.local/bin`
- `/home/desenvolvedor/.local/share/ai-manager/profiles/codex/kivergmail/home/.local/bin`

Os dois conjuntos e os artefatos do build têm hashes iguais:

- CLI: `9ecd3ff22cd1d6b099d1963424dc7c8c55302295cef54d5a24b7adc715f9d815`
- Desktop: `ca3076dd82d39a285c547ad833c59a43156c2a831bd5461738498b2d63a056c7`
- HUD: `89b2b2f79ae0254d3ef23761f07ac9ee8b60f5325d4aeb437b80cc68f6706b52`

`nexus version --json` retorna Linux `amd64`, `1.0.0-beta.10`, Go `go1.26.10`, commit `bf6adf0c`. Os ELF são x86-64 dinâmicos; `ldd` não apontou dependências ausentes. Um backup pré-instalação foi mantido em `/home/desenvolvedor/.local/state/ai-manager/nexus-update-backups/20261023T203808Z-go1.26.10`.

Depois da instalação, os quatro hosts de agentes, o Desktop, o HUD ligado e o serviço Web foram encerrados e iniciados novamente. O SHA dos executáveis carregados em cada processo coincide com o artefato instalado. O Web local (`127.0.0.1:44065`) e o Core Desktop (`127.0.0.1:43253`) retornaram `/api/v1/health` HTTP 200 com `1.0.0-beta.10`. Consulte [processes-final-go1.26.10-bf6adf0c.log](logs/processes-final-go1.26.10-bf6adf0c.log) e [runtimes-final-go1.26.10-bf6adf0c.tsv](logs/runtimes-final-go1.26.10-bf6adf0c.tsv).

## Ambiente

Ubuntu 25.10 AMD64, kernel `6.17.0-23-generic`, GNOME em Wayland; XWayland estava disponível, mas não havia sessão X11 independente. A GPU VMware não tinha aceleração 3D. O Desktop e o HUD iniciaram como aplicações Wails nativas. Detalhes e bibliotecas: [ENVIRONMENT.md](ENVIRONMENT.md).

## Resultados

| PASS | FAIL | BLOCKED | NOT_RUN | Total |
|---:|---:|---:|---:|---:|
| 30 | 7 | 6 | 12 | 55 |

O gate `make quality-full` passou com Go 1.26.10; `make vet`, o scanner de vulnerabilidades e o alvo de testes E2E com race detector passaram. Frontend: 99 arquivos e 612 testes aprovados, além de formatação, lint, typecheck e build. Os E2E de hardening/responsividade, acessibilidade, visual, capacidade e HUD passaram. A matriz detalhada está em [TEST_MATRIX.md](TEST_MATRIX.md), com logs em `logs/` e capturas/checksums em `screenshots/` e [screenshots.sha256](logs/screenshots.sha256).

Os E2E de Composer workspace, isolamento de superfície, execution spine e vague execution spine falharam em seletores/contratos de UI não encontrados na interface servida. O E2E de temas falhou nos presets Ember Studio e Nexus Dusk pelo seletor `.nx-composer-surface`; os 18 presets restantes ficaram `BLOCKED` quando a repetição do mesmo timeout foi encerrada. Os diagnósticos e capturas reais estão nos logs `browser-e2e-*.log` e em `screenshots/e2e/`.

## Companion 2D e HUD

O código e o aplicativo renderizam `GuardianMascot`/`guardianModel`, o Guardian anterior. A branch não contém nem conecta `NexusGuardian.tsx`, `companionModel` ou o atlas e os estados contextuais exigidos pelo Companion 2D da PR #55. A PR segue aberta em draft e não está no SHA testado. A captura nativa do Guardian não foi aceita como evidência do Companion. Resultado obrigatório: **`NO-GO — COMPANION_2D_NOT_INTEGRATED`**. Referências: [COMPANION_VALIDATION.md](COMPANION_VALIDATION.md), [auditoria dos fontes](logs/companion-source-audit.txt) e [PR #55](https://github.com/IAPro-Community/nexus/pull/55).

O HUD antigo foi observado com opções Barra/Mascote/Bandeja; o binário final do HUD também iniciou e serviu health 200. A interação do popover separado e click-through no Wayland não foram homologados. O veredito do HUD e sua cobertura estão em [HUD_VALIDATION.md](HUD_VALIDATION.md).

## Distribuição

O artefato instalado é beta.10, não uma release estável. DEB, pacote GoReleaser, manifesto GitHub Releases, assinatura/checksum oficial, atualização A→B e rollback ficaram sem homologação. O harness oficial exige checkout limpo e executa `--clean`; `dist/` já continha artefatos antigos preservados e o worktree está dirty, portanto não foi executado sobre a distribuição local. Detalhes: [DISTRIBUTION_VALIDATION.md](DISTRIBUTION_VALIDATION.md).

## Correção feita durante a validação

O fallback do alvo `security` no `Makefile` usava `GOTOOLCHAIN=auto`, que escolheu Go 1.25.14 e reportou vulnerabilidades da biblioteca padrão. O projeto e seus workflows fixam Go 1.26.10. A menor correção local fixou esse fallback em `GOTOOLCHAIN=go1.26.10`; `make security` e o `make quality-full` final foram repetidos com sucesso. A alteração permanece sem commit. As correções locais anteriores do comando `nexus login codex`, importação segura de sessão e layout também permanecem no worktree; veja [FIXES_APPLIED.md](FIXES_APPLIED.md).

## Bugs e impedimentos em aberto

- **P1 — obrigatório:** Companion 2D da PR #55 ausente da release.
- **P1 — distribuição:** pacote Linux oficial e fluxo de atualização de uma GitHub Release estável não testados.
- **P2 — E2E:** quatro cenários Composer/superfície falham; dois presets de tema já falham pelo mesmo seletor ausente.
- **P2 — interface:** após importação temporária Codex, o perfil ficou autenticado e o arquivo foi gravado com modo `0600`, mas a janela não apresentou confirmação visível.
- **BLOCKED:** interação nativa do HUD Wayland, clique transparente, sessão X11 independente e chamadas externas reais de provedores.
- **NOT_RUN:** matriz nativa de todos os temas, redimensionamento/teclado/monitores, modos e persistência completos do HUD, rotas e failover reais de agentes, skills, terminais longos, resiliência, redação de logs e teste de repouso.

## Reprodução

Na raiz do repositório, com dependências GTK3/WebKitGTK já presentes:

```bash
git fetch origin --prune
git rev-parse origin/release/v1.0.0
GOTOOLCHAIN=go1.26.10 make quality-full
GOTOOLCHAIN=go1.26.10 make vet test-e2e
GOTOOLCHAIN=go1.26.10 make build build-desktop build-hud
nexus version --json
curl -i http://127.0.0.1:44065/api/v1/health
curl -i http://127.0.0.1:43253/api/v1/health
```

O SHA precisa permanecer `bf6adf0c80ac3531d7291844512a14a017ab0002` para reproduzir esta campanha; para reproduzir os binários, aplique também o worktree local documentado e compare os três SHA-256 registrados. Nenhuma tag ou release estável foi criada.
