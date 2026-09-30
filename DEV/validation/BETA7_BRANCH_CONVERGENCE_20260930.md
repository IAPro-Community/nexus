# Nexus 1.0 Beta 7 — Branch convergence before cleanup

Date: 2026-09-30

## Canonical line

- Canonical development/release branch: `release/v1.0.0`.
- Product version remains `1.0.0-beta.7`.
- `main` and `release/v1.0.0` are never cleanup candidates.
- This convergence was performed selectively; stale branches were not merged wholesale.

Pre-report release anchor after the selective convergence:

`f3d73bb9e390ac178620c448f16d0ef2e5dac80b`

## Ported into the canonical release

The following behavior was still useful and was ported without restoring older
terminal/delegation semantics:

1. Native Desktop/HUD dispatch `__control-host` before Wails startup.
2. Embedded native Core can disable optional hosts-file mutation, so Desktop/HUD
   do not require administrator privileges just to register `nexus.dev`.
3. Wails HTTP traffic remains same-origin while terminal WebSocket traffic uses
   the real loopback Core endpoint.
4. Runtime liveness rejects inactive/STale generations before host probing.
5. STOPPED specialist Agents may be configured before a concrete provider/profile
   is selected; Resource Picker persists the exact resource before execution.
6. New Agent creation no longer invents a fictitious `provider:default`
   resource. Provider selection is only a soft preference until allocation.
7. Explicit provider routing is distinguished from incidental provider mentions.
   Requests such as "chame/use/peça ao Codex" are user-owned provider pins,
   while phrases such as "a documentação do Codex" do not route work.
8. Explicit read-only delegation fails closed when the requested provider has no
   eligible registered resource; it never pretends that another provider
   satisfied the explicit request.
9. Windows case-insensitive checkout collision was removed by keeping only
   `docs/GLOSSARY.md`.

## Superseded by newer canonical implementations

The following old-branch changes were reviewed but were already replaced by
newer code in Beta 7 and therefore were not transplanted:

- Windows provider executable/PATHEXT/npm shim handling is owned by
  `internal/runtime/executable.go` and `ResolveCommand`.
- Windows case-insensitive environment handling is already implemented in
  `EnvSet`.
- quiet subprocess behavior is owned by `internal/executil`.
- Windows console resize notification is already implemented.
- stale runtime cleanup uses SessionHost identity/`HostLive` and host generation.
- frontend runtime selection uses the newer runtime-agent generation mapping.
- shared API CSRF refresh/retry already replaces the older Nexus-local variant.
- Usage/Resource UI already distinguishes measured quota from
  UNKNOWN/UNSUPPORTED without fabricating 0%.
- current Windows homologation pins exact SHA/toolchain and includes browser,
  visual, formatting and security gates.
- installer SourceRef flow already checks out the explicitly requested ref in an
  isolated temporary source checkout.

## Intentionally not ported

These old changes conflict with current Beta 7 invariants or introduce an
unproven execution path:

- automatic/provider-aware ConPTY activation: rejected because the canonical
  release deliberately keeps ConPTY opt-in to preserve provider-native TUI,
  keyboard, mouse and CSI behavior.
- sticky Lead with automatic delegation: rejected. Interactive delegation stays
  OFF unless explicitly authorized; Lead identity is not Mission authorization.
- old specialist execution-preference metadata containing named model candidates:
  not ported because that branch did not connect the metadata to the live
  ResourceScheduler/ModelCatalog. Dead configuration must not be treated as
  executable policy.
- old failed-Mission "restart" convenience endpoint/button: not ported because
  creating a fresh autonomous run can replay external side effects and conflicts
  with the newer durable recovery/intervention/exactly-once safety model.

## Branch preservation status

After convergence, every non-canonical branch currently present in the
repository has an `archive/**` reference with the exact same HEAD SHA.

Audit result at this point:

- non-canonical original branches: 50
- exact archived copies: 50
- originals without an exact archive: 0

The previously stale archives were corrected:

- `fix/release-1.0.0-auto-skill-resource-routing-20260929`
  -> `archive/2026-09-30/fix/release-1.0.0-auto-skill-resource-routing-20260929`
  at `9dbde700ce2f2f12011469d58bc1e1d609cb5960`.
- `release/v1.0.0-beta`
  -> `archive/2026-09-30/release/v1.0.0-beta`
  at `1463e7e1f372e2e7ba24448c79c0a50a8989a896`.

Therefore deleting the non-canonical original branch refs later does not discard
their Git history; the exact heads remain reachable under `archive/**`.

## Validation truth

Static contract review was completed for the selectively ported files and
version alignment remains Beta 7. No open PR remains from the convergence.

GitHub native certification jobs are separate from this branch-convergence
decision. At the time of this report, native certification jobs were skipped and
the self-hosted sanitize job was queued. Do not reinterpret queued/skipped jobs
as a green native certification.

Branch cleanup may proceed based on exact archive preservation, but release
certification remains governed by its own exact-SHA validation gates.
