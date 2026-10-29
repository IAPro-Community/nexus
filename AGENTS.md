# IAPro Nexus — Contrato Oficial para Coding Agents

## STYLE RULES

- New component styles MUST use SCSS Modules.
- Do not create local .css files.
- Existing local CSS touched by the task should be migrated to .module.scss when safe.
- Runtime theme/layout values MUST use CSS Custom Properties.
- Static styling MUST NOT be placed inline.
- Do not use !important.
- Reuse semantic design tokens.
- Reuse existing components before introducing duplicated styles.
- Global styles are restricted to reset, tokens, themes and real global foundations.

## 1. Regra Fundamental
Antes de qualquer alteração substancial no frontend ou backend, leia e siga obrigatoriamente:
* `docs/engineering/WEB_ENGINEERING_STANDARDS.md` (Padrões de Engenharia Frontend)
* `docs/engineering/ENGINEERING_STANDARDS.md` (Padrões Globais do Nexus)

## Product Discovery — regra para novas funcionalidades

Em qualquer PR que introduza ou altere materialmente uma funcionalidade
visível no Nexus, atualizar o catálogo de anúncios em
`web/src/features/work/featureAnnouncements.ts` no mesmo commit (ou
justificar por escrito que o recurso é interno e não deve ser anunciado).

- Adicionar `id` estável, `revision` incremental, `openSurface`, superfície
  real, seletor `data-tour`, título e descrição localizados PT-BR/EN/ES.
- Reutilizar `FeatureTour`/`FeatureHint` e o `ProductTour` compartilhado.
  Não criar um segundo tour engine, ticker ou central de notificações.
- Para uma área nova, incluir `data-feature-tour-scope` no container do
  módulo, e manter seletor/etapa dentro dessa instância de UI para não
  cruzar painéis ou projetos.
- A experiência é opt-in e não pode executar comandos, disparar agentes,
  confirmar custos, modificar plano ou revelar dados sigilosos.
- Cobrir os vínculos de UI, i18n, `featureReveal.test.ts`,
  `tourTargets.test.ts` e `featureAnnouncements.test.ts`, além de E2E
  visual no SHA publicado. Sem execução de teste, informar NO_GO.
- Reutilizar bibliotecas licenciadas adequadamente; antes de instalar
  novos pacotes, verificar licença, lockfile e build reprodutível. Driver.js
  1.10.x é candidato documentado em #103, ainda não instalado.

Especificação: `docs/product/feature-discovery.md`.

## 2. Regras Prescritivas para o Frontend (web/)

1. **Pesquisa Obrigatória Antes de Criar**:
   - Nunca crie um novo componente ou primitiva de UI sem antes pesquisar em `web/src/design-system/primitives/`, `web/src/components/` e `web/src/features/`.
   - Hierarquia obrigatória: `Componente Existente > Design System > Primitiva Radix (@radix-ui/react-*) > Nova Implementação`.
   - Proibido criar design systems paralelos ou variações como `CustomButton`, `NewModal`, `ButtonV2`.

2. **HTML Semântico & Native First**:
   - Use o elemento nativo correto (`<button type="button">`, `<a>`, `<header>`, `<nav>`, `<main>`, `<footer>`, `<form>`, `<label>`, `<select>`).
   - Proibido usar `<div onClick>` ou `<span onClick>` quando existe controle HTML nativo equivalente.
   - Todo botão com apenas ícone (`IconButton`) deve ter `label` acessível ou `aria-label`.

3. **Acessibilidade (WCAG 2.2 AA)**:
   - Toda funcionalidade deve ser navegável por teclado (`Tab`, `Enter`, `Space`, `Escape`, setas).
   - Nunca remova o anel de foco (`outline: none`) sem fornecer `:focus-visible` visível e de alto contraste.
   - Garanta contraste adequado em todos os presets registrados em `THEME_PRESETS`; nunca fixe a validação a uma contagem estática.

4. **Estilização & Design Tokens**:
   - SCSS Modules (`*.module.scss`) é o padrão oficial de estilização encapsulada para componentes.
   - CSS Custom Properties (`var(--nx-*)`) são a autoridade para cores, layout, temas e densidade.
   - Proibido CSS inline estático (`style={{ ... }}`). Estilos inline são permitidos **apenas** para propriedades calculadas dinamicamente em runtime (ex: dimensões e posições de janelas).
   - Proibido adicionar `!important` para resolver conflitos de especificidade.
   - Proibido cores arbitrárias hardcoded (`#fff`, `#333`). Consuma tokens semânticos (`--nx-surface`, `--nx-text`, `--nx-accent`, etc.).

