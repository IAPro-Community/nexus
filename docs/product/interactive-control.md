# Interactive control

Interactive TTY provider launches default to **supervised transport**. `--direct`,
`--supervised`, `--print`, and `NEXUS_LAUNCH_MODE` are resolved centrally.

The supervision boundary is intentionally narrower than the provider UI boundary:
Nexus owns process lifecycle, profiles/resources, attach/reconnect, observability,
capacity/cost and explicit control actions. The provider owns its native TUI,
keyboard protocol, selectors, slash commands, prompts and ordinary Enter
semantics.

CLI shortcuts (`nexus codex`, `nexus agy`, …) therefore remain
**byte-transparent by default**. They do not bind an Interactive Lead and never
treat an ordinary provider prompt as Mission authorization.

`--lead` opts in only to a persistent Nexus Interactive Lead identity/context.
It does **not** opt in to automatic Mission promotion: supervised provider
runtimes persist `nexus.delegation_mode=OFF` initially, even with a Lead bound.
Mission execution stays explicit through Nexus orchestration surfaces such as
Composer/Flow/`nexus run`. An advanced operator may deliberately enable
`NEXUS_DELEGATION_MODE=ASK|AUTO` while using `--lead`.

`--no-lead` remains accepted as a compatibility flag and always wins if both
flags are present. Both flags are Nexus-owned and stripped before provider
arguments are launched.

Even with `--lead`, terminal control sequences are transport data, never prompt
content. CSI and SS3 navigation sequences must reach the provider atomically;
focus/CPR metadata may be scrubbed, but Nexus must not translate arrow keys into
text or reinterpret provider-native selectors.

Web/Desktop promotion requires both an explicitly bound Lead and a delegation
mode other than `OFF`. The `nexus.interactive_lead=true` label alone is not
Mission authorization. Missing/false Lead labels or `delegation_mode=OFF`
keep the terminal byte-transparent even when the runtime is supervised.

Provider-native commands such as `/status`, `/model`, `:...` and interactive
question/selector flows remain provider-owned. Nexus orchestration is invoked
through its own explicit surfaces or by deliberately enabling Lead delegation;
being launched inside Nexus is never sufficient reason to create a Mission.

`/nexus` and `:nexus` are the canonical control prefixes. `//nexus` and
`::nexus` escape one delimiter and forward the provider command literally.
The old `ai` aliases are not part of the supervised control contract.

The terminal input path is covered by focused tests for CSI/SS3 navigation,
split escape sequences and explicit Lead opt-in. Real Codex/Antigravity selector
smokes remain required release evidence.
