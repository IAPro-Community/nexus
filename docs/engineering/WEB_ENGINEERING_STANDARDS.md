# Web Engineering Standards — IAPro Nexus

## Overview

This specification establishes the permanent frontend engineering baseline, architectural conventions, quality gates, and accessibility standards for the IAPro Nexus web platform.

The primary goal is **long-term maintainability, accessibility, predictability, and safety for both human engineers and AI coding agents**.

---

## 1. Core Principles & Priority Hierarchy

When engineering trade-offs occur, decisions must strictly follow this order of priority:

```text
Correctness
    ↓
Security
    ↓
Accessibility (WCAG 2.2 AA)
    ↓
Maintainability
    ↓
Readability & Ergonomics
    ↓
Testability
    ↓
Consistency
    ↓
Performance (measured)
    ↓
Aesthetics
```

* **Readability over cleverness**: Code is written primarily for the next engineer who needs to understand and debug it.
* **No premature micro-optimizations**: Avoid superstitious memoization or complex abstractions without profiling evidence.
* **No escape hatches**: `eslint-disable`, `@ts-ignore`, `@ts-nocheck`, unconstrained `any`, and `!important` are forbidden as convenience shortcuts.

---

## 2. Component Architecture & Reusability Hierarchy

Before creating any new UI component, hook, or utility, follow the **Reusability Decision Hierarchy**:

```text
1. Existing Project Component (e.g., src/design-system/primitives/)
      ↓
2. Existing Design System Primitive
      ↓
3. Installed Accessible Primitive (@radix-ui/react-*)
      ↓
4. Composition of Existing Primitives
      ↓
5. New Implementation (requires documented rationale)
```

### Build vs. Buy Rules for Complex Primitives
For behaviorally complex primitives (Dialog, Select, Combobox, Dropdown, Menu, Tooltip, Popover, Tabs, Accordion, Virtualization, Focus Trap), use established and accessible primitives (`@radix-ui/react-*`, `@floating-ui`, etc.). Never reinvent focus management, ARIA state coordination, or keyboard traps from scratch.

### Single Responsibility & Component Colocation
* **One conceptual component per file**: Avoid bundling multiple distinct components in a single file (`UserCard.tsx`, `UserAvatar.tsx`, `UserActions.tsx`).
* **Colocation of concerns**: Keep unit tests and styles next to the component file:
  ```text
  features/agents/
  ├── AgentCard.tsx
  ├── AgentCard.test.tsx
  └── AgentCard.module.scss
  ```
* **File size heuristics**:
  - `~200 lines`: Review responsibilities.
  - `~300 lines`: Strong candidate for decomposition.
  - `>400 lines`: Requires explicit architectural justification.

### Public Boundaries & Domain Isolation
* `src/shared/` contains only domain-agnostic utilities and design system primitives. `shared/` must never import from `features/`.
* Features must never import internals of other features; communicate via public APIs (`index.ts`) or application events.

---

## 3. Semantic HTML & Native Controls First

Web accessibility begins with native semantic HTML.

```text
Native Semantic HTML  >  HTML + CSS  >  HTML + ARIA  >  Custom ARIA Widget
```

### Mandatory Semantics:
* **Interactive elements**: Use `<button type="button">` or `<button type="submit">` for actions and `<a>` for navigation. Never attach `onClick` handlers to `<div>`, `<span>`, or `<li>` when a native control exists.
* **Landmarks**: Structure interfaces with `<header>`, `<nav>`, `<main>`, `<section>`, `<aside>`, and `<footer>`.
* **Form controls**: Always pair `<label>` with `<input>`, `<select>`, `<textarea>`, and wrap grouped inputs in `<fieldset>` with `<legend>`. Placeholders are visual hints, not labels.
* **ARIA discipline**: ARIA complements semantic HTML; it does not replace it. Redundant or inaccurate ARIA attributes that duplicate native semantics are prohibited.

---

## 4. Accessibility Baseline (WCAG 2.2 AA)

All features must comply with WCAG 2.2 AA standards:
1. **Full Keyboard Navigation**: Every action and interactive element must be operable via keyboard (`Tab`, `Shift+Tab`, `Enter`, `Space`, `Escape`, arrow keys where applicable).
2. **Visible Focus**: Never remove focus rings (`outline: none` or `outline: 0`) without providing a distinct, high-contrast replacement ring (`:focus-visible`).
3. **Contrast Compliance**: Normal text must achieve a contrast ratio >= 4.5:1; UI controls, badges, and large text must achieve >= 3.0:1. Contrast must be automated across all theme presets.
4. **Accessible Names**: All icon-only buttons (`IconButton`) must supply an accessible name via `label` or `aria-label`.
5. **Reduced Motion**: Respect user OS preferences via `@media (prefers-reduced-motion: reduce)` and `data-reduced-motion="true"`.
6. **Responsive by default**: Desktop compactness must not be copied onto phones. Laptop density rules must be gated with `min-width: 821px`. Phone and coarse-pointer layouts use `--nx-touch-min` (44px), `--nx-safe-*` from `env(safe-area-inset-*)`, and a single page scroll — nested scroll regions are a last resort. Core tasks must work at 390px without horizontal page scroll. Hide chrome into an overflow menu rather than deleting desktop actions.

