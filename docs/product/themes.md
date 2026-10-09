# Nexus Theme System 1.0

The Nexus 1.0 theme system is token-driven. Themes change visual semantics without changing
feature layout or introducing theme-specific component trees.

## Architecture

```text
Feature components
  -> Nexus design-system components
    -> headless/specialized primitives (Radix, xterm, XYFlow, assistant-ui)
      -> Nexus semantic CSS variables
        -> Theme preset
```

Feature code should consume Nexus components and semantic variables. It must not branch on
the active theme or hardcode a preset's colors.

## Component policy

Prefer an existing Nexus design-system component before creating local interactive markup.
For behavior-heavy primitives, prefer the libraries already used by Nexus, especially Radix
UI, while keeping Nexus styling and tokens as the visual source of truth.

A new conceptual Nexus component should normally be introduced only when a pattern is
reused across multiple product surfaces or encapsulates meaningful product semantics.
Theme-specific components are not allowed.

Current design-system primitives include Button, IconButton, Badge, Card, Input, Textarea,
Select, Switch, Segmented, Progress, Spinner, EmptyState, InlineAlert, SearchInput, Dialog,
ConfirmDialog, Tooltip, DropdownMenu, ContextDrawer and ContextMenu.

## Presets

The 1.0 registry contains the existing presets plus the approved visual set:

- Graphite Focus — neutral, productive dark theme.
- Vasco Noir — charcoal/black with restrained red accents; no pink/magenta wash.
- Nord Light — low-noise light theme.
- Aurora Ops — dark technical theme with controlled cyan accents.
- Monokai Dark — the existing `monokai` preset refined for calmer UI accents. The stable ID
  remains `monokai` so persisted preferences do not break.
- Obsidian — low-glare graphite theme with restrained violet-blue accents.

Existing presets remain available for compatibility.

### Personalização de fonte e ícones

A seleção da paleta e do acabamento é independente das preferências de aparência:

- **Fonte da interface:** `default` (tipografia Nexus atual), `system` (fonte do sistema operacional) ou `readable` (Verdana/Trebuchet com fallbacks locais). São stacks de fontes locais; nenhuma fonte é descarregada de terceiros nem embutida sem licença. A escolha aplica-se aos rótulos, menus e painéis, mas **não muda as células nem as métricas do xterm**.
- **Traço dos ícones:** `default`, `light` (1,5) e `bold` (2,5) aplicam espessuras via CSS à biblioteca Lucide já instalada. **Não representam packs diferentes ou variantes preenchidas**; isso demandaria um catálogo e mapeamento de ícones independente.
- O mesmo `iapro:nexus:theme:v1` guarda preferências validadas: se o usuário instalou uma versão antiga, campos ausentes voltam para `default`. Mudanças de tema não sobrescrevem fonte ou traço. O usuário pode restaurar ambos aos defaults.
- Os controles usam `Segmented`/ChoiceGroup acessíveis já existentes e permanecem fora da renderização do terminal. A alteração de fonte precisa de verificação visual em resoluções menores.

### Ember Studio — inspiração Sublime Text

- Novo preset `ember-studio` de aparência escura, carvão/grafite e acentos âmbar suaves. É um **tema Nexus original**, não uma cópia, distribuição ou marca oficial do Sublime Text.
- O Sublime separa temas de UI e esquemas de sintaxe, conforme [documentação oficial](https://www.sublimetext.com/docs/themes.html) e [documentação de esquemas](https://www.sublimetext.com/docs/color_schemes.html). Nosso preset atua somente em tokens de **interface**, e não tenta alterar syntax highlighting de xterm ou código. O tema `monokai` já existente permanece disponível.
- `Ember Studio` reaproveita o acabamento `flat` com superfícies sólidas, contraste WCAG unitário e a categoria Clássicos do Código.

### Degradê Suave — Nexus Dusk

- **Nexus Dusk** (`nexus-dusk`) — paleta grafite neutra com quase-preto, uma
  variação sutil de profundidade nos painéis e um acento azul acinzentado.
- `finish: 'subtle-gradient'` ativa um único conjunto de regras CSS
  (`data-finish="subtle-gradient"`) em barras e áreas do workspace;
  o degradê usa pequenas diferenças de luminância, sem neon nem brilho
  forte. Outros 18 temas não herdam o efeito.
- Referência analisada: [Orca da stablyai](https://github.com/stablyai/orca/blob/main/src/renderer/src/assets/main.css),
  cuja interface escura utiliza `#0a0a0a` como background, `#171717` em painéis,
  `#262626` em superfícies secundárias e bordas pouco opacas.
  **Nexus Dusk é um tema original do Nexus, inspirado nessa linguagem visual,
  não um tema oficial importado nem uma reprodução pixel a pixel.**

As variáveis `--nx-*`, o renderer do Nexus e a paleta ANSI do xterm
permanecem independentes. Títulos personalizados/IA e cores de estados
operacionais não são apagados por esse tratamento. Seleção, persistência,
layout, WCAG e Axe seguem o mesmo contrato dos demais presets.

### Minimal / Flat (candidate for beta.10)

- **Slate Flat** (`slate-flat`) — charcoal/slate dark, desaturated teal accent and solid surfaces.
- **Paper Flat** (`paper-flat`) — cool off-white, subtle separators and deep teal accent.
- **Clay Flat** (`clay-flat`) — warm off-white, warm gray hierarchy and terracotta accent.

These presets use `finish: 'flat'` in the existing registry. `ThemeProvider` resolves
it to `data-finish="flat"` and a **shared** CSS override removes nonessential
gradients, bloom and heavyweight elevation on selected workspace surfaces.
Legacy presets remain `data-finish="standard"`; high-contrast schemes always
use the standard finish to preserve their existing accessibility contract.

The design is intentionally **opt-in**: no change to the default Nexus Dark,
the theme storage key, terminal ANSI/palette, custom window title logic or
visual hierarchy outside the three new presets.


## Theme contract

Theme presets own visual variables such as:

- application and elevated backgrounds;
- surfaces;
- borders;
- primary, secondary and muted text;
- primary accent and its hover/soft variants.

Operational semantics remain global. Success, warning, danger and info must retain their
meaning across themes instead of being reinterpreted to match a palette.

## Persistence

Theme preferences continue to use `iapro:nexus:theme:v1`. The key must remain stable during
the 1.0 line unless an explicit migration is introduced.

## Validation

Changes to the theme registry must preserve:

1. Type-safe preset IDs derived from the registry.
2. WCAG AA text contrast and >= 3:1 accent contrast checks.
3. Runtime selection and persistence after reload.
4. Composer visual coverage on desktop, tablet and mobile.
5. Axe checks with no serious or critical violations.
