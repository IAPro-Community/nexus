# Maestro canonical memory — Nexus read client (stage 00B)

**Status:** implementation candidate, behind the normal feature release gate; no UI routing, mutation or legacy-memory migration in this increment.
**Canonical destination:** `release/v1.0.0`
**Requires:** Maestro memory Bridge read v1 (integrated via #50/#52); release Bridge context #179 and Maestro canonical composition #57.

## Architecture

The existing `internal/nexus/intelligence_memory.go` remains **unchanged in this PR** for legacy compatibility. No new memory table, migration or semantic writes were introduced. `internal/control/handoff/maestro_memory.go` adds a capability-negotiated on-demand, read-only client for Maestro-owned observations, returning an ephemeral DTO:

- `ReadMaestroMemory(ctx, workspace, taskID, search, limit)`: reuses the existing `maestroBridgeCall` subprocess and its `initialize` negotiation for `memory.search`; there is no second Bridge transport;
- validates canonical contract `schemaVersion=1`, `authority=maestro`, matching JSON-RPC id, observation IDs, bounded size/page and scope fields;
- applies Nexus redaction to returned summaries;
- does **not** store responses in Nexus, synthesize empty results on Bridge absence, or fall back to shadow `IntelligenceMemory`.

The Maestro Bridge derives Git/repository/task visibility and performs its own sanitization; this client may be used only with a workspace already authorized by Nexus Core. The client does not expose arbitrary branch/worktree/repository IDs or pass a memory path.

## Follow-up sequence

1. **00A Maestro:** canonical read Bridge contract is integrated; keep regression tests and same-SHA compatibility evidence.
2. **00B Nexus:** validate this typed read client and same-version integration with real Maestro. It is intentionally not yet wired into Web/API surfaces.
3. **00C migration design:** inventory legacy install/project habits, identify counterpart types/scopes, dry-run export, backups and rollback.
4. **00D guarded mutation:** add `memory.record` to Maestro Bridge after authorization, idempotency and scope validation; wire Nexus explicit approvals (no bulk shadow writes).
5. **00E cutover:** switch authorised semantic read/write paths to Maestro and retire duplicate writes only after parity, backup and negative tests. Degraded/MISSING stays explicit.

**Never** enable automatic destructive migration, deletion of original rows, or rewriting content without explicit safety gates.

## Tests and evidence

- `go test ./internal/control/handoff -run 'TestDecodeMaestroMemory|TestMaestroMemory' -count=1`
- `go test -race ./internal/control/handoff` in compatible Go from `go.mod`.
- Same-sha integration: test with Maestro PR #42 candidate binary on a real repo with two branches/worktrees, task-scoped observations, malformed JSON, wrong method, timeout and redaction.
- No claim of real end-to-end certification before recorded exit codes and SHA.

Related: Nexus #86, #113; [roadmap](../design/omp-inspired-nexus-evolution-roadmap.md).

## Integração com o Bridge da release

Stage 00B reutiliza `maestroBridgeCall` de `maestro_project_context.go` (introduzido no Nexus #179), com handshake `initialize`, versão 1, capability `memory.read=true`, request ID e limites de resposta verificados. Não adiciona outro cliente JSON-RPC nem outro armazenamento semântico. Sem evidência de Bridge real, o incremento permanece candidato e não habilita UI ou migração automática.