---

## 5. Styling Architecture: SCSS Modules & Design Tokens

The official frontend styling architecture is:

```text
SCSS Modules
+
CSS Custom Properties (Runtime Themes & Metrics)
+
Semantic Design Tokens
```

### Authority of CSS Custom Properties
CSS Custom Properties (`var(--nx-*)`) are the authoritative runtime source for:
* Themes and color palettes (`--nx-surface`, `--nx-text`, `--nx-accent`, `--nx-border`, etc.)
* Layout metrics (`--nx-topbar`, `--nx-rail`, `--nx-taskbar`, `--nx-density-*`)
* Typography, border radiuses, shadows, and z-index layers.

### Strict Rules for Styles:
1. **SCSS Modules for Component Styling**: New component styles must use `ComponentName.module.scss`. Do not create local unencapsulated `.css` files.
2. **Zero Unjustified Inline CSS**: Static presentation styles must reside in SCSS modules or global stylesheets.
3. **Inline Style Exception**: Inline `style={{ ... }}` is permitted *only* for dynamic runtime variables (e.g., coordinates, pane dimensions, CSS variable overrides like `style={{ '--panel-width': `${width}px` }}`).
4. **No `!important`**: Overriding CSS specificity with `!important` is forbidden and enforced via Stylelint (`declaration-no-important: true`).
5. **No Hardcoded Hex/RGB in Components**: Components must consume semantic design tokens rather than arbitrary raw colors (`#ffffff`, `#1a1a1a`).
6. **Modern SCSS/CSS**: Use `@use` and `@forward` instead of deprecated `@import`. Limit selector nesting to <= 3 levels.
7. **Global Styles Allowlist**: Global stylesheets are restricted to `src/index.css`, `src/app/workspace-os.css` (OS core foundation), and `src/styles/*.scss`. Other `.css` files are rejected by `npm run check:styles`.

---

## 6. Internationalization (i18n)

* **Zero Hardcoded UI Strings**: All visible user text, titles, button labels, placeholders, tooltips, validation messages, and empty state copies must be localized via `react-i18next`.
* **Semantic Keys**: Use dot-notated hierarchical keys (`feature.action.label`, `dialog.confirm.title`) rather than raw sentences as keys.
* **Numbers, Dates, and Currencies**: Format dates, relative timestamps, and numbers using `Intl.DateTimeFormat`, `Intl.NumberFormat`, and `Intl.RelativeTimeFormat`.

---

## 7. React & TypeScript Best Practices

### Pure Components & State Discipline
* **Pure rendering**: Components and custom hooks must remain pure functions during render. No side-effects during render.
* **Colocate State**: Keep state as close as possible to the component that owns it. Do not promote local state to global stores prematurely.
* **Avoid `useEffect` for derived values**: If a value can be computed from props or state during render, compute it directly.
* **Stable Keys**: Use unique entity IDs for list items; never use array index when items can be filtered, reordered, or deleted.

### Strict TypeScript
* `strict: true` is enforced across the entire codebase.
* Explicit type unions and discriminated unions must be used for state machines and API models.
* APIs returning nullable collections must be normalized at the boundary using `asArray()` / `(val || [])` to prevent `TypeError: Cannot read properties of null (reading 'length')`.

---

## 8. Quality Gates & Automated Verification

### Local Quality Commands
| Command | Action |
| :--- | :--- |
| `npm --prefix web run check:styles` | Validates stylesheet architecture against global allowlist |
| `npm --prefix web run lint:styles` | Stylelint validation for SCSS & CSS rules |
| `npm --prefix web run quality` | Runs check:styles, format check, ESLint, Stylelint, TypeScript check, and Vitest suite |
| `npm --prefix web run quality:full` | Runs `quality` plus full production build and bundle sync check |
| `make quality` | Comprehensive repository gate (frontend + Go tests + linters) |
| `make web-verify` | Frontend gate report generator (`DEV/validation/FRONTEND_LATEST.md`) |

---

## 9. Exceptions Protocol

When an engineering standard cannot be followed due to an external browser limitation or platform constraint, it must be formally documented with:
1. The specific rule affected.
2. The concrete technical limitation.
3. The temporary mitigation adopted.
4. The plan and ticket for technical debt elimination.

Convenience, speed, or AI generator preference are **never** valid justifications for an exception.
