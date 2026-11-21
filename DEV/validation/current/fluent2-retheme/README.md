# Fluent 2 visual migration

## Discovery

- Requested `ui-ux-pro-max` was unavailable in the session catalog and local project/profile skill directories. The applicable fallback workflow was `skill-frontend-excellence`.
- Intent: `MIGRATE + PRESERVE + NONE`. The classifier returned `IMPROVE` because it does not recognize “retheme”; the explicit Fluent 2 target and instruction to reuse the existing theme system make this a visual-system migration.
- Existing system resolved to the local Nexus design system at `web/src/design-system` (0.95 confidence from 11 imports). No separately versioned theme package or Design Profile exists.
- Fluent 2 is applied through the existing Nexus semantic tokens. No Fluent UI dependency or parallel theme system was added. The approach follows Fluent 2's semantic neutral/brand/status tokens and light/dark variants ([design tokens](https://fluent2.microsoft.design/design-tokens/), [color tokens](https://fluent2.microsoft.design/color-tokens), [typography](https://fluent2.microsoft.design/typography/)).
- Framework versions, routes, principal surfaces, and structured discovery are recorded in `design-system-discovery.json`.

## Scope and changes

This is a presentation-only change. Component structure, routes, behavior, business logic, user theme controls, and all registered theme presets remain in place. No feature announcement or tour was added. NEX identity assets and specialized terminal/code colors were preserved.

The existing semantic theme now uses Fluent-like neutral surfaces, blue brand states, Segoe/system typography, compact control geometry, and shared focus/elevation/scrim tokens. Caption typography and scrim tokens were added where the old system had no semantic alias.

Changed files are grouped as follows:

- Theme sources: `web/src/app/workspace-styles/01-foundation.css`, `web/src/styles/_tokens.scss`, `web/src/styles/_typography.scss`, and `web/src/design-system/theme/{theme.ts,themePresets.ts,theme.test.ts}`.
- Shared workspace styles: `03`, `04`, `07`, and `09`–`17` under `web/src/app/workspace-styles/`.
- Colocated component styles: Nexus app menus and updates, Agents, Browser, Outputs, Settings, Usage, Voice, Composer/tours, Maestro previews, and Project Hub/Work surfaces.
- Existing visual test harnesses: `web/scripts/e2e-composer-themes.mjs` and `web/scripts/e2e-hardening-verify.mjs`.

Responsive review exposed an invalid `:global()` selector in the Project Hub SCSS Module; its short-viewport rule did not hide the hero mark. It now uses a local module class. The Composer read-only fallback also used a flex row for headings and copy; its sections now stack content and its action row wraps on narrow screens. Both fixes keep the existing component tree and behavior.

## Verification evidence

- Baseline captures: `.tempmediaStorage/fluent2-retheme/baseline` (60 PNGs: 20 presets at 390, 768, and 1440 px).
- Final captures: `.tempmediaStorage/fluent2-retheme/after` (60 PNGs with the same matrix). Dark, light, and high-contrast desktop/mobile captures were inspected.
- `npm --prefix web run quality:full`: passed style architecture/allowlist, Prettier, Stylelint, TypeScript, 151 test files / 959 tests, and production build. ESLint had no errors and reported one existing warning at `web/src/features/work/FlowRunSurface.tsx:122` (`plan.current_revision` missing from a `useCallback` dependency list).
- `npm --prefix web run test:e2e:composer-themes`: passed all 20 registered presets at 390, 768, and 1440 px; persistence, responsive Composer bounds, serious/critical Axe checks, and browser/console checks passed.
- `npm --prefix web run test:e2e`: passed Project Hub at 775×483, breakpoints 320×568, 390×844, 768×1024, 1024×768, 1280×800, 1366×768, and 1440×900, shared Radix menu semantics/Escape, keyboard focus and arrow navigation, settings density, and system light/dark switching.
- Theme unit tests passed automated contrast assertions across registered presets and dark/light/high-contrast schemes. `git diff --check` passed.
- Validation ran on the uncommitted working tree based on `release/v1.0.0` HEAD `6c322427d92dd810456d2b267546a1a907ac1603`. It is not a commit, visual homologation, or publication record.
- The Composer theme E2E had stale accordion labels, an assumption that finish followed palette selection, and stale high-contrast expectations. Its selectors and setup now match the existing UI.

## Remaining hard-coded values

- Literal colors remain in `01-foundation.css` and `themePresets.ts` as canonical values behind semantic theme tokens.
- NEX/HUD components and sprites retain identity, mood/status, and pixel-art colors under the NEX visual contract.
- `web/src/styles/_tokens.scss` retains terminal ANSI colors for code/output fidelity.
- Generic modal and drawer scrims use `--nx-scrim-*` tokens. The last two splash literals were tokenized in the follow-up wave (`--nx-shadow-modal` + the accent glow layer, and `--nx-drop-shadow-overlay` for the logo).
- Fixed component measurements (for example widths, heights, spacing, font sizes, radii, and one-pixel borders) remain across 101 stylesheet files matching the source scan. This pass tokenized theme-facing values in the shared shell and touched feature surfaces; it did not normalize every static layout measurement in every feature module. Those remaining values are tracked by `web/scripts/check-token-parity.mjs` and locked in `web/scripts/token-parity-baseline.json`.

