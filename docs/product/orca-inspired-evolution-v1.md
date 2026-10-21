# Orca como referência de produto — Nexus v1.0

> Pesquisa técnica em 21/10/2026. Referência: [Orca](https://www.onorca.dev/) / [stablyai/orca](https://github.com/stablyai/orca). A referência é de **fluxos de UX**, não uma proposta de migrar a arquitetura do Nexus para Electron ou copiar sua identidade visual. Orca é distribuído sob MIT; qualquer incorporação futura de código de terceiros exige a manutenção dos avisos de licença relevantes.

## Objetivo e restrições

Aproximar a experiência diária do Nexus de um ambiente de desenvolvimento orientado a agentes, sem remover seus diferenciais: agentes persistentes, Project, Direct, Composer, Flow/WorkPlan, MissionRunner, Capacity Monitor, ResourceScheduler, perfis/quotas e integração opcional com Maestro.

Restrições da linha `release/v1.0.0`:

- Não substituir o Core Go, o terminal supervisionado PTY/ConPTY, o Web/Desktop existente ou o scheduler.
- Não interceptar entrada nativa de Codex/Claude/OpenCode para simular comandos de UX; preservar `STICKY_LEAD` e a autorização formal de Mission.
- Não misturar identidades de contas nem assumir quotas `UNKNOWN` como disponíveis.
- Não abrir automaticamente acesso SSH ou remoto; o túnel atual tem sua própria fronteira de autenticação.
- Não mudar a interface padrão nem declarar suporte multiplataforma sem evidência exact-SHA.
- Toda adição deve possuir rollback/feature flag quando alterar runtime, shell ou WebView.
- Respeitar `AGENTS.md`, i18n, SCSS Modules, tokens do Nexus e acessibilidade.

## Matriz comparativa

| Referência Orca | Situação encontrada no Nexus 1.0 | Decisão |
| --- | --- | --- |
| Vários agents em worktrees paralelos | O Core já cria worktrees por Agent/Mission em `internal/nexus/worktree.go` | **Reaproveitar**. Acrescentar descoberta, associação e operação visíveis |
| Splits de terminais e panes persistentes | Existem PTY/ConPTY, sessões e superfícies de terminal; paridade visual e de persistência de panes não certificada | **Evoluir o workspace**, não o transporte do provider |
| Terminal GPU/WebGL e scrollback | O Nexus possui terminal web supervisionado; equivalência com Ghostty não foi demonstrada | Priorizar estabilidade, resize, input e reconexão, não trocar renderer |
| Account switcher / usage | Quota, perfis e handoff fazem parte do Core, com estados de confiabilidade | Melhorar descoberta/atalhos, sem outro motor de quota |
| Navegador embutido / Design Mode | Há suporte a browser e Desktop, mas não há contrato homologado de seleção HTML/CSS/screenshot por worktree | Fazer spike isolado, fora do caminho crítico da beta |
| Anotação de diff enviada ao agent | O fluxo de review/verificação existe; interface de anotação em diff não certificada | Criar projeção read-only antes de mutações/feedback |
| GitHub/Linear no workspace | Há comandos Git locais e dados de projeto; não equivalem a uma integração completa de PR/Issues/Linear | Iniciar por GitHub, integração opcional e escopo mínimo |
| Busca global por worktrees/agentes | Busca de projetos e outras buscas locais existem; um índice universal não foi evidenciado | Agregar fontes existentes via command palette, sem segundo indexador |
| SSH worktrees | Túnel HTTPS autenticado não equivale a executar em host SSH remoto | Tratar como nova fronteira de segurança e feature própria |
| Companion iOS/Android | Web remota existe com convite de curta duração; companion nativo não existe como produto certificado | Não incluir no fechamento da v1.0 |

## Primeira vertical slice: descoberta de worktrees

Mudança nesta branch:

- Estender `GET /api/v1/projects/:id/git/branches` com `worktrees` e `worktrees_available`, preservando o contrato anterior.
- Ler `git worktree list --porcelain -z` com contexto e timeout, sem executar `prune`, `remove`, `checkout` ou qualquer operação destrutiva.
- Exibir checkout não primário no diálogo de branches, incluindo branch/diretório e indicadores `detached`, `locked` e `prunable`.
- Desabilitar no cliente a tentativa de checkout de branch já ocupada por outro worktree (Git permanece a autoridade final no backend).
- Localizar labels EN/PT-BR/ES; não duplicar regras de isolamento de Agent.

A descoberta é **somente leitura**. Não significa que Nexus controla, possui, pode remover ou deve iniciar um runtime naquele worktree. O status `worktrees_available=false` é exibido explicitamente; ausência de leitura não deve virar afirmação falsa de que não existem worktrees.

### Gates

1. Go: `go test ./internal/control/web/...`, `go test ./internal/nexus/...`, `gofmt -l`, `go vet`.
2. Web: `bun run --cwd web quality:full` e browser E2E de abertura, foco, filtro, branches ocupadas, mobile e regressão de checkout.
3. Git real: checkout primário + 2 worktrees, diretório com espaço, `detached`, `locked`, `prunable`, branch em uso, Git inexistente.
4. Windows/macOS/Linux: Git path handling, rolagem dentro do diálogo, WebView/zoom e nenhum desvio de PTY/ConPTY.
5. Release: gates no mesmo SHA, sem certificar com logs de commits ancestrais.

## Próximas fatias, por valor e risco

1. **Workspaces paralelos utilizáveis:** ação `Novo worktree` opcional por tarefa, idempotência, branch base explícita, criação atômica, política de limpeza **manual e confirmada**, associação a Project/Agent, abertura de terminal no diretório. Nunca remover worktree com alterações não commitadas ou runtime ativo.
2. **Workspace pane manager:** splits horizontais/verticais, tabs e layout persistido por projeto. Reusar `Project layout`, terminal supervisionado e design system; o mobile deve ter navegação entre painéis, não múltiplas colunas espremidas.
3. **Git review workspace:** diff read-only incremental, arquivos/linhas alterados, comentários locais versionados e envio explícito de feedback ao Agent. Sem aplicar patch ou fazer commit sem confirmação.
4. **Navegação e busca unificadas:** comando rápido sobre projetos, branches, agentes, sessões, worktrees e comandos já registrados. Sem indexação de conteúdo sensível nem polling agressivo.
5. **Uso de contas contextual:** seleção de profile disponível no launcher, proveniência de quota e sinalização `UNKNOWN`; reaproveitar o ResourceScheduler.
6. **Investigação separada:** Design Mode (Chromium e seleção visual), integração GitHub PR/Issues, SSH remoto e companion mobile. Cada um exige threat model, contratos e homologação próprios; nenhum é pré-requisito da release v1.0.

## Critério para incorporação à release

A mudança só entra na `release/v1.0.0` depois de review, qualidade local, regressão Web, smoke nativo aplicável e aprovação dos gates de release. O PR não autoriza merge automático nem publicação de instaladores. **Uma melhoria Orca-inspired não deve transformar a beta atual em uma reescrita de IDE.**
