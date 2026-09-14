# Product interaction model

Nexus has one Core with three surfaces. Users should think in three modes even
though the internal model remains richer:

| Mode | User intent | Core path |
| --- | --- | --- |
| Direct | Work directly with an agent | Project → Session → Provider → Runtime → Workspace/Terminal |
| Automated | Deepen an objective into a strong prompt, then optionally plan | Composer (optional) → WorkPlan → Flow Studio → Run |
| Orchestrated | Execute toward a Definition of Done | Mission → Run/tasks → verification, optionally Maestro |

Composer and Flow stay separate surfaces with one clarification contract.
Installation and project intelligence memories skip repeated habit questions.
CLI and API preserve existing names for backward compatibility.

Maestro contributes method, skills, risk, review, and quality-gate policy. It
does not become the Nexus runtime or persistence layer.
