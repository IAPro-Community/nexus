# Nexus V0 execution boundaries

- Source: `feat/nexus-maximum-delivery@c96885218b3cff319f481aed4423dc5fd7b52a30`.
- NEVER run destructive git commands, overwrite user files, expose secrets, push, merge, publish, or release.
- Preserve the project root worktree and its untracked bootstrap documents.
- Keep Agent identity, runtime broker, Workspace model, Mission/WorkPlan infrastructure, and Maestro independence intact.
- Rollback: discard only changes in `.worktrees/nexus-v0-implementation`; original worktree remains untouched.
- Verify with focused tests, full Go tests, web typecheck/tests/build, and phase exit gates.
