# Follow-up Linux — PRs #57 e #58

Data: 2026-10-25  
Branch local: `release/v1.0.0`  
HEAD base local/remoto: `bf6adf0c80ac3531d7291844512a14a017ab0002`  
Fingerprint das fontes de build: `6370093f2237d6b14d8ad2f23b5444466ef6556d7ee19f78ce1830bd114ba05e` (1.398 arquivos; manifesto em `logs/pr578-current-source-manifest.txt`).  
Estado: overlay local modificado, sem commit; não publicado nem mesclado no GitHub.

## Decisão de escopo do mascote

O produto mantém **um único mascote NEX**. A PR #55 foi fechada após a decisão do produto de não desenvolver personagem nem modo Companion 2D. O código desta integração não adiciona outro renderer, sprite ou modo de personagem. `MASCOT` continua sendo uma apresentação do único NEX no HUD; Barra e Bandeja são outras apresentações do HUD.

## PRs abertas verificadas

| PR | Escopo/estado remoto | Tratamento nesta execução |
|---|---|---|
| #55 | Companion 2D; fechada em 2026-10-25 | Excluída pelo contrato de mascote único. |
| #57 | Visibilidade somente leitura de Git worktrees; Draft, `UNSTABLE` | Alterações integradas no overlay local; Go/Vitest passaram. Sem operação de criação, remoção, prune, checkout ou merge. |
| #58 | Terminais divididos, busca contextual e Direct guiado pelo ResourceScheduler; Draft, `UNSTABLE` | Alterações integradas no overlay local; quality completo, geometria Chromium e build nativo passaram. Corrigida a sobreposição de rótulos PTY e validada visualmente em Wails. Digitação/comandos em ambos os PTYs ainda pendentes. |
| #59 | Correções de restauração HUD/spawn; aberta, `DIRTY` | Não mesclada em bloco: conflita com seis arquivos nativos e o overlay atual já contém `restoreHUDWindowWhenShown` e `CommandNoConsole`, cobertos por testes. |
| #61 | Portal Docusaurus; Draft, `UNSTABLE` | Não substituída: a árvore local já tem um portal expandido; a proposta descreve um catálogo antigo de 48 páginas. A busca do portal local foi corrigida e validada separadamente em [PR61_DOCS_SITE_VALIDATION.md](PR61_DOCS_SITE_VALIDATION.md). |
| #64 | Correção de quality gates; destino `main`, `BLOCKED` | Fora do destino `release/v1.0.0`. |

As PRs #57 e #58 permanecem Draft no GitHub. O overlay local não foi commitado nem enviado, conforme a regra do repositório contra commit/push automáticos. A PR #59 não foi mesclada indiscriminadamente, e nenhuma tag ou release foi publicada.

## Reteste nativo de runtime e anexação — FIX-011

O botão “Anexar Terminal” em runtimes shell sem agente criava um novo shell. FIX-011 separa a criação da sessão da anexação: `ProjectOverviewSurface` exige `onAttachRuntime`, e `WorkspaceSurfaceHost` abre o `runtime_id` existente como `project-shell`, depois ativa Terminais. Dois testes unitários cobrem o roteamento de agente e shell.

`bun run quality:full` passou com 106 arquivos/664 testes. O E2E de terminais passou nas geometrias divididas e larguras 520 px/360 px. No Desktop Wails AMD64 atual em XWayland, anexar um shell iniciado pelo mesmo binário manteve 4 runtimes/4 PTYs e mostrou o prompt existente. Não houve quinto runtime.

Ao substituir o executável no mesmo caminho, runtimes lançados por uma imagem de binário anterior foram rejeitados pela checagem de fingerprint e marcados `STALE`; a interface mostrou `TERMINAL_TRANSPORT_ERROR`. Este caminho cross-image segue BLOCKED para validação de atualização/distribuição. Não foi enfraquecida a compatibilidade fail-closed. Procedimento, hashes e capturas estão em [runtime-attach-validation.txt](logs/runtime-attach-validation.txt).

## Ajuste de sobreposição

A PR #58 propunha observar diretamente o tamanho do container em `TerminalPane`. Esse arquivo já contém `observeTerminalPaneResize`, uma correção local com regressão própria que acompanha redimensionamento do painel sem depender da janela. Preservei essa implementação e não apliquei o observer redundante da PR. Combinei apenas o script `test:e2e:terminal-split` ao `package.json` já alterado localmente. A integração filtrada passou em `git apply --check` antes da aplicação.

