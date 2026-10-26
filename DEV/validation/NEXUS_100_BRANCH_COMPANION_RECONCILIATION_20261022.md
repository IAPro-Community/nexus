# Nexus 1.0.0 — auditoria de convergência e Companion (22/10/2026)

> **Decisão vigente, 2026-10-26:** o produto deve ter um único mascote NEX. Não desenvolver, integrar nem reabrir o modo/personagem Companion 2D. A PR #55 está `CLOSED` (confirmado no GitHub em 26/10; saída em `DEV/validation/linux-100-2026-10-25/logs/current-pr55-product-scope-2026-10-26.json`). O conteúdo abaixo é a fotografia histórica de 22/10; qualquer sugestão de conciliar ou homologar o Companion 2D foi superada por esta decisão.

> Status: **PARCIAL / NO-GO**. Auditoria documental e comparação GitHub, não certificação nativa. Não declarar 1.0 estável, nem mover tags.
> Branch canônica: `release/v1.0.0`. Referência de início: `9ff188967eb72b963bad91c4ff0c108ffdbc867e`. HEAD remoto mudou durante a auditoria; o commit que adiciona este relatório deve ser consultado como o novo HEAD, e qualquer validação futura precisa usar seu SHA exato.

## A. Escopo, método e limitações

- Consultados GitHub REST/API/connector, ref/HEAD, todas as **25 branches retornadas** (release + 24 alternativas), PRs da coleção `pulls?state=all`, comparações `base...head`, diferenças e arquivos dos PRs #54, #55, #57, #58, #59, #61, #62, #49, #52 e #46.
- Branch sem commit em comum ou com commits exclusivos **não** é prova automática de funcionalidade ausente; em cada decisão abaixo conferir os arquivos e implementações atuais. Os contadores são fotografias da auditoria e mudam com push concorrente.
- **BLOCKED**: não foi possível clonar/buildar o repositório no executor desta sessão (sem resolução de github.com no shell). Nenhum `go test`, `go vet`, `bun quality:full`, Playwright, E2E visual ou smoke nativo do **novo SHA** foi executado aqui. Evidências anteriores são históricas.

## B. Inventário das branches (fotografia comparada à 1.0 em 9ff1889)

| Branch ou grupo | Delta observado | Classificação |
| --- | --- | --- |
| `main` | 0 commits exclusivos / 1381 atrás | absorvida historicamente; sem merge |
| `fix/beta7-windows-KIVER-20261007`, `fix/issue-53-windows-agent-startup`, `fix/release-beta10-windows-closure-20261014` | 0 exclusivos | presentes por ancestralidade |
| `integration/v1.0.0-css-and-themes-20261009`, `refactor/v1.0.0-workspace-css-modules-20261009` | 0 exclusivos | presentes por ancestralidade |
| `fix/beta8-manual-opaque-quota` | 1 exclusivo, base muito defasada | verificar equivalência; PR #52 documenta port pontual direto |
| `fix/v1.0.0-gap-closure-20260930` | 38 exclusivos / 1013 atrás | **não** mergear: PR #62 confirmou implementação histórica incompleta; portar apenas deltas comprovados |
| `fix/v1.0.0-runtime-failure-evidence-20261022` | 5 exclusivos / 15 atrás na fotografia | PR #62 **mergeada** na release; equivalência deve ser checada sobre HEAD final |
| `fix/windows-beta10-homologation-20261008` | 175 exclusivos / 811 atrás | contexto histórico e evidências; PRs #45/#46 reconciliadas via #47 |
| `feat/hud-detached-popover` | 23 exclusivos / 236 atrás | PR #56 mergeada; release tem implementação **mais recente**; não sobrescrever |
| `fix/hud-show-restore-v2` | 2 exclusivos / 141 atrás | PR #59 aberta; aproveitar correções pontuais preservando auto-hide e popup |
| `feat/nexus-companion-v1.1` | 36 exclusivos / 248 atrás | PR #55, **v1.1 Draft**; 2D não integrado em 1.0 |
| `feat/orca-worktree-visibility-v1` | 11 exclusivos / 142 atrás | PR #57, Draft e sem gates completos |
| `feat/v1-contextual-workspace-experience` | 36 exclusivos / 61 atrás | PR #58, Draft e sem gates completos |
| `docs/nexus-1.0-portal-20261022` | 4 exclusivos / 126 atrás | PR #61, Draft; build Docusaurus / npm lock / Pages pendentes |
| `release/v1.1.0` | 252 exclusivos / 146 atrás | branch de versão futura; não usar como fonte de merge indiscriminado |
| 7 `wsl-backup/release-v1-0-0/2026-10-21-*` | cada uma 1 commit exclusivo, numerosas diferenças históricas e artefatos temporários | snapshots/backups, **não** mergear; não elevar logs e binários `.tmp` à distribuição |

