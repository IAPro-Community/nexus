# Interactive control

Interactive TTY provider launches default to **supervised transport**. `--direct`,
`--supervised`, `--print`, and `NEXUS_LAUNCH_MODE` are resolved centrally.

The supervision boundary is intentionally narrower than the provider UI boundary:
Nexus owns process lifecycle, profiles/resources, attach/reconnect, observability,
capacity/cost and explicit control actions. The provider owns its native TUI,
keyboard protocol, selectors, slash commands, prompts and ordinary Enter
semantics.

CLI shortcuts (`nexus codex`, `nexus agy`, …) therefore use
**STICKY_LEAD by default on an interactive supervised TTY**: Nexus binds a
persistent Lead identity/context for continuity, while keeping
`nexus.delegation_mode=OFF`. OFF bypasses LineShadow/Promoter entirely, so
ordinary prompts, Enter, slash commands, selectors and control sequences remain
byte-transparent and provider-owned.

The implicit STICKY_LEAD binding is best-effort. If Nexus cannot prepare or bind
its metadata after the provider runtime has started, it falls back to the same
provider-native attach instead of terminating a healthy CLI session.

`--lead` makes persistent Lead binding explicit/required; it still does **not**
authorize Mission promotion. Mission execution remains explicit through Nexus
orchestration surfaces such as Composer/Flow/`nexus run`, or through a
deliberate `NEXUS_DELEGATION_MODE=ASK|AUTO` override.

When delegation is deliberately enabled, the interactive router distinguishes four outcomes: `KEEP_WITH_LEAD`, `DELEGATE`, `PROMOTE` and `ASK`. An explicit request for one registered provider (for example, “chame o Codex para revisar”) may use `DELEGATE`: Nexus runs one bounded read-only headless turn through the registered Usage/resource pool, compiles bounded canonical Skills, and returns the result to the existing Lead without creating a WorkPlan or Mission. Requests that require decomposition, autonomous side effects or multiple workstreams continue through `PROMOTE`/Mission. Provider mentions without routing language do not trigger delegation.

`DELEGATE` never weakens the terminal safety boundary: it is unavailable while delegation mode is `OFF`, it does not replace the Lead provider/session, and it requires a provider execution path with verified read-only isolation. An unavailable preferred provider may fall back only through the registered ResourceScheduler pool and the fallback remains explicit in the result.

`--no-lead` disables STICKY_LEAD and always wins if both flags are present.
`--direct` bypasses supervised transport entirely. All Nexus-owned launch flags
are stripped before provider arguments are launched.

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
split escape sequences, STICKY_LEAD defaults and explicit Lead opt-out. Real Codex/Antigravity selector
smokes remain required release evidence.