## Validação executada no overlay

| ID | Procedimento | Resultado | Evidência |
|---|---|---|---|
| PR578-001 | `GOCACHE=/tmp/nexus-pr578-current-gocache make quality-full` | **PASS** — exit 0; format, lint, Go tests, 105 arquivos/660 testes Vitest, race detector completo e `govulncheck` sem vulnerabilidades. | `logs/pr578-current-quality-full-final.log` |
| PR578-002 | `make web` | **PASS** — produção web compilada e assets oficiais do NEX exportados. | `logs/pr578-current-web-build.log` |
| PR578-003 | `PLAYWRIGHT_EXECUTABLE_PATH=/usr/bin/google-chrome bun run test:e2e:terminal-split` | **PASS** — 7 geometrias: vertical/horizontal em 20/50/80% e viewport 360×640. A regressão adicional usa identidades longas e verifica que o rótulo permanece dentro da aba. | `logs/pr578-current-terminal-split-geometry.log`; `logs/terminal-split-overflow-browser-regression.log` |
| PR578-004 | Builds CLI, Desktop Wails e HUD com Go 1.25.14 e tags `production,gtk3` | **PASS** — três ELFs x86-64; `ldd` sem dependências ausentes. Versão embutida `1.0.0-beta.10`, commit-base `bf6adf0c`. | `logs/pr578-current-binary-file.log`, `logs/pr578-current-desktop-ldd.log`, `logs/pr578-current-hud-ldd.log`, `logs/pr578-current-binary-sha256.log`, `logs/pr578-current-version.json` |
| PR578-005 | Abrir Desktop real com XWayland, perfil XDG temporário e Core embutido; abrir HUD independente no mesmo Core | **PASS** — Desktop visível, Core em loopback, HUD anexado ao mesmo Core; ambos encerrados depois. | `logs/pr578-current-native-desktop.log`, `logs/pr578-current-native-hud.log`, `screenshots/pr578-current-desktop-native-xwayland.png`, `screenshots/pr578-current-hud-native-xwayland.png` |
| PR578-006 | Wails XWayland, duas sessões shell reais e split vertical com caminhos longos | **PASS visual** — rótulos PTY ficaram contidos em suas abas, sem invadir abas vizinhas. Desktop AMD64 SHA-256 `adb7d9d21035452ca7e6c034d3b1ff2dea03357084f0990c2a56652f227e19f7`. | `logs/terminal-split-native-after-fix-validation.txt`; `screenshots/terminal-split-title-clipped-after-fix-xwayland.png` |

## Limites que mantêm o gate aberto

- O layout de duas sessões foi exercitado na janela Wails real; a sobreposição visual foi corrigida. Digitação/comandos ANSI e Unicode em cada PTY, resize por drag/teclado, persistência após reinício e reconexão de sessões divididas ainda não foram validados. XTest não entregou entrada de shell de forma confiável; o caso permanece BLOCKED, não aprovado.
- O teste de worktrees cobre parsing e descoberta em repositório Git real na suíte automatizada; a navegação/seleção pelo diálogo nativo ainda não foi exercitada.
- As duas janelas nativas foram executadas em XWayland numa sessão Wayland. A máquina VMware não oferece aceleração 3D; logs EGL confirmam que a GPU não está disponível. Wayland direto, transparência alpha e desempenho acelerado seguem fora desta evidência.
- O build incorpora o SHA base do Git. O código local adicional está identificado pelo manifesto de fontes `logs/pr578-current-source-manifest.txt`, mas não tem SHA Git imutável.

Esta validação melhora a cobertura e integra localmente as propostas seguras das PRs #57/#58. Ela **não altera o veredito `NO_GO_LINUX`** do relatório principal: versão beta.10, overlay sem commit, runtime completo, HUD com evento real e outros gates da matriz permanecem pendentes.

## Reprodução

```bash
GOCACHE=/tmp/nexus-pr578-current-gocache make quality-full
make web
cd web
PLAYWRIGHT_EXECUTABLE_PATH=/usr/bin/google-chrome bun run test:e2e:terminal-split
```

Os três comandos `go build` completos e hashes estão registrados nos logs acima; os binários temporários ficaram em `/tmp/nexus-linux-100-2026-10-25-pr578/bin/`.
