# Interactive control

Interactive TTY provider launches default to supervised mode. `--direct`,
`--supervised`, `--print`, and `NEXUS_LAUNCH_MODE` are resolved centrally.

CLI shortcuts (`nexus codex`, `nexus agy`, …) stay on SessionHost with a
byte-transparent attach so the provider keyboard (including `/status`) works
like a native CLI. That default does **not** create a "Nexus Interactive Lead"
agent or hold Enter for Mission promotion.

Opt into Lead + LineShadow promotion with `--lead` (Nexus-owned; stripped
before provider args). Web/Desktop **Start** of an agent already carries an
AgentID and is unchanged by this contract.

`/nexus` and `:nexus` are the canonical control prefixes. `//nexus` and
`::nexus` escape one delimiter and forward the provider command literally.
The old `ai` aliases are not part of the supervised control contract.

The byte router is covered by focused tests. Provider PTY E2E and contextual
completion remain pending evidence.
