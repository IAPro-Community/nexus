# Tours guiados — Composer e Flow

**Estado na branch `release/v1.0.0` (27/10/2026): implementado em código, homologação visual e E2E pendentes.**

Composer e Flow possuem **tours separados por situação real do projeto**, construídos
sobre o `ProductTour` existente do Nexus — sem nova biblioteca, sem criar uma
segunda conversa e sem executar comandos ou alterar planos como efeito do tour.

## Como acessar

Dentro de cada área, clique em **Conhecer esta área**. Use **Próximo**,
**Voltar**, **Concluir** ou **Fechar**. O tour pode ser aberto novamente a qualquer
momento; ao fechar antes de concluir, a etapa atual é preservada no
`localStorage` **do navegador**, associada ao projeto e à superfície.
Concluir reinicia o ponto de partida para uma visita futura. Se storage estiver
indisponível, o tour continua funcionando sem persistência.

O tour é **opcional** e não se abre automaticamente, para não conflitar com
o tour global do Workspace OS nem interromper uma sessão de agentes ativa.

## Composer

| Situação | Assuntos explicados |
| --- | --- |
| Primeira elaboração | Proposta do Composer, objetivo em linguagem natural, Ideia/Prompt existente, profundidade, histórico |
| Sessão aberta | Cabeçalho e versões, conversa refinável, painel de briefing/contexto/prompt, artefato e destinos |
| IA indisponível | Configuração/teste do Nexus Intelligence e acesso ao histórico salvo |

O Composer exige uma IA que responda ao teste de conexão para mutações de
planejamento. O tour **não** simula um provedor disponível. Uma
recomendação da IA permanece PROPOSED até a decisão humana. Abrir a
elaboração **não** inicia Agent ou Mission.

## Flow

| Situação | Assuntos explicados |
| --- | --- |
| Biblioteca | Rascunhos vs Missions, criação manual vs Composer, retomada de Flows |
| Editor | Objetivo, líder/versão, contrato imutável do Composer quando presente, pipeline/grafo e etapas |
| Mission | Progresso de revisão congelada, evidências e controles de intervenção/recovery |

O Flow é uma projeção do WorkPlan e o `MissionRunner` é a autoridade
de execução. O botão de criação da biblioteca gera **rascunho manual**, não
decomposição automática com IA. O tour enfatiza que GO requer validação e
aprovação do plano; não dispara GO, arquivos, terminals nem alterações externas.

## Experiência e acessibilidade

- O tour usa os seletores reais `data-tour` das telas. Elementos ausentes,
  condicionais ou invisíveis são pulados, inclusive histórico vazio e
  inspector fechado no mobile.
- Há foco no diálogo, suporte a Esc, contenção de Tab/Shift+Tab, retorno
  do foco, rolamento até o controle, limite de altura no mobile e respeito
  a `prefers-reduced-motion`.
- Títulos, descrições e comandos estão em português (Brasil), inglês e espanhol.
- As etapas são informativas, não coletam prompt, credenciais ou dados pessoais.
  O único dado de persistência é um índice numérico local por área/projeto.

## Arquitetura e fontes de verdade

- `web/src/app/tour/ProductTour.tsx`: overlay e navegação reutilizados;
- `web/src/features/work/featureTour.ts`: contrato tipado das etapas;
- `web/src/features/work/FeatureTour.tsx`: botão de abertura, fechamento e
  progresso por projeto;
- `ComposerStart`, `ComposerWorkspaceHeader`, `ComposerInspector`,
  `ComposerReadOnlyHistory`, `FlowRunsHistorySurface`,
  `FlowDraftEditor`, `FlowRunSurface`: âncoras reais;
- `web/src/i18n/resources.ts`: strings em três idiomas.

Nenhum destes componentes modifica os contratos de Composer, WorkPlan,
MissionRunner, Maestro ou Nexus Intelligence.

## Testes necessários antes de certificar

```bash
cd web
bun install --frozen-lockfile
bunx vitest run src/features/work/featureTour.test.ts src/features/work/featureTourAnchors.test.ts src/app/tour/tour.test.ts
bun run typecheck
bun run format:check
bun run lint
bun run lint:styles
bun run build
bun run test:e2e:composer
bun run test:e2e:composer-spine
bun run test:e2e:composer-vague-spine
bun run test:e2e:composer-themes
```

Na aplicação instalada, executar manualmente para cada projeto:
desktop/móvel, 100–200% zoom, temas claro/escuro, locale PT/EN/ES,
teclado, disponibilidade/offline da IA, Composer inicial e finalizado,
Flow sem plano, Flow editável e Mission executada. Checar screenshot
**do SHA da build**, não somente do HEAD do repo. Sem evidência do build,
o status de design continua **NO_GO**.

## Novidades e ajuda por componente

Além do tour manual, a barra superior oferece **Novidades do Nexus**
e cada área tem um menu contextual para recursos novos, com **Mostrar no
componente** e **Dispensar**. O contador usa `id + revision` por recurso;
uma alteração significativa pode reapresentar o aviso uma vez sem
interromper trabalhos. A ajuda pontual também utiliza Radix Tooltip
do design system já instalado. Veja
[Product Discovery e catálogo de anúncios](feature-discovery.md) para a
política de inclusão de novidades, componentes avaliados, privacidade
e limites do processo de publicação.
