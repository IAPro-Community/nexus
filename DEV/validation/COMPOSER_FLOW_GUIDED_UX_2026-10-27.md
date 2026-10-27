# Auditoria Composer + Flow + Tour — 27/10/2026

**Branch:** `release/v1.0.0`  
**Objetivo:** alinhar onboarding, UX inicial, Flow Studio, entendimento de execução e educação contextual ao design registrado.
**Estado:** CÓDIGO IMPLEMENTADO / TESTES CRIADOS / **SEM CERTIFICAÇÃO DE RELEASE**.

## Inventário de funcionalidades e proveniência

| Jornada | Implementação na branch | Evidência de homologação |
| --- | --- | --- |
| Composer: primeira ideia | `ComposerStart.tsx` com objetivo multilinha, hero contextual, modo/nível e histórico separado | Verificação estática somente |
| Composer: IA offline | `ComposerReadOnlyHistory.tsx` e diagnóstico explícito de Nexus Intelligence | Teste real pendente |
| Composer: briefing vivo | `ComposerSurface.tsx`, `ComposerInspector.tsx`, `useComposerSession.ts` reutilizados | Teste real pendente |
| Composer: decisões e artefatos | `LivingBrief`, `PromptArtifact`, gates e contrato v1 preexistentes | Teste end-to-end pendente |
| Composer → Flow | `composer.handoff.v1` no WorkPlan, com revisão, hash, propostas e bloqueios | Teste end-to-end pendente |
| Flow: criação | `FlowRunsHistorySurface.tsx` cria rascunho manual; conversão rica vem do Composer | Texto/UI alinhados |
| Flow: inspeção | `FlowDraftEditor.tsx`, pipeline e `@xyflow/react` gráfico | Visual QA pendente |
| Flow: GO e gates | `ExecutionPreview`, preflight e `MissionRunner` preexistentes | Regressão pendente |
| Flow: monitoramento | `FlowRunSurface.tsx` com revisão congelada, estados, decisões e evidências | Teste E2E pendente |
| Contextual tour: Composer | `ComposerStart`, sessão e histórico offline | Código e âncoras reais |
| Contextual tour: Flow | Biblioteca, editor e Mission | Código e âncoras reais |
| Tour e acessibilidade | `FeatureTour` + `ProductTour` original, navegação, Esc, foco, viewport mobile e reduced motion | Testes unitários criados; browser pendente |
| i18n | Textos de tour e onboarding PT/EN/ES | Teste de paridade criado; execução pendente |

## Implementação desta rodada

- Reaproveitamento do tour global sem dependência adicional; opt-in por
  **Conhecer esta área**, sem passos que iniciem agentes, criem artefatos ou
  mudem configurações.
- Persistência local isolada por projeto/superfície do último passo aberto;
  conclusão reinicia, fechamento permite retomar, histórico vazio ou
  elementos condicionais são ignorados automaticamente.
- Suporte a retorno do foco, teclado Tab/Shift+Tab/Esc, reduced motion e
  viewport mobile sem criar overlays concorrentes.
- Composer welcome não é mais um formulário técnico de entrada: objetivo
  multilinha com label ligado ao campo e hierarquia de ação.
- Flow Studio diferencia rascunho manual de decomposição por IA, e apresenta
  a jornada Plan → Review → Mission → Evidence, com identidade Nexus.
- Atualização das seleções de Playwright do Composer que dependiam de campos
  antigos, para não quebrar o teste após mudar o JSX.
- Testes de contrato `featureTour.test.ts` e `featureTourAnchors.test.ts`;
  regressão global mantida em `tour.test.ts`.

## Cenários mínimos (prova pendente)

1. Projeto novo → Composer → conhecer área → próximo/voltar/fechar → reabrir
   na etapa gravada → concluir → reabrir do começo.
2. Composer IDEA vs EXISTING_PROMPT; modo QUICK/STANDARD/DEEP; prover IA
   configurada, conexão falha, IA offline. Nunca deixar atividade "pending"
   após erro.
3. Composer sessão ativa → tour de cabeçalho, conversa e inspector; com painel
   escondido/mobile pular passo invisível sem travar.
4. Converter artefato aprovado em Flow, sem iniciar Mission; conferir
   proveniência e não promoção de PROPOSED a CONFIRMED.
5. Flow Studio sem rascunhos / com rascunhos; visita às trilhas e criação manual;
   conhecer área e abrir editor.
6. Editor → modo pipeline/grafo, seleção de etapa, WorkPlan revision; preview
   sem efeitos colaterais, blocker de aprovação e GO apenas com autorização.
7. Mission ativa/concluída → tour de estado, evidência e ações; nenhum passo
   chama endpoint de mutação.
8. PT/EN/ES, tema claro/escuro, Windows/Linux/macOS, zoom 100–200%,
   teclado sem mouse, viewport mobile/tablet e contraste aumentado.
9. Capturar screenshots reais com SHA do artefato compilado e comparar com
   protótipos anteriores, especialmente cabeçalho, onboarding, distribuição
   de painéis e legibilidade.

## Execução e publicação

```bash
git switch release/v1.0.0
git pull --ff-only
go test ./internal/nexus/... ./internal/control/web/...
cd web
bun install --frozen-lockfile
bunx vitest run src/features/work/featureTour.test.ts src/features/work/featureTourAnchors.test.ts src/app/tour/tour.test.ts
bun run format:check
bun run lint
bun run lint:styles
bun run typecheck
bun run test
bun run build
bun run test:e2e:composer
bun run test:e2e:composer-spine
bun run test:e2e:composer-vague-spine
bun run test:e2e:composer-themes
```

**NÃO HOMOLOGADO:** o GitHub connector permite edição de fonte mas não executa
essas suítes, gera build nativa ou fornece screenshots do app instalado.
Não houve validação dos runners de CI neste trabalho. O checkout via shell
neste ambiente falhou por DNS de `github.com`. Não se pode prometer ausência
de regressões nem que uma instalação anterior do Nexus tenha sido atualizada.

**Release gate:** aprovação só após testes e verificação visual do exato SHA
distribuído. A evidência do tour não substitui a evidência do MissionRunner.
