# Optional product capabilities — default OFF

## Contract

Nexus is a session-first Agent/terminal workspace. Projects, persistent Agents,
Direct, PTY/ConPTY, provider accounts, remote control, authenticated access,
manual layout persistence, basic Browser, runtime health, and runtime attention
remain core. They do not depend on the optional-feature switch.

The implementation and persisted records of optional capabilities are retained.
No migration deletes their data. The server is authoritative; neither a
client request, old frontend build, nor localStorage may grant execution.

The only supported activation is a literal `true` for both
`NEXUS_FEATURES_ENABLED` and the corresponding `NEXUS_FEATURE_<KEY>`.
Every absent, malformed, unknown, or false flag remains disabled.

| Key | Scope |
| --- | --- |
| COMPOSER | Composer admission and editing |
| FLOW | Flow creation/editing/compile and plan execution admission |
| MISSIONS | Mission admission, dispatch, recovery, workers and schedules |
| PARALLEL_CHALLENGE | Parallel comparison operation |
| COMPARISON_LAB | Comparison/campaign mutations and UI |
| VOICE_ACTIONS | Voice settings, transcription and capture UI |
| NEX_PROACTIVITY | Optional proactive suggestions; basic Nex remains |
| DESIGN_INSPECT | Advanced design inspection |
| VISUAL_DIFF | Visual comparison |
| SMART_LAYOUTS | Automatic arrangement, not manual splits or layout storage |
| DEVICE_PREVIEW | Browser device presets/viewport action, not ordinary browsing |

For example, opt into Composer/Flow/Missions explicitly in an isolated
development or homologation host:

```sh
NEXUS_FEATURES_ENABLED=true
NEXUS_FEATURE_COMPOSER=true
NEXUS_FEATURE_FLOW=true
NEXUS_FEATURE_MISSIONS=true
```

Do not put these opt-ins in installer defaults or production configuration.

## Runtime contract

The authenticated read-only endpoint `GET /api/v1/system/features` returns
the effective global switch and registered boolean capabilities with
`Cache-Control: no-store`. Web, desktop and mobile use the same bundle and
load the snapshot before the authenticated workspace/HUD is mounted.
An unavailable, older or malformed host fails closed for optional capabilities.
Restart the host and reload/reconnect clients after changing its environment;
no frontend rebuild is required. A stale client cannot authorize server mutations.
The legacy UI groups Composer/Flow/Missions and requires all three opt-ins.

Mission domain admission, resume, intervention, dispatch, scheduling and
worker startup enforce the same policy, including terminal WebSocket
promotion. Recovery and scheduling do not start automatically when disabled.
The CLI provider invocation and ordinary terminal session remain available.
Existing authorized historical reads, pause/cancel of legacy runs, schedule
cancellation, usage import, and Release Desk read-model contracts are retained.
A flag is not an emergency kill switch for an already running external
provider or Maestro process. Stop those via their existing control paths.

## Validation and governance

Legacy optional-feature tests use explicit, scoped opt-ins. Test the default
OFF profile independently of the opt-in profile. No test-only default ON in
production and no bypass of auth, CSRF, control ownership or entitlement.

The existing 48-capability acceptance matrix remains preserved as the previous
full-feature certification contract; its verifier still requires exact-SHA
evidence. Do not mark a capability verified merely because its UI is hidden.
Before publication, version a session-first release certification profile
with core gates mandatory and separately exercise optional modules with
explicit flags ON and OFF. Keep previous evidence and the exact-SHA security,
platform, device and terminal gates. Until the profile and its verifier are
implemented and approved, release status remains NO_GO.

Known limits: low-level internal runner fixtures are not an independent
public execution API. Historical read projections may continue to expose
stored records to authorized callers. All optional entrypoints and dedicated
end-to-end suites must be re-audited at the final SHA.

## Session-first profile inventory (planning)

The versioned `docs/reference/agent-workspace-v1-profiles.json` separates the 39
core capabilities and 9 explicitly gated capabilities while retaining all 48
historic requirements. Unclassified capabilities remain mandatory, including
security, recovery, terminal, cross-platform and two-device evidence.
Run `node scripts/verify-agent-workspace-profiles.mjs` to detect scope drift.
This is not a publication authorization. The signed evidence verifier still
requires 48 capabilities until a profile-bound signed schema and producer
are implemented and the owner approves the new gate.

The layout envelope stores independent core and full profiles; switching flags
preserves the inactive profile, including tabs and splits. Editing one profile
does not overwrite the other profile. Older clients and named snapshot
restoration must be tested before claiming cross-version preservation.

Synthetic documentation screenshots for optional modules use explicit opt-in
in isolated fixture servers, not production defaults. Screenshots must be
regenerated by the existing capture workflow; do not edit the manifest SHA
manually to bypass visual verification.
