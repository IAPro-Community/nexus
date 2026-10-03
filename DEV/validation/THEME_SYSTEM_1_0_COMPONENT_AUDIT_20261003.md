# Theme System 1.0 — Component and Token Audit

Date: 2026-10-03  
Branch: `release/v1.0.0`

## Scope

This audit intentionally avoids redesigning Nexus. The goal is to keep the existing 1.0
product structure while making themes and interactive components more maintainable.

## Resolved

### Shared rich-choice behavior

Theme presets, provider selection and Direct account selection previously implemented their
own radio-card behavior. They now reuse `ChoiceGroup` and `ChoiceOption` from the Nexus
design system.

The shared contract provides:

- `radiogroup` / `radio` semantics;
- Arrow, Home and End keyboard navigation;
- selection through the same interaction path used by pointer input;
- one styling hook for rich selectable cards.

`Segmented` is composed from the same primitives without changing its public API.

### Existing headless primitives reused

Two menus that duplicated portal, outside-click, Escape and positioning behavior were moved
to the Radix DropdownMenu dependency already present in Nexus:

- project create menu;
- workspace arrange-layout menu.

No additional design-system library was introduced.

### Theme token cleanup

The affected 1.0 surfaces now consume semantic tokens instead of literal palette values:

- shell status and notification colors;
- pinned workspace-tab accents;
- splash screen;
- session event tones;
- overview runtime status;
- dashboard accent contrast;
- terminal chrome and overlays;
- composer/conversation status colors;
- role selector and settings theme UI.

Terminal rendering remains intentionally specialized. Xterm keeps a stable terminal palette,
while terminal chrome consumes `--nx-terminal-*` semantic tokens. A light application theme
therefore does not accidentally turn the terminal canvas light.

### Accent-filled control contrast

`--nx-accent-contrast` is derived at runtime from the effective accent color and resolves to
black or white based on WCAG contrast. This also covers manual accent overrides instead of
requiring per-theme exceptions.

Tests now assert >= 4.5:1 for accent-filled control text across every registered preset and
for custom accents in dark/light schemes.

## Intentionally deferred from 1.0

These are not considered blockers:

- `ContextMenu` remains a Nexus primitive with its current centralized implementation.
  Replacing it would require introducing another Radix package or changing point-positioned
  context-menu behavior.
- `Switch` remains centralized in the design system. There is no duplicated feature-level
  switch behavior to justify dependency churn in the release line.
- the window chrome customization popover remains custom because it combines text editing,
  swatches, icon selection and point-triggered positioning; it should be migrated only with
  dedicated interaction regression coverage.
- xterm's internal ANSI palette is not coupled to UI themes. Terminal theming should be a
  separate explicit feature if introduced later.

## Validation performed in-repository

- registered theme palette contrast was recalculated for all 15 presets;
- all registered presets satisfy current text/background and accent/UI contrast gates;
- CSS/SCSS files touched by the audit have balanced declaration braces;
- hardcoded status-color fallbacks in the touched feature modules were removed;
- E2E hardening coverage was updated for shared radio-card keyboard navigation;
- theme E2E coverage includes Graphite Focus, Vasco Noir, Nord Light, Aurora Ops,
  Monokai Dark and Obsidian.

## External execution gate

CI execution is not available in this work session, so the following commands still need to
run in an environment with the repository dependencies installed:

```bash
cd web
npm run format:check
npm run lint
npm run lint:styles
npm run typecheck
npm run test
npm run build
npm run test:e2e:composer-themes
npm run test:e2e-hardening
```

A failure in these gates should be fixed without widening the scope into a visual redesign.
