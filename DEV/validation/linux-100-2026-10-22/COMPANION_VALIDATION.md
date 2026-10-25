# Companion 2D validation

## Result

`NO-GO — COMPANION_2D_NOT_INTEGRATED`

The release branch at `f62fdec0c571ac6a19a904164d3f580799da6bf3` contains and renders the older `GuardianMascot` through `guardianModel`. It does not contain `NexusGuardian.tsx`, `companionModel`, a Companion sprite atlas, or the visual animation assets from PR #55. The Mascot selection in this source selects the older Guardian and cannot be counted as Companion evidence.

## Source and remote evidence

- Exact-SHA path/reference search: `logs/companion-source-audit.log`.
- PR #55 remains open and draft, targets `release/v1.0.0`, and its head is `91d25969bda4c8b0bc6903d577a6e7215947a950`; GitHub reports it cannot merge cleanly. Details: [PR #55](https://github.com/IAPro-Community/nexus/pull/55).
- No merge or cherry-pick was performed. The separate release source was checked directly.
- The older Guardian has states for idle, working, warning, and offline; this does not establish the requested Companion's idle, working, attention, critical, and offline behavior.

## Native exercise

The built Desktop and linked HUD started, but Companion activation, sprite animation, attention capsule, transparency, click-through, reduced motion, and persistence could not be exercised because the 2D Companion code is absent from the release build. No Companion screenshot was captured or substituted with an old Guardian image. Native screenshot restrictions are documented in `screenshots/README.md`.

## Required release gate

This finding is release-blocking. The correct status is `COMPANION_2D_NOT_INTEGRATED`; it is not `COMPANION_2D_NOT_RENDERING` because the new implementation is absent, and it is not merely an interaction-verification block.

