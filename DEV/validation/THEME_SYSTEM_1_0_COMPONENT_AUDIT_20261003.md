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

Tests assert >= 4.5:1 for accent-filled control text across every registered preset and for
custom accents in dark/light schemes. Accent hover states are covered by the same gate.

### Secondary, accent and status text contrast

The audit now protects text tokens as text, rather than treating every palette value as a
decorative accent:

- `--nx-muted` and `--nx-subtle` must stay >= 4.5:1 on app and panel surfaces;
- `--nx-accent-text` must stay >= 4.5:1 on app and panel surfaces;
- `success`, `warning`, `danger` and `info` text must stay >= 4.5:1 on the active surface;
- semantic status colors are resolved by dark/light/high-contrast scheme, with preset overrides
  still taking precedence when required;
- legacy Nexus Light, Midnight, Nord, Dracula and Solarized secondary text values were adjusted
  only where needed to satisfy the gate;
- High Contrast Light now owns explicit accessible status colors;
- Nord Light hover and Solarized accent-text states were corrected without changing layout.

### Choice-group keyboard contract

`ChoiceGroup` now uses a roving tabindex contract: one enabled radio per group participates
in the normal Tab sequence. Arrow keys, Home and End move and select within the active group.
Theme categories own separate radio groups so accordion headers are no longer descendants of a
single global `radiogroup`.

The browser hardening suite checks the roving-tabindex invariant and the migrated Radix menus
check open/close semantics plus Escape focus restoration.

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
- theme E2E coverage now exercises all 15 registered presets, including the approved
  Graphite Focus, Vasco Noir, Nord Light, Aurora Ops, Monokai Dark and Obsidian set; each
  preset is checked for selection, reload persistence, desktop/tablet/mobile layout and
  serious/critical Axe violations.

## External execution gate

The repository self-hosted `nexus-ci` runner is currently leaving the exact-SHA workflow
queued before `Sanitize self-hosted workspace`, so the following execution gates still need
to complete when that runner becomes available:

```bash
cd web
bun install --frozen-lockfile
bun run format:check
bun run typecheck
bun run lint
bun run lint:styles
bun run check:styles
bun run test
bun run build
bun run verify
bun run test:e2e
bun run test:e2e:composer-themes
```

A failure in these gates should be fixed without widening the scope into a visual redesign.
