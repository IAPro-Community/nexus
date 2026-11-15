# NEX-HUD-RIG-V1-RENDER — Approval record

**Approved by:** maestro (IAPro Nexus owner), in-session instruction **"aprovo"**
**Approved at:** 2026-11-15
**Scope:** runtime render of the NEX companion in the web HUD (`GuardianMascot`, 96px `guardianButton`) — swap from the filtered 128px atlas playback to the layered **rig-v1** procedural player.

## What was approved

1. **Rig-v1 as the HUD runtime render**: `nex-motion-core.mjs` + `nex-rig-player.mjs`
   (bytes from branch polish `b7415ba4edb9b3abc3fba822c616449d2d312841`, PRs #175/#178
   reconciled; SHA-256 registered in `NEX_VISUAL_MANIFEST.json → animation.runtime_render`).
   Procedural idle, real 4-module orbit, internal-layer-only motion.
2. **Not a new master**: the rig animates the **same approved NEX-ID-v1 hybrid-v1 layers**
   (`web/public/nex/hybrid-v1/`). The identity lock `NEX-ID-v1` is untouched — this is a
   runtime rendering change using already-approved assets, satisfying
   `runtime_replacement_allowed_only_after_master_review: true` (master review happened:
   visual comparison + explicit owner approval recorded here).
3. **Atlas remains first paint and permanent fallback**: `NexRigSprite` renders the locked
   `<NexPixelSprite>` until the rig reports ready; any load error swaps back to the atlas.
   jsdom/test environments render the atlas only (`GuardianMascot.test.tsx` unchanged).
4. **Reduced motion contract preserved**: the player honors `prefers-reduced-motion`
   internally by showing the byte-faithful `nex_source_original.png` as a static pose;
   the SCSS mood treatment stays predictable (no motion, color only).

## Comparison evidence (what the owner saw)

- Side-by-side mock at the real HUD size (96px) and zoom ×2.5:
  `DEV/validation/nex-rig-compare/hud-antes-depois.png` (working copy also at
  `/tmp/opencode/nex-compare/hud-antes-depois.png`)
- "Before": companion-lab captures at `38470c2c` in
  `DEV/validation/current/companion-lab/captures/` (atlas playback)
- "After": companion-lab re-capture at the integration SHA shows the rig states

## Honest deltas registered

- At 96px the two renders are close in composition; the rig is slightly larger and
  sharper, with orbital modules overflowing the circle (real depth).
- The decisive deltas are crispness at zoom/HiDPI (native layers vs 128px raster) and
  **live procedural motion** (idle + orbit vs baked 8-frame loops).
- **Not covered by this approval**: `hud_compact` 64px variant (still pending, gate
  unchanged), site hero (unchanged), retro HD editable master (still pending).

## Addendum — Expressions v1 (2026-11-15, mesma sessão)

**Approved by:** maestro, in-plan answer **"Aprovo tudo"** + **"Transbordar sutil"**
**Scope:** procedural expression layer on the rig player (`setExpression`, 9 expressions:
ambient/greet/look/thinking/attention/error/resolved/rest/still) + HUD fit allowing subtle
module overflow (~4–6px at rest, more on greet/resolved).

- Still choreography of the SAME approved hybrid-v1 layers; no new art, no new master.
- Movement remains internal-layers-only (anchor transform untouched); reduced motion and
  offline render one frozen representative frame (byte-stable), asserted by the lab.
- DOM hook `canvas[data-nex-expression]` gives tests/e2e an honest assertion surface.
- Rig module SHAs change; `NEX_VISUAL_MANIFEST.json → animation.runtime_render` must be
  updated in the same change, and `scripts/verify-nex-visual.mjs` keeps enforcing them.

## Guard updates (this commit, honest)

- `NEX_VISUAL_MANIFEST.json`: new `animation.runtime_render` block (module SHA-256,
  approval pointer, fallback policy). Atlas fields and SHA untouched.
- `scripts/verify-nex-visual.mjs`: verifies rig modules exist byte-faithful, that the
  rig marker only appears when the rig is ready, and that the atlas fallback stays
  wired with an onError path. The identity-level implementation check
  (`LOCKED_RASTER_ATLAS_FRAMES`) is unchanged.
- `docs/design/NEX_ANIMATION_CONTRACT.md`: rig row added (internal layers only,
  anchor transform untouched).
