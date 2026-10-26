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
