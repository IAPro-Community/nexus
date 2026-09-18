# Account switching

Account switching is a same-provider handoff and must preserve Agent, Project,
workspace, and lineage identity while creating a new runtime generation. A
provider change is a context handoff and cannot reuse the source provider
session ID as a native target session.

Continuity states are recorded honestly, including unverified native resume and
new-session fallback. Runtime-level unit/state evidence exists; full terminal
follow-handoff E2E is still unverified.
