# Nexus 1.0 — Terminais divididos, busca contextual e ResourceScheduler

> Escopo: melhorias aditivas na branch `feat/v1-contextual-workspace-experience`, derivada da `release/v1.0.0`. Os gates do exato SHA permanecem obrigatórios. Este documento não declara homologação.

## A. Split terminal (reutiliza PTY/ConPTY)

O Nexus já tinha um host único de PTYs supervisionados (Terminais) e o `WorkspacePresentationState` persistente por projeto. Criar outro TerminalManager duplicaria estado, socket e lifecycle. A implementação introduz um quarto modo visual `SPLIT` na mesma apresentação, mantendo `TABS`, `DESKTOP` e `MOSAIC` sem alteração de protocolo.

- No cabeçalho de Terminais, escolha **Dividir** (habilitado com duas ou mais sessões PTY).
- Selecione disposição lado a lado ou empilhada. No viewport compacto, a visualização é sempre empilhada, sem mudar a orientação salva para desktop.
- Use o divisor com ponteiro, ou foco no divisor + setas (incrementos de 5%, limites 20%–80%).
- Para escolher o terminal do segundo painel, use o menu de contexto da aba. Clique em outra aba para focar um terminal existente ou substituir o primeiro painel.
- O foco ativo é independente das identidades do primeiro/segundo painel, evitando movimentação inesperada quando o usuário interage com um terminal.
- O layout usa o `WorkspaceLayoutService` v4 já persistido por Project, com campos aditivos: `splitPrimaryViewId`, `splitSecondaryViewId`, `splitOrientation` e `splitRatio`. O schema de persistência e a chave não são alterados.
- As sessões são reutilizadas, não clonadas. O render de painéis ocultos segue o mecanismo keep-alive que já existia; nenhum terminal novo é criado ao dividir.
- Se um PTY for fechado ou não existir, o resolvedor só exibe identidades disponíveis.

## B. Busca e comandos rápidos

`Ctrl/Cmd+K` continua usando a Command Palette existente. A lista agora permite ações de projeto e terminal além da navegação existente e mantém as teclas do provider nativo intactas.

| Prefixo | Escopo | Exemplo |
| --- | --- | --- |
| `#` | Projetos conhecidos | `#escola` |
| `@` | Agentes carregados | `@revisor` |
| `$` | Recursos, contas, escolha de modelo | `$conta` |
| `>` | Comandos gerais | `>terminais` |
| Sem prefixo | Tudo | `split` |

O filtro remove diferenças de acentuação, exige correspondência para todos os termos e mantém ordem estável para resultados equivalentes. A seleção continua explícita: pesquisa **não executa** comandos nem alterna contas automaticamente.

## C. Seleção contextual de contas e modelos no Direct

O Direct Session Launcher passa o tipo de tarefa (`coding`, `planning`, `review` ou `security`) para `POST /api/v1/resources/recommend`. O backend `ResourceScheduler` classifica perfis/contas usando seu próprio contrato de elegibilidade e política. `Project.resource_policy`, que historicamente é uma string JSON, é normalizada com allowlist para `BALANCED | PRESERVE_QUOTA | PREFER_PROVIDER`.

- A conta sugerida é pré-selecionada apenas se existir na lista de contas manualmente elegíveis retornada pelo Core.
- Sem recomendação confiável, o Nexus não escolhe a primeira conta por ordem alfabética nem inventa quota.
- O usuário pode selecionar outra conta; `POST /api/v1/resources/select` permanece o gate final de política `MANUAL`.
- A identidade de provider/profile e o override explícito de `model` são salvos em `AgentConfig` antes do start; o modelo em branco usa o padrão do provider.
- O campo de modelo **não é um catálogo de modelos certificados**. A recomendação atual é de **conta/perfil**, não um ranking modelo+conta. A seleção de modelo digitado pelo usuário é uma preferência manual; o provider é responsável por aceitar/rejeitar o ID no lançamento. Não chamar isto de decisão automática de melhor modelo.
- O `ModelCandidate`/`ResolveTaskModel` do MissionRunner permanece a autoridade para seleção contextual de modelos **quando existe catálogo de candidatos com proveniência**. Disponibilizar tal catálogo no Direct exigirá uma API própria de modelos por provider e critérios de confiança; não deve ser implementada com lista hardcoded no frontend.

## Gates obrigatórios antes de merge

1. `bun run --cwd web quality:full`; format-check, TypeScript, ESLint, Stylelint e Vitest.
2. Playwright em Chromium: layout Split em duas sessões reais, redimensionamento, escolha do secundário, fechar uma sessão, troca de foco, render do painel em outra aba, scroll e suporte de teclado.
3. Reabrir projeto/reiniciar frontend: persistência do modo, da orientação e da proporção; sem reaproveitamento de layout entre projetos.
4. Testes com Mac e Windows/ConPTY: colagem, seletor CLI, setas, Ctrl+C, reconexão e input byte-transparent. Não confundir build cross-platform com homologação nativa.
5. ResourceScheduler: conta recomendada não ser a primeira da lista, ausência de recomendação, quota UNKNOWN/CACHED/UNSUPPORTED, múltiplos perfis, mudança da especialidade e falhas de rede.
6. Modelo: override manual, default nativo, modelo inválido, falha de start e possível Agent parcialmente criado.
7. Acessibilidade: comandos de teclado e leitores de tela, reduced motion, contraste de divisor e viewport compacto.
8. Rebase e certificação same-SHA da release antes de qualquer merge/tag/build público.

## Não escopo

Nenhum novo PTY, renderer, scheduler, modelo de memória, marketplace, runtime remoto ou sistema de conta é criado. Não há atualização da `release/v1.0.0` nesta branch até os gates e o review do PR serem concluídos.

## Correções de hardening — seleção e criação de agente

- A preferência explícita de provedor é repassada para `requirements.prefer_provider` e ativa `PREFER_PROVIDER` no scheduler existente. O `resource_policy` JSON do Project também pode conter `prefer_provider`. Quando essa política não declara nenhum provedor, o launcher usa `BALANCED` sem inventar afinidade.
- Se `selectResource` ou `applyAgentConfig` falhar, a criação é desfeita via `deleteAgent`. Falhas no rollback preservam o ID no erro.
- Se `startAgent` retornar erro, a sessão pode já estar rodando; **não** excluir automaticamente o agente. O usuário é orientado a verificar o painel de Agentes.
- Falhas de UI/refresh depois de um start bem-sucedido também preservam a sessão.
- O botão de submissão impede invocações concorrentes e o diálogo impede fechamento durante a inicialização.

A seleção automática do modelo continua bloqueada até que o Direct obtenha um catálogo confiável de modelos compatíveis por perfil. Não criar lista hardcoded nem tratar nomes de modelos digitados como certificação.

## Split — resize do Project Shell

O `AgentTerminal` já observava mudanças de tamanho do contêiner. O `TerminalPane` do Project Shell agora também observa o contêiner com `ResizeObserver`, agendando o `FitAddon` no próximo frame durante o arraste do Split. Ao desmontar, o observer e o frame pendente são descartados. Continua sendo obrigatório confirmar resize PTY real no browser/ConPTY/macOS.