## Follow-up wave: hierarchy, density and parity gate

Applied after the anchor retheme commit (`a6bbb95d`), on the same release branch.

- Radius mapping outliers were normalized to the canonical scale so a single literal never renders two radii: 7 sites of `6px` tightened to `--nx-radius-sm` (4px), 3 sites of `8px` unified on `--nx-radius-md` (6px), and 1 site of `10px` unified on `--nx-radius-lg` (8px). A stricter per-hunk pairing of the anchor diff showed 11 real outliers; the previously estimated four font-size outliers were false positives of the aggregate method, because every font-size literal already mapped to exactly one token (`13.5px` already resolves to `--nx-font-size-sm`).
- Elevation hierarchy restored: `--nx-elevation-overlay` / `--nx-elevation-modal` now sit above `--nx-elevation-3` with the pre-retheme depth of toasts, menus and dialogs, mirrored in `--nx-shadow-overlay` / `--nx-shadow-modal` aliases for SCSS modules. Horizontal drawer depth returned as `--nx-shadow-drawer-left` / `--nx-shadow-drawer-right` (±8px x-offset). `--nx-shadow-md` / `--nx-shadow-lg` fallbacks now match `--nx-elevation-2` / `-3` layer by layer. High contrast still inherits the dark elevation values, exactly as `--nx-elevation-1..3` already did.
- `--nx-control-min-height` was removed; `.nx-segmented__item` now uses `max(var(--nx-touch-min), var(--nx-density-btn-h, 28px))`, keeping compact at the 32px touch minimum while comfortable grows back to 36px.
- `--nx-space-7` was defined (it was referenced by Composer/Outputs `clamp()` padding without a value).
- The parity gate `web/scripts/check-token-parity.mjs` now runs in `check:styles`. It is property-aware (spacing/radius/font-size), value-exact, and compares literal counts per `file|property|value` against `token-parity-baseline.json`; only regressions fail. A value-identical mechanical substitution applied 516 literals across the retheme-touched files. `scripts/build.mjs` now fails when the bundle is missing either half of the token contract.
- Settings density E2E now measures the segmented control itself: comfortable resolves to 36px min-height and compact to 32px.

## Dedup wave: single definition per token

Applied after the follow-up wave, on the same release branch.

- Every duplicate token *definition* was removed, keeping one owner and one effective value per name:
  - `--nx-focus-ring` existed twice with divergent values (`--nx-accent` in the foundation, `--nx-focus` in the SCSS layer). The foundation keeps it, now resolving through `var(--nx-focus, var(--nx-accent))`, which reproduces the value the bundle already shipped.
  - `--nx-space-2/3/4` were declared identically in both layers; the foundation owns them.
  - `--nx-font-mono` was declared twice with different stacks. The foundation declaration was dead (the typography layer always won later in the bundle), so it was dropped and the typography stack is now the only one.
- The `:root` blocks of `_tokens.scss` and `_typography.scss` were re-emitted by every SCSS module that `@use`s them, because `scopeCssModule` scopes class selectors but not `:root`. The bundle carried 13 copies of the token block and 12 copies of the typography block (~33 KB). Both partials now expose a `root` mixin that `globals.scss` includes once, so the blocks ship exactly once and modules keep consuming them through `var(--nx-*)` at runtime.
- Measured effect: `dist/bundle.css` went from 523,986 to 492,871 bytes, and `--nx-focus-ring`, `--nx-font-mono`, `--nx-space-2`, `--nx-radius-md` and `--nx-font-size-sm` each appear exactly once in the shipped bundle.
- Verification: `npm --prefix web run quality:full` (stylesheet architecture, workspace CSS manifest, token parity 485/0, 151 files / 959 tests, build) and `npm --prefix web run test:e2e` both green; `make install-local` installed the CLI, and `make build-desktop build-hud` rebuilt the two native surfaces so their embedded assets carry the same single-copy bundle.
- Remaining naming overlap, deliberately not touched in this wave: `--nx-radius` (6px) vs the `--nx-radius-md` alias, and `--nx-font` vs `--nx-font-sans` (same stack, two names).

## Onda 3 (deferred)

The remaining `workspace-styles` sections are still global CSS. Migrating them to SCSS Modules (suggested order `16-splash-and-settings` → `15-context-and-responsive` → `09-project-management`, then updating `check-workspace-styles.mjs` and the allowlist) stays open as a separate batch.
