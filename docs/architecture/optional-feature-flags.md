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

For Composer/Flow navigation, the frontend also needs the build-time define
`__NEXUS_EXPERIMENTAL_COMPOSER_FLOW__ = true`; its absence defaults to
false. Enabling the frontend does not authorize backend changes, and enabling
backend features does not silently expose screens in existing builds.

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
