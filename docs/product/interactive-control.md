# Interactive control

Interactive TTY provider launches default to supervised mode. `--direct`,
`--supervised`, `--print`, and `NEXUS_LAUNCH_MODE` are resolved centrally.

CLI shortcuts (`nexus codex`, `nexus agy`, …) stay on SessionHost and, in
V1, bind a persistent Nexus Interactive Lead by default. Complete prompt lines
pass through the same intent/promotion policy used by Web/Desktop: DIRECT work
stays in the provider conversation; PLAN work becomes a durable Mission/Flow.
Provider-native commands such as `/status` and `:...` bypass intent routing
immediately, so supervision does not turn them into Missions.

Use `--no-lead` for the legacy byte-transparent attach with no Mission
promotion. `--lead` remains accepted for explicitness/compatibility. Both are
Nexus-owned and stripped before provider args. Web/Desktop **Start** of an
agent already carries an AgentID and uses the same promotion semantics.

`/nexus` and `:nexus` are the canonical control prefixes. `//nexus` and
`::nexus` escape one delimiter and forward the provider command literally.
The old `ai` aliases are not part of the supervised control contract.

The byte router is covered by focused tests. Provider PTY E2E and contextual
completion remain pending evidence.
