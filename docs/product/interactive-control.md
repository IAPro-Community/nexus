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
`nexus.delegation_mode=OFF`. Ordinary prompts stay with the active provider;
complexity, decomposition heuristics and mentions of the active provider do
not create a Mission or trigger a handoff. An explicit request for another
registered provider can run one bounded in-session turn without creating a
Mission. An explicit Mission request starts Mission admission even when
delegation mode is OFF. If a delegated task may write files, Nexus allows it
only when the source session is YOLO and the selected provider has a verified
workspace-write/no-network contract. Review turns require verified read-only
capability. If eligibility or permissions fail, Nexus reports the exact reason
and asks the user how to proceed; it does not silently change providers or
start a Mission. The `LineShadow` capture must not reinterpret native commands
or menus. Keyboard transparency still requires smoke tests with real providers
on the release SHA.

The implicit STICKY_LEAD binding is best-effort. If Nexus cannot prepare or bind
its metadata after the provider runtime has started, it falls back to the same
provider-native attach instead of terminating a healthy CLI session.

`--lead` makes persistent Lead binding explicit/required. Mission execution
remains explicit through Nexus orchestration surfaces such as
Composer/Flow/`nexus run`, or a direct Mission request in the interactive Lead.
`NEXUS_DELEGATION_MODE=ASK|AUTO` does not turn task complexity alone into
Mission authorization.

`--no-lead` disables STICKY_LEAD and always wins if both flags are present.
`--direct` bypasses supervised transport entirely. All Nexus-owned launch flags
are stripped before provider arguments are launched.

Even with `--lead`, terminal control sequences are transport data, never prompt
content. CSI and SS3 navigation sequences must reach the provider atomically;
focus/CPR metadata may be scrubbed, but Nexus must not translate arrow keys into
text or reinterpret provider-native selectors.

Web/Desktop interactive Lead interception requires
`nexus.interactive_lead=true`; an absent/false label keeps the normal terminal
path free of Lead routing. With the label, explicit provider handoffs and
explicit Mission requests are honored independent of delegation mode;
heuristic task classification alone does not authorize either action. Native
provider input is still subject to terminal escape/keyboard regression gates.

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

## Limites e validação do roteamento interativo

- Código: `internal/nexus/registered_provider_routing.go`,
  `internal/app/control_cmd.go`, `internal/control/web/handler_terminal.go`.
- Testes: `internal/nexus/registered_provider_routing_test.go`,
  `internal/control/web/handler_terminal_input_test.go`.
- Comportamentos a validar: menção comum/negada a provider não delega; o
  provider da sessão atual continua sendo o Lead; pedido explícito a outro
  provider executa um turno; pedido explícito de Mission inicia admissão; Safe
  e Plan bloqueiam escritas delegadas; YOLO exige contrato de isolamento e
  aprovação verificados; comandos `/status` e seletores `Esc`/`Enter` não são
  desviados; falha de elegibilidade mostra a capacidade/recurso rejeitado e
  não repete efeitos nem troca de provider silenciosamente.
- **Estado:** ajuste de código documentado na release; a homologação de
  teclado/TUI nativo ainda exige smoke com providers reais no SHA exato. Veja
  [status](../release-status.md).
