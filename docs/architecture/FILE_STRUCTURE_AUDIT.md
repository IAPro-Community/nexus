# File Structure Audit — IAPro Nexus

Generated: 2026-09-05

## Current Relevant Tree

```text
web/src/
├── App.tsx                          # Root component (2 routes: demo + workspace)
├── index.tsx                        # Entry point
├── index.css                        # Tailwind + xterm imports
├── api.ts                           # Shared HTTP transport + legacy runtime facade
├── nexus/api.ts                     # Typed Nexus domain API facade
├── api.test.ts
├── types.ts                         # Global types (653 lines) ← GOD FILE
├── app/
│   ├── NexusShell.tsx               # App shell
│   ├── NexusWorkspaceApp.tsx        # Main workspace app (754 lines)
│   ├── WorkspaceSurfaceHost.tsx     # Surface host
│   ├── NexusDemoApp.tsx
│   ├── NexusSplashScreen.tsx
│   ├── workspace-os.css             # Workspace styles
│   ├── attention-layout.css
│   ├── commands/                    # Command palette
│   ├── components/                  # App-level components
│   ├── modals/                      # App-level modals
│   └── tour/                        # Product tour
├── components/                      # ← GOD DIRECTORY (flat)
│   ├── Dashboard.tsx (487 lines)
│   ├── Sidebar.tsx
│   ├── TerminalPane.tsx (490 lines)
│   ├── TerminalView.tsx
│   ├── EventsView.tsx
│   ├── ProvidersView.tsx
│   ├── AttentionNotificationCard.tsx
│   ├── AttentionNotificationManager.tsx
│   ├── GlobalAttentionRadar.tsx
│   ├── AttentionIntermediationBanner.tsx
│   ├── ContinueModal.tsx
│   ├── HandoffModal.tsx
│   ├── StartModal.tsx
│   ├── workbench/                   # Empty directory
│   ├── attentionText.ts
│   └── terminalViewModel.ts
├── design-system/
│   ├── index.ts                     # Barrel (export *)
│   ├── primitives/
│   │   ├── index.tsx                # 238 lines, many components in one file
│   │   ├── ContextDrawer.tsx
│   │   └── ContextMenu.tsx
│   └── theme/
│       ├── ThemeProvider.tsx
│       ├── theme.ts
│       └── themePresets.ts
├── features/
│   ├── agents/                      # Agent management
│   ├── overview/                    # Project overview
│   ├── projects/                    # Project management
│   ├── sessions/                    # Sessions
│   ├── settings/                    # Settings
│   ├── shell/                       # Project shell
│   └── work/                        # Work/flow/composer (largest feature)
├── i18n/                            # Internationalization
├── keyboard/                        # Keyboard shortcuts
├── lib/                             # Shared utilities (small)
├── nexus/                           # Nexus-specific components
├── notifications/                   # Notification system
├── services/                        # ← GLOBAL SERVICES (single file)
│   └── WorkspaceLayoutService.ts
├── ui/
│   └── primitives.tsx               # Re-export barrel
└── workspace/                       # Workspace management

internal/                            # Go backend
├── app/                             # CLI entry + commands
├── browser/                         # Browser integration
├── buildinfo/                       # Build metadata
├── control/                         # Control plane (largest)
│   ├── driver/                      # Provider adapters
│   ├── events/                      # Event bus
│   ├── flags/                       # Flag normalization
│   ├── handoff/                     # Session handoff
│   ├── host/                        # Session host (PTY, ring buffer)
│   ├── ids/                         # ID generation
│   ├── launchenv/                   # Launch environment
│   ├── launcher/                    # Process launcher
│   ├── notify/                      # Notifications
│   ├── originpolicy/                # Origin policy
│   ├── protocol/                    # Wire protocol
│   ├── registry/                    # Session registry
│   ├── terminal/                    # Terminal management
│   ├── tui/                         # TUI components
│   ├── web/                         # HTTP server + embedded frontend
│   ├── websocketio/                 # WebSocket I/O
│   └── workspace/                   # Workspace management
├── conversation/                    # Conversation management
├── core/                            # Core domain
│   ├── classifier/                  # Input classifier
│   ├── config/                      # Configuration
│   ├── cooldown/                    # Rate limit cooldown
│   ├── exitcode/                    # Exit codes
│   ├── fallback/                    # Fallback logic
│   ├── model/                       # Core types
│   ├── provider/                    # Provider registry + adapters
│   ├── quota/                       # Quota management
│   ├── scheduler/                   # Account scheduler
│   ├── security/                    # Security engine
│   ├── session/                     # Session management
│   └── telemetry/                   # Telemetry
├── localization/                    # i18n
├── nexus/                           # Nexus business logic (largest)
│   ├── autonomyguard/
│   ├── contextsnapshot/
│   ├── intelligence/                # AI engine
│   ├── maestrogates/
│   ├── plangraph/
│   ├── runner/                      # Mission runner
│   └── store/                       # State store
├── profile/                         # Profile management
├── release/                         # Release management
├── runtime/                         # Runtime isolation
├── testutil/                        # Test utilities
└── tui/                             # TUI data tables
```

