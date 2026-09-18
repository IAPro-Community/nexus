# Implementation plan — Nexus automatic delegation

Base: `37b9ddec49ce72d869bbeb2ad402a1d4dd2bc674`

1. Add the source-agnostic `DelegationMode`, `DelegationDecision` and
   `DelegatedWorkstream` contract, persisted in existing WorkPlan facts.
2. Apply deterministic dispatch-brake heuristics before Flow expansion and
   project independent backend/frontend/QA steps with ownership and DAG
   dependencies; keep simple goals direct.
3. Route every delegated package through the existing `MatchAgents` and
   `ResourceScheduler` path; enrich only auto-created persistent Agents with a
   revisioned `AgentSpec`.
4. Carry the lead identity in the existing MissionRun envelope and expose the
   persisted delegation decision in the existing routing report.
5. Verify focused tests, the Nexus package suite, race/vet/full repository
   gates, and review the diff for duplicate schedulers or provider-bound Agent
   identities.
6. Connect supervised interactive provider input to the persistent Lead by the
   current workspace, route complete prompts through the policy, and return
   delegated Mission completion to the Lead without changing raw `control
   attach` semantics.
