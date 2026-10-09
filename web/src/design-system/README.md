# Nexus Design System

The Nexus UI is intentionally built as a thin product design system over headless and
specialized primitives. Features should reuse this layer instead of rebuilding interaction,
accessibility, focus management, portals, menus, dialogs, or theme behavior.

## Layers

```text
Product feature
  -> Nexus conceptual component / primitive
    -> Radix UI, assistant-ui, XYFlow, xterm, Sonner
      -> Nexus semantic tokens
        -> active theme preset
```

## Rules

1. Prefer an existing export from `web/src/design-system`.
2. Do not branch feature markup by active theme. Themes change semantic tokens only.
3. Do not hardcode arbitrary visual colors in feature CSS/TSX. Use `--nx-*` tokens.
4. Behavior-heavy controls should be backed by an existing headless primitive when one is
   already available in the project.
5. Add a new conceptual component when a pattern is reused by multiple product surfaces or
   when it encapsulates meaningful product semantics. Do not create wrappers solely to move
   markup from one file to another.
6. Preserve semantic status meaning across themes: success, warning, danger and info are not
   decorative palette choices.

## Current primitives

- Button / IconButton
- Badge
- Card
- Input / Textarea / Select
- Switch
- Segmented
- ChoiceGroup / ChoiceOption
- Progress / Spinner / EmptyState / InlineAlert
- Dialog / ConfirmDialog
- Tooltip
- DropdownMenu
- ContextDrawer / ContextMenu
- SearchInput

`ChoiceGroup` and `ChoiceOption` are the shared selection primitive for rich radio-card
interfaces. They provide common semantics and arrow/Home/End keyboard navigation and are used
by theme selection, provider selection and Direct account selection.

## Headless foundations already in use

- Radix UI: dialogs, alert dialogs, dropdown menus and tooltips.
- assistant-ui: conversational surfaces.
- XYFlow: visual flow/canvas behavior.
- xterm: terminal emulation.
- Sonner: toast notifications.

Prefer extending these foundations when a behavior already exists there. Avoid adding another
complete design system (for example MUI, Chakra or a parallel shadcn token layer) to the 1.0
line.

## Themes

Theme definitions live under `design-system/theme`. Component CSS consumes semantic
variables and never theme IDs. See `docs/product/themes.md` for the preset contract and
validation requirements.
