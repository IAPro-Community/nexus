# Nex 2.0 — Implementation Progress Tracker

- **Epic:** Nexus #232
- **Authoritative Base:** `origin/release/v1.0.0` (`8d1f1592ce8c4b12afa66585232ff2d09bd41d93`)
- **Worktree:** `/projetos/worktrees/nex-2-foundation`

| Phase | Name | Status | Commit SHA | Key Checks & Results | Blockers / Notes | Next Task |
|---|---|---|---|---|---|---|
| **P0 (NEX-00)** | Source Reconciliation, Baseline & Default-OFF Flag | **COMPLETE (Linux)** | `fb22bbb7` | Go `-race` tests PASS; Web `typecheck` & HUD Vitest (16/16, 97 tests) + node-tests (8/8) PASS; `verify-nex-identity-lock.mjs` PASS (`FROZEN`); Baseline `e2e-hud` & `e2e-companion-lab` FAIL due to `COMPOSER_FLOW_ENABLED` gating `getAttention()` | Native Windows & macOS `UNVERIFIED` / `BLOCKED` | Completed |
| **P1 (NEX-01)** | Desktop Lab and Renderer ADR | **COMPLETE (Linux)** | `pending-commit` | Go geometry/DPI/window tests `-race` PASS; Web `check:styles`, `lint`, `lint:styles`, `typecheck`, `nex-v2` Vitest (3/3, 7 tests) PASS; Playwright P1 lab (`e2e-nex-v2-lab.mjs`) PASS (`155ms` paint, `16.8ms` p95, `0` clipping, `0` reduced-motion frames); Native Xvfb benchmark measured (`~311 MiB` PSS vs legacy `~447 MiB` PSS); `RENDERER_ADR.md` selects Wails v3 isolated entrypoint | Linux WebKitGTK memory exception documented (`> 60 MiB`); Windows/macOS native QA & owner final visual approval `BLOCKED` | **P2 (NEX-02): Presence Shell and UX** |
| **P2 (NEX-02)** | Presence Shell and UX | PENDING | — | — | Depends on G1 | Lifecycle supervisor, 3 surfaces, lazy panel, 100-cycle stability |
| **P3 (NEX-03)** | Guardian: Operational Parity & Model Watch | PENDING | — | — | Depends on G2 | — |
| **P4 (NEX-04)** | Assistant: Durable Intelligence & Memory | PENDING | — | — | Depends on G3 | — |
| **P5 (NEX-05)** | Developer Tools & Delegation | PENDING | — | — | Depends on G4 | — |
| **P6 (NEX-06)** | Personal & Voice | PENDING | — | — | Depends on G5 | — |
| **P7 (NEX-07)** | Continuous Proactivity & Overnight Workflow | PENDING | — | — | Depends on G6 | — |
| **P8 (NEX-08)** | Release, Migration & Decommission Gate | PENDING | — | — | Depends on G7 | — |
