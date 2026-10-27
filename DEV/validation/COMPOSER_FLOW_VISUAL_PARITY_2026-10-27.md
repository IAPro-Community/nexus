# Composer / Flow — auditoria de paridade visual (27/10/2026)

**Branch:** `release/v1.0.0`  
**Estado:** IMPLEMENTAÇÃO PARCIAL / VISUAL QA PENDENTE

## Evidência de divergência

A captura de usuário de 27/10 mostrava a tela inicial do Composer como um
formulário denso, com campos alinhados num único card, opções técnicas
destacadas e histórico misturado ao início da elaboração. O plano anterior em
`docs/superpowers/plans/2026-09-15-composer-conversation-workspace.md`
previa um workspace conversacional com composer principal, inspeção lateral
e entrada amigável a iniciantes. A presença de checkboxes "done" no plano
não comprova que **todas as superfícies** estão visualmente implementadas.

## Alteração integrada nesta rodada

- `ComposerStart.tsx`: tela inicial com hero explicativo, entrada primária
  para objetivo, opções de profundidade secundárias e histórico separado;
  botões da IA e opções de excluir/abrir preservados.
- `ComposerSurface.module.scss`: estilos responsivos baseados nos tokens Nexus,
  sem criar um novo design system.
- `useComposerSession.ts`: atividade pendente encerrada também quando a
  inicialização falha.
- `ComposerStart.tsx` / `SettingsSurface.tsx`: acesso ao diagnóstico do
  provedor, teste real explícito e distinção entre configuração e resposta.
- Backend: erros de probe categorizados sem expor stdout, paths ou chaves.

## Ainda não implementado/homologado nesta rodada

- Revisão visual completa dos estados **sessão iniciada**, **briefing**,
  **revisões**, **artefato final**, **handoff** e **Flow editor/run** versus os
  protótipos originalmente aprovados.
- Navegação e acessibilidade visual em todos os temas, alto contraste,
  desktop/tablet/mobile e zoom.
- Motion refinado sem animação artificial de progresso.
- Teste E2E com provedor real do usuário, investigação de erros de CLI,
  screenshot e comparação visual nos breakpoints.
- Evidência de build/distribuição do aplicativo instalada pelo usuário a partir
  do SHA final. **Push na branch não atualiza automaticamente o binário em uso**.

## Gate de publicação

Não declarar "layout final" ou "paridade completa" até executar
`bun run typecheck`, `bun run lint`, `bun run build`,
os E2Es de Composer e os testes de regressão de Flow, obter capturas por
breakpoint/tema e comprovar revisão visual aprovada. Reportar o SHA exato da
build entregue, não apenas o HEAD do repositório.
