# Mission Teams operations

## Observe a Mission

Use the existing run surfaces:

```bash
nexus plan show <plan-id>
nexus plan run <plan-id>
nexus run "<goal>"
```

The persisted WorkPlan facts expose the selected preset and reason. A Mission
Run exposes `current_phase_id`, `phase_runs`, package Agent IDs, desired and
actual routing, review finding history, receipts and verification results.

The Web Flow Run surface presents the preset, current phase, active Agent and
the actual provider/profile/model when durable routing evidence is available.

## Normal unattended behavior

Provider quota/rate-limit failure may select a fallback resource without
changing the Agent identity or desired affinity. Review changes are normal
remediation, not `NEEDS_YOU`. Restart resumes persisted phase/package state;
completed packages are not dispatched again.

## Human attention

Attention is reserved for product decisions, destructive/irreversible actions,
missing credentials/MFA, production approval, material ambiguity or an unknown
external dispatch outcome. A repeated review finding is not itself a user
question.

## Safety

Planner, reviewer and verifier permissions inherit the existing
`AutonomyContract ∩ Agent permissions ∩ Skill permissions ∩ runtime
capabilities` boundary. Review is read-only by default; remediation is a
Builder responsibility. Push and deploy remain disabled by the default
contract.
