# Nex 2.0 — Implementation Progress Tracker

- **Epic:** Nexus #232
- **Authoritative Base:** `origin/release/v1.0.0` (`8d1f1592ce8c4b12afa66585232ff2d09bd41d93`)
- **Worktree:** `/projetos/worktrees/nex-2-foundation`

| Phase | Name | Status | Commit SHA | Key Checks & Results | Blockers / Notes | Next Task |
|---|---|---|---|---|---|---|
| **P0 (NEX-00)** | Source Reconciliation, Baseline & Default-OFF Flag | **COMPLETE (Linux)** | `pending-commit` | Go baseline & new flag tests PASS (`internal/desktop`, `cmd/nexus-hud`, `internal/nexus`, `internal/control/web`, `internal/nexus/store`); Web `typecheck` & HUD Vitest (16 files / 97 tests) + node-tests (8/8) PASS; `verify-nex-identity-lock.mjs` PASS (`FROZEN`); `verify-nex-release-readiness.mjs` BLOCKED (2/11); Baseline `e2e-hud` & `e2e-companion-lab` FAIL due to `COMPOSER_FLOW_ENABLED` gating `getAttention()` | Native Windows & macOS hardware `UNVERIFIED` / `BLOCKED` on this Linux host; `e2e-hud` / `e2e-companion-lab` regression documented | Hand back for Gate G0 review, then **P1 (NEX-01): Desktop Lab & Renderer ADR** |
| **P1 (NEX-01)** | Desktop Lab and Renderer ADR | PENDING | — | — | Depends on G0 | Minimal Wails v3 lightweight entrypoint + motion/bounds contract + ADR |
| **P2 (NEX-02)** | Presence Shell and UX | PENDING | — | — | Depends on G1 | — |
| **P3 (NEX-03)** | Guardian: Operational Parity & Model Watch | PENDING | — | — | Depends on G2 | — |
| **P4 (NEX-04)** | Assistant: Durable Intelligence & Memory | PENDING | — | — | Depends on G3 | — |
| **P5 (NEX-05)** | Developer Tools & Delegation | PENDING | — | — | Depends on G4 | — |
| **P6 (NEX-06)** | Personal & Voice | PENDING | — | — | Depends on G5 | — |
| **P7 (NEX-07)** | Continuous Proactivity & Overnight Workflow | PENDING | — | — | Depends on G6 | — |
| **P8 (NEX-08)** | Release, Migration & Decommission Gate | PENDING | — | — | Depends on G7 | — |
