# Nexus 1.1 foundation

Nexus 1.1 evolves the v1 control plane without replacing its execution authorities.

## Non-negotiable boundaries

The canonical decision chain remains:

```text
TaskRequirements
  -> AgentMatcher             (who)
  -> ResourceScheduler        (provider/profile/model/account)
  -> ExecutionTargetResolver  (where)
  -> ExecutionWorkspace       (which isolated checkout/environment)
  -> MissionRunner            (durable lifecycle)
  -> WorkReceipt + Evidence   (proof)
  -> COMPLETED_VERIFIED
```

Agent identity, provider, account, model, machine and workspace are independent dimensions.
Direct provider use must remain direct; v1.1 features must not silently promote a simple
`nexus <provider>` invocation into a Mission or Candidate Run.

## Rollout

All high-risk v1.1 surfaces start disabled and can be rolled back independently:

- `execution_workspaces_v2`
- `review_center`
- `browser_design_mode`
- `candidate_runs`
- `remote_execution`
- `nexus_nodes`
- `computer_use`
- `mobile_companion`

## Reuse before rewrite

Nexus 1.1 must prefer existing production boundaries:

- existing secure git worktree validation is the base for task-scoped execution workspaces;
- existing WorkReceipt, ReviewFinding and EvidenceBundle are the data source for Review Center;
- existing Playwright dependency and browser infrastructure are the base for browser inspection;
- existing Mission Teams and MissionRunner remain the orchestration authority for Candidate Runs;
- existing remote-node blueprint, runtime registry and terminal protocol are the base for remote execution;
- existing provider/account/quota routing remains authoritative.

External implementations may be reused only when their license is compatible and the resulting
runtime fits Nexus boundaries. Orca is MIT-licensed and is a useful reference for worktree UX,
diff review, remote execution and computer-use contracts. Copying substantial Orca source requires
retaining its MIT notice in the corresponding third-party attribution. Prefer adapting small
contracts and behavior over importing its Electron runtime into the Go core.

## Computer use

Computer use is a capability, not a new orchestration engine. The shared contract will expose
observation separately from mutation:

```text
COMPUTER_USE
  SCREEN_CAPTURE
  KEYBOARD_INPUT
  POINTER_INPUT
  WINDOW_MANAGEMENT
```

The master `AllowComputerUse` permission never implies the four granular permissions.
Every permission defaults denied. Platform adapters may use native helpers, but MissionRunner and
AutonomyContract remain the policy authority.

Browser automation remains a separate capability. A browser action must not silently escalate to
desktop input, and desktop input must not bypass provider/workspace/target policy.

## Delivery milestones inside 1.1

1. execution workspace + strict `nexus.yaml` project manifest;
2. review center + source-control/terminal ergonomics;
3. browser context + design mode;
4. candidate runs and independent evaluation;
5. local/SSH/Nexus Node execution targets;
6. integrations, automations, ephemeral environments and computer use;
7. mobile companion;
8. cross-feature hardening and release certification.

These are internal milestones and beta checkpoints, not separate product versions.

## Project manifest

`nexus.yaml` is parsed with `gopkg.in/yaml.v3`, which was already present in the module checksum
set. Unknown fields are rejected, paths must stay relative to the project root, and command lists are
bounded. The manifest describes project/workspace setup; it does not bypass ResourceScheduler or
AutonomyContract.

## Execution target

Nexus 1.1 now persists a typed `ExecutionTarget` alongside each allocated Mission package.
This is a projection over the v1 custom-target contract; it does not replace
`command_template`, ResourceScheduler or RuntimeSession.

```text
AgentMatcher      -> who
ResourceScheduler -> provider/profile/model
Target Resolver   -> local/SSH/Nexus Node/custom
Workspace         -> isolated checkout/environment
MissionRunner     -> lifecycle/evidence
```

Existing v1 custom targets remain valid. SSH configured through the established argv-only
`command_template` is recognized as `SSH` rather than being reimplemented. The current
runtime therefore keeps its tested launcher semantics while v1.1 gains durable target identity
for remote scheduling, node enrollment and computer-use routing.

### Project execution target catalog

`nexus.yaml` may declare named SSH targets under `execution.targets`. The catalog is strict:
target IDs are unique, `local` is reserved, the target workspace/location are bounded, templates
use the existing non-shell argv grammar, the executable must resolve to `ssh`, and values that
match Nexus secret-redaction patterns are rejected. SSH authentication remains owned by the host
SSH configuration/agent; passwords, private keys and tokens do not belong in the manifest.

This catalog is intentionally **declarative in the first rollout**. Loading
`defaultTarget: devbox` does not silently move an Agent or Mission to another machine. Explicit
target activation is a separate step behind `remote_execution`, so a repository checkout cannot
turn into remote code execution merely by adding `nexus.yaml`.

Legacy per-Agent `command_template` targets remain compatible and continue to use the v1 launcher
path. The managed target resolver will build on the catalog rather than replacing that compatibility
surface.

## Candidate execution policy

Candidate Runs use a bounded decision before any extra provider work is created. The decision is
based on task specialty, estimated complexity, estimated risk and routing confidence. Cost is a
constraint after quality eligibility; it is not the reason to start with the cheapest model.

The fan-out is capped at three and remains one for review/verification/documentation work. Easy,
high-confidence work also remains one. When candidate execution is enabled, Execution Preview binds
the planned candidate count into its approval token so GO cannot silently expand cost after approval.

## Quick Commands

Project Quick Commands are declared in `nexus.yaml` and projected into the existing Command Palette.
Nexus does not add a second shell executor. Selecting a Quick Command starts the existing supervised
Project Shell and inserts the validated single-line command after the terminal acquires its normal
writer lease. Nexus does not append Enter in this rollout.

The prefill is consumed from the workspace View immediately after insertion. A stale layout may at
worst re-insert text after a crash; it cannot execute a command on browser refresh. Automatic
execution is intentionally deferred until it has an explicit confirmation/audit contract.

## Browser Design Mode

The Design Mode runtime reuses the Chromium DevTools Protocol over Nexus' existing
`gorilla/websocket` dependency. Playwright remains a development/QA dependency and is not required
by the shipped binary.

`StartDesignBrowser` launches an isolated Chromium profile with DevTools bound to loopback only.
The project-scoped manager owns one Design Mode browser per Project and closes the process plus
temporary screenshots on replacement, stop and Nexus Web shutdown.

The feature-gated `design-mode` Workspace surface is available from the existing Command Palette
and semantic deep link. It can open the browser, start a visual picker, poll the selected CSS
selector, inspect it, display the cropped screenshot and send the bounded textual context to an
existing Agent. The picker intercepts the user's click only to select an element; Nexus does not
synthesize navigation, keyboard or pointer actions in this milestone.

`InspectSelector` produces a bounded `UIContextArtifact` containing outer HTML, a small parent
neighborhood, selected computed styles, text, element bounds, optional source/test attributes, and
a cropped PNG under the Nexus data directory. Password fields and elements marked
`data-nexus-sensitive="true"` are refused. Browser-derived URL, HTML, text and source hints pass
through Nexus secret redaction before they are returned by the API. Agent handoff marks the browser
payload as untrusted data so DOM text cannot legitimately become instructions.

The screenshot is currently a human review artifact in the Design Mode surface; the first Agent
handoff sends the redacted textual artifact only. Image attachment to ContextCapsule/Agent input is
a separate follow-up so provider-specific multimodal support does not leak into this browser layer.

All public Design Mode operations except read-only status remain disabled unless
`browser_design_mode` is enabled (for local development:
`NEXUS_ENABLE_BROWSER_DESIGN_MODE=1`).