5. **Internacionalização (i18n)**:
   - Proibido texto visível hardcoded em JSX/TSX.
   - Todos os títulos, botões, labels, placeholders, tooltips, mensagens de validação e estados vazios devem usar `react-i18next` (`t('chave.semantica')`).
   - Use `Intl.DateTimeFormat` / `Intl.NumberFormat` para datas e números.

6. **Componentização & Limites**:
   - **1 componente por arquivo**: Separe componentes conceituais em arquivos individuais.
   - Heurística de tamanho: `~200 linhas` (revisar), `~300 linhas` (dividir), `>400 linhas` (exige justificativa técnica).
   - Coloque testes (`*.test.tsx`) e estilos (`*.module.scss`) próximos aos componentes (`colocation`).
   - `src/shared/` nunca importa de `src/features/`. Features nunca importam internals de outras features.

7. **TypeScript & Resiliência a Dados Externos**:
   - `strict: true` em todo o projeto.
   - Proibido `any`, `@ts-ignore`, `@ts-nocheck` como atalhos para passar checks.
   - Arrays da API Go podem ser JSON `null`. Nunca use `.length`/`.map` direto em campos nullable sem normalização (`asArray()` de `web/src/lib/safeArray.ts` ou `(val || [])`).

---

## 3. Checklist Obrigatório para Coding Agents

### Before coding
- [ ] Pesquisei se já existe componente equivalente no projeto.
- [ ] Pesquisei se já existe primitiva instalada (`@radix-ui/react-*`, Lucide, etc.).
- [ ] Identifiquei o owner arquitetural da feature (`features/`, `workspace/`, `shared/`).
- [ ] Verifiquei que não estou duplicando código nem criando abstração prematura.

### During coding
- [ ] HTML semântico utilizado (`<button>`, `<main>`, `<header>`, `<label>`, etc.).
- [ ] Estilos de novos componentes em SCSS Modules (`*.module.scss`).
- [ ] Textos da interface internacionalizados via i18n (`t(...)`).
- [ ] Sem CSS inline estático; CSS Custom Properties e tokens semânticos respeitados.
- [ ] Sem novo `!important`.
- [ ] Componentes coesos, 1 componente conceitual por arquivo.
- [ ] Acessibilidade e navegação por teclado preservadas.

### Before completion
- [ ] Formatação passa (`npm --prefix web run format:check` / `make format-check`).
- [ ] ESLint passa (`npm --prefix web run lint`).
- [ ] Stylelint passa (`npm --prefix web run lint:styles`).
- [ ] Verificação de allowlist de estilos passa (`npm --prefix web run check:styles`).
- [ ] TypeScript passa (`npm --prefix web run typecheck`).
- [ ] Testes unitários/componentes passam (`npm --prefix web run test`).
- [ ] Build passa (`npm --prefix web run build`).
- [ ] Gate geral validado (`make quality` ou `make web-verify`).

---

## 4. Backend (Go) & Interface de Terminal

* Entry points em `cmd/` — apenas wiring inicial.
* Lógica em `internal/` dividida por capacidades.
* Interfaces de terminal devem usar o ecossistema Charm (Bubble Tea, Bubbles, Lip Gloss).
* Comandos com saída de dados devem suportar a flag `--json`.
* Validação de qualidade Go: `make lint:go`, `make test:go`, `make security`.

---

## 5. Git Commits

Siga rigorosamente o padrão Conventional Commits:

```text
type(scope): description

Types: feat, fix, docs, style, refactor, perf, test, build, ci, chore, revert
Scopes: terminal, chat, flow, agents, providers, quota, workspace, settings, web, go, infra, security, ui, deps, release
```

---

## 6. Mascote oficial NEX — paridade visual obrigatória

Em qualquer tarefa sobre **mascote, HUD Guardian, onboarding com NEX, sprites, animações, ilustrações de docs ou arte do site**:

