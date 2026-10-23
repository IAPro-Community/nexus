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
`nexus.delegation_mode=OFF`. OFF desativa a **promoção automática de Missions**. Na revisão beta.10,
o caminho STICKY_LEAD da CLI usa `LineShadow` para observar a submissão de
linhas e detectar um pedido explicitamente direcionado a outro provider
registrado para um turno de revisão read-only. Prompts comuns continuam no
Lead (`KEEP`); a exceção só pode produzir `DELEGATE`, sem Mission. Essa
captura não deve reinterpretar comandos/menus nativos. A transparência de
teclado ainda requer smoke com providers reais no SHA da release.

The implicit STICKY_LEAD binding is best-effort. If Nexus cannot prepare or bind
its metadata after the provider runtime has started, it falls back to the same
provider-native attach instead of terminating a healthy CLI session.

`--lead` makes persistent Lead binding explicit/required; it still does **not**
authorize Mission promotion. Mission execution remains explicit through Nexus
orchestration surfaces such as Composer/Flow/`nexus run`, or through a
deliberate `NEXUS_DELEGATION_MODE=ASK|AUTO` override.

`--no-lead` disables STICKY_LEAD and always wins if both flags are present.
`--direct` bypasses supervised transport entirely. All Nexus-owned launch flags
are stripped before provider arguments are launched.

Even with `--lead`, terminal control sequences are transport data, never prompt
content. CSI and SS3 navigation sequences must reach the provider atomically;
focus/CPR metadata may be scrubbed, but Nexus must not translate arrow keys into
text or reinterpret provider-native selectors.

Web/Desktop interactive Lead interception requires
`nexus.interactive_lead=true`; an absent/false label keeps the normal terminal
path free of Lead promotion. With the label, `delegation_mode=OFF` permits
only the same explicit read-only `DELEGATE` exception and never authorizes
Mission creation. `ASK`/`AUTO` are separate opt-in modes. Native provider
input is still subject to terminal escape/keyboard regression gates.

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

## Limites e testes da exceção em OFF (22/10/2026)

- Código: `internal/nexus/registered_provider_routing.go`,
  `internal/app/control_cmd.go`, `internal/control/web/handler_terminal.go`.
- Testes: `internal/nexus/registered_provider_routing_test.go`,
  `internal/control/web/handler_terminal_input_test.go`.
- Comportamentos que devem ser confirmados: menção comum a provider não delega;
  revisão explícita registrada delega uma vez; escrita/implementação não é
  tratada como revisão read-only; comandos `/status` e seletor `Esc`/`Enter`
  não são desviados; falha de roteamento não cria Mission nem repete efeitos.
- **Estado:** implementação presente, sem homologação de teclado/TUI nativo
  no SHA do fechamento. Veja [status](../release-status.md).
