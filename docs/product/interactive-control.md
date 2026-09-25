# Interactive control

Interactive TTY provider launches default to **supervised transport**. `--direct`,
`--supervised`, `--print`, and `NEXUS_LAUNCH_MODE` are resolved centrally.

The supervision boundary is intentionally narrower than the provider UI boundary:
Nexus owns process lifecycle, profiles/resources, attach/reconnect, observability,
capacity/cost and explicit control actions. The provider owns its native TUI,
keyboard protocol, selectors, slash commands, prompts and ordinary Enter
semantics.

CLI shortcuts (`nexus codex`, `nexus agy`, …) therefore remain
**byte-transparent by default**. They do not bind an Interactive Lead and do not
hold Enter for Mission promotion unless the user explicitly launches with
`--lead`.

`--lead` opts in to the Nexus Interactive Lead / Mission-promotion boundary.
`--no-lead` remains accepted as a compatibility flag and always wins if both
flags are present. Both flags are Nexus-owned and stripped before provider
arguments are launched.

Even with `--lead`, terminal control sequences are transport data, never prompt
content. CSI and SS3 navigation sequences must reach the provider atomically;
focus/CPR metadata may be scrubbed, but Nexus must not translate arrow keys into
text or reinterpret provider-native selectors.

Web/Desktop sessions use the persisted `nexus.interactive_lead=true` runtime
label as the explicit boundary. Missing/false labels keep the terminal
byte-transparent even when the runtime is supervised.

Provider-native commands such as `/status`, `/model`, `:...` and interactive
question/selector flows remain provider-owned. Nexus orchestration should be
invoked through its own surfaces or through an explicitly enabled Lead/Mission
boundary, not by silently changing the provider's ordinary terminal behavior.

`/nexus` and `:nexus` are the canonical control prefixes. `//nexus` and
`::nexus` escape one delimiter and forward the provider command literally.
The old `ai` aliases are not part of the supervised control contract.

The terminal input path is covered by focused tests for CSI/SS3 navigation,
split escape sequences and explicit Lead opt-in. Real Codex/Antigravity selector
smokes remain required release evidence.
