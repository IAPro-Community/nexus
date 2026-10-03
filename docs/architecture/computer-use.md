# Computer use

Computer use is a Nexus 1.1 execution capability for workflows that must inspect or operate native
desktop applications. It is not an orchestration engine and it does not replace browser automation.

## Contract

The canonical loop is:

```text
snapshot -> semantic action -> snapshot -> verification
```

Accessibility semantics are preferred over coordinate input. Coordinate-based mouse operations are
fallbacks only.

Observation and mutation permissions are separate in `AutonomyContract`:

- `allow_computer_use`: master capability gate;
- `allow_screen_capture`: pixel capture;
- `allow_keyboard_input`: text, key and semantic value mutation;
- `allow_pointer_input`: click, scroll and drag;
- `allow_window_management`: focus/move/resize.

All default to false. Password managers are blocked by the default policy even if a broad computer
permission is granted.

## Runtime boundary

The Go Core owns policy and protocol. Native implementations are versioned helpers discovered as
`nexus-computer-use-<platform>` or through `NEXUS_COMPUTER_HELPER`. Helpers receive one JSON
request on stdin and emit one JSON response on stdout. Secrets therefore do not need to be placed in
shell arguments or a durable operation file.

This boundary intentionally allows a platform implementation to reuse mature accessibility APIs:

- Windows: UI Automation + native input APIs;
- macOS: Accessibility API + ScreenCaptureKit/CoreGraphics;
- Linux: AT-SPI with compositor-specific capture fallback.

The Orca project (MIT) validates this per-platform helper approach. Nexus uses its own Go protocol
and policy layer rather than importing Orca's Electron runtime. If native source is later ported
substantially from Orca, the pinned upstream commit and MIT notice must be added to third-party
notices before release.

## Current rollout

The contract, policy, helper protocol and diagnostic CLI are present first. Native mutation remains
unavailable until a platform helper passes its own permission, safety and native smoke gates.

```bash
nexus computer status --json
nexus computer capabilities --json
nexus computer permissions --json
```

No unavailable capability is reported as supported.
