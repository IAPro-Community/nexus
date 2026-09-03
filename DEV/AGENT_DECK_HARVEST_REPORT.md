# Agent-Deck Harvest Report

## Overview

Analysis of [agent-deck](https://github.com/tonyle9/agent-deck) (MIT license) for mature behaviors to harvest into Nexus.

## Features Studied

### 1. Session State Detection
- **Upstream**: `agent-deck/internal/detector/` - Heuristic-based terminal state detection
- **Behavior**: Pattern matching on terminal output to detect working/waiting/completed states
- **Decision**: **ADAPT** - Nexus already has `AttentionDetector` in `internal/control/host/attention.go`
- **Code Copied**: No - Nexus implementation is more comprehensive
- **Attribution**: Not required
- **Nexus Test**: `TestAttentionDetector` suite

### 2. Provider Session Discovery
- **Upstream**: `agent-deck/internal/discovery/` - Scans provider config directories for active sessions
- **Behavior**: Reads session files from provider-specific paths
- **Decision**: **ADAPT** - Implemented `ProviderSessionDiscovery` interface in `internal/control/workspace/session_discovery.go`
- **Code Copied**: No - Interface-only adaptation
- **Attribution**: Not required
- **Nexus Test**: `TestSessionDiscoveryManagerRegisterAndDiscover`

### 3. Session Resume Capability
- **Upstream**: `agent-deck/internal/resume/` - Provider-specific resume command construction
- **Behavior**: Builds resume arguments based on provider type
- **Decision**: **ADAPT** - Nexus already has `DriverCapabilities.Resume` and `BuildResumeArgs`
- **Code Copied**: No - Nexus implementation is more mature
- **Attribution**: Not required
- **Nexus Test**: Existing driver tests

### 4. Agent Status Display
- **Upstream**: `agent-deck/internal/display/` - Terminal-based status display
- **Behavior**: Shows agent status in a compact format
- **Decision**: **REJECT** - Nexus uses Charm/Bubble Tea stack for TUI
- **Code Copied**: No
- **Attribution**: Not required

### 5. tmux Session Management
- **Upstream**: `agent-deck/internal/tmux/` - tmux session creation and management
- **Behavior**: Creates tmux sessions for provider isolation
- **Decision**: **REJECT** - Nexus uses its own SessionHost with PTY/ConPTY
- **Code Copied**: No
- **Attribution**: Not required

## Summary

| Feature | Decision | Code Copied | Attribution |
|---|---|---|---|
| Session State Detection | ADAPT | No | Not required |
| Provider Session Discovery | ADAPT | No | Not required |
| Session Resume Capability | ADAPT | No | Not required |
| Agent Status Display | REJECT | No | Not required |
| tmux Session Management | REJECT | No | Not required |

## Key Takeaways

1. **Nexus is more mature** than agent-deck in most areas
2. **Interface-based adaptation** was preferred over code copying
3. **No MIT license obligations** since no code was literally copied
4. **Provider-specific detectors** could be added as future work (Claude Code, Codex, Gemini CLI, Aider)