1. Leia [contrato visual de NEX](docs/design/NEX_32BIT_VISUAL_CONTRACT.md), [manifesto de máquina](docs/design/NEX_VISUAL_MANIFEST.json) e [pranchas](docs/design/assets/nex/README.md) **antes** de gerar/modificar artes.
2. Use [guia de handoff para IA](docs/design/NEX_AI_GENERATION_GUIDE.md) e [checklist](docs/design/NEX_VISUAL_REVIEW_CHECKLIST.md). Sem inspeção do material, **não alegue fidelidade ou identidade oficial**.
3. NEX é **um único robô frontal, 32-bit-inspired**, com duas aletas, dois acoplamentos laterais, um núcleo circular e quatro módulos orbitais destacados (dois laterais, dois inferiores). Não trocar por mascote genérico, nem alterar a marca Nexus.
4. Preserve o Core, os estados observados `idle/working/warning/critical/offline`, a acessibilidade e o modo sem animações; não inferir sucesso ou progresso.
5. Para código ou documentação que altera NEX, execute `node scripts/verify-nex-visual.mjs`, anexe antes/depois e registre divergências, além dos gates habituais.
6. Para modificar movimentos, inspecione também [contrato de animação](docs/design/NEX_ANIMATION_CONTRACT.md): movimentos só em camadas internas, nunca no transform de âncora dos módulos; o modo sem animações e `prefers-reduced-motion` precisam desativar tudo, inclusive hover.
7. Standard/Hero ainda não têm masters de produção; **não** descrever pranchas auxiliares como arte raster original aprovada nem como screenshot real.

## Direção Hero de alta fidelidade (28/10/2026)

**Regra:** o desenho SVG anterior do NEX não passou na comparação perceptual com o PNG. Antes de redesenhar novamente, consultar `docs/design/nex/hybrid-v1/README.md` e `asset-manifest.json`: a abordagem preferida para o site preserva o PNG original como raster e anima recortes autênticos (2 olhos, 4 módulos e reflexos de aletas). Não alegar que a existência dos arquivos em manifesto significa que os bytes já foram commitados. A revisão visual é obrigatória antes de substituir assets do site/HUD. Atlas 360° continua pendente da produção de perspectivas reais.

## 7. NEX Identity Lock — congelamento após aprovação humana
**Direção escolhida em 27/10/2026:** manter o personagem robô original selecionado pelo usuário, reinterpretá-lo em pixel art HD retrô de game (não filtro aplicado ao SVG) e criar 12 ações de idle detalhadas. Leia `docs/design/nex/RETRO_GAME_IDLE_DIRECTION.md` e `docs/design/nex/identity-lock.json.selection_record`; não propor outra silhueta. O lock continua pendente de master editável, hashes e aprovação visual do sprite final, portanto não declarar código atual como retro HD concluído.

- O documento `docs/design/nex/IDENTITY_GOVERNANCE.md`, o manifesto `docs/design/nex/identity-lock.json` e as instruções `docs/design/nex/AGENTS.md` regem **toda nova arte NEX** (site, animação, HUD, agentes, avatar e campanhas).
- A identidade atual segue os contratos da seção 6; as referências recentes são **candidatas**, não um novo master aprovado. Não substitua o sprite da release por artes de conversa.
- Quando o manifesto mudar para `FROZEN`, nenhum agente poderá alterar a identidade, proporções ou masters/hashes sem mudança de versão, comparação, PR e aprovação explícita do owner.
- Rode `node scripts/verify-nex-identity-lock.mjs` e registre a versão do lock em toda entrega visual. Esse teste verifica integridade e não substitui inspeção de semelhança visual.

## 8. NEX — entrega conjunta obrigatória HUD + Site

A decisão final do usuário é **não inserir NEX na página pública até o pacote completo estar pronto nas duas superfícies**. Antes de alterar o mascote, leia [plano completo](docs/design/nex/COMPLETE_DELIVERY_PLAN.md) e [gates rastreáveis](docs/design/nex/RELEASE_READINESS.json).

**Proibido:** publicar Hero parcial, chamar SVG genérico de master aprovado, declarar giro 360° enquanto faltarem vistas reais, substituir o HUD por sprite não homologado ou alterar `status` para READY sem 11 evidências e aprovação explícita do owner no SHA exato.

Execute `node scripts/verify-nex-release-readiness.mjs` para verificar consistência; `--require-ready` deve falhar enquanto restarem pendências. Essa política vale também para o repositório Nexus-Site; a homepage deve permanecer sem NEX enquanto o gate for BLOCKED.
