# Optional product capabilities — default OFF

Nexus Direct, projects, persistent Agents, PTY/ConPTY, sessions, basic Browser,
authentication, remote terminal and the ordinary Attention Center are core.
They are **not** controlled by the experimental global switch.

The optional Composer, Flow, Missions and other advanced capabilities are
preserved in source and storage, but **all are disabled unless explicitly
enabled in the server process**. No localStorage value, query parameter or
client request can authorize an advanced server mutation.

The server accepts only the literal string `true` for each variable:

```sh
NEXUS_FEATURES_ENABLED=true
NEXUS_FEATURE_COMPOSER=true
NEXUS_FEATURE_FLOW=true
NEXUS_FEATURE_MISSIONS=true
NEXUS_FEATURE_PARALLEL_CHALLENGE=true
NEXUS_FEATURE_COMPARISON_LAB=true
NEXUS_FEATURE_VOICE_ACTIONS=true
NEXUS_FEATURE_NEX_PROACTIVITY=true
NEXUS_FEATURE_DESIGN_INSPECT=true
NEXUS_FEATURE_VISUAL_DIFF=true
```

The **global flag defaults to false**, and **each individual flag defaults to
false**. Enabling the global flag alone enables nothing. Enable only
individually approved features; do not copy all examples into production
configuration.

For Composer/Flow navigation, the frontend bundle must be rebuilt with the
same explicitly enabled environment flags. A stale or separately configured
frontend cannot authorize backend changes.

While disabled, historic reads remain accessible to authorized clients for
export/recovery, and cancel/stop actions remain possible for already-running
jobs. New plan, Composer and Mission mutations are rejected with an
actionable error; the database and existing files are never deleted.
Basic terminal launch and Browser sessions are unaffected.

The Nexus startup also skips Mission worker recovery and schedule-loop startup
unless the global and MISSIONS flags are enabled. Restart the server when
changing environment flags. Existing data is not deleted.

The legacy 48-capability acceptance matrix is **not** automatically revised
by these flags. The release owner must separately approve a revised product
contract and exact-SHA validation. Advanced features require regression tests
with defaults OFF and explicit ON, including direct HTTP/CSRF and remote-user
checks.

Known limitation: other optional UI panels that are not tied to the APIs above
still need their own presentation-level guards before being classified as
fully disabled. These flags are not a universal runtime kill switch for
background jobs already started, so use existing cancellation mechanisms.

The CLI also applies the MISSIONS flag to `nexus run <goal>` and supervised
Lead Mission promotion. Invoking `nexus run <provider>` still launches the
provider directly. Internal migration/test helpers are not public entrypoints
and must not be treated as a substitute for these gates.

Browser/voice UI also defaults off. The web esbuild pipeline uses the **same
environment variables** as the backend and emits explicit compile-time flags.
Build the frontend with the selected `NEXUS_FEATURES_ENABLED=true` and
`NEXUS_FEATURE_<KEY>=true` values, then restart the server with matching
values. Composer/Flow UI is one legacy surface group and requires all three
COMPOSER, FLOW and MISSIONS opt-ins. The former
`NEXUS_EXPERIMENTAL_COMPOSER_FLOW` variable no longer enables it implicitly.
The backend always remains authoritative for permission to execute.

## Regression coverage

Legacy optional-feature tests explicitly enable their corresponding flags.
Composer attachment uploads are scoped to Composer sessions; standalone
Agent/terminal operations are not gated by that route. Historical Maestro
usage import and the read-only Release Desk API keep their existing contracts.
Existing Mission schedules may be canceled when creation is disabled; only
exact cancel/pause routes bypass the disabled mutation gate. The CLI plan
create/compile/run commands require Flow opt-in; run additionally requires
Missions. No automatic fallback to a different execution owner.