## Problem Classification

| Problem | Location | Severity | Notes |
|---------|----------|----------|-------|
| **god file** | `web/src/types.ts` (653 lines) | high | Single file with all shared types |
| **god directory** | `web/src/components/` (14 flat files) | high | No feature organization |
| **dead directory** | `web/src/components/workbench/` | medium | Empty directory |
| **incorrect ownership** | `web/src/components/Dashboard.tsx` | high | Domain component in generic dir |
| **incorrect ownership** | `web/src/components/TerminalPane.tsx` | high | Terminal domain in generic dir |
| **incorrect ownership** | `web/src/components/Sidebar.tsx` | high | Navigation domain in generic dir |
| **incorrect ownership** | `web/src/components/ProvidersView.tsx` | high | Provider domain in generic dir |
| **incorrect ownership** | `web/src/components/EventsView.tsx` | medium | Events domain in generic dir |
| **deep import** | `web/src/ui/primitives.tsx` | medium | Re-exports from design-system, adds alias |
| **excessive coupling** | `web/src/design-system/primitives/index.tsx` (238 lines) | medium | Many components in single file |
| **generic bucket** | `web/src/services/` | low | Single service, could be feature-local |
| **generic bucket** | `web/src/lib/` | low | Only safeArray.ts, fine as-is |
| **platform leakage** | None detected in frontend | low | Go handles platform separation well |
| **circular** | None detected | low | Import graph is clean |
| **barrel abuse** | `web/src/design-system/index.ts` uses `export *` | low | Acceptable for design system |
| **barrel abuse** | `web/src/ui/primitives.tsx` re-exports everything | low | Convenience re-export, acceptable |

## Target Architecture

```text
web/src/
├── app/                             # Composition root
│   ├── App.tsx                      # Root component
│   ├── providers/                   # Context providers
│   └── initialization/              # Boot sequence
├── workbench/                       # IDE shell (orchestration only)
│   ├── shell/                       # Shell chrome
│   ├── layout/                      # Layout management
│   ├── navigation/                  # Sidebar, taskbar
│   ├── command-palette/             # Command palette
│   ├── status-bar/                  # Status indicators
│   └── context-menu/                # Context menus
├── features/                        # Product capabilities
│   ├── terminal/                    # Terminal feature
│   ├── chat/                        # Chat/conversation
│   ├── flow/                        # Flow/mission management
│   ├── agents/                      # Agent management
│   ├── providers/                   # Provider views
│   ├── quota/                       # Quota display
│   ├── workspace/                   # Workspace management
│   ├── projects/                    # Project management
│   ├── sessions/                    # Session management
│   ├── settings/                    # Settings
│   ├── overview/                    # Project overview
│   ├── notifications/               # Notification system
│   └── tour/                        # Product tour
├── shared/                          # Truly cross-cutting
│   ├── ui/                          # Design system primitives
│   ├── hooks/                       # Generic hooks
│   ├── lib/                         # Generic utilities
│   ├── styles/                      # Global styles, tokens
│   ├── config/                      # App configuration
│   └── types/                       # Shared type definitions
└── assets/                          # Static assets
```

## Dependency Direction (enforced)

```text
app → workbench → features → shared
```

`shared` never depends on `features`. Features never import from each other's internals.
