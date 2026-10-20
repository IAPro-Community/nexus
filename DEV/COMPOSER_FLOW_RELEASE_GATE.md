# Composer / Flow: experimental visibility gate (Nexus 1.0.0)

Composer and Flow Studio are **not approved for the public 1.0.0 UI**.
Normal builds hide their project navigation, command palette entries, overview
actions, presets, and restored window tabs. Direct project deep links and
popout links are redirected to Overview. The workspace also rejects attempts
to open these surfaces programmatically.

The underlying Composer sessions, WorkPlans, MissionRuns, API routes and
execution engine are **not deleted or stopped**. This is a frontend release
gate, not a backend authorization or security boundary.

## Stable release (default)

```sh
cd web
npm run build
```

The build embeds `__NEXUS_EXPERIMENTAL_COMPOSER_FLOW__ = false`.

## Developer / homologation build (explicit opt-in)

POSIX shell:

```sh
cd web
NEXUS_EXPERIMENTAL_COMPOSER_FLOW=1 npm run build
```

PowerShell:

```powershell
cd web
$env:NEXUS_EXPERIMENTAL_COMPOSER_FLOW = '1'
npm run build
Remove-Item Env:NEXUS_EXPERIMENTAL_COMPOSER_FLOW
```

The flag is consumed **at build time** by `web/scripts/build.mjs`. It is not
a per-user toggle, browser URL parameter or localStorage flag. The embedded
web assets for desktop/server must be rebuilt for this change to take effect.
Do not publish opt-in build artifacts as the stable 1.0.0 release.

## Validation checklist

- Build without flag: Composer and Flow absent in rail, Overview, commands,
  saved layouts and Flow-specific layout preset
- Open `/p/<project>/work`, `/p/<project>/missions`, run deep link
  and popout: no experimental UI is rendered
- Existing layouts with non-closable Composer tabs recover to stable tabs
- Build with flag: Composer and Flow navigation, drafts and history reappear
- Verify terminals, agents, direct sessions, Maestro, resources, settings and
  HUD remain functional
- Confirm no backend data or mission state is deleted or stopped

The public release should still be tested on Linux and Windows. Unit tests
are in `web/src/app/composerFlowFlag.test.ts`.
