# NEX-ID-v1 — Approval record

**Approved by:** maestro (IAPro Nexus owner), in-session instruction "aprovo os dois"
**Approved at:** 2026-11-12
**Scope:** NEX hero raster master + hud-32bit idle atlas (the two pending candidates)

## What was approved

1. **Hero raster master** (`commercial-polished` variant): layered OpenRaster master
   `docs/design/nex/hybrid-v1/nex_hybrid_master.ora` (SHA-256 `68a9b3b90722721bacd4067bac3e9d6560acf144898ae60c3d7968692c652f32`),
   reconstructed byte-for-byte from the six versioned base64 parts; source original
   `web/public/nex/hybrid-v1/nex_source_original.png` (SHA-256 `ef7a4aeb7b24d2ab5f199f913d351414b8fafb0b6208b3ab2b665013adcb9118`).
   Perceptual QA divergence of the blink reconstruction (0.787/255 RGB mean) was presented and accepted.
2. **hud-32bit idle atlas**: `docs/design/nex/experimental-atlas/nex-idle-atlas-128.png` (96 frames = 12 idle actions x 8),
   storyboard `nex-idle-12-storyboard.jpg` and pivot map `nex-idle-atlas.json`.

## Recorded divergence

The approved atlas is a **filtered pixelation derivative** of the approved raster, not hand-drawn
pixel clusters as originally stated in `RETRO_GAME_IDLE_DIRECTION.md`. The owner explicitly approved
this treatment on 2026-11-12; hand-drawn true-pixel refinement remains an optional follow-up and
does not block this lock. The 64px Compact HUD variant is NOT covered by this approval.

## Effect

- `identity-lock.json` status: CANDIDATE_PENDING_APPROVAL -> FROZEN (NEX-ID-v1).
- RELEASE_READINESS gates `master_source` and `site_hero` move to PASSED with this record as evidence.
- `hud_compact` stays BLOCKED until the 64px Compact variant exists.
