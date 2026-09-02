# IAPro Nexus Product Contract

Nexus is a **PROJECT-CENTRIC AGENTIC WORKSPACE / AGENT OPERATIONS LAYER**.

```text
Project
└── Workspace
    ├── Persistent Agents
    ├── Project Shells
    ├── Files / Project Tools
    ├── Composer
    │   └── Flow Draft
    └── Flow Runs
```

Agent, Runtime, Session and View are distinct. Persistent Agents have stable
AgentIDs; RuntimeGenerations are ephemeral. Closing an Agent View does not stop
the Agent. Project Shells are ephemeral runtimes and never create Agents.

Direct Agent works without Composer and independently of Context Readiness.
Composer is optional; a Flow belongs to Composer; a Flow Draft never executes;
only Approve & Run starts a Flow Run. WorkPlan remains internal and Mission
Runner remains the only runner. The existing Resource Scheduler remains the
only resource scheduler. Maestro is optional and independent; Nexus does not
reimplement semantic memory.

Nexus never fabricates quota, tokens, costs, tests, verification, WorkReceipts,
evidence or provider capabilities. Completion requires factual verification.
Unknown Flow states fail closed. Dependents receive bounded ContextCapsules and
factual WorkReceipts, never raw transcripts.
