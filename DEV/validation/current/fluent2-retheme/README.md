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
- `16-splash-and-settings.css` retains two black shadow/drop-shadow values for the splash/brand glow. Generic modal and drawer scrims use `--nx-scrim-*` tokens.
- Fixed component measurements (for example widths, heights, spacing, font sizes, radii, and one-pixel borders) remain across 101 stylesheet files matching the source scan. This pass tokenized theme-facing values in the shared shell and touched feature surfaces; it did not normalize every static layout measurement in every feature module. Those remaining values are the main follow-up if the goal is to make all UI geometry adjustable through tokens.
