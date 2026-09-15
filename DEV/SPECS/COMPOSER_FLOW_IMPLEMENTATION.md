# Composer → Flow contract

This campaign evolves the Nexus Composer only. The Flow canvas, WorkPlan,
Mission Runner, Scheduler and runtime remain owned by their existing modules
and are read-only integration targets.

The Composer owns the durable discovery revision, Motivation Map, provenance,
prompt artifacts and the explainable FlowSuitability assessment. A single
canonical prompt plus skill envelope is finalized (prompt variants and
destination receipts are no longer written by Finalize). Maestro remains
authoritative for method, process, canonical skills, gates and risk advice. A
missing Maestro is surfaced as `MAESTRO_DEGRADED`; the Composer does not
synthesize skills or advice.

`PromptArtifact → MaterializePromptArtifactAsFlow` is the existing handoff.
The handoff carries the artifact identity, revision, content hash, structured
brief, Motivation Map, selected skills and context in WorkPlan facts. When
FlowSuitability is `FLOW_BENEFICIAL` or `FLOW_STRONGLY_RECOMMENDED`, materialize
calls `DecomposePromptIntoFlowProposal` (same engine as Conductor) to produce a
real DAG; `DIRECT_FIT` keeps a single package. It creates a draft WorkPlan only;
it does not start an Agent, MissionRun or runtime.

The visual Flow (`FlowDraftEditor`) is a projection, not a second Composer plan.
Any Flow editor mutation must return through the canonical WorkPlan contract.
Layout is not execution semantics.

Web `auto_plan` / `GeneratePlanFromIntent` is retired at the HTTP boundary.
Empty Flow drafts are created without intelligence; rich multi-step plans come
from Composer → Turn into Flow (or CLI Conductor dispatch).

Legacy Composer sessions remain readable: absent revision/provenance fields are
represented by compatibility defaults and are never destructively migrated.
Tables for historical variants/receipts/`composer_revisions` may remain in the
schema without active writers.
