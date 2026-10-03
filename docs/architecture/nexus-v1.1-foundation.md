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

1. execution workspace + project manifest;
2. review center + source-control/terminal ergonomics;
3. browser context + design mode;
4. candidate runs and independent evaluation;
5. local/SSH/Nexus Node execution targets;
6. integrations, automations, ephemeral environments and computer use;
7. mobile companion;
8. cross-feature hardening and release certification.

These are internal milestones and beta checkpoints, not separate product versions.