## C. PRs e features relevantes

| Recurso | Origem | Release 1.0 | Funciona? | Decisão |
| --- | --- | --- | --- | --- |
| Guardian original SVG/CSS | `GuardianMascot.tsx` | **Sim** | parcialmente comprovado em evidência Windows anterior; não neste SHA | preservar |
| Nexus Companion 2D 36 sprites, balão, capsule, atenção | PR #55 | **Não** (arquivos `NexusGuardian.tsx` e `web/public/companion/*` dão 404 na release) | **NÃO HOMOLOGADO** | não fazer merge v1.1 antigo por cima do HUD atual; reconciliar a arquitetura e homologar |
| HUD / popover desacoplado | PR #56 + alterações novas | **Sim** | gates e click-through Windows em SHAs anteriores; atual pendente | preservar |
| Inicialização HUD linkado Windows sem `SW_HIDE` | PR #59; port concorrente encontrado no HEAD | **Sim** no código recente | visual exato SHA não comprovado | preservar `CommandNoConsole` |
| Restauração de geometria no primeiro `WindowShow` | PR #59 | **Não** na fotografia inspecionada | teste da branch PR PASS, release pendente | portar somente método/evento + teste |
| Worktrees Git somente leitura | PR #57 | **Não** nos arquivos da PR | não executado na release | Draft; certificar antes de 1.0 |
| Split persistente, command palette, Direct recomendado | PR #58 | **Não** nos arquivos da PR | não executado na release | Draft; validar PTY, keyboard, E2E e quotas UNKNOWN |
| Codex/OpenCode Windows | PR #54 | **Sim**, PR mergeada | testes na PR/SHAs anteriores; regressão do HEAD pendente | preservar |
| Temas Dusk / Ember / Slate/Paper/Clay Flat | PR #49 (fechada sem merge) | **Sim**, IDs confirmados em `themePresets.ts` na release | visual/cross-theme atual pendente | sem duplicação |
| Seleção manual em quota opaca | PR #52 (fechada sem merge) | **Port declarada na PR** | testes novos pendentes | conferir equivalência funcional; não merge histórico |
| Reconciliation beta10 Windows/engenharia | PRs #45/#46 -> #47 | **Integrada por #47** | gates de release continuam pendentes | preservar |
| Typed runtime failures | PR #62 | **Mergeada em release** | exato SHA pendente | preservar |
| Portal Docusaurus | PR #61 | **Não**, Draft | npm build/Pages pendentes | não anunciar URL/portal publicado |

**Outras PRs fechadas sem merge:** #24, #26, #28, #29, #31, #33, #34, #35, #36, #38, #39 e as históricas acima devem continuar classificadas por equivalência sem cherry-pick automático. #38/#39 precederam delegação seletiva #40; não presumir equivalência integral sem testes. Composer e Flow **não são recursos homologados para 1.0**. O inventário público de 36 features (`docs/reference/feature-matrix.md`) é catálogo, não certificação.

## D. Causa da ausência visual do Companion

1. **Confirmado — ausência de código e assets do Companion 2D #55** na 1.0; a PR permanece em `release/v1.1.0`, Draft, 36 commits próprios e 248 commits atrás quando comparada à release.
2. **Confirmado — existe outro mascote 2D original**, `GuardianMascot.tsx` SVG com animação CSS, seleção `BAR/MASCOT/TRAY` e `HUDAppearanceMascot` no backend. Não confundir esse Guardian com o atlas de 36 quadros do #55.
3. **Confirmado — o HUD Web tem `defaultHudPreferences.enabled: false`**, dependente da opção experimental; o HUD nativo usa `HUDWindowState.Appearance` próprio. Não inferir que ambos têm o mesmo default.
4. **Risco real Windows — janela do HUD linkado pode ser invisível** quando iniciada com `SW_HIDE`; correção `CommandNoConsole` apareceu no HEAD remoto durante esta auditoria, mas não há smoke nativo do HEAD final.
5. **Bloqueio de integração #55:** sua branch tem `hud_window.go` e `DesktopHudApp.tsx` antigos, sem as evoluções mais recentes de popover separado, hit-testing e restore. Merge integral regrediria a experiência. A PR documenta hit-testing retangular dos balões, roteamento Desktop/Web e validação nativa pendentes.

