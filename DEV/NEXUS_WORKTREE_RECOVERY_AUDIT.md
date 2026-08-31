# Nexus Worktree Recovery Audit

Recovery snapshot: `.superpowers/recovery/20260830-203500/`  
Baseline: `c96885218b3cff319f481aed4423dc5fd7b52a30`

| Worktree | Path | Branch | HEAD | Commits after baseline | Dirty tracked | Untracked | Nexus V0 evidence | Role |
|---|---|---|---|---:|---:|---:|---|---|
| Root | `/projetos/tools/IAPro-Nexus-Workspace-OS-Handoff-2026-08-29/ai-manager` | `feat/nexus-maximum-delivery` | `c968852` | 0 | yes | yes | terminal protocol consumer fix; bootstrap docs | BASELINE_ROOT |
| V0 implementation | `.worktrees/nexus-v0-implementation` | `feat/nexus-v0-implementation` | canonical recovery HEAD | 2 recovery commits plus docs | clean after consolidation | none | Workspace presentation, desktop windows, launcher, Ask Agent, attention and protocol work | PRIMARY_IMPLEMENTATION |
| Production validation | `.worktrees/production-validation` | `feat/nexus-production-validation` | `58d2bf0` | 0 relative to baseline (baseline is descendant) | no | no | historical validation artifacts; no active V0 source | SECONDARY_IMPLEMENTATION |

## Recovery decisions

- The V0 implementation worktree is the canonical worktree because it contains the only active V0 source additions and tests.
- Root functional changes were compared file-by-file. The terminal parser/consumer fix was recovered selectively; bootstrap docs and reports were not treated as product implementation.
- The production-validation worktree is preserved unchanged; its history is older than the official baseline and no unique active source needed recovery.
- No worktree, branch, commit, untracked file or dirty patch was deleted. Recovery artifacts include tracked/staged patches, status, HEAD/branch, and copied untracked files per worktree.
- A shell snapshot-loop error initially produced no file copy; it was corrected with explicit paths and verified by artifact counts.