**Contrato para concluir o Companion:** rebase lógico seletivo (não sobrescrita) da projeção/Atlas/render, incorporar ao HUD atual com modo opcional, herdar `useHudHitRegions`, detached popover, preferências atuais, posicionamento, transparência e `prefers-reduced-motion`; garantir que interação transparente não bloqueie cliques; testar Windows macOS Linux X11/Wayland; registrar screenshots reais no SHA final. Até isso, **BLOCKED / não homologado**.

## E. Regressão Windows: port seletivo da PR #59

- Conservar `ServiceStartup` e a condição `AutoHide && !GetHUDExpanded()`: o patch literal da PR #59 retirava este guard, o que pode esconder a cápsula ao trocar foco com popup.
- Usar `sync.OnceFunc` ao receber `events.Common.WindowShow` e mover corpo de `RestoreHUDWindow` a helper; não reaplicar estado salvo em futuros shows.
- Manter `CommandNoConsole` para `hudlink.Spawn`, com `CREATE_NO_WINDOW` sem `SW_HIDE`. Os arquivos desta metade mudaram **concorrentemente** durante a auditoria e foram encontrados no branch remoto — não aplicar nova cópia.
- Adicionar regressão para `WindowShow` e conferir o comportamento com `Appearance=BAR`, sem alterar defaults do Mascot existente.
- Estas mudanças são **integradas em source apenas quando o commit deste relatório confirmar os arquivos**; nunca declarar PASS nativo sem smoke.

## F. Homologação e distribuição

- Fonte: `VERSION` e `web/package.json` continuam em **1.0.0-beta.10**.
- `go.mod` declara Go 1.25.0; validar build tags GTK3 no Linux conforme o CI, e matriz Windows/Linux/macOS Intel/Apple Silicon.
- No HEAD inicial `9ff1889`, GitHub Actions: **CI cancelled** e **Unsigned Source Release Preview skipped**. Nenhum status de commit em `/status`. Não usar isso como PASS.
- Releases públicas verificadas: última preview não assinada consultada `preview-f23ed9b88725e9bc4df80ded19a89b2e7f8a326a`, com pacotes Linux/Windows/macOS Intel/ARM64 e `SHA256SUMS.txt`. **É SHA anterior** ao HEAD e não inclui a PR #55.
- `docs/release-status.md` e `DEV/validation/current/RELEASE_STATUS.md` indicam **NO-GO**; testes automatizados de mascot toggle e click-through constam em SHAs históricos, não certificam esta conciliação.
- Atualização via GitHub Releases, preservação de perfis/sessões, checksums internos, assinatura Windows Authenticode e notarização macOS: **não homologados neste SHA**. Ausência de certificado não bloqueia preview unsigned, mas impede declaração de pacote assinado.
- Não publicar tag estável, não tocar no repositório antigo `nexus-releases`; nenhum workflow acionado deliberadamente.
- **Contradição de licenciamento a resolver:** repo público não é sinônimo de software open source. PR #61 declara licença PolyForm Strict; confirmar licença efetiva antes de material publicitário anunciar open source irrestrito.

## G. Veredito e próximos gates

**NO-GO PARA RELEASE 1.0 ESTÁVEL.** O diagnóstico do Companion está concluído; implementação 2D #55 ainda está em branch v1.1 e requer conciliação específica. HUD nativo deve ser testado visualmente após a correção de `WindowShow`. Worktrees e split permanecem Draft por decisão de risco até seus gates. Nenhuma das três classes deve ser confundida: `source integrado` ≠ `build compilado` ≠ `homologação nativa`.

Gates exigidos no SHA do último commit: `gofmt -l`, `go test -count=1 ./...`, `go vet ./...`, `go test -race -count=1 ./...`, `bun install --cwd web --frozen-lockfile`, `bun run --cwd web quality:full`, Playwright HUD/Desktop/Workspaces, smokes PTY/ConPTY e build/matriz nativa, update migration/rollback e certificação com evidência. Qualquer `NOT_RUN` permanece **BLOCKED**, nunca PASS.
